package geppetto

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/profiles"
)

func (m *moduleRuntime) requireProfileRegistryReader(method string) (profiles.RegistryReader, error) {
	if m.profileRegistry == nil {
		return nil, fmt.Errorf("%s requires a configured profile registry", method)
	}
	return m.profileRegistry, nil
}

func (m *moduleRuntime) requireProfileRegistryWriter(method string) (profiles.RegistryWriter, error) {
	if m.profileRegistryWriter != nil {
		return m.profileRegistryWriter, nil
	}
	if m.profileRegistry == nil {
		return nil, fmt.Errorf("%s requires a configured profile registry", method)
	}
	writer, ok := m.profileRegistry.(profiles.RegistryWriter)
	if !ok {
		return nil, fmt.Errorf("%s requires a writable profile registry", method)
	}
	return writer, nil
}

func parseOptionalRegistrySlug(raw any) (profiles.RegistrySlug, error) {
	rawSlug := strings.TrimSpace(toString(raw, ""))
	if rawSlug == "" {
		return "", nil
	}
	return profiles.ParseRegistrySlug(rawSlug)
}

func parseRequiredProfileSlug(raw any, field string) (profiles.ProfileSlug, error) {
	rawSlug := strings.TrimSpace(toString(raw, ""))
	if rawSlug == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return profiles.ParseProfileSlug(rawSlug)
}

func decodeProfileValue(raw any) (*profiles.Profile, error) {
	obj := decodeMap(raw)
	if obj == nil {
		return nil, fmt.Errorf("profile must be an object")
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("marshal profile: %w", err)
	}
	var profile profiles.Profile
	if err := json.Unmarshal(b, &profile); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	return &profile, nil
}

func decodeProfilePatchValue(raw any) (profiles.ProfilePatch, error) {
	obj := decodeMap(raw)
	if obj == nil {
		return profiles.ProfilePatch{}, fmt.Errorf("profile patch must be an object")
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return profiles.ProfilePatch{}, fmt.Errorf("marshal profile patch: %w", err)
	}
	var patch profiles.ProfilePatch
	if err := json.Unmarshal(b, &patch); err != nil {
		return profiles.ProfilePatch{}, fmt.Errorf("decode profile patch: %w", err)
	}
	return patch, nil
}

func parseMutationOptions(raw any) (profiles.RegistrySlug, profiles.WriteOptions, error) {
	opts := decodeMap(raw)
	if opts == nil {
		return "", profiles.WriteOptions{}, nil
	}

	registrySlug, err := parseOptionalRegistrySlug(opts["registrySlug"])
	if err != nil {
		return "", profiles.WriteOptions{}, err
	}

	writeObjRaw, hasWrite := opts["write"]
	if !hasWrite || writeObjRaw == nil {
		return registrySlug, profiles.WriteOptions{}, nil
	}
	writeObj := decodeMap(writeObjRaw)
	if writeObj == nil {
		return "", profiles.WriteOptions{}, fmt.Errorf("write option must be an object")
	}

	expectedVersion := toInt(writeObj["expectedVersion"], toInt(writeObj["expected_version"], 0))
	if expectedVersion < 0 {
		return "", profiles.WriteOptions{}, fmt.Errorf("write.expectedVersion must be >= 0")
	}

	return registrySlug, profiles.WriteOptions{
		ExpectedVersion: uint64(expectedVersion),
		Actor:           strings.TrimSpace(toString(writeObj["actor"], "")),
		Source:          strings.TrimSpace(toString(writeObj["source"], "")),
	}, nil
}

func encodeResolvedProfile(resolved *profiles.ResolvedProfile) map[string]any {
	if resolved == nil {
		return nil
	}
	out := map[string]any{
		"registrySlug":       resolved.RegistrySlug.String(),
		"profileSlug":        resolved.ProfileSlug.String(),
		"runtimeKey":         resolved.RuntimeKey.String(),
		"runtimeFingerprint": resolved.RuntimeFingerprint,
		"effectiveRuntime":   cloneJSONValue(resolved.EffectiveRuntime),
	}
	if resolved.EffectiveStepSettings != nil {
		out["effectiveStepSettings"] = cloneJSONValue(resolved.EffectiveStepSettings)
	}
	if len(resolved.Metadata) > 0 {
		out["metadata"] = cloneJSONMap(resolved.Metadata)
	}
	return out
}

func (m *moduleRuntime) profilesListRegistries(call goja.FunctionCall) goja.Value {
	registry, err := m.requireProfileRegistryReader("profiles.listRegistries")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	rows, err := registry.ListRegistries(context.Background())
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(cloneJSONValue(rows))
}

