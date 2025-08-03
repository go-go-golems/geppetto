package ai

import (
	"github.com/go-go-golems/geppetto/pkg/steps/adapter"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

type StandardStepFactory struct {
	Settings *settings.StepSettings
}

func (s *StandardStepFactory) NewStep(
	options ...chat.StepOption,
) (chat.Step, error) {
	// Delegate to the new EngineStepFactory which uses the adapter pattern
	// This preserves all existing functionality while using the new engine architecture
	engineFactory := &adapter.EngineStepFactory{
		Settings: s.Settings,
	}
	return engineFactory.NewStep(options...)
}

func IsAnyScaleEngine(s string) bool {
	return true
}
