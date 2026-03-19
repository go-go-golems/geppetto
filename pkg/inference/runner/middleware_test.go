package runner

import (
	"context"
	"errors"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	gepmiddleware "github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"testing"
)

var (
	testTurnMetaDirect        = turns.TurnMetaK[string]("runner_test", "direct", 1)
	testTurnMetaResolvedLabel = turns.TurnMetaK[string]("runner_test", "resolved_label", 1)
)

type captureEngine struct {
	lastInput *turns.Turn
}

var _ engine.Engine = (*captureEngine)(nil)

func (e *captureEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	_ = ctx
	if t != nil {
		e.lastInput = t.Clone()
		return t.Clone(), nil
	}
	return &turns.Turn{}, nil
}

type markerDefinition struct{}

func (d markerDefinition) Name() string {
	return "mark"
}

func (d markerDefinition) ConfigJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"label": map[string]any{"type": "string"},
		},
	}
}

func (d markerDefinition) Build(ctx context.Context, deps middlewarecfg.BuildDeps, cfg any) (gepmiddleware.Middleware, error) {
	_ = ctx
	label := ""
	if m, ok := cfg.(map[string]any); ok {
		if s, ok := m["label"].(string); ok {
			label = s
		}
	}
	suffix, _ := deps.Get("suffix")
	return func(next gepmiddleware.HandlerFunc) gepmiddleware.HandlerFunc {
		return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
			_ = testTurnMetaResolvedLabel.Set(&t.Metadata, label+":"+suffix.(string))
			return next(ctx, t)
		}
	}, nil
}

func TestBuildEngineFromBaseAppliesDirectMiddlewaresAndSystemPrompt(t *testing.T) {
	base := &captureEngine{}
	r := New()

	direct := func(next gepmiddleware.HandlerFunc) gepmiddleware.HandlerFunc {
		return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
			_ = testTurnMetaDirect.Set(&t.Metadata, "yes")
			return next(ctx, t)
		}
	}

	eng, mws, err := r.buildEngineFromBase(context.Background(), base, Runtime{
		SystemPrompt: "You are a concise helper.",
		Middlewares:  []gepmiddleware.Middleware{direct},
	})
	if err != nil {
		t.Fatalf("buildEngineFromBase: %v", err)
	}
	if len(mws) != 3 {
		t.Fatalf("expected 3 middlewares, got %d", len(mws))
	}

	turn := &turns.Turn{}
	turns.AppendBlock(turn, turns.NewUserTextBlock("hello"))
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	if base.lastInput == nil {
		t.Fatal("expected base engine to capture input")
	}
	gotDirect, ok, err := testTurnMetaDirect.Get(base.lastInput.Metadata)
	if err != nil {
		t.Fatalf("testTurnMetaDirect.Get: %v", err)
	}
	if !ok || gotDirect != "yes" {
		t.Fatalf("expected direct middleware metadata, got ok=%v value=%q", ok, gotDirect)
	}
	if len(base.lastInput.Blocks) == 0 || base.lastInput.Blocks[0].Kind != turns.BlockKindSystem {
		t.Fatalf("expected first block to be a system block, got %#v", base.lastInput.Blocks)
	}
	text, _ := base.lastInput.Blocks[0].Payload[turns.PayloadKeyText].(string)
	if text != "You are a concise helper." {
		t.Fatalf("unexpected system prompt block text: %q", text)
	}
}

func TestResolveMiddlewaresBuildsChainFromMiddlewareUses(t *testing.T) {
	registry := middlewarecfg.NewInMemoryDefinitionRegistry()
	if err := registry.RegisterDefinition(markerDefinition{}); err != nil {
		t.Fatalf("RegisterDefinition: %v", err)
	}

	r := New(
		WithMiddlewareDefinitions(registry),
		WithMiddlewareBuildDeps(middlewarecfg.BuildDeps{
			Values: map[string]any{"suffix": "deps"},
		}),
	)
	base := &captureEngine{}

	eng, mws, err := r.buildEngineFromBase(context.Background(), base, Runtime{
		MiddlewareUses: []middlewarecfg.Use{
			{
				Name:   "mark",
				Config: map[string]any{"label": "runtime"},
			},
		},
	})
	if err != nil {
		t.Fatalf("buildEngineFromBase: %v", err)
	}
	if len(mws) != 2 {
		t.Fatalf("expected 2 middlewares, got %d", len(mws))
	}

	turn := &turns.Turn{}
	turns.AppendBlock(turn, turns.NewUserTextBlock("hello"))
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	gotLabel, ok, err := testTurnMetaResolvedLabel.Get(base.lastInput.Metadata)
	if err != nil {
		t.Fatalf("testTurnMetaResolvedLabel.Get: %v", err)
	}
	if !ok || gotLabel != "runtime:deps" {
		t.Fatalf("expected resolved middleware metadata, got ok=%v value=%q", ok, gotLabel)
	}
}

func TestBuildEngineRequiresInferenceSettings(t *testing.T) {
	r := New()
	_, _, err := r.buildEngine(context.Background(), Runtime{})
	if !errors.Is(err, ErrRuntimeInferenceSettingsNil) {
		t.Fatalf("expected ErrRuntimeInferenceSettingsNil, got %v", err)
	}
}
