package factory

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/tokencount"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	openai_responses "github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/pkg/errors"
)

func NewFromStepSettings(ss *settings.StepSettings) (tokencount.Counter, error) {
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
	case string(types.ApiTypeOpenAIResponses):
		return openai_responses.NewTokenCounter(ss), nil
	case string(types.ApiTypeClaude), "anthropic":
		return claude.NewTokenCounter(ss), nil
	default:
		return nil, errors.Errorf("token count: provider %q is not supported", provider)
	}
}
