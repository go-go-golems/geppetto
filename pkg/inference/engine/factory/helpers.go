package factory

import (
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
)

// NewEngineFromStepSettings creates an engine directly from step settings.
// This is a convenience function that creates a StandardEngineFactory and uses it to create an engine.
func NewEngineFromStepSettings(stepSettings *settings.StepSettings, options ...engine.Option) (engine.Engine, error) {
	factory := NewStandardEngineFactory()
	return factory.CreateEngine(stepSettings, options...)
}

// NewEngineFromParsedLayers creates an engine from parsed layers.
// This is a convenience function that:
// 1. Creates new step settings
// 2. Updates them from the parsed layers
// 3. Creates and returns an engine
func NewEngineFromParsedLayers(parsedLayers *layers.ParsedLayers, options ...engine.Option) (engine.Engine, error) {
	// Create step settings
	stepSettings, err := settings.NewStepSettings()
	if err != nil {
		return nil, err
	}

	// Update step settings from parsed layers
	err = stepSettings.UpdateFromParsedLayers(parsedLayers)
	if err != nil {
		return nil, err
	}

	// Create engine using step settings
	return NewEngineFromStepSettings(stepSettings, options...)
}
