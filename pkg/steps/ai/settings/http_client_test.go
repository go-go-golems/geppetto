package settings

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestEnsureHTTPClient_UsesExplicitProxyURL(t *testing.T) {
	proxyURL := "http://proxy.internal:8080"
	cs := &ClientSettings{
		ProxyURL:             &proxyURL,
		ProxyFromEnvironment: ptr(false),
	}

	client, err := EnsureHTTPClient(cs)
	if err != nil {
		t.Fatalf("EnsureHTTPClient: %v", err)
	}
	if client == http.DefaultClient {
		t.Fatalf("expected explicit proxy to build a dedicated client")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	req, err := http.NewRequest(http.MethodGet, "https://api.example.com", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	gotProxy, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy: %v", err)
	}
	if gotProxy == nil || gotProxy.String() != proxyURL {
		t.Fatalf("expected proxy %q, got %#v", proxyURL, gotProxy)
	}
	if cs.HTTPClient != client {
		t.Fatalf("expected EnsureHTTPClient to cache the created client")
	}
}

func TestEnsureHTTPClient_DisablesEnvironmentProxyWhenRequested(t *testing.T) {
	cs := &ClientSettings{
		ProxyFromEnvironment: ptr(false),
	}

	client, err := EnsureHTTPClient(cs)
	if err != nil {
		t.Fatalf("EnsureHTTPClient: %v", err)
	}
	if client == http.DefaultClient {
		t.Fatalf("expected direct mode to build a dedicated client")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatalf("expected direct mode to disable transport proxy")
	}
}

func TestEnsureHTTPClient_ReusesDefaultClientForDefaultSettings(t *testing.T) {
	cs := NewClientSettings()

	client, err := EnsureHTTPClient(cs)
	if err != nil {
		t.Fatalf("EnsureHTTPClient: %v", err)
	}
	if client != http.DefaultClient {
		t.Fatalf("expected default settings to reuse http.DefaultClient")
	}
	if cs.HTTPClient != nil {
		t.Fatalf("expected default-client reuse to avoid caching a new client")
	}
}

func TestEnsureHTTPClient_AppliesNonDefaultTimeout(t *testing.T) {
	timeoutSeconds := 123
	cs := &ClientSettings{
		TimeoutSeconds:       &timeoutSeconds,
		ProxyFromEnvironment: ptr(true),
	}

	client, err := EnsureHTTPClient(cs)
	if err != nil {
		t.Fatalf("EnsureHTTPClient: %v", err)
	}
	if client.Timeout != 123*time.Second {
		t.Fatalf("expected timeout 123s, got %s", client.Timeout)
	}
}

func TestEnsureHTTPClient_TimeoutSecondsOverridesSeededTimeout(t *testing.T) {
	cs := NewClientSettings()
	timeoutSeconds := 123
	cs.TimeoutSeconds = &timeoutSeconds

	client, err := EnsureHTTPClient(cs)
	if err != nil {
		t.Fatalf("EnsureHTTPClient: %v", err)
	}
	if client.Timeout != 123*time.Second {
		t.Fatalf("expected timeout 123s from TimeoutSeconds, got %s", client.Timeout)
	}
}

func TestEnsureHTTPClient_RejectsMalformedProxyURL(t *testing.T) {
	proxyURL := "://bad"
	cs := &ClientSettings{
		ProxyURL:             &proxyURL,
		ProxyFromEnvironment: ptr(false),
	}

	_, err := EnsureHTTPClient(cs)
	if err == nil {
		t.Fatalf("expected malformed proxy URL error")
	}
}

func TestRedactedProxyURL_HidesPassword(t *testing.T) {
	got := RedactedProxyURL("http://user:secret@proxy.internal:8080")
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("Parse(redacted): %v", err)
	}
	if u.User == nil {
		t.Fatalf("expected user info to remain present")
	}
	if password, ok := u.User.Password(); !ok || password != "xxxxx" {
		t.Fatalf("expected password to be redacted, got ok=%v password=%q", ok, password)
	}
}

func TestExplainHTTPClientDecision_DefaultSettings(t *testing.T) {
	decision := ExplainHTTPClientDecision(NewClientSettings())
	if decision.Mode != "default-client" {
		t.Fatalf("expected default-client mode, got %q", decision.Mode)
	}
	if decision.ProxyMode != "environment" {
		t.Fatalf("expected environment proxy mode, got %q", decision.ProxyMode)
	}
	if len(decision.Reasons) == 0 || decision.Reasons[0] == "" {
		t.Fatal("expected non-empty reasons")
	}
}

func TestExplainHTTPClientDecision_CustomTimeout(t *testing.T) {
	timeoutSeconds := 123
	decision := ExplainHTTPClientDecision(&ClientSettings{
		TimeoutSeconds:       &timeoutSeconds,
		ProxyFromEnvironment: ptr(true),
	})
	if decision.Mode != "custom-client" {
		t.Fatalf("expected custom-client mode, got %q", decision.Mode)
	}
	if decision.EffectiveTimeout != "2m3s" {
		t.Fatalf("expected effective timeout 2m3s, got %q", decision.EffectiveTimeout)
	}
	if len(decision.Reasons) == 0 || decision.Reasons[0] == "" {
		t.Fatal("expected timeout override reason")
	}
}

func TestExplainHTTPClientDecision_InjectedClient(t *testing.T) {
	decision := ExplainHTTPClientDecision(&ClientSettings{HTTPClient: &http.Client{Timeout: 5 * time.Second}})
	if decision.Mode != "injected-client" {
		t.Fatalf("expected injected-client mode, got %q", decision.Mode)
	}
	if decision.EffectiveTimeout != "5s" {
		t.Fatalf("expected injected timeout 5s, got %q", decision.EffectiveTimeout)
	}
	if len(decision.Reasons) == 0 || decision.Reasons[0] != "client.HTTPClient was injected explicitly" {
		t.Fatalf("unexpected reasons: %#v", decision.Reasons)
	}
}

func ptr[T any](v T) *T { return &v }
