package geminisdk

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

type recordingTransport func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = recordingTransport(nil)

func (f recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestEveryGeminiServiceUsesBoundClient(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		for _, service := range []string{"generate", "cache", "count", "embed", "file"} {
			t.Run(service+map[bool]string{false: "", true: "/redirect"}[redirect], func(t *testing.T) {
				calls := 0
				c, err := connection.New(connection.Config{Provider: "google", BaseURL: "https://gateway.example", Auth: connection.StaticToken("GEMINI-SECRET", connection.TokenHeaders{"x-goog-api-key": ""}), HTTPClient: &http.Client{Transport: recordingTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					require.Equal(t, "gateway.example", req.URL.Host)
					require.Equal(t, "GEMINI-SECRET", req.Header.Get("x-goog-api-key"))
					require.Empty(t, req.URL.Query().Get("key"))
					require.Contains(t, req.URL.Path, "/v1beta/")
					if redirect {
						return &http.Response{StatusCode: 307, Header: http.Header{"Location": {"https://foreign.example/stolen"}}, Body: http.NoBody, Request: req}, nil
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"name":"cachedContents/test","totalTokens":3,"embeddings":[{"values":[1,2]}]}`)), Request: req}, nil
				})}})
				require.NoError(t, err)
				client, err := NewClient(t.Context(), c, "v1beta")
				require.NoError(t, err)
				contents := []*genai.Content{genai.NewContentFromText("test", genai.RoleUser)}
				switch service {
				case "generate":
					_, err = client.Models.GenerateContent(t.Context(), "gemini-test", contents, nil)
				case "cache":
					_, err = client.Caches.Get(t.Context(), "cachedContents/test", nil)
				case "count":
					_, err = client.Models.CountTokens(t.Context(), "gemini-test", contents, nil)
				case "embed":
					_, err = client.Models.EmbedContent(t.Context(), "gemini-test", contents, nil)
				case "file":
					_, err = client.Files.Get(t.Context(), "files/test", nil)
				}
				if redirect {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, 1, calls)
			})
		}
	}
}
