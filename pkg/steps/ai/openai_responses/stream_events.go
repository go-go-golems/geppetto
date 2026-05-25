package openai_responses

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/streamhelpers"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
)

func normalizeResponsesEventName(eventName string) string {
	switch eventName {
	case "response.reasoning.delta":
		return "response.reasoning_text.delta"
	case "response.reasoning.done":
		return "response.reasoning_text.done"
	default:
		return eventName
	}
}

func toInt(v any) (int, bool) {
	const maxInt = int(^uint(0) >> 1)
	const minInt = -maxInt - 1

	switch x := v.(type) {
	case float64:
		if x > float64(maxInt) || x < float64(minInt) {
			return 0, false
		}
		return int(x), true
	case float32:
		f := float64(x)
		if f > float64(maxInt) || f < float64(minInt) {
			return 0, false
		}
		return int(x), true
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		if x > int64(maxInt) || x < int64(minInt) {
			return 0, false
		}
		return int(x), true
	case uint:
		if uint64(x) > uint64(maxInt) {
			return 0, false
		}
		return int(x), true
	case uint32:
		if uint64(x) > uint64(maxInt) {
			return 0, false
		}
		return int(x), true
	case uint64:
		if x > uint64(maxInt) {
			return 0, false
		}
		return int(x), true
	default:
		return 0, false
	}
}

