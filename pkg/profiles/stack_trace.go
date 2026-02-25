package profiles

import (
	"encoding/json"
	"sort"
	"strings"
)

type ProfileStackTrace struct {
	PathValues   map[string]any              `json:"path_values,omitempty"`
	OrderedPaths []string                    `json:"ordered_paths,omitempty"`
	Trace        map[string]ProfilePathTrace `json:"trace,omitempty"`
}

type ProfilePathTrace struct {
	Path  string                  `json:"path"`
	Value any                     `json:"value"`
	Steps []ProfileStackTraceStep `json:"steps,omitempty"`
}

type ProfileStackTraceStep struct {
	RegistrySlug   RegistrySlug `json:"registry_slug"`
	ProfileSlug    ProfileSlug  `json:"profile_slug"`
	ProfileSource  string       `json:"profile_source,omitempty"`
	ProfileVersion uint64       `json:"profile_version,omitempty"`
	LayerIndex     int          `json:"layer_index"`
	Path           string       `json:"path"`
	Value          any          `json:"value"`
}

type ProfileDebugPathTrace struct {
	Path  string                  `json:"path"`
	Value any                     `json:"value,omitempty"`
	Steps []ProfileStackTraceStep `json:"steps,omitempty"`
}

type ProfileTraceDebugPayload struct {
	Paths []ProfileDebugPathTrace `json:"paths,omitempty"`
}

func NewProfileStackTrace() *ProfileStackTrace {
	return &ProfileStackTrace{
		PathValues: map[string]any{},
		Trace:      map[string]ProfilePathTrace{},
	}
}

func BuildProfileStackTrace(layers []ProfileStackLayer, merged StackMergeResult) *ProfileStackTrace {
	trace := NewProfileStackTrace()
	for i, layer := range layers {
		if layer.Profile == nil {
			continue
		}

		for _, write := range collectTraceWrites("/runtime/step_settings_patch", layer.Profile.Runtime.StepSettingsPatch) {
			trace.recordStep(write.path, write.value, layer, i)
		}
		if strings.TrimSpace(layer.Profile.Runtime.SystemPrompt) != "" {
			trace.recordStep("/runtime/system_prompt", layer.Profile.Runtime.SystemPrompt, layer, i)
		}
		if layer.Profile.Runtime.Tools != nil {
			trace.recordStep("/runtime/tools", append([]string(nil), layer.Profile.Runtime.Tools...), layer, i)
		}
		for mwIndex, middleware := range layer.Profile.Runtime.Middlewares {
			key := middlewareMergeKey(middleware, mwIndex)
			trace.recordStep("/runtime/middlewares/"+escapeJSONPointerToken(key), deepCopyAny(middleware), layer, i)
		}

		keys := make([]string, 0, len(layer.Profile.Extensions))
		for key := range layer.Profile.Extensions {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			base := "/extensions/" + escapeJSONPointerToken(key)
			for _, write := range collectTraceWrites(base, layer.Profile.Extensions[key]) {
				trace.recordStep(write.path, write.value, layer, i)
			}
		}

		trace.recordStep("/policy/allow_overrides", layer.Profile.Policy.AllowOverrides, layer, i)
		trace.recordStep("/policy/read_only", layer.Profile.Policy.ReadOnly, layer, i)
		trace.recordStep("/policy/allowed_override_keys", append([]string(nil), layer.Profile.Policy.AllowedOverrideKeys...), layer, i)
		trace.recordStep("/policy/denied_override_keys", append([]string(nil), layer.Profile.Policy.DeniedOverrideKeys...), layer, i)
	}

	for _, write := range collectTraceWrites("/runtime/step_settings_patch", merged.Runtime.StepSettingsPatch) {
		trace.setFinal(write.path, write.value)
	}
	if strings.TrimSpace(merged.Runtime.SystemPrompt) != "" {
		trace.setFinal("/runtime/system_prompt", merged.Runtime.SystemPrompt)
	}
	if merged.Runtime.Tools != nil {
		trace.setFinal("/runtime/tools", append([]string(nil), merged.Runtime.Tools...))
	}
	for mwIndex, middleware := range merged.Runtime.Middlewares {
		key := middlewareMergeKey(middleware, mwIndex)
		trace.setFinal("/runtime/middlewares/"+escapeJSONPointerToken(key), deepCopyAny(middleware))
	}
	for extensionKey, extensionValue := range merged.Extensions {
		base := "/extensions/" + escapeJSONPointerToken(extensionKey)
		for _, write := range collectTraceWrites(base, extensionValue) {
			trace.setFinal(write.path, write.value)
		}
	}

	trace.setFinal("/policy/allow_overrides", merged.Policy.AllowOverrides)
	trace.setFinal("/policy/read_only", merged.Policy.ReadOnly)
	trace.setFinal("/policy/allowed_override_keys", append([]string(nil), merged.Policy.AllowedOverrideKeys...))
	trace.setFinal("/policy/denied_override_keys", append([]string(nil), merged.Policy.DeniedOverrideKeys...))
	trace.rebuildOrderedPaths()
	return trace
}

func (t *ProfileStackTrace) LatestValue(path string) (any, bool) {
	if t == nil {
		return nil, false
	}
	value, ok := t.PathValues[path]
	if !ok {
		return nil, false
	}
	return deepCopyAny(value), true
}

