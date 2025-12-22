package serde

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLRoundTripTypedMaps(t *testing.T) {
	// Create a Turn with typed map keys
	turn := &turns.Turn{
		ID:    "test-turn-id",
		RunID: "test-run-id",
		Blocks: []turns.Block{
			{
				ID:   "block-1",
				Kind: turns.BlockKindSystem,
				Payload: map[string]interface{}{
					"text": "test system message",
				},
			},
		},
	}
	require.NoError(t, turns.DataSet(&turn.Data, turns.KeyAgentMode, "test-mode"))
	require.NoError(t, turns.MetadataSet(&turn.Metadata, turns.KeyTurnMetaModel, "test-model"))
	require.NoError(t, turns.MetadataSet(&turn.Metadata, turns.KeyTurnMetaProvider, "test-provider"))
	require.NoError(t, turns.MetadataSet(&turn.Metadata, turns.KeyTurnMetaStopReason, "stop"))
	// also exercise engine-owned typed key (ToolConfig)
	require.NoError(t, turns.DataSet(&turn.Data, engine.KeyToolConfig, engine.ToolConfig{Enabled: true}))

	require.NoError(t, turns.BlockMetadataSet(&turn.Blocks[0].Metadata, turns.KeyBlockMetaMiddleware, "test-middleware"))
	require.NoError(t, turns.BlockMetadataSet(&turn.Blocks[0].Metadata, turns.KeyBlockMetaClaudeOriginalContent, []any{"test-content"}))

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

	// Verify Data contents
	gotMode, ok, err := turns.DataGet(roundTripTurn.Data, turns.KeyAgentMode)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-mode", gotMode, "AgentMode should match")

	gotCfg, ok, err := turns.DataGet(roundTripTurn.Data, engine.KeyToolConfig)
	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, gotCfg.Enabled, "ToolConfig should match")

	// Verify Metadata contents
	gotModel, ok, err := turns.MetadataGet(roundTripTurn.Metadata, turns.KeyTurnMetaModel)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-model", gotModel, "Model should match")

	gotProvider, ok, err := turns.MetadataGet(roundTripTurn.Metadata, turns.KeyTurnMetaProvider)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-provider", gotProvider, "Provider should match")

	gotStop, ok, err := turns.MetadataGet(roundTripTurn.Metadata, turns.KeyTurnMetaStopReason)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "stop", gotStop, "StopReason should match")

	// Verify Block Metadata map contents
	require.Len(t, roundTripTurn.Blocks, 1, "Should have one block")
	block := roundTripTurn.Blocks[0]
	assert.Equal(t, turn.Blocks[0].ID, block.ID, "Block ID should match")
	assert.Equal(t, turn.Blocks[0].Kind, block.Kind, "Block Kind should match")
	gotMW, ok, err := turns.BlockMetadataGet(block.Metadata, turns.KeyBlockMetaMiddleware)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-middleware", gotMW, "Middleware metadata should match")
}

func TestYAMLRoundTripEmptyMaps(t *testing.T) {
	// Test with empty maps
	turn := &turns.Turn{
		ID:       "test-turn-id",
		RunID:    "test-run-id",
		Blocks: []turns.Block{
			{
				ID:       "block-1",
				Kind:     turns.BlockKindUser,
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
	assert.Equal(t, 0, roundTripTurn.Data.Len(), "Data should be empty")
	assert.Equal(t, 0, roundTripTurn.Metadata.Len(), "Metadata should be empty")
	require.Len(t, roundTripTurn.Blocks, 1, "Should have one block")
	assert.Equal(t, 0, roundTripTurn.Blocks[0].Metadata.Len(), "Block Metadata should be empty")
}
