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
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/rs/zerolog/log"
)

func (e *Engine) runNonStreamingInference(ctx context.Context, t *turns.Turn, httpClient *http.Client, url string, body []byte, apiKey string, metadata events.EventMetadata, startTime time.Time) (*turns.Turn, error) {
	// Non-streaming
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
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
