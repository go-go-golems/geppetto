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
    flushToolCalls := func() {
        if len(pendingToolCalls) == 0 {
            return
        }
        msgs_ = append(msgs_, go_openai.ChatCompletionMessage{
            Role:      "assistant",
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
                role := "assistant"
                switch b.Kind {
                case turns.BlockKindUser:
                    role = "user"
                case turns.BlockKindSystem:
                    role = "system"
                default:
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
                    Role:       "tool",
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
                        msg := go_openai.ChatCompletionMessage{Role: "assistant", Content: text}
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
// Removed obsolete messageToOpenAIMessage (conversation-based)
