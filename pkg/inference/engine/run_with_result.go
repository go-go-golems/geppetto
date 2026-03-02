package engine

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
)

var ErrEngineNil = errors.New("engine is nil")

type InferenceResult = turns.InferenceResult

type InferenceFinishClass = turns.InferenceFinishClass

const (
	InferenceFinishClassCompleted        = turns.InferenceFinishClassCompleted
	InferenceFinishClassMaxTokens        = turns.InferenceFinishClassMaxTokens
	InferenceFinishClassToolCallsPending = turns.InferenceFinishClassToolCallsPending
	InferenceFinishClassInterrupted      = turns.InferenceFinishClassInterrupted
	InferenceFinishClassError            = turns.InferenceFinishClassError
	InferenceFinishClassUnknown          = turns.InferenceFinishClassUnknown
)

// EngineWithResult is an optional extension for engines that can return structured
// inference metadata directly.
type EngineWithResult interface {
	RunInferenceWithResult(ctx context.Context, t *turns.Turn) (*turns.Turn, *InferenceResult, error)
}

// RunInferenceWithResult runs inference and returns a normalized canonical inference result.
//
// Backward compatibility path:
// - Prefer EngineWithResult when implemented by the engine.
// - Otherwise call RunInference and derive result from turn metadata.
func RunInferenceWithResult(ctx context.Context, eng Engine, t *turns.Turn) (*turns.Turn, *InferenceResult, error) {
	if eng == nil {
		return nil, nil, ErrEngineNil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if withResult, ok := eng.(EngineWithResult); ok {
		out, result, err := withResult.RunInferenceWithResult(ctx, t)
		if out == nil {
			out = t
		}
		if out == nil {
			out = &turns.Turn{}
		}
		if err != nil {
			return out, result, err
		}
		if result == nil {
			synth := SynthesizeInferenceResult(out)
			result = &synth
		} else {
			normalizeInferenceResult(result, out)
		}
		if setErr := turns.KeyTurnMetaInferenceResult.Set(&out.Metadata, *result); setErr != nil {
			return out, result, errors.Wrap(setErr, "set canonical inference_result")
		}
		if setErr := MirrorLegacyInferenceKeys(out, *result); setErr != nil {
			return out, result, setErr
		}
		return out, result, nil
	}

	out, err := eng.RunInference(ctx, t)
	if out == nil {
		out = t
	}
	if out == nil {
		out = &turns.Turn{}
	}
	if err != nil {
		return out, nil, err
	}

	result, ok, getErr := ExtractInferenceResult(out)
	if getErr != nil {
		return out, nil, getErr
	}
	if !ok {
		result = SynthesizeInferenceResult(out)
	} else {
		normalizeInferenceResult(&result, out)
	}

	if setErr := turns.KeyTurnMetaInferenceResult.Set(&out.Metadata, result); setErr != nil {
		return out, nil, errors.Wrap(setErr, "set canonical inference_result")
	}
	if setErr := MirrorLegacyInferenceKeys(out, result); setErr != nil {
		return out, nil, setErr
	}
	return out, &result, nil
}

// ExtractInferenceResult returns canonical inference_result when present.
func ExtractInferenceResult(t *turns.Turn) (InferenceResult, bool, error) {
	if t == nil {
		return InferenceResult{}, false, nil
	}
	res, ok, err := turns.KeyTurnMetaInferenceResult.Get(t.Metadata)
	if err != nil {
		return InferenceResult{}, false, errors.Wrap(err, "get turn inference_result")
	}
	return res, ok, nil
}

// SynthesizeInferenceResult derives an inference result from legacy scalar metadata.
func SynthesizeInferenceResult(t *turns.Turn) InferenceResult {
	if t == nil {
		return InferenceResult{FinishClass: InferenceFinishClassUnknown}
	}
	result := InferenceResult{}
	if v, ok, err := turns.KeyTurnMetaProvider.Get(t.Metadata); err == nil && ok {
		result.Provider = strings.TrimSpace(v)
	}
	if v, ok, err := turns.KeyTurnMetaModel.Get(t.Metadata); err == nil && ok {
		result.Model = strings.TrimSpace(v)
	}
	if v, ok, err := turns.KeyTurnMetaStopReason.Get(t.Metadata); err == nil && ok {
		result.StopReason = strings.TrimSpace(v)
	}
	if v, ok, err := turns.KeyTurnMetaUsage.Get(t.Metadata); err == nil && ok {
		result.Usage = decodeLegacyUsage(v)
	}

	hasToolCalls := false
	for _, b := range t.Blocks {
		if b.Kind == turns.BlockKindToolCall {
			hasToolCalls = true
			break
		}
	}
	result.FinishClass = InferFinishClass(result.StopReason, hasToolCalls)
	result.Truncated = isTruncatedStopReason(result.StopReason)
	return result
}

// MirrorLegacyInferenceKeys keeps legacy scalar metadata in sync during migration.
func MirrorLegacyInferenceKeys(t *turns.Turn, r InferenceResult) error {
	if t == nil {
		return nil
	}
	if strings.TrimSpace(r.Provider) != "" {
		if err := turns.KeyTurnMetaProvider.Set(&t.Metadata, strings.TrimSpace(r.Provider)); err != nil {
			return errors.Wrap(err, "set turn provider")
		}
	}
	if strings.TrimSpace(r.Model) != "" {
		if err := turns.KeyTurnMetaModel.Set(&t.Metadata, strings.TrimSpace(r.Model)); err != nil {
			return errors.Wrap(err, "set turn model")
		}
	}
	if strings.TrimSpace(r.StopReason) != "" {
		if err := turns.KeyTurnMetaStopReason.Set(&t.Metadata, strings.TrimSpace(r.StopReason)); err != nil {
			return errors.Wrap(err, "set turn stop reason")
		}
	}
	if r.Usage != nil {
		if err := turns.KeyTurnMetaUsage.Set(&t.Metadata, r.Usage); err != nil {
			return errors.Wrap(err, "set turn usage")
		}
	}
	return nil
}

func InferFinishClass(stopReason string, hasToolCalls bool) InferenceFinishClass {
	sr := strings.ToLower(strings.TrimSpace(stopReason))
	if hasToolCalls {
		return InferenceFinishClassToolCallsPending
	}
	switch sr {
	case "max_tokens", "max_output_tokens", "length", "model_length":
		return InferenceFinishClassMaxTokens
	case "", "unknown":
		return InferenceFinishClassUnknown
	case "interrupted", "cancelled", "canceled":
		return InferenceFinishClassInterrupted
	case "error", "failed":
		return InferenceFinishClassError
	default:
		return InferenceFinishClassCompleted
	}
}

func normalizeInferenceResult(result *InferenceResult, t *turns.Turn) {
	if result == nil {
		return
	}
	if strings.TrimSpace(result.StopReason) == "" {
		if t != nil {
			if reason, ok, err := turns.KeyTurnMetaStopReason.Get(t.Metadata); err == nil && ok {
				result.StopReason = strings.TrimSpace(reason)
			}
		}
	}

	hasToolCalls := false
	if t != nil {
		for _, b := range t.Blocks {
			if b.Kind == turns.BlockKindToolCall {
				hasToolCalls = true
				break
			}
		}
	}
	if result.FinishClass == "" {
		result.FinishClass = InferFinishClass(result.StopReason, hasToolCalls)
	}
	if !result.Truncated {
		result.Truncated = isTruncatedStopReason(result.StopReason)
	}
}

func isTruncatedStopReason(stopReason string) bool {
	switch strings.ToLower(strings.TrimSpace(stopReason)) {
	case "max_tokens", "max_output_tokens", "length", "model_length":
		return true
	default:
		return false
	}
}

func decodeLegacyUsage(v any) *turns.InferenceUsage {
	if v == nil {
		return nil
	}
	if usage, ok := v.(*turns.InferenceUsage); ok {
		return usage
	}
	if usage, ok := v.(turns.InferenceUsage); ok {
		u := usage
		return &u
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var usage turns.InferenceUsage
	if err := json.Unmarshal(b, &usage); err != nil {
		return nil
	}
	return &usage
}
