package engineprofiles

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"gopkg.in/yaml.v3"
)

func cloneInferenceSettings(in *aistepssettings.InferenceSettings) *aistepssettings.InferenceSettings {
	if in == nil {
		return nil
	}
	return in.Clone()
}

// MergeInferenceSettings overlays engine profile inference settings onto a base
// settings object using the same merge semantics used for engine profile stack
// resolution. Overlay wins for conflicting scalar values while nested maps merge
// recursively, except that credential key/endpoint pairs are atomic.
func MergeInferenceSettings(base *aistepssettings.InferenceSettings, overlay *aistepssettings.InferenceSettings) (*aistepssettings.InferenceSettings, error) {
	return mergeInferenceSettings(base, overlay)
}

func mergeInferenceSettings(base *aistepssettings.InferenceSettings, overlay *aistepssettings.InferenceSettings) (*aistepssettings.InferenceSettings, error) {
	if base == nil {
		return cloneInferenceSettings(overlay), nil
	}
	if overlay == nil {
		return cloneInferenceSettings(base), nil
	}

	baseMap, err := inferenceSettingsToMap(base)
	if err != nil {
		return nil, err
	}
	overlayMap, err := inferenceSettingsToMap(overlay)
	if err != nil {
		return nil, err
	}
	merged := mergeInferenceSettingsMap(baseMap, overlayMap)
	mergedMap, ok := merged.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("merged inference settings must resolve to an object")
	}
	result, err := inferenceSettingsFromMap(mergedMap)
	if err != nil {
		return nil, err
	}
	if result.API != nil && overlay.API != nil {
		prior := aistepssettings.CredentialSlots{}
		if base.API != nil {
			prior = aistepssettings.CredentialSlots{APIKeys: base.API.APIKeys, BaseURLs: base.API.BaseUrls}
		}
		pairs, err := aistepssettings.OverlayCredentialSlots(prior, aistepssettings.CredentialSlots{APIKeys: overlay.API.APIKeys, BaseURLs: overlay.API.BaseUrls})
		if err != nil {
			return nil, err
		}
		result.API.APIKeys, result.API.BaseUrls = pairs.APIKeys, pairs.BaseURLs
	}
	if result.Embeddings != nil && overlay.Embeddings != nil {
		prior := aistepssettings.CredentialSlots{}
		if base.Embeddings != nil {
			prior = aistepssettings.CredentialSlots{APIKeys: base.Embeddings.APIKeys, BaseURLs: base.Embeddings.BaseURLs}
		}
		pairs, err := aistepssettings.OverlayCredentialSlots(prior, aistepssettings.CredentialSlots{APIKeys: overlay.Embeddings.APIKeys, BaseURLs: overlay.Embeddings.BaseURLs})
		if err != nil {
			return nil, err
		}
		result.Embeddings.APIKeys, result.Embeddings.BaseURLs = pairs.APIKeys, pairs.BaseURLs
	}
	return result, nil
}

func mergeInferenceSettingsMap(base map[string]any, overlay map[string]any) any {
	ret, _ := mergeExtensionValue(base, overlay).(map[string]any)
	if ret == nil {
		ret = map[string]any{}
	}
	baseModelInfo, baseHasModelInfo := base["model_info"].(map[string]any)
	overlayModelInfo, overlayHasModelInfo := overlay["model_info"].(map[string]any)
	if baseHasModelInfo && overlayHasModelInfo {
		ret["model_info"] = mergeModelInfoMapForInferenceSettings(baseModelInfo, overlayModelInfo)
	}
	return ret
}

func mergeModelInfoMapForInferenceSettings(base map[string]any, overlay map[string]any) map[string]any {
	merged, _ := mergeExtensionValue(base, overlay).(map[string]any)
	if merged == nil {
		merged = map[string]any{}
	}
	// Cost rates use value types, so a cost object should be replaced as a unit
	// instead of recursively preserving omitted base rates.
	if overlayCost, ok := overlay["cost"]; ok {
		merged["cost"] = deepCopyAny(overlayCost)
	}
	return merged
}

func inferenceSettingsToMap(in *aistepssettings.InferenceSettings) (map[string]any, error) {
	if in == nil {
		return nil, nil
	}
	b, err := yaml.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("marshal inference settings: %w", err)
	}
	if len(b) == 0 {
		return nil, nil
	}
	var out map[string]any
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("decode inference settings as map: %w", err)
	}
	normalizeInferenceSettingsMap(out)
	return out, nil
}

func inferenceSettingsFromMap(in map[string]any) (*aistepssettings.InferenceSettings, error) {
	if len(in) == 0 {
		return nil, nil
	}
	b, err := yaml.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("marshal merged inference settings: %w", err)
	}
	var out aistepssettings.InferenceSettings
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("decode merged inference settings: %w", err)
	}
	return &out, nil
}

func normalizeInferenceSettingsMap(in map[string]any) {
	if len(in) == 0 {
		return
	}
	client, ok := in["client"].(map[string]any)
	if !ok {
		return
	}
	if timeoutSeconds, ok := client["timeout_second"]; ok {
		client["timeout_second"] = coerceTimeoutSeconds(timeoutSeconds)
		delete(client, "timeout")
		return
	}
	if timeoutRaw, ok := client["timeout"]; ok {
		if seconds, ok := durationValueToSeconds(timeoutRaw); ok {
			client["timeout_second"] = seconds
		}
		delete(client, "timeout")
	}
}

func coerceTimeoutSeconds(v any) any {
	switch n := v.(type) {
	case int, int64, uint64, float64:
		return n
	case string:
		if seconds, ok := durationValueToSeconds(n); ok {
			return seconds
		}
		if parsed, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return parsed
		}
	}
	return v
}

func durationValueToSeconds(v any) (int, bool) {
	switch raw := v.(type) {
	case string:
		d, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			return 0, false
		}
		return int(d.Seconds()), true
	case int:
		return raw, true
	case int64:
		return int(raw), true
	case float64:
		return int(raw), true
	default:
		return 0, false
	}
}
