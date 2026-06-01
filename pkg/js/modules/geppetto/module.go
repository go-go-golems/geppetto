package geppetto

import (
	"fmt"
	"io"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/js/runtimebridge"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

const (
	// ModuleName is the name used from JavaScript: require("geppetto").
	ModuleName = "geppetto"
	// hiddenRefKey stores Go references on JS objects created by this module.
	hiddenRefKey = "__geppetto_ref"
)

// MiddlewareFactory resolves a named Go middleware from JS options.
type MiddlewareFactory func(options map[string]any) (middleware.Middleware, error)

// Options configures module behavior for a specific runtime.
type Options struct {
	RuntimeOwner              runtimeowner.RuntimeOwner
	GoToolRegistry            tools.ToolRegistry
	GoMiddlewareFactories     map[string]MiddlewareFactory
	EngineProfileRegistry     profiles.RegistryReader
	EngineProfileRegistrySpec []string
	DefaultInferenceSettings  *aistepssettings.InferenceSettings
	UseDefaultProfileResolve  bool
	DefaultProfileResolve     profiles.ResolveInput
	MiddlewareSchemas         middlewarecfg.DefinitionRegistry
	ExtensionCodecs           profiles.ExtensionCodecRegistry
	ExtensionSchemas          map[string]map[string]any
	DefaultEventSinks         []events.EventSink
	DefaultSnapshotHook       toolloop.SnapshotHook
	DefaultPersister          enginebuilder.TurnPersister
	Logger                    zerolog.Logger
}

// NewLoader returns the native geppetto module loader for use with a require
// registry or an xgoja provider wrapper.
func NewLoader(opts Options) require.ModuleLoader {
	mod := &module{opts: opts}
	return mod.Loader
}

// Register registers the geppetto native module on a require registry.
func Register(reg *require.Registry, opts Options) {
	if reg == nil {
		return
	}
	reg.RegisterNativeModule(ModuleName, NewLoader(opts))
}

type module struct {
	opts Options
}

type moduleRuntime struct {
	vm           *goja.Runtime
	runtimeOwner runtimeowner.RuntimeOwner
	bridge       *runtimebridge.Bridge

	logger zerolog.Logger

	goToolRegistry                  tools.ToolRegistry
	goMiddlewareFactories           map[string]MiddlewareFactory
	defaultInferenceSettings        *aistepssettings.InferenceSettings
	profileRegistry                 profiles.RegistryReader
	profileRegistryCloser           io.Closer
	profileRegistrySpec             []string
	baseEngineProfileRegistry       profiles.RegistryReader
	baseEngineProfileRegistryCloser io.Closer
	baseEngineProfileRegistrySpec   []string
	useDefaultProfileResolve        bool
	defaultProfileResolve           profiles.ResolveInput
	middlewareSchemas               middlewarecfg.DefinitionRegistry
	extensionCodecs                 profiles.ExtensionCodecRegistry
	extensionSchemas                map[string]map[string]any
	defaultEventSinks               []events.EventSink
	defaultSnapshotHook             toolloop.SnapshotHook
	defaultPersister                enginebuilder.TurnPersister
}

func newRuntime(vm *goja.Runtime, opts Options) *moduleRuntime {
	lg := opts.Logger
	if lg.GetLevel() == zerolog.NoLevel {
		lg = zlog.Logger
	}
	m := &moduleRuntime{
		vm:                            vm,
		runtimeOwner:                  opts.RuntimeOwner,
		logger:                        lg,
		goToolRegistry:                opts.GoToolRegistry,
		goMiddlewareFactories:         map[string]MiddlewareFactory{},
		defaultInferenceSettings:      cloneInferenceSettings(opts.DefaultInferenceSettings),
		profileRegistry:               opts.EngineProfileRegistry,
		profileRegistrySpec:           append([]string(nil), opts.EngineProfileRegistrySpec...),
		baseEngineProfileRegistrySpec: append([]string(nil), opts.EngineProfileRegistrySpec...),
		useDefaultProfileResolve:      opts.UseDefaultProfileResolve,
		defaultProfileResolve:         opts.DefaultProfileResolve,
		middlewareSchemas:             opts.MiddlewareSchemas,
		extensionCodecs:               opts.ExtensionCodecs,
		extensionSchemas:              cloneNestedStringAnyMap(opts.ExtensionSchemas),
		defaultEventSinks:             append([]events.EventSink(nil), opts.DefaultEventSinks...),
		defaultSnapshotHook:           opts.DefaultSnapshotHook,
		defaultPersister:              opts.DefaultPersister,
	}
	if closer, ok := opts.EngineProfileRegistry.(io.Closer); ok && closer != nil {
		m.profileRegistryCloser = closer
	}
	m.baseEngineProfileRegistry = m.profileRegistry
	m.baseEngineProfileRegistryCloser = m.profileRegistryCloser
	if m.runtimeOwner != nil {
		m.bridge = runtimebridge.New(m.runtimeOwner)
	}
	for k, v := range defaultGoMiddlewareFactories(lg) {
		m.goMiddlewareFactories[k] = v
	}
	for k, v := range opts.GoMiddlewareFactories {
		m.goMiddlewareFactories[k] = v
	}
	return m
}

// Loader is the native-module entrypoint for goja_nodejs require.
func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	rt := newRuntime(vm, m.opts)
	exports := moduleObj.Get("exports").(*goja.Object)
	rt.installExports(exports)
}

