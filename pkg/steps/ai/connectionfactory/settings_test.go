package connectionfactory

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/stretchr/testify/require"
)

type recordingTransport func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = recordingTransport(nil)

func (f recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func sourceSettings(protocol string) *settings.InferenceSettings {
	p := types.ApiType(protocol)
	return &settings.InferenceSettings{Chat: &settings.ChatSettings{ApiType: &p}, API: settings.NewAPISettings()}
}

func TestResponsesRequiresItsOwnCompleteSlot(t *testing.T) {
	for _, tc := range []struct{ name, protocol, keySlot, urlSlot string }{
		{"legacy protocol", "openai-responses", "openai-responses", "openai-responses"},
		{"OpenAI fields", "open-responses", "openai", "openai"},
		{"legacy fields", "open-responses", "openai-responses", "openai-responses"},
		{"foreign endpoint", "open-responses", "open-responses", "openai"},
		{"foreign key", "open-responses", "openai", "open-responses"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := sourceSettings(tc.protocol)
			s.API.APIKeys[tc.keySlot+"-api-key"] = "SECRET"
			s.API.BaseUrls[tc.urlSlot+"-base-url"] = "https://gateway.example/v1"
			c, err := New(s, nil)
			require.Error(t, err)
			require.Nil(t, c)
			require.NotContains(t, err.Error(), "SECRET")
		})
	}
}

func TestResponsesUsesOnlyCanonicalSlotAndPolicy(t *testing.T) {
	s := sourceSettings("open-responses")
	s.API.APIKeys["open-responses-api-key"] = "CANONICAL"
	s.API.APIKeys["openai-api-key"] = "UNRELATED"
	s.API.BaseUrls["open-responses-base-url"] = "https://gateway.example/v1"
	s.API.BaseUrls["openai-base-url"] = "https://other.example/v1"
	calls := 0
	s.Client = &settings.ClientSettings{HTTPClient: &http.Client{Transport: recordingTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "https://gateway.example/v1/responses", r.URL.String())
		require.Equal(t, "Bearer CANONICAL", r.Header.Get("Authorization"))
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}")), Request: r}, nil
	})}}
	c, err := New(s, nil)
	require.NoError(t, err)
	req, err := c.NewRequest(t.Context(), "POST", "responses", strings.NewReader("{}"))
	require.NoError(t, err)
	resp, err := c.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, 1, calls)
	// An unrelated slot's permissive policy cannot authorize canonical HTTP.
	s.API.BaseUrls["open-responses-base-url"] = "http://127.0.0.1/v1"
	s.API.AllowHTTP["openai"] = true
	s.API.AllowLocalNetworks["openai"] = true
	_, err = New(s, nil)
	require.Error(t, err)
	s.API.AllowHTTP["open-responses"] = true
	s.API.AllowLocalNetworks["open-responses"] = true
	_, err = New(s, nil)
	require.NoError(t, err)
}

type tokenSource struct{ calls *int }

var _ credentials.BearerTokenSource = tokenSource{}

func (s tokenSource) BearerToken(context.Context, credentials.Request) (string, error) {
	*s.calls++
	return "DYNAMIC", nil
}

func TestSourceIsNotAccessedAtConstructionAndUsesClaudeOAuthHeaders(t *testing.T) {
	s := sourceSettings("claude")
	s.API.BaseUrls["claude-base-url"] = "https://gateway.example"
	lookups := 0
	s.Client = &settings.ClientSettings{HTTPClient: &http.Client{Transport: recordingTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer DYNAMIC", r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("x-api-key"))
		require.Contains(t, r.Header.Get("anthropic-beta"), "oauth-2025-04-20")
		require.Equal(t, "cli", r.Header.Get("x-app"))
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: http.NoBody, Request: r}, nil
	})}}
	c, err := New(s, tokenSource{calls: &lookups})
	require.NoError(t, err)
	require.Zero(t, lookups)
	req, err := c.NewRequest(t.Context(), "POST", "messages", nil)
	require.NoError(t, err)
	resp, err := c.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, 1, lookups)
}
