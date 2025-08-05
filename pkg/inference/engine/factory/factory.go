package factory

import (
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/pkg/errors"
)

// EngineFactory creates AI inference engines based on provider settings.
// This interface allows external control over which AI provider engine is used
// without the calling code needing to know specific implementations.
type EngineFactory interface {
	// CreateEngine creates an Engine instance based on the provided settings.
	// The actual provider is determined from settings.Chat.ApiType.
	// Returns an error if the provider is unsupported or configuration is invalid.
	CreateEngine(settings *settings.StepSettings, options ...engine.Option) (engine.Engine, error)

	// SupportedProviders returns a list of provider names this factory supports.
	// Provider names match the ApiType constants (e.g., "openai", "claude", "gemini").
	SupportedProviders() []string

	// DefaultProvider returns the name of the default provider used when
	// settings.Chat.ApiType is nil or not specified.
	DefaultProvider() string
}

// StandardEngineFactory is the default implementation of EngineFactory.
// It supports creating engines for OpenAI, Claude, and other configured providers.
// Provider selection is based on settings.Chat.ApiType with fallback to OpenAI.
type StandardEngineFactory struct {
	// ClaudeTools are the tools to pass to Claude engines
	// This can be empty for basic text generation
	ClaudeTools []api.Tool
}

// NewStandardEngineFactory creates a new StandardEngineFactory with optional Claude tools.
func NewStandardEngineFactory(claudeTools ...api.Tool) *StandardEngineFactory {
	return &StandardEngineFactory{
		ClaudeTools: claudeTools,
	}
}

// CreateEngine creates an Engine instance based on the provider specified in settings.Chat.ApiType.
// If no ApiType is specified, defaults to OpenAI.
// Supported providers: openai, anyscale, fireworks, claude, anthropic, gemini.
func (f *StandardEngineFactory) CreateEngine(settings *settings.StepSettings, options ...engine.Option) (engine.Engine, error) {
	if settings == nil {
		return nil, errors.New("settings cannot be nil")
	}

	// Determine provider from settings
	provider := f.DefaultProvider()
	if settings.Chat != nil && settings.Chat.ApiType != nil {
		provider = strings.ToLower(string(*settings.Chat.ApiType))
	}

	// Validate that we have the required settings
	if err := f.validateSettings(settings, provider); err != nil {
		return nil, errors.Wrapf(err, "invalid settings for provider %s", provider)
	}

	// Create engine based on provider
	switch provider {
	case string(types.ApiTypeOpenAI), string(types.ApiTypeAnyScale), string(types.ApiTypeFireworks):
		return openai.NewOpenAIEngine(settings, options...)

	case string(types.ApiTypeClaude), "anthropic":
		return claude.NewClaudeEngine(settings, f.ClaudeTools, options...)

	case string(types.ApiTypeGemini):
		// TODO: Implement GeminiEngine when available
		return nil, errors.Errorf("provider %s is not yet implemented", provider)

	default:
		supported := strings.Join(f.SupportedProviders(), ", ")
		return nil, errors.Errorf("unsupported provider %s. Supported providers: %s", provider, supported)
	}
}

// SupportedProviders returns the list of AI providers this factory can create engines for.
func (f *StandardEngineFactory) SupportedProviders() []string {
	return []string{
		string(types.ApiTypeOpenAI),
		string(types.ApiTypeAnyScale),
		string(types.ApiTypeFireworks),
		string(types.ApiTypeClaude),
		"anthropic", // alias for claude
		// string(types.ApiTypeGemini), // TODO: uncomment when implemented
	}
}

// DefaultProvider returns the default provider name used when no ApiType is specified.
func (f *StandardEngineFactory) DefaultProvider() string {
	return string(types.ApiTypeOpenAI)
}

// validateSettings performs basic validation of settings for the specified provider.
func (f *StandardEngineFactory) validateSettings(settings *settings.StepSettings, provider string) error {
	if settings.Chat == nil {
		return errors.New("chat settings cannot be nil")
	}

	if settings.API == nil {
		return errors.New("API settings cannot be nil")
	}

	// Validate provider-specific requirements
	switch provider {
	case string(types.ApiTypeOpenAI), string(types.ApiTypeAnyScale), string(types.ApiTypeFireworks):
		return f.validateOpenAISettings(settings, provider)

	case string(types.ApiTypeClaude), "anthropic":
		return f.validateClaudeSettings(settings, provider)

	case string(types.ApiTypeGemini):
		return f.validateGeminiSettings(settings, provider)

	default:
		return errors.Errorf("unknown provider %s", provider)
	}
}

// validateOpenAISettings validates settings required for OpenAI-compatible providers.
func (f *StandardEngineFactory) validateOpenAISettings(settings *settings.StepSettings, provider string) error {
	// Check for API key
	apiKeyName := provider + "-api-key"
	if _, ok := settings.API.APIKeys[apiKeyName]; !ok {
		return errors.Errorf("missing API key %s", apiKeyName)
	}

	// Base URL is optional for OpenAI (uses default), but required for others
	baseURLName := provider + "-base-url"
	if provider != string(types.ApiTypeOpenAI) {
		if _, ok := settings.API.BaseUrls[baseURLName]; !ok {
			return errors.Errorf("missing base URL %s for provider %s", baseURLName, provider)
		}
	}

	return nil
}

// validateClaudeSettings validates settings required for Claude/Anthropic provider.
func (f *StandardEngineFactory) validateClaudeSettings(settings *settings.StepSettings, provider string) error {
	// Claude uses "claude" as the key regardless of "anthropic" alias
	actualProvider := string(types.ApiTypeClaude)

	// Check for API key
	apiKeyName := actualProvider + "-api-key"
	if _, ok := settings.API.APIKeys[apiKeyName]; !ok {
		return errors.Errorf("missing API key %s", apiKeyName)
	}

	// Check for base URL
	baseURLName := actualProvider + "-base-url"
	if _, ok := settings.API.BaseUrls[baseURLName]; !ok {
		return errors.Errorf("missing base URL %s", baseURLName)
	}

	// Check for Claude-specific settings
	if settings.Claude == nil {
		return errors.New("Claude-specific settings cannot be nil")
	}

	// Check for client settings (required by ClaudeEngine)
	if settings.Client == nil {
		return errors.New("client settings cannot be nil for Claude provider")
	}

	return nil
}

// validateGeminiSettings validates settings required for Gemini provider.
func (f *StandardEngineFactory) validateGeminiSettings(settings *settings.StepSettings, provider string) error {
	// Check for API key
	apiKeyName := provider + "-api-key"
	if _, ok := settings.API.APIKeys[apiKeyName]; !ok {
		return errors.Errorf("missing API key %s", apiKeyName)
	}

	// Check for base URL
	baseURLName := provider + "-base-url"
	if _, ok := settings.API.BaseUrls[baseURLName]; !ok {
		return errors.Errorf("missing base URL %s", baseURLName)
	}

	// Check for Gemini-specific settings
	if settings.Gemini == nil {
		return errors.New("Gemini-specific settings cannot be nil")
	}

	return nil
}

// Compile-time check that StandardEngineFactory implements EngineFactory
var _ EngineFactory = (*StandardEngineFactory)(nil)
