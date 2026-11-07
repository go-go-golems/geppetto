package factory

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connectionfactory"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
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
	connection             *connection.Connection
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

	if f.connection != nil && f.bearerTokenSource != nil {
		return nil, errors.New("connection already owns authentication")
	}
	// Normalize the protocol once without mutating caller settings. Authentication
	// remains attached to the connection, independent of protocol aliases.
	normalized := *settings
	normalized.Chat = settings.Chat.Clone()
	normalizedType := types.ApiType(provider)
	if provider == "anthropic" {
		normalizedType = types.ApiTypeClaude
	}
	normalized.Chat.ApiType = &normalizedType
	settings = &normalized

	c := f.connection
	if c == nil {
		var err error
		c, err = connectionfactory.New(settings, f.bearerTokenSource)
		if err != nil {
			return nil, err
		}
	}
	switch provider {
	case string(types.ApiTypeOpenAI), string(types.ApiTypeAnyScale), string(types.ApiTypeFireworks):
		return openai.NewOpenAIEngine(c, settings, f.openAIOptions...)
	case string(types.ApiTypeOpenResponses):
		return openai_responses.NewEngine(c, settings, f.openAIResponsesOptions...)
	case string(types.ApiTypeClaude), "anthropic":
		return claude.NewClaudeEngine(c, settings, f.claudeOptions...)
	case string(types.ApiTypeGemini):
		return gemini.NewGeminiEngine(c, settings, f.geminiOptions...)
	default:
		return nil, errors.Errorf("unsupported provider %s", provider)
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

// validateSettings checks payload requirements only. The connection constructor
// owns credential, endpoint and HTTP-client validation, including slot selection.
func (f *StandardEngineFactory) validateSettings(s *settings.InferenceSettings, provider string) error {
	if s.Chat == nil {
		return errors.New("chat settings cannot be nil")
	}
	switch provider {
	case string(types.ApiTypeOpenAI), string(types.ApiTypeOpenResponses), string(types.ApiTypeAnyScale), string(types.ApiTypeFireworks):
		return nil
	case string(types.ApiTypeClaude), "anthropic":
		if s.Claude == nil {
			return errors.New("Claude-specific settings cannot be nil")
		}
	case string(types.ApiTypeGemini):
		if s.Gemini == nil {
			return errors.New("Gemini-specific settings cannot be nil")
		}
	default:
		return errors.Errorf("unknown provider %s", provider)
	}
	return nil
}

// WithConnection passes one resolved API authority to the selected protocol.
func WithConnection(c *connection.Connection) StandardEngineFactoryOption {
	return func(f *StandardEngineFactory) { f.connection = c }
}

// Compile-time check that StandardEngineFactory implements EngineFactory
var _ EngineFactory = (*StandardEngineFactory)(nil)
