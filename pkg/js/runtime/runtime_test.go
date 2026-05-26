package runtime

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/dop251/goja"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
)

type runtimeInitFunc struct {
	id string
	fn func(ctx *gojengine.RuntimeContext) error
}

func (f runtimeInitFunc) ID() string { return f.id }

func (f runtimeInitFunc) InitRuntime(ctx *gojengine.RuntimeContext) error {
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

func TestNewRuntime_SkipsNilRuntimeInitializers(t *testing.T) {
	var initCalls atomic.Int64

	rt, err := NewRuntime(context.Background(), Options{
		RuntimeInitializers: []gojengine.RuntimeInitializer{
			nil,
			runtimeInitFunc{
				id: "real-init",
				fn: func(ctx *gojengine.RuntimeContext) error {
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

func TestNewRuntime_RegistersGeppettoAndCustomBindings(t *testing.T) {
	var reducerCalls atomic.Int64

	rt, err := NewRuntime(context.Background(), Options{
		RuntimeInitializers: []gojengine.RuntimeInitializer{
			runtimeInitFunc{
				id: "pinocchio-like-bindings",
				fn: func(ctx *gojengine.RuntimeContext) error {
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
		if (!gp || typeof gp.createSession !== "function") {
			throw new Error("missing geppetto binding");
		}
		registerSemReducer("llm.delta", function(ev) { return ev; });
	`); err != nil {
		t.Fatalf("js run failed: %v", err)
	}

	if reducerCalls.Load() != 1 {
		t.Fatalf("registerSemReducer calls = %d, want 1", reducerCalls.Load())
	}
}
