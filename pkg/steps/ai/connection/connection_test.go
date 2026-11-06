package connection_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

type roundTrip func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = roundTrip(nil)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func response(r *http.Request, status int) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: r}
}

func config(rt http.RoundTripper) connection.Config {
	return connection.Config{
		Provider: "gateway", BaseURL: "https://gateway.example/v1",
		Routes: connection.Routes{"chat": connection.Relative("chat/completions")},
		Auth:   connection.StaticBearer("SECRET"), HTTPClient: &http.Client{Transport: rt},
	}
}

type tokenSource struct {
	lookups atomic.Int32
	refresh atomic.Int32
}

var _ credentials.UnauthorizedBearerTokenSource = (*tokenSource)(nil)

func (s *tokenSource) BearerToken(_ context.Context, req credentials.Request) (string, error) {
	s.lookups.Add(1)
	if req.Provider != "gateway" || req.BaseURL != "https://gateway.example/v1" {
		return "", errors.New("wrong credential binding")
	}
	return "FIRST", nil
}

func (s *tokenSource) BearerTokenAfterUnauthorized(_ context.Context, req credentials.Request, rejected string) (string, error) {
	s.refresh.Add(1)
	if rejected != "FIRST" || req.BaseURL != "https://gateway.example/v1" {
		return "", errors.New("wrong rejected token or binding")
	}
	return "SECOND", nil
}

type foreignRoute struct{}

func (foreignRoute) Resolve(aitransport.RouteRequest) (*url.URL, error) {
	return url.Parse("https://foreign.example/responses")
}

func TestRefusesDestinationBeforeCredentialLookup(t *testing.T) {
	var calls atomic.Int32
	source := &tokenSource{}
	cfg := config(roundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return response(r, 200), nil
	}))
	cfg.Auth = connection.Bearer(source)
	cfg.Routes["foreign"] = foreignRoute{}
	c, err := connection.New(cfg)
	require.NoError(t, err)
	_, err = c.NewRequest(t.Context(), "POST", "foreign", nil)
	require.ErrorIs(t, err, security.ErrOriginNotAllowed)
	for _, target := range []string{"https://foreign.example/v1", "http://gateway.example/v1", "https://gateway.example:444/v1"} {
		req, err := http.NewRequestWithContext(t.Context(), "POST", target, nil)
		require.NoError(t, err)
		_, err = c.Do(req)
		require.ErrorIs(t, err, security.ErrOriginNotAllowed)
	}
	req, err := c.NewRequest(t.Context(), "POST", "chat", nil)
	require.NoError(t, err)
	req.Host = "foreign.example"
	_, err = c.Do(req)
	require.ErrorContains(t, err, "Host override")
	require.Zero(t, source.lookups.Load())
	require.Zero(t, calls.Load())
}

func TestSnapshotsDeclarationAndKeepsSecretsInsideConnection(t *testing.T) {
	cfg := config(roundTrip(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "gateway.example", r.URL.Host)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer SECRET", r.Header.Get("Authorization"))
		require.Equal(t, "TEAM-SECRET", r.Header.Get("X-Team"))
		return response(r, 200), nil
	}))
	cfg.Headers = http.Header{"X-Team": {"TEAM-SECRET"}}
	c, err := connection.New(cfg)
	require.NoError(t, err)
	cfg.BaseURL = "https://foreign.example"
	cfg.Headers.Set("X-Team", "CHANGED")
	cfg.Routes["chat"] = foreignRoute{}
	cfg.HTTPClient.Transport = roundTrip(func(*http.Request) (*http.Response, error) { return nil, errors.New("mutated client") })
	req, err := c.NewRequest(t.Context(), "POST", "chat", strings.NewReader("payload"))
	require.NoError(t, err)
	require.Empty(t, req.Header.Get("Authorization"))
	require.Empty(t, req.Header.Get("X-Team"))
	resp, err := c.DoWithObserver(req, func(debug *http.Request) {
		require.Equal(t, "<redacted>", debug.Header.Get("Authorization"))
		require.Equal(t, "<redacted>", debug.Header.Get("X-Team"))
	})
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, "<redacted>", resp.Request.Header.Get("Authorization"))
	require.Empty(t, req.Header.Get("Authorization"))
	require.NotContains(t, fmt.Sprintf("%#v %+v", c, cfg.Auth), "SECRET")
}

