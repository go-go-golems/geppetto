package umans

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/stretchr/testify/require"
)

type roundTrip func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = roundTrip(nil)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConnectionRequiresKeyAndBindsIssuer(t *testing.T) {
	_, err := NewConnection("", nil)
	require.Error(t, err)
	calls := 0
	c, err := NewConnection("UMANS-SECRET", &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "api.code.umans.ai", req.URL.Host)
		require.Equal(t, "Bearer UMANS-SECRET", req.Header.Get("Authorization"))
		require.Equal(t, "UMANS-SECRET", req.Header.Get("x-api-key"))
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"model":"test"}`)), Request: req}, nil
	})})
	require.NoError(t, err)
	_, err = api.NewClient(c).SendMessage(t.Context(), &api.MessageRequest{Model: "test"})
	require.NoError(t, err)
	foreign, err := http.NewRequestWithContext(t.Context(), "POST", "https://api.anthropic.com/v1/messages", nil)
	require.NoError(t, err)
	_, err = c.Do(foreign)
	require.ErrorIs(t, err, security.ErrOriginNotAllowed)
	require.Equal(t, 1, calls)
}
