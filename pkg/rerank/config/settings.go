// Package config holds the Glazed/YAML configuration for the rerank provider
// primitive, mirroring the pattern in pkg/embeddings/config.
package config

import (
	_ "embed"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/huandu/go-clone"
)

// RerankConfig contains the semantic configuration for a rerank provider.
//
// Endpoint and credential values are not stored here; they use the existing
// InferenceSettings.API maps (BaseUrls, APIKeys, AllowHTTP,
// AllowLocalNetworks) and InferenceSettings.Client (timeout, proxy), exactly
// like embeddings. This keeps profile composition consistent across all
// model-service primitives.
type RerankConfig struct {
	// Type specifies the provider type (e.g. "llamacpp"). Currently only
	// "llamacpp" is supported.
	Type string `yaml:"type,omitempty" glazed:"rerank-type"`
	// Engine specifies the model to use for reranking.
	Engine string `yaml:"engine,omitempty" glazed:"rerank-engine"`
	// MaxRequestBytes bounds the encoded request body. Defaults to 2 MiB.
	MaxRequestBytes int64 `yaml:"max_request_bytes,omitempty" glazed:"rerank-max-request-bytes"`
	// MaxResponseBytes bounds the response body. Defaults to 1 MiB.
	MaxResponseBytes int64 `yaml:"max_response_bytes,omitempty" glazed:"rerank-max-response-bytes"`
}

func NewRerankConfig() (*RerankConfig, error) {
	s := &RerankConfig{}

	p, err := NewRerankValueSection()
	if err != nil {
		return nil, err
	}
	err = p.InitializeStructFromFieldDefaults(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c *RerankConfig) Clone() *RerankConfig {
	return clone.Clone(c).(*RerankConfig)
}

//go:embed "flags/rerank.yaml"
var rerankFlagsYAML []byte

type RerankValueSection struct {
	*schema.SectionImpl `yaml:",inline"`
}

// RerankSlug is the YAML/Glazed section slug for rerank settings.
const RerankSlug = "rerank"

func NewRerankValueSection(options ...schema.SectionOption) (*RerankValueSection, error) {
	ret, err := schema.NewSectionFromYAML(rerankFlagsYAML, options...)
	if err != nil {
		return nil, err
	}
	return &RerankValueSection{SectionImpl: ret}, nil
}
