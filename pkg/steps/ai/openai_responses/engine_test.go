package openai_responses

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
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
	out := make([]events.Event, len(s.events))
	copy(out, s.events)
	return out
}

func ptr[T any](v T) *T { return &v }

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestRunInference_StreamingErrorReturnsFailureAndNoFinalEvent(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := "event: error\n" +
				"data: {\"error\":{\"message\":\"stream broke\",\"code\":\"upstream_failure\"}}\n\n"
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.StepSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-4o-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err == nil {
		t.Fatalf("expected streaming error to be returned")
	}
	if !strings.Contains(err.Error(), "stream") {
		t.Fatalf("expected stream-related error, got %q", err.Error())
	}

	var errorEvents int
	var finalEvents int
	for _, event := range sink.snapshot() {
		if event.Type() == events.EventTypeError {
			errorEvents++
		}
		if event.Type() == events.EventTypeFinal {
			finalEvents++
		}
	}
	if errorEvents == 0 {
		t.Fatalf("expected at least one error event")
	}
	if finalEvents != 0 {
		t.Fatalf("did not expect final success event on streaming failure, got %d", finalEvents)
	}
}

func TestRunInference_StreamingReasoningTextEventsArePublished(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := strings.Join([]string{
				"event: response.output_item.added",
				`data: {"item":{"type":"reasoning","id":"rs_1"}}`,
				"",
				"event: response.reasoning_text.delta",
				`data: {"delta":"Thinking "}`,
				"",
				"event: response.reasoning_text.delta",
				`data: {"delta":"hard."}`,
				"",
				"event: response.reasoning_text.done",
				`data: {"text":"Thinking hard."}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"reasoning","id":"rs_1"}}`,
				"",
				"event: response.output_item.added",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"42"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.completed",
				`data: {"response":{"usage":{"input_tokens":10,"output_tokens":5,"input_tokens_details":{"cached_tokens":4},"output_tokens_details":{"reasoning_tokens":3}}}}`,
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.StepSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var reasoningDeltaEvents int
	var reasoningDoneEvents int
	var thinkingPartialEvents int
	var finalEvents int
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
			finalEvents++
			finalUsage = e.Metadata().Usage
			if e.Metadata().Extra != nil {
				if s, ok := e.Metadata().Extra["thinking_text"].(string); ok {
					finalThinkingText = s
				}
			}
		}
	}

	if reasoningDeltaEvents == 0 {
		t.Fatalf("expected reasoning delta events")
	}
	if reasoningDoneEvents == 0 {
		t.Fatalf("expected reasoning done events")
	}
	if thinkingPartialEvents == 0 {
		t.Fatalf("expected mirrored partial-thinking events for reasoning text")
	}
	if finalEvents != 1 {
		t.Fatalf("expected exactly one final event, got %d", finalEvents)
	}
	if finalThinkingText != "Thinking hard." {
		t.Fatalf("expected final metadata thinking_text to be propagated, got %q", finalThinkingText)
	}
	if finalUsage == nil {
		t.Fatalf("expected final usage metadata")
	}
	if finalUsage.InputTokens != 10 || finalUsage.OutputTokens != 5 || finalUsage.CachedTokens != 4 {
		t.Fatalf("expected usage input=10 output=5 cached=4, got input=%d output=%d cached=%d", finalUsage.InputTokens, finalUsage.OutputTokens, finalUsage.CachedTokens)
	}
}

