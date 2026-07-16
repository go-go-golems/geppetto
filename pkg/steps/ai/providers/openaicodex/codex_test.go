package openaicodex

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type sourceFunc struct {
	credential  func(context.Context, credentials.Request) (Credential, error)
	replacement func(context.Context, credentials.Request, Credential) (Credential, error)
}

func (s sourceFunc) Credential(ctx context.Context, request credentials.Request) (Credential, error) {
	return s.credential(ctx, request)
}

func (s sourceFunc) CredentialAfterUnauthorized(ctx context.Context, request credentials.Request, rejected Credential) (Credential, error) {
	return s.replacement(ctx, request, rejected)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return parsed
}

func TestRoute_UsesFixedCodexResponsePath(t *testing.T) {
	target, err := (Route{}).Resolve(aitransport.RouteRequest{})
	if err == nil || target != nil {
		t.Fatal("route accepted an empty operation")
	}
	base := mustURL(t, "https://chatgpt.com/backend-api")
	request, err := aitransport.ResolveAndValidate(
		Provider, "responses", base, Route{}, func(*url.URL) error { return nil },
	)
	if err != nil {
		t.Fatalf("ResolveAndValidate: %v", err)
	}
	resolved := request.URL()
	if got, want := resolved.Path, "/backend-api/codex/responses"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestResponsesCore_UsesCodexRouteHeadersAndOneRefreshReplay(t *testing.T) {
	var requests, refreshes int
	source := sourceFunc{
		credential: func(context.Context, credentials.Request) (Credential, error) {
			return Credential{BearerToken: "first-token", AccountID: "first-account"}, nil
		},
		replacement: func(context.Context, credentials.Request, Credential) (Credential, error) {
			refreshes++
			return Credential{BearerToken: "replacement-token", AccountID: "replacement-account"}, nil
		},
	}
	adapter, err := RequestTransport(source, Options{Originator: "test-originator"})
	if err != nil {
		t.Fatalf("RequestTransport: %v", err)
	}
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if got, want := request.URL.Path, "/backend-api/codex/responses"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		if got := request.Header.Get("originator"); got != "test-originator" {
			t.Fatalf("originator = %q", got)
		}
		if got := request.Header.Get("OpenAI-Beta"); got != responsesBeta {
			t.Fatalf("OpenAI-Beta = %q", got)
		}
		status := http.StatusUnauthorized
		body := `{"error":"expired"}`
		if requests == 1 {
			if got := request.Header.Get("Authorization"); got != "Bearer first-token" {
				t.Fatalf("first authorization = %q", got)
			}
			if got := request.Header.Get("chatgpt-account-id"); got != "first-account" {
				t.Fatalf("first account = %q", got)
			}
		} else {
			status = http.StatusOK
			body = "data: [DONE]\\n\\n"
			if got := request.Header.Get("Authorization"); got != "Bearer replacement-token" {
				t.Fatalf("replacement authorization = %q", got)
			}
			if got := request.Header.Get("chatgpt-account-id"); got != "replacement-account" {
				t.Fatalf("replacement account = %q", got)
			}
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})}
	model := "codex-test"
	engine, err := openai_responses.NewEngine(&settings.InferenceSettings{
		API:    &settings.APISettings{BaseUrls: map[string]string{"open-responses-base-url": "https://chatgpt.com/backend-api"}},
		Chat:   &settings.ChatSettings{Engine: &model},
		Client: &settings.ClientSettings{HTTPClient: client},
	}, openai_responses.WithRequestTransport(adapter))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	_, err = engine.RunInference(context.Background(), &turns.Turn{Blocks: []turns.Block{turns.NewUserTextBlock("ping")}})
	if err != nil {
		t.Fatalf("RunInference: %v", err)
	}
	if requests != 2 || refreshes != 1 {
		t.Fatalf("requests=%d refreshes=%d, want 2/1", requests, refreshes)
	}
}

func TestRequestTransport_InjectsOnlyCodexHeadersAndReplaysTypedReplacement(t *testing.T) {
	var refreshes int
	source := sourceFunc{
		credential: func(_ context.Context, request credentials.Request) (Credential, error) {
			if request.Provider != Provider || request.BaseURL != "https://chatgpt.com/backend-api" {
				t.Fatalf("credential request = %#v", request)
			}
			return Credential{BearerToken: "first-token", AccountID: "first-account"}, nil
		},
		replacement: func(_ context.Context, _ credentials.Request, rejected Credential) (Credential, error) {
			refreshes++
			if rejected.BearerToken != "first-token" || rejected.AccountID != "first-account" {
				t.Fatalf("rejected credential = %#v", rejected)
			}
			return Credential{BearerToken: "replacement-token", AccountID: "replacement-account"}, nil
		},
	}
	config, err := RequestTransport(source, Options{Originator: "test-originator", UserAgent: "test-agent"})
	if err != nil {
		t.Fatalf("RequestTransport: %v", err)
	}
	if !config.DisableDefaultBearer {
		t.Fatal("Codex transport did not disable ordinary bearer middleware")
	}
	base := mustURL(t, "https://chatgpt.com/backend-api")
	request, err := aitransport.ResolveAndValidate(config.Provider, "responses", base, config.RouteResolver, func(*url.URL) error { return nil })
	if err != nil {
		t.Fatalf("ResolveAndValidate: %v", err)
	}
	chain, err := aitransport.NewChain(config.Middlewares...)
	if err != nil {
		t.Fatalf("NewChain: %v", err)
	}
	firstHeaders, err := aitransport.NewHeaderSet(make(http.Header), config.HeaderRules...)
	if err != nil {
		t.Fatalf("NewHeaderSet: %v", err)
	}
	firstAttempt, err := chain.BeforeRequest(context.Background(), request, aitransport.AttemptState{}, firstHeaders)
	if err != nil {
		t.Fatalf("BeforeRequest first: %v", err)
	}
	if got := firstHeaders.RedactedCopy().Get("Authorization"); got != "<redacted>" {
		t.Fatalf("redacted authorization = %q", got)
	}
	if got := firstHeaders.RedactedCopy().Get("chatgpt-account-id"); got != "<redacted>" {
		t.Fatalf("redacted account = %q", got)
	}
	if got := firstHeaders.RedactedCopy().Get("originator"); got != "test-originator" {
		t.Fatalf("originator = %q", got)
	}
	decision, err := chain.AfterResponse(context.Background(), request, firstAttempt, aitransport.ResponseMetadata{StatusCode: http.StatusUnauthorized, RetryEligible: true})
	if err != nil || decision != aitransport.Retry {
		t.Fatalf("AfterResponse = %v, %v", decision, err)
	}
	secondHeaders, err := aitransport.NewHeaderSet(make(http.Header), config.HeaderRules...)
	if err != nil {
		t.Fatalf("NewHeaderSet second: %v", err)
	}
	_, err = chain.BeforeRequest(context.Background(), request, firstAttempt, secondHeaders)
	if err != nil {
		t.Fatalf("BeforeRequest replay: %v", err)
	}
	if got := secondHeaders.RedactedCopy().Get("Authorization"); got != "<redacted>" {
		t.Fatalf("replay debug authorization = %q", got)
	}
	if got := secondHeaders.RedactedCopy().Get("chatgpt-account-id"); got != "<redacted>" {
		t.Fatalf("replay debug account = %q", got)
	}
	if refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1", refreshes)
	}
}
