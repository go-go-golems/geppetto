package oauth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials/oauth"
)

func TestAuthorizationURLUsesPKCES256AndOfflineAccess(t *testing.T) {
	client := newClient(t, "https://issuer.example.test/authorize", "https://issuer.example.test/token")
	pkce := oauth.NewPKCE()

	raw, err := client.AuthorizationURL("state-value", pkce)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("state") != "state-value" {
		t.Fatalf("state = %q", query.Get("state"))
	}
	if query.Get("code_challenge") != pkce.Challenge || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("PKCE query = %v", query)
	}
	if query.Get("access_type") != "offline" {
		t.Fatalf("access_type = %q, want offline", query.Get("access_type"))
	}
	if query.Get("scope") != "inference profile" {
		t.Fatalf("scope = %q", query.Get("scope"))
	}
}

func TestExchangeAuthorizationCodeNormalizesCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("grant_type") != "authorization_code" || request.Form.Get("code") != "authorization-code" {
			t.Fatalf("unexpected form: %v", request.Form)
		}
		if request.Form.Get("code_verifier") == "" {
			t.Fatal("missing PKCE verifier")
		}
		writeToken(t, writer, map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer server.Close()

	client := newClient(t, server.URL+"/authorize", server.URL+"/token")
	credential, err := client.ExchangeAuthorizationCode(context.Background(), "authorization-code", oauth.NewPKCE())
	if err != nil {
		t.Fatal(err)
	}
	if credential.AccessToken != "access-token" || credential.RefreshToken != "refresh-token" || credential.ExpiresAt.Before(time.Now().Add(59*time.Minute)) {
		t.Fatalf("credential = %#v", credential)
	}
}

func TestRefreshForcesGrantAndPreservesUnrotatedRefreshToken(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("grant_type") != "refresh_token" || request.Form.Get("refresh_token") != "previous-refresh" {
			t.Fatalf("unexpected form: %v", request.Form)
		}
		writeToken(t, writer, map[string]any{
			"access_token": "replacement-access",
			"token_type":   "Bearer",
			"expires_in":   600,
		})
	}))
	defer server.Close()

	client := newClient(t, server.URL+"/authorize", server.URL+"/token")
	credential, err := client.Refresh(context.Background(), credentials.Credential{
		AccessToken:  "rejected-but-not-expired",
		RefreshToken: "previous-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || credential.AccessToken != "replacement-access" || credential.RefreshToken != "previous-refresh" {
		t.Fatalf("calls=%d credential=%#v", calls, credential)
	}
}

func TestRefreshCanRequireRotatedRefreshTokenAndRedactsEndpointFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.FormValue("refresh_token") == "endpoint-secret" {
			writeToken(t, writer, map[string]any{"access_token": "replacement-access", "expires_in": 60})
			return
		}
		http.Error(writer, `{"error_description":"token=response-secret"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	strict, err := oauth.NewClient(oauth.Config{
		AuthorizationURL: server.URL + "/authorize",
		TokenURL:         server.URL + "/token",
		ClientID:         "test-client",
		RedirectURL:      "http://127.0.0.1:12345/callback",
	}, oauth.WithRefreshTokenPolicy(oauth.RequireReplacementRefreshToken))
	if err != nil {
		t.Fatal(err)
	}
	_, err = strict.Refresh(context.Background(), credentials.Credential{RefreshToken: "endpoint-secret"})
	if err == nil || strings.Contains(err.Error(), "replacement-access") || strings.Contains(err.Error(), "endpoint-secret") {
		t.Fatalf("expected redacted missing-refresh error, got %v", err)
	}

	_, err = strict.Refresh(context.Background(), credentials.Credential{RefreshToken: "wrong-refresh"})
	if err == nil || strings.Contains(err.Error(), "response-secret") || strings.Contains(err.Error(), "wrong-refresh") {
		t.Fatalf("expected redacted endpoint error, got %v", err)
	}
}

func TestAuthorizationURLRejectsMismatchedPKCE(t *testing.T) {
	client := newClient(t, "https://issuer.example.test/authorize", "https://issuer.example.test/token")
	_, err := client.AuthorizationURL("state", oauth.PKCE{Verifier: "verifier", Challenge: "not-derived"})
	if err == nil {
		t.Fatal("expected mismatched PKCE error")
	}
}

func newClient(t *testing.T, authorizationURL, tokenURL string) *oauth.Client {
	t.Helper()
	client, err := oauth.NewClient(oauth.Config{
		AuthorizationURL: authorizationURL,
		TokenURL:         tokenURL,
		ClientID:         "test-client",
		RedirectURL:      "http://127.0.0.1:12345/callback",
		Scopes:           []string{"inference", "profile"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestStateGenerationAndValidation(t *testing.T) {
	state, err := oauth.NewState()
	if err != nil || state == "" {
		t.Fatalf("NewState = %q, %v", state, err)
	}
	if err := oauth.ValidateState(state, state); err != nil {
		t.Fatalf("ValidateState = %v", err)
	}
	if err := oauth.ValidateState(state, "other"); err == nil || strings.Contains(err.Error(), state) {
		t.Fatalf("invalid state error = %v", err)
	}
}

func writeToken(t *testing.T, writer http.ResponseWriter, payload map[string]any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		t.Fatal(err)
	}
}
