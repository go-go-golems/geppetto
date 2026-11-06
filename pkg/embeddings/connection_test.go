package embeddings

import (
	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

type embeddingRoundTrip func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = embeddingRoundTrip(nil)

func (f embeddingRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func testEmbeddingConnection(t *testing.T, provider, key, baseURL string, client *http.Client) *connection.Connection {
	t.Helper()
	if client == nil {
		if strings.HasPrefix(baseURL, "http://127.0.0.1:") {
			client = &http.Client{}
		} else {
			client = &http.Client{Transport: embeddingRoundTrip(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 400, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"test"}}`)), Request: req}, nil
			})}
		}
	}
	routes := connection.Routes{"embed": connection.Relative("embeddings")}
	if provider == "cohere" {
		routes = CohereRoutes()
	}
	if provider == "ollama" {
		routes = OllamaRoutes()
	}
	auth := connection.StaticBearer(key)
	if provider == "ollama" {
		auth = connection.Authentication{}
	}
	c, err := connection.New(connection.Config{Provider: provider, BaseURL: baseURL, Auth: auth, Routes: routes, HTTPClient: client, OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true}})
	require.NoError(t, err)
	return c
}

func TestEmbeddingCredentialPairsAreAtomic(t *testing.T) {
	for _, providerType := range []string{"openai", "cohere"} {
		for _, variant := range []string{"inherit", "local-key-only", "local-url-only", "local-pair", "blank-local", "option-key-only", "option-url-only", "option-pair"} {
			t.Run(providerType+"/"+variant, func(t *testing.T) {
				calls := 0
				wantHost, wantKey := "parent.example", "PARENT"
				client := &http.Client{Transport: embeddingRoundTrip(func(req *http.Request) (*http.Response, error) {
					calls++
					require.Equal(t, wantHost, req.URL.Host)
					require.Equal(t, "Bearer "+wantKey, req.Header.Get("Authorization"))
					return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"data":[{"embedding":[1,2]}],"embeddings":{"float":[[1,2]]}}`)), Request: req}, nil
				})}
				keySlot, urlSlot := providerType+"-api-key", providerType+"-base-url"
				local := &config.EmbeddingsConfig{Type: providerType, Engine: "embedding", Dimensions: 2, APIKeys: map[string]string{}, BaseURLs: map[string]string{}}
				in := &settings.InferenceSettings{API: &settings.APISettings{APIKeys: map[string]string{keySlot: "PARENT"}, BaseUrls: map[string]string{urlSlot: "https://parent.example/v1"}}, Client: &settings.ClientSettings{HTTPClient: client}, Embeddings: local}
				var options []ProviderOption
				switch variant {
				case "local-key-only":
					local.APIKeys[keySlot] = "LOCAL"
				case "local-url-only":
					local.BaseURLs[urlSlot] = "https://local.example/v1"
				case "local-pair":
					local.APIKeys[keySlot] = "LOCAL"
					local.BaseURLs[urlSlot] = "https://local.example/v1"
					wantHost, wantKey = "local.example", "LOCAL"
				case "blank-local":
					local.APIKeys[keySlot] = "  "
					local.BaseURLs[urlSlot] = " "
				case "option-key-only":
					options = []ProviderOption{WithAPIKey("OPTION")}
				case "option-url-only":
					options = []ProviderOption{WithBaseURL("https://option.example/v1")}
				case "option-pair":
					options = []ProviderOption{WithAPIKey("OPTION"), WithBaseURL("https://option.example/v1")}
					wantHost, wantKey = "option.example", "OPTION"
				}
				p, err := NewSettingsFactoryFromInferenceSettings(in).NewProvider(options...)
				if strings.HasSuffix(variant, "-only") {
					require.Error(t, err)
					require.Zero(t, calls)
					return
				}
				require.NoError(t, err)
				// Changing source maps after binding cannot repoint the credential.
				in.API.APIKeys[keySlot] = "MUTATED"
				in.API.BaseUrls[urlSlot] = "https://foreign.example"
				local.APIKeys[keySlot] = "MUTATED"
				local.BaseURLs[urlSlot] = "https://foreign.example"
				values, err := p.GenerateEmbedding(t.Context(), "test")
				require.NoError(t, err)
				require.Equal(t, []float32{1, 2}, values)
				require.Equal(t, 1, calls)
			})
		}
	}
}
