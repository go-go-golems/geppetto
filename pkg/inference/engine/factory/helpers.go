package factory

import (
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

// NewEngineFromSettings creates an engine directly from inference settings.
// This is a convenience function that creates a StandardEngineFactory and uses it to create an engine.
func NewEngineFromSettings(stepSettings *settings.InferenceSettings) (engine.Engine, error) {
	factory := NewStandardEngineFactory()
	return factory.CreateEngine(stepSettings)
}

// NewEngineFromParsedValues creates an engine from parsed values.
// This is a convenience function that:
// 1. Creates new inference settings
// 2. Updates them from parsed values
// 3. Creates and returns an engine
func NewEngineFromParsedValues(parsedValues *values.Values) (engine.Engine, error) {
	// Create inference settings
	stepSettings, err := settings.NewInferenceSettings()
	if err != nil {
		return nil, err
	}

	// Update inference settings from parsed values
	err = stepSettings.UpdateFromParsedValues(parsedValues)
	if err != nil {
		return nil, err
	}

	// Create engine using inference settings
	return NewEngineFromSettings(stepSettings)
}
