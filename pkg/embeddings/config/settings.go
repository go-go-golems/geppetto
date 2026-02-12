package config

import (
	_ "embed"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/huandu/go-clone"
)

// EmbeddingsConfig contains the minimal configuration needed for embeddings
type EmbeddingsConfig struct {
	// Type specifies the provider type (e.g. "openai", "ollama")
	Type string `glazed:"embeddings-type"`
	// Engine specifies the model to use (e.g. "text-embedding-ada-002" for OpenAI)
	Engine string `glazed:"embeddings-engine"`
	// Dimensions specifies the embedding dimensions (defaults to 1536 for OpenAI)
	Dimensions int `glazed:"embeddings-dimensions"`
	// APIKeys maps provider types to their API keys
	APIKeys map[string]string `yaml:"api_keys,omitempty" glazed:"*-api-key"`
	// BaseURLs maps provider types to their base URLs
	BaseURLs map[string]string `yaml:"base_urls,omitempty" glazed:"*-base-url"`

	// Caching settings
	CacheType       string `glazed:"embeddings-cache-type"`
	CacheMaxSize    int64  `glazed:"embeddings-cache-max-size"`
	CacheMaxEntries int    `glazed:"embeddings-cache-max-entries"`
	CacheDirectory  string `glazed:"embeddings-cache-directory"`
}

func NewEmbeddingsConfig() (*EmbeddingsConfig, error) {
	s := &EmbeddingsConfig{
		APIKeys:  make(map[string]string),
		BaseURLs: make(map[string]string),
	}

	p, err := NewEmbeddingsValueSection()
	if err != nil {
		return nil, err
	}
	err = p.InitializeStructFromFieldDefaults(s)
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

type EmbeddingsValueSection struct {
	*schema.SectionImpl `yaml:",inline"`
}

const EmbeddingsSlug = "embeddings"
const EmbeddingsApiKeySlug = "embeddings-api-key"

func NewEmbeddingsValueSection(options ...schema.SectionOption) (*EmbeddingsValueSection, error) {
	ret, err := schema.NewSectionFromYAML(embeddingsFlagsYAML, options...)
	if err != nil {
		return nil, err
	}
	return &EmbeddingsValueSection{SectionImpl: ret}, nil
}

func NewEmbeddingsApiKeyValue() (*schema.SectionImpl, error) {
	return schema.NewSection(EmbeddingsApiKeySlug,
		"Embeddings API Key Settings",
		schema.WithFields(
			fields.New(
				"openai-api-key",
				fields.TypeString,
				fields.WithHelp("The API key for the OpenAI embeddings provider"),
			),
		),
	)
}
