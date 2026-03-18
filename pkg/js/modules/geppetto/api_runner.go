package geppetto

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/profiles"
	"github.com/go-go-golems/geppetto/pkg/turns"
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

func (m *moduleRuntime) runnerPrepare(call goja.FunctionCall) goja.Value {
	prepared, err := m.prepareRunnerCall(call)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.newPreparedRunObject(prepared)
}

func (m *moduleRuntime) runnerRun(call goja.FunctionCall) goja.Value {
	prepared, err := m.prepareRunnerCall(call)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	opts, err := m.parseRunOptions(call.Arguments, 1)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	out, err := prepared.session.runSync(nil, opts)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	v, err := m.encodeTurnValue(out)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return v
}

func (m *moduleRuntime) runnerStart(call goja.FunctionCall) goja.Value {
	prepared, err := m.prepareRunnerCall(call)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	opts, err := m.parseRunOptions(call.Arguments, 1)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.attachPreparedRunToHandle(prepared, prepared.session.start(nil, opts))
}

func (m *moduleRuntime) prepareRunnerCall(call goja.FunctionCall) (*preparedRunRef, error) {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		return nil, fmt.Errorf("runner.prepare requires options object with engine")
	}
	return m.prepareRunnerOptions(call.Arguments[0])
}

func (m *moduleRuntime) prepareRunnerOptions(v goja.Value) (*preparedRunRef, error) {
	obj := v.ToObject(m.vm)
	if obj == nil {
		return nil, fmt.Errorf("runner options must be an object")
	}

	b := m.newBuilderRef()
	if err := m.applyBuilderOptions(b, v); err != nil {
		return nil, err
	}

	runtimeVal := obj.Get("runtime")
	var runtimeRef *runnerResolvedRuntimeRef
	if runtimeVal != nil && !goja.IsUndefined(runtimeVal) && !goja.IsNull(runtimeVal) {
		var err error
		runtimeRef, err = m.requireRunnerResolvedRuntime(runtimeVal)
		if err != nil {
			return nil, err
		}
		if err := m.applyRunnerResolvedRuntime(b, runtimeRef); err != nil {
			return nil, err
		}
	}

	sr, err := b.buildSession()
	if err != nil {
		return nil, err
	}
	if sessionIDVal := obj.Get("sessionId"); sessionIDVal != nil && !goja.IsUndefined(sessionIDVal) && !goja.IsNull(sessionIDVal) {
		if sessionID := strings.TrimSpace(toString(sessionIDVal.Export(), "")); sessionID != "" {
			sr.session.SessionID = sessionID
		}
	}

	turn, err := m.buildPreparedTurn(obj, sr.runtimeMetadata)
	if err != nil {
		return nil, err
	}
	sr.session.Append(turn)

	return &preparedRunRef{
		api:     m,
		session: sr,
		turn:    turn,
		runtime: cloneRunnerResolvedRuntimeRef(runtimeRef),
	}, nil
}

func (m *moduleRuntime) requireRunnerResolvedRuntime(v goja.Value) (*runnerResolvedRuntimeRef, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *runnerResolvedRuntimeRef:
		return cloneRunnerResolvedRuntimeRef(x), nil
	}

	obj := v.ToObject(m.vm)
	if obj == nil {
		return nil, fmt.Errorf("runner runtime must be an object")
	}
	input := decodeMap(obj.Export())
	if input == nil {
		input = map[string]any{}
	}
	return m.resolveRunnerRuntime(obj, input)
}

func (m *moduleRuntime) applyRunnerResolvedRuntime(b *builderRef, runtime *runnerResolvedRuntimeRef) error {
	if runtime == nil {
		return nil
	}
	for idx, spec := range runtime.MiddlewareRefs {
		mw, err := m.materializeMiddlewareSpec(spec)
		if err != nil {
			return fmt.Errorf("runner runtime middleware %d: %w", idx, err)
		}
		b.middlewares = append(b.middlewares, mw)
	}
	if len(runtime.ToolNames) > 0 {
		b.runtimeToolNames = append([]string(nil), runtime.ToolNames...)
	}
	b.runtimeMetadata = mergeRuntimeMetadata(b.runtimeMetadata, runtime.RuntimeMetadata)
	return nil
}

