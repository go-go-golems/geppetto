package factory

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelperFunctionsExist(t *testing.T) {
	// Test that the helper functions exist and can be called
	// This is a basic smoke test without requiring API keys

	// Create basic step settings
	stepSettings, err := settings.NewStepSettings()
	require.NoError(t, err)

	// Test NewEngineFromStepSettings function exists (will fail due to missing API key, but that's expected)
	_, err = NewEngineFromStepSettings(stepSettings)
	assert.Error(t, err)                              // Expected to fail due to missing API key
	assert.Contains(t, err.Error(), "openai-api-key") // Should mention the missing API key

	// Test NewEngineFromParsedLayers function exists
	parsedValues := values.New()
	_, err = NewEngineFromParsedLayers(parsedValues)
	assert.Error(t, err) // Expected to fail due to missing required settings
}

func TestHelperFunctionSignatures(t *testing.T) {
	// Test that the function signatures are correct by checking they compile
	// and can accept the expected parameters

	stepSettings, err := settings.NewStepSettings()
	require.NoError(t, err)

	parsedValues := values.New()

	// These should compile but will fail at runtime due to missing config
	_ = func() (engine.Engine, error) { return NewEngineFromStepSettings(stepSettings) }
	_ = func() (engine.Engine, error) { return NewEngineFromParsedLayers(parsedValues) }

	// If we get here, the function signatures are correct
	assert.True(t, true)
}
