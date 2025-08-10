package turns

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
        Kind:    BlockKindUser,
        Role:    RoleUser,
        Payload: map[string]any{PayloadKeyText: text},
    }
}

// NewAssistantTextBlock returns a Block representing assistant LLM text output.
func NewAssistantTextBlock(text string) Block {
    return Block{
        Kind:    BlockKindLLMText,
        Role:    RoleAssistant,
        Payload: map[string]any{PayloadKeyText: text},
    }
}

// NewSystemTextBlock returns a Block representing a system directive.
func NewSystemTextBlock(text string) Block {
    return Block{
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
        Kind:   BlockKindToolCall,
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
        Kind:   BlockKindToolUse,
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



