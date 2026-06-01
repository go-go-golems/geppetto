package geppetto

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/dop251/goja"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
)

type inferenceRegistryRef struct {
	api      *moduleRuntime
	registry profiles.RegistryReader
	closer   io.Closer
	sources  []string
	closed   bool
	owned    bool
}

func (m *moduleRuntime) inferenceProfilesLoad(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("inferenceProfiles.load requires registry source string or string array"))
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
	return m.newInferenceRegistryObject(&inferenceRegistryRef{
		api:      m,
		registry: chain,
		closer:   chain,
		sources:  append([]string(nil), entries...),
		owned:    true,
	})
}

func (m *moduleRuntime) inferenceProfilesResolve(call goja.FunctionCall) goja.Value {
	registry, err := m.requireEngineProfileRegistryReader("inferenceProfiles.resolve")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	sources := append([]string(nil), m.profileRegistrySpec...)
	in, err := m.resolveInputFromCall(call, "inferenceProfiles.resolve")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.resolveInferenceSettingsFromRegistry(registry, sources, in)
}

func (m *moduleRuntime) inferenceProfilesDefault(call goja.FunctionCall) goja.Value {
	registry, err := m.requireEngineProfileRegistryReader("inferenceProfiles.default")
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	var closer io.Closer
	if c, ok := registry.(io.Closer); ok {
		closer = c
	}
	return m.newInferenceRegistryObject(&inferenceRegistryRef{
		api:      m,
		registry: registry,
		closer:   closer,
		sources:  append([]string(nil), m.profileRegistrySpec...),
		owned:    false,
	})
}

func (m *moduleRuntime) newInferenceRegistryObject(ref *inferenceRegistryRef) *goja.Object {
	if ref == nil {
		ref = &inferenceRegistryRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "listRegistries", func(goja.FunctionCall) goja.Value {
		reader := ref.requireOpen("registry.listRegistries")
		rows, err := reader.ListRegistries(context.Background())
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(cloneJSONValue(rows))
	})
	m.mustSet(o, "listProfiles", func(call goja.FunctionCall) goja.Value {
		reader := ref.requireOpen("registry.listProfiles")
		registrySlug, err := optionalRegistrySlugFromArg(call, 0)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		rows, err := reader.ListEngineProfiles(context.Background(), registrySlug)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(cloneJSONValue(rows))
	})
	m.mustSet(o, "resolve", func(call goja.FunctionCall) goja.Value {
		reader := ref.requireOpen("registry.resolve")
		in, err := m.resolveInputFromCall(call, "registry.resolve")
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.resolveInferenceSettingsFromRegistry(reader, ref.sources, in)
	})
	m.mustSet(o, "close", func(goja.FunctionCall) goja.Value {
		if ref.closed {
			return goja.Undefined()
		}
		ref.closed = true
		if ref.owned && ref.closer != nil {
			if err := ref.closer.Close(); err != nil {
				panic(m.vm.NewGoError(err))
			}
		}
		return goja.Undefined()
	})
	m.mustSet(o, "sources", append([]string(nil), ref.sources...))
	return o
}

func (r *inferenceRegistryRef) requireOpen(method string) profiles.RegistryReader {
	if r == nil || r.registry == nil {
		panic(r.api.vm.NewGoError(fmt.Errorf("%s requires an initialized inference registry", method)))
	}
	if r.closed {
		panic(r.api.vm.NewGoError(fmt.Errorf("%s called after registry.close", method)))
	}
	return r.registry
}

func optionalRegistrySlugFromArg(call goja.FunctionCall, index int) (profiles.RegistrySlug, error) {
	if len(call.Arguments) <= index || goja.IsUndefined(call.Arguments[index]) || goja.IsNull(call.Arguments[index]) {
		return "", nil
	}
	return parseOptionalRegistrySlug(call.Arguments[index].Export())
}

func (m *moduleRuntime) resolveInputFromCall(call goja.FunctionCall, method string) (profiles.ResolveInput, error) {
	in := profiles.ResolveInput{}
	if m.defaultProfileResolve.RegistrySlug != "" {
		in.RegistrySlug = m.defaultProfileResolve.RegistrySlug
	}
	if m.defaultProfileResolve.EngineProfileSlug != "" {
		in.EngineProfileSlug = m.defaultProfileResolve.EngineProfileSlug
	}
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		return in, nil
	}
	raw := call.Arguments[0].Export()
	if s := strings.TrimSpace(toString(raw, "")); s != "" {
		profileSlug, err := profiles.ParseEngineProfileSlug(s)
		if err != nil {
			return in, err
		}
		in.EngineProfileSlug = profileSlug
		return in, nil
	}
	opts := decodeMap(raw)
	if opts == nil {
		return in, fmt.Errorf("%s expects a profile string or options object", method)
	}
	if rawRegistrySlug, ok := opts["registry"]; ok {
		registrySlug, err := parseOptionalRegistrySlug(rawRegistrySlug)
		if err != nil {
			return in, err
		}
		in.RegistrySlug = registrySlug
	}
	if rawRegistrySlug, ok := opts["registrySlug"]; ok {
		registrySlug, err := parseOptionalRegistrySlug(rawRegistrySlug)
		if err != nil {
			return in, err
		}
		in.RegistrySlug = registrySlug
	}
	if rawProfileSlug := strings.TrimSpace(toString(opts["profile"], "")); rawProfileSlug != "" {
		profileSlug, err := profiles.ParseEngineProfileSlug(rawProfileSlug)
		if err != nil {
			return in, err
		}
		in.EngineProfileSlug = profileSlug
	}
	if rawProfileSlug := strings.TrimSpace(toString(opts["profileSlug"], "")); rawProfileSlug != "" {
		profileSlug, err := profiles.ParseEngineProfileSlug(rawProfileSlug)
		if err != nil {
			return in, err
		}
		in.EngineProfileSlug = profileSlug
	}
	return in, nil
}

func (m *moduleRuntime) resolveInferenceSettingsFromRegistry(registry profiles.RegistryReader, sources []string, in profiles.ResolveInput) goja.Value {
	resolved, err := registry.ResolveEngineProfile(context.Background(), in)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	settings, err := m.effectiveInferenceSettingsForResolvedProfile(resolved)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	provenance := provenanceFromResolved(resolved, sources)
	return m.newInferenceSettingsObject(m.newInferenceSettingsRef(settings, provenance))
}
