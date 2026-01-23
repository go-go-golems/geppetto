package toolhelpers

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

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

	// If we don't yet have a tool_use for call-1, request it.
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

	// Tool results are present; finalize with assistant output.
	turns.AppendBlock(out, turns.NewAssistantTextBlock("done"))
	return out, nil
}

func TestRunToolCallingLoop_ExecutesSimpleToolAndReturnsFinalTurn(t *testing.T) {
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

	eng := &toolCallingFakeEngine{}
	initial := &turns.Turn{}
	if err := turns.KeyTurnMetaSessionID.Set(&initial.Metadata, "run-1"); err != nil {
		t.Fatalf("KeyTurnMetaSessionID.Set: %v", err)
	}
	turns.AppendBlock(initial, turns.NewUserTextBlock("please echo"))

	cfg := NewToolConfig().WithMaxIterations(3)
	out, err := RunToolCallingLoop(context.Background(), eng, initial, reg, cfg)
	if err != nil {
		t.Fatalf("RunToolCallingLoop: %v", err)
	}
	if eng.calls.Load() < 2 {
		t.Fatalf("expected engine to be called at least twice, got %d", eng.calls.Load())
	}

	// Ensure tool_use block is present and contains the echoed text.
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

	// Ensure we eventually got assistant text indicating completion.
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
}
