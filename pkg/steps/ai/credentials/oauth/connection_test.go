package oauth_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials/oauth"
	"github.com/stretchr/testify/require"
	oauth2 "golang.org/x/oauth2"
)

type grantTransport func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = grantTransport(nil)

func (f grantTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestTokenGrantsUseBoundConnection(t *testing.T) {
	for _, grant := range []string{"exchange", "refresh"} {
		for _, redirect := range []bool{false, true} {
			t.Run(grant+map[bool]string{false: "", true: "/redirect"}[redirect], func(t *testing.T) {
				calls := 0
				client := &http.Client{Transport: grantTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					require.Equal(t, "issuer.example", req.URL.Host)
					require.Equal(t, "/token", req.URL.Path)
					require.NoError(t, req.ParseForm())
					user, password, ok := req.BasicAuth()
					// OAuth2 may probe body auth after a failed header-auth attempt.
					if !ok {
						user, password = req.Form.Get("client_id"), req.Form.Get("client_secret")
					}
					require.Equal(t, "client-id", user)
					require.Equal(t, "CLIENT-SECRET", password)
					if grant == "refresh" {
						require.Equal(t, "REFRESH-SECRET", req.Form.Get("refresh_token"))
					} else {
						require.Equal(t, "CODE-SECRET", req.Form.Get("code"))
					}
					if redirect {
						return &http.Response{StatusCode: 307, Header: http.Header{"Location": {"https://foreign.example/stolen"}}, Body: http.NoBody, Request: req}, nil
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"access_token":"ACCESS","refresh_token":"ROTATED","token_type":"Bearer","expires_in":3600}`)), Request: req}, nil
				})}
				c, err := oauth.NewClient(oauth.Config{AuthorizationURL: "https://issuer.example/authorize", TokenURL: "https://issuer.example/token", ClientID: "client-id", ClientSecret: "CLIENT-SECRET", RedirectURL: "http://localhost/callback"})
				require.NoError(t, err)
				ctx := context.WithValue(t.Context(), oauth2.HTTPClient, client)
				var credential credentials.Credential
				if grant == "exchange" {
					credential, err = c.ExchangeAuthorizationCode(ctx, "CODE-SECRET", oauth.NewPKCE())
				} else {
					credential, err = c.Refresh(ctx, credentials.Credential{RefreshToken: "REFRESH-SECRET"})
				}
				if redirect {
					require.Error(t, err)
					require.NotContains(t, err.Error(), "SECRET")
				} else {
					require.NoError(t, err)
					require.Equal(t, "ACCESS", credential.AccessToken)
				}
				wantCalls := 1
				if grant == "exchange" && redirect {
					wantCalls = 2
				}
				require.Equal(t, wantCalls, calls)
			})
		}
	}
}
