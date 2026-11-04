// Package config holds the Glazed/YAML configuration for the decision
// provider primitive, mirroring pkg/rerank/config.
package config

import (
	_ "embed"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/huandu/go-clone"
)

// DecisionConfig is the semantic configuration of a decision provider.
//
// Endpoints and credentials are not stored here: they live in
// InferenceSettings.API (api_keys["systemone-api-key"],
// base_urls["systemone-base-url"], allow_http["decision"],
// allow_local_networks["decision"]) so profile stacking treats every
// model-service primitive alike. Every YAML tag is omitempty so a profile
// overlay that sets only some fields does not erase the others.
type DecisionConfig struct {
	// Type is the provider type; "systemone" is supported.
	Type string `yaml:"type,omitempty" glazed:"decision-type"`
	// Engine is the decision model, e.g. "djev".
	Engine string `yaml:"engine,omitempty" glazed:"decision-engine"`
	// MaxRequestBytes bounds the encoded request body (default 8 MiB).
	MaxRequestBytes int64 `yaml:"max_request_bytes,omitempty" glazed:"decision-max-request-bytes"`
	// MaxResponseBytes bounds the response body (default 1 MiB).
	MaxResponseBytes int64 `yaml:"max_response_bytes,omitempty" glazed:"decision-max-response-bytes"`
}

// NewDecisionConfig returns a config initialised from the section defaults.
func NewDecisionConfig() (*DecisionConfig, error) {
	s := &DecisionConfig{}
	p, err := NewDecisionValueSection()
	if err != nil {
		return nil, err
	}
	if err := p.InitializeStructFromFieldDefaults(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Clone deep-copies the config.
func (c *DecisionConfig) Clone() *DecisionConfig {
	return clone.Clone(c).(*DecisionConfig)
}

//go:embed "flags/decision.yaml"
var decisionFlagsYAML []byte

// DecisionValueSection is the Glazed section for decision settings.
type DecisionValueSection struct {
	*schema.SectionImpl `yaml:",inline"`
}

// DecisionSlug is the YAML/Glazed section slug for decision settings.
const DecisionSlug = "decision"

// NewDecisionValueSection builds the section from flags/decision.yaml.
func NewDecisionValueSection(options ...schema.SectionOption) (*DecisionValueSection, error) {
	ret, err := schema.NewSectionFromYAML(decisionFlagsYAML, options...)
	if err != nil {
		return nil, err
	}
	return &DecisionValueSection{SectionImpl: ret}, nil
}
