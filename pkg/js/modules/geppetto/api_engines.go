package geppetto

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
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

func inferAPIType(model string) aitypes.ApiType {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(m, "gemini"):
		return aitypes.ApiTypeGemini
	case strings.Contains(m, "claude"):
		return aitypes.ApiTypeClaude
	case strings.HasPrefix(m, "o1"), strings.HasPrefix(m, "o3"), strings.HasPrefix(m, "o4"), strings.HasPrefix(m, "gpt-5"):
		return aitypes.ApiTypeOpenAIResponses
	default:
		return aitypes.ApiTypeOpenAI
	}
}

func profileFromOptions(opts map[string]any) string {
	if opts != nil {
		if p := strings.TrimSpace(toString(opts["profile"], "")); p != "" {
			return p
		}
	}
	return "4o-mini"
}

func (m *moduleRuntime) inferenceSettingsFromEngineOptions(opts map[string]any) (*aistepssettings.InferenceSettings, string, error) {
	ss, err := aistepssettings.NewInferenceSettings()
	if err != nil {
		return nil, "", err
	}

	resolvedProfile := profileFromOptions(opts)
	model := resolvedProfile
	if opts != nil {
		if override := strings.TrimSpace(toString(opts["model"], "")); override != "" {
			model = override
		}
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "4o-mini"
	}

	apiTypeRaw := ""
	if opts != nil {
		apiTypeRaw = strings.TrimSpace(toString(opts["apiType"], ""))
		if apiTypeRaw == "" {
			apiTypeRaw = strings.TrimSpace(toString(opts["provider"], ""))
		}
	}
	apiType := inferAPIType(model)
	if apiTypeRaw != "" {
		apiType = aitypes.ApiType(strings.ToLower(apiTypeRaw))
	}

	ss.Chat.Engine = &model
	ss.Chat.ApiType = &apiType

	if opts != nil {
		if tRaw, ok := opts["temperature"]; ok {
			t := float64(toInt(tRaw, -1))
			switch v := tRaw.(type) {
			case float64:
				t = v
			case float32:
				t = float64(v)
			}
			if t >= 0 {
				ss.Chat.Temperature = &t
			}
		}
		if topPRaw, ok := opts["topP"]; ok {
			topP := float64(toInt(topPRaw, -1))
			switch v := topPRaw.(type) {
			case float64:
				topP = v
			case float32:
				topP = float64(v)
			}
			if topP >= 0 {
				ss.Chat.TopP = &topP
			}
		}
		if maxTok := toInt(opts["maxTokens"], -1); maxTok > 0 {
			ss.Chat.MaxResponseTokens = &maxTok
		}
		if timeoutSec := toInt(opts["timeoutSeconds"], 0); timeoutSec > 0 {
			d := time.Duration(timeoutSec) * time.Second
			ss.Client.Timeout = &d
			ss.Client.TimeoutSeconds = &timeoutSec
		}
		if timeoutMS := toInt(opts["timeoutMs"], 0); timeoutMS > 0 {
			d := time.Duration(timeoutMS) * time.Millisecond
			sec := int(d.Seconds())
			ss.Client.Timeout = &d
			ss.Client.TimeoutSeconds = &sec
		}
	}

	key := ""
	if opts != nil {
		key = strings.TrimSpace(toString(opts["apiKey"], ""))
	}

	// Keep OpenAI key alias populated for responses engine and OpenAI-compatible providers.
	switch apiType {
	case aitypes.ApiTypeOpenAIResponses:
		if key != "" {
			ss.API.APIKeys["openai-api-key"] = key
			ss.API.APIKeys["openai-responses-api-key"] = key
		}
	case aitypes.ApiTypeOpenAI, aitypes.ApiTypeAnyScale, aitypes.ApiTypeFireworks:
		if key != "" {
			ss.API.APIKeys[string(apiType)+"-api-key"] = key
			ss.API.APIKeys["openai-api-key"] = key
		}
	case aitypes.ApiTypeGemini, aitypes.ApiTypeClaude, aitypes.ApiTypeOllama, aitypes.ApiTypeMistral, aitypes.ApiTypePerplexity, aitypes.ApiTypeCohere:
		if key != "" {
			ss.API.APIKeys[string(apiType)+"-api-key"] = key
		}
	}

	if opts != nil {
		if baseURL := strings.TrimSpace(toString(opts["baseURL"], "")); baseURL != "" {
			ss.API.BaseUrls[string(apiType)+"-base-url"] = baseURL
			if apiType == aitypes.ApiTypeOpenAIResponses {
				ss.API.BaseUrls["openai-base-url"] = baseURL
			}
		}
	}
	ensureInferenceSettingsProviderDefaults(ss)

	return ss, resolvedProfile, nil
}

