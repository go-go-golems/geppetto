package toolloop

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type toolCallingFakeEngine struct {
	calls               atomic.Int64
	sawToolConfig       bool
	seenToolConfig      engine.ToolConfig
	sawToolDefinitions  bool
	seenToolDefinitions engine.ToolDefinitions
}

func (e *toolCallingFakeEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	callNum := e.calls.Add(1)

	if callNum == 1 && t != nil {
		if cfg, ok, err := engine.KeyToolConfig.Get(t.Data); err == nil && ok {
			e.sawToolConfig = true
			e.seenToolConfig = cfg
		}
		if defs, ok, err := engine.KeyToolDefinitions.Get(t.Data); err == nil && ok {
			e.sawToolDefinitions = true
			e.seenToolDefinitions = defs
		}
	}

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

	loopCfg := NewLoopConfig().WithMaxIterations(3)
	toolCfg := tools.DefaultToolConfig()
	loop := New(
		WithEngine(eng),
		WithRegistry(reg),
		WithLoopConfig(loopCfg),
		WithToolConfig(toolCfg),
		WithStepController(sc),
	)
	out, err := loop.RunLoop(ctx, initial)
	if err != nil {
		t.Fatalf("RunLoop: %v", err)
	}
	if eng.calls.Load() < 2 {
		t.Fatalf("expected engine to be called at least twice, got %d", eng.calls.Load())
	}
	if !eng.sawToolConfig {
		t.Fatalf("expected engine to see tool_config on the first inference turn")
	}
	if !eng.seenToolConfig.Enabled {
		t.Fatalf("expected tool_config.enabled to be true on the first inference turn")
	}
	if !eng.sawToolDefinitions {
		t.Fatalf("expected engine to see tool_definitions on the first inference turn")
	}
	if len(eng.seenToolDefinitions) != 1 {
		t.Fatalf("expected one persisted tool definition, got %d", len(eng.seenToolDefinitions))
	}
	if eng.seenToolDefinitions[0].Name != "echo" {
		t.Fatalf("expected persisted tool definition for echo, got %q", eng.seenToolDefinitions[0].Name)
	}
	if eng.seenToolDefinitions[0].Description == "" {
		t.Fatalf("expected persisted tool definition description to be present")
	}
	if eng.seenToolDefinitions[0].Parameters == nil {
		t.Fatalf("expected persisted tool definition parameters to be present")
	}
	if gotType, _ := eng.seenToolDefinitions[0].Parameters["type"].(string); gotType != "object" {
		t.Fatalf("expected persisted tool definition parameters.type to be object, got %q", gotType)
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
