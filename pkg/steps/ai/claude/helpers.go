package claude

import (
	"encoding/base64"
	"encoding/json"

	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/pkg/errors"
)

// Removed obsolete messageToClaudeMessage (conversation-based)

// MakeMessageRequest builds a Claude MessageRequest from settings and a conversation
// Removed obsolete MakeMessageRequest (conversation-based)

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
	// Buffer messages that must come after a tool_use → tool_result pair
	delayedMsgs := []api.Message{}
	toolPhaseActive := false
	flushDelayed := func() {
		if len(delayedMsgs) > 0 {
			msgs = append(msgs, delayedMsgs...)
			delayedMsgs = nil
		}
	}
	systemPrompt := ""
	hasSystemPrompt := false
	if t != nil {
		for _, b := range t.Blocks {
			switch b.Kind {
			case turns.BlockKindSystem:
				text := ""
				if v, ok := b.Payload[turns.PayloadKeyText]; ok {
					if s, ok2 := v.(string); ok2 {
						text = s
					} else if bb, err := json.Marshal(v); err == nil {
						text = string(bb)
					}
				}
				if !hasSystemPrompt {
					systemPrompt = text
					hasSystemPrompt = true
				} else if text != "" {
					msg := api.Message{Role: RoleUser, Content: []api.Content{api.NewTextContent(text)}}
					if toolPhaseActive {
						delayedMsgs = append(delayedMsgs, msg)
					} else {
						msgs = append(msgs, msg)
					}
				}
			case turns.BlockKindUser:
				// If preserved Claude content is present, pass through directly
				if orig, ok := b.Metadata[turns.MetaKeyClaudeOriginalContent]; ok && orig != nil {
					if arr, ok2 := orig.([]api.Content); ok2 && len(arr) > 0 {
						msg := api.Message{Role: RoleUser, Content: arr}
						if toolPhaseActive {
							delayedMsgs = append(delayedMsgs, msg)
						} else {
							msgs = append(msgs, msg)
						}
						break
					}
				}
				text := ""
				if v, ok := b.Payload[turns.PayloadKeyText]; ok {
					if s, ok2 := v.(string); ok2 {
						text = s
					} else if bb, err := json.Marshal(v); err == nil {
						text = string(bb)
					}
				}
				parts := []api.Content{}
				if text != "" {
					parts = append(parts, api.NewTextContent(text))
				}
				// optional images from payload
				if imgs, ok := b.Payload[turns.PayloadKeyImages].([]map[string]any); ok && len(imgs) > 0 {
					for _, img := range imgs {
						mediaType, _ := img["media_type"].(string)
						if raw, ok := img["content"]; ok && raw != nil {
							var base64Content string
							switch rv := raw.(type) {
							case []byte:
								base64Content = base64.StdEncoding.EncodeToString(rv)
							case string:
								base64Content = rv
							}
							if base64Content != "" {
								parts = append(parts, api.NewImageContent(mediaType, base64Content))
							}
						}
					}
				}
				if len(parts) > 0 {
					msg := api.Message{Role: RoleUser, Content: parts}
					if toolPhaseActive {
						delayedMsgs = append(delayedMsgs, msg)
					} else {
						msgs = append(msgs, msg)
					}
				}
			case turns.BlockKindLLMText:
				// Allow preserved Claude content on assistant blocks too
				if orig, ok := b.Metadata[turns.MetaKeyClaudeOriginalContent]; ok && orig != nil {
					if arr, ok2 := orig.([]api.Content); ok2 && len(arr) > 0 {
						msg := api.Message{Role: RoleAssistant, Content: arr}
						if toolPhaseActive {
							delayedMsgs = append(delayedMsgs, msg)
						} else {
							msgs = append(msgs, msg)
						}
						break
					}
				}
				text := ""
				if v, ok := b.Payload[turns.PayloadKeyText]; ok {
					if s, ok2 := v.(string); ok2 {
						text = s
					} else if bb, err := json.Marshal(v); err == nil {
						text = string(bb)
					}
				}
				if text != "" {
					msg := api.Message{Role: RoleAssistant, Content: []api.Content{api.NewTextContent(text)}}
					if toolPhaseActive {
						delayedMsgs = append(delayedMsgs, msg)
					} else {
						msgs = append(msgs, msg)
					}
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
				msgs = append(msgs, api.Message{Role: RoleAssistant, Content: []api.Content{api.NewToolUseContent(toolID, name, argsStr)}})
				toolPhaseActive = true
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
				msgs = append(msgs, api.Message{Role: RoleUser, Content: []api.Content{api.NewToolResultContent(toolID, result)}})
				// After emitting tool_result, flush any delayed messages and end phase
				flushDelayed()
				toolPhaseActive = false
			case turns.BlockKindOther:
				if v, ok := b.Payload[turns.PayloadKeyText]; ok {
					if s, ok2 := v.(string); ok2 && s != "" {
						msg := api.Message{Role: RoleAssistant, Content: []api.Content{api.NewTextContent(s)}}
						if toolPhaseActive {
							delayedMsgs = append(delayedMsgs, msg)
						} else {
							msgs = append(msgs, msg)
						}
					}
				}
			}
		}
	}

	// If we ended without a tool_result, append any delayed messages to avoid dropping content
	flushDelayed()

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
