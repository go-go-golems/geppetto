package runner

import (
	"context"
	"sync"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type emittingEngine struct {
	mu    sync.Mutex
	calls int
}

var _ engine.Engine = (*emittingEngine)(nil)

func (e *emittingEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()

	events.PublishEventToContext(ctx, events.NewInfoEvent(events.EventMetadata{}, "runner-test", map[string]interface{}{"source": "emitting-engine"}))

	out := t.Clone()
	if out == nil {
		out = &turns.Turn{}
	}
	turns.AppendBlock(out, turns.NewAssistantTextBlock("done"))
	return out, nil
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

func (s *capturingSink) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

func newFakeEngineRunner(t *testing.T, eng engine.Engine) *Runner {
	t.Helper()

	r := New()
	r.engineFactory = func(*settings.InferenceSettings) (engine.Engine, error) {
		return eng, nil
	}
	return r
}

func TestStartReturnsHandleAndCompletes(t *testing.T) {
	eng := &emittingEngine{}
	r := newFakeEngineRunner(t, eng)

	prepared, handle, err := r.Start(context.Background(), StartRequest{
		Prompt: "hello",
		Runtime: Runtime{
			InferenceSettings: newTestInferenceSettings(t),
		},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if prepared == nil || handle == nil {
		t.Fatalf("expected prepared run and handle, got prepared=%#v handle=%#v", prepared, handle)
	}

	out, err := handle.Wait()
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if out == nil || len(out.Blocks) != 2 {
		t.Fatalf("expected user and assistant blocks, got %#v", out)
	}
	if out.Blocks[1].Role != turns.RoleAssistant {
		t.Fatalf("expected assistant output block, got %s", out.Blocks[1].Role)
	}
}

func TestRunReturnsCompletedTurn(t *testing.T) {
	eng := &emittingEngine{}
	r := newFakeEngineRunner(t, eng)

	prepared, out, err := r.Run(context.Background(), StartRequest{
		Prompt: "hello",
		Runtime: Runtime{
			InferenceSettings: newTestInferenceSettings(t),
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if prepared == nil || prepared.Session == nil {
		t.Fatalf("expected prepared session, got %#v", prepared)
	}
	if out == nil || len(out.Blocks) != 2 {
		t.Fatalf("expected completed output turn, got %#v", out)
	}
}

func TestRunPublishesEventsToRequestSinks(t *testing.T) {
	eng := &emittingEngine{}
	r := newFakeEngineRunner(t, eng)
	sink := &capturingSink{}

	_, _, err := r.Run(context.Background(), StartRequest{
		Prompt: "hello",
		Runtime: Runtime{
			InferenceSettings: newTestInferenceSettings(t),
		},
		EventSinks: []events.EventSink{sink},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sink.Count() == 0 {
		t.Fatal("expected at least one event to be published to the sink")
	}
}
