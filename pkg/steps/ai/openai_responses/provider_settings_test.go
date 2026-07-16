package openai_responses

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type bearerTokenSourceFunc func(context.Context, credentials.Request) (string, error)

func (f bearerTokenSourceFunc) BearerToken(ctx context.Context, request credentials.Request) (string, error) {
	return f(ctx, request)
}

type unauthorizedBearerTokenSourceFunc struct {
	bearer       func(context.Context, credentials.Request) (string, error)
	unauthorized func(context.Context, credentials.Request, string) (string, error)
}

func (f unauthorizedBearerTokenSourceFunc) BearerToken(ctx context.Context, request credentials.Request) (string, error) {
	return f.bearer(ctx, request)
}

func (f unauthorizedBearerTokenSourceFunc) BearerTokenAfterUnauthorized(ctx context.Context, request credentials.Request, rejected string) (string, error) {
	return f.unauthorized(ctx, request, rejected)
}

type fixedResponsesRoute struct {
	target *url.URL
}

func (r fixedResponsesRoute) Resolve(aitransport.RouteRequest) (*url.URL, error) {
	target := *r.target
	return &target, nil
}

type capturingResponsesDebugTap struct {
	header http.Header
}

func (t *capturingResponsesDebugTap) OnHTTP(request *http.Request, _ []byte) {
	t.header = request.Header.Clone()
}
func (*capturingResponsesDebugTap) OnHTTPResponse(*http.Response, []byte) {}
func (*capturingResponsesDebugTap) OnSSE(string, []byte)                  {}
func (*capturingResponsesDebugTap) OnProviderObject(string, any)          {}
func (*capturingResponsesDebugTap) OnTurnBeforeConversion([]byte)         {}

func testResponsesRequestTransport(t *testing.T, rawURL, apiKey string, source credentials.BearerTokenSource, request credentials.Request) responsesRequestTransport {
	t.Helper()
	target, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse target URL: %v", err)
	}
	context, err := aitransport.ResolveAndValidate(
		request.Provider,
		responsesOperation,
		target,
		fixedResponsesRoute{target: target},
		func(*url.URL) error { return nil },
	)
	if err != nil {
		t.Fatalf("ResolveAndValidate: %v", err)
	}
	chain, err := aitransport.NewChain(&responsesBearerMiddleware{
		api:               &settings.APISettings{APIKeys: map[string]string{"open-responses-api-key": apiKey}},
		apiType:           types.ApiTypeOpenResponses,
		source:            source,
		credentialRequest: request,
	})
	if err != nil {
		t.Fatalf("NewChain: %v", err)
	}
	return responsesRequestTransport{
		request:     context,
		chain:       chain,
		headerRules: []aitransport.HeaderRule{{Name: "Authorization", Sensitive: true}},
	}
}

