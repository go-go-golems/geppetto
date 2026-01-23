package toolloop

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type toolCallingFakeEngine struct {
	calls atomic.Int64
}

func (e *toolCallingFakeEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	e.calls.Add(1)

	out := &turns.Turn{}
	if t != nil {
		out = t.Clone()
	}

	hasUse := false
	for _, b := range out.Blocks {
		if b.Kind == turns.BlockKindToolUse {
			if id, ok := b.Payload[turns.PayloadKeyID].(string); ok && id == "call-1" {
				hasUse = true
				break
			}
		}
	}
	if !hasUse {
		turns.AppendBlock(out, turns.NewToolCallBlock("call-1", "echo", map[string]any{"text": "hello"}))
		return out, nil
	}

	turns.AppendBlock(out, turns.NewAssistantTextBlock("done"))
	return out, nil
}

type capturingSink struct {
	mu      sync.Mutex
	events  []events.Event
	onPause func(pauseID string)
}

func (s *capturingSink) PublishEvent(e events.Event) error {
	s.mu.Lock()
	s.events = append(s.events, e)
	onPause := s.onPause
	s.mu.Unlock()
	if dp, ok := e.(*events.EventDebuggerPause); ok && onPause != nil {
		onPause(dp.PauseID)
	}
	return nil
}

func TestLoop_ExecutesToolsAndEmitsPauseEventsWhenEnabled(t *testing.T) {
	t.Parallel()

	reg := tools.NewInMemoryToolRegistry()
	type echoIn struct {
		Text string `json:"text"`
	}
	echoTool, err := tools.NewToolFromFunc("echo", "Echo back the provided text", func(in echoIn) (map[string]any, error) {
		return map[string]any{"echo": in.Text}, nil
	})
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := reg.RegisterTool("echo", *echoTool); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	sc := NewStepController()
	sc.Enable(StepScope{SessionID: "run-1"})

	sink := &capturingSink{
		onPause: func(pauseID string) {
			_, _ = sc.Continue(pauseID)
		},
	}

	ctx := events.WithEventSinks(context.Background(), sink)

	eng := &toolCallingFakeEngine{}
	initial := &turns.Turn{}
	_ = turns.KeyTurnMetaSessionID.Set(&initial.Metadata, "run-1")
	turns.AppendBlock(initial, turns.NewUserTextBlock("please echo"))

	cfg := NewToolConfig().WithMaxIterations(3)
	loop := New(
		WithEngine(eng),
		WithRegistry(reg),
		WithConfig(cfg),
		WithStepController(sc),
	)
	out, err := loop.RunLoop(ctx, initial)
	if err != nil {
		t.Fatalf("RunLoop: %v", err)
	}
	if eng.calls.Load() < 2 {
		t.Fatalf("expected engine to be called at least twice, got %d", eng.calls.Load())
	}

	foundUse := false
	foundResult := ""
	for _, b := range out.Blocks {
		if b.Kind != turns.BlockKindToolUse {
			continue
		}
		if id, ok := b.Payload[turns.PayloadKeyID].(string); !ok || id != "call-1" {
			continue
		}
		foundUse = true
		if s, ok := b.Payload[turns.PayloadKeyResult].(string); ok {
			foundResult = s
		}
	}
	if !foundUse {
		t.Fatalf("expected tool_use block for call-1")
	}
	if !strings.Contains(foundResult, "hello") {
		t.Fatalf("expected tool_use result to contain 'hello', got %q", foundResult)
	}

	foundAssistant := false
	for _, b := range out.Blocks {
		if b.Kind == turns.BlockKindLLMText {
			if s, ok := b.Payload[turns.PayloadKeyText].(string); ok && s == "done" {
				foundAssistant = true
				break
			}
		}
	}
	if !foundAssistant {
		t.Fatalf("expected final assistant text block")
	}

	sink.mu.Lock()
	defer sink.mu.Unlock()
	pauseCount := 0
	for _, e := range sink.events {
		if _, ok := e.(*events.EventDebuggerPause); ok {
			pauseCount++
		}
	}
	if pauseCount < 1 {
		t.Fatalf("expected at least one debugger.pause event, got %d", pauseCount)
	}
}
