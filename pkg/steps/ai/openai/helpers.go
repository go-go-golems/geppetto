package openai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	go_openai "github.com/sashabaranov/go-openai"
)

func GetToolCallString(toolCalls []go_openai.ToolCall) string {
	msg := ""
	for _, call := range toolCalls {
		msg += call.Function.Name
		msg += call.Function.Arguments
	}

	return msg
}

type ToolCallMerger struct {
	toolCalls map[int]go_openai.ToolCall
}

func NewToolCallMerger() *ToolCallMerger {
	return &ToolCallMerger{
		toolCalls: make(map[int]go_openai.ToolCall),
	}
}

func (tcm *ToolCallMerger) AddToolCalls(toolCalls []go_openai.ToolCall) {
	for _, call := range toolCalls {
		index := 0
		if call.Index != nil {
			index = *call.Index
		}
		if existing, found := tcm.toolCalls[index]; found {
			existing.Function.Name += call.Function.Name
			existing.Function.Arguments += call.Function.Arguments
			tcm.toolCalls[index] = existing
		} else {
			tcm.toolCalls[index] = call
		}
	}
}

func (tcm *ToolCallMerger) GetToolCalls() []go_openai.ToolCall {
	var result []go_openai.ToolCall
	for _, call := range tcm.toolCalls {
		result = append(result, call)
	}

	return result
}

func IsOpenAiEngine(engine string) bool {
	if strings.HasPrefix(engine, "gpt") {
		return true
	}
	if strings.HasPrefix(engine, "text-") {
		return true
	}

	return false
}

func isReasoningModel(engine string) bool {
	m := strings.ToLower(strings.TrimSpace(engine))
	return strings.HasPrefix(m, "o1") ||
		strings.HasPrefix(m, "o3") ||
		strings.HasPrefix(m, "o4") ||
		strings.HasPrefix(m, "gpt-5")
}

// Removed obsolete MakeCompletionRequest (conversation-based)

