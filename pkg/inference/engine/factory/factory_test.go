package factory

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	openai_responses "github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardEngineFactory_SupportedProviders(t *testing.T) {
	factory := NewStandardEngineFactory()

	providers := factory.SupportedProviders()

	assert.Contains(t, providers, string(types.ApiTypeOpenAI))
	assert.Contains(t, providers, string(types.ApiTypeClaude))
	assert.Contains(t, providers, "anthropic")
	assert.NotEmpty(t, providers)
}

func TestStandardEngineFactory_DefaultProvider(t *testing.T) {
	factory := NewStandardEngineFactory()

	defaultProvider := factory.DefaultProvider()

	assert.Equal(t, string(types.ApiTypeOpenAI), defaultProvider)
}

func TestStandardEngineFactory_CreateEngine_NilSettings(t *testing.T) {
	factory := NewStandardEngineFactory()

	engine, err := factory.CreateEngine(nil)

	assert.Nil(t, engine)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "settings cannot be nil")
}

func TestStandardEngineFactory_CreateEngine_OpenAI_Success(t *testing.T) {
	factory := NewStandardEngineFactory()

	// Create minimal valid settings for OpenAI
	settings := createValidOpenAISettings()

	engine, err := factory.CreateEngine(settings)

	require.NoError(t, err)
	assert.NotNil(t, engine)
	assert.IsType(t, &openai.OpenAIEngine{}, engine)
}

func TestStandardEngineFactory_CreateEngine_Claude_Success(t *testing.T) {
	factory := NewStandardEngineFactory()

	// Create minimal valid settings for Claude
	settings := createValidClaudeSettings()

	engine, err := factory.CreateEngine(settings)

	require.NoError(t, err)
	assert.NotNil(t, engine)
	assert.IsType(t, &claude.ClaudeEngine{}, engine)
}

func TestStandardEngineFactory_CreateEngine_UnsupportedProvider(t *testing.T) {
	factory := NewStandardEngineFactory()

	settings, err := settings.NewStepSettings()
	require.NoError(t, err)

	// Set an unsupported provider
	unsupportedProvider := types.ApiType("unsupported")
	settings.Chat.ApiType = &unsupportedProvider

	engine, err := factory.CreateEngine(settings)

	assert.Nil(t, engine)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestStandardEngineFactory_CreateEngine_MissingAPIKey(t *testing.T) {
	factory := NewStandardEngineFactory()

	settings, err := settings.NewStepSettings()
	require.NoError(t, err)

	openaiType := types.ApiTypeOpenAI
	settings.Chat.ApiType = &openaiType
	// Don't set API key - this should cause validation error

	engine, err := factory.CreateEngine(settings)

	assert.Nil(t, engine)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing API key")
}

func TestStandardEngineFactory_CreateEngine_DefaultsToOpenAI(t *testing.T) {
	factory := NewStandardEngineFactory()

	settings := createValidOpenAISettings()
	// Don't set ApiType - should default to OpenAI
	settings.Chat.ApiType = nil

	engine, err := factory.CreateEngine(settings)

	require.NoError(t, err)
	assert.NotNil(t, engine)
	assert.IsType(t, &openai.OpenAIEngine{}, engine)
}

func TestStandardEngineFactory_CreateEngine_AutoRoutesReasoningModelsToResponses(t *testing.T) {
	factory := NewStandardEngineFactory()

	settings := createValidOpenAISettings()
	settings.Chat.ApiType = nil // default provider path
	engineName := "gpt-5-mini"
	settings.Chat.Engine = &engineName

	engine, err := factory.CreateEngine(settings)

	require.NoError(t, err)
	assert.NotNil(t, engine)
	assert.IsType(t, &openai_responses.Engine{}, engine)
}

// Helper function to create valid OpenAI settings for testing
func createValidOpenAISettings() *settings.StepSettings {
	settings, _ := settings.NewStepSettings()

	openaiType := types.ApiTypeOpenAI
	settings.Chat.ApiType = &openaiType
	settings.API.APIKeys["openai-api-key"] = "test-api-key"
	settings.API.BaseUrls["openai-base-url"] = "https://api.openai.com/v1"

	return settings
}

// Helper function to create valid Claude settings for testing
func createValidClaudeSettings() *settings.StepSettings {
	settings, _ := settings.NewStepSettings()

	claudeType := types.ApiTypeClaude
	settings.Chat.ApiType = &claudeType
	settings.API.APIKeys["claude-api-key"] = "test-api-key"
	settings.API.BaseUrls["claude-base-url"] = "https://api.anthropic.com"

	return settings
}
