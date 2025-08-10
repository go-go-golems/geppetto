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
            text, _ := getString(b.Payload, "text")
            msgs = append(msgs, conversation.NewChatMessage(conversation.RoleUser, text))
        case BlockKindLLMText:
            text, _ := getString(b.Payload, "text")
            msgs = append(msgs, conversation.NewChatMessage(conversation.RoleAssistant, text))
        case BlockKindSystem:
            text, _ := getString(b.Payload, "text")
            msgs = append(msgs, conversation.NewChatMessage(conversation.RoleSystem, text))
        case BlockKindToolCall:
            // ToolUseContent represents a tool call request from the assistant
            name, _ := getString(b.Payload, "name")
            toolID, _ := getString(b.Payload, "id")
            args := toJSONRawMessage(b.Payload["args"])
            tuc := &conversation.ToolUseContent{
                ToolID: toolID,
                Name:   name,
                Input:  args,
                Type:   "function",
            }
            msgs = append(msgs, conversation.NewMessage(tuc))
        case BlockKindToolUse:
            // ToolResultContent captures tool execution results
            toolID, _ := getString(b.Payload, "id")
            resultStr := toJSONString(b.Payload["result"])
            trc := &conversation.ToolResultContent{
                ToolID: toolID,
                Result: resultStr,
            }
            msgs = append(msgs, conversation.NewMessage(trc))
        case BlockKindOther:
            // ignore or map to assistant text if payload.text exists
            if text, ok := getString(b.Payload, "text"); ok {
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
    order := 0
    for i := startIdx; i < len(updated); i++ {
        msg := updated[i]
        switch c := msg.Content.(type) {
        case *conversation.ChatMessageContent:
            payload := map[string]any{"text": c.Text}
            kind := BlockKindLLMText
            switch c.Role {
            case conversation.RoleUser:
                kind = BlockKindUser
            case conversation.RoleAssistant:
                kind = BlockKindLLMText
            case conversation.RoleSystem:
                kind = BlockKindSystem
            }
            blocks = append(blocks, Block{Order: order, Kind: kind, Role: string(c.Role), Payload: payload})
            order++
        case *conversation.ToolUseContent:
            // tool_call
            var args any
            if len(c.Input) > 0 {
                _ = json.Unmarshal(c.Input, &args)
            }
            payload := map[string]any{"id": c.ToolID, "name": c.Name, "args": args}
            blocks = append(blocks, Block{Order: order, Kind: BlockKindToolCall, Payload: payload})
            order++
        case *conversation.ToolResultContent:
            payload := map[string]any{"id": c.ToolID, "result": c.Result}
            blocks = append(blocks, Block{Order: order, Kind: BlockKindToolUse, Payload: payload})
            order++
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


