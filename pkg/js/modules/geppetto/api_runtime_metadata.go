package geppetto

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/profiles"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type resolvedRuntimeMaterialization struct {
	Middlewares     []middleware.Middleware
	ToolNames       []string
	RuntimeMetadata map[string]any
}

func (m *moduleRuntime) newResolvedProfileObject(resolved *profiles.ResolvedProfile) *goja.Object {
	out := m.vm.NewObject()
	m.attachRef(out, &resolvedProfileRef{Resolved: cloneResolvedProfile(resolved)})
	m.mustSet(out, "registrySlug", resolved.RegistrySlug.String())
	m.mustSet(out, "profileSlug", resolved.ProfileSlug.String())
	m.mustSet(out, "runtimeKey", resolved.RuntimeKey.String())
	m.mustSet(out, "runtimeFingerprint", resolved.RuntimeFingerprint)
	m.mustSet(out, "effectiveRuntime", cloneJSONValue(resolved.EffectiveRuntime))
	if len(resolved.Metadata) > 0 {
		m.mustSet(out, "metadata", cloneJSONMap(resolved.Metadata))
	}
	return out
}

func cloneResolvedProfile(in *profiles.ResolvedProfile) *profiles.ResolvedProfile {
	if in == nil {
		return nil
	}
	out := &profiles.ResolvedProfile{
		RegistrySlug:       in.RegistrySlug,
		ProfileSlug:        in.ProfileSlug,
		RuntimeKey:         in.RuntimeKey,
		EffectiveRuntime:   cloneRuntimeSpec(in.EffectiveRuntime),
		RuntimeFingerprint: in.RuntimeFingerprint,
		Metadata:           cloneJSONMap(in.Metadata),
	}
	return out
}

func cloneRuntimeSpec(in profiles.RuntimeSpec) profiles.RuntimeSpec {
	out := profiles.RuntimeSpec{
		SystemPrompt: in.SystemPrompt,
		Tools:        append([]string(nil), in.Tools...),
	}
	if len(in.Middlewares) > 0 {
		out.Middlewares = make([]profiles.MiddlewareUse, 0, len(in.Middlewares))
		for _, use := range in.Middlewares {
			cloned := profiles.MiddlewareUse{
				Name:   strings.TrimSpace(use.Name),
				ID:     strings.TrimSpace(use.ID),
				Config: cloneJSONValue(use.Config),
			}
			if use.Enabled != nil {
				enabled := *use.Enabled
				cloned.Enabled = &enabled
			}
			out.Middlewares = append(out.Middlewares, cloned)
		}
	}
	return out
}

func (m *moduleRuntime) requireResolvedProfile(v goja.Value) (*profiles.ResolvedProfile, error) {
	ref := m.getRef(v)
	switch x := ref.(type) {
	case *resolvedProfileRef:
		return cloneResolvedProfile(x.Resolved), nil
	case *profiles.ResolvedProfile:
		return cloneResolvedProfile(x), nil
	}

	raw := decodeMap(v.Export())
	if raw == nil {
		return nil, fmt.Errorf("resolved profile must be an object")
	}

	return decodeResolvedProfile(raw)
}

func decodeResolvedProfile(raw map[string]any) (*profiles.ResolvedProfile, error) {
	if raw == nil {
		return nil, fmt.Errorf("resolved profile must not be nil")
	}

	registrySlug, err := parseOptionalRegistrySlug(raw["registrySlug"])
	if err != nil {
		return nil, fmt.Errorf("decode registrySlug: %w", err)
	}
	profileSlug, err := parseRequiredProfileSlug(raw["profileSlug"], "profileSlug")
	if err != nil {
		return nil, fmt.Errorf("decode profileSlug: %w", err)
	}
	runtimeKeyRaw := strings.TrimSpace(toString(raw["runtimeKey"], ""))
	if runtimeKeyRaw == "" {
		return nil, fmt.Errorf("runtimeKey is required")
	}
	runtimeKey, err := profiles.ParseRuntimeKey(runtimeKeyRaw)
	if err != nil {
		return nil, fmt.Errorf("decode runtimeKey: %w", err)
	}
	effectiveRuntime, err := decodeRuntimeSpec(raw["effectiveRuntime"])
	if err != nil {
		return nil, fmt.Errorf("decode effectiveRuntime: %w", err)
	}

	return &profiles.ResolvedProfile{
		RegistrySlug:       registrySlug,
		ProfileSlug:        profileSlug,
		RuntimeKey:         runtimeKey,
		EffectiveRuntime:   effectiveRuntime,
		RuntimeFingerprint: strings.TrimSpace(toString(raw["runtimeFingerprint"], "")),
		Metadata:           cloneJSONMap(decodeMap(raw["metadata"])),
	}, nil
}

