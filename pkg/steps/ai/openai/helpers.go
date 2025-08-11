package openai

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/go-go-golems/geppetto/pkg/conversation"
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

func MakeCompletionRequest(
	settings *settings.StepSettings,
	messages conversation.Conversation,
) (*go_openai.ChatCompletionRequest, error) {
	clientSettings := settings.Client
	if clientSettings == nil {
		return nil, steps.ErrMissingClientSettings
	}
	openaiSettings := settings.OpenAI
	if openaiSettings == nil {
		return nil, errors.New("no openai settings")
	}

	engine := ""

	chatSettings := settings.Chat
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	msgs_ := []go_openai.ChatCompletionMessage{}

	// Accumulate consecutive tool-use messages into a single assistant message with multiple tool_calls
	pendingToolCalls := []go_openai.ToolCall{}
	flushToolCalls := func() {
		if len(pendingToolCalls) == 0 {
			return
		}
		msgs_ = append(msgs_, go_openai.ChatCompletionMessage{
			Role:      string(conversation.RoleAssistant),
			ToolCalls: pendingToolCalls,
		})
		pendingToolCalls = nil
	}

	for _, msg := range messages {
		// Debug log type and metadata, but not full body
		msgType := string(msg.Content.ContentType())
		role := ""
		switch c := msg.Content.(type) {
        case *conversation.ChatMessageContent:
            role = string(c.Role)
            // Skip empty-text chat messages to avoid null content being sent to OpenAI
            if strings.TrimSpace(c.Text) == "" && len(c.Images) == 0 {
                log.Debug().Str("content_type", msgType).Str("role", role).Msg("OpenAI request: skipping empty chat message")
                continue
            }
			// Regular chat message flushes pending tool calls first
			flushToolCalls()
		case *conversation.ToolUseContent:
			// Log tool-use message id and name
			argPreview := string(c.Input)
			if len(argPreview) > 120 {
				argPreview = argPreview[:120] + "…"
			}
			log.Debug().
				Str("content_type", "tool-use").
				Str("tool_id", c.ToolID).
				Str("name", c.Name).
				Str("input_preview", argPreview).
				Msg("OpenAI request tool-use message")

			// Accumulate into a single assistant message with multiple tool_calls
			pendingToolCalls = append(pendingToolCalls, go_openai.ToolCall{
				ID:   c.ToolID,
				Type: go_openai.ToolTypeFunction,
				Function: go_openai.FunctionCall{
					Name:      c.Name,
					Arguments: string(c.Input),
				},
			})
			// Skip per-message conversion; continue to next message
			// without appending an individual assistant message
			metaKeys := []string{}
			if msg.Metadata != nil {
				for k := range msg.Metadata {
					metaKeys = append(metaKeys, k)
				}
			}
			log.Debug().
				Str("content_type", msgType).
				Str("role", role).
				Strs("meta_keys", metaKeys).
				Msg("OpenAI request message (batched tool-use)")
			continue
		case *conversation.ToolResultContent:
			// Log tool-result tool id only (no body)
			log.Debug().
				Str("content_type", "tool-result").
				Str("tool_id", c.ToolID).
				Msg("OpenAI request tool-result message")

			// Tool result messages must immediately follow the single assistant tool_calls message
			flushToolCalls()
		}
		metaKeys := []string{}
		if msg.Metadata != nil {
			for k := range msg.Metadata {
				metaKeys = append(metaKeys, k)
			}
		}
		log.Debug().
			Str("content_type", msgType).
			Str("role", role).
			Strs("meta_keys", metaKeys).
			Msg("OpenAI request message")

        msgs_ = append(msgs_, messageToOpenAIMessage(msg))
	}

	// Flush any remaining pending tool calls at the end
	flushToolCalls()

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

	n := 1
	if openaiSettings.N != nil {
		n = *openaiSettings.N
	}
	stream := chatSettings.Stream
	stop := chatSettings.Stop
	presencePenalty := 0.0
	if openaiSettings.PresencePenalty != nil {
		presencePenalty = *openaiSettings.PresencePenalty
	}
	frequencyPenalty := 0.0
	if openaiSettings.FrequencyPenalty != nil {
		frequencyPenalty = *openaiSettings.FrequencyPenalty
	}

	log.Debug().
		Str("model", engine).
		Int("max_tokens", maxTokens).
		Float64("temperature", temperature).
		Float64("top_p", topP).
		Int("n", n).
		Bool("stream", stream).
		Strs("stop", stop).
		Float64("presence_penalty", presencePenalty).
		Float64("frequency_penalty", frequencyPenalty).
		Msg("Making request to openai")

	var streamOptions *go_openai.StreamOptions
	if stream && !strings.Contains(engine, "mistral") {
		streamOptions = &go_openai.StreamOptions{IncludeUsage: true}
	}

	req := go_openai.ChatCompletionRequest{
		Model:            engine,
		Messages:         msgs_,
		MaxTokens:        maxTokens,
		Temperature:      float32(temperature),
		TopP:             float32(topP),
		N:                n,
		Stream:           stream,
		Stop:             stop,
		StreamOptions:    streamOptions,
		PresencePenalty:  float32(presencePenalty),
		FrequencyPenalty: float32(frequencyPenalty),
		// TODO(manuel, 2023-03-28) Properly load logit bias
		// See https://github.com/go-go-golems/geppetto/issues/48
		LogitBias: nil,
	}
	return &req, nil
}

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
    flushToolCalls := func() {
        if len(pendingToolCalls) == 0 {
            return
        }
        msgs_ = append(msgs_, go_openai.ChatCompletionMessage{
            Role:      string(conversation.RoleAssistant),
            ToolCalls: pendingToolCalls,
        })
        toolPhaseActive = true
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
    }

    if t != nil {
        for _, b := range t.Blocks {
            switch b.Kind {
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
                role := string(conversation.RoleAssistant)
                switch b.Kind {
                case turns.BlockKindUser:
                    role = string(conversation.RoleUser)
                case turns.BlockKindSystem:
                    role = string(conversation.RoleSystem)
                default:
                    role = string(conversation.RoleAssistant)
                }
                msg := go_openai.ChatCompletionMessage{Role: role, Content: text}
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
                result := ""
                if v, ok := b.Payload[turns.PayloadKeyResult]; ok {
                    switch tv := v.(type) {
                    case string:
                        result = tv
                    case []byte:
                        result = string(tv)
                    default:
                        if bb, err := json.Marshal(v); err == nil {
                            result = string(bb)
                        }
                    }
                }
                msgs_ = append(msgs_, go_openai.ChatCompletionMessage{
                    Role:       string(conversation.RoleTool),
                    Content:    result,
                    ToolCallID: toolID,
                })
                // Do not end tool phase yet; allow multiple tool_use in sequence
            case turns.BlockKindOther:
                // Ignore unknown blocks unless they carry text
                if v, ok := b.Payload[turns.PayloadKeyText]; ok {
                    text := ""
                    _ = assignString(&text, v)
                    if text != "" {
                        msg := go_openai.ChatCompletionMessage{Role: string(conversation.RoleAssistant), Content: text}
                        if len(pendingToolCalls) > 0 || toolPhaseActive {
                            delayedChats = append(delayedChats, msg)
                        } else {
                            msgs_ = append(msgs_, msg)
                        }
                    }
                }
            }
            // If we just finished a tool phase and encounter a non-tool_use and no pending calls, close phase
            if toolPhaseActive {
                // Lookahead is implicit: end phase when current block is not ToolUse and there are no pending calls
                if b.Kind != turns.BlockKindToolUse && len(pendingToolCalls) == 0 {
                    endToolPhase()
                }
            }
        }
    }

    // Flush any remaining tool calls
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

    log.Debug().
        Str("model", engine).
        Int("max_tokens", maxTokens).
        Float64("temperature", temperature).
        Float64("top_p", topP).
        Int("n", n).
        Bool("stream", stream).
        Strs("stop", stop).
        Float64("presence_penalty", presencePenalty).
        Float64("frequency_penalty", frequencyPenalty).
        Msg("Making request to openai from turn blocks")

    var streamOptions *go_openai.StreamOptions
    if stream && !strings.Contains(engine, "mistral") {
        streamOptions = &go_openai.StreamOptions{IncludeUsage: true}
    }

    req := go_openai.ChatCompletionRequest{
        Model:            engine,
        Messages:         msgs_,
        MaxTokens:        maxTokens,
        Temperature:      float32(temperature),
        TopP:             float32(topP),
        N:                n,
        Stream:           stream,
        Stop:             stop,
        StreamOptions:    streamOptions,
        PresencePenalty:  float32(presencePenalty),
        FrequencyPenalty: float32(frequencyPenalty),
        LogitBias:        nil,
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
func messageToOpenAIMessage(msg *conversation.Message) go_openai.ChatCompletionMessage {
	switch content := msg.Content.(type) {
	case *conversation.ChatMessageContent:
		res := go_openai.ChatCompletionMessage{
			Role:    string(content.Role),
			Content: content.Text,
		}

		if len(content.Images) > 0 {
			res = go_openai.ChatCompletionMessage{
				Role: string(content.Role),
				MultiContent: []go_openai.ChatMessagePart{
					{
						Type: go_openai.ChatMessagePartTypeText,
						Text: content.Text,
					},
				},
			}

			for _, img := range content.Images {
				imagePart := go_openai.ChatMessagePart{
					Type: go_openai.ChatMessagePartTypeImageURL,
					ImageURL: &go_openai.ChatMessageImageURL{
						URL:    img.ImageURL,
						Detail: go_openai.ImageURLDetail(img.Detail),
					},
				}
				if img.ImageURL == "" {
					// base64 encoded Content
					imagePart.ImageURL.URL =
						fmt.Sprintf(
							"data:%s;base64,%s",
							img.MediaType,
							base64.StdEncoding.EncodeToString(img.ImageContent),
						)
				}
				res.MultiContent = append(res.MultiContent, imagePart)
			}
		}

		// TODO(manuel, 2024-06-04) This should actually pass in a ToolUse content in the conversation
		// This is how claude expects it, and I also added a comment to ContentType's definition in message.go
		//
		// NOTE(manuel, 2024-06-04) It seems that these metadata keys are never set anywhere anyway
		metadata := msg.Metadata
		if metadata != nil {
			functionCall := metadata["function_call"]
			if functionCall_, ok := functionCall.(*go_openai.FunctionCall); ok {
				res.FunctionCall = functionCall_
			}

			toolCalls := metadata["tool_calls"]
			if toolCalls_, ok := toolCalls.([]go_openai.ToolCall); ok {
				res.ToolCalls = toolCalls_
			}

			toolCallID := metadata["tool_call_id"]
			if toolCallID_, ok := toolCallID.(string); ok {
				res.ToolCallID = toolCallID_
			}
		}
		return res

	case *conversation.ToolResultContent:
		res := go_openai.ChatCompletionMessage{
			Role:       string(conversation.RoleTool),
			Content:    content.Result,
			ToolCallID: content.ToolID,
		}

		return res

	case *conversation.ToolUseContent:
		// openai encodes tool use messages within the assistant completion message
		// TODO(manuel, 2024-06-25) This should be aggregated into a multi call chat message content, instead of being serialized individually
		res := go_openai.ChatCompletionMessage{
			Role: string(conversation.RoleAssistant),
			ToolCalls: []go_openai.ToolCall{
				{
					ID:   content.ToolID,
					Type: "function",
					Function: go_openai.FunctionCall{
						Name:      content.Name,
						Arguments: string(content.Input),
					},
				},
			},
		}

		return res
	}

	return go_openai.ChatCompletionMessage{}
}
