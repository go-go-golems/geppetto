package openai

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
