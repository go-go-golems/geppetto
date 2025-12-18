package serde

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLRoundTripTypedMaps(t *testing.T) {
	// Create a Turn with typed map keys
	turn := &turns.Turn{
		ID:    "test-turn-id",
		RunID: "test-run-id",
		Data: map[turns.TurnDataKey]interface{}{
			turns.DataKeyAgentMode: "test-mode",
		},
		Metadata: map[turns.TurnMetadataKey]interface{}{
			turns.TurnMetaKeyModel:      "test-model",
			turns.TurnMetaKeyProvider:   "test-provider",
			turns.TurnMetaKeyStopReason: "stop",
		},
		Blocks: []turns.Block{
			{
				ID:   "block-1",
				Kind: turns.BlockKindSystem,
				Metadata: map[turns.BlockMetadataKey]interface{}{
					turns.BlockMetaKeyMiddleware:            "test-middleware",
					turns.BlockMetaKeyClaudeOriginalContent: []interface{}{"test-content"},
				},
				Payload: map[string]interface{}{
					"text": "test system message",
				},
			},
		},
	}

	// Marshal to YAML
	yamlData, err := ToYAML(turn, Options{})
	require.NoError(t, err, "ToYAML should succeed")
	require.NotEmpty(t, yamlData, "YAML data should not be empty")

	// Unmarshal from YAML
	roundTripTurn, err := FromYAML(yamlData)
	require.NoError(t, err, "FromYAML should succeed")
	require.NotNil(t, roundTripTurn, "Round-trip turn should not be nil")

	// Verify Data map keys are preserved
	assert.Equal(t, turn.ID, roundTripTurn.ID, "Turn ID should match")
	assert.Equal(t, turn.RunID, roundTripTurn.RunID, "Run ID should match")

	// Verify Data map contents
	assert.Equal(t, turn.Data[turns.DataKeyAgentMode], roundTripTurn.Data[turns.DataKeyAgentMode], "AgentMode should match")

	// Verify Metadata map contents
	assert.Equal(t, turn.Metadata[turns.TurnMetaKeyModel], roundTripTurn.Metadata[turns.TurnMetaKeyModel], "Model should match")
	assert.Equal(t, turn.Metadata[turns.TurnMetaKeyProvider], roundTripTurn.Metadata[turns.TurnMetaKeyProvider], "Provider should match")
	assert.Equal(t, turn.Metadata[turns.TurnMetaKeyStopReason], roundTripTurn.Metadata[turns.TurnMetaKeyStopReason], "StopReason should match")

	// Verify Block Metadata map contents
	require.Len(t, roundTripTurn.Blocks, 1, "Should have one block")
	block := roundTripTurn.Blocks[0]
	assert.Equal(t, turn.Blocks[0].ID, block.ID, "Block ID should match")
	assert.Equal(t, turn.Blocks[0].Kind, block.Kind, "Block Kind should match")
	assert.Equal(t, turn.Blocks[0].Metadata[turns.BlockMetaKeyMiddleware], block.Metadata[turns.BlockMetaKeyMiddleware], "Middleware metadata should match")
}

func TestYAMLRoundTripEmptyMaps(t *testing.T) {
	// Test with empty maps
	turn := &turns.Turn{
		ID:       "test-turn-id",
		RunID:    "test-run-id",
		Data:     map[turns.TurnDataKey]interface{}{},
		Metadata: map[turns.TurnMetadataKey]interface{}{},
		Blocks: []turns.Block{
			{
				ID:       "block-1",
				Kind:     turns.BlockKindUser,
				Metadata: map[turns.BlockMetadataKey]interface{}{},
				Payload:  map[string]interface{}{},
			},
		},
	}

	yamlData, err := ToYAML(turn, Options{})
	require.NoError(t, err, "ToYAML should succeed")

	roundTripTurn, err := FromYAML(yamlData)
	require.NoError(t, err, "FromYAML should succeed")
	require.NotNil(t, roundTripTurn, "Round-trip turn should not be nil")

	assert.Equal(t, turn.ID, roundTripTurn.ID, "Turn ID should match")
	assert.NotNil(t, roundTripTurn.Data, "Data map should be initialized")
	assert.NotNil(t, roundTripTurn.Metadata, "Metadata map should be initialized")
	require.Len(t, roundTripTurn.Blocks, 1, "Should have one block")
	assert.NotNil(t, roundTripTurn.Blocks[0].Metadata, "Block Metadata map should be initialized")
}
