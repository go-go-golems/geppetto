package factory

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	openai_responses "github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"

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
	CreateEngine(settings *settings.InferenceSettings) (engine.Engine, error)

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
	openAIResponsesOptions []openai_responses.EngineOption
	openAIOptions          []openai.EngineOption
	claudeOptions          []claude.EngineOption
	geminiOptions          []gemini.EngineOption
	bearerTokenSource      credentials.BearerTokenSource
}

// StandardEngineFactoryOption configures StandardEngineFactory.
type StandardEngineFactoryOption func(*StandardEngineFactory)

// WithOpenAIResponsesOptions passes options to OpenAI Responses engines created
// by the factory. This keeps generic factory callers compatible while allowing
// apps to attach observer hooks when they explicitly need them.
func WithOpenAIResponsesOptions(opts ...openai_responses.EngineOption) StandardEngineFactoryOption {
	return func(f *StandardEngineFactory) {
		f.openAIResponsesOptions = append(f.openAIResponsesOptions, opts...)
	}
}

// WithOpenAIOptions passes options to OpenAI Chat Completions engines created
// by the factory. This mirrors WithOpenAIResponsesOptions for apps that want
// provider-agnostic construction plus provider-specific observability hooks.
func WithOpenAIOptions(opts ...openai.EngineOption) StandardEngineFactoryOption {
	return func(f *StandardEngineFactory) {
		f.openAIOptions = append(f.openAIOptions, opts...)
	}
}

// WithClaudeOptions passes options to Claude engines created by the factory.
func WithClaudeOptions(opts ...claude.EngineOption) StandardEngineFactoryOption {
	return func(f *StandardEngineFactory) {
		f.claudeOptions = append(f.claudeOptions, opts...)
	}
}

// WithGeminiOptions passes options to Gemini engines created by the factory.
func WithGeminiOptions(opts ...gemini.EngineOption) StandardEngineFactoryOption {
	return func(f *StandardEngineFactory) {
		f.geminiOptions = append(f.geminiOptions, opts...)
	}
}

// WithBearerTokenSource supplies OpenAI-compatible engines with a request-time
// bearer credential source. It is authoritative over static API-key settings.
func WithBearerTokenSource(source credentials.BearerTokenSource) StandardEngineFactoryOption {
	return func(f *StandardEngineFactory) {
		f.bearerTokenSource = source
	}
}

func isResponsesProvider(provider string) bool {
	return provider == string(types.ApiTypeOpenResponses) || provider == string(types.ApiTypeOpenAIResponses)
}

// NewStandardEngineFactory creates a new StandardEngineFactory.
func NewStandardEngineFactory(opts ...StandardEngineFactoryOption) *StandardEngineFactory {
	f := &StandardEngineFactory{}
	for _, opt := range opts {
		if opt != nil {
			opt(f)
		}
	}
	return f
}

// CreateEngine creates an Engine instance based on the provider specified in settings.Chat.ApiType.
// If no ApiType is specified, defaults to OpenAI.
// Supported providers: openai, anyscale, fireworks, claude, anthropic, gemini.
func (f *StandardEngineFactory) CreateEngine(settings *settings.InferenceSettings) (engine.Engine, error) {
	if settings == nil {
		return nil, errors.New("settings cannot be nil")
	}

	// Determine provider from settings
	provider := f.DefaultProvider()
	if settings.Chat != nil && settings.Chat.ApiType != nil {
		provider = strings.ToLower(string(*settings.Chat.ApiType))
	}
	if provider == string(types.ApiTypeOpenAI) && settings != nil && settings.Chat != nil && settings.Chat.Engine != nil {
		model := strings.ToLower(strings.TrimSpace(*settings.Chat.Engine))
		if isReasoningModelForSettings(settings, model) {
			log.Warn().
				Str("model", model).
				Str("provider", provider).
				Str("recommended_provider", string(types.ApiTypeOpenResponses)).
				Msg("Thinking model selected with openai api type; thinking stream events may be missing unless open-responses is used")
		}
	}

	// Validate that we have the required settings
	if err := f.validateSettings(settings, provider); err != nil {
		return nil, errors.Wrapf(err, "invalid settings for provider %s", provider)
	}

	// Create engine based on provider
	switch provider {
	case string(types.ApiTypeOpenAI), string(types.ApiTypeAnyScale), string(types.ApiTypeFireworks):
		opts := append([]openai.EngineOption(nil), f.openAIOptions...)
		if f.bearerTokenSource != nil {
			opts = append(opts, openai.WithBearerTokenSource(f.bearerTokenSource))
		}
		return openai.NewOpenAIEngine(settings, opts...)

	case string(types.ApiTypeOpenResponses), string(types.ApiTypeOpenAIResponses):
		opts := append([]openai_responses.EngineOption(nil), f.openAIResponsesOptions...)
		if f.bearerTokenSource != nil {
			opts = append(opts, openai_responses.WithBearerTokenSource(f.bearerTokenSource))
		}
		return openai_responses.NewEngine(settings, opts...)

	case string(types.ApiTypeClaude), "anthropic":
		opts := append([]claude.EngineOption(nil), f.claudeOptions...)
		if f.bearerTokenSource != nil {
			opts = append(opts, claude.WithBearerTokenSource(f.bearerTokenSource))
		}
		return claude.NewClaudeEngine(settings, opts...)

	case string(types.ApiTypeGemini):
		return gemini.NewGeminiEngine(settings, f.geminiOptions...)

	default:
		supported := strings.Join(f.SupportedProviders(), ", ")
		return nil, errors.Errorf("unsupported provider %s. Supported providers: %s", provider, supported)
	}
}

