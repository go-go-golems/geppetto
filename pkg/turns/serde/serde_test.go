package serde

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLRoundTripTypedMaps(t *testing.T) {
	// Create a Turn with typed map keys
	turn := &turns.Turn{
		ID: "test-turn-id",
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
	require.NoError(t, turns.KeyTurnMetaSessionID.Set(&turn.Metadata, "test-session-id"))
	require.NoError(t, turns.KeyAgentMode.Set(&turn.Data, "test-mode"))
	require.NoError(t, turns.KeyTurnMetaModel.Set(&turn.Metadata, "test-model"))
	require.NoError(t, turns.KeyTurnMetaProvider.Set(&turn.Metadata, "test-provider"))
	require.NoError(t, turns.KeyTurnMetaStopReason.Set(&turn.Metadata, "stop"))
	// also exercise engine-owned typed key (ToolConfig)
	require.NoError(t, engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{
		Enabled:           true,
		ToolChoice:        engine.ToolChoiceAuto,
		MaxIterations:     4,
		ExecutionTimeout:  30 * time.Second,
		MaxParallelTools:  2,
		ToolErrorHandling: engine.ToolErrorRetry,
		RetryConfig: engine.RetryConfig{
			MaxRetries:    3,
			BackoffBase:   2 * time.Second,
			BackoffFactor: 2,
		},
	}))
	type toolParams struct {
		Text string `json:"text"`
	}
	reflector := &jsonschema.Reflector{DoNotReference: true}
	params := reflector.Reflect(toolParams{})
	require.NoError(t, engine.KeyToolDefinitions.Set(&turn.Data, engine.ToolDefinitions{
		{
			Name:        "echo",
			Description: "Echoes text",
			Parameters:  schemaToMapForTest(t, params),
			Examples: []engine.ToolExample{
				{
					Input:       map[string]any{"text": "hello"},
					Output:      map[string]any{"echo": "hello"},
					Description: "Simple echo example",
				},
			},
			Tags:    []string{"debug"},
			Version: "v1",
		},
	}))

	require.NoError(t, turns.KeyBlockMetaMiddleware.Set(&turn.Blocks[0].Metadata, "test-middleware"))
	// Help Go inference pick T=any (key is BlockMetaKey[any]) rather than T=[]any (from the literal).
	require.NoError(t, turns.KeyBlockMetaClaudeOriginalContent.Set(&turn.Blocks[0].Metadata, any([]any{"test-content"})))

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
	gotSessionID, ok, err := turns.KeyTurnMetaSessionID.Get(roundTripTurn.Metadata)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-session-id", gotSessionID, "SessionID should match")

	// Verify Data contents
	gotMode, ok, err := turns.KeyAgentMode.Get(roundTripTurn.Data)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-mode", gotMode, "AgentMode should match")

	// YAML round-trip decodes structs as map[string]any, but typed keys now best-effort decode
	// structured values via JSON re-marshal/unmarshal.
	toolCfg, ok, err := engine.KeyToolConfig.Get(roundTripTurn.Data)
	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, toolCfg.Enabled, "ToolConfig.enabled should match")
	assert.Equal(t, engine.ToolChoiceAuto, toolCfg.ToolChoice, "ToolConfig.tool_choice should match")
	assert.Equal(t, 4, toolCfg.MaxIterations, "ToolConfig.max_iterations should match")
	assert.Equal(t, 30*time.Second, toolCfg.ExecutionTimeout, "ToolConfig.execution_timeout should match")
	assert.Equal(t, 2, toolCfg.MaxParallelTools, "ToolConfig.max_parallel_tools should match")
	assert.Equal(t, engine.ToolErrorRetry, toolCfg.ToolErrorHandling, "ToolConfig.tool_error_handling should match")
	assert.Equal(t, 3, toolCfg.RetryConfig.MaxRetries, "ToolConfig.retry_config.max_retries should match")
	assert.Equal(t, 2*time.Second, toolCfg.RetryConfig.BackoffBase, "ToolConfig.retry_config.backoff_base should match")
	assert.Equal(t, 2.0, toolCfg.RetryConfig.BackoffFactor, "ToolConfig.retry_config.backoff_factor should match")
	// Assert the decoded map form is present and has the expected fields.
	rawToolCfgKey := turns.DataK[any](turns.GeppettoNamespaceKey, turns.ToolConfigValueKey, 1)
	rawCfg, ok, err := rawToolCfgKey.Get(roundTripTurn.Data)
	require.NoError(t, err)
	require.True(t, ok)
	cfgMap, ok := rawCfg.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, cfgMap["enabled"], "ToolConfig.enabled should match")
	assert.Equal(t, "auto", cfgMap["tool_choice"], "ToolConfig.tool_choice should use stable YAML field names")

	toolDefs, ok, err := engine.KeyToolDefinitions.Get(roundTripTurn.Data)
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, toolDefs, 1)
	assert.Equal(t, "echo", toolDefs[0].Name, "ToolDefinitions[0].name should match")
	assert.Equal(t, "Echoes text", toolDefs[0].Description, "ToolDefinitions[0].description should match")
	require.NotNil(t, toolDefs[0].Parameters, "ToolDefinitions[0].parameters should be present")
	assert.Equal(t, "object", toolDefs[0].Parameters["type"], "ToolDefinitions[0].parameters.type should match")
	assert.Equal(t, []string{"debug"}, toolDefs[0].Tags, "ToolDefinitions[0].tags should match")
	assert.Equal(t, "v1", toolDefs[0].Version, "ToolDefinitions[0].version should match")
	require.Len(t, toolDefs[0].Examples, 1)
	assert.Equal(t, "Simple echo example", toolDefs[0].Examples[0].Description, "ToolDefinitions[0].examples[0].description should match")

	// Verify Metadata contents
	gotModel, ok, err := turns.KeyTurnMetaModel.Get(roundTripTurn.Metadata)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-model", gotModel, "Model should match")

	gotProvider, ok, err := turns.KeyTurnMetaProvider.Get(roundTripTurn.Metadata)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-provider", gotProvider, "Provider should match")

	gotStop, ok, err := turns.KeyTurnMetaStopReason.Get(roundTripTurn.Metadata)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "stop", gotStop, "StopReason should match")

	// Verify Block Metadata map contents
	require.Len(t, roundTripTurn.Blocks, 1, "Should have one block")
	block := roundTripTurn.Blocks[0]
	assert.Equal(t, turn.Blocks[0].ID, block.ID, "Block ID should match")
	assert.Equal(t, turn.Blocks[0].Kind, block.Kind, "Block Kind should match")
	gotMW, ok, err := turns.KeyBlockMetaMiddleware.Get(block.Metadata)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "test-middleware", gotMW, "Middleware metadata should match")
}

func TestYAMLRoundTripEmptyMaps(t *testing.T) {
	// Test with empty maps
	turn := &turns.Turn{
		ID: "test-turn-id",
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

func schemaToMapForTest(t *testing.T, schema *jsonschema.Schema) map[string]any {
	t.Helper()

	b, err := json.Marshal(schema)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}
