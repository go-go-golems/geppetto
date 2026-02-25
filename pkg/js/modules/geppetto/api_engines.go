package geppetto

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/profiles"
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

func inferAPIKeyFromEnv(apiType aitypes.ApiType) string {
	switch apiType {
	case aitypes.ApiTypeOpenAI, aitypes.ApiTypeOpenAIResponses, aitypes.ApiTypeAnyScale, aitypes.ApiTypeFireworks:
		return strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	case aitypes.ApiTypeOllama:
		return strings.TrimSpace(os.Getenv("OLLAMA_API_KEY"))
	case aitypes.ApiTypeMistral:
		return strings.TrimSpace(os.Getenv("MISTRAL_API_KEY"))
	case aitypes.ApiTypePerplexity:
		return strings.TrimSpace(os.Getenv("PERPLEXITY_API_KEY"))
	case aitypes.ApiTypeCohere:
		return strings.TrimSpace(os.Getenv("COHERE_API_KEY"))
	case aitypes.ApiTypeGemini:
		if v := strings.TrimSpace(os.Getenv("GEMINI_API_KEY")); v != "" {
			return v
		}
		if v := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY")); v != "" {
			return v
		}
		return ""
	case aitypes.ApiTypeClaude:
		return strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	}
	return ""
}

func profileFromPrecedence(explicitProfile string, opts map[string]any) string {
	if p := strings.TrimSpace(explicitProfile); p != "" {
		return p
	}
	if opts != nil {
		if p := strings.TrimSpace(toString(opts["profile"], "")); p != "" {
			return p
		}
	}
	if p := strings.TrimSpace(os.Getenv("PINOCCHIO_PROFILE")); p != "" {
		return p
	}
	return "4o-mini"
}

