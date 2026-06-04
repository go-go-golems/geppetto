package geppetto

import (
	"fmt"

	"github.com/dop251/goja"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func (m *moduleRuntime) requireEngineRef(v goja.Value) (*engineRef, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *engineRef:
		return x, nil
	case engine.Engine:
		return &engineRef{Name: "engine", Engine: x}, nil
	default:
		return nil, fmt.Errorf("expected engine reference, got %T (value: %v)", ref, v)
	}
}

func (m *moduleRuntime) requireToolRegistry(v goja.Value) (tools.ToolRegistry, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *toolRegistryRef:
		return x.registry, nil
	case tools.ToolRegistry:
		return x, nil
	default:
		return nil, fmt.Errorf("expected tool registry reference, got %T (value: %v)", ref, v)
	}
}

func (m *moduleRuntime) newEngineObject(ref *engineRef) goja.Value {
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "name", ref.Name)
	if len(ref.Metadata) > 0 {
		m.mustSet(o, "metadata", cloneJSONMap(ref.Metadata))
	}
	if ref.ModelInfo != nil {
		m.mustSet(o, "modelInfo", m.newModelInfoObject(ref.ModelInfo))
	}
	return o
}

func ensureInferenceSettingsProviderDefaults(ss *aistepssettings.InferenceSettings) {
	if ss == nil || ss.Chat == nil || ss.Chat.ApiType == nil {
		return
	}
	if ss.API == nil {
		ss.API = aistepssettings.NewAPISettings()
	}
	if ss.API.BaseUrls == nil {
		ss.API.BaseUrls = map[string]string{}
	}
	if *ss.Chat.ApiType == aitypes.ApiTypeClaude {
		if _, ok := ss.API.BaseUrls["claude-base-url"]; !ok {
			ss.API.BaseUrls["claude-base-url"] = "https://api.anthropic.com"
		}
	}
}

func (m *moduleRuntime) effectiveInferenceSettingsForResolvedProfile(resolved *profiles.ResolvedEngineProfile) (*aistepssettings.InferenceSettings, error) {
	if resolved == nil {
		return nil, fmt.Errorf("resolved profile is required")
	}
	if resolved.InferenceSettings == nil {
		if m == nil || m.defaultInferenceSettings == nil {
			return nil, fmt.Errorf("resolved profile has no inference settings")
		}
		return cloneInferenceSettings(m.defaultInferenceSettings), nil
	}
	if m == nil || m.defaultInferenceSettings == nil {
		return resolved.InferenceSettings, nil
	}
	return profiles.MergeInferenceSettings(m.defaultInferenceSettings, resolved.InferenceSettings)
}
