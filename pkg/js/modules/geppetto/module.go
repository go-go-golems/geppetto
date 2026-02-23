package geppetto

import (
	"fmt"
	"io"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/js/runtimebridge"
	"github.com/go-go-golems/geppetto/pkg/profiles"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	// ModuleName is the name used from JavaScript: require("geppetto").
	ModuleName = "geppetto"
	// PluginsModuleName is the name for shared plugin contract helpers:
	// require("geppetto/plugins").
	PluginsModuleName = ModuleName + "/plugins"
	// hiddenRefKey stores Go references on JS objects created by this module.
	hiddenRefKey = "__geppetto_ref"
)

// MiddlewareFactory resolves a named Go middleware from JS options.
type MiddlewareFactory func(options map[string]any) (middleware.Middleware, error)

// Options configures module behavior for a specific runtime.
type Options struct {
	Runner                runtimeowner.Runner
	GoToolRegistry        tools.ToolRegistry
	GoMiddlewareFactories map[string]MiddlewareFactory
	ProfileRegistry       profiles.RegistryReader
	ProfileRegistryWriter profiles.RegistryWriter
	MiddlewareSchemas     middlewarecfg.DefinitionRegistry
	ExtensionCodecs       profiles.ExtensionCodecRegistry
	ExtensionSchemas      map[string]map[string]any
	DefaultEventSinks     []events.EventSink
	DefaultSnapshotHook   toolloop.SnapshotHook
	DefaultPersister      enginebuilder.TurnPersister
	Logger                zerolog.Logger
}

// Register registers the geppetto native module on a require registry.
func Register(reg *require.Registry, opts Options) {
	if reg == nil {
		return
	}
	mod := &module{opts: opts}
	reg.RegisterNativeModule(ModuleName, mod.Loader)
	reg.RegisterNativeModule(PluginsModuleName, mod.pluginsLoader)
}

type module struct {
	opts Options
}

type moduleRuntime struct {
	vm     *goja.Runtime
	runner runtimeowner.Runner
	bridge *runtimebridge.Bridge

	logger zerolog.Logger

	goToolRegistry            tools.ToolRegistry
	goMiddlewareFactories     map[string]MiddlewareFactory
	profileRegistry           profiles.RegistryReader
	profileRegistryWriter     profiles.RegistryWriter
	profileRegistryCloser     io.Closer
	profileRegistryOwned      bool
	profileRegistrySpec       []string
	baseProfileRegistry       profiles.RegistryReader
	baseProfileRegistryWriter profiles.RegistryWriter
	baseProfileRegistryCloser io.Closer
	baseProfileRegistrySpec   []string
	middlewareSchemas         middlewarecfg.DefinitionRegistry
	extensionCodecs           profiles.ExtensionCodecRegistry
	extensionSchemas          map[string]map[string]any
	defaultEventSinks         []events.EventSink
	defaultSnapshotHook       toolloop.SnapshotHook
	defaultPersister          enginebuilder.TurnPersister
}

func newRuntime(vm *goja.Runtime, opts Options) *moduleRuntime {
	lg := opts.Logger
	if lg.GetLevel() == zerolog.NoLevel {
		lg = log.Logger
	}
	m := &moduleRuntime{
		vm:                    vm,
		runner:                opts.Runner,
		logger:                lg,
		goToolRegistry:        opts.GoToolRegistry,
		goMiddlewareFactories: map[string]MiddlewareFactory{},
		profileRegistry:       opts.ProfileRegistry,
		profileRegistryWriter: opts.ProfileRegistryWriter,
		middlewareSchemas:     opts.MiddlewareSchemas,
		extensionCodecs:       opts.ExtensionCodecs,
		extensionSchemas:      cloneNestedStringAnyMap(opts.ExtensionSchemas),
		defaultEventSinks:     append([]events.EventSink(nil), opts.DefaultEventSinks...),
		defaultSnapshotHook:   opts.DefaultSnapshotHook,
		defaultPersister:      opts.DefaultPersister,
	}
	if closer, ok := opts.ProfileRegistry.(io.Closer); ok && closer != nil {
		m.profileRegistryCloser = closer
	}
	if m.profileRegistryWriter == nil {
		if rw, ok := opts.ProfileRegistry.(profiles.RegistryWriter); ok {
			m.profileRegistryWriter = rw
		}
	}
	m.baseProfileRegistry = m.profileRegistry
	m.baseProfileRegistryWriter = m.profileRegistryWriter
	m.baseProfileRegistryCloser = m.profileRegistryCloser
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
	m.mustSet(enginesObj, "fromProfile", m.engineFromProfile)
	m.mustSet(enginesObj, "fromConfig", m.engineFromConfig)
	m.mustSet(enginesObj, "fromFunction", m.engineFromFunction)
	m.mustSet(exports, "engines", enginesObj)

	profilesObj := m.vm.NewObject()
	m.mustSet(profilesObj, "listRegistries", m.profilesListRegistries)
	m.mustSet(profilesObj, "getRegistry", m.profilesGetRegistry)
	m.mustSet(profilesObj, "listProfiles", m.profilesListProfiles)
	m.mustSet(profilesObj, "getProfile", m.profilesGetProfile)
	m.mustSet(profilesObj, "resolve", m.profilesResolve)
	m.mustSet(profilesObj, "createProfile", m.profilesCreateProfile)
	m.mustSet(profilesObj, "updateProfile", m.profilesUpdateProfile)
	m.mustSet(profilesObj, "deleteProfile", m.profilesDeleteProfile)
	m.mustSet(profilesObj, "setDefaultProfile", m.profilesSetDefaultProfile)
	m.mustSet(profilesObj, "connectStack", m.profilesConnectStack)
	m.mustSet(profilesObj, "disconnectStack", m.profilesDisconnectStack)
	m.mustSet(profilesObj, "getConnectedSources", m.profilesGetConnectedSources)
	m.mustSet(exports, "profiles", profilesObj)

	schemasObj := m.vm.NewObject()
	m.mustSet(schemasObj, "listMiddlewares", m.schemasListMiddlewares)
	m.mustSet(schemasObj, "listExtensions", m.schemasListExtensions)
	m.mustSet(exports, "schemas", schemasObj)

	mwsObj := m.vm.NewObject()
	m.mustSet(mwsObj, "fromJS", m.middlewareFromJS)
	m.mustSet(mwsObj, "go", m.middlewareFromGo)
	m.mustSet(exports, "middlewares", mwsObj)

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
