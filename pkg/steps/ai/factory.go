package ai

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/pkg/errors"
)

type StandardStepFactory struct {
	Settings *settings.StepSettings
}

func (s *StandardStepFactory) NewStep(
	options ...chat.StepOption,
) (chat.Step, error) {
	settings_ := s.Settings.Clone()

	if settings_.Chat == nil || settings_.Chat.Engine == nil {
		return nil, errors.New("no chat engine specified")
	}

	var ret chat.Step
	var err error
	if settings_.Chat.ApiType != nil {
		switch *settings_.Chat.ApiType {
		case ai_types.ApiTypeOpenAI, ai_types.ApiTypeAnyScale, ai_types.ApiTypeFireworks:
			ret, err = openai.NewStep(settings_)
			if err != nil {
				return nil, err
			}

		case ai_types.ApiTypeClaude:
			ret, err = claude.NewChatStep(settings_, []api.Tool{})
			if err != nil {
				return nil, err
			}

		case ai_types.ApiTypeGemini:
			ret, err = gemini.NewChatStep(settings_)
			if err != nil {
				return nil, err
			}

		case ai_types.ApiTypeOllama:
			return nil, errors.New("ollama is not supported")

		case ai_types.ApiTypeMistral:
			return nil, errors.New("mistral is not supported")

		case ai_types.ApiTypePerplexity:
			return nil, errors.New("perplexity is not supported")

		case ai_types.ApiTypeCohere:
			return nil, errors.New("cohere is not supported")
		}

	} else {
		switch {
		case openai.IsOpenAiEngine(*settings_.Chat.Engine):
			apiType := ai_types.ApiTypeOpenAI
			settings_.Chat.ApiType = &apiType
			ret, err = openai.NewStep(settings_)
			if err != nil {
				return nil, err
			}

		case claude.IsClaudeEngine(*settings_.Chat.Engine):
			apiType := ai_types.ApiTypeClaude
			settings_.Chat.ApiType = &apiType
			ret = claude.NewStep(settings_)

		case gemini.IsGeminiEngine(*settings_.Chat.Engine):
			apiType := ai_types.ApiTypeGemini
			settings_.Chat.ApiType = &apiType
			ret, err = gemini.NewChatStep(settings_)
			if err != nil {
				return nil, err
			}

		default:
		}
	}

	// Apply step options
	for _, option := range options {
		err := option(ret)
		if err != nil {
			return nil, err
		}
	}

	// Wrap with caching if configured
	if ret != nil && settings_.Chat != nil {
		ret, err = settings_.Chat.WrapWithCache(ret, options...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to wrap step with cache")
		}

	}

	return ret, nil
}

func IsAnyScaleEngine(s string) bool {
	return true
}
