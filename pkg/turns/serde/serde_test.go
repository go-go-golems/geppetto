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
	// Help Go inference pick T=any (key is Key[any]) rather than T=[]any (from the literal).
	require.NoError(t, turns.BlockMetadataSet(&turn.Blocks[0].Metadata, turns.KeyBlockMetaClaudeOriginalContent, any([]any{"test-content"})))

	t.Logf("seeded: turn.Data.Len=%d, turn.Metadata.Len=%d, block0.Metadata.Len=%d", turn.Data.Len(), turn.Metadata.Len(), turn.Blocks[0].Metadata.Len())

	// Marshal to YAML
	yamlData, err := ToYAML(turn, Options{})
	require.NoError(t, err, "ToYAML should succeed")
	require.NotEmpty(t, yamlData, "YAML data should not be empty")
	t.Logf("yaml:\n%s", string(yamlData))

	// Unmarshal from YAML
	roundTripTurn, err := FromYAML(yamlData)
	require.NoError(t, err, "FromYAML should succeed")
	require.NotNil(t, roundTripTurn, "Round-trip turn should not be nil")
	t.Logf("roundtrip: turn.Data.Len=%d, turn.Metadata.Len=%d, block0.Metadata.Len=%d", roundTripTurn.Data.Len(), roundTripTurn.Metadata.Len(), roundTripTurn.Blocks[0].Metadata.Len())

	// Verify Data map keys are preserved
	assert.Equal(t, turn.ID, roundTripTurn.ID, "Turn ID should match")
	assert.Equal(t, turn.RunID, roundTripTurn.RunID, "Run ID should match")

	// Verify Data contents
	gotMode, ok, err := turns.DataGet(roundTripTurn.Data, turns.KeyAgentMode)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-mode", gotMode, "AgentMode should match")

	// NOTE: YAML round-trip decodes structs as map[string]any, so strict typed reads for struct values
	// (like engine.ToolConfig) will fail unless we add an explicit type/codec registry or switch storage strategy.
	_, ok, err = turns.DataGet(roundTripTurn.Data, engine.KeyToolConfig)
	require.True(t, ok)
	require.Error(t, err)
	// Assert the decoded map form is present and has the expected fields.
	rawToolCfgKey := turns.K[any](turns.GeppettoNamespaceKey, turns.ToolConfigValueKey, 1)
	rawCfg, ok, err := turns.DataGet(roundTripTurn.Data, rawToolCfgKey)
	require.NoError(t, err)
	require.True(t, ok)
	cfgMap, ok := rawCfg.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, cfgMap["enabled"], "ToolConfig.enabled should match")

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
		ID:    "test-turn-id",
		RunID: "test-run-id",
		Blocks: []turns.Block{
			{
				ID:      "block-1",
				Kind:    turns.BlockKindUser,
				Payload: map[string]interface{}{},
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
