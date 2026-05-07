package openai_responses

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/streamhelpers"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func (e *Engine) runStreamingInference(ctx context.Context, t *turns.Turn, httpClient *http.Client, url string, body []byte, apiKey string, metadata events.EventMetadata, tap engine.DebugTap, startTime time.Time, reqBody responsesRequest) (*turns.Turn, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	log.Trace().Msg("Responses: initiating HTTP request (streaming)")
	if tap != nil {
		tap.OnHTTP(req, body)
	}
	// #nosec G704 -- URL is validated above with ValidateOutboundURL.
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Responses: HTTP request failed")
		if tap != nil {
			tap.OnProviderObject("http.error", map[string]any{"error": err.Error()})
		}
		return nil, err
	}
	defer resp.Body.Close()
	log.Debug().Int("status", resp.StatusCode).Str("content_type", resp.Header.Get("Content-Type")).Msg("Responses: HTTP response received")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var m map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&m)
		log.Debug().Interface("error_body", m).Int("status", resp.StatusCode).Msg("Responses: HTTP error")
		if tap != nil {
			tap.OnHTTPResponse(resp, mustMarshalJSON(m))
		}
		return nil, fmt.Errorf("responses api error: status=%d body=%v", resp.StatusCode, m)
	}
	reader := bufio.NewReader(resp.Body)
	var eventName string
	var message string
	var dataBuf strings.Builder
	var inputTokens, outputTokens, cachedTokens, reasoningTokens int
	var stopReason *string
	var streamErr error
	var thinkBuf strings.Builder
	var sayBuf strings.Builder
	var currentReasoningText strings.Builder
	var currentReasoningSummary strings.Builder
	currentReasoningItemID := ""
	lastReasoningItemID := ""
	assistantByItem := map[string]string{}
	var summaryBuf strings.Builder
	currentResponseID := ""
	// Placeholder for potential future pairing of reasoning with assistant item id
	// (keep declared logic out until needed to avoid unused var)
	// Accumulate function_call tool uses
	type pendingCall struct {
		callID, name, itemID string
		outputIndex          *int
		status               string
		args                 strings.Builder
	}
	callsByItem := map[string]*pendingCall{}
	finalCalls := []pendingCall{}
	// Track encrypted reasoning content for the current reasoning item only.
	var currentReasoningEncryptedContent string
	var currentReasoningOutputIndex *int
	var lastReasoningOutputIndex *int
	var currentReasoningSummaryIndex *int
	var lastReasoningSummaryIndex *int
	var currentReasoningStatus string
	var latestMessageItemID string
	var latestMessageOutputIndex *int
	var latestMessageStatus string
	log.Trace().Msg("Responses: starting SSE read loop")
	flush := func() error {
		if dataBuf.Len() == 0 {
			return nil
		}
		raw := dataBuf.String()
		dataBuf.Reset()
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return nil
		}
		if respObj, ok := m["response"].(map[string]any); ok {
			if id, ok := respObj["id"].(string); ok && id != "" {
				currentResponseID = id
			}
		}
		if id, ok := m["response_id"].(string); ok && id != "" {
			currentResponseID = id
		}
		providerEventType := normalizeResponsesEventName(eventName)
		if providerEventType == "" {
			if typ, ok := m["type"].(string); ok {
				providerEventType = normalizeResponsesEventName(typ)
			}
		}
		e.observeProviderEvent(ctx, metadata, reqBody.Model, currentResponseID, providerEventType, m)
		appendAssistantChunk := func(chunk string) {
			if chunk == "" {
				return
			}
			message += chunk
			sayBuf.WriteString(chunk)
			e.publishEvent(ctx, events.NewPartialCompletionEvent(metadata, chunk, message))
		}
		backfillAssistantChunk := func(itemID, fullChunk string) {
			if fullChunk == "" {
				return
			}
			current := message
			if itemID != "" {
				if streamed := assistantByItem[itemID]; streamed != "" {
					current = streamed
				}
			}
			if strings.HasSuffix(current, fullChunk) {
				return
			}
			overlap := 0
			maxOverlap := len(fullChunk)
			if len(current) < maxOverlap {
				maxOverlap = len(current)
			}
			for i := maxOverlap; i > 0; i-- {
				if strings.HasSuffix(current, fullChunk[:i]) {
					overlap = i
					break
				}
			}
			missing := fullChunk[overlap:]
			if missing == "" {
				return
			}
			if itemID != "" {
				assistantByItem[itemID] += missing
			}
			appendAssistantChunk(missing)
		}
		backfillReasoningText := func(fullText string) {
			if fullText == "" {
				return
			}
			current := currentReasoningText.String()
			if strings.HasSuffix(current, fullText) {
				return
			}
			overlap := 0
			maxOverlap := len(fullText)
			if len(current) < maxOverlap {
				maxOverlap = len(current)
			}
			for i := maxOverlap; i > 0; i-- {
				if strings.HasSuffix(current, fullText[:i]) {
					overlap = i
					break
				}
			}
			missing := fullText[overlap:]
			if missing == "" {
				return
			}
			currentReasoningText.WriteString(missing)
			normalized := streamhelpers.NormalizeReasoningDelta(thinkBuf.String(), missing)
			thinkBuf.WriteString(normalized)
			e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, normalized, thinkBuf.String()))
		}
		chunkFromValue := func(v any) string {
			switch tv := v.(type) {
			case string:
				return tv
			default:
				if tv == nil {
					return ""
				}
				b, err := json.Marshal(tv)
				if err != nil {
					return ""
				}
				return string(b)
			}
		}
		switch providerEventType {
		case "response.output_item.added":
			if it, ok := m["item"].(map[string]any); ok {
				if typ, ok := it["type"].(string); ok {
					switch typ {
					case "reasoning":
						currentReasoningItemID = ""
						currentReasoningText.Reset()
						currentReasoningSummary.Reset()
						currentReasoningEncryptedContent = ""
						currentReasoningOutputIndex = nil
						currentReasoningSummaryIndex = nil
						currentReasoningStatus = ""
						if v, ok := it["id"].(string); ok && v != "" {
							currentReasoningItemID = v
						}
						if status, ok := it["status"].(string); ok && status != "" {
							currentReasoningStatus = status
						}
						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
							currentReasoningOutputIndex = &idx
						}
						e.publishEvent(ctx, events.NewInfoEvent(metadata, "thinking-started", providerData("openai_responses", currentResponseID, currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex)))
						// Capture encrypted reasoning content when present.
						if enc, ok := it["encrypted_content"].(string); ok && enc != "" {
							currentReasoningEncryptedContent = enc
						}
					case "message":
						e.publishEvent(ctx, events.NewInfoEvent(metadata, "output-started", nil))
						if v, ok := it["id"].(string); ok && v != "" {
							latestMessageItemID = v
						}
						if status, ok := it["status"].(string); ok && status != "" {
							latestMessageStatus = status
						}
						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
							latestMessageOutputIndex = &idx
						}
					case "web_search_call":
						itemID := ""
						if v, ok := it["id"].(string); ok {
							itemID = v
						}
						if act, ok := it["action"].(map[string]any); ok {
							if at, ok := act["type"].(string); ok && at == "search" {
								q := ""
								if v, ok := act["query"].(string); ok {
									q = v
								}
								e.publishEvent(ctx, events.NewWebSearchStarted(metadata, itemID, q))
							}
							if at, ok := act["type"].(string); ok && at == "open_page" {
								u := ""
								if v, ok := act["url"].(string); ok {
									u = v
								}
								e.publishEvent(ctx, events.NewWebSearchOpenPage(metadata, itemID, u))
							}
						}
					}
				}
			}
		case "response.web_search_call.in_progress":
			itemID := ""
			if v, ok := m["item_id"].(string); ok {
				itemID = v
			}
			// Query will be available later in output_item.done, so emit without query for now
			e.publishEvent(ctx, events.NewWebSearchStarted(metadata, itemID, ""))
		case "response.web_search_call.searching":
			itemID := ""
			if v, ok := m["item_id"].(string); ok {
				itemID = v
			}
			e.publishEvent(ctx, events.NewWebSearchSearching(metadata, itemID))
		case "response.web_search_call.completed":
			itemID := ""
			if v, ok := m["item_id"].(string); ok {
				itemID = v
			}
			e.publishEvent(ctx, events.NewWebSearchDone(metadata, itemID))
		case "error":
			// Provider-level error event during streaming
			if errObj, ok := m["error"].(map[string]any); ok {
				msgStr := ""
				if v, ok := errObj["message"].(string); ok {
					msgStr = v
				}
				codeStr := ""
				if v, ok := errObj["code"].(string); ok {
					codeStr = v
				}
				if msgStr == "" {
					msgStr = "responses stream error"
				}
				if codeStr != "" {
					streamErr = fmt.Errorf("responses stream error (%s): %s", codeStr, msgStr)
				} else {
					streamErr = errors.New(msgStr)
				}
			} else {
				streamErr = fmt.Errorf("responses stream error")
			}
			e.publishEvent(ctx, events.NewErrorEvent(metadata, streamErr))
			if tap != nil {
				tap.OnProviderObject("stream.error", m)
			}
		case "response.failed":
			// Response failed; try to extract nested error
			if respObj, ok := m["response"].(map[string]any); ok {
				if errObj, ok2 := respObj["error"].(map[string]any); ok2 {
					msgStr := ""
					if v, ok := errObj["message"].(string); ok {
						msgStr = v
					}
					codeStr := ""
					if v, ok := errObj["code"].(string); ok {
						codeStr = v
					}
					if msgStr == "" {
						msgStr = "responses failed"
					}
					if codeStr != "" {
						streamErr = fmt.Errorf("responses failed (%s): %s", codeStr, msgStr)
					} else {
						streamErr = errors.New(msgStr)
					}
				} else {
					streamErr = fmt.Errorf("responses failed")
				}
			} else {
				streamErr = fmt.Errorf("responses failed")
			}
			e.publishEvent(ctx, events.NewErrorEvent(metadata, streamErr))
			if tap != nil {
				tap.OnProviderObject("response.failed", m)
			}
		case "response.reasoning_summary_part.added":
			if itemID := itemIDFromProviderObject(m); itemID != "" {
				currentReasoningItemID = itemID
			}
			if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
				currentReasoningSummaryIndex = &idx
				lastReasoningSummaryIndex = &idx
			}
			// Start of a summary piece – forward as streaming info event
			e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary-started", providerData("openai_responses", currentResponseID, currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex)))
		case "response.reasoning_summary_text.delta":
			if itemID := itemIDFromProviderObject(m); itemID != "" {
				currentReasoningItemID = itemID
				lastReasoningItemID = itemID
			}
			if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
				currentReasoningSummaryIndex = &idx
				lastReasoningSummaryIndex = &idx
			}
			if v, ok := m["delta"].(string); ok && v != "" {
				before := summaryBuf.Len()
				normalized := streamhelpers.NormalizeReasoningSummaryDelta(summaryBuf.String(), v)
				e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, currentResponseID, providerEventType, m, len(v), len(normalized), before+len(normalized))
				summaryBuf.WriteString(normalized)
				currentReasoningSummary.WriteString(normalized)
				// Emit thinking partials for live reasoning summary text
				e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, normalized, summaryBuf.String()))
			} else if s, ok := m["text"].(string); ok && s != "" {
				before := summaryBuf.Len()
				normalized := streamhelpers.NormalizeReasoningSummaryDelta(summaryBuf.String(), s)
				e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, currentResponseID, providerEventType, m, len(s), len(normalized), before+len(normalized))
				summaryBuf.WriteString(normalized)
				currentReasoningSummary.WriteString(normalized)
				e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, normalized, summaryBuf.String()))
			}
		case "response.reasoning_summary_part.done":
			if itemID := itemIDFromProviderObject(m); itemID != "" {
				currentReasoningItemID = itemID
				lastReasoningItemID = itemID
			}
			if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
				currentReasoningSummaryIndex = &idx
				lastReasoningSummaryIndex = &idx
			}
			// End of a summary piece – forward as streaming info event
			e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary-ended", providerData("openai_responses", currentResponseID, currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex)))
		case "response.reasoning_text.delta":
			if itemID := itemIDFromProviderObject(m); itemID != "" {
				currentReasoningItemID = itemID
				lastReasoningItemID = itemID
			}
			if idx, ok := intFromProviderNumber(m["output_index"]); ok {
				currentReasoningOutputIndex = &idx
				lastReasoningOutputIndex = &idx
			}
			if d, ok := m["delta"].(string); ok && d != "" {
				before := thinkBuf.Len()
				normalized := streamhelpers.NormalizeReasoningDelta(thinkBuf.String(), d)
				e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, currentResponseID, providerEventType, m, len(d), len(normalized), before+len(normalized))
				thinkBuf.WriteString(normalized)
				currentReasoningText.WriteString(d)
				e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, d, thinkBuf.String()))
			} else if s, ok := m["text"].(string); ok && s != "" {
				before := thinkBuf.Len()
				normalized := streamhelpers.NormalizeReasoningDelta(thinkBuf.String(), s)
				e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, currentResponseID, providerEventType, m, len(s), len(normalized), before+len(normalized))
				thinkBuf.WriteString(normalized)
				currentReasoningText.WriteString(s)
				e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, s, thinkBuf.String()))
			}
		case "response.reasoning_text.done":
			if s, ok := m["text"].(string); ok && s != "" {
				// Done payloads can repeat already-streamed deltas for the current
				// item, but some providers send reasoning text only in the done
				// event. Backfill any missing suffix and emit the canonical
				// EventThinkingPartial so live reasoning renderers see the update.
				backfillReasoningText(s)
			}
		case "response.output_item.done":
			if it, ok := m["item"].(map[string]any); ok {
				if typ, ok := it["type"].(string); ok {
					switch typ {
					case "reasoning":
						if itemID := itemIDFromProviderObject(m); itemID != "" {
							currentReasoningItemID = itemID
							lastReasoningItemID = itemID
						}
						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
							currentReasoningOutputIndex = &idx
							lastReasoningOutputIndex = &idx
						}
						e.publishEvent(ctx, events.NewInfoEvent(metadata, "thinking-ended", providerData("openai_responses", currentResponseID, currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex)))
						// Append a reasoning block with encrypted content if present.
						rb := turns.Block{Kind: turns.BlockKindReasoning}
						payload := map[string]any{}
						if id, ok := it["id"].(string); ok && id != "" {
							rb.ID = id
							payload[turns.PayloadKeyItemID] = id
						}
						if currentReasoningItemID != "" && rb.ID == "" {
							rb.ID = currentReasoningItemID
							payload[turns.PayloadKeyItemID] = currentReasoningItemID
						}
						if status, ok := it["status"].(string); ok && status != "" {
							currentReasoningStatus = status
						}
						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
							currentReasoningOutputIndex = &idx
							lastReasoningOutputIndex = &idx
						}
						if terminalText := reasoningTextFromProviderContent(it["content"]); terminalText != "" {
							backfillReasoningText(terminalText)
						}
						if text := strings.TrimSpace(currentReasoningText.String()); text != "" {
							payload[turns.PayloadKeyText] = text
						}
						enc := currentReasoningEncryptedContent
						if v, ok := it["encrypted_content"].(string); ok && v != "" {
							enc = v
						}
						if enc != "" {
							payload[turns.PayloadKeyEncryptedContent] = enc
						}
						summary := reasoningSummaryEntriesFromPayload(it)
						if len(summary) == 0 {
							summary = reasoningSummaryEntriesFromText(currentReasoningSummary.String())
						}
						if len(summary) > 0 {
							payload[turns.PayloadKeySummary] = summary
						}
						rb.Payload = payload
						setOpenAIResponsesBlockMetadata(&rb, currentResponseID, currentReasoningOutputIndex, "reasoning", currentReasoningStatus)
						turns.AppendBlock(t, rb)
						currentReasoningItemID = ""
						currentReasoningText.Reset()
						currentReasoningSummary.Reset()
						currentReasoningEncryptedContent = ""
						currentReasoningOutputIndex = nil
						currentReasoningSummaryIndex = nil
						currentReasoningStatus = ""
						if tap != nil {
							tap.OnProviderObject("output.reasoning", it)
						}
					case "message":
						e.publishEvent(ctx, events.NewInfoEvent(metadata, "output-ended", nil))
						itemID := ""
						if v, ok := it["id"].(string); ok && v != "" {
							latestMessageItemID = v
							itemID = v
						}
						if status, ok := it["status"].(string); ok && status != "" {
							latestMessageStatus = status
						}
						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
							latestMessageOutputIndex = &idx
						}
						if rawContent, ok := it["content"].([]any); ok {
							for _, item := range rawContent {
								content, ok := item.(map[string]any)
								if !ok {
									continue
								}
								typ, _ := content["type"].(string)
								switch typ {
								case "output_text", "text":
									if s, ok := content["text"].(string); ok && s != "" {
										backfillAssistantChunk(itemID, s)
									}
								case "output_json":
									backfillAssistantChunk(itemID, chunkFromValue(content["json"]))
								}
							}
						}
						if tap != nil {
							tap.OnProviderObject("output.message", it)
						}
					case "function_call":
						// finalize function_call and publish ToolCall event
						name := ""
						if v, ok := it["name"].(string); ok {
							name = v
						}
						callID := ""
						if v, ok := it["call_id"].(string); ok {
							callID = v
						}
						itemID := ""
						if v, ok := it["id"].(string); ok {
							itemID = v
						}
						args := ""
						if v, ok := it["arguments"].(string); ok && v != "" {
							args = v
						}
						status := ""
						if v, ok := it["status"].(string); ok && v != "" {
							status = v
						}
						var outputIndex *int
						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
							outputIndex = &idx
						}
						if args == "" {
							if pc := callsByItem[itemID]; pc != nil {
								args = pc.args.String()
							}
						}
						if callID != "" && name != "" {
							e.publishEvent(ctx, events.NewToolCallEvent(metadata, events.ToolCall{ID: callID, Name: name, Input: args}))
							var b strings.Builder
							b.WriteString(args)
							finalCalls = append(finalCalls, pendingCall{callID: callID, name: name, itemID: itemID, outputIndex: outputIndex, status: status, args: b})
						}
					case "web_search_call":
						// Extract search query from action if available
						query := ""
						if action, ok := it["action"].(map[string]any); ok {
							if q, ok := action["query"].(string); ok {
								query = q
							}
						}
						itemID := ""
						if v, ok := it["id"].(string); ok {
							itemID = v
						}
						// Log the final query info at debug level
						if query != "" {
							log.Debug().Str("query", query).Str("item_id", itemID).Msg("Responses: web_search completed with query")
						}
						// Note: Don't emit another Done event here, already emitted by response.web_search_call.completed
					}
				}
			}
		case "response.output_text.delta":
			// Stream assistant text deltas
			itemID := ""
			if v, ok := m["item_id"].(string); ok && v != "" {
				itemID = v
			}
			if d, ok := m["delta"].(string); ok && d != "" {
				if itemID != "" {
					assistantByItem[itemID] += d
				}
				appendAssistantChunk(d)
				log.Trace().Int("delta_len", len(d)).Int("message_len", len(message)).Msg("Responses: text delta")
			} else if tv, ok := m["text"].(map[string]any); ok {
				if d, ok := tv["delta"].(string); ok && d != "" {
					if itemID != "" {
						assistantByItem[itemID] += d
					}
					appendAssistantChunk(d)
					log.Trace().Int("delta_len", len(d)).Int("message_len", len(message)).Msg("Responses: text delta (nested)")
				}
			}
			if tap != nil {
				tap.OnSSE(eventName, []byte(raw))
			}
		case "response.output_json.delta":
			itemID := ""
			if v, ok := m["item_id"].(string); ok && v != "" {
				itemID = v
			}
			if d, ok := m["delta"].(string); ok && d != "" {
				if itemID != "" {
					assistantByItem[itemID] += d
				}
				appendAssistantChunk(d)
			}
			if tap != nil {
				tap.OnSSE(eventName, []byte(raw))
			}
		case "response.output_json.done":
			itemID := ""
			if v, ok := m["item_id"].(string); ok && v != "" {
				itemID = v
			}
			if j, ok := m["json"]; ok {
				backfillAssistantChunk(itemID, chunkFromValue(j))
			}
			if tap != nil {
				tap.OnSSE(eventName, []byte(raw))
			}
		case "response.output_text.annotation.added":
			if ann, ok := m["annotation"].(map[string]any); ok {
				title, _ := ann["title"].(string)
				url, _ := ann["url"].(string)
				var startPtr, endPtr, outPtr, contPtr, annPtr *int
				if v, ok := ann["start_index"].(float64); ok {
					i := int(v)
					startPtr = &i
				}
				if v, ok := ann["end_index"].(float64); ok {
					i := int(v)
					endPtr = &i
				}
				if v, ok := m["output_index"].(float64); ok {
					i := int(v)
					outPtr = &i
				}
				if v, ok := m["content_index"].(float64); ok {
					i := int(v)
					contPtr = &i
				}
				if v, ok := m["annotation_index"].(float64); ok {
					i := int(v)
					annPtr = &i
				}
				e.publishEvent(ctx, events.NewCitation(metadata, title, url, startPtr, endPtr, outPtr, contPtr, annPtr))
			}
		case "response.function_call_arguments.delta":
			// Accumulate function_call arguments by item_id
			itemID := ""
			if v, ok := m["item_id"].(string); ok {
				itemID = v
			}
			if itemID != "" {
				pc := callsByItem[itemID]
				if pc == nil {
					pc = &pendingCall{itemID: itemID}
					callsByItem[itemID] = pc
				}
				if d, ok := m["delta"].(string); ok && d != "" {
					pc.args.WriteString(d)
				}
			}
		case "response.function_call_arguments.done":
			itemID := ""
			if v, ok := m["item_id"].(string); ok {
				itemID = v
			}
			if d, ok := m["arguments"].(string); ok && d != "" {
				if pc := callsByItem[itemID]; pc != nil {
					pc.args.Reset()
					pc.args.WriteString(d)
				}
			}
		// No assistant text in this event; only arguments aggregation
		case "response.completed":
			if totals, ok := parseUsageTotalsFromEnvelope(m); ok {
				inputTokens = totals.inputTokens
				outputTokens = totals.outputTokens
				cachedTokens = totals.cachedTokens
				reasoningTokens = totals.reasoningTokens
			}
			// optional stop reason, sometimes nested
			if sr, ok := m["stop_reason"].(string); ok && sr != "" {
				stopReason = &sr
			} else if respObj, ok := m["response"].(map[string]any); ok {
				if sr, ok := respObj["stop_reason"].(string); ok && sr != "" {
					stopReason = &sr
				}
			}
			if tap != nil {
				tap.OnProviderObject("response.completed", m)
			}
		}
		return nil
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			log.Debug().Err(err).Msg("Responses: error reading SSE line")
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			_ = flush()
			eventName = ""
			if err == io.EOF {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			if tap != nil {
				tap.OnSSE(eventName, []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))))
			}
			if err == io.EOF {
				_ = flush()
				break
			}
			continue
		}
		if err == io.EOF {
			_ = flush()
			break
		}
	}

	if inputTokens > 0 || outputTokens > 0 || cachedTokens > 0 {
		if metadata.Usage == nil {
			metadata.Usage = &events.Usage{}
		}
		metadata.Usage.InputTokens = inputTokens
		metadata.Usage.OutputTokens = outputTokens
		metadata.Usage.CachedTokens = cachedTokens
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]any{}
	}
	if reasoningTokens > 0 {
		metadata.Extra["reasoning_tokens"] = reasoningTokens
	}
	metadata.Extra["thinking_text"] = thinkBuf.String()
	metadata.Extra["saying_text"] = sayBuf.String()
	if summaryBuf.Len() > 0 {
		metadata.Extra["reasoning_summary_text"] = summaryBuf.String()
		// Publish a friendly info event with the complete summary and provider identity when available.
		data := providerData("openai_responses", currentResponseID, lastReasoningItemID, lastReasoningOutputIndex, lastReasoningSummaryIndex)
		if data == nil {
			data = map[string]any{}
		}
		data["text"] = summaryBuf.String()
		e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary", data))
	}
	if stopReason != nil {
		metadata.StopReason = stopReason
	}
	d := time.Since(startTime).Milliseconds()
	dm := int64(d)
	metadata.DurationMs = &dm
	if streamErr != nil {
		log.Debug().Err(streamErr).Msg("Responses: stream ended with provider error")
		return nil, streamErr
	}
	if strings.TrimSpace(message) != "" {
		ab := turns.NewAssistantTextBlock(message)
		if latestMessageItemID != "" {
			if ab.Payload == nil {
				ab.Payload = map[string]any{}
			}
			ab.Payload[turns.PayloadKeyItemID] = latestMessageItemID
		}
		setOpenAIResponsesBlockMetadata(&ab, currentResponseID, latestMessageOutputIndex, "message", latestMessageStatus)
		turns.AppendBlock(t, ab)
	}
	// Append tool_call blocks captured via Responses API
	for _, pc := range finalCalls {
		var args any
		if err := json.Unmarshal([]byte(pc.args.String()), &args); err != nil {
			args = map[string]any{}
		}
		b := turns.NewToolCallBlock(pc.callID, pc.name, args)
		// Preserve provider output item id so we can reference it later if needed
		if b.Payload == nil {
			b.Payload = map[string]any{}
		}
		if pc.itemID != "" {
			b.Payload[turns.PayloadKeyItemID] = pc.itemID
		}
		setOpenAIResponsesBlockMetadata(&b, currentResponseID, pc.outputIndex, "function_call", pc.status)
		turns.AppendBlock(t, b)
	}
	result := engine.BuildInferenceResultFromEventMetadata(metadata, responsesInferenceProvider(e.settings), len(finalCalls) > 0)
	if err := engine.PersistInferenceResult(t, result); err != nil {
		log.Warn().Err(err).Msg("Responses: failed to persist canonical inference_result")
	}
	e.publishEvent(ctx, events.NewFinalEvent(metadata, message))
	return t, nil

}
