//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	openai_responses "github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openaisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type capturingSink struct {
	mu     sync.Mutex
	events []events.Event
}

func (s *capturingSink) PublishEvent(event events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *capturingSink) snapshot() []events.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]events.Event, len(s.events))
	copy(out, s.events)
	return out
}

func ptr[T any](v T) *T { return &v }

type scenario struct {
	name string
	sse  string
}

func runScenario(sc scenario) (map[string]int, error) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(sc.sse)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := openai_responses.NewEngine(&settings.StepSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine:            ptr("gpt-5-mini"),
			Stream:            true,
			MaxResponseTokens: ptr(256),
		},
		OpenAI: &openaisettings.Settings{
			ReasoningEffort:  ptr("medium"),
			ReasoningSummary: ptr("auto"),
		},
	})
	if err != nil {
		return nil, err
	}

	sink := &capturingSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a test assistant."),
		turns.NewUserTextBlock("Return a short answer."),
	}}
	_, err = eng.RunInference(ctx, turn)
	if err != nil {
		return nil, err
	}

	counts := map[string]int{}
	for _, ev := range sink.snapshot() {
		counts["type:"+string(ev.Type())]++
		switch tev := ev.(type) {
		case *events.EventInfo:
			counts["info:"+tev.Message]++
		case *events.EventReasoningTextDelta:
			counts["reasoning_text_delta"]++
		case *events.EventReasoningTextDone:
			counts["reasoning_text_done"]++
		}
	}
	return counts, nil
}

func renderCounts(name string, counts map[string]int) {
	fmt.Printf("=== %s ===\n", name)
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s=%d\n", k, counts[k])
	}
	fmt.Println()
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	summarySSE := strings.Join([]string{
		"event: response.output_item.added",
		`data: {"item":{"type":"reasoning","id":"rs_summary_1"}}`,
		"",
		"event: response.reasoning_summary_part.added",
		`data: {"item_id":"rs_summary_1","summary_index":0}`,
		"",
		"event: response.reasoning_summary_text.delta",
		`data: {"item_id":"rs_summary_1","summary_index":0,"delta":"Reasoning summary chunk."}`,
		"",
		"event: response.reasoning_summary_part.done",
		`data: {"item_id":"rs_summary_1","summary_index":0}`,
		"",
		"event: response.output_item.done",
		`data: {"item":{"type":"reasoning","id":"rs_summary_1"}}`,
		"",
		"event: response.output_item.added",
		`data: {"item":{"type":"message","id":"msg_summary_1"}}`,
		"",
		"event: response.output_text.delta",
		`data: {"delta":"42"}`,
		"",
		"event: response.output_item.done",
		`data: {"item":{"type":"message","id":"msg_summary_1"}}`,
		"",
		"event: response.completed",
		`data: {"response":{"usage":{"input_tokens":12,"output_tokens":3,"output_tokens_details":{"reasoning_tokens":9}}}}`,
		"",
	}, "\n")

	reasoningTextSSE := strings.Join([]string{
		"event: response.output_item.added",
		`data: {"item":{"type":"reasoning","id":"rs_text_1"}}`,
		"",
		"event: response.reasoning_text.delta",
		`data: {"item_id":"rs_text_1","delta":"Reasoning text chunk."}`,
		"",
		"event: response.reasoning_text.done",
		`data: {"item_id":"rs_text_1","text":"Reasoning text done."}`,
		"",
		"event: response.output_item.done",
		`data: {"item":{"type":"reasoning","id":"rs_text_1"}}`,
		"",
		"event: response.output_item.added",
		`data: {"item":{"type":"message","id":"msg_text_1"}}`,
		"",
		"event: response.output_text.delta",
		`data: {"delta":"42"}`,
		"",
		"event: response.output_item.done",
		`data: {"item":{"type":"message","id":"msg_text_1"}}`,
		"",
		"event: response.completed",
		`data: {"response":{"usage":{"input_tokens":12,"output_tokens":3,"output_tokens_details":{"reasoning_tokens":9}}}}`,
		"",
	}, "\n")

	cases := []scenario{
		{name: "summary-delta-events", sse: summarySSE},
		{name: "reasoning-text-events", sse: reasoningTextSSE},
	}

	for _, c := range cases {
		counts, err := runScenario(c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scenario %q failed: %v\n", c.name, err)
			os.Exit(1)
		}
		renderCounts(c.name, counts)
	}
}
