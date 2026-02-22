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
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/serde"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Engine implements the Engine interface for OpenAI Responses API calls.
type Engine struct {
	settings *settings.StepSettings
}

type usageTotals struct {
	inputTokens     int
	outputTokens    int
	cachedTokens    int
	reasoningTokens int
}

func NewEngine(s *settings.StepSettings) (*Engine, error) {
	return &Engine{settings: s}, nil
}

// publishEvent publishes events to configured sinks and context sinks.
func (e *Engine) publishEvent(ctx context.Context, event events.Event) {
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

	// Attach tools to Responses request when present
	if t != nil {
		var engineTools []engine.ToolDefinition
		if reg, ok := tools.RegistryFrom(ctx); ok && reg != nil {
			for _, td := range reg.ListTools() {
				engineTools = append(engineTools, engine.ToolDefinition{
					Name:        td.Name,
					Description: td.Description,
					Parameters:  td.Parameters,
					Examples:    []engine.ToolExample{},
					Tags:        td.Tags,
					Version:     td.Version,
				})
			}
		}

		var toolCfg engine.ToolConfig
		if cfg, ok, err := engine.KeyToolConfig.Get(t.Data); err != nil {
			return nil, errors.Wrap(err, "get tool config")
		} else if ok {
			toolCfg = cfg
		}

		if len(engineTools) > 0 && toolCfg.Enabled {
			converted, err := e.PrepareToolsForResponses(engineTools, toolCfg)
			if err != nil {
				return nil, err
			}
			if arr, ok := converted.([]any); ok && len(arr) > 0 {
				reqBody.Tools = arr
				// Responses API: omit tool_choice for function tools to allow model selection
				reqBody.ToolChoice = nil
				// parallel_tool_calls preference
				if toolCfg.MaxParallelTools > 1 {
					b := true
					reqBody.ParallelToolCalls = &b
				} else if toolCfg.MaxParallelTools == 1 {
					b := false
					reqBody.ParallelToolCalls = &b
				}
				log.Debug().Int("tool_count", len(arr)).Interface("tool_choice", reqBody.ToolChoice).Msg("Responses: tools attached to request")
			}
		}
		// Optionally include server-side tools (Responses built-ins) when provided on the Turn
		if builtins, ok, err := turns.KeyResponsesServerTools.Get(t.Data); err != nil {
			return nil, errors.Wrap(err, "get responses server tools")
		} else if ok && len(builtins) > 0 {
			// Append alongside function tools
			reqBody.Tools = append(reqBody.Tools, builtins...)
			log.Debug().Int("builtin_tool_count", len(builtins)).Msg("Responses: server-side tools attached to request")
		}
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
		preview := make([]map[string]any, 0, len(reqBody.Input))
		for _, it := range reqBody.Input {
			pparts := make([]map[string]any, 0, len(it.Content))
			for _, c := range it.Content {
				seg := c.Text
				if len(seg) > 80 {
					seg = seg[:80] + "…"
				}
				pparts = append(pparts, map[string]any{"type": c.Type, "len": len(c.Text), "text": seg})
			}
			preview = append(preview, map[string]any{"role": it.Role, "parts": pparts})
		}
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
	if err := security.ValidateOutboundURL(url, security.OutboundURLOptions{
		AllowHTTP: false,
	}); err != nil {
		return nil, errors.Wrap(err, "invalid openai responses URL")
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
	log.Debug().Str("url", url).Int("body_len", len(b)).Bool("stream", reqBody.Stream).Msg("Responses: sending request")
	e.publishEvent(ctx, events.NewStartEvent(metadata))

	// Attach DebugTap if present on context
	var tap engine.DebugTap
	if t2, ok := engine.DebugTapFrom(ctx); ok {
		tap = t2
	}

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
		if tap != nil {
			tap.OnHTTP(req, b)
		}
		// #nosec G704 -- URL is validated above with ValidateOutboundURL.
		resp, err := http.DefaultClient.Do(req)
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
		var summaryBuf strings.Builder
		// Placeholder for potential future pairing of reasoning with assistant item id
		// (keep declared logic out until needed to avoid unused var)
		// Accumulate function_call tool uses
		type pendingCall struct {
			callID, name, itemID string
			args                 strings.Builder
		}
		callsByItem := map[string]*pendingCall{}
		finalCalls := []pendingCall{}
		// Track latest encrypted reasoning content observed during this response
		var latestEncryptedContent string
		var latestMessageItemID string
		log.Trace().Msg("Responses: starting SSE read loop")
		// Redact helper for sensitive fields when logging SSE payloads
		redactString := func(s string) string {
			if len(s) <= 12 {
				return "****"
			}
			// keep small prefix/suffix, hide middle
			pre := 6
			suf := 6
			if len(s) < pre+suf+1 {
				pre = len(s) / 2
				suf = len(s) - pre
			}
			return s[:pre] + "-****-" + s[len(s)-suf:]
		}
		var redact func(v any) any
		redact = func(v any) any {
			switch tv := v.(type) {
			case map[string]any:
				m2 := make(map[string]any, len(tv))
				for k, val := range tv {
					if k == "encrypted_content" {
						if s, ok := val.(string); ok {
							m2[k] = redactString(s)
							continue
						}
					}
					m2[k] = redact(val)
				}
				return m2
			case []any:
				arr := make([]any, len(tv))
				for i, el := range tv {
					arr[i] = redact(el)
				}
				return arr
			default:
				return v
			}
		}
		flush := func() error {
			if dataBuf.Len() == 0 {
				return nil
			}
			raw := dataBuf.String()
			dataBuf.Reset()
			var m map[string]any
			if err := json.Unmarshal([]byte(raw), &m); err != nil {
				log.Debug().Err(err).Str("event", eventName).Int("raw_len", len(raw)).Msg("Responses: failed to unmarshal SSE data")
				return nil
			}
			// Log redacted payload at trace level
			if zerolog.GlobalLevel() <= zerolog.TraceLevel {
				if rb, err := json.Marshal(redact(m)); err == nil {
					log.Trace().Str("event", eventName).RawJSON("data", rb).Msg("Responses: SSE event")
				}
			}
			appendAssistantChunk := func(chunk string) {
				if strings.TrimSpace(chunk) == "" {
					return
				}
				message += chunk
				sayBuf.WriteString(chunk)
				e.publishEvent(ctx, events.NewPartialCompletionEvent(metadata, chunk, message))
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
			switch eventName {
			case "response.output_item.added":
				if it, ok := m["item"].(map[string]any); ok {
					if typ, ok := it["type"].(string); ok {
						switch typ {
						case "reasoning":
							e.publishEvent(ctx, events.NewInfoEvent(metadata, "thinking-started", nil))
							// Capture encrypted reasoning content when present
							if enc, ok := it["encrypted_content"].(string); ok && enc != "" {
								latestEncryptedContent = enc
							}
						case "message":
							e.publishEvent(ctx, events.NewInfoEvent(metadata, "output-started", nil))
							if v, ok := it["id"].(string); ok && v != "" {
								latestMessageItemID = v
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
			case "response.reasoning_text.delta":
				if d, ok := m["delta"].(string); ok && d != "" {
					thinkBuf.WriteString(d)
					e.publishEvent(ctx, events.NewReasoningTextDelta(metadata, d))
					// Mirror to partial-thinking so existing UIs still render live reasoning text.
					e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, d, thinkBuf.String()))
				} else if s, ok := m["text"].(string); ok && s != "" {
					thinkBuf.WriteString(s)
					e.publishEvent(ctx, events.NewReasoningTextDelta(metadata, s))
					e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, s, thinkBuf.String()))
				}
			case "response.reasoning_text.done":
				fullText := thinkBuf.String()
				if s, ok := m["text"].(string); ok && s != "" {
					// Keep accumulated reasoning across multiple items. Done payloads
					// can repeat already-streamed deltas for the current item.
					if !strings.HasSuffix(fullText, s) {
						thinkBuf.WriteString(s)
						fullText = thinkBuf.String()
					}
				}
				e.publishEvent(ctx, events.NewReasoningTextDone(metadata, fullText))
			case "response.output_item.done":
				if it, ok := m["item"].(map[string]any); ok {
					if typ, ok := it["type"].(string); ok {
						switch typ {
						case "reasoning":
							e.publishEvent(ctx, events.NewInfoEvent(metadata, "thinking-ended", nil))
							// Append a reasoning block with encrypted content if present
							rb := turns.Block{Kind: turns.BlockKindReasoning}
							if id, ok := it["id"].(string); ok && id != "" {
								rb.ID = id
							}
							payload := map[string]any{}
							enc := latestEncryptedContent
							if v, ok := it["encrypted_content"].(string); ok && v != "" {
								enc = v
							}
							if enc != "" {
								payload[turns.PayloadKeyEncryptedContent] = enc
							}
							rb.Payload = payload
							turns.AppendBlock(t, rb)
							if tap != nil {
								tap.OnProviderObject("output.reasoning", it)
							}
						case "message":
							e.publishEvent(ctx, events.NewInfoEvent(metadata, "output-ended", nil))
							if v, ok := it["id"].(string); ok && v != "" {
								latestMessageItemID = v
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
											appendAssistantChunk(s)
										}
									case "output_json":
										appendAssistantChunk(chunkFromValue(content["json"]))
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
							if args == "" {
								if pc := callsByItem[itemID]; pc != nil {
									args = pc.args.String()
								}
							}
							if callID != "" && name != "" {
								e.publishEvent(ctx, events.NewToolCallEvent(metadata, events.ToolCall{ID: callID, Name: name, Input: args}))
								var b strings.Builder
								b.WriteString(args)
								finalCalls = append(finalCalls, pendingCall{callID: callID, name: name, itemID: itemID, args: b})
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
				if d, ok := m["delta"].(string); ok && d != "" {
					appendAssistantChunk(d)
					log.Trace().Int("delta_len", len(d)).Int("message_len", len(message)).Msg("Responses: text delta")
				} else if tv, ok := m["text"].(map[string]any); ok {
					if d, ok := tv["delta"].(string); ok && d != "" {
						appendAssistantChunk(d)
						log.Trace().Int("delta_len", len(d)).Int("message_len", len(message)).Msg("Responses: text delta (nested)")
					}
				}
				if tap != nil {
					tap.OnSSE(eventName, []byte(raw))
				}
			case "response.output_json.delta":
				if d, ok := m["delta"].(string); ok && d != "" {
					appendAssistantChunk(d)
				}
				if tap != nil {
					tap.OnSSE(eventName, []byte(raw))
				}
			case "response.output_json.done":
				if j, ok := m["json"]; ok {
					doneChunk := chunkFromValue(j)
					if doneChunk != "" && !strings.HasSuffix(message, doneChunk) {
						appendAssistantChunk(doneChunk)
					}
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
					log.Debug().
						Int("input_tokens", inputTokens).
						Int("output_tokens", outputTokens).
						Int("cached_tokens", cachedTokens).
						Int("reasoning_tokens", reasoningTokens).
						Msg("Responses: usage parsed")
				}
				// optional stop reason, sometimes nested
				if sr, ok := m["stop_reason"].(string); ok && sr != "" {
					stopReason = &sr
				} else if respObj, ok := m["response"].(map[string]any); ok {
					if sr, ok := respObj["stop_reason"].(string); ok && sr != "" {
						stopReason = &sr
					}
				}
				if stopReason != nil {
					log.Debug().Str("stop_reason", *stopReason).Msg("Responses: stop reason observed")
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
			if line != "" {
				preview := line
				if len(preview) > 200 {
					preview = preview[:200] + "…"
				}
				log.Trace().Str("line", preview).Msg("Responses: SSE line")
			}
			if line == "" {
				_ = flush()
				eventName = ""
				if err == io.EOF {
					log.Trace().Msg("Responses: EOF while reading SSE")
					break
				}
				continue
			}
			if strings.HasPrefix(line, "event:") {
				eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				log.Trace().Str("event", eventName).Msg("Responses: SSE event name")
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
					log.Trace().Msg("Responses: EOF while reading SSE")
					_ = flush()
					break
				}
				continue
			}
			if err == io.EOF {
				log.Trace().Msg("Responses: EOF while reading SSE")
				_ = flush()
				break
			}
		}
		log.Debug().Msg("Responses: SSE loop ended")
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
			// Publish a friendly info event with the complete summary
			e.publishEvent(ctx, events.NewInfoEvent(metadata, "reasoning-summary", map[string]any{"text": summaryBuf.String()}))
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
			turns.AppendBlock(t, b)
		}
		e.publishEvent(ctx, events.NewFinalEvent(metadata, message))
		return t, nil
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
	resp, err := http.DefaultClient.Do(req)
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
	for _, oi := range rr.Output {
		// Capture reasoning items (non-streaming)
		if oi.Type == "reasoning" {
			b := turns.Block{ID: oi.ID, Kind: turns.BlockKindReasoning, Payload: map[string]any{}}
			if oi.EncryptedContent != "" {
				b.Payload[turns.PayloadKeyEncryptedContent] = oi.EncryptedContent
			}
			turns.AppendBlock(t, b)
		}
		if oi.Type == "message" && oi.ID != "" {
			latestMessageItemID = oi.ID
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
	e.publishEvent(ctx, events.NewFinalEvent(metadata, message))
	return t, nil
}

func mustMarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func parseUsageTotalsFromEnvelope(envelope map[string]any) (usageTotals, bool) {
	if envelope == nil {
		return usageTotals{}, false
	}
	usage, ok := envelope["usage"].(map[string]any)
	if !ok {
		if respObj, hasResponse := envelope["response"].(map[string]any); hasResponse {
			usage, ok = respObj["usage"].(map[string]any)
		}
	}
	if !ok || usage == nil {
		return usageTotals{}, false
	}
	return parseUsageTotals(usage), true
}

func parseUsageTotalsFromResponse(resp responsesResponse) (usageTotals, bool) {
	if totals, ok := parseUsageTotalsFromRawUsage(resp.Usage); ok {
		return totals, true
	}
	if resp.Response != nil {
		if totals, ok := parseUsageTotalsFromRawUsage(resp.Response.Usage); ok {
			return totals, true
		}
	}
	return usageTotals{}, false
}

func parseUsageTotalsFromRawUsage(raw json.RawMessage) (usageTotals, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return usageTotals{}, false
	}
	var usage map[string]any
	if err := json.Unmarshal(raw, &usage); err != nil {
		return usageTotals{}, false
	}
	return parseUsageTotals(usage), true
}

func parseUsageTotals(usage map[string]any) usageTotals {
	ret := usageTotals{}
	if v, ok := toInt(usage["input_tokens"]); ok {
		ret.inputTokens = v
	}
	if v, ok := toInt(usage["output_tokens"]); ok {
		ret.outputTokens = v
	}
	if inputDetails, ok := usage["input_tokens_details"].(map[string]any); ok {
		if v, ok := toInt(inputDetails["cached_tokens"]); ok {
			ret.cachedTokens = v
		}
	} else if v, ok := toInt(usage["cached_tokens"]); ok {
		ret.cachedTokens = v
	}
	if outputDetails, ok := usage["output_tokens_details"].(map[string]any); ok {
		if v, ok := toInt(outputDetails["reasoning_tokens"]); ok {
			ret.reasoningTokens = v
		}
	} else if v, ok := toInt(usage["reasoning_tokens"]); ok {
		ret.reasoningTokens = v
	}
	return ret
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
