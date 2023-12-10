package ai

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
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
	switch {
	case openai.IsOpenAiEngine(*settings_.Chat.Engine):
		ret = openai.NewStep(settings_)

	case claude.IsClaudeEngine(*settings_.Chat.Engine):
		ret = claude.NewStep(settings_)

	case IsAnyScaleEngine(*settings_.Chat.Engine):
		ret = openai.NewStep(settings_)
	default:
		return nil, errors.Errorf("unknown chat engine: %s", *settings_.Chat.Engine)
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
