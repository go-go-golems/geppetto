package embeddings

import (
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// EmbeddingsConfig contains the minimal configuration needed for embeddings
type EmbeddingsConfig struct {
	// Type specifies the provider type (e.g. "openai", "ollama")
	Type string
	// Engine specifies the model to use (e.g. "text-embedding-ada-002" for OpenAI)
	Engine string
	// Dimensions specifies the embedding dimensions (defaults to 1536 for OpenAI)
	Dimensions int `glazed.parameter:"dimensions"`
	// APIKeys maps provider types to their API keys
	APIKeys map[string]string
	// BaseURLs maps provider types to their base URLs
	BaseURLs map[string]string
}

// SettingsFactory creates embedding providers based on configuration
type SettingsFactory struct {
	config *EmbeddingsConfig
}

// NewSettingsFactory creates a new factory that uses the provided configuration
func NewSettingsFactory(config *EmbeddingsConfig) *SettingsFactory {
	return &SettingsFactory{
		config: config,
	}
}

// NewProvider creates a new embedding provider based on the configuration
func (f *SettingsFactory) NewProvider() (Provider, error) {
	if f.config == nil {
		return nil, fmt.Errorf("no configuration provided")
	}

	if f.config.Type == "" {
		return nil, fmt.Errorf("no embeddings type specified in configuration")
	}

	if f.config.Engine == "" {
		return nil, fmt.Errorf("no embeddings model specified in configuration")
	}

	dimensions := 0
	if f.config.Dimensions != 0 {
		dimensions = f.config.Dimensions
	} else {
		if f.config.Type == "openai" {
			dimensions = 1536 // Default for OpenAI
		} else {
			return nil, fmt.Errorf("no dimensions specified for embeddings")
		}
	}

	switch f.config.Type {
	case "ollama":
		baseURL := "http://localhost:11434"
		if f.config.BaseURLs != nil {
			if url, ok := f.config.BaseURLs["ollama-base-url"]; ok {
				baseURL = url
			}
		}
		return NewOllamaProvider(baseURL, f.config.Engine, dimensions), nil

	case "openai":
		apiKey := ""
		if f.config.APIKeys != nil {
			if key, ok := f.config.APIKeys["openai-api-key"]; ok {
				apiKey = key
			}
		}
		if apiKey == "" {
			return nil, fmt.Errorf("no API key provided for OpenAI")
		}

		return NewOpenAIProvider(apiKey, openai.EmbeddingModel(f.config.Engine), dimensions), nil

	default:
		return nil, fmt.Errorf("unsupported provider type for embeddings: %s", f.config.Type)
	}
}

// NewCachedProvider creates a new cached embedding provider based on the configuration
// maxSize determines how many embeddings to keep in cache (default 1000)
func (f *SettingsFactory) NewCachedProvider(maxSize int) (Provider, error) {
	provider, err := f.NewProvider()
	if err != nil {
		return nil, err
	}
	return NewCachedProvider(provider, maxSize), nil
}

// NewSettingsFactoryFromStepSettings creates a new factory from StepSettings for backwards compatibility
func NewSettingsFactoryFromStepSettings(stepSettings interface{ CreateEmbeddingsConfig() *EmbeddingsConfig }) *SettingsFactory {
	config := stepSettings.CreateEmbeddingsConfig()
	return NewSettingsFactory(config)
}
