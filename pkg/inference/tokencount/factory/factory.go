package factory

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/tokencount"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	openai_responses "github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/pkg/errors"
)

func NewFromSettings(ss *settings.InferenceSettings) (tokencount.Counter, error) {
	return NewFromSettingsWithBearerTokenSource(ss, nil)
}

// NewFromSettingsWithBearerTokenSource creates a token counter with an
// optional Go-only request-time credential source.
func NewFromSettingsWithBearerTokenSource(ss *settings.InferenceSettings, source credentials.BearerTokenSource) (tokencount.Counter, error) {
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
		return openai_responses.NewTokenCounter(ss), nil
	case string(types.ApiTypeClaude), "anthropic":
		if source != nil {
			return claude.NewTokenCounter(ss, claude.WithTokenCountOAuthBearerTokenSource(source)), nil
		}
		return claude.NewTokenCounter(ss), nil
	default:
		return nil, errors.Errorf("token count: provider %q is not supported", provider)
	}
}
