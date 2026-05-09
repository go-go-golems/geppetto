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
	responseCompleted := false
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
	// Accumulate function_call tool uses.
	callsByItem := map[string]*responsesPendingCall{}
	finalCalls := []responsesPendingCall{}
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
	inferenceScopeID := metadata.InferenceID
	if inferenceScopeID == "" {
		inferenceScopeID = metadata.ID.String()
	}
	providerCallCorr := events.BuildProviderCallCorrelation("openai_responses", inferenceScopeID, "", 0, "")
	providerCallCorr.Model = reqBody.Model
	providerCallCorr.TurnID = metadata.TurnID
	e.publishEvent(ctx, events.NewProviderCallStartedEvent(metadata, providerCallCorr))
	streamState := newResponsesStreamState(reqBody, providerCallCorr, tap)
	responsesSegmentCorr := func(itemID string, outputIndex, summaryIndex *int, segmentType string) events.Correlation {
		streamState.currentResponseID = currentResponseID
		return streamState.segmentCorrelation(itemID, outputIndex, summaryIndex, segmentType)
	}
	toolCorr := func(itemID, callID string, outputIndex *int) events.Correlation {
		streamState.currentResponseID = currentResponseID
		return streamState.toolCorrelation(itemID, callID, outputIndex)
	}
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
		appendAssistantChunk := func(itemID string, outputIndex *int, chunk string) {
			if chunk == "" {
				return
			}
			message += chunk
			sayBuf.WriteString(chunk)
			text := message
			if itemID != "" && assistantByItem[itemID] != "" {
				text = assistantByItem[itemID]
			}
			e.publishEvent(ctx, events.NewTextDeltaEvent(metadata, responsesSegmentCorr(itemID, outputIndex, nil, events.SegmentTypeText), chunk, text, 0))
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
			appendAssistantChunk(itemID, latestMessageOutputIndex, missing)
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
			e.publishEvent(ctx, events.NewReasoningDeltaEvent(metadata, responsesSegmentCorr(currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex, events.SegmentTypeReasoning), normalized, thinkBuf.String(), 0))
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
						e.publishEvent(ctx, events.NewReasoningSegmentStartedEvent(metadata, responsesSegmentCorr(currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex, events.SegmentTypeReasoning), "provider"))
						// Capture encrypted reasoning content when present.
						if enc, ok := it["encrypted_content"].(string); ok && enc != "" {
							currentReasoningEncryptedContent = enc
						}
					case "message":
						if v, ok := it["id"].(string); ok && v != "" {
							latestMessageItemID = v
						}
						if status, ok := it["status"].(string); ok && status != "" {
							latestMessageStatus = status
						}
						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
							latestMessageOutputIndex = &idx
						}
						e.publishEvent(ctx, events.NewTextSegmentStartedEvent(metadata, responsesSegmentCorr(latestMessageItemID, latestMessageOutputIndex, nil, events.SegmentTypeText), "assistant"))
					case "function_call":
						itemID := ""
						if v, ok := it["id"].(string); ok && v != "" {
							itemID = v
						}
						callID := ""
						if v, ok := it["call_id"].(string); ok && v != "" {
							callID = v
						}
						name := ""
						if v, ok := it["name"].(string); ok && v != "" {
							name = v
						}
						var outputIndex *int
						if idx, ok := intFromProviderNumber(m["output_index"]); ok {
							outputIndex = &idx
						}
						if itemID != "" {
							callsByItem[itemID] = &responsesPendingCall{callID: callID, name: name, itemID: itemID, outputIndex: outputIndex}
						}
						e.publishEvent(ctx, events.NewToolCallStartedEvent(metadata, toolCorr(itemID, callID, outputIndex), callID, name))
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
				e.publishEvent(ctx, events.NewReasoningDeltaEvent(metadata, responsesSegmentCorr(currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex, events.SegmentTypeReasoning), normalized, summaryBuf.String(), 0))
			} else if s, ok := m["text"].(string); ok && s != "" {
				before := summaryBuf.Len()
				normalized := streamhelpers.NormalizeReasoningSummaryDelta(summaryBuf.String(), s)
				e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, currentResponseID, providerEventType, m, len(s), len(normalized), before+len(normalized))
				summaryBuf.WriteString(normalized)
				currentReasoningSummary.WriteString(normalized)
				e.publishEvent(ctx, events.NewReasoningDeltaEvent(metadata, responsesSegmentCorr(currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex, events.SegmentTypeReasoning), normalized, summaryBuf.String(), 0))
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
				e.publishEvent(ctx, events.NewReasoningDeltaEvent(metadata, responsesSegmentCorr(currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex, events.SegmentTypeReasoning), d, thinkBuf.String(), 0))
			} else if s, ok := m["text"].(string); ok && s != "" {
				before := thinkBuf.Len()
				normalized := streamhelpers.NormalizeReasoningDelta(thinkBuf.String(), s)
				e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, currentResponseID, providerEventType, m, len(s), len(normalized), before+len(normalized))
				thinkBuf.WriteString(normalized)
				currentReasoningText.WriteString(s)
				e.publishEvent(ctx, events.NewReasoningDeltaEvent(metadata, responsesSegmentCorr(currentReasoningItemID, currentReasoningOutputIndex, currentReasoningSummaryIndex, events.SegmentTypeReasoning), s, thinkBuf.String(), 0))
			}
		case "response.reasoning_text.done":
			if s, ok := m["text"].(string); ok && s != "" {
				// Done payloads can repeat already-streamed deltas for the current
				// item, but some providers send reasoning text only in the done
				// event. Backfill any missing suffix and emit the canonical
				// reasoning delta so live reasoning renderers see the update.
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
						finalReasoningText := strings.TrimSpace(currentReasoningText.String())
						finalReasoningStatus := currentReasoningStatus
						e.publishEvent(ctx, events.NewReasoningSegmentFinishedEvent(metadata, responsesSegmentCorr(lastReasoningItemID, lastReasoningOutputIndex, lastReasoningSummaryIndex, events.SegmentTypeReasoning), finalReasoningText, finalReasoningStatus))
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
						segmentText := message
						if itemID != "" && assistantByItem[itemID] != "" {
							segmentText = assistantByItem[itemID]
						}
						e.publishEvent(ctx, events.NewTextSegmentFinishedEvent(metadata, responsesSegmentCorr(itemID, latestMessageOutputIndex, nil, events.SegmentTypeText), segmentText, latestMessageStatus))
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
							e.publishEvent(ctx, events.NewToolCallRequestedEvent(metadata, toolCorr(itemID, callID, outputIndex), callID, name, args))
							var b strings.Builder
							b.WriteString(args)
							finalCalls = append(finalCalls, responsesPendingCall{callID: callID, name: name, itemID: itemID, outputIndex: outputIndex, status: status, args: b})
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
				appendAssistantChunk(itemID, latestMessageOutputIndex, d)
				log.Trace().Int("delta_len", len(d)).Int("message_len", len(message)).Msg("Responses: text delta")
			} else if tv, ok := m["text"].(map[string]any); ok {
				if d, ok := tv["delta"].(string); ok && d != "" {
					if itemID != "" {
						assistantByItem[itemID] += d
					}
					appendAssistantChunk(itemID, latestMessageOutputIndex, d)
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
				appendAssistantChunk(itemID, latestMessageOutputIndex, d)
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
					pc = &responsesPendingCall{itemID: itemID}
					callsByItem[itemID] = pc
				}
				if idx, ok := intFromProviderNumber(m["output_index"]); ok {
					pc.outputIndex = &idx
				}
				if d, ok := m["delta"].(string); ok && d != "" {
					pc.args.WriteString(d)
					e.publishEvent(ctx, events.NewToolCallArgumentsDeltaEvent(metadata, toolCorr(itemID, pc.callID, pc.outputIndex), toolCorr(itemID, pc.callID, pc.outputIndex).ToolCallID, d, pc.args.String(), 0))
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
			responseCompleted = true
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
	terminal := consumeResponsesSSE(ctx, reader, tap, &eventName, &dataBuf, flush)
	if terminal.Kind == responsesStreamTerminalError && streamErr == nil {
		streamErr = terminal.Err
	}
	if streamErr != nil {
		terminal = responsesStreamTerminal{Kind: responsesStreamTerminalError, Err: streamErr}
		log.Debug().Err(streamErr).Msg("Responses: stream ended with provider error")
	}
	state := &responsesStreamState{
		reqBody:                          reqBody,
		providerCallCorr:                 providerCallCorr,
		tap:                              tap,
		message:                          message,
		inputTokens:                      inputTokens,
		outputTokens:                     outputTokens,
		cachedTokens:                     cachedTokens,
		reasoningTokens:                  reasoningTokens,
		stopReason:                       stopReason,
		responseCompleted:                responseCompleted,
		streamErr:                        streamErr,
		currentResponseID:                currentResponseID,
		finalCalls:                       finalCalls,
		currentReasoningItemID:           currentReasoningItemID,
		lastReasoningItemID:              lastReasoningItemID,
		currentReasoningOutputIndex:      currentReasoningOutputIndex,
		lastReasoningOutputIndex:         lastReasoningOutputIndex,
		currentReasoningSummaryIndex:     currentReasoningSummaryIndex,
		lastReasoningSummaryIndex:        lastReasoningSummaryIndex,
		currentReasoningStatus:           currentReasoningStatus,
		latestMessageItemID:              latestMessageItemID,
		latestMessageOutputIndex:         latestMessageOutputIndex,
		latestMessageStatus:              latestMessageStatus,
		assistantByItem:                  assistantByItem,
		callsByItem:                      callsByItem,
		currentReasoningEncryptedContent: currentReasoningEncryptedContent,
	}
	state.thinkBuf.WriteString(thinkBuf.String())
	state.sayBuf.WriteString(sayBuf.String())
	state.summaryBuf.WriteString(summaryBuf.String())
	state.currentReasoningText.WriteString(currentReasoningText.String())
	state.currentReasoningSummary.WriteString(currentReasoningSummary.String())

	metadata = finalizeResponsesStreamMetadata(metadata, state, startTime, terminal)
	if state.summaryBuf.Len() > 0 {
		// Publish a friendly info event with the complete summary and provider identity when available.
		data := providerData("openai_responses", state.currentResponseID, state.lastReasoningItemID, state.lastReasoningOutputIndex, state.lastReasoningSummaryIndex)
		if data == nil {
			data = map[string]any{}
		}
		data["text"] = state.summaryBuf.String()
		e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary", data))
	}

	includeToolCalls := terminal.Kind == responsesStreamTerminalEOF
	toolCallCount := appendResponsesFinalTurnBlocks(t, state, includeToolCalls)
	persistResponsesInferenceResult(t, metadata, responsesInferenceProvider(e.settings), includeToolCalls && toolCallCount > 0)
	finishClass := responsesFinishClass(state, terminal, toolCallCount)
	stopReasonValue := ""
	if metadata.StopReason != nil {
		stopReasonValue = *metadata.StopReason
	}
	e.publishEvent(ctx, events.NewProviderCallFinishedEvent(metadata, state.providerCallCorrelation(), stopReasonValue, finishClass, metadata.Usage, metadata.DurationMs, includeToolCalls && toolCallCount > 0))
	if terminal.Err != nil {
		return t, terminal.Err
	}
	return t, nil

}

func consumeResponsesSSE(
	ctx context.Context,
	reader *bufio.Reader,
	tap engine.DebugTap,
	eventName *string,
	dataBuf *strings.Builder,
	flush func() error,
) responsesStreamTerminal {
	for {
		select {
		case <-ctx.Done():
			return responsesStreamTerminal{Kind: responsesStreamTerminalCancelled, Err: ctx.Err()}
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			log.Debug().Err(err).Msg("Responses: error reading SSE line")
			return responsesStreamTerminal{Kind: responsesStreamTerminalError, Err: err}
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			_ = flush()
			*eventName = ""
			if err == io.EOF {
				return responsesStreamTerminal{Kind: responsesStreamTerminalEOF}
			}
			continue
		}
		if strings.HasPrefix(line, "event:") {
			*eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(data)
			if tap != nil {
				tap.OnSSE(*eventName, []byte(data))
			}
			if err == io.EOF {
				_ = flush()
				return responsesStreamTerminal{Kind: responsesStreamTerminalEOF}
			}
			continue
		}
		if err == io.EOF {
			_ = flush()
			return responsesStreamTerminal{Kind: responsesStreamTerminalEOF}
		}
	}
}

func streamKindForResponsesSegment(segmentType string) string {
	switch segmentType {
	case events.SegmentTypeText:
		return events.StreamKindContent
	case events.SegmentTypeReasoning:
		return events.StreamKindReasoning
	case events.SegmentTypeTool:
		return events.StreamKindToolCall
	default:
		return ""
	}
}
