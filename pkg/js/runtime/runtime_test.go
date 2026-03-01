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
