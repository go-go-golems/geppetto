package factory

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/tokencount"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	openai_responses "github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/pkg/errors"
)

type counterOptions struct {
	connection *connection.Connection
	source     credentials.BearerTokenSource
}
type Option func(*counterOptions)

func WithConnection(c *connection.Connection) Option {
	return func(o *counterOptions) { o.connection = c }
}
func WithBearerTokenSource(source credentials.BearerTokenSource) Option {
	return func(o *counterOptions) { o.source = source }
}

// NewFromSettings selects a counter protocol. A connection owns the complete
// authority; configuration-based callers resolve their binding at construction.
func NewFromSettings(ss *settings.InferenceSettings, opts ...Option) (tokencount.Counter, error) {
	var options counterOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.connection != nil && options.source != nil {
		return nil, errors.New("connection already owns authentication")
	}
	if ss == nil {
		return nil, errors.New("token count: settings cannot be nil")
	}
	if ss.Chat == nil {
		return nil, errors.New("token count: chat settings cannot be nil")
	}

	provider := string(types.ApiTypeOpenAI)
	if ss.Chat.ApiType != nil && strings.TrimSpace(string(*ss.Chat.ApiType)) != "" {
		provider = strings.ToLower(strings.TrimSpace(string(*ss.Chat.ApiType)))
	}

	switch provider {
	case string(types.ApiTypeOpenResponses), string(types.ApiTypeOpenAIResponses):
		var opts []openai_responses.TokenCounterOption
		if options.connection != nil {
			opts = append(opts, openai_responses.WithTokenCountConnection(options.connection))
		}
		if options.source != nil {
			opts = append(opts, openai_responses.WithTokenCountBearerTokenSource(options.source))
		}
		return openai_responses.NewTokenCounter(ss, opts...)
	case string(types.ApiTypeClaude), "anthropic":
		var opts []claude.TokenCounterOption
		if options.connection != nil {
			opts = append(opts, claude.WithTokenCountConnection(options.connection))
		}
		if options.source != nil {
			opts = append(opts, claude.WithTokenCountOAuthBearerTokenSource(options.source))
		}
		return claude.NewTokenCounter(ss, opts...)
	default:
		return nil, errors.Errorf("token count: provider %q is not supported", provider)
	}
}