func (m *moduleRuntime) installExports(exports *goja.Object) {
	m.mustSet(exports, "version", "0.1.0")
	m.installConsts(exports)

	inferenceProfilesObj := m.vm.NewObject()
	m.mustSet(inferenceProfilesObj, "load", m.inferenceProfilesLoad)
	m.mustSet(inferenceProfilesObj, "resolve", m.inferenceProfilesResolve)
	m.mustSet(inferenceProfilesObj, "default", m.inferenceProfilesDefault)
	m.mustSet(exports, "inferenceProfiles", inferenceProfilesObj)
	m.mustSet(exports, "engine", m.engineBuilder)
	m.mustSet(exports, "agent", m.agentBuilder)
	m.mustSet(exports, "turn", m.turnBuilder)
	m.mustSet(exports, "tool", m.toolBuilder)
	m.mustSet(exports, "toolRegistry", m.toolRegistryBuilder)
	m.installSchemaNamespace(exports)

	eventsObj := m.vm.NewObject()
	m.mustSet(eventsObj, "collector", m.eventsCollector)
	m.mustSet(exports, "events", eventsObj)
}

func (m *moduleRuntime) mustSet(o *goja.Object, key string, v any) {
	if err := o.Set(key, v); err != nil {
		panic(m.vm.NewGoError(fmt.Errorf("set %s: %w", key, err)))
	}
}

func (m *moduleRuntime) attachRef(o *goja.Object, ref any) {
	// Set the value first so goja stores the Go pointer as-is (m.vm.ToValue
	// would wrap struct pointers in a proxy whose Export() returns a map).
	// Then redefine the property to make it non-enumerable/non-writable/non-configurable.
	_ = o.Set(hiddenRefKey, ref)
	_ = o.DefineDataProperty(hiddenRefKey, o.Get(hiddenRefKey),
		goja.FLAG_FALSE, // writable
		goja.FLAG_FALSE, // enumerable
		goja.FLAG_FALSE, // configurable
	)
}

func (m *moduleRuntime) getRef(v goja.Value) any {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	if obj, ok := v.(*goja.Object); ok {
		raw := obj.Get(hiddenRefKey)
		if raw != nil && !goja.IsUndefined(raw) && !goja.IsNull(raw) {
			return raw.Export()
		}
	}
	return v.Export()
}

func cloneInferenceSettings(in *aistepssettings.InferenceSettings) *aistepssettings.InferenceSettings {
	if in == nil {
		return nil
	}
	return in.Clone()
}
