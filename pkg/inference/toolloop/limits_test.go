package toolloop

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	gepsession "github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/toolblocks"
)

// alwaysToolEngine calls the echo tool on every inference. On a call with tool
// choice "none" it answers with text, unless ignoreNone is set (an engine that
// ignores tool choice) or failFinal is set (the final call fails).
type alwaysToolEngine struct {
	calls      atomic.Int64
	ignoreNone bool
	failFinal  error

	mu          sync.Mutex
	choices     []engine.ToolChoice
	callIndexes []int
}

func (e *alwaysToolEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	n := e.calls.Add(1)
	cfg, _, _ := engine.KeyToolConfig.Get(t.Data)
	idx, _ := gepsession.ProviderCallIndexFromContext(ctx)
	e.mu.Lock()
	e.choices = append(e.choices, cfg.ToolChoice)
	e.callIndexes = append(e.callIndexes, idx)
	e.mu.Unlock()
	if cfg.ToolChoice == engine.ToolChoiceNone && !e.ignoreNone {
		if e.failFinal != nil {
			return t, e.failFinal
		}
		turns.AppendBlock(t, turns.NewAssistantTextBlock("final answer"))
		return t, nil
	}
	turns.AppendBlock(t, turns.NewToolCallBlock(fmt.Sprintf("call-%d", n), "echo", map[string]any{"text": "again"}))
	return t, nil
}

func echoRegistry(t *testing.T, executed *atomic.Int64) tools.ToolRegistry {
	t.Helper()
	reg := tools.NewInMemoryToolRegistry()
	type echoIn struct {
		Text string `json:"text"`
	}
	echoTool, err := tools.NewToolFromFunc("echo", "Echo back the provided text", func(in echoIn) (map[string]any, error) {
		executed.Add(1)
		return map[string]any{"echo": in.Text}, nil
	})
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := reg.RegisterTool("echo", *echoTool); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}
	return reg
}

func userTurn() *turns.Turn {
	initial := &turns.Turn{}
	turns.AppendBlock(initial, turns.NewUserTextBlock("keep echoing"))
	return initial
}

func lastText(t *turns.Turn) string {
	for i := len(t.Blocks) - 1; i >= 0; i-- {
		if t.Blocks[i].Kind == turns.BlockKindLLMText {
			s, _ := t.Blocks[i].Payload[turns.PayloadKeyText].(string)
			return s
		}
	}
	return ""
}

func TestLoop_MaxIterationsErrorIsTyped(t *testing.T) {
	t.Parallel()
	var executed atomic.Int64
	eng := &alwaysToolEngine{}
	loop := New(WithEngine(eng), WithRegistry(echoRegistry(t, &executed)), WithLoopConfig(NewLoopConfig().WithMaxIterations(2)))

	out, err := loop.RunLoop(context.Background(), userTurn())
	if !errors.Is(err, ErrMaxIterations) {
		t.Fatalf("want ErrMaxIterations, got %v", err)
	}
	var lim *MaxIterationsError
	if !errors.As(err, &lim) || lim.MaxIterations != 2 || lim.Answered {
		t.Fatalf("want *MaxIterationsError{2, false}, got %#v", lim)
	}
	if got := err.Error(); got != "max iterations (2) reached" {
		t.Fatalf("error text changed: %q", got)
	}
	if out == nil || eng.calls.Load() != 2 || executed.Load() != 2 {
		t.Fatalf("want the turn so far after 2 calls and 2 tool runs, got out=%v calls=%d executed=%d", out != nil, eng.calls.Load(), executed.Load())
	}
}

func TestLoop_FinalAnswerAtTheLimit(t *testing.T) {
	t.Parallel()
	var executed atomic.Int64
	eng := &alwaysToolEngine{}
	var promptID string
	cfg := NewLoopConfig().WithMaxIterations(2).WithFinalAnswer(FinalAnswer{Prompt: func(n int) turns.Block {
		b := turns.NewUserTextBlock(fmt.Sprintf("you used %d rounds; answer now", n))
		promptID = b.ID
		return b
	}})
	loop := New(WithEngine(eng), WithRegistry(echoRegistry(t, &executed)), WithLoopConfig(cfg))

	out, err := loop.RunLoop(context.Background(), userTurn())
	var lim *MaxIterationsError
	if !errors.As(err, &lim) || !lim.Answered {
		t.Fatalf("want an answered *MaxIterationsError, got %v", err)
	}
	if lastText(out) != "final answer" {
		t.Fatalf("want the final answer in the turn, got %q", lastText(out))
	}
	if eng.calls.Load() != 3 || executed.Load() != 2 {
		t.Fatalf("want 3 calls (2 + final) and 2 tool runs, got %d and %d", eng.calls.Load(), executed.Load())
	}
	if eng.choices[2] != engine.ToolChoiceNone || eng.callIndexes[2] != 2 {
		t.Fatalf("final call: want tool choice none at provider index 2, got %q at %d", eng.choices[2], eng.callIndexes[2])
	}
	found := false
	for _, b := range out.Blocks {
		if b.ID == promptID && promptID != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the caller's prompt block (id %q) is in the turn", promptID)
	}
	if c, _, _ := engine.KeyToolConfig.Get(out.Data); c.ToolChoice == engine.ToolChoiceNone {
		t.Fatalf("the loop's tool config is restored after the final call, got %q", c.ToolChoice)
	}
}

