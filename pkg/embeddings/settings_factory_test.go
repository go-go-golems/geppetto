package embeddings

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"io"
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
			BaseURLs:   map[string]string{"ollama-base-url": "http://embedding.example:11434", "openai-base-url": "https://api.openai.com/v1"},
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
			BaseURLs:   map[string]string{"openai-base-url": "https://api.openai.com/v1"},
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
	if ollamaProvider.connection.BaseURL() != "http://embedding.example:11434" {
		t.Fatalf("baseURL = %q, want embedding-local URL", ollamaProvider.connection.BaseURL())
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

func TestNewSettingsFactoryFromInferenceSettingsSkipsBlankLocalCohereValuesAndInjectsClient(t *testing.T) {
	sent := false
	client := &http.Client{Timeout: 7 * time.Second, Transport: embeddingRoundTrip(func(req *http.Request) (*http.Response, error) {
		sent = true
		require.Equal(t, "Bearer top-level-key", req.Header.Get("Authorization"))
		require.Equal(t, "/cohere/v2/embed", req.URL.Path)
		deadline, ok := req.Context().Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(7*time.Second), deadline, time.Second)
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"embeddings":{"float":[[1,2]]}}`)), Request: req}, nil
	})}
	in := &settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:            map[string]string{"cohere-api-key": "top-level-key"},
			BaseUrls:           map[string]string{"cohere-base-url": "https://proxy.example/cohere"},
			AllowHTTP:          map[string]bool{},
			AllowLocalNetworks: map[string]bool{},
		},
		Client: &settings.ClientSettings{HTTPClient: client},
		Embeddings: &config.EmbeddingsConfig{
			Type:     "cohere",
			Engine:   "embed-v4.0",
			APIKeys:  map[string]string{"cohere-api-key": "  "},
			BaseURLs: map[string]string{"cohere-base-url": ""},
		},
	}

	provider, err := NewSettingsFactoryFromInferenceSettings(in).NewProvider()
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	cohereProvider, ok := provider.(*CohereProvider)
	if !ok {
		t.Fatalf("provider = %T, want *CohereProvider", provider)
	}
	if cohereProvider.connection.BaseURL() != "https://proxy.example/cohere" {
		t.Fatalf("base URL = %q, want non-blank top-level fallback", cohereProvider.connection.BaseURL())
	}
	if embeddingEndpoint(t, cohereProvider.connection) != "https://proxy.example/cohere/v2/embed" {
		t.Fatalf("endpoint = %q, want path appended to shared base", embeddingEndpoint(t, cohereProvider.connection))
	}
	values, err := provider.GenerateEmbedding(t.Context(), "test")
	require.NoError(t, err)
	require.Equal(t, []float32{1, 2}, values)
	require.True(t, sent)
	require.Equal(t, 7*time.Second, client.Timeout)

}

func TestNewSettingsFactoryFromInferenceSettingsConstructsCohereProvider(t *testing.T) {
	in := &settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys: map[string]string{"cohere-api-key": "top-level-key"},
		},
		Embeddings: &config.EmbeddingsConfig{
			Type:     "cohere",
			Engine:   "embed-v4.0",
			APIKeys:  map[string]string{"cohere-api-key": "embedding-key"},
			BaseURLs: map[string]string{"cohere-base-url": "https://api.cohere.com"},
		},
	}

	provider, err := NewSettingsFactoryFromInferenceSettings(in).NewProvider()
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	cohereProvider, ok := provider.(*CohereProvider)
	if !ok {
		t.Fatalf("provider = %T, want *CohereProvider", provider)
	}
	if cohereProvider.connection.BaseURL() != "https://api.cohere.com" {
		t.Fatalf("base URL = %q, want hosted API root", cohereProvider.connection.BaseURL())
	}
	if embeddingEndpoint(t, cohereProvider.connection) != "https://api.cohere.com/v2/embed" {
		t.Fatalf("endpoint = %q, want hosted embed endpoint", embeddingEndpoint(t, cohereProvider.connection))
	}
	model := cohereProvider.GetModel()
	if model.Name != "embed-v4.0" {
		t.Fatalf("model = %q, want embed-v4.0", model.Name)
	}
}

func TestNewSettingsFactoryFromInferenceSettingsCohereHonorsBaseURLOverride(t *testing.T) {
	in := &settings.InferenceSettings{
		Embeddings: &config.EmbeddingsConfig{
			Type:     "cohere",
			Engine:   "embed-v4.0",
			APIKeys:  map[string]string{"cohere-api-key": "key"},
			BaseURLs: map[string]string{"cohere-base-url": "https://proxy.example.com/v2/embed"},
		},
	}

	provider, err := NewSettingsFactoryFromInferenceSettings(in).NewProvider()
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	cohereProvider, ok := provider.(*CohereProvider)
	if !ok {
		t.Fatalf("provider = %T, want *CohereProvider", provider)
	}
	if cohereProvider.connection.BaseURL() != "https://proxy.example.com/v2/embed" {
		t.Fatalf("base URL = %q, want override", cohereProvider.connection.BaseURL())
	}
}

func TestNewSettingsFactoryFromInferenceSettingsCohereRequiresAPIKey(t *testing.T) {
	in := &settings.InferenceSettings{
		Embeddings: &config.EmbeddingsConfig{Type: "cohere", Engine: "embed-v4.0"},
	}

	_, err := NewSettingsFactoryFromInferenceSettings(in).NewProvider()
	if err == nil {
		t.Fatal("NewProvider returned nil error, want missing-key error")
	}
	if !strings.Contains(err.Error(), "no API key provided for Cohere") {
		t.Fatalf("error = %q, want Cohere key diagnostic", err.Error())
	}
}

func embeddingEndpoint(t *testing.T, c *connection.Connection) string {
	t.Helper()
	route, err := c.Resolve("embed")
	require.NoError(t, err)
	target := route.URL()
	return target.String()
}
