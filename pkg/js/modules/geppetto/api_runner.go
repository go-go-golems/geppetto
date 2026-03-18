package geppetto

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/profiles"
)

func (m *moduleRuntime) runnerResolveRuntime(call goja.FunctionCall) goja.Value {
	input := map[string]any{}
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		obj := call.Arguments[0].ToObject(m.vm)
		if obj == nil {
			panic(m.vm.NewTypeError("runner.resolveRuntime expects an object"))
		}
		input = decodeMap(obj.Export())
		if input == nil {
			input = map[string]any{}
		}
		runtime, err := m.resolveRunnerRuntime(obj, input)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.newRunnerResolvedRuntimeObject(runtime)
	}

	runtime, err := m.resolveRunnerRuntime(nil, input)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.newRunnerResolvedRuntimeObject(runtime)
}

func (m *moduleRuntime) resolveRunnerRuntime(obj *goja.Object, input map[string]any) (*runnerResolvedRuntimeRef, error) {
	out := &runnerResolvedRuntimeRef{
		RuntimeMetadata: map[string]any{},
		Metadata:        map[string]any{},
	}

	if profileRaw, ok := input["profile"]; ok && profileRaw != nil {
		profileInput := decodeMap(profileRaw)
		if profileInput == nil {
			return nil, fmt.Errorf("runner.resolveRuntime profile must be an object")
		}
		registry, err := m.requireProfileRegistryReader("runner.resolveRuntime")
		if err != nil {
			return nil, err
		}
		in := profiles.ResolveInput{}
		if registrySlug, err := parseOptionalRegistrySlug(profileInput["registrySlug"]); err != nil {
			return nil, err
		} else {
			in.RegistrySlug = registrySlug
		}
		if rawProfileSlug := strings.TrimSpace(toString(profileInput["profileSlug"], "")); rawProfileSlug != "" {
			profileSlug, err := profiles.ParseProfileSlug(rawProfileSlug)
			if err != nil {
				return nil, err
			}
			in.ProfileSlug = profileSlug
		}
		resolved, err := registry.ResolveEffectiveProfile(context.Background(), in)
		if err != nil {
			return nil, err
		}
		out = buildRunnerRuntimeFromResolvedProfile(resolved)
	}

	if systemPrompt := strings.TrimSpace(toString(input["systemPrompt"], "")); systemPrompt != "" {
		out.SystemPrompt = systemPrompt
	}

	if rawMws, ok := input["middlewares"]; ok && rawMws != nil {
		if obj == nil {
			return nil, fmt.Errorf("runner.resolveRuntime middlewares require a live JS object")
		}
		mwsVal := obj.Get("middlewares")
		if mwsVal == nil || goja.IsUndefined(mwsVal) || goja.IsNull(mwsVal) {
			return nil, fmt.Errorf("runner.resolveRuntime middlewares must not be null")
		}
		mwSpecs, err := m.decodeRunnerMiddlewareSpecs(mwsVal)
		if err != nil {
			return nil, err
		}
		out.MiddlewareRefs = append(out.MiddlewareRefs, mwSpecs...)
	}

	if rawToolNames, ok := input["toolNames"]; ok && rawToolNames != nil {
		out.ToolNames = decodeStringArray(rawToolNames)
	}
	if runtimeKey := strings.TrimSpace(toString(input["runtimeKey"], "")); runtimeKey != "" {
		out.RuntimeMetadata["runtime_key"] = runtimeKey
	}
	if fingerprint := strings.TrimSpace(toString(input["runtimeFingerprint"], "")); fingerprint != "" {
		out.RuntimeMetadata["runtime_fingerprint"] = fingerprint
	}
	if version, ok := decodePositiveUint64(input["profileVersion"]); ok {
		out.RuntimeMetadata["profile.version"] = version
	}
	if metadata := cloneJSONMap(decodeMap(input["metadata"])); len(metadata) > 0 {
		out.Metadata = mergeRuntimeMetadata(out.Metadata, metadata)
	}

	return out, nil
}