func (m *moduleRuntime) buildPreparedTurn(obj *goja.Object, runtimeMetadata map[string]any) (*turns.Turn, error) {
	var seed *turns.Turn
	seedVal := obj.Get("seedTurn")
	if seedVal != nil && !goja.IsUndefined(seedVal) && !goja.IsNull(seedVal) {
		var err error
		seed, err = m.decodeTurnValue(seedVal)
		if err != nil {
			return nil, err
		}
	}
	var prompt string
	if promptVal := obj.Get("prompt"); promptVal != nil && !goja.IsUndefined(promptVal) && !goja.IsNull(promptVal) {
		prompt = strings.TrimSpace(toString(promptVal.Export(), ""))
	}
	if seed == nil && prompt == "" {
		return nil, fmt.Errorf("runner requires prompt or seedTurn")
	}
	if seed == nil {
		seed = &turns.Turn{}
	}
	seed = seed.Clone()
	if seed == nil {
		seed = &turns.Turn{}
	}
	seed.ID = ""
	if prompt != "" {
		turns.AppendBlock(seed, turns.NewUserTextBlock(prompt))
	}
	stampTurnRuntimeMetadata(seed, runtimeMetadata)
	return seed, nil
}

func (m *moduleRuntime) newPreparedRunObject(prepared *preparedRunRef) goja.Value {
	o := m.vm.NewObject()
	m.attachRef(o, prepared)
	if prepared == nil {
		return o
	}
	m.mustSet(o, "session", m.newSessionObject(prepared.session))
	turnValue, err := m.encodeTurnValue(prepared.turn)
	if err == nil {
		m.mustSet(o, "turn", turnValue)
	}
	m.mustSet(o, "runtime", m.newRunnerResolvedRuntimeObject(prepared.runtime))
	m.mustSet(o, "run", func(call goja.FunctionCall) goja.Value {
		opts, err := m.parseRunOptions(call.Arguments, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		out, err := prepared.session.runSync(nil, opts)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		v, err := m.encodeTurnValue(out)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return v
	})
	m.mustSet(o, "start", func(call goja.FunctionCall) goja.Value {
		opts, err := m.parseRunOptions(call.Arguments, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.attachPreparedRunToHandle(prepared, prepared.session.start(nil, opts))
	})
	return o
}

func (m *moduleRuntime) attachPreparedRunToHandle(prepared *preparedRunRef, handle goja.Value) goja.Value {
	obj := handle.ToObject(m.vm)
	if obj == nil || prepared == nil {
		return handle
	}
	m.mustSet(obj, "session", m.newSessionObject(prepared.session))
	if turnValue, err := m.encodeTurnValue(prepared.turn); err == nil {
		m.mustSet(obj, "turn", turnValue)
	}
	m.mustSet(obj, "runtime", m.newRunnerResolvedRuntimeObject(prepared.runtime))
	return obj
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
		setRunnerSystemPrompt(out, systemPrompt)
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
		setRunnerSystemPrompt(out, out.SystemPrompt)
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

func setRunnerSystemPrompt(out *runnerResolvedRuntimeRef, prompt string) {
	if out == nil {
		return
	}
	out.SystemPrompt = strings.TrimSpace(prompt)
	filtered := make([]any, 0, len(out.MiddlewareRefs))
	for _, spec := range out.MiddlewareRefs {
		goRef, ok := spec.(*goMiddlewareRef)
		if ok && strings.TrimSpace(goRef.Name) == "systemPrompt" {
			continue
		}
		filtered = append(filtered, spec)
	}
	out.MiddlewareRefs = filtered
	if out.SystemPrompt == "" {
		return
	}
	out.MiddlewareRefs = append([]any{&goMiddlewareRef{
		Name:    "systemPrompt",
		Options: map[string]any{"prompt": out.SystemPrompt},
	}}, out.MiddlewareRefs...)
}
