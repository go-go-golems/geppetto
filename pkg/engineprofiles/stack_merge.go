package engineprofiles

import (
	"fmt"
	"strings"
)

type StackMergeResult struct {
	Runtime    RuntimeSpec
	Extensions map[string]any
}

// MergeEngineProfileStackLayers merges base->leaf stack layers deterministically.
func MergeEngineProfileStackLayers(layers []EngineProfileStackLayer) (StackMergeResult, error) {
	result := StackMergeResult{
		Runtime: RuntimeSpec{
			SystemPrompt: "",
			Middlewares:  nil,
			Tools:        nil,
		},
		Extensions: map[string]any{},
	}

	mergedMiddlewares := make([]MiddlewareUse, 0, 8)

	for _, layer := range layers {
		if layer.EngineProfile == nil {
			continue
		}
		profile := layer.EngineProfile

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
	}

	if len(mergedMiddlewares) > 0 {
		result.Runtime.Middlewares = mergedMiddlewares
	}
	if len(result.Extensions) == 0 {
		result.Extensions = nil
	}

	return result, nil
}

// MergeEngineProfileStackLayersWithTrace merges base->leaf stack layers and returns
// middlewarecfg-style path traces for field-level merge provenance.
func MergeEngineProfileStackLayersWithTrace(layers []EngineProfileStackLayer) (StackMergeResult, *ProfileStackTrace, error) {
	merged, err := MergeEngineProfileStackLayers(layers)
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
