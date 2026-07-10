package geppetto

import (
	"fmt"

	"github.com/dop251/goja"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	claudesettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	geminisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/gemini"
	openaisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
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

func ensureInferenceSettingsProviderDefaults(ss *aistepssettings.InferenceSettings) error {
	if ss == nil || ss.Chat == nil || ss.Chat.ApiType == nil {
		return nil
	}
	if ss.API == nil {
		ss.API = aistepssettings.NewAPISettings()
	}
	if ss.API.BaseUrls == nil {
		ss.API.BaseUrls = map[string]string{}
	}
	if ss.API.APIKeys == nil {
		ss.API.APIKeys = map[string]string{}
	}
	if ss.API.AllowHTTP == nil {
		ss.API.AllowHTTP = map[string]bool{}
	}
	if ss.API.AllowLocalNetworks == nil {
		ss.API.AllowLocalNetworks = map[string]bool{}
	}
	if ss.Client == nil {
		ss.Client = aistepssettings.NewClientSettings()
	}

	switch *ss.Chat.ApiType {
	case aitypes.ApiTypeOpenAI, aitypes.ApiTypeAnyScale, aitypes.ApiTypeFireworks:
		if ss.OpenAI == nil {
			settings, err := openaisettings.NewSettings()
			if err != nil {
				return fmt.Errorf("initialize OpenAI provider settings: %w", err)
			}
			ss.OpenAI = settings
		}
	case aitypes.ApiTypeClaude, aitypes.ApiType("anthropic"):
		if ss.Claude == nil {
			settings, err := claudesettings.NewSettings()
			if err != nil {
				return fmt.Errorf("initialize Claude provider settings: %w", err)
			}
			ss.Claude = settings
		}
		if _, ok := ss.API.BaseUrls["claude-base-url"]; !ok {
			ss.API.BaseUrls["claude-base-url"] = "https://api.anthropic.com"
		}
	case aitypes.ApiTypeGemini:
		if ss.Gemini == nil {
			settings, err := geminisettings.NewSettings()
			if err != nil {
				return fmt.Errorf("initialize Gemini provider settings: %w", err)
			}
			ss.Gemini = settings
		}
	case aitypes.ApiTypeOpenResponses, aitypes.ApiTypeOpenAIResponses,
		aitypes.ApiTypeOllama, aitypes.ApiTypeMistral, aitypes.ApiTypePerplexity, aitypes.ApiTypeCohere:
		// These API types do not have a provider-specific settings object that
		// needs materialization here. Unsupported types remain unsupported in
		// the engine factory; normalization must not imply provider support.
	}

	return nil
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
