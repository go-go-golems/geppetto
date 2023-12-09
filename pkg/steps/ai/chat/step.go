package chat

import (
	context1 "context"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/pkg/errors"
	"time"
)

type Step interface {
	steps.Step[[]*context.Message, string]
	SetStreaming(bool)
}

type StandardStepFactory struct {
	Settings *settings.StepSettings
}

func (s *StandardStepFactory) NewStepFromLayers(layers map[string]*layers.ParsedParameterLayer) (
	steps.Step[[]*context.Message, string],
	error,
) {
	settings_ := s.Settings.Clone()
	err := settings_.UpdateFromParsedLayers(layers)
	if err != nil {
		return nil, err
	}

	if settings_.Chat == nil || settings_.Chat.Engine == nil {
		return nil, errors.New("no chat engine specified")
	}

	if openai.IsOpenAiEngine(*settings_.Chat.Engine) {
		return &openai.Step{
			Settings: settings_,
		}, nil
	}

	if claude.IsClaudeEngine(*settings_.Chat.Engine) {
		return &claude.Step{
			Settings: settings_,
		}, nil
	}

	if IsAnyScaleEngine(*settings_.Chat.Engine) {
		return &openai.Step{
			Settings: settings_,
		}, nil
	}

	return nil, errors.Errorf("unknown chat engine: %s", *settings_.Chat.Engine)
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
