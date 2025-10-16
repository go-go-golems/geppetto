package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"bufio"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func (e *OpenAIEngine) runResponses(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	startTime := time.Now()
	// Build HTTP request to /v1/responses
	reqBody, err := buildResponsesRequest(e.settings, t)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	baseURL := "https://api.openai.com/v1"
	apiKey := ""
	if e.settings != nil && e.settings.API != nil {
		if v, ok := e.settings.API.BaseUrls["openai-base-url"]; ok && v != "" {
			baseURL = v
		}
		if v, ok := e.settings.API.APIKeys["openai-api-key"]; ok {
			apiKey = v
		}
	}
	url := strings.TrimRight(baseURL, "/") + "/responses"

	// Prepare metadata for events
	metadata := events.EventMetadata{
		ID: uuid.New(),
		LLMInferenceData: events.LLMInferenceData{
			Model: func() string { if reqBody.Model != "" { return reqBody.Model }; return "" }(),
			Temperature: nil,
			TopP:        nil,
			MaxTokens:   reqBody.MaxOutputTokens,
		},
	}
	if t != nil {
		metadata.RunID = t.RunID
		metadata.TurnID = t.ID
	}
    log.Debug().Str("url", url).Int("input_items", len(reqBody.Input)).Bool("stream", reqBody.Stream).Msg("Responses: sending request")
	e.publishEvent(ctx, events.NewStartEvent(metadata))

	// Streaming when configured
	if e.settings != nil && e.settings.Chat != nil && e.settings.Chat.Stream {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(b)))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			var m map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&m)
            log.Debug().Interface("error_body", m).Int("status", resp.StatusCode).Msg("Responses: HTTP error")
			return nil, fmt.Errorf("responses api error: status=%d body=%v", resp.StatusCode, m)
		}
		reader := bufio.NewReader(resp.Body)
		var eventName string
		var message string
		var dataBuf strings.Builder
		var inputTokens, outputTokens, reasoningTokens int
		var stopReason *string
		flush := func() error {
			if dataBuf.Len() == 0 {
				return nil
			}
			raw := dataBuf.String()
			dataBuf.Reset()
            log.Trace().Str("event", eventName).RawJSON("data", []byte(raw)).Msg("Responses: SSE event")
			var m map[string]any
			if err := json.Unmarshal([]byte(raw), &m); err != nil {
				return nil
			}
            switch eventName {
            case "response.output_item.added":
                if it, ok := m["item"].(map[string]any); ok {
                    if typ, ok := it["type"].(string); ok {
                        switch typ {
                        case "reasoning":
                            e.publishEvent(ctx, events.NewInfoEvent(metadata, "thinking-started", nil))
                        case "message":
                            e.publishEvent(ctx, events.NewInfoEvent(metadata, "output-started", nil))
                        }
                    }
                }
            case "response.output_item.done":
                if it, ok := m["item"].(map[string]any); ok {
                    if typ, ok := it["type"].(string); ok {
                        switch typ {
                        case "reasoning":
                            e.publishEvent(ctx, events.NewInfoEvent(metadata, "thinking-ended", nil))
                        case "message":
                            e.publishEvent(ctx, events.NewInfoEvent(metadata, "output-ended", nil))
                        }
                    }
                }
			case "response.output_text.delta":
				if v, ok := m["delta"].(string); ok && v != "" {
					message += v
                    log.Trace().Int("delta_len", len(v)).Int("message_len", len(message)).Msg("Responses: text delta")
					e.publishEvent(ctx, events.NewPartialCompletionEvent(metadata, v, message))
				} else if tv, ok := m["text"].(map[string]any); ok {
					if d, ok := tv["delta"].(string); ok {
						message += d
                        log.Trace().Int("delta_len", len(d)).Int("message_len", len(message)).Msg("Responses: text delta (nested)")
						e.publishEvent(ctx, events.NewPartialCompletionEvent(metadata, d, message))
					}
				}
			case "response.completed":
				if u, ok := m["usage"].(map[string]any); ok {
					if v, ok := u["input_tokens"].(float64); ok { inputTokens = int(v) }
					if v, ok := u["output_tokens"].(float64); ok { outputTokens = int(v) }
					if v, ok := u["reasoning_tokens"].(float64); ok { reasoningTokens = int(v) }
				}
				if sr, ok := m["stop_reason"].(string); ok && sr != "" {
					stopReason = &sr
				}
			}
			return nil
		}
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				_ = flush()
				eventName = ""
				continue
			}
			if strings.HasPrefix(line, "event:") {
				eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				if dataBuf.Len() > 0 { dataBuf.WriteByte('\n') }
				dataBuf.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
				continue
			}
		}
		if inputTokens > 0 || outputTokens > 0 {
			if metadata.Usage == nil { metadata.Usage = &events.Usage{} }
			metadata.Usage.InputTokens = inputTokens
			metadata.Usage.OutputTokens = outputTokens
		}
		if metadata.Extra == nil { metadata.Extra = map[string]any{} }
		if reasoningTokens > 0 { metadata.Extra["reasoning_tokens"] = reasoningTokens }
		if stopReason != nil { metadata.StopReason = stopReason }
		d := time.Since(startTime).Milliseconds(); dm := int64(d); metadata.DurationMs = &dm
		if strings.TrimSpace(message) != "" {
			turns.AppendBlock(t, turns.NewAssistantTextBlock(message))
		}
		e.publishEvent(ctx, events.NewFinalEvent(metadata, message))
		return t, nil
	}

	// Non-streaming
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(b)))
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" { req.Header.Set("Authorization", "Bearer "+apiKey) }
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var m map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&m)
        log.Debug().Interface("error_body", m).Int("status", resp.StatusCode).Msg("Responses: HTTP error (non-streaming)")
		return nil, fmt.Errorf("responses api error: status=%d body=%v", resp.StatusCode, m)
	}
	var rr responsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil { return nil, err }
	var message string
	for _, oi := range rr.Output {
		for _, c := range oi.Content {
			if c.Type == "output_text" || c.Type == "text" { message += c.Text }
		}
	}
	if strings.TrimSpace(message) != "" { turns.AppendBlock(t, turns.NewAssistantTextBlock(message)) }
	d := time.Since(startTime).Milliseconds(); dm := int64(d); metadata.DurationMs = &dm
	e.publishEvent(ctx, events.NewFinalEvent(metadata, message))
	return t, nil
}


