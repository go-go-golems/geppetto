package geppetto

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
)

func (m *moduleRuntime) requireEngineProfileRegistryReader(method string) (profiles.RegistryReader, error) {
	if m.profileRegistry == nil {
		return nil, fmt.Errorf("%s requires a configured profile registry", method)
	}
	return m.profileRegistry, nil
}

func parseOptionalRegistrySlug(raw any) (profiles.RegistrySlug, error) {
	rawSlug := strings.TrimSpace(toString(raw, ""))
	if rawSlug == "" {
		return "", nil
	}
	return profiles.ParseRegistrySlug(rawSlug)
}

func parseRequiredEngineProfileSlug(raw any, field string) (profiles.EngineProfileSlug, error) {
	rawSlug := strings.TrimSpace(toString(raw, ""))
	if rawSlug == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return profiles.ParseEngineProfileSlug(rawSlug)
}

func (m *moduleRuntime) profilesListRegistries(call goja.FunctionCall) goja.Value {
	registry, err := m.requireEngineProfileRegistryReader("profiles.listRegistries")
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
	registry, err := m.requireEngineProfileRegistryReader("profiles.getRegistry")
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

func (m *moduleRuntime) profilesListEngineProfiles(call goja.FunctionCall) goja.Value {
	registry, err := m.requireEngineProfileRegistryReader("profiles.listProfiles")
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

	rows, err := registry.ListEngineProfiles(context.Background(), registrySlug)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(cloneJSONValue(rows))
}

func (m *moduleRuntime) profilesGetEngineProfile(call goja.FunctionCall) goja.Value {
	registry, err := m.requireEngineProfileRegistryReader("profiles.getProfile")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("profiles.getProfile requires profileSlug"))
	}

	profileSlug, err := parseRequiredEngineProfileSlug(call.Arguments[0].Export(), "profileSlug")
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

	ret, err := registry.GetEngineProfile(context.Background(), registrySlug, profileSlug)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(cloneJSONValue(ret))
}

func (m *moduleRuntime) profilesResolve(call goja.FunctionCall) goja.Value {
	registry, err := m.requireEngineProfileRegistryReader("profiles.resolve")
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
			panic(m.vm.NewTypeError("profiles.resolve expects an object"))
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
	return m.newResolvedEngineProfileObject(resolved)
}

func decodeEngineProfileRegistrySources(raw any) ([]string, error) {
	if raw == nil {
		return nil, fmt.Errorf("profile registry sources are required")
	}
	switch v := raw.(type) {
	case string:
		return profiles.ParseEngineProfileRegistrySourceEntries(v)
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

	entries, err := decodeEngineProfileRegistrySources(call.Arguments[0].Export())
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
	m.profileRegistry = m.baseEngineProfileRegistry
	m.profileRegistryCloser = m.baseEngineProfileRegistryCloser
	m.profileRegistryOwned = false
	m.profileRegistrySpec = append([]string(nil), m.baseEngineProfileRegistrySpec...)
	return goja.Undefined()
}

func (m *moduleRuntime) profilesGetConnectedSources(call goja.FunctionCall) goja.Value {
	return m.toJSValue(append([]string(nil), m.profileRegistrySpec...))
}
