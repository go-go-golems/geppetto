package turns

import (
	"gopkg.in/yaml.v3"
	"strings"
)

// BlockKind represents the kind of a block within a Turn.
type BlockKind int

const (
	BlockKindUser BlockKind = iota
	BlockKindLLMText
	BlockKindToolCall
	BlockKindToolUse
	BlockKindSystem
	// BlockKindReasoning represents provider reasoning items (e.g., OpenAI encrypted reasoning).
	BlockKindReasoning
	BlockKindOther
)

// String returns a human-readable identifier for the BlockKind.
func (k BlockKind) String() string {
	switch k {
	case BlockKindUser:
		return "user"
	case BlockKindLLMText:
		return "llm_text"
	case BlockKindToolCall:
		return "tool_call"
	case BlockKindToolUse:
		return "tool_use"
	case BlockKindSystem:
		return "system"
	case BlockKindReasoning:
		return "reasoning"
	case BlockKindOther:
		return "other"
	default:
		return "other"
	}
}

// YAML serialization for BlockKind using stable string names
func (k BlockKind) MarshalYAML() (interface{}, error) {
	return k.String(), nil
}

func (k *BlockKind) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		*k = BlockKindOther
		return nil
	}
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "user":
		*k = BlockKindUser
	case "llm_text":
		*k = BlockKindLLMText
	case "tool_call":
		*k = BlockKindToolCall
	case "tool_use":
		*k = BlockKindToolUse
	case "system":
		*k = BlockKindSystem
	case "reasoning":
		*k = BlockKindReasoning
	case "other", "":
		*k = BlockKindOther
	default:
		// Unknown kind – map to Other but keep original for visibility
		*k = BlockKindOther
	}
	return nil
}

// Block represents a single atomic unit within a Turn.
type Block struct {
	ID      string         `yaml:"id,omitempty"`
	TurnID  string         `yaml:"turn_id,omitempty"`
	Kind    BlockKind      `yaml:"kind"`
	Role    string         `yaml:"role,omitempty"`
	Payload map[string]any `yaml:"payload,omitempty"`
	// Metadata stores arbitrary metadata about the block
	Metadata map[BlockMetadataKey]interface{} `yaml:"metadata,omitempty"`
}

// Turn contains an ordered list of Blocks and associated metadata.
type Turn struct {
	ID     string  `yaml:"id,omitempty"`
	RunID  string  `yaml:"run_id,omitempty"`
	Blocks []Block `yaml:"blocks"`
	// Metadata stores arbitrary metadata about the turn
	Metadata map[TurnMetadataKey]interface{} `yaml:"metadata,omitempty"`
	// Data stores the application data payload associated with this turn
	Data map[TurnDataKey]interface{} `yaml:"data,omitempty"`
}

// PrependBlock inserts a block at the beginning of the Turn's block slice.
func PrependBlock(t *Turn, b Block) {
	if t == nil {
		return
	}
	t.Blocks = append([]Block{b}, t.Blocks...)
}

// Run captures a multi-turn session.
type Run struct {
	ID    string
	Name  string
	Turns []Turn
	// Metadata stores arbitrary metadata about the run
	Metadata map[RunMetadataKey]interface{}
}

// AppendBlock appends a Block to a Turn, assigning an order if missing.
func AppendBlock(t *Turn, b Block) {
	t.Blocks = append(t.Blocks, b)
}

// AppendBlocks appends multiple Blocks ensuring increasing order.
func AppendBlocks(t *Turn, blocks ...Block) {
	for _, b := range blocks {
		AppendBlock(t, b)
	}
}

// FindLastBlocksByKind returns blocks of the requested kinds from the Turn ordered by Order asc.
func FindLastBlocksByKind(t Turn, kinds ...BlockKind) []Block {
	lookup := map[BlockKind]bool{}
	for _, k := range kinds {
		lookup[k] = true
	}
	ret := make([]Block, 0, len(t.Blocks))
	for _, b := range t.Blocks {
		if lookup[b.Kind] {
			ret = append(ret, b)
		}
	}
	return ret
}

// SetTurnMetadata inserts or updates a Turn metadata entry keyed by typed key.
func SetTurnMetadata(t *Turn, key TurnMetadataKey, value interface{}) {
	if t.Metadata == nil {
		t.Metadata = make(map[TurnMetadataKey]interface{})
	}
	t.Metadata[key] = value
}

// SetBlockMetadata inserts or updates a Block metadata entry by typed key.
func SetBlockMetadata(b *Block, key BlockMetadataKey, value interface{}) {
	if b.Metadata == nil {
		b.Metadata = make(map[BlockMetadataKey]interface{})
	}
	b.Metadata[key] = value
}
