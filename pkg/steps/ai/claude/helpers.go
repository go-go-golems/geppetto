package claude

import (
    "encoding/base64"
    "encoding/json"

    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/steps"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/go-go-golems/glazed/pkg/helpers/cast"
    "github.com/pkg/errors"
)

// messageToClaudeMessage converts a conversation message to a Claude API message
func messageToClaudeMessage(msg *conversation.Message) api.Message {
	switch content := msg.Content.(type) {
	case *conversation.ChatMessageContent:
		// If original Claude content was preserved in metadata, use it
		if claudeContent, exists := msg.Metadata["claude_original_content"]; exists {
			if originalContent, ok := claudeContent.([]api.Content); ok {
				return api.Message{
					Role:    string(content.Role),
					Content: originalContent,
				}
			}
		}

		res := api.Message{
			Role: string(content.Role),
			Content: []api.Content{
				api.NewTextContent(content.Text),
			},
		}
		for _, img := range content.Images {
			res.Content = append(res.Content, api.NewImageContent(img.MediaType, base64.StdEncoding.EncodeToString(img.ImageContent)))
		}
		return res

	case *conversation.ToolUseContent:
		// Claude expects tool_use in assistant role
		return api.Message{
			Role: string(conversation.RoleAssistant),
			Content: []api.Content{
				api.NewToolUseContent(content.ToolID, content.Name, string(content.Input)),
			},
		}

	case *conversation.ToolResultContent:
		// Claude expects tool results in user role
		return api.Message{
			Role: string(conversation.RoleUser),
			Content: []api.Content{
				api.NewToolResultContent(content.ToolID, content.Result),
			},
		}
	}

	return api.Message{}
}

// MakeMessageRequest builds a Claude MessageRequest from settings and a conversation
func MakeMessageRequest(
	s *settings.StepSettings,
	messages conversation.Conversation,
) (*api.MessageRequest, error) {
	if s.Client == nil {
		return nil, steps.ErrMissingClientSettings
	}
	if s.Claude == nil {
		return nil, errors.New("no claude settings")
	}

	chatSettings := s.Chat
	engine := ""
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	msgs := []api.Message{}
	systemPrompt := ""
	for _, m := range messages {
		if chatMsg, ok := m.Content.(*conversation.ChatMessageContent); ok && chatMsg.Role == conversation.RoleSystem {
			systemPrompt = chatMsg.Text
			continue
		}
		msgs = append(msgs, messageToClaudeMessage(m))
	}

	temperature := 0.0
	if chatSettings.Temperature != nil {
		temperature = *chatSettings.Temperature
	}
	topP := 0.0
	if chatSettings.TopP != nil {
		topP = *chatSettings.TopP
	}
	maxTokens := 1024
	if chatSettings.MaxResponseTokens != nil && *chatSettings.MaxResponseTokens > 0 {
		maxTokens = *chatSettings.MaxResponseTokens
	}

	req := &api.MessageRequest{
		Model:         engine,
		Messages:      msgs,
		MaxTokens:     maxTokens,
		Metadata:      nil,
		StopSequences: chatSettings.Stop,
		Stream:        chatSettings.Stream,
		System:        systemPrompt,
		Temperature:   cast.WrapAddr[float64](temperature),
		Tools:         nil,
		TopK:          nil,
		TopP:          cast.WrapAddr[float64](topP),
	}

	return req, nil
}

// MakeMessageRequestFromTurn builds a Claude MessageRequest directly from a Turn's blocks,
// avoiding any dependency on conversation.Conversation.
func MakeMessageRequestFromTurn(
    s *settings.StepSettings,
    t *turns.Turn,
) (*api.MessageRequest, error) {
    if s.Client == nil {
        return nil, steps.ErrMissingClientSettings
    }
    if s.Claude == nil {
        return nil, errors.New("no claude settings")
    }

    chatSettings := s.Chat
    engine := ""
    if chatSettings.Engine != nil {
        engine = *chatSettings.Engine
    } else {
        return nil, errors.New("no engine specified")
    }

    msgs := []api.Message{}
    systemPrompt := ""
    if t != nil {
        for _, b := range t.Blocks {
            switch b.Kind {
            case turns.BlockKindSystem:
                if v, ok := b.Payload[turns.PayloadKeyText]; ok {
                    if s, ok2 := v.(string); ok2 {
                        systemPrompt = s
                    } else if bb, err := json.Marshal(v); err == nil {
                        systemPrompt = string(bb)
                    }
                }
            case turns.BlockKindUser:
                text := ""
                if v, ok := b.Payload[turns.PayloadKeyText]; ok {
                    if s, ok2 := v.(string); ok2 {
                        text = s
                    } else if bb, err := json.Marshal(v); err == nil {
                        text = string(bb)
                    }
                }
                if text != "" {
                    msgs = append(msgs, api.Message{Role: string(conversation.RoleUser), Content: []api.Content{api.NewTextContent(text)}})
                }
            case turns.BlockKindLLMText:
                text := ""
                if v, ok := b.Payload[turns.PayloadKeyText]; ok {
                    if s, ok2 := v.(string); ok2 {
                        text = s
                    } else if bb, err := json.Marshal(v); err == nil {
                        text = string(bb)
                    }
                }
                if text != "" {
                    msgs = append(msgs, api.Message{Role: string(conversation.RoleAssistant), Content: []api.Content{api.NewTextContent(text)}})
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
                        argsStr = tv
                    case json.RawMessage:
                        argsStr = string(tv)
                    default:
                        if bb, err := json.Marshal(v); err == nil {
                            argsStr = string(bb)
                        }
                    }
                }
                msgs = append(msgs, api.Message{Role: string(conversation.RoleAssistant), Content: []api.Content{api.NewToolUseContent(toolID, name, argsStr)}})
            case turns.BlockKindToolUse:
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
                msgs = append(msgs, api.Message{Role: string(conversation.RoleUser), Content: []api.Content{api.NewToolResultContent(toolID, result)}})
            case turns.BlockKindOther:
                if v, ok := b.Payload[turns.PayloadKeyText]; ok {
                    if s, ok2 := v.(string); ok2 && s != "" {
                        msgs = append(msgs, api.Message{Role: string(conversation.RoleAssistant), Content: []api.Content{api.NewTextContent(s)}})
                    }
                }
            }
        }
    }

    temperature := 0.0
    if chatSettings.Temperature != nil {
        temperature = *chatSettings.Temperature
    }
    topP := 0.0
    if chatSettings.TopP != nil {
        topP = *chatSettings.TopP
    }
    maxTokens := 1024
    if chatSettings.MaxResponseTokens != nil && *chatSettings.MaxResponseTokens > 0 {
        maxTokens = *chatSettings.MaxResponseTokens
    }

    req := &api.MessageRequest{
        Model:         engine,
        Messages:      msgs,
        MaxTokens:     maxTokens,
        Metadata:      nil,
        StopSequences: chatSettings.Stop,
        Stream:        chatSettings.Stream,
        System:        systemPrompt,
        Temperature:   cast.WrapAddr[float64](temperature),
        Tools:         nil,
        TopK:          nil,
        TopP:          cast.WrapAddr[float64](topP),
    }
    return req, nil
}

// assignString writes a string representation of v into out when possible.
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


// end helpers

// Just like in the openai package, we merge the received tool calls and messages from streaming
