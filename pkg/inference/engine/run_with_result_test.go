package engine

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

type legacyEngine struct{}

func (legacyEngine) RunInference(_ context.Context, t *turns.Turn) (*turns.Turn, error) {
	if t == nil {
		t = &turns.Turn{}
	}
	turns.AppendBlock(t, turns.NewAssistantTextBlock("ok"))
	_ = turns.KeyTurnMetaProvider.Set(&t.Metadata, "legacy")
	_ = turns.KeyTurnMetaModel.Set(&t.Metadata, "legacy-model")
	_ = turns.KeyTurnMetaStopReason.Set(&t.Metadata, "max_tokens")
	_ = turns.KeyTurnMetaUsage.Set(&t.Metadata, turns.InferenceUsage{InputTokens: 12, OutputTokens: 4})
	return t, nil
}

type directResultEngine struct{}

func (directResultEngine) RunInference(_ context.Context, t *turns.Turn) (*turns.Turn, error) {
	return t, nil
}

func (directResultEngine) RunInferenceWithResult(_ context.Context, t *turns.Turn) (*turns.Turn, *InferenceResult, error) {
	if t == nil {
		t = &turns.Turn{}
	}
	res := &InferenceResult{
		Provider:   "direct",
		Model:      "direct-model",
		StopReason: "end_turn",
	}
	return t, res, nil
}

func TestRunInferenceWithResult_LegacyEngineSynthesizesCanonicalResult(t *testing.T) {
	turn := &turns.Turn{}
	out, res, err := RunInferenceWithResult(context.Background(), legacyEngine{}, turn)
	if err != nil {
		t.Fatalf("RunInferenceWithResult: %v", err)
	}
	if out == nil || res == nil {
		t.Fatalf("expected output and result")
	}
	if res.Provider != "legacy" {
		t.Fatalf("expected provider legacy, got %q", res.Provider)
	}
	if res.StopReason != "max_tokens" {
		t.Fatalf("expected stop_reason=max_tokens, got %q", res.StopReason)
	}
	if !res.Truncated {
		t.Fatalf("expected truncated=true")
	}
	if res.FinishClass != InferenceFinishClassMaxTokens {
		t.Fatalf("expected finish_class=%q, got %q", InferenceFinishClassMaxTokens, res.FinishClass)
	}
	if res.Usage == nil || res.Usage.InputTokens != 12 || res.Usage.OutputTokens != 4 {
		t.Fatalf("expected usage 12/4, got %+v", res.Usage)
	}

	stored, ok, err := turns.KeyTurnMetaInferenceResult.Get(out.Metadata)
	if err != nil {
		t.Fatalf("get canonical inference_result: %v", err)
	}
	if !ok {
		t.Fatalf("expected canonical inference_result key")
	}
	if stored.StopReason != "max_tokens" {
		t.Fatalf("expected stored stop_reason=max_tokens, got %q", stored.StopReason)
	}
}

func TestRunInferenceWithResult_DirectEngineResultIsNormalizedAndMirrored(t *testing.T) {
	out, res, err := RunInferenceWithResult(context.Background(), directResultEngine{}, &turns.Turn{})
	if err != nil {
		t.Fatalf("RunInferenceWithResult: %v", err)
	}
	if out == nil || res == nil {
		t.Fatalf("expected output and result")
	}
	if res.FinishClass != InferenceFinishClassCompleted {
		t.Fatalf("expected completed finish class, got %q", res.FinishClass)
	}
	if res.Truncated {
		t.Fatalf("expected truncated=false")
	}

	storedStopReason, ok, err := turns.KeyTurnMetaStopReason.Get(out.Metadata)
	if err != nil {
		t.Fatalf("get stop_reason: %v", err)
	}
	if !ok || storedStopReason != "end_turn" {
		t.Fatalf("expected mirrored stop_reason=end_turn, got %q (ok=%v)", storedStopReason, ok)
	}

	storedProvider, ok, err := turns.KeyTurnMetaProvider.Get(out.Metadata)
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if !ok || storedProvider != "direct" {
		t.Fatalf("expected mirrored provider=direct, got %q (ok=%v)", storedProvider, ok)
	}
}

func TestSynthesizeInferenceResult_PrefersToolCallsPending(t *testing.T) {
	turn := &turns.Turn{}
	turns.AppendBlock(turn, turns.NewToolCallBlock("call-1", "tool", map[string]any{"x": 1}))
	_ = turns.KeyTurnMetaStopReason.Set(&turn.Metadata, "max_tokens")

	res := SynthesizeInferenceResult(turn)
	if res.FinishClass != InferenceFinishClassToolCallsPending {
		t.Fatalf("expected finish class tool_calls_pending, got %q", res.FinishClass)
	}
	if !res.Truncated {
		t.Fatalf("expected truncated=true for max_tokens")
	}
}
