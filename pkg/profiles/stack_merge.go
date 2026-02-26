package profiles

import (
	"fmt"
	"sort"
	"strings"
)

type StackMergeResult struct {
	Runtime    RuntimeSpec
	Policy     PolicySpec
	Extensions map[string]any
}

// MergeProfileStackLayers merges base->leaf stack layers deterministically.
func MergeProfileStackLayers(layers []ProfileStackLayer) (StackMergeResult, error) {
	result := StackMergeResult{
		Runtime: RuntimeSpec{
			StepSettingsPatch: nil,
			SystemPrompt:      "",
			Middlewares:       nil,
			Tools:             nil,
		},
		Policy: PolicySpec{
			AllowOverrides: false,
		},
		Extensions: map[string]any{},
	}

	policies := make([]PolicySpec, 0, len(layers))
	mergedMiddlewares := make([]MiddlewareUse, 0, 8)

	for _, layer := range layers {
		if layer.Profile == nil {
			continue
		}
		profile := layer.Profile

		mergedPatch, err := MergeRuntimeStepSettingsPatches(result.Runtime.StepSettingsPatch, profile.Runtime.StepSettingsPatch)
		if err != nil {
			return StackMergeResult{}, err
		}
		result.Runtime.StepSettingsPatch = mergedPatch

		if strings.TrimSpace(profile.Runtime.SystemPrompt) != "" {
			result.Runtime.SystemPrompt = profile.Runtime.SystemPrompt
		}
		if profile.Runtime.Tools != nil {
			result.Runtime.Tools = append([]string(nil), profile.Runtime.Tools...)
		}
		if profile.Runtime.Middlewares != nil {
			mergedMiddlewares = mergeMiddlewareLayers(mergedMiddlewares, profile.Runtime.Middlewares)
		}

		for extensionKey, extensionValue := range profile.Extensions {
			existing, ok := result.Extensions[extensionKey]
			if !ok {
				result.Extensions[extensionKey] = deepCopyAny(extensionValue)
				continue
			}
			result.Extensions[extensionKey] = mergeExtensionValue(existing, extensionValue)
		}

		policies = append(policies, clonePolicySpec(profile.Policy))
	}

	if len(mergedMiddlewares) > 0 {
		result.Runtime.Middlewares = mergedMiddlewares
	}
	if len(result.Extensions) == 0 {
		result.Extensions = nil
	}
	result.Policy = mergePolicyLayersRestrictive(policies)

	return result, nil
}

// MergeProfileStackLayersWithTrace merges base->leaf stack layers and returns
// middlewarecfg-style path traces for field-level merge provenance.
func MergeProfileStackLayersWithTrace(layers []ProfileStackLayer) (StackMergeResult, *ProfileStackTrace, error) {
	merged, err := MergeProfileStackLayers(layers)
	if err != nil {
		return StackMergeResult{}, nil, err
	}
	trace := BuildProfileStackTrace(layers, merged)
	return merged, trace, nil
}

func mergeMiddlewareLayers(base []MiddlewareUse, overlay []MiddlewareUse) []MiddlewareUse {
	ret := cloneMiddlewares(base)
	keyIndex := map[string]int{}
	for i, mw := range ret {
		keyIndex[middlewareMergeKey(mw, i)] = i
	}

	for i, mw := range overlay {
		next := MiddlewareUse{
			Name:    strings.TrimSpace(mw.Name),
			ID:      strings.TrimSpace(mw.ID),
			Enabled: cloneBoolPtr(mw.Enabled),
			Config:  deepCopyAny(mw.Config),
		}
		key := middlewareMergeKey(next, i)
		if index, ok := keyIndex[key]; ok {
			ret[index] = next
			continue
		}
		keyIndex[key] = len(ret)
		ret = append(ret, next)
	}
	return ret
}

func middlewareMergeKey(mw MiddlewareUse, index int) string {
	name := strings.TrimSpace(mw.Name)
	id := strings.TrimSpace(mw.ID)
	if id != "" {
		return name + "#" + id
	}
	return fmt.Sprintf("%s[%d]", name, index)
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

func mergePolicyLayersRestrictive(layers []PolicySpec) PolicySpec {
	if len(layers) == 0 {
		return PolicySpec{}
	}

	result := PolicySpec{
		AllowOverrides: true,
		ReadOnly:       false,
	}

	var allowedIntersection map[string]struct{}
	deniedUnion := map[string]struct{}{}

	for _, layer := range layers {
		result.AllowOverrides = result.AllowOverrides && layer.AllowOverrides
		result.ReadOnly = result.ReadOnly || layer.ReadOnly

		layerAllowed := normalizePolicyKeys(layer.AllowedOverrideKeys)
		if len(layerAllowed) > 0 {
			if allowedIntersection == nil {
				allowedIntersection = map[string]struct{}{}
				for key := range layerAllowed {
					allowedIntersection[key] = struct{}{}
				}
			} else {
				for key := range allowedIntersection {
					if _, ok := layerAllowed[key]; !ok {
						delete(allowedIntersection, key)
					}
				}
			}
		}

		layerDenied := normalizePolicyKeys(layer.DeniedOverrideKeys)
		for key := range layerDenied {
			deniedUnion[key] = struct{}{}
		}
	}

	for key := range deniedUnion {
		if allowedIntersection != nil {
			delete(allowedIntersection, key)
		}
	}

	result.DeniedOverrideKeys = sortedPolicyKeys(deniedUnion)
	if allowedIntersection != nil {
		result.AllowedOverrideKeys = sortedPolicyKeys(allowedIntersection)
	}
	return result
}

func normalizePolicyKeys(keys []string) map[string]struct{} {
	ret := map[string]struct{}{}
	for _, key := range keys {
		normalized := canonicalOverrideKey(key)
		if normalized == "" {
			continue
		}
		ret[normalized] = struct{}{}
	}
	return ret
}

func sortedPolicyKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	ret := make([]string, 0, len(set))
	for key := range set {
		ret = append(ret, key)
	}
	sort.Strings(ret)
	return ret
}
