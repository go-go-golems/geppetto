package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aisettingsopenai "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type capturingEventSink struct {
	mu     sync.Mutex
	events []events.Event
}

func (s *capturingEventSink) PublishEvent(event events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *capturingEventSink) snapshot() []events.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	ret := make([]events.Event, len(s.events))
	copy(ret, s.events)
	return ret
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func loadChatFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", "chat-stream", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

func newStreamingTestEngine(t *testing.T) *OpenAIEngine {
	t.Helper()
	engine := "gpt-4o-mini"
	apiType := ai_types.ApiTypeOpenAI
	eng, err := NewOpenAIEngine(&aisettings.InferenceSettings{
		API: &aisettings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Client: &aisettings.ClientSettings{HTTPClient: http.DefaultClient},
		OpenAI: &aisettingsopenai.Settings{},
		Chat: &aisettings.ChatSettings{
			ApiType: &apiType,
			Engine:  &engine,
			Stream:  true,
		},
	})
	if err != nil {
		t.Fatalf("NewOpenAIEngine: %v", err)
	}
	return eng
}

func withFixtureTransport(t *testing.T, fixtureName string, fn func()) {
	t.Helper()
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/chat/completions" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer test" {
				t.Fatalf("expected authorization header, got %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(loadChatFixture(t, fixtureName))),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()
	fn()
}

func TestRunInference_StreamTogetherReasoningPublishesEventsAndPersistsBlock(t *testing.T) {
	withFixtureTransport(t, "together_reasoning.sse", func() {
		eng := newStreamingTestEngine(t)
		sink := &capturingEventSink{}
		ctx := events.WithEventSinks(context.Background(), sink)
		turn := &turns.Turn{Blocks: []turns.Block{
			turns.NewSystemTextBlock("You are a careful assistant."),
			turns.NewUserTextBlock("Hello"),
		}}

		out, err := eng.RunInference(ctx, turn)
		if err != nil {
			t.Fatalf("RunInference: %v", err)
		}

		var reasoningDeltaEvents int
		var reasoningDoneEvents int
		var thinkingPartialEvents int
		var finalThinkingText string
		var finalUsage *events.Usage
		for _, event := range sink.snapshot() {
			switch e := event.(type) {
			case *events.EventReasoningTextDelta:
				reasoningDeltaEvents++
			case *events.EventReasoningTextDone:
				reasoningDoneEvents++
			case *events.EventThinkingPartial:
				thinkingPartialEvents++
			case *events.EventFinal:
				finalUsage = e.Metadata().Usage
				if e.Metadata().Extra != nil {
					if s, ok := e.Metadata().Extra["thinking_text"].(string); ok {
						finalThinkingText = s
					}
				}
			}
		}

		if reasoningDeltaEvents != 2 {
			t.Fatalf("expected 2 reasoning delta events, got %d", reasoningDeltaEvents)
		}
		if reasoningDoneEvents != 1 {
			t.Fatalf("expected 1 reasoning done event, got %d", reasoningDoneEvents)
		}
		if thinkingPartialEvents != 2 {
			t.Fatalf("expected 2 partial-thinking events, got %d", thinkingPartialEvents)
		}
		if finalThinkingText != "Thinking hard." {
			t.Fatalf("expected final thinking_text to be propagated, got %q", finalThinkingText)
		}
		if finalUsage == nil || finalUsage.InputTokens != 10 || finalUsage.OutputTokens != 5 || finalUsage.CachedTokens != 4 {
			t.Fatalf("unexpected final usage: %#v", finalUsage)
		}

		if len(out.Blocks) < 2 {
			t.Fatalf("expected reasoning and assistant blocks, got %d blocks", len(out.Blocks))
		}
		if out.Blocks[len(out.Blocks)-2].Kind != turns.BlockKindReasoning {
			t.Fatalf("expected reasoning block before assistant block, got kind %v", out.Blocks[len(out.Blocks)-2].Kind)
		}
		if got, _ := out.Blocks[len(out.Blocks)-2].Payload[turns.PayloadKeyText].(string); got != "Thinking hard." {
			t.Fatalf("expected reasoning text persistence, got %q", got)
		}
		if out.Blocks[len(out.Blocks)-1].Kind != turns.BlockKindLLMText {
			t.Fatalf("expected final assistant text block, got kind %v", out.Blocks[len(out.Blocks)-1].Kind)
		}
		if got, _ := out.Blocks[len(out.Blocks)-1].Payload[turns.PayloadKeyText].(string); got != "42" {
			t.Fatalf("expected assistant text 42, got %q", got)
		}
	})
}

func TestRunInference_StreamDeepSeekReasoningContentPublishesEvents(t *testing.T) {
	withFixtureTransport(t, "deepseek_reasoning_content.sse", func() {
		eng := newStreamingTestEngine(t)
		sink := &capturingEventSink{}
		ctx := events.WithEventSinks(context.Background(), sink)
		turn := &turns.Turn{Blocks: []turns.Block{
			turns.NewUserTextBlock("Hello"),
		}}

		out, err := eng.RunInference(ctx, turn)
		if err != nil {
			t.Fatalf("RunInference: %v", err)
		}

		var reasoningDeltaEvents int
		for _, event := range sink.snapshot() {
			switch event.(type) {
			case *events.EventReasoningTextDelta:
				reasoningDeltaEvents++
			}
		}
		if reasoningDeltaEvents != 2 {
			t.Fatalf("expected 2 reasoning delta events, got %d", reasoningDeltaEvents)
		}

		if len(out.Blocks) < 2 {
			t.Fatalf("expected reasoning and assistant blocks, got %d", len(out.Blocks))
		}
		if got, _ := out.Blocks[len(out.Blocks)-2].Payload[turns.PayloadKeyText].(string); got != "Think twice." {
			t.Fatalf("expected reasoning text Think twice., got %q", got)
		}
	})
}

func TestRunInference_StreamTextOnlyBehaviorIsUnchanged(t *testing.T) {
	withFixtureTransport(t, "text_only.sse", func() {
		eng := newStreamingTestEngine(t)
		sink := &capturingEventSink{}
		ctx := events.WithEventSinks(context.Background(), sink)
		turn := &turns.Turn{Blocks: []turns.Block{
			turns.NewUserTextBlock("Hello"),
		}}

		out, err := eng.RunInference(ctx, turn)
		if err != nil {
			t.Fatalf("RunInference: %v", err)
		}

		var reasoningEvents int
		var partialEvents int
		for _, event := range sink.snapshot() {
			switch event.(type) {
			case *events.EventReasoningTextDelta, *events.EventReasoningTextDone, *events.EventThinkingPartial:
				reasoningEvents++
			case *events.EventPartialCompletion:
				partialEvents++
			}
		}
		if reasoningEvents != 0 {
			t.Fatalf("expected no reasoning events for text-only stream, got %d", reasoningEvents)
		}
		if partialEvents != 2 {
			t.Fatalf("expected 2 partial completion events, got %d", partialEvents)
		}
		if len(out.Blocks) == 0 {
			t.Fatalf("expected output blocks")
		}
		last := out.Blocks[len(out.Blocks)-1]
		if last.Kind != turns.BlockKindLLMText {
			t.Fatalf("expected assistant text block, got kind %v", last.Kind)
		}
		if got, _ := last.Payload[turns.PayloadKeyText].(string); got != "Hello" {
			t.Fatalf("expected assistant text Hello, got %q", got)
		}
	})
}

func TestRunInference_ForcesStreamingRequestBodyEvenWhenProfileStreamDisabled(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if got, _ := payload["stream"].(bool); !got {
				t.Fatalf("expected stream=true in request body, got %#v", payload["stream"])
			}
			streamOptions, _ := payload["stream_options"].(map[string]any)
			if got, _ := streamOptions["include_usage"].(bool); !got {
				t.Fatalf("expected stream_options.include_usage=true, got %#v", payload["stream_options"])
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(loadChatFixture(t, "text_only.sse"))),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	engine := "gpt-4o-mini"
	apiType := ai_types.ApiTypeOpenAI
	eng, err := NewOpenAIEngine(&aisettings.InferenceSettings{
		API: &aisettings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Client: &aisettings.ClientSettings{HTTPClient: http.DefaultClient},
		OpenAI: &aisettingsopenai.Settings{},
		Chat: &aisettings.ChatSettings{
			ApiType: &apiType,
			Engine:  &engine,
			Stream:  false,
		},
	})
	if err != nil {
		t.Fatalf("NewOpenAIEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("Hello"),
	}}

	out, err := eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("RunInference: %v", err)
	}
	if len(out.Blocks) == 0 {
		t.Fatal("expected output blocks")
	}
}

func TestRunInference_StreamToolCallsAreMergedAndUsagePreserved(t *testing.T) {
	withFixtureTransport(t, "tool_calls_fragmented.sse", func() {
		eng := newStreamingTestEngine(t)
		sink := &capturingEventSink{}
		ctx := events.WithEventSinks(context.Background(), sink)
		turn := &turns.Turn{Blocks: []turns.Block{
			turns.NewUserTextBlock("Search for cats"),
		}}

		out, err := eng.RunInference(ctx, turn)
		if err != nil {
			t.Fatalf("RunInference: %v", err)
		}

		var toolCallEvents int
		var finalUsage *events.Usage
		for _, event := range sink.snapshot() {
			switch e := event.(type) {
			case *events.EventToolCall:
				toolCallEvents++
				if e.ToolCall.Name != "lookup" {
					t.Fatalf("expected merged tool call name lookup, got %q", e.ToolCall.Name)
				}
				if e.ToolCall.Input != "{\"q\":\"cats\"}" {
					t.Fatalf("expected merged tool args, got %q", e.ToolCall.Input)
				}
			case *events.EventFinal:
				finalUsage = e.Metadata().Usage
			}
		}
		if toolCallEvents != 1 {
			t.Fatalf("expected one merged tool call event, got %d", toolCallEvents)
		}
		if finalUsage == nil || finalUsage.InputTokens != 8 || finalUsage.OutputTokens != 4 {
			t.Fatalf("unexpected final usage: %#v", finalUsage)
		}

		if len(out.Blocks) == 0 {
			t.Fatalf("expected tool call block in output turn")
		}
		last := out.Blocks[len(out.Blocks)-1]
		if last.Kind != turns.BlockKindToolCall {
			t.Fatalf("expected final block to be tool call, got kind %v", last.Kind)
		}
		if got, _ := last.Payload[turns.PayloadKeyName].(string); got != "lookup" {
			t.Fatalf("expected tool call name lookup, got %q", got)
		}
	})
}