func (m *moduleRuntime) stepSettingsFromEngineOptions(explicitProfile string, opts map[string]any) (*aistepssettings.StepSettings, string, error) {
	ss, err := aistepssettings.NewStepSettings()
	if err != nil {
		return nil, "", err
	}

	resolvedProfile := profileFromPrecedence(explicitProfile, opts)
	model := resolvedProfile
	if opts != nil && strings.TrimSpace(explicitProfile) == "" {
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
	if key == "" {
		key = inferAPIKeyFromEnv(apiType)
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
	if apiType == aitypes.ApiTypeClaude {
		if _, ok := ss.API.BaseUrls["claude-base-url"]; !ok {
			ss.API.BaseUrls["claude-base-url"] = "https://api.anthropic.com"
		}
	}

	return ss, resolvedProfile, nil
}

func (m *moduleRuntime) engineFromStepSettings(explicitProfile string, opts map[string]any, fromProfile bool) (*engineRef, error) {
	ss, resolvedProfile, err := m.stepSettingsFromEngineOptions(explicitProfile, opts)
	if err != nil {
		return nil, err
	}
	eng, err := enginefactory.NewEngineFromStepSettings(ss)
	if err != nil {
		return nil, err
	}
	name := "config"
	if fromProfile {
		name = "profile:" + resolvedProfile
	}
	return &engineRef{Name: name, Engine: eng}, nil
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

func (m *moduleRuntime) engineFromProfile(call goja.FunctionCall) goja.Value {
	profile := ""
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		profile = call.Arguments[0].String()
	}
	var opts map[string]any
	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
		opts = decodeMap(call.Arguments[1].Export())
	}
	ref, err := m.engineFromResolvedProfile(profile, opts)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.newEngineObject(ref)
}

func (m *moduleRuntime) engineFromResolvedProfile(explicitProfile string, opts map[string]any) (*engineRef, error) {
	if m.profileRegistry == nil {
		return nil, fmt.Errorf("engines.fromProfile requires a configured profile registry")
	}

	in := profiles.ResolveInput{}
	if profile := strings.TrimSpace(explicitProfile); profile != "" {
		parsedProfileSlug, err := profiles.ParseProfileSlug(profile)
		if err != nil {
			return nil, err
		}
		in.ProfileSlug = parsedProfileSlug
	}
	if opts != nil {
		if registrySlugRaw := strings.TrimSpace(toString(opts["registrySlug"], "")); registrySlugRaw != "" {
			parsedRegistrySlug, err := profiles.ParseRegistrySlug(registrySlugRaw)
			if err != nil {
				return nil, err
			}
			in.RegistrySlug = parsedRegistrySlug
		}
		if runtimeKeyRaw := strings.TrimSpace(toString(opts["runtimeKey"], "")); runtimeKeyRaw != "" {
			parsedRuntimeKey, err := profiles.ParseRuntimeKey(runtimeKeyRaw)
			if err != nil {
				return nil, err
			}
			in.RuntimeKeyFallback = parsedRuntimeKey
		}
		if rawOverrides, ok := opts["requestOverrides"]; ok && rawOverrides != nil {
			decoded := decodeMap(rawOverrides)
			if decoded == nil {
				return nil, fmt.Errorf("requestOverrides must be an object")
			}
			in.RequestOverrides = decoded
		}
	}

	resolved, err := m.profileRegistry.ResolveEffectiveProfile(context.Background(), in)
	if err != nil {
		return nil, err
	}
	ss := resolved.EffectiveStepSettings.Clone()
	ensureStepSettingsAPIKeyFromEnv(ss)
	eng, err := enginefactory.NewEngineFromStepSettings(ss)
	if err != nil {
		return nil, err
	}

	return &engineRef{
		Name:   fmt.Sprintf("profile:%s/%s", resolved.RegistrySlug, resolved.ProfileSlug),
		Engine: eng,
		Metadata: map[string]any{
			"profileRegistry":    resolved.RegistrySlug.String(),
			"profileSlug":        resolved.ProfileSlug.String(),
			"runtimeFingerprint": resolved.RuntimeFingerprint,
			"resolvedMetadata":   cloneJSONMap(resolved.Metadata),
		},
	}, nil
}

func ensureStepSettingsAPIKeyFromEnv(ss *aistepssettings.StepSettings) {
	if ss == nil || ss.Chat == nil || ss.Chat.ApiType == nil {
		return
	}
	apiType := *ss.Chat.ApiType
	key := inferAPIKeyFromEnv(apiType)
	if strings.TrimSpace(key) == "" {
		return
	}
	if ss.API.APIKeys == nil {
		ss.API.APIKeys = map[string]string{}
	}

	switch apiType {
	case aitypes.ApiTypeOpenAIResponses:
		if _, ok := ss.API.APIKeys["openai-api-key"]; !ok {
			ss.API.APIKeys["openai-api-key"] = key
		}
		if _, ok := ss.API.APIKeys["openai-responses-api-key"]; !ok {
			ss.API.APIKeys["openai-responses-api-key"] = key
		}
	case aitypes.ApiTypeOpenAI, aitypes.ApiTypeAnyScale, aitypes.ApiTypeFireworks:
		apiKeyName := string(apiType) + "-api-key"
		if _, ok := ss.API.APIKeys[apiKeyName]; !ok {
			ss.API.APIKeys[apiKeyName] = key
		}
		if _, ok := ss.API.APIKeys["openai-api-key"]; !ok {
			ss.API.APIKeys["openai-api-key"] = key
		}
	case aitypes.ApiTypeGemini, aitypes.ApiTypeClaude, aitypes.ApiTypeOllama, aitypes.ApiTypeMistral, aitypes.ApiTypePerplexity, aitypes.ApiTypeCohere:
		apiKeyName := string(apiType) + "-api-key"
		if _, ok := ss.API.APIKeys[apiKeyName]; !ok {
			ss.API.APIKeys[apiKeyName] = key
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
	ref, err := m.engineFromStepSettings("", opts, false)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.newEngineObject(ref)
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