func (m *moduleRuntime) profilesGetRegistry(call goja.FunctionCall) goja.Value {
	registry, err := m.requireProfileRegistryReader("profiles.getRegistry")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	var registrySlug profiles.RegistrySlug
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		registrySlug, err = parseOptionalRegistrySlug(call.Arguments[0].Export())
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
	}

	ret, err := registry.GetRegistry(context.Background(), registrySlug)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(cloneJSONValue(ret))
}

func (m *moduleRuntime) profilesListProfiles(call goja.FunctionCall) goja.Value {
	registry, err := m.requireProfileRegistryReader("profiles.listProfiles")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	var registrySlug profiles.RegistrySlug
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		registrySlug, err = parseOptionalRegistrySlug(call.Arguments[0].Export())
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
	}

	rows, err := registry.ListProfiles(context.Background(), registrySlug)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(cloneJSONValue(rows))
}

func (m *moduleRuntime) profilesGetProfile(call goja.FunctionCall) goja.Value {
	registry, err := m.requireProfileRegistryReader("profiles.getProfile")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("profiles.getProfile requires profileSlug"))
	}

	profileSlug, err := parseRequiredProfileSlug(call.Arguments[0].Export(), "profileSlug")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	var registrySlug profiles.RegistrySlug
	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
		registrySlug, err = parseOptionalRegistrySlug(call.Arguments[1].Export())
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
	}

	ret, err := registry.GetProfile(context.Background(), registrySlug, profileSlug)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(cloneJSONValue(ret))
}

func (m *moduleRuntime) profilesResolve(call goja.FunctionCall) goja.Value {
	registry, err := m.requireProfileRegistryReader("profiles.resolve")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	in := profiles.ResolveInput{}
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		opts := decodeMap(call.Arguments[0].Export())
		if opts == nil {
			panic(m.vm.NewTypeError("profiles.resolve expects an object"))
		}
		if registrySlug, err := parseOptionalRegistrySlug(opts["registrySlug"]); err != nil {
			panic(m.vm.NewGoError(err))
		} else {
			in.RegistrySlug = registrySlug
		}
		if rawProfileSlug := strings.TrimSpace(toString(opts["profileSlug"], "")); rawProfileSlug != "" {
			profileSlug, err := profiles.ParseProfileSlug(rawProfileSlug)
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
			in.ProfileSlug = profileSlug
		}
		runtimeKeyRaw := strings.TrimSpace(toString(opts["runtimeKeyFallback"], strings.TrimSpace(toString(opts["runtimeKey"], ""))))
		if runtimeKeyRaw != "" {
			runtimeKey, err := profiles.ParseRuntimeKey(runtimeKeyRaw)
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
			in.RuntimeKeyFallback = runtimeKey
		}
		if rawOverrides, ok := opts["requestOverrides"]; ok && rawOverrides != nil {
			overrides := decodeMap(rawOverrides)
			if overrides == nil {
				panic(m.vm.NewTypeError("profiles.resolve requestOverrides must be an object"))
			}
			in.RequestOverrides = overrides
		}
	}

	resolved, err := registry.ResolveEffectiveProfile(context.Background(), in)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(encodeResolvedProfile(resolved))
}

func (m *moduleRuntime) profilesCreateProfile(call goja.FunctionCall) goja.Value {
	writer, err := m.requireProfileRegistryWriter("profiles.createProfile")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("profiles.createProfile requires a profile object"))
	}

	profile, err := decodeProfileValue(call.Arguments[0].Export())
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	var optsRaw any
	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
		optsRaw = call.Arguments[1].Export()
	}
	registrySlug, writeOpts, err := parseMutationOptions(optsRaw)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	created, err := writer.CreateProfile(context.Background(), registrySlug, profile, writeOpts)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(cloneJSONValue(created))
}

func (m *moduleRuntime) profilesUpdateProfile(call goja.FunctionCall) goja.Value {
	writer, err := m.requireProfileRegistryWriter("profiles.updateProfile")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if len(call.Arguments) < 2 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("profiles.updateProfile requires profileSlug and patch"))
	}

	profileSlug, err := parseRequiredProfileSlug(call.Arguments[0].Export(), "profileSlug")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	patch, err := decodeProfilePatchValue(call.Arguments[1].Export())
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	var optsRaw any
	if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) && !goja.IsNull(call.Arguments[2]) {
		optsRaw = call.Arguments[2].Export()
	}
	registrySlug, writeOpts, err := parseMutationOptions(optsRaw)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	updated, err := writer.UpdateProfile(context.Background(), registrySlug, profileSlug, patch, writeOpts)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(cloneJSONValue(updated))
}

