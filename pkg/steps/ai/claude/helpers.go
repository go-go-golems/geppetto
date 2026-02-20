package claude

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"

	infengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
				if orig, ok, err := turns.KeyBlockMetaClaudeOriginalContent.Get(b.Metadata); err != nil {
					return nil, errors.Wrap(err, "get claude original content (user block)")
				} else if ok && orig != nil {
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
				if orig, ok, err := turns.KeyBlockMetaClaudeOriginalContent.Get(b.Metadata); err != nil {
					return nil, errors.Wrap(err, "get claude original content (assistant block)")
				} else if ok && orig != nil {
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
			case turns.BlockKindReasoning:
				// Reasoning blocks are not part of Claude's message protocol yet; skip.
				continue
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
				result := toolUsePayloadToJSONString(b.Payload)
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

	// Determine effective sampling settings while respecting Claude constraint:
	// temperature and top_p cannot both be specified for some models.
	// Default for both is 1.0 — when set to default, we omit the fields.
	const defaultSampling = 1.0
	const eps = 1e-9

	var temperaturePtr *float64
	if chatSettings.Temperature != nil {
		if math.Abs(*chatSettings.Temperature-defaultSampling) > eps {
			v := *chatSettings.Temperature
			temperaturePtr = &v
		}
	}

	var topPPtr *float64
	if chatSettings.TopP != nil {
		if math.Abs(*chatSettings.TopP-defaultSampling) > eps {
			v := *chatSettings.TopP
			topPPtr = &v
		}
	}

	if temperaturePtr != nil && topPPtr != nil {
		return nil, errors.New("both temperature and top_p are set to non-default values; Claude models require only one to be specified")
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
		Temperature:   temperaturePtr,
		Tools:         nil,
		TopK:          nil,
		TopP:          topPPtr,
	}

	// Apply provider-native structured output schema when configured.
	if chatSettings.IsStructuredOutputEnabled() {
		cfg, err := chatSettings.StructuredOutputConfig()
		if err != nil {
			if chatSettings.StructuredOutputRequireValid {
				return nil, err
			}
			log.Warn().Err(err).Msg("Claude request: ignoring invalid structured output configuration")
		} else if cfg != nil {
			req.OutputFormat = &api.OutputFormat{
				Type:   "json_schema",
				Name:   cfg.Name,
				Schema: cfg.Schema,
			}
		}
	}

	// Apply per-turn InferenceConfig overrides (Turn.Data > StepSettings.Inference).
	infCfg := infengine.ResolveInferenceConfig(t, s.Inference)
	if infCfg != nil {
		if infCfg.ThinkingBudget != nil && *infCfg.ThinkingBudget > 0 {
			req.Thinking = &api.ThinkingParam{
				Type:         "enabled",
				BudgetTokens: *infCfg.ThinkingBudget,
			}
		}
		if infCfg.Temperature != nil {
			v := *infCfg.Temperature
			req.Temperature = &v
		}
		if infCfg.TopP != nil {
			v := *infCfg.TopP
			req.TopP = &v
		}
		if infCfg.MaxResponseTokens != nil && *infCfg.MaxResponseTokens > 0 {
			req.MaxTokens = *infCfg.MaxResponseTokens
		}
		if len(infCfg.Stop) > 0 {
			req.StopSequences = infCfg.Stop
		}
	}

	// Apply Claude-specific per-turn overrides from Turn.Data.
	if claudeCfg := infengine.ResolveClaudeInferenceConfig(t); claudeCfg != nil {
		if claudeCfg.UserID != nil {
			req.Metadata = &api.Metadata{UserID: *claudeCfg.UserID}
		}
		if claudeCfg.TopK != nil {
			req.TopK = claudeCfg.TopK
		}
	}

	// Apply StructuredOutputConfig from Turn.Data (per-turn override).
	if t != nil {
		if soCfg, ok, err := infengine.KeyStructuredOutputConfig.Get(t.Data); err == nil && ok && soCfg.IsEnabled() {
			if err := soCfg.Validate(); err == nil {
				req.OutputFormat = &api.OutputFormat{
					Type:   "json_schema",
					Name:   soCfg.Name,
					Schema: soCfg.Schema,
				}
			}
		}
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

// Just like in the openai package, we merge the received tool calls and messages from streaming
