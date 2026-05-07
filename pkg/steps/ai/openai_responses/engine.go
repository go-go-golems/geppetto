package openai_responses

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/runtimeattrib"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/serde"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Engine implements the Engine interface for Open Responses-compatible API calls.
type Engine struct {
	settings            *settings.InferenceSettings
	observer            geppettoobs.Observer
	observabilityConfig geppettoobs.Config
}

func NewEngine(s *settings.InferenceSettings, opts ...EngineOption) (*Engine, error) {
	e := &Engine{settings: s, observabilityConfig: geppettoobs.DefaultConfig()}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	e.observabilityConfig = e.observabilityConfig.Normalized()
	return e, nil
}

// publishEvent publishes events to configured sinks and context sinks.
func (e *Engine) publishEvent(ctx context.Context, event events.Event) {
	e.observePublish(ctx, event, geppettoobs.StageGeppettoPublishStarted, nil)
	events.PublishEventToContext(ctx, event)
	e.observePublish(ctx, event, geppettoobs.StageGeppettoPublishDone, nil)
}

func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	startTime := time.Now()

	// Capture turn state before conversion if DebugTap is present
	if tap, ok := engine.DebugTapFrom(ctx); ok && t != nil {
		if turnYAML, err := serde.ToYAML(t, serde.Options{}); err == nil {
			tap.OnTurnBeforeConversion(turnYAML)
		}
	}

	// Build HTTP request to /v1/responses
	reqBody, err := e.buildResponsesRequest(t)
	if err != nil {
		return nil, err
	}
	if err := e.attachToolsToResponsesRequest(ctx, t, &reqBody); err != nil {
		return nil, err
	}
	// Debug: succinct preview of input items and tool blocks present on Turn
	if t != nil {
		toolCalls := 0
		toolUses := 0
		for _, b := range t.Blocks {
			if b.Kind == turns.BlockKindToolCall {
				toolCalls++
			}
			if b.Kind == turns.BlockKindToolUse {
				toolUses++
			}
		}
		log.Debug().Int("tool_call_blocks", toolCalls).Int("tool_use_blocks", toolUses).Msg("Responses: Turn tool blocks present")
	}
	{
		preview := previewResponsesInput(reqBody.Input)
		log.Debug().Int("input_items", len(reqBody.Input)).Interface("input_preview", preview).Msg("Responses: request input summary")
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	// Debug: summarize request
	log.Debug().
		Str("model", reqBody.Model).
		Bool("stream", reqBody.Stream).
		Int("input_items", len(reqBody.Input)).
		Int("include_len", len(reqBody.Include)).
		Msg("Responses: built request")

	apiKey := ""
	if e.settings != nil && e.settings.API != nil {
		apiKey = responsesAPIKey(e.settings.API)
	}
	url := responsesEndpoint(func() *settings.APISettings {
		if e.settings == nil {
			return nil
		}
		return e.settings.API
	}(), "/responses")
	if err := security.ValidateOutboundURL(url, security.OutboundURLOptions{
		AllowHTTP: false,
	}); err != nil {
		return nil, errors.Wrap(err, "invalid responses URL")
	}
	httpClient, err := settings.EnsureHTTPClient(func() *settings.ClientSettings {
		if e.settings == nil {
			return nil
		}
		return e.settings.Client
	}())
	if err != nil {
		return nil, errors.Wrap(err, "resolve responses HTTP client")
	}

	// Prepare metadata for events
	metadata := events.EventMetadata{
		ID: uuid.New(),
		LLMInferenceData: events.LLMInferenceData{
			Model: func() string {
				if reqBody.Model != "" {
					return reqBody.Model
				}
				return ""
			}(),
			Temperature: nil,
			TopP:        nil,
			MaxTokens:   reqBody.MaxOutputTokens,
		},
	}
	if t != nil {
		if sid, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok {
			metadata.SessionID = sid
		}
		if iid, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok {
			metadata.InferenceID = iid
		}
		metadata.TurnID = t.ID
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]any{}
	}
	if e.settings != nil {
		metadata.Extra[events.MetadataSettingsSlug] = e.settings.GetMetadata()
	}
	runtimeattrib.AddRuntimeAttributionToExtra(metadata.Extra, t)
	log.Debug().Str("url", url).Int("body_len", len(b)).Bool("stream", reqBody.Stream).Msg("Responses: sending request")
	e.publishEvent(ctx, events.NewStartEvent(metadata))

	// Attach DebugTap if present on context
	var tap engine.DebugTap
	if t2, ok := engine.DebugTapFrom(ctx); ok {
		tap = t2
	}

	// Streaming when configured
	if e.settings != nil && e.settings.Chat != nil && e.settings.Chat.Stream {
		return e.runStreamingInference(ctx, t, httpClient, url, b, apiKey, metadata, tap, startTime, reqBody)
	}

	// Non-streaming
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	log.Trace().Msg("Responses: initiating HTTP request (non-streaming)")
	// #nosec G704 -- URL is validated above with ValidateOutboundURL.
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	log.Debug().Int("status", resp.StatusCode).Str("content_type", resp.Header.Get("Content-Type")).Msg("Responses: HTTP response received (non-streaming)")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var m map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&m)
		log.Debug().Interface("error_body", m).Int("status", resp.StatusCode).Msg("Responses: HTTP error (non-streaming)")
		return nil, fmt.Errorf("responses api error: status=%d body=%v", resp.StatusCode, m)
	}
	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var rr responsesResponse
	if err := json.Unmarshal(rawResponse, &rr); err != nil {
		return nil, err
	}
	var message string
	latestMessageItemID := ""
	var latestMessageOutputIndex *int
	latestMessageStatus := ""
	for outputIndex, oi := range rr.Output {
		idx := outputIndex
		// Capture reasoning items (non-streaming)
		if oi.Type == "reasoning" {
			b := turns.Block{ID: oi.ID, Kind: turns.BlockKindReasoning, Payload: map[string]any{}}
			if oi.ID != "" {
				b.Payload[turns.PayloadKeyItemID] = oi.ID
			}
			if text := reasoningTextFromOutputContent(oi.Content); text != "" {
				b.Payload[turns.PayloadKeyText] = text
			}
			if len(oi.Summary) > 0 {
				b.Payload[turns.PayloadKeySummary] = append([]any(nil), oi.Summary...)
			}
			if oi.EncryptedContent != "" {
				b.Payload[turns.PayloadKeyEncryptedContent] = oi.EncryptedContent
			}
			setOpenAIResponsesBlockMetadata(&b, rr.ID, &idx, "reasoning", oi.Status)
			turns.AppendBlock(t, b)
		}
		if oi.Type == "message" && oi.ID != "" {
			latestMessageItemID = oi.ID
			latestMessageOutputIndex = &idx
			latestMessageStatus = oi.Status
		}
		for _, c := range oi.Content {
			switch c.Type {
			case "output_text", "text":
				message += c.Text
			case "output_json":
				if c.JSON != nil {
					if b, err := json.Marshal(c.JSON); err == nil {
						message += string(b)
					}
				}
			}
		}
	}
	if strings.TrimSpace(message) != "" {
		ab := turns.NewAssistantTextBlock(message)
		if latestMessageItemID != "" {
			if ab.Payload == nil {
				ab.Payload = map[string]any{}
			}
			ab.Payload[turns.PayloadKeyItemID] = latestMessageItemID
		}
		setOpenAIResponsesBlockMetadata(&ab, rr.ID, latestMessageOutputIndex, "message", latestMessageStatus)
		turns.AppendBlock(t, ab)
	}
	if totals, ok := parseUsageTotalsFromResponse(rr); ok {
		if totals.inputTokens > 0 || totals.outputTokens > 0 || totals.cachedTokens > 0 {
			if metadata.Usage == nil {
				metadata.Usage = &events.Usage{}
			}
			metadata.Usage.InputTokens = totals.inputTokens
			metadata.Usage.OutputTokens = totals.outputTokens
			metadata.Usage.CachedTokens = totals.cachedTokens
		}
		if totals.reasoningTokens > 0 {
			if metadata.Extra == nil {
				metadata.Extra = map[string]any{}
			}
			metadata.Extra["reasoning_tokens"] = totals.reasoningTokens
		}
	}
	d := time.Since(startTime).Milliseconds()
	dm := int64(d)
	metadata.DurationMs = &dm
	result := engine.BuildInferenceResultFromEventMetadata(metadata, responsesInferenceProvider(e.settings), false)
	if err := engine.PersistInferenceResult(t, result); err != nil {
		log.Warn().Err(err).Msg("Responses: failed to persist canonical inference_result")
	}
	e.publishEvent(ctx, events.NewFinalEvent(metadata, message))
	return t, nil
}
