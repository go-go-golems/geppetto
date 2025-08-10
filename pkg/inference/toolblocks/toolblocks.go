package toolblocks

import (
	"encoding/json"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// ToolCall represents a pending tool invocation described by a Turn block.
type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

// ToolResult represents the outcome of executing a tool call.
type ToolResult struct {
	ID      string
	Content string
	Error   string
}

// ExtractPendingToolCalls finds tool_call blocks that don't yet have a matching tool_use block.
func ExtractPendingToolCalls(t *turns.Turn) []ToolCall {
	if t == nil {
		return nil
	}
	used := make(map[string]bool)
	for _, b := range t.Blocks {
		if b.Kind == turns.BlockKindToolUse {
			if id, ok := b.Payload["id"].(string); ok && id != "" {
				used[id] = true
			}
		}
	}
	var calls []ToolCall
	for _, b := range t.Blocks {
		if b.Kind != turns.BlockKindToolCall {
			continue
		}
		id, _ := b.Payload["id"].(string)
		if id == "" || used[id] {
			continue
		}
		name, _ := b.Payload["name"].(string)
		var args map[string]any
		if raw := b.Payload["args"]; raw != nil {
			switch v := raw.(type) {
			case map[string]any:
				args = v
			case string:
				_ = json.Unmarshal([]byte(v), &args)
			case json.RawMessage:
				_ = json.Unmarshal(v, &args)
			default:
				if bts, err := json.Marshal(v); err == nil {
					_ = json.Unmarshal(bts, &args)
				}
			}
		}
		if args == nil {
			args = map[string]any{}
		}
		calls = append(calls, ToolCall{ID: id, Name: name, Arguments: args})
	}
	return calls
}

// AppendToolResultsBlocks appends tool_use blocks to the Turn from provided results.
func AppendToolResultsBlocks(t *turns.Turn, results []ToolResult) {
    for _, r := range results {
        result := any(r.Content)
        if r.Error != "" {
            result = "Error: " + r.Error
        }
        turns.AppendBlock(t, turns.NewToolUseBlock(r.ID, result))
    }
}
