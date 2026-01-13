package conversation

import (
	"fmt"
	"maps"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/google/uuid"
)

// ConversationState is the canonical container for multi-turn blocks.
type ConversationState struct {
	ID       string
	RunID    string
	Blocks   []turns.Block
	Data     turns.Data
	Metadata turns.Metadata
	Version  int64
}

// OrderingStrategy applies deterministic ordering rules to a block slice.
type OrderingStrategy func([]turns.Block) ([]turns.Block, error)

// SnapshotConfig controls how a Turn snapshot is produced.
type SnapshotConfig struct {
	IncludeSystem     bool
	IncludeToolBlocks bool
	IncludeReasoning  bool

	NormalizeOrdering bool
	OrderingStrategy  OrderingStrategy

	EnforceResponsesAdj bool
	EnforceToolPairing  bool
}

// DefaultSnapshotConfig returns a configuration that includes all block kinds.
func DefaultSnapshotConfig() SnapshotConfig {
	return SnapshotConfig{
		IncludeSystem:     true,
		IncludeToolBlocks: true,
		IncludeReasoning:  true,
	}
}

// NewConversationState creates a new ConversationState with a stable ID.
func NewConversationState(runID string) *ConversationState {
	return &ConversationState{
		ID:    uuid.NewString(),
		RunID: runID,
	}
}

// Apply applies a single mutation and increments the version.
func (cs *ConversationState) Apply(m Mutation) error {
	if cs == nil {
		return fmt.Errorf("conversation state is nil")
	}
	if m == nil {
		return fmt.Errorf("mutation is nil")
	}
	if err := m.Apply(cs); err != nil {
		return fmt.Errorf("mutation %s failed: %w", m.Name(), err)
	}
	cs.Version++
	return nil
}

// ApplyAll applies multiple mutations sequentially.
func (cs *ConversationState) ApplyAll(muts ...Mutation) error {
	for _, m := range muts {
		if err := cs.Apply(m); err != nil {
			return err
		}
	}
	return nil
}

// Snapshot returns a Turn for inference based on the canonical blocks.
func (cs *ConversationState) Snapshot(cfg SnapshotConfig) (*turns.Turn, error) {
	blocks, err := cs.snapshotBlocks(cfg)
	if err != nil {
		return nil, err
	}
	id := cs.ID
	if id == "" {
		id = uuid.NewString()
	}
	return &turns.Turn{
		ID:       id,
		RunID:    cs.RunID,
		Blocks:   blocks,
		Data:     cs.Data.Clone(),
		Metadata: cs.Metadata.Clone(),
	}, nil
}

// Validate runs snapshot validation rules without returning a Turn.
func (cs *ConversationState) Validate(cfg SnapshotConfig) error {
	_, err := cs.snapshotBlocks(cfg)
	return err
}

func (cs *ConversationState) snapshotBlocks(cfg SnapshotConfig) ([]turns.Block, error) {
	if cs == nil {
		return nil, fmt.Errorf("conversation state is nil")
	}
	cfg = normalizeSnapshotConfig(cfg)
	blocks := make([]turns.Block, 0, len(cs.Blocks))
	for _, b := range cs.Blocks {
		if !cfg.includeBlock(b) {
			continue
		}
		blocks = append(blocks, cloneBlock(b))
	}
	if cfg.NormalizeOrdering {
		if cfg.OrderingStrategy == nil {
			return nil, fmt.Errorf("normalize ordering requires ordering strategy")
		}
		ordered, err := cfg.OrderingStrategy(blocks)
		if err != nil {
			return nil, err
		}
		blocks = ordered
	}
	if err := validateBlocks(blocks, cfg); err != nil {
		return nil, err
	}
	return blocks, nil
}

func normalizeSnapshotConfig(cfg SnapshotConfig) SnapshotConfig {
	if !cfg.IncludeSystem && !cfg.IncludeToolBlocks && !cfg.IncludeReasoning {
		cfg.IncludeSystem = true
		cfg.IncludeToolBlocks = true
		cfg.IncludeReasoning = true
	}
	return cfg
}

func (cfg SnapshotConfig) includeBlock(b turns.Block) bool {
	switch b.Kind {
	case turns.BlockKindSystem:
		return cfg.IncludeSystem
	case turns.BlockKindToolCall, turns.BlockKindToolUse:
		return cfg.IncludeToolBlocks
	case turns.BlockKindReasoning:
		return cfg.IncludeReasoning
	case turns.BlockKindUser, turns.BlockKindLLMText, turns.BlockKindOther:
		return true
	}
	return true
}

func cloneBlock(b turns.Block) turns.Block {
	out := b
	if len(b.Payload) > 0 {
		out.Payload = maps.Clone(b.Payload)
	}
	out.Metadata = b.Metadata.Clone()
	return out
}

func validateBlocks(blocks []turns.Block, cfg SnapshotConfig) error {
	if cfg.EnforceResponsesAdj {
		if err := validateReasoningAdjacency(blocks); err != nil {
			return err
		}
	}
	if cfg.EnforceToolPairing {
		if err := validateToolPairing(blocks); err != nil {
			return err
		}
	}
	return nil
}

func validateReasoningAdjacency(blocks []turns.Block) error {
	for i, b := range blocks {
		if b.Kind != turns.BlockKindReasoning {
			continue
		}
		nextIdx := i + 1
		if nextIdx >= len(blocks) {
			return fmt.Errorf("reasoning block %q missing immediate follower", blockID(b))
		}
		next := blocks[nextIdx]
		if next.Kind != turns.BlockKindLLMText && next.Kind != turns.BlockKindToolCall {
			return fmt.Errorf("reasoning block %q followed by %s", blockID(b), next.Kind.String())
		}
	}
	return nil
}

func validateToolPairing(blocks []turns.Block) error {
	seen := make(map[string]bool)
	for _, b := range blocks {
		switch b.Kind {
		case turns.BlockKindToolCall:
			id, _ := b.Payload[turns.PayloadKeyID].(string)
			if id == "" {
				return fmt.Errorf("tool_call block missing id")
			}
			seen[id] = true
		case turns.BlockKindToolUse:
			id, _ := b.Payload[turns.PayloadKeyID].(string)
			if id == "" {
				return fmt.Errorf("tool_use block missing id")
			}
			if !seen[id] {
				return fmt.Errorf("tool_use for unknown id %q", id)
			}
		case turns.BlockKindUser, turns.BlockKindLLMText, turns.BlockKindSystem, turns.BlockKindReasoning, turns.BlockKindOther:
			continue
		}
	}
	return nil
}

func blockID(b turns.Block) string {
	if b.ID != "" {
		return b.ID
	}
	if id, ok := b.Payload[turns.PayloadKeyID].(string); ok && id != "" {
		return id
	}
	return "<unknown>"
}