func (e *Engine) handleResponsesProviderEvent(
	ctx context.Context,
	t *turns.Turn,
	metadata events.EventMetadata,
	reqBody responsesRequest,
	tap engine.DebugTap,
	streamState *responsesStreamState,
	eventName string,
	providerEventType string,
	raw string,
	m map[string]any,
) {
	responsesSegmentCorr := func(itemID string, outputIndex, summaryIndex *int, segmentType string) events.Correlation {
		return streamState.segmentCorrelation(itemID, outputIndex, summaryIndex, segmentType)
	}
	toolCorr := func(itemID, callID string, outputIndex *int) events.Correlation {
		return streamState.toolCorrelation(itemID, callID, outputIndex)
	}
	appendAssistantChunk := func(itemID string, outputIndex *int, chunk string) {
		if chunk == "" {
			return
		}
		streamState.message += chunk
		streamState.sayBuf.WriteString(chunk)
		text := streamState.message
		if itemID != "" && streamState.assistantByItem[itemID] != "" {
			text = streamState.assistantByItem[itemID]
		}
		e.publishEvent(ctx, events.NewTextDeltaEvent(metadata, responsesSegmentCorr(itemID, outputIndex, nil, events.SegmentTypeText), chunk, text, 0))
	}
	backfillAssistantChunk := func(itemID, fullChunk string) {
		if fullChunk == "" {
			return
		}
		current := streamState.message
		if itemID != "" {
			if streamed := streamState.assistantByItem[itemID]; streamed != "" {
				current = streamed
			}
		}
		missing := missingProviderSuffix(current, fullChunk)
		if missing == "" {
			return
		}
		if itemID != "" {
			streamState.assistantByItem[itemID] += missing
		}
		appendAssistantChunk(itemID, streamState.latestMessageOutputIndex, missing)
	}
	reasoningSourceForSummaryIndex := func(summaryIndex *int) string {
		if summaryIndex != nil {
			return "summary"
		}
		return "thinking"
	}
	backfillReasoningText := func(fullText string) {
		if fullText == "" {
			return
		}
		current := streamState.currentReasoningText.String()
		missing := missingProviderSuffix(current, fullText)
		if missing == "" {
			return
		}
		streamState.currentReasoningText.WriteString(missing)
		normalized := streamhelpers.NormalizeReasoningDelta(streamState.thinkBuf.String(), missing)
		streamState.thinkBuf.WriteString(normalized)
		e.publishEvent(ctx, events.NewReasoningDeltaEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "thinking", normalized, streamState.thinkBuf.String(), 0))
	}
	switch providerEventType {
	case "response.output_item.added":
		if it, ok := m["item"].(map[string]any); ok {
			if typ, ok := it["type"].(string); ok {
				switch typ {
				case "reasoning":
					streamState.currentReasoningItemID = ""
					streamState.currentReasoningText.Reset()
					streamState.currentReasoningSummary.Reset()
					streamState.currentReasoningEncryptedContent = ""
					streamState.currentReasoningOutputIndex = nil
					streamState.currentReasoningSummaryIndex = nil
					streamState.currentReasoningStatus = ""
					if v, ok := it["id"].(string); ok && v != "" {
						streamState.currentReasoningItemID = v
					}
					if status, ok := it["status"].(string); ok && status != "" {
						streamState.currentReasoningStatus = status
					}
					if idx, ok := intFromProviderNumber(m["output_index"]); ok {
						streamState.currentReasoningOutputIndex = &idx
					}
					e.publishEvent(ctx, events.NewReasoningSegmentStartedEvent(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "provider"))
					// Capture encrypted reasoning content when present.
					if enc, ok := it["encrypted_content"].(string); ok && enc != "" {
						streamState.currentReasoningEncryptedContent = enc
					}
				case "message":
					if v, ok := it["id"].(string); ok && v != "" {
						streamState.latestMessageItemID = v
					}
					if status, ok := it["status"].(string); ok && status != "" {
						streamState.latestMessageStatus = status
					}
					if idx, ok := intFromProviderNumber(m["output_index"]); ok {
						streamState.latestMessageOutputIndex = &idx
					}
					e.publishEvent(ctx, events.NewTextSegmentStartedEvent(metadata, responsesSegmentCorr(streamState.latestMessageItemID, streamState.latestMessageOutputIndex, nil, events.SegmentTypeText), "assistant"))
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
						streamState.callsByItem[itemID] = &responsesPendingCall{callID: callID, name: name, itemID: itemID, outputIndex: outputIndex}
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
				streamState.streamErr = fmt.Errorf("responses stream error (%s): %s", codeStr, msgStr)
			} else {
				streamState.streamErr = errors.New(msgStr)
			}
		} else {
			streamState.streamErr = fmt.Errorf("responses stream error")
		}
		e.publishEvent(ctx, events.NewErrorEvent(metadata, streamState.streamErr))
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
					streamState.streamErr = fmt.Errorf("responses failed (%s): %s", codeStr, msgStr)
				} else {
					streamState.streamErr = errors.New(msgStr)
				}
			} else {
				streamState.streamErr = fmt.Errorf("responses failed")
			}
		} else {
			streamState.streamErr = fmt.Errorf("responses failed")
		}
		e.publishEvent(ctx, events.NewErrorEvent(metadata, streamState.streamErr))
		if tap != nil {
			tap.OnProviderObject("response.failed", m)
		}
	case "response.reasoning_summary_part.added":
		if itemID := itemIDFromProviderObject(m); itemID != "" {
			streamState.currentReasoningItemID = itemID
		}
		if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
			streamState.currentReasoningSummaryIndex = &idx
			streamState.lastReasoningSummaryIndex = &idx
		}
		// Start of a summary piece – forward as streaming info event
		e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary-started", providerData("openai_responses", streamState.currentResponseID, streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex)))
	case "response.reasoning_summary_text.delta":
		if itemID := itemIDFromProviderObject(m); itemID != "" {
			streamState.currentReasoningItemID = itemID
			streamState.lastReasoningItemID = itemID
		}
		if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
			streamState.currentReasoningSummaryIndex = &idx
			streamState.lastReasoningSummaryIndex = &idx
		}
		if v, ok := m["delta"].(string); ok && v != "" {
			before := streamState.summaryBuf.Len()
			normalized := streamhelpers.NormalizeReasoningSummaryDelta(streamState.summaryBuf.String(), v)
			e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, streamState.currentResponseID, providerEventType, m, len(v), len(normalized), before+len(normalized))
			streamState.summaryBuf.WriteString(normalized)
			streamState.currentReasoningSummary.WriteString(normalized)
			e.publishEvent(ctx, events.NewReasoningDeltaEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "summary", normalized, streamState.summaryBuf.String(), 0))
		} else if s, ok := m["text"].(string); ok && s != "" {
			before := streamState.summaryBuf.Len()
			normalized := streamhelpers.NormalizeReasoningSummaryDelta(streamState.summaryBuf.String(), s)
			e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, streamState.currentResponseID, providerEventType, m, len(s), len(normalized), before+len(normalized))
			streamState.summaryBuf.WriteString(normalized)
			streamState.currentReasoningSummary.WriteString(normalized)
			e.publishEvent(ctx, events.NewReasoningDeltaEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "summary", normalized, streamState.summaryBuf.String(), 0))
		}
	case "response.reasoning_summary_part.done":
		if itemID := itemIDFromProviderObject(m); itemID != "" {
			streamState.currentReasoningItemID = itemID
			streamState.lastReasoningItemID = itemID
		}
		if idx, ok := intFromProviderNumber(m["summary_index"]); ok {
			streamState.currentReasoningSummaryIndex = &idx
			streamState.lastReasoningSummaryIndex = &idx
		}
		// End of a summary piece – forward as streaming info event
		e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary-ended", providerData("openai_responses", streamState.currentResponseID, streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex)))
	case "response.reasoning_text.delta":
		if itemID := itemIDFromProviderObject(m); itemID != "" {
			streamState.currentReasoningItemID = itemID
			streamState.lastReasoningItemID = itemID
		}
		if idx, ok := intFromProviderNumber(m["output_index"]); ok {
			streamState.currentReasoningOutputIndex = &idx
			streamState.lastReasoningOutputIndex = &idx
		}
		if d, ok := m["delta"].(string); ok && d != "" {
			before := streamState.thinkBuf.Len()
			normalized := streamhelpers.NormalizeReasoningDelta(streamState.thinkBuf.String(), d)
			e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, streamState.currentResponseID, providerEventType, m, len(d), len(normalized), before+len(normalized))
			streamState.thinkBuf.WriteString(normalized)
			streamState.currentReasoningText.WriteString(d)
			e.publishEvent(ctx, events.NewReasoningDeltaEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "thinking", d, streamState.thinkBuf.String(), 0))
		} else if s, ok := m["text"].(string); ok && s != "" {
			before := streamState.thinkBuf.Len()
			normalized := streamhelpers.NormalizeReasoningDelta(streamState.thinkBuf.String(), s)
			e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, streamState.currentResponseID, providerEventType, m, len(s), len(normalized), before+len(normalized))
			streamState.thinkBuf.WriteString(normalized)
			streamState.currentReasoningText.WriteString(s)
			e.publishEvent(ctx, events.NewReasoningDeltaEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), "thinking", s, streamState.thinkBuf.String(), 0))
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
						streamState.currentReasoningItemID = itemID
						streamState.lastReasoningItemID = itemID
					}
					if idx, ok := intFromProviderNumber(m["output_index"]); ok {
						streamState.currentReasoningOutputIndex = &idx
						streamState.lastReasoningOutputIndex = &idx
					}
					// Append a reasoning block with encrypted content if present.
					rb := turns.Block{Kind: turns.BlockKindReasoning}
					payload := map[string]any{}
					if id, ok := it["id"].(string); ok && id != "" {
						rb.ID = id
						payload[turns.PayloadKeyItemID] = id
					}
					if streamState.currentReasoningItemID != "" && rb.ID == "" {
						rb.ID = streamState.currentReasoningItemID
						payload[turns.PayloadKeyItemID] = streamState.currentReasoningItemID
					}
					if status, ok := it["status"].(string); ok && status != "" {
						streamState.currentReasoningStatus = status
					}
					if idx, ok := intFromProviderNumber(m["output_index"]); ok {
						streamState.currentReasoningOutputIndex = &idx
						streamState.lastReasoningOutputIndex = &idx
					}
					if terminalText := reasoningTextFromProviderContent(it["content"]); terminalText != "" {
						backfillReasoningText(terminalText)
					}
					if text := strings.TrimSpace(streamState.currentReasoningText.String()); text != "" {
						payload[turns.PayloadKeyText] = text
					}
					enc := streamState.currentReasoningEncryptedContent
					if v, ok := it["encrypted_content"].(string); ok && v != "" {
						enc = v
					}
					if enc != "" {
						payload[turns.PayloadKeyEncryptedContent] = enc
					}
					summary := reasoningSummaryEntriesFromPayload(it)
					if len(summary) == 0 {
						summary = reasoningSummaryEntriesFromText(streamState.currentReasoningSummary.String())
					}
					if len(summary) > 0 {
						payload[turns.PayloadKeySummary] = summary
					}
					rb.Payload = payload
					setOpenAIResponsesBlockMetadata(&rb, streamState.currentResponseID, streamState.currentReasoningOutputIndex, "reasoning", streamState.currentReasoningStatus)
					turns.AppendBlock(t, rb)
					finalReasoningText := strings.TrimSpace(streamState.currentReasoningText.String())
					finalReasoningStatus := streamState.currentReasoningStatus
					e.publishEvent(ctx, events.NewReasoningSegmentFinishedEventWithSource(metadata, responsesSegmentCorr(streamState.currentReasoningItemID, streamState.currentReasoningOutputIndex, streamState.currentReasoningSummaryIndex, events.SegmentTypeReasoning), reasoningSourceForSummaryIndex(streamState.currentReasoningSummaryIndex), finalReasoningText, finalReasoningStatus))
					streamState.currentReasoningItemID = ""
					streamState.currentReasoningText.Reset()
					streamState.currentReasoningSummary.Reset()
					streamState.currentReasoningEncryptedContent = ""
					streamState.currentReasoningOutputIndex = nil
					streamState.currentReasoningSummaryIndex = nil
					streamState.currentReasoningStatus = ""
					if tap != nil {
						tap.OnProviderObject("output.reasoning", it)
					}
				case "message":
					itemID := ""
					if v, ok := it["id"].(string); ok && v != "" {
						streamState.latestMessageItemID = v
						itemID = v
					}
					if status, ok := it["status"].(string); ok && status != "" {
						streamState.latestMessageStatus = status
					}
					if idx, ok := intFromProviderNumber(m["output_index"]); ok {
						streamState.latestMessageOutputIndex = &idx
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
								backfillAssistantChunk(itemID, responsesChunkFromValue(content["json"]))
							}
						}
					}
					segmentText := streamState.message
					if itemID != "" && streamState.assistantByItem[itemID] != "" {
						segmentText = streamState.assistantByItem[itemID]
					}
					e.publishEvent(ctx, events.NewTextSegmentFinishedEvent(metadata, responsesSegmentCorr(itemID, streamState.latestMessageOutputIndex, nil, events.SegmentTypeText), segmentText, streamState.latestMessageStatus))
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
					if pc := streamState.callsByItem[itemID]; pc != nil {
						if callID == "" {
							callID = pc.callID
						}
						if name == "" {
							name = pc.name
						}
						if outputIndex == nil {
							outputIndex = pc.outputIndex
						}
						if status == "" {
							status = pc.status
						}
						if args == "" {
							args = pc.args.String()
						}
					}
					if callID != "" && name != "" {
						e.publishEvent(ctx, events.NewToolCallRequestedEvent(metadata, toolCorr(itemID, callID, outputIndex), callID, name, args))
						var b strings.Builder
						b.WriteString(args)
						streamState.finalCalls = append(streamState.finalCalls, responsesPendingCall{callID: callID, name: name, itemID: itemID, outputIndex: outputIndex, status: status, args: b})
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
				streamState.assistantByItem[itemID] += d
			}
			appendAssistantChunk(itemID, streamState.latestMessageOutputIndex, d)
			log.Trace().Int("delta_len", len(d)).Int("message_len", len(streamState.message)).Msg("Responses: text delta")
		} else if tv, ok := m["text"].(map[string]any); ok {
			if d, ok := tv["delta"].(string); ok && d != "" {
				if itemID != "" {
					streamState.assistantByItem[itemID] += d
				}
				appendAssistantChunk(itemID, streamState.latestMessageOutputIndex, d)
				log.Trace().Int("delta_len", len(d)).Int("message_len", len(streamState.message)).Msg("Responses: text delta (nested)")
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
				streamState.assistantByItem[itemID] += d
			}
			appendAssistantChunk(itemID, streamState.latestMessageOutputIndex, d)
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
			backfillAssistantChunk(itemID, responsesChunkFromValue(j))
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
			pc := streamState.callsByItem[itemID]
			if pc == nil {
				pc = &responsesPendingCall{itemID: itemID}
				streamState.callsByItem[itemID] = pc
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
			if pc := streamState.callsByItem[itemID]; pc != nil {
				pc.args.Reset()
				pc.args.WriteString(d)
			}
		}
	// No assistant text in this event; only arguments aggregation
	case "response.completed":
		streamState.responseCompleted = true
		if totals, ok := parseUsageTotalsFromEnvelope(m); ok {
			streamState.inputTokens = totals.inputTokens
			streamState.outputTokens = totals.outputTokens
			streamState.cachedTokens = totals.cachedTokens
			streamState.reasoningTokens = totals.reasoningTokens
		}
		// optional stop reason, sometimes nested
		if sr, ok := m["stop_reason"].(string); ok && sr != "" {
			streamState.stopReason = &sr
		} else if respObj, ok := m["response"].(map[string]any); ok {
			if sr, ok := respObj["stop_reason"].(string); ok && sr != "" {
				streamState.stopReason = &sr
			}
		}
		if tap != nil {
			tap.OnProviderObject("response.completed", m)
		}
	}

}