func (m *moduleRuntime) engineFromInferenceSettings(opts map[string]any) (*engineRef, error) {
	ss, resolvedProfile, err := m.inferenceSettingsFromEngineOptions(opts)
	if err != nil {
		return nil, err
	}
	eng, err := enginefactory.NewEngineFromSettings(ss)
	if err != nil {
		return nil, err
	}
	_ = resolvedProfile
	return &engineRef{Name: "config", Engine: eng}, nil
}

func (m *moduleRuntime) newEngineObject(ref *engineRef) goja.Value {
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "name", ref.Name)
	if len(ref.Metadata) > 0 {
		m.mustSet(o, "metadata", cloneJSONMap(ref.Metadata))
	}
	return o
}

func (m *moduleRuntime) engineEcho(call goja.FunctionCall) goja.Value {
	reply := "READY"
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		opts := decodeMap(call.Arguments[0].Export())
		if opts != nil {
			reply = toString(opts["reply"], reply)
		}
	}
	ref := &engineRef{
		Name:   "echo",
		Engine: &echoEngine{reply: reply},
	}
	return m.newEngineObject(ref)
}

func ensureInferenceSettingsProviderDefaults(ss *aistepssettings.InferenceSettings) {
	if ss == nil || ss.Chat == nil || ss.Chat.ApiType == nil {
		return
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

func (m *moduleRuntime) engineFromConfig(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("fromConfig requires options object"))
	}
	opts := decodeMap(call.Arguments[0].Export())
	if opts == nil {
		panic(m.vm.NewTypeError("fromConfig requires options object"))
	}
	ref, err := m.engineFromInferenceSettings(opts)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.newEngineObject(ref)
}

func (m *moduleRuntime) engineFromResolvedProfile(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("fromResolvedProfile requires resolved profile argument"))
	}
	resolved, err := m.requireResolvedEngineProfile(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if resolved.InferenceSettings == nil {
		panic(m.vm.NewGoError(fmt.Errorf("resolved profile has no inference settings")))
	}
	eng, err := enginefactory.NewEngineFromSettings(resolved.InferenceSettings)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.newEngineObject(&engineRef{
		Name:   "resolvedProfile",
		Engine: eng,
		Metadata: map[string]any{
			"profileSlug":  resolved.EngineProfileSlug.String(),
			"registrySlug": resolved.RegistrySlug.String(),
		},
	})
}

func (m *moduleRuntime) engineFromProfile(call goja.FunctionCall) goja.Value {
	registry, err := m.requireEngineProfileRegistryReader("engines.fromProfile")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	in := profiles.ResolveInput{}
	if m.defaultProfileResolve.RegistrySlug != "" {
		in.RegistrySlug = m.defaultProfileResolve.RegistrySlug
	}
	if m.defaultProfileResolve.EngineProfileSlug != "" {
		in.EngineProfileSlug = m.defaultProfileResolve.EngineProfileSlug
	}
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		opts := decodeMap(call.Arguments[0].Export())
		if opts == nil {
			panic(m.vm.NewTypeError("fromProfile expects options object"))
		}
		if registrySlug, err := parseOptionalRegistrySlug(opts["registrySlug"]); err != nil {
			panic(m.vm.NewGoError(err))
		} else {
			in.RegistrySlug = registrySlug
		}
		if rawEngineProfileSlug := strings.TrimSpace(toString(opts["profileSlug"], "")); rawEngineProfileSlug != "" {
			profileSlug, err := profiles.ParseEngineProfileSlug(rawEngineProfileSlug)
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
			in.EngineProfileSlug = profileSlug
		}
	}
	resolved, err := registry.ResolveEngineProfile(context.Background(), in)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.engineFromResolvedProfile(goja.FunctionCall{Arguments: []goja.Value{m.newResolvedEngineProfileObject(resolved)}})
}

func (m *moduleRuntime) engineFromFunction(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(m.vm.NewTypeError("fromFunction requires a JS function"))
	}
	fn, ok := goja.AssertFunction(call.Arguments[0])
	if !ok {
		panic(m.vm.NewTypeError("fromFunction expects callable argument"))
	}
	ref := &engineRef{
		Name: "jsFunction",
		Engine: &jsCallableEngine{
			api: m,
			fn:  fn,
		},
	}
	return m.newEngineObject(ref)
}
