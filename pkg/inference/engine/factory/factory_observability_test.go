package factory

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type captureFactoryObserver struct {
	mu      sync.Mutex
	records []geppettoobs.Record
}

func (o *captureFactoryObserver) OnGeppettoRecord(_ context.Context, rec geppettoobs.Record) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.records = append(o.records, rec)
}

func (o *captureFactoryObserver) snapshot() []geppettoobs.Record {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]geppettoobs.Record, len(o.records))
	copy(out, o.records)
	return out
}

type factoryRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f factoryRoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestStandardEngineFactory_WithOpenAIOptionsPassesObservabilityOptions(t *testing.T) {
	obs := &captureFactoryObserver{}
	factory := NewStandardEngineFactory(
		WithOpenAIOptions(
			openai.WithObserver(obs),
			openai.WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceEvents}),
		),
	)
	settings := createValidOpenAISettings()
	engineName := "gpt-test"
	settings.Chat.Engine = &engineName
	settings.Client.HTTPClient = &http.Client{Transport: factoryRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		body := strings.Join([]string{
			`data: {"id":"chatcmpl-factory","object":"chat.completion.chunk","model":"gpt-test","choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})}

	eng, err := factory.CreateEngine(settings)
	if err != nil {
		t.Fatalf("CreateEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_factory", Blocks: []turns.Block{turns.NewUserTextBlock("say ok")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	records := obs.snapshot()
	var sawStarted bool
	for _, rec := range records {
		if rec.Stage == geppettoobs.StageProviderRoutedEvent {
			t.Fatalf("TraceEvents should not emit provider records: %#v", rec)
		}
		if rec.Stage == geppettoobs.StageGeppettoPublishStarted && rec.EventType == "final" && rec.TurnID == "turn_factory" {
			sawStarted = true
		}
		if rec.Stage == geppettoobs.StageGeppettoPublishDone {
			t.Fatalf("did not expect publish-done records: %#v", rec)
		}
	}
	if !sawStarted {
		t.Fatalf("factory-created OpenAI engine did not emit final publish-started record: %#v", records)
	}
}
