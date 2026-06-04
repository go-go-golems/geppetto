package runtime

import (
	"bytes"
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dop251/goja"
	gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	gojengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
	"github.com/rs/zerolog"
)

type runtimeInitFunc struct {
	id string
	fn func(ctx *gojengine.RuntimeInitializationContext) error
}

func (f runtimeInitFunc) ID() string { return f.id }

func (f runtimeInitFunc) InitRuntime(ctx *gojengine.RuntimeInitializationContext) error {
	if f.fn == nil {
		return nil
	}
	return f.fn(ctx)
}

func TestNewRuntime_IncludeDefaultModulesFalseOnlyRegistersGeppetto(t *testing.T) {
	rt, err := NewRuntime(context.Background(), Options{})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	if _, err := rt.VM.RunString(`require("geppetto")`); err != nil {
		t.Fatalf("geppetto module missing: %v", err)
	}
	if _, err := rt.VM.RunString(`require("path")`); err == nil {
		t.Fatalf("require(path) succeeded with IncludeDefaultModules=false, want missing module")
	}
}

func TestNewRuntime_IncludeDefaultModulesTrueRegistersDefaultModules(t *testing.T) {
	rt, err := NewRuntime(context.Background(), Options{IncludeDefaultModules: true})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	value, err := rt.VM.RunString(`require("path").join("a", "b")`)
	if err != nil {
		t.Fatalf("path default module missing: %v", err)
	}
	if got := value.String(); got != "a/b" {
		t.Fatalf("path.join result = %q, want %q", got, "a/b")
	}
}

func TestNewRuntime_StartupContextDoesNotOwnRuntimeLifetime(t *testing.T) {
	startupCtx, cancel := context.WithCancel(context.Background())
	rt, err := NewRuntime(startupCtx, Options{})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	cancel()

	select {
	case <-rt.Context().Done():
		t.Fatalf("runtime lifetime was canceled when startup context was canceled")
	default:
	}

	if _, err := rt.VM.RunString(`require("geppetto")`); err != nil {
		t.Fatalf("geppetto module unusable after startup context cancel: %v", err)
	}
}

func TestNewRuntime_SkipsNilRuntimeInitializers(t *testing.T) {
	var initCalls atomic.Int64

	rt, err := NewRuntime(context.Background(), Options{
		RuntimeInitializers: []gojengine.RuntimeInitializer{
			nil,
			runtimeInitFunc{
				id: "real-init",
				fn: func(ctx *gojengine.RuntimeInitializationContext) error {
					initCalls.Add(1)
					return ctx.VM.Set("runtimeInitWorked", true)
				},
			},
			nil,
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	value, err := rt.VM.RunString(`runtimeInitWorked === true`)
	if err != nil {
		t.Fatalf("js run failed: %v", err)
	}
	if got := value.ToBoolean(); !got {
		t.Fatalf("runtimeInitWorked = false, want true")
	}
	if initCalls.Load() != 1 {
		t.Fatalf("init calls = %d, want 1", initCalls.Load())
	}
}

func TestNewRuntime_DefaultJSEventsInitializerLogsListenerErrors(t *testing.T) {
	var logs bytes.Buffer
	logger := zerolog.New(&logs)
	rt, err := NewRuntime(context.Background(), Options{
		IncludeDefaultModules: true,
		ModuleOptions:         gp.Options{Logger: logger},
	})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()
	manager, ok := jsevents.FromRuntime(rt)
	if !ok {
		t.Fatalf("jsevents manager was not installed")
	}
	ret, err := rt.Owner.Call(context.Background(), "test.setupThrowingEmitter", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			const EventEmitter = require("events");
			globalThis.throwingEmitter = new EventEmitter();
			globalThis.throwingEmitter.on("boom", () => { throw new Error("listener exploded"); });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return manager.AdoptEmitterOnOwner(vm.Get("throwingEmitter"))
	})
	if err != nil {
		t.Fatalf("setup throwing emitter failed: %v", err)
	}
	ref, ok := ret.(*jsevents.EmitterRef)
	if !ok || ref == nil {
		t.Fatalf("setup returned %T, want *jsevents.EmitterRef", ret)
	}
	if err := ref.EmitWithBuilder(context.Background(), "boom", func(vm *goja.Runtime) ([]goja.Value, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	for i := 0; i < 100; i++ {
		if err := rt.Owner.WaitIdle(context.Background()); err != nil {
			t.Fatalf("WaitIdle failed: %v", err)
		}
		logText := logs.String()
		if strings.Contains(logText, "EventEmitter listener dispatch failed") && strings.Contains(logText, "listener exploded") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("logs did not include listener failure: %s", logs.String())
}

func TestNewRuntime_RegistersGeppettoAndCustomBindings(t *testing.T) {
	var reducerCalls atomic.Int64

	rt, err := NewRuntime(context.Background(), Options{
		RuntimeInitializers: []gojengine.RuntimeInitializer{
			runtimeInitFunc{
				id: "pinocchio-like-bindings",
				fn: func(ctx *gojengine.RuntimeInitializationContext) error {
					return ctx.VM.Set("registerSemReducer", func(call goja.FunctionCall) goja.Value {
						reducerCalls.Add(1)
						return goja.Undefined()
					})
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	if _, err := rt.VM.RunString(`
		const gp = require("geppetto");
		if (!gp || typeof gp.agent !== "function") {
			throw new Error("missing geppetto binding");
		}
		if (Object.prototype.hasOwnProperty.call(gp, "turn")) {
			throw new Error("gp.turn must not be public");
		}
		registerSemReducer("llm.delta", function(ev) { return ev; });
	`); err != nil {
		t.Fatalf("js run failed: %v", err)
	}

	if reducerCalls.Load() != 1 {
		t.Fatalf("registerSemReducer calls = %d, want 1", reducerCalls.Load())
	}
}
