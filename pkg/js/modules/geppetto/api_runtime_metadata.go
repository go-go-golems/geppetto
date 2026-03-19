package geppetto

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"gopkg.in/yaml.v3"
)

func (m *moduleRuntime) newResolvedEngineProfileObject(resolved *profiles.ResolvedEngineProfile) *goja.Object {
	out := m.vm.NewObject()
	m.attachRef(out, &resolvedEngineProfileRef{Resolved: cloneResolvedEngineProfile(resolved)})
	m.mustSet(out, "registrySlug", resolved.RegistrySlug.String())
	m.mustSet(out, "profileSlug", resolved.EngineProfileSlug.String())
	if resolved.InferenceSettings != nil {
		m.mustSet(out, "inferenceSettings", encodeInferenceSettingsValue(resolved.InferenceSettings))
	}
	if len(resolved.Metadata) > 0 {
		m.mustSet(out, "metadata", cloneJSONMap(resolved.Metadata))
	}
	return out
}

func cloneResolvedEngineProfile(in *profiles.ResolvedEngineProfile) *profiles.ResolvedEngineProfile {
	if in == nil {
		return nil
	}
	return &profiles.ResolvedEngineProfile{
		RegistrySlug:      in.RegistrySlug,
		EngineProfileSlug: in.EngineProfileSlug,
		InferenceSettings: cloneJSInferenceSettings(in.InferenceSettings),
		Metadata:          cloneJSONMap(in.Metadata),
	}
}

func cloneJSInferenceSettings(in *aistepssettings.InferenceSettings) *aistepssettings.InferenceSettings {
	if in == nil {
		return nil
	}
	return in.Clone()
}

func (m *moduleRuntime) requireResolvedEngineProfile(v goja.Value) (*profiles.ResolvedEngineProfile, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *resolvedEngineProfileRef:
		return cloneResolvedEngineProfile(x.Resolved), nil
	case *profiles.ResolvedEngineProfile:
		return cloneResolvedEngineProfile(x), nil
	}

	raw := decodeMap(v.Export())
	if raw == nil {
		return nil, fmt.Errorf("resolved profile must be an object")
	}

	return decodeResolvedEngineProfile(raw)
}

func decodeResolvedEngineProfile(raw map[string]any) (*profiles.ResolvedEngineProfile, error) {
	if raw == nil {
		return nil, fmt.Errorf("resolved profile must not be nil")
	}

	registrySlug, err := parseOptionalRegistrySlug(raw["registrySlug"])
	if err != nil {
		return nil, fmt.Errorf("decode registrySlug: %w", err)
	}
	profileSlug, err := parseRequiredEngineProfileSlug(raw["profileSlug"], "profileSlug")
	if err != nil {
		return nil, fmt.Errorf("decode profileSlug: %w", err)
	}
	inferenceSettings, err := decodeInferenceSettings(raw["inferenceSettings"])
	if err != nil {
		return nil, fmt.Errorf("decode inferenceSettings: %w", err)
	}

	return &profiles.ResolvedEngineProfile{
		RegistrySlug:      registrySlug,
		EngineProfileSlug: profileSlug,
		InferenceSettings: inferenceSettings,
		Metadata:          cloneJSONMap(decodeMap(raw["metadata"])),
	}, nil
}

func decodeInferenceSettings(raw any) (*aistepssettings.InferenceSettings, error) {
	if raw == nil {
		return nil, nil
	}
	obj := decodeMap(raw)
	if obj == nil {
		return nil, fmt.Errorf("inference settings must be an object")
	}
	b, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var out aistepssettings.InferenceSettings
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func encodeInferenceSettingsValue(in *aistepssettings.InferenceSettings) any {
	if in == nil {
		return nil
	}
	b, err := yaml.Marshal(in)
	if err != nil {
		return cloneJSONValue(in)
	}
	var out any
	if err := yaml.Unmarshal(b, &out); err != nil {
		return cloneJSONValue(in)
	}
	return out
}

func decodePositiveUint64(v any) (uint64, bool) {
	switch n := v.(type) {
	case uint64:
		if n > 0 {
			return n, true
		}
	case int:
		if n > 0 {
			return uint64(n), true
		}
	case int64:
		if n > 0 {
			return uint64(n), true
		}
	case float64:
		if n > 0 && float64(uint64(n)) == n {
			return uint64(n), true
		}
	}
	return 0, false
}

func mergeRuntimeMetadata(base map[string]any, extra map[string]any) map[string]any {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := cloneJSONMap(base)
	if out == nil {
		out = map[string]any{}
	}
	for k, v := range extra {
		out[k] = cloneJSONValue(v)
	}
	return out
}

func stampTurnRuntimeMetadata(t *turns.Turn, runtimeMetadata map[string]any) {
	if t == nil || len(runtimeMetadata) == 0 {
		return
	}
	attrib := map[string]any{}
	if existing, ok, err := turns.KeyTurnMetaRuntime.Get(t.Metadata); err == nil && ok {
		if m, ok := existing.(map[string]any); ok {
			attrib = mergeRuntimeMetadata(attrib, m)
		}
	}
	attrib = mergeRuntimeMetadata(attrib, runtimeMetadata)
	if len(attrib) == 0 {
		return
	}
	_ = turns.KeyTurnMetaRuntime.Set(&t.Metadata, attrib)
}

func materializeToolRegistry(base tools.ToolRegistry, toolNames []string) (tools.ToolRegistry, error) {
	if len(toolNames) == 0 {
		return base, nil
	}
	if base == nil {
		return nil, fmt.Errorf("resolved runtime requested tools but no tool registry is configured")
	}

	filtered := tools.NewInMemoryToolRegistry()
	for _, rawName := range toolNames {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		def, err := base.GetTool(name)
		if err != nil {
			return nil, fmt.Errorf("resolved runtime tool %q not present in registry: %w", name, err)
		}
		if err := filtered.RegisterTool(name, *def); err != nil {
			return nil, err
		}
	}

	return filtered, nil
}
