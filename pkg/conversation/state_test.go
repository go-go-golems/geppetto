package conversation

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/require"
)

func TestSnapshotReasoningAdjacencyRequiresFollower(t *testing.T) {
	cs := &ConversationState{
		Blocks: []turns.Block{
			reasoningBlock("r1"),
			turns.NewUserTextBlock("hello"),
		},
	}
	cfg := DefaultSnapshotConfig()
	cfg.EnforceResponsesAdj = true

	_, err := cs.Snapshot(cfg)
	require.Error(t, err)
}

func TestSnapshotReasoningAdjacencyAllowsAssistant(t *testing.T) {
	cs := &ConversationState{
		Blocks: []turns.Block{
			reasoningBlock("r1"),
			turns.NewAssistantTextBlock("done"),
		},
	}
	cfg := DefaultSnapshotConfig()
	cfg.EnforceResponsesAdj = true

	_, err := cs.Snapshot(cfg)
	require.NoError(t, err)
}

func TestSnapshotReasoningAdjacencyAllowsToolCall(t *testing.T) {
	cs := &ConversationState{
		Blocks: []turns.Block{
			reasoningBlock("r1"),
			turns.NewToolCallBlock("call-1", "lookup", map[string]any{"id": 1}),
			turns.NewToolUseBlock("call-1", "ok"),
		},
	}
	cfg := DefaultSnapshotConfig()
	cfg.EnforceResponsesAdj = true

	_, err := cs.Snapshot(cfg)
	require.NoError(t, err)
}

func TestSnapshotToolPairingRejectsMissingCall(t *testing.T) {
	cs := &ConversationState{
		Blocks: []turns.Block{
			turns.NewToolUseBlock("call-1", "ok"),
		},
	}
	cfg := DefaultSnapshotConfig()
	cfg.EnforceToolPairing = true

	_, err := cs.Snapshot(cfg)
	require.Error(t, err)
}

func TestSnapshotToolPairingAcceptsCall(t *testing.T) {
	cs := &ConversationState{
		Blocks: []turns.Block{
			turns.NewToolCallBlock("call-1", "lookup", map[string]any{"id": 1}),
			turns.NewToolUseBlock("call-1", "ok"),
		},
	}
	cfg := DefaultSnapshotConfig()
	cfg.EnforceToolPairing = true

	_, err := cs.Snapshot(cfg)
	require.NoError(t, err)
}

func reasoningBlock(id string) turns.Block {
	return turns.Block{
		ID:      id,
		Kind:    turns.BlockKindReasoning,
		Payload: map[string]any{turns.PayloadKeyEncryptedContent: "enc"},
	}
}
