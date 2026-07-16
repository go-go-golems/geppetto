package transport

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type routeResolverFunc func(RouteRequest) (*url.URL, error)

func (f routeResolverFunc) Resolve(request RouteRequest) (*url.URL, error) {
	return f(request)
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL %q: %v", raw, err)
	}
	return parsed
}

func TestResolveAndValidate_ValidatesResolvedTargetAndReturnsReadOnlyContext(t *testing.T) {
	base := mustURL(t, "https://example.test/v1")
	var validated string
	context, err := ResolveAndValidate(
		"openai-codex",
		"responses",
		base,
		routeResolverFunc(func(request RouteRequest) (*url.URL, error) {
			resolverBase := request.BaseURL()
			resolverBase.Host = "mutated.example.test"
			if base.Host != "example.test" {
				t.Fatalf("resolver mutated caller base URL: %q", base.Host)
			}
			if request.Operation() != "responses" {
				t.Fatalf("operation = %q", request.Operation())
			}
			return mustURL(t, "https://example.test/backend-api/codex/responses"), nil
		}),
		func(target *url.URL) error {
			validated = target.String()
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ResolveAndValidate: %v", err)
	}
	if validated != "https://example.test/backend-api/codex/responses" {
		t.Fatalf("validated URL = %q", validated)
	}
	if context.Provider() != "openai-codex" || context.Operation() != "responses" {
		t.Fatalf("context = provider %q operation %q", context.Provider(), context.Operation())
	}
	copyURL := context.URL()
	copyURL.Host = "changed.example.test"
	if got := context.URL().Host; got != "example.test" {
		t.Fatalf("URL copy mutated request context: %q", got)
	}
}

func TestResolveAndValidate_FailsBeforeContextWhenTargetRejected(t *testing.T) {
	validatorCalled := false
	_, err := ResolveAndValidate(
		"openai-codex",
		"responses",
		mustURL(t, "https://example.test"),
		routeResolverFunc(func(RouteRequest) (*url.URL, error) {
			return mustURL(t, "http://untrusted.test/codex/responses"), nil
		}),
		func(*url.URL) error {
			validatorCalled = true
			return errors.New("target denied")
		},
	)
	if err == nil || !strings.Contains(err.Error(), "validate provider route") {
		t.Fatalf("error = %v, want validation error", err)
	}
	if !validatorCalled {
		t.Fatal("validator was not called")
	}
}

func TestHeaderSet_OnlyAllowsDeclaredHeadersAndRedactsSensitiveValues(t *testing.T) {
	target := make(http.Header)
	headers, err := NewHeaderSet(target,
		HeaderRule{Name: "Authorization", Sensitive: true},
		HeaderRule{Name: "X-Provider-Account", Sensitive: true},
		HeaderRule{Name: "X-Provider-Beta"},
	)
	if err != nil {
		t.Fatalf("NewHeaderSet: %v", err)
	}
	if err := headers.Set("authorization", "Bearer token-value"); err != nil {
		t.Fatalf("set authorization: %v", err)
	}
	if err := headers.Set("X-Provider-Account", "account-value"); err != nil {
		t.Fatalf("set account: %v", err)
	}
	if err := headers.Set("X-Provider-Beta", "responses=v1"); err != nil {
		t.Fatalf("set beta: %v", err)
	}
	if got := target.Get("Authorization"); got != "Bearer token-value" {
		t.Fatalf("authorization = %q", got)
	}

	redacted := headers.RedactedCopy()
	if got := redacted.Get("Authorization"); got != "<redacted>" {
		t.Fatalf("redacted authorization = %q", got)
	}
	if got := redacted.Get("X-Provider-Account"); got != "<redacted>" {
		t.Fatalf("redacted account = %q", got)
	}
	if got := redacted.Get("X-Provider-Beta"); got != "responses=v1" {
		t.Fatalf("beta = %q", got)
	}

	for _, name := range []string{"Host", "Content-Length", "X-Undeclared"} {
		if err := headers.Set(name, "value"); err == nil {
			t.Fatalf("Set(%q) unexpectedly succeeded", name)
		}
	}
	if target.Get("Host") != "" || target.Get("Content-Length") != "" || target.Get("X-Undeclared") != "" {
		t.Fatalf("forbidden header was written: %#v", target)
	}
}

func TestHeaderSet_RejectsConflictsAndDoesNotRevealValue(t *testing.T) {
	headers, err := NewHeaderSet(make(http.Header), HeaderRule{Name: "Authorization", Sensitive: true})
	if err != nil {
		t.Fatalf("NewHeaderSet: %v", err)
	}
	if err := headers.Set("Authorization", "Bearer first"); err != nil {
		t.Fatalf("first Set: %v", err)
	}
	if err := headers.Set("Authorization", "Bearer first"); err != nil {
		t.Fatalf("idempotent Set: %v", err)
	}
	const secret = "Bearer second-secret-value"
	err = headers.Set("Authorization", secret)
	if err == nil || !strings.Contains(err.Error(), "already set") {
		t.Fatalf("conflict error = %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("conflict error leaked header value: %q", err)
	}
	err = headers.Set("Authorization", "bad\r\nheader")
	if err == nil || !strings.Contains(err.Error(), "invalid value") {
		t.Fatalf("invalid value error = %v", err)
	}
}

func TestNewHeaderSet_RejectsUnsafeRules(t *testing.T) {
	for _, name := range []string{"Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "Proxy-Connection"} {
		_, err := NewHeaderSet(make(http.Header), HeaderRule{Name: name})
		if err == nil {
			t.Fatalf("NewHeaderSet(%q) unexpectedly succeeded", name)
		}
	}
	_, err := NewHeaderSet(make(http.Header), HeaderRule{Name: "X-A"}, HeaderRule{Name: "x-a"})
	if err == nil {
		t.Fatalf("duplicate header rule unexpectedly succeeded")
	}
}

type middlewareFunc struct {
	before func(context.Context, RequestContext, HeaderWriter) (Attempt, error)
	after  func(context.Context, RequestContext, Attempt, ResponseMetadata) (ResponseDecision, error)
}

func (m middlewareFunc) BeforeRequest(ctx context.Context, request RequestContext, headers HeaderWriter) (Attempt, error) {
	return m.before(ctx, request, headers)
}

func (m middlewareFunc) AfterResponse(ctx context.Context, request RequestContext, attempt Attempt, response ResponseMetadata) (ResponseDecision, error) {
	return m.after(ctx, request, attempt, response)
}

func TestChain_AppliesRequestForwardAndResponseReverse(t *testing.T) {
	request, err := ResolveAndValidate(
		"openai-codex", "responses", mustURL(t, "https://example.test"),
		routeResolverFunc(func(RouteRequest) (*url.URL, error) { return mustURL(t, "https://example.test/codex/responses"), nil }),
		func(*url.URL) error { return nil },
	)
	if err != nil {
		t.Fatalf("ResolveAndValidate: %v", err)
	}
	var calls []string
	first := middlewareFunc{
		before: func(_ context.Context, _ RequestContext, headers HeaderWriter) (Attempt, error) {
			calls = append(calls, "before-first")
			return "first", headers.Set("X-First", "one")
		},
		after: func(_ context.Context, _ RequestContext, attempt Attempt, _ ResponseMetadata) (ResponseDecision, error) {
			calls = append(calls, "after-first-"+attempt.(string))
			return Continue, nil
		},
	}
	second := middlewareFunc{
		before: func(_ context.Context, _ RequestContext, headers HeaderWriter) (Attempt, error) {
			calls = append(calls, "before-second")
			return "second", headers.Set("X-Second", "two")
		},
		after: func(_ context.Context, _ RequestContext, attempt Attempt, _ ResponseMetadata) (ResponseDecision, error) {
			calls = append(calls, "after-second-"+attempt.(string))
			return Retry, nil
		},
	}
	chain, err := NewChain(first, second)
	if err != nil {
		t.Fatalf("NewChain: %v", err)
	}
	headers, err := NewHeaderSet(make(http.Header), HeaderRule{Name: "X-First"}, HeaderRule{Name: "X-Second"})
	if err != nil {
		t.Fatalf("NewHeaderSet: %v", err)
	}
	attempts, err := chain.BeforeRequest(context.Background(), request, headers)
	if err != nil {
		t.Fatalf("BeforeRequest: %v", err)
	}
	decision, err := chain.AfterResponse(context.Background(), request, attempts, ResponseMetadata{StatusCode: http.StatusUnauthorized})
	if err != nil {
		t.Fatalf("AfterResponse: %v", err)
	}
	if decision != Retry {
		t.Fatalf("decision = %v, want Retry", decision)
	}
	wantCalls := []string{"before-first", "before-second", "after-second-second", "after-first-first"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", calls, wantCalls)
	}
}

func TestChain_RejectsUnknownDecisionAndMismatchedAttemptState(t *testing.T) {
	request, err := ResolveAndValidate(
		"provider", "operation", mustURL(t, "https://example.test"),
		routeResolverFunc(func(RouteRequest) (*url.URL, error) { return mustURL(t, "https://example.test/path"), nil }),
		func(*url.URL) error { return nil },
	)
	if err != nil {
		t.Fatalf("ResolveAndValidate: %v", err)
	}
	middleware := middlewareFunc{
		before: func(context.Context, RequestContext, HeaderWriter) (Attempt, error) { return nil, nil },
		after: func(context.Context, RequestContext, Attempt, ResponseMetadata) (ResponseDecision, error) {
			return ResponseDecision(99), nil
		},
	}
	chain, err := NewChain(middleware)
	if err != nil {
		t.Fatalf("NewChain: %v", err)
	}
	if _, err := chain.AfterResponse(context.Background(), request, AttemptState{}, ResponseMetadata{}); err == nil {
		t.Fatal("mismatched attempt state unexpectedly succeeded")
	}
	headers, err := NewHeaderSet(make(http.Header))
	if err != nil {
		t.Fatalf("NewHeaderSet: %v", err)
	}
	attempts, err := chain.BeforeRequest(context.Background(), request, headers)
	if err != nil {
		t.Fatalf("BeforeRequest: %v", err)
	}
	if _, err := chain.AfterResponse(context.Background(), request, attempts, ResponseMetadata{}); err == nil {
		t.Fatal("unknown decision unexpectedly succeeded")
	}
}