// MakeCompletionRequestFromTurn builds an OpenAI ChatCompletionRequest directly from a Turn's blocks,
// avoiding any dependency on conversation.Conversation.
func MakeCompletionRequestFromTurn(
	settings *settings.StepSettings,
	t *turns.Turn,
) (*go_openai.ChatCompletionRequest, error) {
	if settings.Client == nil {
		return nil, steps.ErrMissingClientSettings
	}
	if settings.OpenAI == nil {
		return nil, errors.New("no openai settings")
	}

	chatSettings := settings.Chat
	engine := ""
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	var msgs_ []go_openai.ChatCompletionMessage

	// Accumulate tool-calls and ensure any tool results are placed immediately after
	pendingToolCalls := []go_openai.ToolCall{}
	toolPhaseActive := false // true after flushing assistant tool_calls, until we exit tool result sequence
	delayedChats := []go_openai.ChatCompletionMessage{}
	// Track expected tool_call ids after a flush so we do not end the tool phase
	// until all of them have corresponding tool messages. This prevents interleaving
	// user/system messages between tool results when external middleware injects blocks.
	expectedToolIDs := map[string]bool{}
	remainingExpected := 0
	flushToolCalls := func() {
		if len(pendingToolCalls) == 0 {
			return
		}
		msgs_ = append(msgs_, go_openai.ChatCompletionMessage{
			Role:      "assistant",
			ToolCalls: pendingToolCalls,
		})
		// Enter tool phase and prepare expected ids; clear pending calls so we don't re-emit them later
		expectedToolIDs = map[string]bool{}
		for _, tc := range pendingToolCalls {
			if tc.ID != "" {
				expectedToolIDs[tc.ID] = false
			}
		}
		remainingExpected = len(pendingToolCalls)
		log.Debug().Int("expected_tool_uses", remainingExpected).Msg("OpenAI request: flushed assistant tool_calls; starting tool phase")
		toolPhaseActive = true
		pendingToolCalls = nil
	}
	endToolPhase := func() {
		if toolPhaseActive {
			// After finishing tool_use sequence, emit any delayed chat messages
			if len(delayedChats) > 0 {
				msgs_ = append(msgs_, delayedChats...)
				delayedChats = nil
			}
		}
		toolPhaseActive = false
		pendingToolCalls = nil
		expectedToolIDs = map[string]bool{}
		remainingExpected = 0
	}

	if t != nil {
		for _, b := range t.Blocks {
			switch b.Kind {
			case turns.BlockKindReasoning:
				// Skip reasoning blocks in ChatCompletions requests; only Responses API understands them.
				continue
			case turns.BlockKindUser, turns.BlockKindLLMText, turns.BlockKindSystem:
				// If we have pending tool calls but haven't emitted tool results yet,
				// delay chat messages until after tool_use messages to satisfy provider ordering.
				text := ""
				if v, ok := b.Payload[turns.PayloadKeyText]; ok {
					switch sv := v.(type) {
					case string:
						text = strings.TrimSpace(sv)
					case []byte:
						text = strings.TrimSpace(string(sv))
					default:
						bb, _ := json.Marshal(v)
						text = strings.TrimSpace(string(bb))
					}
				}
				if text == "" {
					log.Debug().Str("role", b.Role).Msg("OpenAI request: skipping empty text block")
					continue
				}
				var role string
				switch b.Kind {
				case turns.BlockKindUser:
					role = "user"
				case turns.BlockKindSystem:
					role = "system"
				case turns.BlockKindLLMText:
					role = "assistant"
				case turns.BlockKindToolCall:
					role = "assistant"
				case turns.BlockKindToolUse:
					role = "tool"
				case turns.BlockKindOther:
					role = "assistant"
				case turns.BlockKindReasoning:
					role = "assistant"
				}
				// Check for images array in payload to construct MultiContent
				var msg go_openai.ChatCompletionMessage
				if imgs, ok := b.Payload[turns.PayloadKeyImages].([]map[string]any); ok && len(imgs) > 0 {
					parts := []go_openai.ChatMessagePart{{Type: go_openai.ChatMessagePartTypeText, Text: text}}
					for _, img := range imgs {
						mediaType, _ := img["media_type"].(string)
						url, _ := img["url"].(string)
						// content can be []byte or base64 string
						var base64Content string
						if raw, ok := img["content"]; ok && raw != nil {
							switch rv := raw.(type) {
							case []byte:
								base64Content = base64.StdEncoding.EncodeToString(rv)
							case string:
								// assume already base64
								base64Content = rv
							}
						}
						imageURL := url
						if imageURL == "" && base64Content != "" {
							imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, base64Content)
						}
						parts = append(parts, go_openai.ChatMessagePart{
							Type: go_openai.ChatMessagePartTypeImageURL,
							ImageURL: &go_openai.ChatMessageImageURL{
								URL:    imageURL,
								Detail: go_openai.ImageURLDetailAuto,
							},
						})
					}
					msg = go_openai.ChatCompletionMessage{Role: role, MultiContent: parts}
				} else {
					msg = go_openai.ChatCompletionMessage{Role: role, Content: text}
				}
				if len(pendingToolCalls) > 0 && !toolPhaseActive {
					// Buffer until we can place after tool_use
					delayedChats = append(delayedChats, msg)
				} else if toolPhaseActive {
					// We're in tool phase (emitting tool results), buffer to after tool results
					delayedChats = append(delayedChats, msg)
				} else {
					msgs_ = append(msgs_, msg)
				}

			case turns.BlockKindToolCall:
				name := ""
				if v, ok := b.Payload[turns.PayloadKeyName]; ok {
					_ = assignString(&name, v)
				}
				toolID := ""
				if v, ok := b.Payload[turns.PayloadKeyID]; ok {
					_ = assignString(&toolID, v)
				}
				argsStr := "{}"
				if v, ok := b.Payload[turns.PayloadKeyArgs]; ok && v != nil {
					switch tv := v.(type) {
					case string:
						if strings.TrimSpace(tv) != "" {
							argsStr = tv
						}
					case json.RawMessage:
						if len(tv) > 0 {
							argsStr = string(tv)
						}
					default:
						if bb, err := json.Marshal(v); err == nil {
							argsStr = string(bb)
						}
					}
				}
				// Debug: record detected tool_call block
				log.Debug().
					Str("tool_id", toolID).
					Str("name", name).
					Int("args_len", len(argsStr)).
					Msg("OpenAI request: encountered tool_call block")
				// Start or continue accumulating tool calls; reset tool phase if starting a new group
				if !toolPhaseActive && len(pendingToolCalls) == 0 {
					delayedChats = nil
				}
				pendingToolCalls = append(pendingToolCalls, go_openai.ToolCall{
					ID:   toolID,
					Type: go_openai.ToolTypeFunction,
					Function: go_openai.FunctionCall{
						Name:      name,
						Arguments: argsStr,
					},
				})

			case turns.BlockKindToolUse:
				// Tool results must immediately follow assistant tool_calls
				flushToolCalls()
				toolID := ""
				_ = assignString(&toolID, b.Payload[turns.PayloadKeyID])
				result := toolUsePayloadToJSONString(b.Payload)
				// Debug: record detected tool_use block
				log.Debug().
					Str("tool_id", toolID).
					Int("result_len", len(result)).
					Msg("OpenAI request: encountered tool_use block")
				msgs_ = append(msgs_, go_openai.ChatCompletionMessage{
					Role:       "tool",
					Content:    result,
					ToolCallID: toolID,
				})
				// Mark this tool id as satisfied if it was expected
				if toolPhaseActive {
					if _, ok := expectedToolIDs[toolID]; ok && !expectedToolIDs[toolID] {
						expectedToolIDs[toolID] = true
						if remainingExpected > 0 {
							remainingExpected--
						}
						log.Debug().Str("tool_id", toolID).Int("remaining_expected", remainingExpected).Msg("OpenAI request: recorded tool_use for expected id")
					}
					// If all expected tool results have arrived, end the tool phase now
					if remainingExpected == 0 {
						endToolPhase()
					}
				}
				// Do not end tool phase due to control-flow here; we handle it via remainingExpected
			case turns.BlockKindOther:
				// Ignore unknown blocks unless they carry text
				if v, ok := b.Payload[turns.PayloadKeyText]; ok {
					text := ""
					_ = assignString(&text, v)
					if text != "" {
						msg := go_openai.ChatCompletionMessage{Role: "assistant", Content: text}
						if len(pendingToolCalls) > 0 || toolPhaseActive {
							delayedChats = append(delayedChats, msg)
						} else {
							msgs_ = append(msgs_, msg)
						}
					}
				}
			}
			// Only end the tool phase if there are no pending calls and no remaining expected tool ids
			if toolPhaseActive {
				if b.Kind != turns.BlockKindToolUse && len(pendingToolCalls) == 0 && remainingExpected == 0 {
					endToolPhase()
				}
			}
		}
	}

	// Flush any remaining tool calls (normally none, since we clear after flush)
	if len(pendingToolCalls) > 0 {
		flushToolCalls()
	}
	// End tool phase and emit any delayed chats if needed
	endToolPhase()

	// Copy of parameter handling from MakeCompletionRequest
	temperature := 0.0
	if chatSettings.Temperature != nil {
		temperature = *chatSettings.Temperature
	}
	topP := 0.0
	if chatSettings.TopP != nil {
		topP = *chatSettings.TopP
	}
	maxTokens := 32
	if chatSettings.MaxResponseTokens != nil {
		maxTokens = *chatSettings.MaxResponseTokens
	}
	maxCompletionTokens := 0
	n := 1
	if settings.OpenAI.N != nil {
		n = *settings.OpenAI.N
	}
	stream := chatSettings.Stream
	stop := chatSettings.Stop
	presencePenalty := 0.0
	if settings.OpenAI.PresencePenalty != nil {
		presencePenalty = *settings.OpenAI.PresencePenalty
	}
	frequencyPenalty := 0.0
	if settings.OpenAI.FrequencyPenalty != nil {
		frequencyPenalty = *settings.OpenAI.FrequencyPenalty
	}

	if isReasoningModel(engine) {
		maxCompletionTokens = maxTokens
		maxTokens = 0
		temperature = 0
		topP = 0
		n = 1
		presencePenalty = 0
		frequencyPenalty = 0
	}

	log.Debug().
		Str("model", engine).
		Int("max_tokens", maxTokens).
		Int("max_completion_tokens", maxCompletionTokens).
		Float64("temperature", temperature).
		Float64("top_p", topP).
		Int("n", n).
		Bool("stream", stream).
		Strs("stop", stop).
		Float64("presence_penalty", presencePenalty).
		Float64("frequency_penalty", frequencyPenalty).
		Msg("Making request to openai from turn blocks")

	// Debug: summarize the final message sequence for adjacency and content previews
	for i, m := range msgs_ {
		// tool_call ids within assistant message
		var toolIDs []string
		for _, tc := range m.ToolCalls {
			if tc.ID != "" {
				toolIDs = append(toolIDs, tc.ID)
			}
		}
		content := m.Content
		if content == "" && len(m.MultiContent) > 0 {
			// If multi content, show text parts concatenated for preview
			var parts []string
			for _, p := range m.MultiContent {
				if p.Type == go_openai.ChatMessagePartTypeText && p.Text != "" {
					parts = append(parts, p.Text)
				}
			}
			content = strings.Join(parts, " ")
		}
		preview := content
		if len(preview) > 160 {
			preview = preview[:160] + "…"
		}
		log.Debug().
			Int("idx", i).
			Str("role", m.Role).
			Int("tool_call_count", len(m.ToolCalls)).
			Strs("tool_call_ids", toolIDs).
			Str("tool_call_id", m.ToolCallID).
			Str("content_preview", preview).
			Msg("OpenAI request message")
	}

	var streamOptions *go_openai.StreamOptions
	if stream && !strings.Contains(engine, "mistral") {
		streamOptions = &go_openai.StreamOptions{IncludeUsage: true}
	}

	req := go_openai.ChatCompletionRequest{
		Model:               engine,
		Messages:            msgs_,
		MaxTokens:           maxTokens,
		MaxCompletionTokens: maxCompletionTokens,
		Temperature:         float32(temperature),
		TopP:                float32(topP),
		N:                   n,
		Stream:              stream,
		Stop:                stop,
		StreamOptions:       streamOptions,
		PresencePenalty:     float32(presencePenalty),
		FrequencyPenalty:    float32(frequencyPenalty),
		LogitBias:           nil,
	}

	// Apply provider-native structured output schema when configured.
	if chatSettings.IsStructuredOutputEnabled() {
		cfg, err := chatSettings.StructuredOutputConfig()
		if err != nil {
			if chatSettings.StructuredOutputRequireValid {
				return nil, err
			}
			log.Warn().Err(err).Msg("OpenAI request: ignoring invalid structured output configuration")
		} else if cfg != nil {
			schemaBytes, err := json.Marshal(cfg.Schema)
			if err != nil {
				if chatSettings.StructuredOutputRequireValid {
					return nil, errors.Wrap(err, "marshal structured output schema")
				}
				log.Warn().Err(err).Msg("OpenAI request: ignoring non-serializable structured output schema")
			} else {
				req.ResponseFormat = &go_openai.ChatCompletionResponseFormat{
					Type: go_openai.ChatCompletionResponseFormatTypeJSONSchema,
					JSONSchema: &go_openai.ChatCompletionResponseFormatJSONSchema{
						Name:        cfg.Name,
						Description: cfg.Description,
						Schema:      json.RawMessage(schemaBytes),
						Strict:      cfg.StrictOrDefault(),
					},
				}
			}
		}
	}

	return &req, nil
}

