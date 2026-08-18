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
// images is a slice of maps with keys:
//   - "media_type" (string) for inline content
//   - either "url" (string), "content" ([]byte/base64), or provider-specific "file_id" (string)
//   - optional "detail" for providers that support image detail selection
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
	return NewToolUseBlockWithError(id, result, "")
}

// NewToolUseBlockWithError returns a Block capturing the result of a tool execution, including
// an optional error string.
// id must match the corresponding tool_call id.
// result holds the execution output (any JSON-serializable value or string). If the tool failed,
// result may be nil and error should be set.
func NewToolUseBlockWithError(id string, result any, err string) Block {
	return Block{
		ID:   uuid.NewString(),
		Kind: BlockKindToolUse,
		Payload: map[string]any{
			PayloadKeyID:     id,
			PayloadKeyResult: result,
			PayloadKeyError:  err,
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

// ImagesFromPayload returns the images stored under PayloadKeyImages in a
// block payload as []map[string]any.
//
// It tolerates the shapes that occur in practice:
//   - []map[string]any, as produced by NewUserMultimodalBlock
//   - []any whose entries are map[string]any, as produced by YAML/JSON decoding
//     (for example serde.FromYAML or a JSON turn store)
//   - []any whose entries are map[any]any, as produced by yaml.v2-style decoders
//
// Entries that are not maps are skipped. A missing or nil value returns nil.
// Provider adapters must use this (or BlockImages) instead of asserting the
// concrete slice type, otherwise images silently disappear after a turn has
// been persisted and reloaded.
func ImagesFromPayload(payload map[string]any) []map[string]any {
	if payload == nil {
		return nil
	}
	raw, ok := payload[PayloadKeyImages]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []map[string]any:
		return v
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			switch m := item.(type) {
			case map[string]any:
				out = append(out, m)
			case map[any]any:
				conv := make(map[string]any, len(m))
				for k, val := range m {
					ks, ok := k.(string)
					if !ok {
						continue
					}
					conv[ks] = val
				}
				out = append(out, conv)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return nil
	}
}

// BlockImages returns the images attached to a block. See ImagesFromPayload
// for the accepted shapes.
func BlockImages(b Block) []map[string]any {
	return ImagesFromPayload(b.Payload)
}

// HasImages reports whether the block carries at least one image entry.
func HasImages(b Block) bool {
	return len(BlockImages(b)) > 0
}
