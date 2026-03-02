package engine

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// InferenceUsageFromEventUsage converts event usage metadata to canonical turn usage.
func InferenceUsageFromEventUsage(u *events.Usage) *turns.InferenceUsage {
	if u == nil {
		return nil
	}
	return &turns.InferenceUsage{
		InputTokens:              u.InputTokens,
		OutputTokens:             u.OutputTokens,
		CachedTokens:             u.CachedTokens,
		CacheCreationInputTokens: u.CacheCreationInputTokens,
		CacheReadInputTokens:     u.CacheReadInputTokens,
	}
}

// BuildInferenceResultFromEventMetadata maps final event metadata to canonical inference_result.
func BuildInferenceResultFromEventMetadata(metadata events.EventMetadata, provider string, hasToolCalls bool) InferenceResult {
	var maxTokens *int
	if metadata.MaxTokens != nil {
		v := *metadata.MaxTokens
		maxTokens = &v
	}

	ret := InferenceResult{
		Provider:   strings.TrimSpace(provider),
		Model:      strings.TrimSpace(metadata.Model),
		Usage:      InferenceUsageFromEventUsage(metadata.Usage),
		MaxTokens:  maxTokens,
		DurationMs: metadata.DurationMs,
	}
	if metadata.StopReason != nil {
		ret.StopReason = strings.TrimSpace(*metadata.StopReason)
	}
	if len(metadata.Extra) > 0 {
		ret.Extra = make(map[string]any, len(metadata.Extra))
		for k, v := range metadata.Extra {
			ret.Extra[k] = v
		}
	}
	ret.FinishClass = InferFinishClass(ret.StopReason, hasToolCalls)
	ret.Truncated = isTruncatedStopReason(ret.StopReason)
	return ret
}

// PersistInferenceResult stores canonical and legacy metadata fields on the turn.
func PersistInferenceResult(t *turns.Turn, result InferenceResult) error {
	if t == nil {
		return nil
	}
	if err := turns.KeyTurnMetaInferenceResult.Set(&t.Metadata, result); err != nil {
		return err
	}
	return MirrorLegacyInferenceKeys(t, result)
}
