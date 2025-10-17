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
    // Debug: summarize request
    log.Debug().
        Str("model", reqBody.Model).
        Bool("stream", reqBody.Stream).
        Int("input_items", len(reqBody.Input)).
        Int("include_len", len(reqBody.Include)).
        Msg("Responses: built request")

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
    log.Debug().Str("url", url).Int("body_len", len(b)).Bool("stream", reqBody.Stream).Msg("Responses: sending request")
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

        log.Trace().Msg("Responses: initiating HTTP request (streaming)")
        resp, err := http.DefaultClient.Do(req)
		if err != nil {
            log.Debug().Err(err).Msg("Responses: HTTP request failed")
			return nil, err
		}
		defer resp.Body.Close()
        log.Debug().Int("status", resp.StatusCode).Str("content_type", resp.Header.Get("Content-Type")).Msg("Responses: HTTP response received")
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
        var streamErr error
        var thinkBuf strings.Builder
        var sayBuf strings.Builder
        var summaryBuf strings.Builder
        log.Trace().Msg("Responses: starting SSE read loop")
		flush := func() error {
			if dataBuf.Len() == 0 {
				return nil
			}
			raw := dataBuf.String()
			dataBuf.Reset()
            log.Trace().Str("event", eventName).RawJSON("data", []byte(raw)).Msg("Responses: SSE event")
			var m map[string]any
			if err := json.Unmarshal([]byte(raw), &m); err != nil {
                log.Debug().Err(err).Str("event", eventName).Int("raw_len", len(raw)).Msg("Responses: failed to unmarshal SSE data")
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
            case "error":
                // Provider-level error event during streaming
                if errObj, ok := m["error"].(map[string]any); ok {
                    msgStr := ""
                    if v, ok := errObj["message"].(string); ok { msgStr = v }
                    codeStr := ""
                    if v, ok := errObj["code"].(string); ok { codeStr = v }
                    if msgStr == "" { msgStr = "responses stream error" }
                    if codeStr != "" { streamErr = fmt.Errorf("responses stream error (%s): %s", codeStr, msgStr) } else { streamErr = fmt.Errorf(msgStr) }
                } else {
                    streamErr = fmt.Errorf("responses stream error")
                }
                e.publishEvent(ctx, events.NewErrorEvent(metadata, streamErr))
            case "response.failed":
                // Response failed; try to extract nested error
                if respObj, ok := m["response"].(map[string]any); ok {
                    if errObj, ok2 := respObj["error"].(map[string]any); ok2 {
                        msgStr := ""
                        if v, ok := errObj["message"].(string); ok { msgStr = v }
                        codeStr := ""
                        if v, ok := errObj["code"].(string); ok { codeStr = v }
                        if msgStr == "" { msgStr = "responses failed" }
                        if codeStr != "" { streamErr = fmt.Errorf("responses failed (%s): %s", codeStr, msgStr) } else { streamErr = fmt.Errorf(msgStr) }
                    } else {
                        streamErr = fmt.Errorf("responses failed")
                    }
                } else {
                    streamErr = fmt.Errorf("responses failed")
                }
                e.publishEvent(ctx, events.NewErrorEvent(metadata, streamErr))
            case "response.reasoning_summary_part.added":
                // Start of a summary piece – forward as streaming info event
                e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary-started", nil))
            case "response.reasoning_summary_text.delta":
                if v, ok := m["delta"].(string); ok && v != "" {
                    summaryBuf.WriteString(v)
                    // Emit thinking partials for live reasoning summary text
                    e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, v, summaryBuf.String()))
                } else if s, ok := m["text"].(string); ok && s != "" {
                    summaryBuf.WriteString(s)
                    e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, s, summaryBuf.String()))
                }
            case "response.reasoning_summary_part.done":
                // End of a summary piece – forward as streaming info event
                e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary-ended", nil))
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
                    // Output deltas are not thinking tokens; always treat as output
                    sayBuf.WriteString(v)
                    log.Trace().Int("delta_len", len(v)).Int("message_len", len(message)).Msg("Responses: text delta")
                    e.publishEvent(ctx, events.NewPartialCompletionEvent(metadata, v, message))
                } else if tv, ok := m["text"].(map[string]any); ok {
                    if d, ok := tv["delta"].(string); ok {
                        message += d
                        sayBuf.WriteString(d)
                        log.Trace().Int("delta_len", len(d)).Int("message_len", len(message)).Msg("Responses: text delta (nested)")
                        e.publishEvent(ctx, events.NewPartialCompletionEvent(metadata, d, message))
                    }
                }
            case "response.completed":
                // usage may be nested under response.usage
                var usage map[string]any
                if u, ok := m["usage"].(map[string]any); ok {
                    usage = u
                } else if respObj, ok := m["response"].(map[string]any); ok {
                    if u2, ok2 := respObj["usage"].(map[string]any); ok2 {
                        usage = u2
                    }
                }
                if usage != nil {
                    if v, ok := usage["input_tokens"].(float64); ok { inputTokens = int(v) }
                    if v, ok := usage["output_tokens"].(float64); ok { outputTokens = int(v) }
                    // reasoning tokens may be nested under output_tokens_details
                    if od, ok := usage["output_tokens_details"].(map[string]any); ok {
                        if v, ok := od["reasoning_tokens"].(float64); ok { reasoningTokens = int(v) }
                    } else if v, ok := usage["reasoning_tokens"].(float64); ok {
                        reasoningTokens = int(v)
                    }
                    log.Debug().Int("input_tokens", inputTokens).Int("output_tokens", outputTokens).Int("reasoning_tokens", reasoningTokens).Msg("Responses: usage parsed")
                }
                // optional stop reason, sometimes nested
                if sr, ok := m["stop_reason"].(string); ok && sr != "" {
                    stopReason = &sr
                } else if respObj, ok := m["response"].(map[string]any); ok {
                    if sr, ok := respObj["stop_reason"].(string); ok && sr != "" { stopReason = &sr }
                }
                if stopReason != nil { log.Debug().Str("stop_reason", *stopReason).Msg("Responses: stop reason observed") }
			}
			return nil
		}
		for {
            line, err := reader.ReadString('\n')
			if err != nil {
                if err.Error() != "EOF" {
                    log.Debug().Err(err).Msg("Responses: error reading SSE line")
                } else {
                    log.Trace().Msg("Responses: EOF while reading SSE")
                }
				break
			}
			line = strings.TrimRight(line, "\r\n")
            if line != "" {
                preview := line
                if len(preview) > 200 { preview = preview[:200] + "…" }
                log.Trace().Str("line", preview).Msg("Responses: SSE line")
            }
			if line == "" {
				_ = flush()
				eventName = ""
				continue
			}
			if strings.HasPrefix(line, "event:") {
				eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
                log.Trace().Str("event", eventName).Msg("Responses: SSE event name")
				continue
			}
			if strings.HasPrefix(line, "data:") {
				if dataBuf.Len() > 0 { dataBuf.WriteByte('\n') }
				dataBuf.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
				continue
			}
		}
        log.Debug().Msg("Responses: SSE loop ended")
		if inputTokens > 0 || outputTokens > 0 {
			if metadata.Usage == nil { metadata.Usage = &events.Usage{} }
			metadata.Usage.InputTokens = inputTokens
			metadata.Usage.OutputTokens = outputTokens
		}
        if metadata.Extra == nil { metadata.Extra = map[string]any{} }
        if reasoningTokens > 0 { metadata.Extra["reasoning_tokens"] = reasoningTokens }
        metadata.Extra["thinking_text"] = thinkBuf.String()
        metadata.Extra["saying_text"] = sayBuf.String()
        if summaryBuf.Len() > 0 {
            metadata.Extra["reasoning_summary_text"] = summaryBuf.String()
            // Publish a friendly info event with the complete summary
            e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary", map[string]any{"text": summaryBuf.String()}))
        }
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
    log.Trace().Msg("Responses: initiating HTTP request (non-streaming)")
    resp, err := http.DefaultClient.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
    log.Debug().Int("status", resp.StatusCode).Str("content_type", resp.Header.Get("Content-Type")).Msg("Responses: HTTP response received (non-streaming)")
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


