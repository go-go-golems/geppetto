package examplebuilder

import (
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
)

// ParsedLayersEngineBuilder is a minimal EngineBuilder for examples.
//
// It intentionally ignores overrides and mostly treats profileSlug as advisory;
// examples already embed provider selection in ParsedLayers.
type ParsedLayersEngineBuilder struct {
	parsed *layers.ParsedLayers
	sink   events.EventSink
}

func NewParsedLayersEngineBuilder(parsed *layers.ParsedLayers, sink events.EventSink) *ParsedLayersEngineBuilder {
	return &ParsedLayersEngineBuilder{
		parsed: parsed,
		sink:   sink,
	}
}

func (b *ParsedLayersEngineBuilder) Build(convID, profileSlug string, overrides map[string]any) (engine.Engine, events.EventSink, error) {
	eng, err := factory.NewEngineFromParsedLayers(b.parsed)
	if err != nil {
		return nil, nil, err
	}
	return eng, b.sink, nil
}
