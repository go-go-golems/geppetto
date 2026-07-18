package embeddings

import (
	"fmt"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"gopkg.in/yaml.v3"
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

func TestNewSettingsFactoryFromInferenceSettingsFileCacheUsesDefaultsForOmittedLimits(t *testing.T) {
	cacheDirectory := t.TempDir()
	profile := fmt.Sprintf(`
inference_settings:
  embeddings:
    type: ollama
    engine: nomic-embed-text
    dimensions: 768
    cache_type: file
    cache_directory: %q
`, cacheDirectory)
	var decoded struct {
		InferenceSettings *settings.InferenceSettings `yaml:"inference_settings"`
	}
	if err := yaml.Unmarshal([]byte(profile), &decoded); err != nil {
		t.Fatalf("unmarshal profile YAML: %v", err)
	}

	provider, err := NewSettingsFactoryFromInferenceSettings(decoded.InferenceSettings).NewProvider()
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	diskCache, ok := provider.(*DiskCacheProvider)
	if !ok {
		t.Fatalf("provider = %T, want *DiskCacheProvider", provider)
	}
	if got, want := diskCache.maxSize, int64(1<<30); got != want {
		t.Errorf("cache max size = %d, want default %d", got, want)
	}
	if got, want := diskCache.maxEntries, 10000; got != want {
		t.Errorf("cache max entries = %d, want default %d", got, want)
	}
}

func TestNewSettingsFactoryFromInferenceSettingsDecodesFileCacheFromYAML(t *testing.T) {
	cacheDirectory := t.TempDir()
	profile := fmt.Sprintf(`
inference_settings:
  embeddings:
    type: ollama
    engine: nomic-embed-text
    dimensions: 768
    cache_type: file
    cache_max_size: 12345
    cache_max_entries: 7
    cache_directory: %q
`, cacheDirectory)
	var decoded struct {
		InferenceSettings *settings.InferenceSettings `yaml:"inference_settings"`
	}
	if err := yaml.Unmarshal([]byte(profile), &decoded); err != nil {
		t.Fatalf("unmarshal profile YAML: %v", err)
	}
	if decoded.InferenceSettings == nil || decoded.InferenceSettings.Embeddings == nil {
		t.Fatal("profile YAML did not decode embedding settings")
	}
	embeddingSettings := decoded.InferenceSettings.Embeddings
	if embeddingSettings.CacheType != "file" || embeddingSettings.CacheMaxSize != 12345 || embeddingSettings.CacheMaxEntries != 7 || embeddingSettings.CacheDirectory != cacheDirectory {
		t.Fatalf("cache settings = %#v, want file cache settings from profile YAML", embeddingSettings)
	}

	provider, err := NewSettingsFactoryFromInferenceSettings(decoded.InferenceSettings).NewProvider()
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	diskCache, ok := provider.(*DiskCacheProvider)
	if !ok {
		t.Fatalf("provider = %T, want *DiskCacheProvider", provider)
	}
	if diskCache.directory != cacheDirectory {
		t.Fatalf("cache directory = %q, want %q", diskCache.directory, cacheDirectory)
	}
}