func isReasoningModelForSettings(settings *settings.InferenceSettings, model string) bool {
	if settings != nil && settings.ModelInfo != nil && settings.ModelInfo.Reasoning != nil {
		return *settings.ModelInfo.Reasoning
	}
	return isReasoningModel(model)
}

func isReasoningModel(model string) bool {
	return strings.HasPrefix(model, "o1") ||
		strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "o4") ||
		strings.HasPrefix(model, "gpt-5")
}

// SupportedProviders returns the list of AI providers this factory can create engines for.
func (f *StandardEngineFactory) SupportedProviders() []string {
	return []string{
		string(types.ApiTypeOpenAI),
		string(types.ApiTypeOpenResponses),
		string(types.ApiTypeOpenAIResponses),
		string(types.ApiTypeAnyScale),
		string(types.ApiTypeFireworks),
		string(types.ApiTypeClaude),
		"anthropic", // alias for claude
		string(types.ApiTypeGemini),
	}
}

// DefaultProvider returns the default provider name used when no ApiType is specified.
func (f *StandardEngineFactory) DefaultProvider() string {
	return string(types.ApiTypeOpenAI)
}

// validateSettings performs basic validation of settings for the specified provider.
func (f *StandardEngineFactory) validateSettings(settings *settings.InferenceSettings, provider string) error {
	if settings.Chat == nil {
		return errors.New("chat settings cannot be nil")
	}

	if settings.API == nil {
		return errors.New("API settings cannot be nil")
	}

	// Validate provider-specific requirements
	switch provider {
	case string(types.ApiTypeOpenAI), string(types.ApiTypeOpenResponses), string(types.ApiTypeOpenAIResponses), string(types.ApiTypeAnyScale), string(types.ApiTypeFireworks):
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
func (f *StandardEngineFactory) validateOpenAISettings(settings *settings.InferenceSettings, provider string) error {
	// A request-time bearer source is authoritative; otherwise preserve the
	// existing static API-key validation and Responses aliases.
	if f.bearerTokenSource == nil {
		apiKeyName := provider + "-api-key"
		if _, ok := settings.API.APIKeys[apiKeyName]; !ok {
			if isResponsesProvider(provider) {
				if _, ok2 := settings.API.APIKeys[string(types.ApiTypeOpenResponses)+"-api-key"]; ok2 {
					return nil
				}
				if _, ok2 := settings.API.APIKeys[string(types.ApiTypeOpenAIResponses)+"-api-key"]; ok2 {
					return nil
				}
				if _, ok2 := settings.API.APIKeys[string(types.ApiTypeOpenAI)+"-api-key"]; !ok2 {
					return errors.Errorf("missing API key %s (or fallback open-responses-api-key, openai-responses-api-key, openai-api-key)", apiKeyName)
				}
			} else {
				return errors.Errorf("missing API key %s", apiKeyName)
			}
		}
	}

	// Base URL is optional for OpenAI (uses default), but required for others
	baseURLName := provider + "-base-url"
	// Base URL optional for openai and responses providers; required for other OpenAI-compatible providers
	if provider != string(types.ApiTypeOpenAI) && !isResponsesProvider(provider) {
		if _, ok := settings.API.BaseUrls[baseURLName]; !ok {
			return errors.Errorf("missing base URL %s for provider %s", baseURLName, provider)
		}
	}

	return nil
}

// validateClaudeSettings validates settings required for Claude/Anthropic provider.
func (f *StandardEngineFactory) validateClaudeSettings(settings *settings.InferenceSettings, provider string) error {
	// Claude uses "claude" as the key regardless of "anthropic" alias
	actualProvider := string(types.ApiTypeClaude)

	// A request-time source is authoritative over static API-key settings.
	apiKeyName := actualProvider + "-api-key"
	if f.bearerTokenSource == nil {
		if _, ok := settings.API.APIKeys[apiKeyName]; !ok {
			return errors.Errorf("missing API key %s", apiKeyName)
		}
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
func (f *StandardEngineFactory) validateGeminiSettings(settings *settings.InferenceSettings, provider string) error {
	// Check for API key
	apiKeyName := provider + "-api-key"
	if _, ok := settings.API.APIKeys[apiKeyName]; !ok {
		return errors.Errorf("missing API key %s", apiKeyName)
	}

	// Base URL optional for Gemini official client; use default endpoint when absent

	// Check for Gemini-specific settings
	if settings.Gemini == nil {
		return errors.New("Gemini-specific settings cannot be nil")
	}

	return nil
}

// Compile-time check that StandardEngineFactory implements EngineFactory
var _ EngineFactory = (*StandardEngineFactory)(nil)
