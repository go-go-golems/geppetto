package embeddings

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

func TestNewSettingsFactoryFromInferenceSettingsCopiesEmbeddingLocalAPIMaps(t *testing.T) {
	in := &settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "top-level-key"},
			BaseUrls: map[string]string{"ollama-base-url": "http://top-level.example:11434"},
		},
		Embeddings: &config.EmbeddingsConfig{
			Type:       "openai",
			Engine:     "text-embedding-3-small",
			Dimensions: 1536,
			APIKeys:    map[string]string{"openai-api-key": "embedding-key"},
			BaseURLs:   map[string]string{"ollama-base-url": "http://embedding.example:11434"},
		},
	}

	factory := NewSettingsFactoryFromInferenceSettings(in)
	if factory.config.APIKeys["openai-api-key"] != "embedding-key" {
		t.Fatalf("openai api key = %q, want embedding-local key", factory.config.APIKeys["openai-api-key"])
	}
	if factory.config.BaseURLs["ollama-base-url"] != "http://embedding.example:11434" {
		t.Fatalf("ollama base url = %q, want embedding-local URL", factory.config.BaseURLs["ollama-base-url"])
	}
}

func TestNewSettingsFactoryFromInferenceSettingsOpenAIUsesEmbeddingLocalAPIKey(t *testing.T) {
	in := &settings.InferenceSettings{
		Embeddings: &config.EmbeddingsConfig{
			Type:       "openai",
			Engine:     "text-embedding-3-small",
			Dimensions: 1536,
			APIKeys:    map[string]string{"openai-api-key": "embedding-key"},
		},
	}

	provider, err := NewSettingsFactoryFromInferenceSettings(in).NewProvider()
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	if _, ok := provider.(*OpenAIProvider); !ok {
		t.Fatalf("provider = %T, want *OpenAIProvider", provider)
	}
}

func TestNewSettingsFactoryFromInferenceSettingsOllamaUsesEmbeddingLocalBaseURL(t *testing.T) {
	in := &settings.InferenceSettings{
		Embeddings: &config.EmbeddingsConfig{
			Type:       "ollama",
			Engine:     "nomic-embed-text",
			Dimensions: 768,
			BaseURLs:   map[string]string{"ollama-base-url": "http://embedding.example:11434"},
		},
	}

	provider, err := NewSettingsFactoryFromInferenceSettings(in).NewProvider()
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	ollamaProvider, ok := provider.(*OllamaProvider)
	if !ok {
		t.Fatalf("provider = %T, want *OllamaProvider", provider)
	}
	if ollamaProvider.baseURL != "http://embedding.example:11434" {
		t.Fatalf("baseURL = %q, want embedding-local URL", ollamaProvider.baseURL)
	}
}