func TestRedirectScopeNativeAndSDK(t *testing.T) {
	for _, sdk := range []bool{false, true} {
		for _, status := range []int{301, 302, 303, 307, 308} {
			t.Run(fmt.Sprintf("sdk=%t/status=%d", sdk, status), func(t *testing.T) {
				var calls int
				cfg := config(roundTrip(func(r *http.Request) (*http.Response, error) {
					calls++
					require.Equal(t, "gateway.example", r.URL.Host)
					require.Equal(t, "Bearer SECRET", r.Header.Get("Authorization"))
					resp := response(r, status)
					resp.Header.Set("Location", "https://foreign.example/stolen")
					return resp, nil
				}))
				c, err := connection.New(cfg)
				require.NoError(t, err)
				req, err := c.NewRequest(t.Context(), "POST", "chat", strings.NewReader("payload"))
				require.NoError(t, err)
				if sdk {
					_, err = c.SDKClient().Do(req)
				} else {
					_, err = c.Do(req)
				}
				require.ErrorIs(t, err, security.ErrOriginNotAllowed)
				require.Equal(t, 1, calls)
			})
		}
	}
}

func TestSameOriginRedirectAndHostRedirectCallback(t *testing.T) {
	for _, hostile := range []bool{false, true} {
		t.Run(fmt.Sprint(hostile), func(t *testing.T) {
			var calls int
			cfg := config(roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "Bearer SECRET", r.Header.Get("Authorization"))
				if calls == 1 {
					resp := response(r, 307)
					resp.Header.Set("Location", "/v1/final")
					return resp, nil
				}
				require.Equal(t, "/v1/final", r.URL.Path)
				return response(r, 200), nil
			}))
			if hostile {
				cfg.HTTPClient.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
					req.URL.Host = "foreign.example"
					req.Host = "foreign.example"
					return nil
				}
			}
			c, err := connection.New(cfg)
			require.NoError(t, err)
			req, err := c.NewRequest(t.Context(), "POST", "chat", strings.NewReader("x"))
			require.NoError(t, err)
			resp, err := c.SDKClient().Do(req)
			if hostile {
				require.ErrorIs(t, err, security.ErrOriginNotAllowed)
				require.Equal(t, 1, calls)
			} else {
				require.NoError(t, err)
				_ = resp.Body.Close()
				require.Equal(t, 2, calls)
			}
		})
	}
}

func TestUnauthorizedReplayIsBoundedAndPerRequest(t *testing.T) {
	source := &tokenSource{}
	var calls atomic.Int32
	cfg := config(roundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if string(body) != "payload" {
			return nil, errors.New("request body was not replayed")
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer FIRST" && auth != "Bearer SECOND" {
			return nil, errors.New("unexpected token")
		}
		return response(r, 401), nil
	}))
	cfg.Auth = connection.Bearer(source)
	c, err := connection.New(cfg)
	require.NoError(t, err)
	var group errgroup.Group
	for range 12 {
		group.Go(func() error {
			req, err := c.NewRequest(t.Context(), "POST", "chat", strings.NewReader("payload"))
			if err != nil {
				return err
			}
			resp, err := c.Do(req)
			if err == nil {
				_ = resp.Body.Close()
			}
			return err
		})
	}
	require.NoError(t, group.Wait())
	require.EqualValues(t, 24, calls.Load())
	require.EqualValues(t, 12, source.lookups.Load())
	require.EqualValues(t, 12, source.refresh.Load())
}

func TestNonReplayableBodyDoesNotRefresh(t *testing.T) {
	source := &tokenSource{}
	cfg := config(roundTrip(func(r *http.Request) (*http.Response, error) {
		_ = r.Body.Close()
		return response(r, 401), nil
	}))
	cfg.Auth = connection.Bearer(source)
	c, err := connection.New(cfg)
	require.NoError(t, err)
	req, err := c.NewRequest(t.Context(), "POST", "chat", io.NopCloser(strings.NewReader("payload")))
	require.NoError(t, err)
	resp, err := c.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Zero(t, source.refresh.Load())
}

func TestInvalidDeclarationsAndRelativeRoutes(t *testing.T) {
	for _, endpoint := range []string{"", " ", "/", "https://user:SECRET@gateway.example", "https://gateway.example/?key=SECRET", "https://gateway.example/#SECRET", "http://gateway.example", "https://localhost"} {
		cfg := config(nil)
		cfg.BaseURL = endpoint
		_, err := connection.New(cfg)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "SECRET")
	}
	for _, path := range []string{"https://foreign.example", "//foreign.example", "/root", "../other", "%2e%2e/other", "a/../b", "a?key=SECRET", "a#SECRET"} {
		cfg := config(nil)
		cfg.Routes["chat"] = connection.Relative(path)
		c, err := connection.New(cfg)
		require.NoError(t, err)
		_, err = c.NewRequest(t.Context(), "POST", "chat", nil)
		require.Error(t, err, path)
		require.NotContains(t, err.Error(), "SECRET")
	}
	for _, header := range []string{"Host", "Content-Length", "Connection"} {
		cfg := config(nil)
		cfg.Headers = http.Header{header: {"SECRET"}}
		_, err := connection.New(cfg)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "SECRET")
	}
}
