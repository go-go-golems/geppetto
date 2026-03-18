package engineprofiles

import aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"

type StackMergeResult struct {
	InferenceSettings *aistepssettings.InferenceSettings
	Extensions        map[string]any
}

// MergeEngineProfileStackLayers merges base->leaf stack layers deterministically.
func MergeEngineProfileStackLayers(layers []EngineProfileStackLayer) (StackMergeResult, error) {
	result := StackMergeResult{
		Extensions: map[string]any{},
	}

	for _, layer := range layers {
		if layer.EngineProfile == nil {
			continue
		}
		profile := layer.EngineProfile

		mergedSettings, err := mergeInferenceSettings(result.InferenceSettings, profile.InferenceSettings)
		if err != nil {
			return StackMergeResult{}, err
		}
		result.InferenceSettings = mergedSettings

		for extensionKey, extensionValue := range profile.Extensions {
			existing, ok := result.Extensions[extensionKey]
			if !ok {
				result.Extensions[extensionKey] = deepCopyAny(extensionValue)
				continue
			}
			result.Extensions[extensionKey] = mergeExtensionValue(existing, extensionValue)
		}
	}

	if len(result.Extensions) == 0 {
		result.Extensions = nil
	}

	return result, nil
}

func mergeExtensionValue(base any, overlay any) any {
	baseMap, baseIsMap := base.(map[string]any)
	overlayMap, overlayIsMap := overlay.(map[string]any)
	if !baseIsMap || !overlayIsMap {
		return deepCopyAny(overlay)
	}

	ret := deepCopyStringAnyMap(baseMap)
	for key, overlayValue := range overlayMap {
		if existingValue, ok := ret[key]; ok {
			ret[key] = mergeExtensionValue(existingValue, overlayValue)
			continue
		}
		ret[key] = deepCopyAny(overlayValue)
	}
	return ret
}
