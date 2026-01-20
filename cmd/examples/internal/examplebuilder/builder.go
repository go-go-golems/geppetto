package examplebuilder

import (
	"encoding/json"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/events"
	gebuilder "github.com/go-go-golems/geppetto/pkg/inference/builder"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
)

type simpleEngineConfig struct {
	sig string
}

func (c simpleEngineConfig) Signature() string { return c.sig }

// ParsedLayersEngineBuilder is a minimal EngineBuilder for examples.
//
// It intentionally ignores overrides and mostly treats profileSlug as advisory;
// examples already embed provider selection in ParsedLayers.
type ParsedLayersEngineBuilder struct {
	parsed     *layers.ParsedLayers
	sink       events.EventSink
	engineOpts []engine.Option
}

func NewParsedLayersEngineBuilder(parsed *layers.ParsedLayers, sink events.EventSink, engineOpts ...engine.Option) *ParsedLayersEngineBuilder {
	return &ParsedLayersEngineBuilder{
		parsed:     parsed,
		sink:       sink,
		engineOpts: append([]engine.Option(nil), engineOpts...),
	}
}

func (b *ParsedLayersEngineBuilder) Build(convID, profileSlug string, overrides map[string]any) (engine.Engine, events.EventSink, gebuilder.EngineConfig, error) {
	eng, err := factory.NewEngineFromParsedLayers(b.parsed, b.engineOpts...)
	if err != nil {
		return nil, nil, nil, err
	}
	cfg, err := b.BuildConfig(profileSlug, overrides)
	if err != nil {
		return nil, nil, nil, err
	}
	return eng, b.sink, cfg, nil
}

func (b *ParsedLayersEngineBuilder) BuildConfig(profileSlug string, overrides map[string]any) (gebuilder.EngineConfig, error) {
	var overridesJSON string
	if overrides != nil {
		if b, err := json.Marshal(overrides); err == nil {
			overridesJSON = string(b)
		} else {
			overridesJSON = fmt.Sprintf("<marshal error: %v>", err)
		}
	}
	return simpleEngineConfig{sig: fmt.Sprintf("profile=%s overrides=%s", profileSlug, overridesJSON)}, nil
}

func (b *ParsedLayersEngineBuilder) BuildFromConfig(convID string, config gebuilder.EngineConfig) (engine.Engine, events.EventSink, error) {
	eng, err := factory.NewEngineFromParsedLayers(b.parsed, b.engineOpts...)
	if err != nil {
		return nil, nil, err
	}
	return eng, b.sink, nil
}
