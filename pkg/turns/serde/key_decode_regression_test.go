package serde

import (
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/require"
)

func TestFromYAML_TypedKeysDecodeFromGenericValues(t *testing.T) {
	y := []byte(`
id: t
blocks: []
data:
  geppetto.tool_config@v1:
    enabled: true
    tool_choice: required
    max_parallel_tools: 2
    execution_timeout: 2s
    retry_config:
      max_retries: 2
      backoff_base: 100ms
      backoff_factor: 2.0
  geppetto.agent_mode_allowed_tools@v1:
    - search
    - calc
`)
	turn, err := FromYAML(y)
	require.NoError(t, err)

	toolCfg, ok, err := engine.KeyToolConfig.Get(turn.Data)
	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, toolCfg.Enabled)
	require.Equal(t, engine.ToolChoiceRequired, toolCfg.ToolChoice)
	require.Equal(t, 2, toolCfg.MaxParallelTools)
	require.Equal(t, 2*time.Second, toolCfg.ExecutionTimeout)
	require.Equal(t, 2, toolCfg.RetryConfig.MaxRetries)
	require.Equal(t, 100*time.Millisecond, toolCfg.RetryConfig.BackoffBase)
	require.Equal(t, 2.0, toolCfg.RetryConfig.BackoffFactor)

	allowed, ok, err := turns.KeyAgentModeAllowedTools.Get(turn.Data)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"search", "calc"}, allowed)
}

func TestTypedKeyDecode_ErrorOnUncoercibleSliceElement(t *testing.T) {
	y := []byte(`
id: t
blocks: []
data:
  geppetto.agent_mode_allowed_tools@v1:
    - search
    - 123
`)
	turn, err := FromYAML(y)
	require.NoError(t, err)

	_, ok, err := turns.KeyAgentModeAllowedTools.Get(turn.Data)
	require.Error(t, err)
	require.True(t, ok)
}
