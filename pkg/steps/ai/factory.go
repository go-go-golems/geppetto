package ai

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
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
		case settings.ApiTypeOpenAI, settings.ApiTypeAnyScale, settings.ApiTypeFireworks:
			ret, err = openai.NewStep(settings_)
			if err != nil {
				return nil, err
			}

		case settings.ApiTypeClaude:
			ret, err = claude.NewChatStep(settings_, []api.Tool{})

		case settings.ApiTypeOllama:
			return nil, errors.New("ollama is not supported")

		case settings.ApiTypeMistral:
			return nil, errors.New("mistral is not supported")

		case settings.ApiTypePerplexity:
			return nil, errors.New("perplexity is not supported")

		case settings.ApiTypeCohere:
			return nil, errors.New("cohere is not supported")
		}

	} else {

		switch {
		case openai.IsOpenAiEngine(*settings_.Chat.Engine):
			apiType := settings.ApiTypeOpenAI
			settings_.Chat.ApiType = &apiType
			ret, err = openai.NewStep(settings_)
			if err != nil {
				return nil, err
			}

		case claude.IsClaudeEngine(*settings_.Chat.Engine):
			apiType := settings.ApiTypeClaude
			settings_.Chat.ApiType = &apiType
			ret = claude.NewStep(settings_)

		default:
		}
	}

	for _, option := range options {
		err := option(ret)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func IsAnyScaleEngine(s string) bool {
	return true
}
