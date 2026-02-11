package openai_responses

import (
	"context"
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