func TestOpenResponsesStreamRetriesOneProvider401WithReplacementBearer(t *testing.T) {
	var requests int
	var refreshes int
	source := unauthorizedBearerTokenSourceFunc{
		bearer: func(context.Context, credentials.Request) (string, error) { return "stale-token", nil },
		unauthorized: func(_ context.Context, request credentials.Request, rejected string) (string, error) {
			refreshes++
			if request.Provider != "open-responses" || request.BaseURL != "https://provider.example.test/v1" || rejected != "stale-token" {
				t.Fatalf("unexpected refresh request=%#v rejected=%q", request, rejected)
			}
			return "replacement-token", nil
		},
	}
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		want := "Bearer stale-token"
		status := http.StatusUnauthorized
		body := `{"error":"expired"}`
		if requests == 2 {
			want = "Bearer replacement-token"
			status = http.StatusOK
			body = "data: [DONE]\\n\\n"
		}
		if got := request.Header.Get("Authorization"); got != want {
			t.Fatalf("Authorization = %q, want %q", got, want)
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})}

	credentialRequest := credentials.Request{Provider: "open-responses", BaseURL: "https://provider.example.test/v1"}
	response, err := openResponsesStream(context.Background(), client, testResponsesRequestTransport(t, "https://provider.example.test/v1/responses", "stale-token", source, credentialRequest), []byte(`{"model":"test"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if requests != 2 || refreshes != 1 {
		t.Fatalf("requests=%d refreshes=%d, want 2/1", requests, refreshes)
	}
}

func TestOpenResponsesRequestRedactsMiddlewareCredentialsFromDebugTap(t *testing.T) {
	const secret = "Bearer middleware-secret"
	source := bearerTokenSourceFunc(func(context.Context, credentials.Request) (string, error) {
		return "middleware-secret", nil
	})
	credentialRequest := credentials.Request{Provider: "open-responses", BaseURL: "https://provider.example.test/v1"}
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.Header.Get("Authorization"); got != secret {
			t.Fatalf("outbound Authorization = %q, want secret", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: [DONE]\\n\\n")), Request: request}, nil
	})}
	tap := &capturingResponsesDebugTap{}
	response, err := openResponsesStream(context.Background(), client, testResponsesRequestTransport(t, "https://provider.example.test/v1/responses", "", source, credentialRequest), []byte(`{"model":"test"}`), tap)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if tap.header == nil {
		t.Fatal("debug tap did not receive request")
	}
	if got := tap.header.Get("Authorization"); got != "<redacted>" {
		t.Fatalf("debug Authorization = %q", got)
	}
	if strings.Contains(tap.header.Get("Authorization"), "middleware-secret") {
		t.Fatalf("debug Authorization leaked credential: %q", tap.header.Get("Authorization"))
	}
}

func TestResponsesRunInferenceDoesNotResolveBearerBeforeInvalidRoute(t *testing.T) {
	calls := 0
	source := bearerTokenSourceFunc(func(context.Context, credentials.Request) (string, error) {
		calls++
		return "must-not-be-requested", nil
	})
	model := "test-model"
	engine, err := NewEngine(&settings.InferenceSettings{
		API:  &settings.APISettings{BaseUrls: map[string]string{"open-responses-base-url": "http://provider.example.test/v1"}},
		Chat: &settings.ChatSettings{Engine: &model},
	}, WithBearerTokenSource(source))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	_, err = engine.RunInference(context.Background(), &turns.Turn{Blocks: []turns.Block{turns.NewUserTextBlock("hello")}})
	if err == nil || !strings.Contains(err.Error(), "http scheme is not allowed") {
		t.Fatalf("RunInference error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("bearer source calls = %d, want 0", calls)
	}
}

func TestOpenResponsesStreamDoesNotRetrySecondProvider401(t *testing.T) {
	var requests int
	var refreshes int
	source := unauthorizedBearerTokenSourceFunc{
		bearer: func(context.Context, credentials.Request) (string, error) { return "stale-token", nil },
		unauthorized: func(context.Context, credentials.Request, string) (string, error) {
			refreshes++
			return "replacement-token", nil
		},
	}
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"error":"still unauthorized"}`)), Request: request}, nil
	})}

	credentialRequest := credentials.Request{Provider: "open-responses", BaseURL: "https://provider.example.test/v1"}
	_, err := openResponsesStream(context.Background(), client, testResponsesRequestTransport(t, "https://provider.example.test/v1/responses", "stale-token", source, credentialRequest), []byte(`{"model":"test"}`), nil)
	if err == nil || !strings.Contains(err.Error(), "status=401") {
		t.Fatalf("expected second 401 error, got %v", err)
	}
	if requests != 2 || refreshes != 1 {
		t.Fatalf("requests=%d refreshes=%d, want 2/1", requests, refreshes)
	}
}

func TestResolveResponsesBearerTokenPreservesContextCancellation(t *testing.T) {
	for name, sourceErr := range map[string]error{
		"canceled": context.Canceled,
		"deadline": context.DeadlineExceeded,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := resolveResponsesBearerToken(
				context.Background(),
				&settings.APISettings{},
				types.ApiTypeOpenResponses,
				bearerTokenSourceFunc(func(context.Context, credentials.Request) (string, error) {
					return "", sourceErr
				}),
			)
			if !errors.Is(err, sourceErr) {
				t.Fatalf("expected %v to be preserved, got %v", sourceErr, err)
			}
		})
	}
}

func TestResolveResponsesBearerTokenUsesSourceAndRedactsSourceErrors(t *testing.T) {
	api := &settings.APISettings{BaseUrls: map[string]string{"open-responses-base-url": "https://responses.example/v1"}}
	var seen credentials.Request
	source := bearerTokenSourceFunc(func(_ context.Context, request credentials.Request) (string, error) {
		seen = request
		return "refreshed-token", nil
	})
	token, err := resolveResponsesBearerToken(context.Background(), api, types.ApiTypeOpenResponses, source)
	if err != nil || token != "refreshed-token" {
		t.Fatalf("resolveResponsesBearerToken() = %q, %v", token, err)
	}
	if seen.Provider != "open-responses" || seen.BaseURL != "https://responses.example/v1" {
		t.Fatalf("credential request = %#v", seen)
	}

	_, err = resolveResponsesBearerToken(context.Background(), api, types.ApiTypeOpenResponses,
		bearerTokenSourceFunc(func(context.Context, credentials.Request) (string, error) {
			return "", errors.New("refresh token is sensitive")
		}),
	)
	if err == nil || strings.Contains(err.Error(), "sensitive") {
		t.Fatalf("source error leaked: %v", err)
	}
}

