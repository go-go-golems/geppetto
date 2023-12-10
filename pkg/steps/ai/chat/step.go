package chat

import (
	context1 "context"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	"time"
)

type Step interface {
	steps.Step[[]*context.Message, string]
	SetStreaming(bool)
	SetOnPartial(func(string) error)
}

type StandardStepFactory struct {
	Settings *settings.StepSettings
}

type StepOption func(Step)

func WithStreaming(b bool) StepOption {
	return func(s Step) {
		s.SetStreaming(b)
	}
}

func WithOnPartial(f func(string) error) StepOption {
	return func(s Step) {
		s.SetOnPartial(f)
	}
}

func (s *StandardStepFactory) NewStep(
	options ...StepOption,
) (Step, error) {
	settings_ := s.Settings.Clone()

	if settings_.Chat == nil || settings_.Chat.Engine == nil {
		return nil, errors.New("no chat engine specified")
	}

	var ret Step
	switch {
	case openai.IsOpenAiEngine(*settings_.Chat.Engine):
		ret = &openai.Step{Settings: settings_}

	case claude.IsClaudeEngine(*settings_.Chat.Engine):
		ret = &claude.Step{Settings: settings_}

	case IsAnyScaleEngine(*settings_.Chat.Engine):
		ret = &openai.Step{Settings: settings_}
	default:
		return nil, errors.Errorf("unknown chat engine: %s", *settings_.Chat.Engine)
	}

	for _, option := range options {
		option(ret)
	}

	return ret, nil
}

func IsAnyScaleEngine(s string) bool {
	return true
}

type AddToHistoryStep struct {
	manager *context.Manager
	role    string
}

var _ steps.Step[string, string] = &AddToHistoryStep{}

func (a *AddToHistoryStep) Start(ctx context1.Context, input string) (steps.StepResult[string], error) {
	a.manager.AddMessages(&context.Message{
		Text: input,
		Time: time.Time{},
		Role: a.role,
	})

	return steps.Resolve(input), nil
}

type RunnableStep struct {
	c       context.GeppettoRunnable
	manager *context.Manager
}

var _ steps.Step[interface{}, string] = &RunnableStep{}

func (r *RunnableStep) Start(ctx context1.Context, input interface{}) (steps.StepResult[string], error) {
	return r.c.RunWithManager(ctx, r.manager)
}
