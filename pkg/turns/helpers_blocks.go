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

// WithClaudeOriginalContent attaches Claude-native content blocks to a block's metadata for lossless roundtrips.
func WithClaudeOriginalContent(b Block, original any) Block {
	return WithBlockMetadata(b, map[string]any{MetaKeyClaudeOriginalContent: original})
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

// WithBlockMetadata sets key/value pairs on a copy of the block's Metadata and returns it.
func WithBlockMetadata(b Block, kvs map[string]interface{}) Block {
	if len(kvs) == 0 {
		return b
	}
	// Clone existing metadata map to avoid aliasing
	cloned := make(map[string]interface{}, len(b.Metadata)+len(kvs))
	for k, v := range b.Metadata {
		cloned[k] = v
	}
	for k, v := range kvs {
		cloned[k] = v
	}
	b.Metadata = cloned
	return b
}

// HasBlockMetadata returns true if the block's Metadata contains key==value.
func HasBlockMetadata(b Block, key string, value string) bool {
	if b.Metadata == nil {
		return false
	}
	v, ok := b.Metadata[key]
	if !ok {
		return false
	}
	if sv, ok := v.(string); ok {
		return sv == value
	}
	return false
}

// RemoveBlocksByMetadata removes all blocks from the Turn where Metadata[key] equals any of the provided values.
// It returns the number of removed blocks.
func RemoveBlocksByMetadata(t *Turn, key string, values ...string) int {
	if t == nil || len(t.Blocks) == 0 {
		return 0
	}
	// Build quick lookup set
	valSet := map[string]struct{}{}
	for _, v := range values {
		valSet[v] = struct{}{}
	}

	kept := make([]Block, 0, len(t.Blocks))
	removed := 0
	for _, b := range t.Blocks {
		if b.Metadata != nil {
			if v, ok := b.Metadata[key]; ok {
				if sv, ok2 := v.(string); ok2 {
					if _, match := valSet[sv]; match {
						removed++
						continue
					}
				}
			}
		}
		kept = append(kept, b)
	}
	t.Blocks = kept
	return removed
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
