package turns

import "github.com/google/uuid"

// Convenience constructors for commonly used Block shapes.

// Role string constants used for human roles in blocks.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// NewUserTextBlock returns a Block representing a user text message.
func NewUserTextBlock(text string) Block {
	return Block{
		ID:      uuid.NewString(),
		Kind:    BlockKindUser,
		Role:    RoleUser,
		Payload: map[string]any{PayloadKeyText: text},
	}
}

// NewUserMultimodalBlock creates a user block with text and optional images.
// images is a slice of maps with keys: "media_type" (string), and either "url" (string) or "content" ([]byte/base64).
func NewUserMultimodalBlock(text string, images []map[string]any) Block {
	payload := map[string]any{PayloadKeyText: text}
	if len(images) > 0 {
		payload[PayloadKeyImages] = images
	}
	return Block{
		ID:      uuid.NewString(),
		Kind:    BlockKindUser,
		Role:    RoleUser,
		Payload: payload,
	}
}

// NewAssistantTextBlock returns a Block representing assistant LLM text output.
func NewAssistantTextBlock(text string) Block {
	return Block{
		ID:      uuid.NewString(),
		Kind:    BlockKindLLMText,
		Role:    RoleAssistant,
		Payload: map[string]any{PayloadKeyText: text},
	}
}

// NewSystemTextBlock returns a Block representing a system directive.
func NewSystemTextBlock(text string) Block {
	return Block{
		ID:      uuid.NewString(),
		Kind:    BlockKindSystem,
		Role:    RoleSystem,
		Payload: map[string]any{PayloadKeyText: text},
	}
}

// NewToolCallBlock returns a Block requesting invocation of a tool.
// id is a provider- or runtime-assigned identifier used to correlate tool_use results.
// name is the tool/function name. args contains the structured input (any JSON-serializable value).
func NewToolCallBlock(id string, name string, args any) Block {
	return Block{
		ID:   id,
		Kind: BlockKindToolCall,
		Payload: map[string]any{
			PayloadKeyID:   id,
			PayloadKeyName: name,
			PayloadKeyArgs: args,
		},
	}
}

// NewToolUseBlock returns a Block capturing the result of a tool execution.
// id must match the corresponding tool_call id.
// result holds the execution output (any JSON-serializable value or string).
func NewToolUseBlock(id string, result any) Block {
	return Block{
		ID:   uuid.NewString(),
		Kind: BlockKindToolUse,
		Payload: map[string]any{
			PayloadKeyID:     id,
			PayloadKeyResult: result,
		},
	}
}

// InsertBlockBeforeLast inserts the given block as the second-to-last entry in the turn.
// If the turn has fewer than 1 blocks, it appends the block normally.
func InsertBlockBeforeLast(t *Turn, b Block) {
	if t == nil {
		return
	}
	if len(t.Blocks) >= 1 {
		last := t.Blocks[len(t.Blocks)-1]
		t.Blocks = t.Blocks[:len(t.Blocks)-1]
		AppendBlock(t, b)
		AppendBlock(t, last)
		return
	}
	AppendBlock(t, b)
}
