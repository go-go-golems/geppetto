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
// recursively.
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
	merged := mergeExtensionValue(baseMap, overlayMap)
	mergedMap, ok := merged.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("merged inference settings must resolve to an object")
	}
	return inferenceSettingsFromMap(mergedMap)
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
