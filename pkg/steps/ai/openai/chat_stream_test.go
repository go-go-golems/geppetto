package openai

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

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

func TestResolveBearerTokenPreservesContextCancellation(t *testing.T) {
	for name, sourceErr := range map[string]error{
		"canceled": context.Canceled,
		"deadline": context.DeadlineExceeded,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := resolveBearerToken(
				context.Background(),
				&settings.APISettings{},
				ai_types.ApiTypeOpenAI,
				"https://provider.example.test/v1",
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

func TestOpenChatCompletionStreamRetriesOneProvider401WithReplacementBearer(t *testing.T) {
	var requests int
	var refreshes int
	source := unauthorizedBearerTokenSourceFunc{
		bearer: func(context.Context, credentials.Request) (string, error) { return "stale-token", nil },
		unauthorized: func(_ context.Context, request credentials.Request, rejected string) (string, error) {
			refreshes++
			if request.Provider != "openai" || request.BaseURL != "https://provider.example.test/v1" || rejected != "stale-token" {
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

	stream, err := openChatCompletionStream(context.Background(), chatStreamConfig{
		baseURL:           "https://provider.example.test/v1",
		endpoint:          "https://provider.example.test/v1/chat/completions",
		apiKey:            "stale-token",
		httpClient:        client,
		bearerTokenSource: source,
		credentialRequest: credentials.Request{Provider: "openai", BaseURL: "https://provider.example.test/v1"},
	}, map[string]any{"model": "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			t.Errorf("close stream: %v", err)
		}
	}()
	if requests != 2 || refreshes != 1 {
		t.Fatalf("requests=%d refreshes=%d, want 2/1", requests, refreshes)
	}
}

func TestOpenChatCompletionStreamRedactsUnauthorizedSourceError(t *testing.T) {
	source := unauthorizedBearerTokenSourceFunc{
		bearer: func(context.Context, credentials.Request) (string, error) { return "stale-token", nil },
		unauthorized: func(context.Context, credentials.Request, string) (string, error) {
			return "", errors.New("replacement-token is sensitive")
		},
	}
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"error":"expired"}`)), Request: request}, nil
	})}

	_, err := openChatCompletionStream(context.Background(), chatStreamConfig{
		endpoint:          "https://provider.example.test/v1/chat/completions",
		apiKey:            "stale-token",
		httpClient:        client,
		bearerTokenSource: source,
		credentialRequest: credentials.Request{Provider: "openai", BaseURL: "https://provider.example.test/v1"},
	}, map[string]any{"model": "test"})
	if err == nil || strings.Contains(err.Error(), "sensitive") || strings.Contains(err.Error(), "replacement-token") {
		t.Fatalf("expected redacted unauthorized source error, got %v", err)
	}
}

func TestOpenChatCompletionStreamDoesNotRetrySecondProvider401(t *testing.T) {
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

	_, err := openChatCompletionStream(context.Background(), chatStreamConfig{
		endpoint:          "https://provider.example.test/v1/chat/completions",
		apiKey:            "stale-token",
		httpClient:        client,
		bearerTokenSource: source,
		credentialRequest: credentials.Request{Provider: "openai", BaseURL: "https://provider.example.test/v1"},
	}, map[string]any{"model": "test"})
	if err == nil || !strings.Contains(err.Error(), "status=401") {
		t.Fatalf("expected second 401 error, got %v", err)
	}
	if requests != 2 || refreshes != 1 {
		t.Fatalf("requests=%d refreshes=%d, want 2/1", requests, refreshes)
	}
}

func TestOpenChatCompletionStream_404HintsWhenBaseURLLooksLikeChatEndpoint(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	_, err := openChatCompletionStream(t.Context(), chatStreamConfig{
		baseURL:    "https://pass.wafer.ai/v1/chat/completions",
		endpoint:   server.URL + "/v1/chat/completions/chat/completions",
		apiKey:     "test-key",
		httpClient: server.Client(),
	}, map[string]any{"model": "test"})
	if err == nil {
		t.Fatal("expected 404 error")
	}
	msg := err.Error()
	for _, want := range []string{
		"status=404",
		"possible OpenAI-compatible base URL misconfiguration",
		"already looks like a chat completions endpoint",
		"Geppetto appends /chat/completions internally",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error to contain %q, got %q", want, msg)
		}
	}
}

func TestSuspiciousChatCompletionBaseURLReason(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "provider root", url: "https://pass.wafer.ai/v1", want: false},
		{name: "provider root trailing slash", url: "https://pass.wafer.ai/v1/", want: false},
		{name: "chat completions endpoint", url: "https://pass.wafer.ai/v1/chat/completions", want: true},
		{name: "extra path after v1", url: "https://example.com/v1/openai", want: true},
		{name: "rootless provider", url: "https://example.com", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := suspiciousChatCompletionBaseURLReason(tt.url)
			if got != tt.want {
				t.Fatalf("suspiciousChatCompletionBaseURLReason(%q)=%v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestReadSSEFrame_MultilineDataAndEvent(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(strings.Join([]string{
		"event: chunk",
		`data: {"delta":"Hello"}`,
		`data: {"delta":" world"}`,
		"",
	}, "\n")))

	frame, err := readSSEFrame(reader)
	if err != nil {
		t.Fatalf("readSSEFrame: %v", err)
	}
	if frame.Event != "chunk" {
		t.Fatalf("expected event=chunk, got %q", frame.Event)
	}
	want := "{\"delta\":\"Hello\"}\n{\"delta\":\" world\"}"
	if frame.Data != want {
		t.Fatalf("expected combined data %q, got %q", want, frame.Data)
	}
}

func TestNormalizeChatStreamEvent_PrefersReasoningAlias(t *testing.T) {
	event := normalizeChatStreamEvent(map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"content":           "",
					"reasoning":         "Thinking",
					"reasoning_content": "Ignored",
				},
				"finish_reason": nil,
			},
		},
	})

	if event.DeltaReasoning != "Thinking" {
		t.Fatalf("expected Together reasoning alias to win, got %q", event.DeltaReasoning)
	}
}

func TestNormalizeChatStreamEvent_FallsBackToReasoningContent(t *testing.T) {
	event := normalizeChatStreamEvent(map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"reasoning_content": "DeepSeek thinking",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(3),
			"completion_tokens": float64(5),
			"prompt_tokens_details": map[string]any{
				"cached_tokens": float64(2),
			},
			"completion_tokens_details": map[string]any{
				"reasoning_tokens": float64(4),
			},
		},
	})

	if event.DeltaReasoning != "DeepSeek thinking" {
		t.Fatalf("expected reasoning_content fallback, got %q", event.DeltaReasoning)
	}
	if event.FinishReason == nil || *event.FinishReason != "stop" {
		t.Fatalf("expected finish reason stop, got %#v", event.FinishReason)
	}
	if event.Usage == nil {
		t.Fatalf("expected usage to be normalized")
	}
	if event.Usage.promptTokens != 3 || event.Usage.completionTokens != 5 || event.Usage.cachedTokens != 2 || event.Usage.reasoningTokens != 4 {
		t.Fatalf(
			"unexpected usage normalization: prompt=%d completion=%d cached=%d reasoning=%d",
			event.Usage.promptTokens,
			event.Usage.completionTokens,
			event.Usage.cachedTokens,
			event.Usage.reasoningTokens,
		)
	}
}

func TestNormalizeChatStreamEvent_NormalizesToolCalls(t *testing.T) {
	event := normalizeChatStreamEvent(map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"index": float64(0),
							"id":    "call_1",
							"type":  "function",
							"function": map[string]any{
								"name":      "lookup",
								"arguments": "{\"q\":\"cats\"}",
							},
						},
					},
				},
			},
		},
	})

	if len(event.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(event.ToolCalls))
	}
	call := event.ToolCalls[0]
	if call.Index == nil || *call.Index != 0 {
		t.Fatalf("expected tool call index 0, got %#v", call.Index)
	}
	if call.ID != "call_1" {
		t.Fatalf("expected tool call id call_1, got %q", call.ID)
	}
	if call.Function.Name != "lookup" {
		t.Fatalf("expected tool call name lookup, got %q", call.Function.Name)
	}
	if call.Function.Arguments != "{\"q\":\"cats\"}" {
		t.Fatalf("unexpected tool call arguments: %q", call.Function.Arguments)
	}
}

func TestResolveChatStreamConfigRejectsLocalHTTPByDefault(t *testing.T) {
	apiSettings := settings.NewAPISettings()
	apiSettings.APIKeys["openai-api-key"] = "test-key"
	apiSettings.BaseUrls["openai-base-url"] = "http://127.0.0.1:9999/v1"

	_, err := resolveChatStreamConfig(context.Background(), apiSettings, nil, ai_types.ApiTypeOpenAI, nil)
	if err == nil {
		t.Fatal("expected local HTTP base URL to be rejected by default")
	}
	if !strings.Contains(err.Error(), "http scheme is not allowed") {
		t.Fatalf("expected http rejection, got %v", err)
	}
}

func TestResolveChatStreamConfigAllowsLocalHTTPWhenProfileOptsIn(t *testing.T) {
	apiSettings := settings.NewAPISettings()
	apiSettings.APIKeys["openai-api-key"] = "test-key"
	apiSettings.BaseUrls["openai-base-url"] = "http://127.0.0.1:9999/v1"
	apiSettings.AllowHTTP["openai"] = true
	apiSettings.AllowLocalNetworks["openai"] = true

	cfg, err := resolveChatStreamConfig(context.Background(), apiSettings, nil, ai_types.ApiTypeOpenAI, nil)
	if err != nil {
		t.Fatalf("resolveChatStreamConfig: %v", err)
	}
	if got := cfg.endpoint; got != "http://127.0.0.1:9999/v1/chat/completions" {
		t.Fatalf("endpoint = %q", got)
	}
}
