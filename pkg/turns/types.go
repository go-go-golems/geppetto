package turns

import (
	"sort"
)

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

// MetadataKV stores a metadata key/value with a source namespace.
type MetadataKV struct {
	Source string
	Key    string
	Value  string
}

// Block represents a single atomic unit within a Turn.
type Block struct {
	ID       string
	TurnID   string
	Order    int
	Kind     BlockKind
	Role     string
	Payload  map[string]any
	Metadata []MetadataKV
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

// Run captures a multi-turn session.
type Run struct {
	ID       string
	Name     string
	Turns    []Turn
	Metadata []MetadataKV
}

// AppendBlock appends a Block to a Turn, assigning an order if missing.
func AppendBlock(t *Turn, b Block) {
	nextOrder := len(t.Blocks)
	if b.Order <= 0 {
		b.Order = nextOrder
	}
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
	sort.Slice(ret, func(i, j int) bool { return ret[i].Order < ret[j].Order })
	return ret
}

// UpsertTurnMetadata inserts or updates a Turn metadata entry keyed by (source,key).
func SetTurnMetadata(t *Turn, key string, value interface{}) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]interface{})
	}
	t.Metadata[key] = value
}

// UpsertBlockMetadata inserts or updates a Block metadata entry keyed by (source,key).
func UpsertBlockMetadata(b *Block, kv MetadataKV) {
	for i := range b.Metadata {
		if b.Metadata[i].Source == kv.Source && b.Metadata[i].Key == kv.Key {
			b.Metadata[i].Value = kv.Value
			return
		}
	}
	b.Metadata = append(b.Metadata, kv)
}
