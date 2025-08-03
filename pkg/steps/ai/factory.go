package ai

import (
	"github.com/go-go-golems/geppetto/pkg/steps/adapter"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

// Deprecated: StandardStepFactory is deprecated and will be removed in a future version.
// Use inference.StandardEngineFactory instead, which provides a simpler and more 
// powerful way to create AI inference engines.
//
// Migration guide:
// - Replace ai.StandardStepFactory with inference.StandardEngineFactory
// - Replace factory.NewStep() with factory.CreateEngine(settings, options...)
// - Use Engine.RunInference() instead of steps.Resolve() and steps.Bind()
//
// Example migration:
//   Old: factory := &ai.StandardStepFactory{Settings: settings}
//        step, err := factory.NewStep()
//   
//   New: factory := inference.NewStandardEngineFactory()
//        engine, err := factory.CreateEngine(settings)
//        message, err := engine.RunInference(ctx, conversation)
//
// For more information, see the Engine-first architecture documentation.
type StandardStepFactory struct {
	Settings *settings.StepSettings
}

// Deprecated: Use inference.StandardEngineFactory.CreateEngine() instead.
// This method is part of the deprecated Steps API.
//
// Migration:
//   factory := inference.NewStandardEngineFactory()
//   engine, err := factory.CreateEngine(settings, options...)
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