func decodeRuntimeSpec(raw any) (profiles.RuntimeSpec, error) {
	out := profiles.RuntimeSpec{}
	if raw == nil {
		return out, nil
	}
	obj := decodeMap(raw)
	if obj == nil {
		return out, fmt.Errorf("runtime spec must be an object")
	}
	out.SystemPrompt = strings.TrimSpace(toString(obj["system_prompt"], ""))

	for _, rawTool := range decodeSlice(obj["tools"]) {
		name := strings.TrimSpace(toString(rawTool, ""))
		if name == "" {
			continue
		}
		out.Tools = append(out.Tools, name)
	}

	for idx, rawMW := range decodeSlice(obj["middlewares"]) {
		cfg := decodeMap(rawMW)
		if cfg == nil {
			return out, fmt.Errorf("middleware entry %d must be an object", idx)
		}
		name := strings.TrimSpace(toString(cfg["name"], ""))
		if name == "" {
			return out, fmt.Errorf("middleware entry %d requires name", idx)
		}
		use := profiles.MiddlewareUse{
			Name:   name,
			ID:     strings.TrimSpace(toString(cfg["id"], "")),
			Config: cloneJSONValue(cfg["config"]),
		}
		if enabledRaw, ok := cfg["enabled"].(bool); ok {
			enabled := enabledRaw
			use.Enabled = &enabled
		}
		out.Middlewares = append(out.Middlewares, use)
	}

	return out, nil
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

func (m *moduleRuntime) materializeResolvedProfile(resolved *profiles.ResolvedProfile) (*resolvedRuntimeMaterialization, error) {
	if resolved == nil {
		return nil, fmt.Errorf("resolved profile must not be nil")
	}

	out := &resolvedRuntimeMaterialization{
		ToolNames:       append([]string(nil), resolved.EffectiveRuntime.Tools...),
		RuntimeMetadata: map[string]any{},
	}

	if prompt := strings.TrimSpace(resolved.EffectiveRuntime.SystemPrompt); prompt != "" {
		mw, err := m.resolveGoMiddleware("systemPrompt", map[string]any{"prompt": prompt})
		if err != nil {
			return nil, fmt.Errorf("materialize system prompt middleware: %w", err)
		}
		out.Middlewares = append(out.Middlewares, mw)
	}

	for idx, use := range resolved.EffectiveRuntime.Middlewares {
		if use.Enabled != nil && !*use.Enabled {
			continue
		}
		cfg := cloneJSONMap(decodeMap(use.Config))
		mw, err := m.resolveGoMiddleware(use.Name, cfg)
		if err != nil {
			return nil, fmt.Errorf("materialize middleware %d (%s): %w", idx, use.Name, err)
		}
		out.Middlewares = append(out.Middlewares, mw)
	}

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

	return out, nil
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
		return nil, fmt.Errorf("resolved profile requested tools but no tool registry is configured")
	}

	filtered := tools.NewInMemoryToolRegistry()
	for _, rawName := range toolNames {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		def, err := base.GetTool(name)
		if err != nil {
			return nil, fmt.Errorf("resolved profile tool %q not present in registry: %w", name, err)
		}
		if err := filtered.RegisterTool(name, *def); err != nil {
			return nil, err
		}
	}

	return filtered, nil
}

func (m *moduleRuntime) applyResolvedProfile(b *builderRef, resolved *profiles.ResolvedProfile) error {
	mat, err := m.materializeResolvedProfile(resolved)
	if err != nil {
		return err
	}
	b.middlewares = append(b.middlewares, mat.Middlewares...)
	if len(mat.ToolNames) > 0 {
		b.runtimeToolNames = append([]string(nil), mat.ToolNames...)
	}
	b.runtimeMetadata = mergeRuntimeMetadata(b.runtimeMetadata, mat.RuntimeMetadata)
	return nil
}