func (m *moduleRuntime) profilesDeleteProfile(call goja.FunctionCall) goja.Value {
	writer, err := m.requireProfileRegistryWriter("profiles.deleteProfile")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("profiles.deleteProfile requires profileSlug"))
	}

	profileSlug, err := parseRequiredProfileSlug(call.Arguments[0].Export(), "profileSlug")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	var optsRaw any
	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
		optsRaw = call.Arguments[1].Export()
	}
	registrySlug, writeOpts, err := parseMutationOptions(optsRaw)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	if err := writer.DeleteProfile(context.Background(), registrySlug, profileSlug, writeOpts); err != nil {
		panic(m.vm.NewGoError(err))
	}
	return goja.Undefined()
}

func (m *moduleRuntime) profilesSetDefaultProfile(call goja.FunctionCall) goja.Value {
	writer, err := m.requireProfileRegistryWriter("profiles.setDefaultProfile")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("profiles.setDefaultProfile requires profileSlug"))
	}

	profileSlug, err := parseRequiredProfileSlug(call.Arguments[0].Export(), "profileSlug")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	var optsRaw any
	if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
		optsRaw = call.Arguments[1].Export()
	}
	registrySlug, writeOpts, err := parseMutationOptions(optsRaw)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	if err := writer.SetDefaultProfile(context.Background(), registrySlug, profileSlug, writeOpts); err != nil {
		panic(m.vm.NewGoError(err))
	}
	return goja.Undefined()
}

func decodeProfileRegistrySources(raw any) ([]string, error) {
	if raw == nil {
		return nil, fmt.Errorf("profile registry sources are required")
	}
	switch v := raw.(type) {
	case string:
		return profiles.ParseProfileRegistrySourceEntries(v)
	case []string:
		ret := make([]string, 0, len(v))
		for i, entry := range v {
			s := strings.TrimSpace(entry)
			if s == "" {
				return nil, fmt.Errorf("profile registry source entry %d is empty", i)
			}
			ret = append(ret, s)
		}
		return ret, nil
	case []any:
		ret := make([]string, 0, len(v))
		for i, rawEntry := range v {
			s := strings.TrimSpace(toString(rawEntry, ""))
			if s == "" {
				return nil, fmt.Errorf("profile registry source entry %d must be a non-empty string", i)
			}
			ret = append(ret, s)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("profile registry sources must be a comma-separated string or string array")
	}
}

func (m *moduleRuntime) profilesConnectStack(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("profiles.connectStack requires profile registry sources"))
	}

	entries, err := decodeProfileRegistrySources(call.Arguments[0].Export())
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if len(entries) == 0 {
		panic(m.vm.NewGoError(fmt.Errorf("profile registry sources must not be empty")))
	}

	specs, err := profiles.ParseRegistrySourceSpecs(entries)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	chain, err := profiles.NewChainedRegistryFromSourceSpecs(context.Background(), specs)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	// Close previously owned chain to avoid leaking source handles.
	if m.profileRegistryOwned && m.profileRegistryCloser != nil {
		_ = m.profileRegistryCloser.Close()
	}

	m.profileRegistry = chain
	m.profileRegistryWriter = chain
	m.profileRegistryCloser = chain
	m.profileRegistryOwned = true
	m.profileRegistrySpec = append([]string(nil), entries...)

	rows, err := chain.ListRegistries(context.Background())
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(map[string]any{
		"sources":    append([]string(nil), m.profileRegistrySpec...),
		"registries": cloneJSONValue(rows),
	})
}

func (m *moduleRuntime) profilesDisconnectStack(call goja.FunctionCall) goja.Value {
	if m.profileRegistryOwned && m.profileRegistryCloser != nil {
		_ = m.profileRegistryCloser.Close()
	}
	m.profileRegistry = m.baseProfileRegistry
	m.profileRegistryWriter = m.baseProfileRegistryWriter
	m.profileRegistryCloser = m.baseProfileRegistryCloser
	m.profileRegistryOwned = false
	m.profileRegistrySpec = append([]string(nil), m.baseProfileRegistrySpec...)
	return goja.Undefined()
}

func (m *moduleRuntime) profilesGetConnectedSources(call goja.FunctionCall) goja.Value {
	return m.toJSValue(append([]string(nil), m.profileRegistrySpec...))
}
