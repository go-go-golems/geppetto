package conversation

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/google/uuid"
)

// Mutation represents a deterministic change to the conversation.
type Mutation interface {
	Apply(cs *ConversationState) error
	Name() string
}

type appendBlockMutation struct {
	block turns.Block
}

func (m appendBlockMutation) Apply(cs *ConversationState) error {
	if cs == nil {
		return fmt.Errorf("conversation state is nil")
	}
	b := m.block
	ensureBlockID(&b)
	cs.Blocks = append(cs.Blocks, b)
	return nil
}

func (m appendBlockMutation) Name() string { return "append_block" }

// MutateAppendBlock appends a single block to the conversation.
func MutateAppendBlock(block turns.Block) Mutation {
	return appendBlockMutation{block: block}
}

// MutateAppendBlocks appends multiple blocks to the conversation.
func MutateAppendBlocks(blocks ...turns.Block) Mutation {
	return appendBlocksMutation{blocks: blocks}
}

type appendBlocksMutation struct {
	blocks []turns.Block
}

func (m appendBlocksMutation) Apply(cs *ConversationState) error {
	if cs == nil {
		return fmt.Errorf("conversation state is nil")
	}
	for _, b := range m.blocks {
		block := b
		ensureBlockID(&block)
		cs.Blocks = append(cs.Blocks, block)
	}
	return nil
}

func (m appendBlocksMutation) Name() string { return "append_blocks" }

// MutateAppendUserText appends a user text block.
func MutateAppendUserText(text string) Mutation {
	return appendTextMutation{role: turns.RoleUser, text: text}
}

// MutateAppendAssistantText appends an assistant text block.
func MutateAppendAssistantText(text string) Mutation {
	return appendTextMutation{role: turns.RoleAssistant, text: text}
}

// MutateAppendToolCall appends a tool_call block.
func MutateAppendToolCall(id, name string, args any) Mutation {
	return appendToolCallMutation{id: id, name: name, args: args}
}

// MutateAppendToolResult appends a tool_use block.
func MutateAppendToolResult(id string, result any) Mutation {
	return appendToolResultMutation{id: id, result: result}
}

// SystemPromptOptions defines how to insert a system prompt block.
type SystemPromptOptions struct {
	Text          string
	MetadataKey   turns.BlockMetadataKey
	MetadataValue string
}

// MutateEnsureSystemPrompt ensures a single system prompt block is present.
func MutateEnsureSystemPrompt(opts SystemPromptOptions) Mutation {
	return ensureSystemPromptMutation{opts: opts}
}

type ensureSystemPromptMutation struct {
	opts SystemPromptOptions
}

func (m ensureSystemPromptMutation) Apply(cs *ConversationState) error {
	if cs == nil {
		return fmt.Errorf("conversation state is nil")
	}
	text := strings.TrimSpace(m.opts.Text)
	if text == "" {
		return fmt.Errorf("system prompt text is empty")
	}
	if m.opts.MetadataKey == "" || m.opts.MetadataValue == "" {
		return fmt.Errorf("system prompt metadata key/value required for idempotency")
	}
	filtered := make([]turns.Block, 0, len(cs.Blocks))
	for _, b := range cs.Blocks {
		if blockHasMetadataValue(b, m.opts.MetadataKey, m.opts.MetadataValue) {
			continue
		}
		filtered = append(filtered, b)
	}
	newBlock := turns.NewSystemTextBlock(text)
	if err := setBlockMetadata(&newBlock, m.opts.MetadataKey, m.opts.MetadataValue); err != nil {
		return err
	}
	cs.Blocks = append([]turns.Block{newBlock}, filtered...)
	return nil
}

func (m ensureSystemPromptMutation) Name() string { return "ensure_system_prompt" }

type appendTextMutation struct {
	role string
	text string
}

func (m appendTextMutation) Apply(cs *ConversationState) error {
	if cs == nil {
		return fmt.Errorf("conversation state is nil")
	}
	text := strings.TrimSpace(m.text)
	if text == "" {
		return fmt.Errorf("text is empty")
	}
	var block turns.Block
	switch m.role {
	case turns.RoleUser:
		block = turns.NewUserTextBlock(text)
	case turns.RoleAssistant:
		block = turns.NewAssistantTextBlock(text)
	default:
		return fmt.Errorf("unsupported role %q", m.role)
	}
	cs.Blocks = append(cs.Blocks, block)
	return nil
}

func (m appendTextMutation) Name() string { return "append_text" }

type appendToolCallMutation struct {
	id   string
	name string
	args any
}

func (m appendToolCallMutation) Apply(cs *ConversationState) error {
	if cs == nil {
		return fmt.Errorf("conversation state is nil")
	}
	if strings.TrimSpace(m.id) == "" {
		return fmt.Errorf("tool_call id is empty")
	}
	if strings.TrimSpace(m.name) == "" {
		return fmt.Errorf("tool_call name is empty")
	}
	cs.Blocks = append(cs.Blocks, turns.NewToolCallBlock(m.id, m.name, m.args))
	return nil
}

func (m appendToolCallMutation) Name() string { return "append_tool_call" }

type appendToolResultMutation struct {
	id     string
	result any
}

func (m appendToolResultMutation) Apply(cs *ConversationState) error {
	if cs == nil {
		return fmt.Errorf("conversation state is nil")
	}
	if strings.TrimSpace(m.id) == "" {
		return fmt.Errorf("tool_result id is empty")
	}
	cs.Blocks = append(cs.Blocks, turns.NewToolUseBlock(m.id, m.result))
	return nil
}

func (m appendToolResultMutation) Name() string { return "append_tool_result" }

func ensureBlockID(b *turns.Block) {
	if b == nil {
		return
	}
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
}

func blockHasMetadataValue(b turns.Block, key turns.BlockMetadataKey, value string) bool {
	found := false
	b.Metadata.Range(func(k turns.BlockMetadataKey, v any) bool {
		if k == key {
			if s, ok := v.(string); ok && s == value {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func setBlockMetadata(b *turns.Block, key turns.BlockMetadataKey, value string) error {
	if b == nil {
		return fmt.Errorf("block is nil")
	}
	metaKey := turns.BlockMetaKeyFromID[string](key)
	return metaKey.Set(&b.Metadata, value)
}
