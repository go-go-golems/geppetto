package bootstrap

import (
	"github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

const InferenceObservabilitySectionSlug = "observability-settings"

type InferenceObservabilitySettings struct {
	TraceLevel string `glazed:"geppetto-trace-level"`
	MaxRecords int    `glazed:"geppetto-trace-max-records"`
}

func (s InferenceObservabilitySettings) Config() (observability.Config, error) {
	level, err := observability.ParseTraceLevel(s.TraceLevel)
	if err != nil {
		return observability.Config{}, err
	}
	cfg := observability.DefaultConfig()
	cfg.Level = level
	return cfg.Normalized(), nil
}

func NewInferenceObservabilitySection() (schema.Section, error) {
	return schema.NewSection(
		InferenceObservabilitySectionSlug,
		"Inference observability",
		schema.WithFields(
			fields.New(
				"geppetto-trace-level",
				fields.TypeString,
				fields.WithDefault(string(observability.TraceOff)),
				fields.WithHelp("Geppetto trace level for first-slice observability: off, events, provider (raw stream strings are not captured)"),
			),
			fields.New(
				"geppetto-trace-max-records",
				fields.TypeInteger,
				fields.WithDefault(100000),
				fields.WithHelp("Maximum Geppetto observer records retained by app recorders"),
			),
		),
	)
}
