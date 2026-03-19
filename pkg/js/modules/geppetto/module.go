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
	"github.com/rs/zerolog/log"
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
	Runner                   runtimeowner.Runner
	GoToolRegistry           tools.ToolRegistry
	GoMiddlewareFactories    map[string]MiddlewareFactory
	EngineProfileRegistry    profiles.RegistryReader
	DefaultInferenceSettings *aistepssettings.InferenceSettings
	UseDefaultProfileResolve bool
	DefaultProfileResolve    profiles.ResolveInput
	MiddlewareSchemas        middlewarecfg.DefinitionRegistry
	ExtensionCodecs          profiles.ExtensionCodecRegistry
	ExtensionSchemas         map[string]map[string]any
	DefaultEventSinks        []events.EventSink
	DefaultSnapshotHook      toolloop.SnapshotHook
	DefaultPersister         enginebuilder.TurnPersister
	Logger                   zerolog.Logger
}

// Register registers the geppetto native module on a require registry.
func Register(reg *require.Registry, opts Options) {
	if reg == nil {
		return
	}
	mod := &module{opts: opts}
	reg.RegisterNativeModule(ModuleName, mod.Loader)
}

type module struct {
	opts Options
}

type moduleRuntime struct {
	vm     *goja.Runtime
	runner runtimeowner.Runner
	bridge *runtimebridge.Bridge

	logger zerolog.Logger

	goToolRegistry                  tools.ToolRegistry
	goMiddlewareFactories           map[string]MiddlewareFactory
	defaultInferenceSettings        *aistepssettings.InferenceSettings
	profileRegistry                 profiles.RegistryReader
	profileRegistryCloser           io.Closer
	profileRegistryOwned            bool
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
		lg = log.Logger
	}
	m := &moduleRuntime{
		vm:                       vm,
		runner:                   opts.Runner,
		logger:                   lg,
		goToolRegistry:           opts.GoToolRegistry,
		goMiddlewareFactories:    map[string]MiddlewareFactory{},
		defaultInferenceSettings: cloneInferenceSettings(opts.DefaultInferenceSettings),
		profileRegistry:          opts.EngineProfileRegistry,
		useDefaultProfileResolve: opts.UseDefaultProfileResolve,
		defaultProfileResolve:    opts.DefaultProfileResolve,
		middlewareSchemas:        opts.MiddlewareSchemas,
		extensionCodecs:          opts.ExtensionCodecs,
		extensionSchemas:         cloneNestedStringAnyMap(opts.ExtensionSchemas),
		defaultEventSinks:        append([]events.EventSink(nil), opts.DefaultEventSinks...),
		defaultSnapshotHook:      opts.DefaultSnapshotHook,
		defaultPersister:         opts.DefaultPersister,
	}
	if closer, ok := opts.EngineProfileRegistry.(io.Closer); ok && closer != nil {
		m.profileRegistryCloser = closer
	}
	m.baseEngineProfileRegistry = m.profileRegistry
	m.baseEngineProfileRegistryCloser = m.profileRegistryCloser
	if m.runner != nil {
		m.bridge = runtimebridge.New(m.runner)
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
	m.mustSet(exports, "createBuilder", m.createBuilder)
	m.mustSet(exports, "createSession", m.createSession)
	m.mustSet(exports, "runInference", m.runInference)

	turnsObj := m.vm.NewObject()
	m.mustSet(turnsObj, "normalize", m.turnsNormalize)
	m.mustSet(turnsObj, "newTurn", m.turnsNewTurn)
	m.mustSet(turnsObj, "appendBlock", m.turnsAppendBlock)
	m.mustSet(turnsObj, "newUserBlock", m.turnsNewUserBlock)
	m.mustSet(turnsObj, "newSystemBlock", m.turnsNewSystemBlock)
	m.mustSet(turnsObj, "newAssistantBlock", m.turnsNewAssistantBlock)
	m.mustSet(turnsObj, "newToolCallBlock", m.turnsNewToolCallBlock)
	m.mustSet(turnsObj, "newToolUseBlock", m.turnsNewToolUseBlock)
	m.mustSet(exports, "turns", turnsObj)

	enginesObj := m.vm.NewObject()
	m.mustSet(enginesObj, "echo", m.engineEcho)
	m.mustSet(enginesObj, "fromConfig", m.engineFromConfig)
	m.mustSet(enginesObj, "fromProfile", m.engineFromProfile)
	m.mustSet(enginesObj, "fromResolvedProfile", m.engineFromResolvedProfile)
	m.mustSet(enginesObj, "fromFunction", m.engineFromFunction)
	m.mustSet(exports, "engines", enginesObj)

	profilesObj := m.vm.NewObject()
	m.mustSet(profilesObj, "listRegistries", m.profilesListRegistries)
	m.mustSet(profilesObj, "getRegistry", m.profilesGetRegistry)
	m.mustSet(profilesObj, "listProfiles", m.profilesListEngineProfiles)
	m.mustSet(profilesObj, "getProfile", m.profilesGetEngineProfile)
	m.mustSet(profilesObj, "resolve", m.profilesResolve)
	m.mustSet(profilesObj, "connectStack", m.profilesConnectStack)
	m.mustSet(profilesObj, "disconnectStack", m.profilesDisconnectStack)
	m.mustSet(profilesObj, "getConnectedSources", m.profilesGetConnectedSources)
	m.mustSet(exports, "profiles", profilesObj)

	runnerObj := m.vm.NewObject()
	m.mustSet(runnerObj, "resolveRuntime", m.runnerResolveRuntime)
	m.mustSet(runnerObj, "prepare", m.runnerPrepare)
	m.mustSet(runnerObj, "run", m.runnerRun)
	m.mustSet(runnerObj, "start", m.runnerStart)
	m.mustSet(exports, "runner", runnerObj)

	schemasObj := m.vm.NewObject()
	m.mustSet(schemasObj, "listMiddlewares", m.schemasListMiddlewares)
	m.mustSet(schemasObj, "listExtensions", m.schemasListExtensions)
	m.mustSet(exports, "schemas", schemasObj)

	mwsObj := m.vm.NewObject()
	m.mustSet(mwsObj, "fromJS", m.middlewareFromJS)
	m.mustSet(mwsObj, "go", m.middlewareFromGo)
	m.mustSet(exports, "middlewares", mwsObj)

	eventsObj := m.vm.NewObject()
	m.mustSet(eventsObj, "collector", m.eventsCollector)
	m.mustSet(exports, "events", eventsObj)

	toolsObj := m.vm.NewObject()
	m.mustSet(toolsObj, "createRegistry", m.toolsCreateRegistry)
	m.mustSet(exports, "tools", toolsObj)
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