func TestResponsesAPIKeyPrefersOpenResponsesKey(t *testing.T) {
	api := &settings.APISettings{
		APIKeys: map[string]string{
			"openai-api-key":           "openai-key",
			"openai-responses-api-key": "legacy-key",
			"open-responses-api-key":   "new-key",
		},
	}

	got := responsesAPIKey(api)
	if got != "new-key" {
		t.Fatalf("expected open-responses key, got %q", got)
	}
}

func TestResponsesAPIKeyFallsBackToLegacyAlias(t *testing.T) {
	api := &settings.APISettings{
		APIKeys: map[string]string{
			"openai-api-key":           "openai-key",
			"openai-responses-api-key": "legacy-key",
		},
	}

	got := responsesAPIKey(api)
	if got != "legacy-key" {
		t.Fatalf("expected legacy openai-responses key, got %q", got)
	}
}

func TestResponsesBaseURLPrefersOpenResponsesBaseURL(t *testing.T) {
	api := &settings.APISettings{
		BaseUrls: map[string]string{
			"openai-base-url":           "https://openai.example/v1",
			"openai-responses-base-url": "https://legacy.example/v1",
			"open-responses-base-url":   "https://new.example/v1",
		},
	}

	got := responsesBaseURL(api)
	if got != "https://new.example/v1" {
		t.Fatalf("expected open-responses base URL, got %q", got)
	}
}

func TestResponsesBaseURLFallsBackToOpenAIBaseURL(t *testing.T) {
	api := &settings.APISettings{
		BaseUrls: map[string]string{
			"openai-base-url": "https://openai.example/v1",
		},
	}

	got := responsesBaseURL(api)
	if got != "https://openai.example/v1" {
		t.Fatalf("expected openai fallback base URL, got %q", got)
	}
}

func TestResponsesEndpointBuildsProviderURL(t *testing.T) {
	api := &settings.APISettings{
		BaseUrls: map[string]string{
			"open-responses-base-url": "https://responses.example/v1/",
		},
	}

	got := responsesEndpoint(api, "responses/input_tokens")
	if got != "https://responses.example/v1/responses/input_tokens" {
		t.Fatalf("unexpected endpoint %q", got)
	}
}

func TestResponsesAPITypeNormalizesLegacyAliases(t *testing.T) {
	apiType := settingsWithAPIType("openai-responses")

	got := responsesAPIType(apiType)
	if got != types.ApiTypeOpenResponses {
		t.Fatalf("expected open-responses api type, got %q", got)
	}
}

func TestResponsesInferenceProviderUsesCanonicalUnderscoreName(t *testing.T) {
	apiType := settingsWithAPIType("openai")

	got := responsesInferenceProvider(apiType)
	if got != "open_responses" {
		t.Fatalf("expected open_responses provider, got %q", got)
	}
}

func settingsWithAPIType(apiType string) *settings.InferenceSettings {
	ret := &settings.InferenceSettings{
		Chat: &settings.ChatSettings{},
	}
	v := types.ApiType(apiType)
	ret.Chat.ApiType = &v
	return ret
}

func TestResponsesOutboundURLOptionsPrefersOpenResponsesAlias(t *testing.T) {
	api := settings.NewAPISettings()
	api.AllowHTTP["openai"] = false
	api.AllowHTTP["open-responses"] = true
	api.AllowLocalNetworks["openai"] = false
	api.AllowLocalNetworks["open-responses"] = true

	opts := responsesOutboundURLOptions(api)
	if !opts.AllowHTTP || !opts.AllowLocalNetworks {
		t.Fatalf("responses outbound options = %#v, want open-responses alias", opts)
	}
}

func TestResponsesOutboundURLOptionsFallsBackToOpenAI(t *testing.T) {
	api := settings.NewAPISettings()
	api.AllowHTTP["openai"] = true
	api.AllowLocalNetworks["openai"] = true

	opts := responsesOutboundURLOptions(api)
	if !opts.AllowHTTP || !opts.AllowLocalNetworks {
		t.Fatalf("responses outbound options = %#v, want openai fallback", opts)
	}
}
