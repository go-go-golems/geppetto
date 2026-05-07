package bootstrap

import (
	"github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

const InferenceObservabilitySectionSlug = "observability-settings"

type InferenceObservabilitySettings struct {
	TraceLevel         string `glazed:"geppetto-trace-level"`
	MaxRecords         int    `glazed:"geppetto-trace-max-records"`
	MaxPayloadBytes    int    `glazed:"geppetto-trace-max-payload-bytes"`
	RedactProviderData bool   `glazed:"geppetto-trace-redact-provider-data"`
}

func (s InferenceObservabilitySettings) Config() (observability.Config, error) {
	level, err := observability.ParseTraceLevel(s.TraceLevel)
	if err != nil {
		return observability.Config{}, err
	}
	cfg := observability.DefaultConfig()
	cfg.Level = level
	if s.MaxPayloadBytes > 0 {
		cfg.MaxPayloadBytes = s.MaxPayloadBytes
	}
	cfg.RedactProviderData = s.RedactProviderData
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
			fields.New(
				"geppetto-trace-max-payload-bytes",
				fields.TypeInteger,
				fields.WithDefault(observability.DefaultMaxPayloadBytes),
				fields.WithHelp("Maximum string bytes retained inside provider object_json, event_json, and metadata_json payload fields"),
			),
			fields.New(
				"geppetto-trace-redact-provider-data",
				fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Redact sensitive provider fields in object_json/event_json/metadata_json"),
			),
		),
	)
}