// assignString attempts to write a string representation of v into out.
func assignString(out *string, v interface{}) bool {
	if out == nil {
		return false
	}
	switch tv := v.(type) {
	case string:
		*out = tv
		return true
	case []byte:
		*out = string(tv)
		return true
	default:
		bb, err := json.Marshal(v)
		if err == nil {
			*out = string(bb)
			return true
		}
	}
	return false
}

func toolUsePayloadToJSONString(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	resultVal := payload[turns.PayloadKeyResult]
	errStr, _ := payload[turns.PayloadKeyError].(string)
	if errStr == "" {
		return anyToJSONString(resultVal)
	}

	out := map[string]any{"error": errStr}
	if resultVal != nil {
		if s, ok := resultVal.(string); ok {
			var obj any
			if json.Unmarshal([]byte(s), &obj) == nil {
				out["result"] = obj
			} else {
				out["result"] = s
			}
		} else {
			out["result"] = resultVal
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, errStr)
	}
	return string(b)
}

func anyToJSONString(v any) string {
	if v == nil {
		return ""
	}
	switch tv := v.(type) {
	case string:
		return tv
	case []byte:
		return string(tv)
	default:
		if bb, err := json.Marshal(v); err == nil {
			return string(bb)
		}
		return fmt.Sprintf("%v", v)
	}
}

func MakeClient(apiSettings *settings.APISettings, apiType ai_types.ApiType) (*go_openai.Client, error) {
	apiKey, ok := apiSettings.APIKeys[string(apiType)+"-api-key"]
	if !ok {
		return nil, errors.Errorf("no API key for %s", apiType)
	}
	baseURL, ok := apiSettings.BaseUrls[string(apiType)+"-base-url"]
	if !ok {
		return nil, errors.Errorf("no base URL for %s", apiType)
	}
	config := go_openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	client := go_openai.NewClientWithConfig(config)
	return client, nil
}

// TODO(manuel, 2024-06-25) We actually need a processor like the content block merger here, where we merge tool use blocks with previous messages to properly create openai messages
// So we would need to get a message block, then slurp up all the following tool use messages, and then finally emit the message block
// Removed obsolete messageToOpenAIMessage (conversation-based)
