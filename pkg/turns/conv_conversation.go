package turns

import (
	"encoding/json"

	"github.com/go-go-golems/geppetto/pkg/conversation"
)

// BuildConversationFromTurn converts a Turn's blocks into a conversation.Conversation
// according to the mapping rules in the design document.
func BuildConversationFromTurn(t *Turn) conversation.Conversation {
	if t == nil {
		return nil
	}
	msgs := make(conversation.Conversation, 0, len(t.Blocks))
	for _, b := range t.Blocks {
		switch b.Kind {
		case BlockKindUser:
			text, _ := getString(b.Payload, PayloadKeyText)
			msgs = append(msgs, conversation.NewChatMessage(conversation.RoleUser, text))
		case BlockKindLLMText:
			text, _ := getString(b.Payload, PayloadKeyText)
			msgs = append(msgs, conversation.NewChatMessage(conversation.RoleAssistant, text))
		case BlockKindSystem:
			text, _ := getString(b.Payload, PayloadKeyText)
			if text != "" {
				msgs = append(msgs, conversation.NewChatMessage(conversation.RoleSystem, text))
			}
		case BlockKindToolCall:
			// ToolUseContent represents a tool call request from the assistant
			name, _ := getString(b.Payload, PayloadKeyName)
			toolID, _ := getString(b.Payload, PayloadKeyID)
			args := toJSONRawMessage(b.Payload[PayloadKeyArgs])
			tuc := &conversation.ToolUseContent{
				ToolID: toolID,
				Name:   name,
				Input:  args,
				Type:   "function",
			}
			msgs = append(msgs, conversation.NewMessage(tuc))
		case BlockKindToolUse:
			// ToolResultContent captures tool execution results
			toolID, _ := getString(b.Payload, PayloadKeyID)
			resultStr := toJSONString(b.Payload[PayloadKeyResult])
			trc := &conversation.ToolResultContent{
				ToolID: toolID,
				Result: resultStr,
			}
			msgs = append(msgs, conversation.NewMessage(trc))
		case BlockKindOther:
			// ignore or map to assistant text if payload.text exists
			if text, ok := getString(b.Payload, PayloadKeyText); ok {
				msgs = append(msgs, conversation.NewChatMessage(conversation.RoleAssistant, text))
			}
		}
	}
	return msgs
}

// BlocksFromConversationDelta converts the newly appended messages in updated
// (starting at startIdx) into Blocks to be appended to the Turn.
func BlocksFromConversationDelta(updated conversation.Conversation, startIdx int) []Block {
	if updated == nil || startIdx >= len(updated) {
		return nil
	}
	blocks := make([]Block, 0, len(updated)-startIdx)
	for i := startIdx; i < len(updated); i++ {
		msg := updated[i]
		switch c := msg.Content.(type) {
		case *conversation.ChatMessageContent:
			payload := map[string]any{PayloadKeyText: c.Text}
			kind := BlockKindLLMText
			switch c.Role {
			case conversation.RoleUser:
				kind = BlockKindUser
			case conversation.RoleAssistant:
				kind = BlockKindLLMText
			case conversation.RoleSystem:
				kind = BlockKindSystem
			case conversation.RoleTool:
				kind = BlockKindOther
			}
			blocks = append(blocks, Block{ID: conversation.NewNodeID().String(), Kind: kind, Role: string(c.Role), Payload: payload})
		case *conversation.ToolUseContent:
			// tool_call
			var args any
			if len(c.Input) > 0 {
				_ = json.Unmarshal(c.Input, &args)
			}
			payload := map[string]any{PayloadKeyID: c.ToolID, PayloadKeyName: c.Name, PayloadKeyArgs: args}
			blocks = append(blocks, Block{ID: c.ToolID, Kind: BlockKindToolCall, Payload: payload})
		case *conversation.ToolResultContent:
			payload := map[string]any{PayloadKeyID: c.ToolID, PayloadKeyResult: c.Result}
			blocks = append(blocks, Block{ID: conversation.NewNodeID().String(), Kind: BlockKindToolUse, Payload: payload})
		}
	}
	return blocks
}

func getString(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	if !ok {
		return "", false
	}
	if s, ok := v.(string); ok {
		return s, true
	}
	return "", false
}

func toJSONRawMessage(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case json.RawMessage:
		return t
	case string:
		return json.RawMessage([]byte(t))
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		return b
	}
}

func toJSONString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