func TestLoop_FinalAnswerRefusesToolCalls(t *testing.T) {
	t.Parallel()
	var executed atomic.Int64
	eng := &alwaysToolEngine{ignoreNone: true}
	cfg := NewLoopConfig().WithMaxIterations(1).WithFinalAnswer(FinalAnswer{})
	loop := New(WithEngine(eng), WithRegistry(echoRegistry(t, &executed)), WithLoopConfig(cfg))

	out, err := loop.RunLoop(context.Background(), userTurn())
	var lim *MaxIterationsError
	if !errors.As(err, &lim) || !lim.Answered {
		t.Fatalf("want an answered *MaxIterationsError, got %v", err)
	}
	if executed.Load() != 1 {
		t.Fatalf("the final response's tool call must not run, got %d runs", executed.Load())
	}
	if pending := toolblocks.ExtractPendingToolCalls(out); len(pending) != 0 {
		t.Fatalf("every call is answered, got %d pending", len(pending))
	}
	last := out.Blocks[len(out.Blocks)-1]
	if e, _ := last.Payload[turns.PayloadKeyError].(string); last.Kind != turns.BlockKindToolUse || e != DefaultRefusedToolResult {
		t.Fatalf("want a refused tool result last, got kind %v error %q", last.Kind, e)
	}
}

func TestLoop_FinalAnswerFailureKeepsBothErrors(t *testing.T) {
	t.Parallel()
	var executed atomic.Int64
	providerDown := errors.New("provider: 529 overloaded")
	eng := &alwaysToolEngine{failFinal: providerDown}
	cfg := NewLoopConfig().WithMaxIterations(1).WithFinalAnswer(FinalAnswer{})
	loop := New(WithEngine(eng), WithRegistry(echoRegistry(t, &executed)), WithLoopConfig(cfg))

	out, err := loop.RunLoop(context.Background(), userTurn())
	if !errors.Is(err, ErrMaxIterations) || !errors.Is(err, providerDown) {
		t.Fatalf("want both ErrMaxIterations and the provider error, got %v", err)
	}
	var lim *MaxIterationsError
	if !errors.As(err, &lim) || lim.Answered {
		t.Fatalf("want an unanswered *MaxIterationsError, got %#v", lim)
	}
	if out == nil {
		t.Fatalf("want the turn so far")
	}
	if c, _, _ := engine.KeyToolConfig.Get(out.Data); c.ToolChoice == engine.ToolChoiceNone {
		t.Fatalf("the loop's tool config is restored after a failed final call")
	}
}

func TestLoop_FinalAnswerSkippedWhenCancelled(t *testing.T) {
	t.Parallel()
	var executed atomic.Int64
	ctx, cancel := context.WithCancel(context.Background())
	eng := &alwaysToolEngine{}
	cfg := NewLoopConfig().WithMaxIterations(1).WithFinalAnswer(FinalAnswer{})
	// cancel while the only round's tool runs, so the loop reaches the limit cancelled
	reg := tools.NewInMemoryToolRegistry()
	type echoIn struct {
		Text string `json:"text"`
	}
	tool, err := tools.NewToolFromFunc("echo", "Echo", func(in echoIn) (map[string]any, error) {
		executed.Add(1)
		cancel()
		return map[string]any{"echo": in.Text}, nil
	})
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := reg.RegisterTool("echo", *tool); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}
	loop := New(WithEngine(eng), WithRegistry(reg), WithLoopConfig(cfg))

	_, err = loop.RunLoop(ctx, userTurn())
	if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrMaxIterations) {
		t.Fatalf("want cancellation at the limit, got %v", err)
	}
	if eng.calls.Load() != 1 {
		t.Fatalf("no final call after cancellation, got %d calls", eng.calls.Load())
	}
}

func TestLoop_RoundGuardStopsBeforeTheNextCall(t *testing.T) {
	t.Parallel()
	var executed atomic.Int64
	overBudget := errors.New("over budget")
	eng := &alwaysToolEngine{}
	var rounds []int
	guard := func(_ context.Context, tt *turns.Turn, round int) error {
		rounds = append(rounds, round)
		if tt == nil {
			return errors.New("guard got no turn")
		}
		if round == 2 {
			return overBudget
		}
		return nil
	}
	cfg := NewLoopConfig().WithMaxIterations(5).WithFinalAnswer(FinalAnswer{})
	loop := New(WithEngine(eng), WithRegistry(echoRegistry(t, &executed)), WithLoopConfig(cfg), WithRoundGuard(guard))

	out, err := loop.RunLoop(context.Background(), userTurn())
	if !errors.Is(err, overBudget) || errors.Is(err, ErrMaxIterations) {
		t.Fatalf("want exactly the guard's error, got %v", err)
	}
	if eng.calls.Load() != 2 || out == nil {
		t.Fatalf("want 2 calls and the turn so far, got %d calls (out=%v)", eng.calls.Load(), out != nil)
	}
	if fmt.Sprint(rounds) != "[0 1 2]" {
		t.Fatalf("guard rounds: %v", rounds)
	}
}

func TestLoop_RoundGuardAlsoGuardsTheFinalAnswer(t *testing.T) {
	t.Parallel()
	var executed atomic.Int64
	overBudget := errors.New("over budget")
	eng := &alwaysToolEngine{}
	guard := func(_ context.Context, _ *turns.Turn, round int) error {
		if round == 2 {
			return overBudget
		}
		return nil
	}
	cfg := NewLoopConfig().WithMaxIterations(2).WithFinalAnswer(FinalAnswer{})
	loop := New(WithEngine(eng), WithRegistry(echoRegistry(t, &executed)), WithLoopConfig(cfg), WithRoundGuard(guard))

	_, err := loop.RunLoop(context.Background(), userTurn())
	if !errors.Is(err, overBudget) || !errors.Is(err, ErrMaxIterations) {
		t.Fatalf("want the guard's error at the limit, got %v", err)
	}
	if eng.calls.Load() != 2 {
		t.Fatalf("the final call is guarded too, got %d calls", eng.calls.Load())
	}
}
