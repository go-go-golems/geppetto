package config

import (
	_ "embed"

	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/huandu/go-clone"
)

// EmbeddingsConfig contains the minimal configuration needed for embeddings
type EmbeddingsConfig struct {
	// Type specifies the provider type (e.g. "openai", "ollama")
	Type string `glazed.parameter:"embeddings-type"`
	// Engine specifies the model to use (e.g. "text-embedding-ada-002" for OpenAI)
	Engine string `glazed.parameter:"embeddings-engine"`
	// Dimensions specifies the embedding dimensions (defaults to 1536 for OpenAI)
	Dimensions int `glazed.parameter:"embeddings-dimensions"`
	// APIKeys maps provider types to their API keys
	APIKeys map[string]string `yaml:"api_keys,omitempty" glazed.parameter:"*-api-key"`
	// BaseURLs maps provider types to their base URLs
	BaseURLs map[string]string `yaml:"base_urls,omitempty" glazed.parameter:"*-base-url"`

	// Caching settings
	CacheType       string `glazed.parameter:"embeddings-cache-type"`
	CacheMaxSize    int64  `glazed.parameter:"embeddings-cache-max-size"`
	CacheMaxEntries int    `glazed.parameter:"embeddings-cache-max-entries"`
	CacheDirectory  string `glazed.parameter:"embeddings-cache-directory"`
}

func NewEmbeddingsConfig() (*EmbeddingsConfig, error) {
	s := &EmbeddingsConfig{
		APIKeys:  make(map[string]string),
		BaseURLs: make(map[string]string),
	}

	p, err := NewEmbeddingsParameterLayer()
	if err != nil {
		return nil, err
	}
	err = p.InitializeStructFromParameterDefaults(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c *EmbeddingsConfig) Clone() *EmbeddingsConfig {
	return clone.Clone(c).(*EmbeddingsConfig)
}

//go:embed "flags/embeddings.yaml"
var embeddingsFlagsYAML []byte

type EmbeddingsParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

const EmbeddingsSlug = "embeddings"

func NewEmbeddingsParameterLayer(options ...layers.ParameterLayerOptions) (*EmbeddingsParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(embeddingsFlagsYAML, options...)
	if err != nil {
		return nil, err
	}
	return &EmbeddingsParameterLayer{ParameterLayerImpl: ret}, nil
}
