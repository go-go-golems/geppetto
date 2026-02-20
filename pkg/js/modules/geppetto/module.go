package geppetto

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
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
	Loop                  *eventloop.EventLoop
	GoToolRegistry        tools.ToolRegistry
	GoMiddlewareFactories map[string]MiddlewareFactory
	Logger                zerolog.Logger
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
	vm   *goja.Runtime
	loop *eventloop.EventLoop

	logger zerolog.Logger

	goToolRegistry        tools.ToolRegistry
	goMiddlewareFactories map[string]MiddlewareFactory
}

func newRuntime(vm *goja.Runtime, opts Options) *moduleRuntime {
	lg := opts.Logger
	if lg.GetLevel() == zerolog.NoLevel {
		lg = log.Logger
	}
	m := &moduleRuntime{
		vm:                    vm,
		loop:                  opts.Loop,
		logger:                lg,
		goToolRegistry:        opts.GoToolRegistry,
		goMiddlewareFactories: map[string]MiddlewareFactory{},
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