func TestRunInference_StreamingReasoningTextDonePreservesAccumulatedThinking(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := strings.Join([]string{
				"event: response.output_item.added",
				`data: {"item":{"type":"reasoning","id":"rs_1"}}`,
				"",
				"event: response.reasoning_text.delta",
				`data: {"delta":"First "}`,
				"",
				"event: response.reasoning_text.delta",
				`data: {"delta":"thought."}`,
				"",
				"event: response.reasoning_text.done",
				`data: {"text":"First thought.","item_id":"rs_1"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"reasoning","id":"rs_1"}}`,
				"",
				"event: response.output_item.added",
				`data: {"item":{"type":"reasoning","id":"rs_2"}}`,
				"",
				"event: response.reasoning_text.done",
				`data: {"text":" Second thought.","item_id":"rs_2"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"reasoning","id":"rs_2"}}`,
				"",
				"event: response.output_item.added",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"42"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.completed",
				`data: {"response":{"usage":{"input_tokens":10,"output_tokens":5,"output_tokens_details":{"reasoning_tokens":3}}}}`,
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.StepSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var reasoningDoneEvents int
	var finalThinkingText string
	for _, event := range sink.snapshot() {
		switch e := event.(type) {
		case *events.EventReasoningTextDone:
			reasoningDoneEvents++
		case *events.EventFinal:
			if e.Metadata().Extra != nil {
				if s, ok := e.Metadata().Extra["thinking_text"].(string); ok {
					finalThinkingText = s
				}
			}
		}
	}

	if reasoningDoneEvents != 2 {
		t.Fatalf("expected two reasoning done events, got %d", reasoningDoneEvents)
	}
	if finalThinkingText != "First thought. Second thought." {
		t.Fatalf("expected combined thinking_text, got %q", finalThinkingText)
	}
}

func TestRunInference_NonStreamingUsageIncludesCachedTokens(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := `{
  "output": [
    {
      "type": "message",
      "id": "msg_1",
      "content": [
        {"type": "output_text", "text": "Done"}
      ]
    }
  ],
  "usage": {
    "input_tokens": 12,
    "output_tokens": 7,
    "input_tokens_details": {"cached_tokens": 5},
    "output_tokens_details": {"reasoning_tokens": 2}
  }
}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.StepSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: false,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var finalEvents int
	var finalUsage *events.Usage
	var reasoningTokens int
	for _, event := range sink.snapshot() {
		switch e := event.(type) {
		case *events.EventFinal:
			finalEvents++
			finalUsage = e.Metadata().Usage
			if e.Metadata().Extra != nil {
				if v, ok := e.Metadata().Extra["reasoning_tokens"].(int); ok {
					reasoningTokens = v
				}
			}
		}
	}

	if finalEvents != 1 {
		t.Fatalf("expected exactly one final event, got %d", finalEvents)
	}
	if finalUsage == nil {
		t.Fatalf("expected final usage metadata")
	}
	if finalUsage.InputTokens != 12 || finalUsage.OutputTokens != 7 || finalUsage.CachedTokens != 5 {
		t.Fatalf("expected usage input=12 output=7 cached=5, got input=%d output=%d cached=%d", finalUsage.InputTokens, finalUsage.OutputTokens, finalUsage.CachedTokens)
	}
	if reasoningTokens != 2 {
		t.Fatalf("expected reasoning_tokens=2 in metadata extra, got %d", reasoningTokens)
	}
}

func TestParseUsageTotalsFromResponse_NestedResponseUsage(t *testing.T) {
	rr := responsesResponse{
		Response: &responsesResponseNested{
			Usage: json.RawMessage(`{
  "input_tokens": 9,
  "output_tokens": 4,
  "input_tokens_details": {"cached_tokens": 3},
  "output_tokens_details": {"reasoning_tokens": 1}
}`),
		},
	}

	totals, ok := parseUsageTotalsFromResponse(rr)
	if !ok {
		t.Fatalf("expected usage totals from nested response.usage")
	}
	if totals.inputTokens != 9 || totals.outputTokens != 4 || totals.cachedTokens != 3 || totals.reasoningTokens != 1 {
		t.Fatalf(
			"unexpected totals: input=%d output=%d cached=%d reasoning=%d",
			totals.inputTokens,
			totals.outputTokens,
			totals.cachedTokens,
			totals.reasoningTokens,
		)
	}
}
