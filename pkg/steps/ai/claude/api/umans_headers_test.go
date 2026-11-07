package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/stretchr/testify/require"
)

type recordingRoundTrip func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = recordingRoundTrip(nil)

func (r recordingRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return r(req) }

func TestClientSendsBoundAnthropicAuthentication(t *testing.T) {
	for _, oauth := range []bool{false, true} {
		t.Run(map[bool]string{false: "gateway", true: "oauth"}[oauth], func(t *testing.T) {
			carriers := connection.TokenHeaders{"x-api-key": "", "Authorization": "Bearer "}
			defaults := http.Header{}
			if oauth {
				delete(carriers, "x-api-key")
				defaults.Set("anthropic-beta", "claude-code-20250219,oauth-2025-04-20")
				defaults.Set("x-app", "cli")
			}
			var sent bool
			c, err := connection.New(connection.Config{Provider: "claude", BaseURL: "https://api.anthropic.com", Routes: Routes(), Auth: connection.StaticToken("test-token", carriers), Headers: defaults, HTTPClient: &http.Client{Transport: recordingRoundTrip(func(req *http.Request) (*http.Response, error) {
				sent = true
				require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
				require.Equal(t, defaultAPIVersion, req.Header.Get("anthropic-version"))
				if oauth {
					require.Empty(t, req.Header.Get("x-api-key"))
					require.Equal(t, "claude-code-20250219,oauth-2025-04-20", req.Header.Get("anthropic-beta"))
					require.Equal(t, "cli", req.Header.Get("x-app"))
				} else {
					require.Equal(t, "test-token", req.Header.Get("x-api-key"))
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"model":"test"}`)), Request: req}, nil
			})}})
			require.NoError(t, err)
			_, err = NewClient(c).SendMessage(t.Context(), &MessageRequest{Model: "test"})
			require.NoError(t, err)
			require.True(t, sent)
		})
	}
}