func buildRunnerRuntimeFromResolvedProfile(resolved *profiles.ResolvedProfile) *runnerResolvedRuntimeRef {
	out := &runnerResolvedRuntimeRef{
		RuntimeMetadata: map[string]any{},
		Metadata:        map[string]any{},
	}
	if resolved == nil {
		return out
	}
	out.Metadata = cloneJSONMap(resolved.Metadata)
	out.SystemPrompt = strings.TrimSpace(resolved.EffectiveRuntime.SystemPrompt)
	out.ToolNames = append([]string(nil), resolved.EffectiveRuntime.Tools...)
	if runtimeKey := strings.TrimSpace(resolved.RuntimeKey.String()); runtimeKey != "" {
		out.RuntimeMetadata["runtime_key"] = runtimeKey
	}
	if fingerprint := strings.TrimSpace(resolved.RuntimeFingerprint); fingerprint != "" {
		out.RuntimeMetadata["runtime_fingerprint"] = fingerprint
	}
	if profileSlug := strings.TrimSpace(resolved.ProfileSlug.String()); profileSlug != "" {
		out.RuntimeMetadata["profile.slug"] = profileSlug
	}
	if registrySlug := strings.TrimSpace(resolved.RegistrySlug.String()); registrySlug != "" {
		out.RuntimeMetadata["profile.registry"] = registrySlug
	}
	if version, ok := decodePositiveUint64(resolved.Metadata["profile.version"]); ok {
		out.RuntimeMetadata["profile.version"] = version
	}
	if out.SystemPrompt != "" {
		out.MiddlewareRefs = append(out.MiddlewareRefs, &goMiddlewareRef{
			Name:    "systemPrompt",
			Options: map[string]any{"prompt": out.SystemPrompt},
		})
	}
	for _, use := range resolved.EffectiveRuntime.Middlewares {
		if use.Enabled != nil && !*use.Enabled {
			continue
		}
		out.MiddlewareRefs = append(out.MiddlewareRefs, &goMiddlewareRef{
			Name:    use.Name,
			Options: cloneJSONMap(decodeMap(use.Config)),
		})
	}
	return out
}

func (m *moduleRuntime) decodeRunnerMiddlewareSpecs(v goja.Value) ([]any, error) {
	obj := v.ToObject(m.vm)
	if obj == nil {
		return nil, fmt.Errorf("runner.resolveRuntime middlewares must be an array")
	}
	lengthVal := obj.Get("length")
	if lengthVal == nil || goja.IsUndefined(lengthVal) {
		return nil, fmt.Errorf("runner.resolveRuntime middlewares must be an array")
	}
	n := int(lengthVal.ToInteger())
	out := make([]any, 0, n)
	for i := 0; i < n; i++ {
		item := obj.Get(fmt.Sprintf("%d", i))
		if item == nil || goja.IsUndefined(item) || goja.IsNull(item) {
			continue
		}
		spec, err := m.decodeMiddlewareSpecValue(item)
		if err != nil {
			return nil, fmt.Errorf("middleware %d: %w", i, err)
		}
		out = append(out, spec)
	}
	return out, nil
}

func (m *moduleRuntime) newRunnerResolvedRuntimeObject(ref *runnerResolvedRuntimeRef) goja.Value {
	o := m.vm.NewObject()
	m.attachRef(o, cloneRunnerResolvedRuntimeRef(ref))
	if ref == nil {
		m.mustSet(o, "middlewares", m.vm.NewArray())
		return o
	}
	if ref.SystemPrompt != "" {
		m.mustSet(o, "systemPrompt", ref.SystemPrompt)
	}
	mwArray := m.vm.NewArray()
	for i, spec := range ref.MiddlewareRefs {
		v, err := m.middlewareObjectFromSpec(spec)
		if err != nil {
			continue
		}
		_ = mwArray.Set(fmt.Sprintf("%d", i), v)
	}
	m.mustSet(o, "middlewares", mwArray)
	if len(ref.ToolNames) > 0 {
		m.mustSet(o, "toolNames", append([]string(nil), ref.ToolNames...))
	}
	if runtimeKey := strings.TrimSpace(toString(ref.RuntimeMetadata["runtime_key"], "")); runtimeKey != "" {
		m.mustSet(o, "runtimeKey", runtimeKey)
	}
	if fingerprint := strings.TrimSpace(toString(ref.RuntimeMetadata["runtime_fingerprint"], "")); fingerprint != "" {
		m.mustSet(o, "runtimeFingerprint", fingerprint)
	}
	if version, ok := decodePositiveUint64(ref.RuntimeMetadata["profile.version"]); ok {
		m.mustSet(o, "profileVersion", version)
	}
	if len(ref.Metadata) > 0 {
		m.mustSet(o, "metadata", cloneJSONMap(ref.Metadata))
	}
	return o
}

func cloneRunnerResolvedRuntimeRef(in *runnerResolvedRuntimeRef) *runnerResolvedRuntimeRef {
	if in == nil {
		return nil
	}
	out := &runnerResolvedRuntimeRef{
		SystemPrompt:    in.SystemPrompt,
		ToolNames:       append([]string(nil), in.ToolNames...),
		RuntimeMetadata: cloneJSONMap(in.RuntimeMetadata),
		Metadata:        cloneJSONMap(in.Metadata),
	}
	if len(in.MiddlewareRefs) > 0 {
		out.MiddlewareRefs = make([]any, 0, len(in.MiddlewareRefs))
		for _, spec := range in.MiddlewareRefs {
			switch x := spec.(type) {
			case *jsMiddlewareRef:
				out.MiddlewareRefs = append(out.MiddlewareRefs, cloneJSMiddlewareRef(x))
			case *goMiddlewareRef:
				out.MiddlewareRefs = append(out.MiddlewareRefs, cloneGoMiddlewareRef(x))
			}
		}
	}
	return out
}

func decodeStringArray(raw any) []string {
	values := decodeSlice(raw)
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		s := strings.TrimSpace(toString(item, ""))
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
