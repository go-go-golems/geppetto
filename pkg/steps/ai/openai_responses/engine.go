package openai_responses

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/runtimeattrib"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/serde"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Engine implements the Engine interface for Open Responses-compatible API calls.
type Engine struct {
	settings            *settings.InferenceSettings
	observer            geppettoobs.Observer
	observabilityConfig geppettoobs.Config

	connection *connection.Connection
}

func NewEngine(c *connection.Connection, s *settings.InferenceSettings, opts ...EngineOption) (*Engine, error) {
	if c == nil {
		return nil, errors.New("API connection is required")
	}
	if s == nil || s.Chat == nil {
		return nil, errors.New("model chat settings are required")
	}
	e := &Engine{settings: settings.PayloadSettings(s), connection: c, observabilityConfig: geppettoobs.DefaultConfig()}
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

	stream := true
	reqBody.Stream = &stream
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	// Debug: summarize request
	log.Debug().
		Str("model", reqBody.Model).
		Int("input_items", len(reqBody.Input)).
		Int("include_len", len(reqBody.Include)).
		Msg("Responses: built request")

	route, err := e.connection.Resolve(responsesOperation)
	if err != nil {
		return nil, err
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
	resolvedURL := route.URL()
	log.Debug().Str("url", resolvedURL.String()).Int("body_len", len(b)).Msg("Responses: sending request")

	// Attach DebugTap if present on context
	var tap engine.DebugTap
	if t2, ok := engine.DebugTapFrom(ctx); ok {
		tap = t2
	}

	// Responses always uses streaming internally so provider-to-canonical event
	// normalization has one lifecycle path. Profiles may still carry chat.stream
	// for other engines, but this engine forces the request and runtime path.
	return e.runStreamingInference(ctx, t, b, metadata, tap, startTime, reqBody)
}
