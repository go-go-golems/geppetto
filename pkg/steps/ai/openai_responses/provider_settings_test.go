package openai_responses

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
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

	response, err := openResponsesStream(context.Background(), client, "https://provider.example.test/v1/responses", []byte(`{"model":"test"}`), "stale-token", source, credentials.Request{Provider: "open-responses", BaseURL: "https://provider.example.test/v1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if requests != 2 || refreshes != 1 {
		t.Fatalf("requests=%d refreshes=%d, want 2/1", requests, refreshes)
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

	_, err := openResponsesStream(context.Background(), client, "https://provider.example.test/v1/responses", []byte(`{"model":"test"}`), "stale-token", source, credentials.Request{Provider: "open-responses", BaseURL: "https://provider.example.test/v1"}, nil)
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
