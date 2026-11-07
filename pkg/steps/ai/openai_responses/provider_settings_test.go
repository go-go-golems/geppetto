package openai_responses

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
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

func testResponsesConnection(t *testing.T, client *http.Client, rawURL, apiKey string, source credentials.BearerTokenSource, request credentials.Request) *connection.Connection {
	t.Helper()
	target, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	auth := connection.StaticBearer(apiKey)
	if source != nil {
		auth = connection.Bearer(source)
	}
	c, err := connection.New(connection.Config{Provider: request.Provider, BaseURL: request.BaseURL, Routes: connection.Routes{responsesOperation: fixedResponsesRoute{target: target}}, Auth: auth, HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	return c
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
	response, err := openResponsesStream(context.Background(), testResponsesConnection(t, client, "https://provider.example.test/v1/responses", "stale-token", source, credentialRequest), []byte(`{"model":"test"}`), nil)
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
	response, err := openResponsesStream(context.Background(), testResponsesConnection(t, client, "https://provider.example.test/v1/responses", "", source, credentialRequest), []byte(`{"model":"test"}`), tap)
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
	_, err := NewEngine(&settings.InferenceSettings{
		API:  &settings.APISettings{BaseUrls: map[string]string{"open-responses-base-url": "http://provider.example.test/v1"}},
		Chat: &settings.ChatSettings{Engine: &model},
	}, WithBearerTokenSource(source))

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
	_, err := openResponsesStream(context.Background(), testResponsesConnection(t, client, "https://provider.example.test/v1/responses", "stale-token", source, credentialRequest), []byte(`{"model":"test"}`), nil)
	if err == nil || !strings.Contains(err.Error(), "status=401") {
		t.Fatalf("expected second 401 error, got %v", err)
	}
	if requests != 2 || refreshes != 1 {
		t.Fatalf("requests=%d refreshes=%d, want 2/1", requests, refreshes)
	}
}

func TestResponsesCredentialCancellationAndSourceErrors(t *testing.T) {
	for _, sourceErr := range []error{context.Canceled, context.DeadlineExceeded, errors.New("SECRET")} {
		calls := 0
		source := bearerTokenSourceFunc(func(_ context.Context, request credentials.Request) (string, error) {
			calls++
			if request.Provider != "open-responses" || request.BaseURL != "https://responses.example/v1" {
				t.Fatalf("wrong binding: %+v", request)
			}
			return "", sourceErr
		})
		api := &settings.APISettings{BaseUrls: map[string]string{"open-responses-base-url": "https://responses.example/v1"}}
		c, err := resolveResponsesConnection(&settings.InferenceSettings{API: api}, source, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = openResponsesStream(t.Context(), c, []byte(`{}`), nil)
		if err == nil || strings.Contains(err.Error(), "SECRET") {
			t.Fatalf("unsafe source error: %v", err)
		}
		if sourceErr == context.Canceled || sourceErr == context.DeadlineExceeded {
			if !errors.Is(err, sourceErr) {
				t.Fatalf("lost cancellation: %v", err)
			}
		}
		if calls != 1 {
			t.Fatalf("source calls: %d", calls)
		}
	}
}

func TestResponsesAPIKeyPrefersOpenResponsesKey(t *testing.T) {
	api := &settings.APISettings{
		APIKeys: map[string]string{
			"openai-api-key":           "openai-key",
			"openai-responses-api-key": "legacy-key",
			"open-responses-api-key":   "new-key",
		},
		BaseUrls: map[string]string{"open-responses-base-url": "https://responses.example/v1"},
	}

	slot, err := resolveResponsesSlot(api, false)
	if err != nil {
		t.Fatal(err)
	}
	got := slot.key
	if got != "new-key" {
		t.Fatalf("expected open-responses key, got %q", got)
	}
}

// GEPPETTO-CREDSCOPE-001: the key and the URL come from the same slot. An
// open-responses key without open-responses-base-url used to be paired with
// openai-base-url (whose flag default is api.openai.com), sending one
// provider's key to another.
func TestResponsesSlotPairsKeyWithItsOwnURL(t *testing.T) {
	leak := &settings.APISettings{
		APIKeys:  map[string]string{"open-responses-api-key": "vendor-key"},
		BaseUrls: map[string]string{"openai-base-url": "https://api.openai.com/v1"},
	}
	if _, err := resolveResponsesSlot(leak, false); err == nil {
		t.Fatal("open-responses key without its own URL must be an error")
	}

	cases := []struct {
		name      string
		api       *settings.APISettings
		sourced   bool
		wantKey   string
		wantURL   string
		wantAPITy types.ApiType
	}{
		{"openai slot keeps its URL", &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "k"},
			BaseUrls: map[string]string{"openai-base-url": "https://gw.example/v1", "open-responses-base-url": "https://other.example/v1"},
		}, false, "k", "https://gw.example/v1", types.ApiTypeOpenAI},
		{"openai slot defaults to OpenAI", &settings.APISettings{
			APIKeys: map[string]string{"openai-api-key": "k"},
		}, false, "k", openAIDefaultBaseURL, types.ApiTypeOpenAI},
		{"legacy alias may use openai-base-url", &settings.APISettings{
			APIKeys:  map[string]string{"openai-responses-api-key": "k"},
			BaseUrls: map[string]string{"openai-base-url": "https://gw.example/v1"},
		}, false, "k", "https://gw.example/v1", types.ApiTypeOpenAIResponses},
		{"token source picks the first slot with a URL", &settings.APISettings{
			BaseUrls: map[string]string{"openai-responses-base-url": "https://pi.example/v1", "openai-base-url": "https://api.openai.com/v1"},
		}, true, "", "https://pi.example/v1", types.ApiTypeOpenAIResponses},
		{"no key anywhere picks the first slot with a URL (Codex)", &settings.APISettings{
			BaseUrls: map[string]string{"open-responses-base-url": "https://chatgpt.com/backend-api"},
		}, false, "", "https://chatgpt.com/backend-api", types.ApiTypeOpenResponses},
	}
	for _, tc := range cases {
		got, err := resolveResponsesSlot(tc.api, tc.sourced)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got.key != tc.wantKey || got.baseURL != tc.wantURL || got.apiType != tc.wantAPITy {
			t.Fatalf("%s: got %+v", tc.name, got)
		}
	}
}

func TestResponsesAPIKeyFallsBackToLegacyAlias(t *testing.T) {
	api := &settings.APISettings{
		APIKeys: map[string]string{
			"openai-api-key":           "openai-key",
			"openai-responses-api-key": "legacy-key",
		},
	}

	slot, err := resolveResponsesSlot(api, false)
	if err != nil {
		t.Fatal(err)
	}
	got := slot.key
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

	slot, err := resolveResponsesSlot(api, false)
	if err != nil {
		t.Fatal(err)
	}
	got := slot.baseURL
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

	slot, err := resolveResponsesSlot(api, false)
	if err != nil {
		t.Fatal(err)
	}
	got := slot.baseURL
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

	source, _ := credentials.NewStaticBearerTokenSource("test-key")
	c, err := resolveResponsesConnection(&settings.InferenceSettings{API: api}, source, nil)
	if err != nil {
		t.Fatal(err)
	}
	route, err := c.Resolve("input-tokens")
	if err != nil {
		t.Fatal(err)
	}
	target := route.URL()
	got := target.String()
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
