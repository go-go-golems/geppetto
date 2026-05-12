package settings

import (
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

// InputModality represents an input modality a model can accept.
// Known constants are provided for common modalities, but custom provider
// modality strings intentionally round-trip through YAML/JSON.
type InputModality string

const (
	InputModalityText  InputModality = "text"
	InputModalityImage InputModality = "image"
	InputModalityAudio InputModality = "audio"
	InputModalityVideo InputModality = "video"
	InputModalityPDF   InputModality = "pdf"
)

// ModelCost holds model pricing in USD per million tokens.
// A nil *ModelCost means pricing is unknown. A non-nil ModelCost with zero
// values means the model is free or local for the corresponding token buckets.
type ModelCost struct {
	Input      float64 `json:"input" yaml:"input" mapstructure:"input"`
	Output     float64 `json:"output" yaml:"output" mapstructure:"output"`
	CacheRead  float64 `json:"cache_read,omitempty" yaml:"cache_read,omitempty" mapstructure:"cache_read,omitempty"`
	CacheWrite float64 `json:"cache_write,omitempty" yaml:"cache_write,omitempty" mapstructure:"cache_write,omitempty"`
}

// ModelInfo describes static model capabilities and limits loaded from engine
// profiles as part of inference_settings.model_info.
type ModelInfo struct {
	ID                   *string         `json:"id,omitempty" yaml:"id,omitempty" mapstructure:"id,omitempty"`
	Name                 *string         `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name,omitempty"`
	Reasoning            *bool           `json:"reasoning,omitempty" yaml:"reasoning,omitempty" mapstructure:"reasoning,omitempty"`
	Input                []InputModality `json:"input,omitempty" yaml:"input,omitempty" mapstructure:"input,omitempty"`
	ContextWindow        *int            `json:"context_window,omitempty" yaml:"context_window,omitempty" mapstructure:"context_window,omitempty"`
	QualityHighWatermark *int            `json:"quality_high_watermark,omitempty" yaml:"quality_high_watermark,omitempty" mapstructure:"quality_high_watermark,omitempty"`
	MaxOutputTokens      *int            `json:"max_output_tokens,omitempty" yaml:"max_output_tokens,omitempty" mapstructure:"max_output_tokens,omitempty"`
	Cost                 *ModelCost      `json:"cost,omitempty" yaml:"cost,omitempty" mapstructure:"cost,omitempty"`
	Metadata             map[string]any  `json:"metadata,omitempty" yaml:"metadata,omitempty" mapstructure:"metadata,omitempty"`
}

func NewModelInfo() *ModelInfo { return &ModelInfo{} }

func (c *ModelCost) Clone() *ModelCost {
	if c == nil {
		return nil
	}
	out := *c
	return &out
}

func (m *ModelInfo) Clone() *ModelInfo {
	if m == nil {
		return nil
	}
	out := &ModelInfo{
		ID:                   cloneStringPtr(m.ID),
		Name:                 cloneStringPtr(m.Name),
		Reasoning:            cloneBoolPtr(m.Reasoning),
		Input:                append([]InputModality(nil), m.Input...),
		ContextWindow:        cloneIntPtr(m.ContextWindow),
		QualityHighWatermark: cloneIntPtr(m.QualityHighWatermark),
		MaxOutputTokens:      cloneIntPtr(m.MaxOutputTokens),
		Cost:                 m.Cost.Clone(),
		Metadata:             deepCopyModelInfoMap(m.Metadata),
	}
	return out
}

func (m *ModelInfo) Validate() error {
	if m == nil {
		return nil
	}
	if m.ContextWindow != nil && *m.ContextWindow < 0 {
		return fmt.Errorf("model_info.context_window must be non-negative")
	}
	if m.QualityHighWatermark != nil && *m.QualityHighWatermark < 0 {
		return fmt.Errorf("model_info.quality_high_watermark must be non-negative")
	}
	if m.MaxOutputTokens != nil && *m.MaxOutputTokens < 0 {
		return fmt.Errorf("model_info.max_output_tokens must be non-negative")
	}
	if m.ContextWindow != nil && m.QualityHighWatermark != nil && *m.QualityHighWatermark > *m.ContextWindow {
		return fmt.Errorf("model_info.quality_high_watermark must be <= model_info.context_window")
	}
	if m.Cost != nil {
		if m.Cost.Input < 0 || m.Cost.Output < 0 || m.Cost.CacheRead < 0 || m.Cost.CacheWrite < 0 {
			return fmt.Errorf("model_info.cost values must be non-negative")
		}
	}
	return nil
}

func (m *ModelInfo) EffectiveContextLimit() int {
	if m == nil || m.ContextWindow == nil {
		return 0
	}
	if m.QualityHighWatermark != nil && *m.QualityHighWatermark < *m.ContextWindow {
		return *m.QualityHighWatermark
	}
	return *m.ContextWindow
}

func (m *ModelInfo) HardContextLimit() int {
	if m == nil || m.ContextWindow == nil {
		return 0
	}
	return *m.ContextWindow
}

func (m *ModelInfo) ComputeCost(usage *turns.InferenceUsage) *float64 {
	if m == nil || m.Cost == nil || usage == nil {
		return nil
	}

	cacheReadTokens := usage.CacheReadInputTokens
	if cacheReadTokens == 0 {
		cacheReadTokens = usage.CachedTokens
	}
	cacheWriteTokens := usage.CacheCreationInputTokens
	standardInputTokens := usage.InputTokens - cacheReadTokens - cacheWriteTokens
	if standardInputTokens < 0 {
		standardInputTokens = 0
	}

	total := m.Cost.Input * float64(standardInputTokens) / 1_000_000
	total += m.Cost.Output * float64(usage.OutputTokens) / 1_000_000
	total += m.Cost.CacheRead * float64(cacheReadTokens) / 1_000_000
	total += m.Cost.CacheWrite * float64(cacheWriteTokens) / 1_000_000
	return &total
}

func ApplyModelInfoCost(result *turns.InferenceResult, info *ModelInfo) {
	if result == nil || info == nil || result.Usage == nil {
		return
	}
	if cost := info.ComputeCost(result.Usage); cost != nil {
		result.Cost = cost
	}
}

func MergeModelInfo(base, overlay *ModelInfo) *ModelInfo {
	if base == nil {
		return overlay.Clone()
	}
	if overlay == nil {
		return base.Clone()
	}
	out := base.Clone()
	if overlay.ID != nil {
		out.ID = cloneStringPtr(overlay.ID)
	}
	if overlay.Name != nil {
		out.Name = cloneStringPtr(overlay.Name)
	}
	if overlay.Reasoning != nil {
		out.Reasoning = cloneBoolPtr(overlay.Reasoning)
	}
	if overlay.Input != nil {
		out.Input = append([]InputModality(nil), overlay.Input...)
	}
	if overlay.ContextWindow != nil {
		out.ContextWindow = cloneIntPtr(overlay.ContextWindow)
	}
	if overlay.QualityHighWatermark != nil {
		out.QualityHighWatermark = cloneIntPtr(overlay.QualityHighWatermark)
	}
	if overlay.MaxOutputTokens != nil {
		out.MaxOutputTokens = cloneIntPtr(overlay.MaxOutputTokens)
	}
	if overlay.Cost != nil {
		out.Cost = overlay.Cost.Clone()
	}
	if overlay.Metadata != nil {
		out.Metadata = mergeModelInfoMaps(out.Metadata, overlay.Metadata)
	}
	return out
}

func cloneStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func cloneBoolPtr(v *bool) *bool {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func cloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func mergeModelInfoMaps(base, overlay map[string]any) map[string]any {
	out := deepCopyModelInfoMap(base)
	if out == nil {
		out = map[string]any{}
	}
	for k, v := range overlay {
		if existing, ok := out[k]; ok {
			out[k] = mergeModelInfoValue(existing, v)
			continue
		}
		out[k] = deepCopyModelInfoAny(v)
	}
	return out
}

func mergeModelInfoValue(base, overlay any) any {
	baseMap, baseOK := base.(map[string]any)
	overlayMap, overlayOK := overlay.(map[string]any)
	if baseOK && overlayOK {
		return mergeModelInfoMaps(baseMap, overlayMap)
	}
	return deepCopyModelInfoAny(overlay)
}

func deepCopyModelInfoMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = deepCopyModelInfoAny(v)
	}
	return out
}

func deepCopyModelInfoAny(in any) any {
	switch v := in.(type) {
	case map[string]any:
		return deepCopyModelInfoMap(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = deepCopyModelInfoAny(item)
		}
		return out
	default:
		return in
	}
}
