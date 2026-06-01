package geppetto

import (
	"fmt"
	"sort"

	"github.com/dop251/goja"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

type inferenceSettingsProvenance struct {
	RegistrySlug string           `json:"registrySlug,omitempty"`
	ProfileSlug  string           `json:"profileSlug,omitempty"`
	StackLineage []map[string]any `json:"stackLineage,omitempty"`
	Sources      []string         `json:"sources,omitempty"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

type inferenceSettingsRef struct {
	api        *moduleRuntime
	settings   *aistepssettings.InferenceSettings
	provenance inferenceSettingsProvenance
}

func (m *moduleRuntime) newInferenceSettingsRef(settings *aistepssettings.InferenceSettings, provenance inferenceSettingsProvenance) *inferenceSettingsRef {
	return &inferenceSettingsRef{
		api:        m,
		settings:   cloneInferenceSettings(settings),
		provenance: provenance.clone(),
	}
}

func (m *moduleRuntime) newInferenceSettingsObject(ref *inferenceSettingsRef) *goja.Object {
	if ref == nil {
		ref = &inferenceSettingsRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref.cloneFor(m))
	m.mustSet(o, "toJSON", func(goja.FunctionCall) goja.Value {
		return m.toJSValue(ref.redactedSnapshot())
	})
	m.mustSet(o, "clone", func(goja.FunctionCall) goja.Value {
		return m.newInferenceSettingsObject(ref.cloneFor(m))
	})
	m.mustSet(o, "debug", func(goja.FunctionCall) goja.Value {
		return m.toJSValue(ref.debugSnapshot())
	})
	return o
}

func (m *moduleRuntime) requireInferenceSettingsRef(v goja.Value) (*inferenceSettingsRef, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *inferenceSettingsRef:
		return x.cloneFor(m), nil
	case *aistepssettings.InferenceSettings:
		return m.newInferenceSettingsRef(x, inferenceSettingsProvenance{}), nil
	default:
		return nil, fmt.Errorf("expected registry-resolved InferenceSettings wrapper, got %T", ref)
	}
}

func (r *inferenceSettingsRef) cloneFor(api *moduleRuntime) *inferenceSettingsRef {
	if r == nil {
		return nil
	}
	return &inferenceSettingsRef{
		api:        api,
		settings:   cloneInferenceSettings(r.settings),
		provenance: r.provenance.clone(),
	}
}

func (p inferenceSettingsProvenance) clone() inferenceSettingsProvenance {
	return inferenceSettingsProvenance{
		RegistrySlug: p.RegistrySlug,
		ProfileSlug:  p.ProfileSlug,
		StackLineage: cloneSliceOfStringAnyMaps(p.StackLineage),
		Sources:      append([]string(nil), p.Sources...),
		Metadata:     cloneJSONMap(p.Metadata),
	}
}

func provenanceFromResolved(resolved *profiles.ResolvedEngineProfile, sources []string) inferenceSettingsProvenance {
	if resolved == nil {
		return inferenceSettingsProvenance{Sources: append([]string(nil), sources...)}
	}
	lineage := make([]map[string]any, 0, len(resolved.StackLineage))
	for _, entry := range resolved.StackLineage {
		item := map[string]any{
			"registrySlug": entry.RegistrySlug.String(),
			"profileSlug":  entry.EngineProfileSlug.String(),
		}
		if entry.Version != 0 {
			item["version"] = entry.Version
		}
		if entry.Source != "" {
			item["source"] = entry.Source
		}
		lineage = append(lineage, item)
	}
	return inferenceSettingsProvenance{
		RegistrySlug: resolved.RegistrySlug.String(),
		ProfileSlug:  resolved.EngineProfileSlug.String(),
		StackLineage: lineage,
		Sources:      append([]string(nil), sources...),
		Metadata:     cloneJSONMap(resolved.Metadata),
	}
}

func (r *inferenceSettingsRef) redactedSnapshot() map[string]any {
	snapshot := map[string]any{}
	if r != nil && r.settings != nil {
		if encoded, ok := encodeInferenceSettingsValue(r.settings).(map[string]any); ok {
			snapshot = cloneJSONMap(encoded)
		} else if encoded := encodeInferenceSettingsValue(r.settings); encoded != nil {
			snapshot["settings"] = cloneJSONValue(encoded)
		}
	}
	redactSecretsInPlace(snapshot)
	if r != nil {
		snapshot["provenance"] = r.provenance.toMap()
	}
	return snapshot
}

func (r *inferenceSettingsRef) debugSnapshot() map[string]any {
	out := map[string]any{
		"kind":     "InferenceSettings",
		"snapshot": r.redactedSnapshot(),
	}
	if r != nil {
		out["provenance"] = r.provenance.toMap()
	}
	if r != nil && r.settings != nil && r.settings.API != nil {
		out["apiKeyNames"] = sortedStringMapKeys(r.settings.API.APIKeys)
		out["baseURLNames"] = sortedStringMapKeys(r.settings.API.BaseUrls)
	}
	return out
}

func (p inferenceSettingsProvenance) toMap() map[string]any {
	out := map[string]any{}
	if p.RegistrySlug != "" {
		out["registrySlug"] = p.RegistrySlug
	}
	if p.ProfileSlug != "" {
		out["profileSlug"] = p.ProfileSlug
	}
	if len(p.StackLineage) > 0 {
		out["stackLineage"] = cloneSliceOfStringAnyMaps(p.StackLineage)
	}
	if len(p.Sources) > 0 {
		out["sources"] = append([]string(nil), p.Sources...)
	}
	if len(p.Metadata) > 0 {
		out["metadata"] = cloneJSONMap(p.Metadata)
	}
	return out
}

func cloneSliceOfStringAnyMaps(in []map[string]any) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, cloneJSONMap(item))
	}
	return out
}

func redactSecretsInPlace(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			if isSecretSnapshotKey(k) {
				x[k] = redactSecretValue(child)
				continue
			}
			redactSecretsInPlace(child)
		}
	case []any:
		for _, child := range x {
			redactSecretsInPlace(child)
		}
	}
}

func isSecretSnapshotKey(key string) bool {
	switch key {
	case "api_keys", "apiKeys", "api_key", "apiKey", "key", "secret", "token":
		return true
	default:
		return false
	}
}

func redactSecretValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k := range x {
			out[k] = "[REDACTED]"
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = "[REDACTED]"
		}
		return out
	case nil:
		return nil
	default:
		return "[REDACTED]"
	}
}

func sortedStringMapKeys(in map[string]string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for k := range in {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