func (t *ProfileStackTrace) History(path string) []ProfileStackTraceStep {
	if t == nil {
		return nil
	}
	pathTrace, ok := t.Trace[path]
	if !ok {
		return nil
	}
	ret := make([]ProfileStackTraceStep, 0, len(pathTrace.Steps))
	for _, step := range pathTrace.Steps {
		ret = append(ret, ProfileStackTraceStep{
			RegistrySlug:   step.RegistrySlug,
			ProfileSlug:    step.ProfileSlug,
			ProfileSource:  step.ProfileSource,
			ProfileVersion: step.ProfileVersion,
			LayerIndex:     step.LayerIndex,
			Path:           step.Path,
			Value:          deepCopyAny(step.Value),
		})
	}
	return ret
}

func (t *ProfileStackTrace) BuildDebugPayload() *ProfileTraceDebugPayload {
	if t == nil {
		return &ProfileTraceDebugPayload{}
	}
	paths := t.debugPathsInStableOrder()
	out := make([]ProfileDebugPathTrace, 0, len(paths))
	for _, path := range paths {
		trace, ok := t.Trace[path]
		if ok {
			out = append(out, ProfileDebugPathTrace{
				Path:  path,
				Value: deepCopyAny(trace.Value),
				Steps: t.History(path),
			})
			continue
		}
		value, ok := t.PathValues[path]
		if !ok {
			out = append(out, ProfileDebugPathTrace{Path: path})
			continue
		}
		out = append(out, ProfileDebugPathTrace{
			Path:  path,
			Value: deepCopyAny(value),
		})
	}
	return &ProfileTraceDebugPayload{Paths: out}
}

func (t *ProfileStackTrace) MarshalDebugPayload() ([]byte, error) {
	return json.Marshal(t.BuildDebugPayload())
}

func (t *ProfileStackTrace) recordStep(path string, value any, layer ProfileStackLayer, layerIndex int) {
	if t == nil || strings.TrimSpace(path) == "" {
		return
	}
	path = strings.TrimSpace(path)
	if t.PathValues == nil {
		t.PathValues = map[string]any{}
	}
	if t.Trace == nil {
		t.Trace = map[string]ProfilePathTrace{}
	}

	pathTrace := t.Trace[path]
	pathTrace.Path = path
	pathTrace.Value = deepCopyAny(value)
	pathTrace.Steps = append(pathTrace.Steps, ProfileStackTraceStep{
		RegistrySlug:   layer.RegistrySlug,
		ProfileSlug:    layer.ProfileSlug,
		ProfileSource:  strings.TrimSpace(layer.Profile.Metadata.Source),
		ProfileVersion: layer.Profile.Metadata.Version,
		LayerIndex:     layerIndex,
		Path:           path,
		Value:          deepCopyAny(value),
	})
	t.Trace[path] = pathTrace
	t.PathValues[path] = deepCopyAny(pathTrace.Value)
}

func (t *ProfileStackTrace) setFinal(path string, value any) {
	if t == nil || strings.TrimSpace(path) == "" {
		return
	}
	path = strings.TrimSpace(path)
	if t.PathValues == nil {
		t.PathValues = map[string]any{}
	}
	if t.Trace == nil {
		t.Trace = map[string]ProfilePathTrace{}
	}
	pathTrace := t.Trace[path]
	pathTrace.Path = path
	pathTrace.Value = deepCopyAny(value)
	t.Trace[path] = pathTrace
	t.PathValues[path] = deepCopyAny(value)
}

func (t *ProfileStackTrace) rebuildOrderedPaths() {
	if t == nil {
		return
	}
	seen := map[string]struct{}{}
	paths := make([]string, 0, len(t.PathValues)+len(t.Trace))
	for path := range t.PathValues {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	for path := range t.Trace {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	t.OrderedPaths = paths
}

func (t *ProfileStackTrace) debugPathsInStableOrder() []string {
	if t == nil {
		return nil
	}
	if len(t.OrderedPaths) > 0 {
		return append([]string(nil), t.OrderedPaths...)
	}
	seen := map[string]struct{}{}
	paths := make([]string, 0, len(t.PathValues)+len(t.Trace))
	for path := range t.PathValues {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	for path := range t.Trace {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

type traceWrite struct {
	path  string
	value any
}

func collectTraceWrites(basePath string, value any) []traceWrite {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" || value == nil {
		return nil
	}

	obj, isMap := value.(map[string]any)
	if !isMap {
		return []traceWrite{{
			path:  basePath,
			value: deepCopyAny(value),
		}}
	}
	if len(obj) == 0 {
		return []traceWrite{{
			path:  basePath,
			value: map[string]any{},
		}}
	}

	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	ret := make([]traceWrite, 0, len(keys))
	for _, key := range keys {
		childPath := basePath + "/" + escapeJSONPointerToken(key)
		ret = append(ret, collectTraceWrites(childPath, obj[key])...)
	}
	return ret
}

func escapeJSONPointerToken(token string) string {
	token = strings.ReplaceAll(token, "~", "~0")
	token = strings.ReplaceAll(token, "/", "~1")
	return token
}
