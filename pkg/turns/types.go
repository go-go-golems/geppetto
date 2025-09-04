package turns

// BlockKind represents the kind of a block within a Turn.
type BlockKind int

const (
	BlockKindUser BlockKind = iota
	BlockKindLLMText
	BlockKindToolCall
	BlockKindToolUse
	BlockKindSystem
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
	case BlockKindOther:
		return "other"
	default:
		return "other"
	}
}

// Block represents a single atomic unit within a Turn.
type Block struct {
	ID      string
	TurnID  string
	Kind    BlockKind
	Role    string
	Payload map[string]any
	// Metadata stores arbitrary metadata about the block
	Metadata map[string]interface{}
}

// Turn contains an ordered list of Blocks and associated metadata.
type Turn struct {
	ID     string
	RunID  string
	Blocks []Block
	// Metadata stores arbitrary metadata about the turn
	Metadata map[string]interface{}
	// Data stores the application data payload associated with this turn
	Data map[string]interface{}
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
	Metadata map[string]interface{}
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

// SetTurnMetadata inserts or updates a Turn metadata entry keyed by simple key.
func SetTurnMetadata(t *Turn, key string, value interface{}) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]interface{})
	}
	t.Metadata[key] = value
}

// SetBlockMetadata inserts or updates a Block metadata entry by simple key.
func SetBlockMetadata(b *Block, key string, value interface{}) {
	if b.Metadata == nil {
		b.Metadata = make(map[string]interface{})
	}
	b.Metadata[key] = value
}
