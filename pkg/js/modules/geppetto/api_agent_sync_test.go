package geppetto

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/dop251/goja"
	inferenceengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type synchronousToolCallingEngine struct {
	calls atomic.Int64
}

var _ inferenceengine.Engine = (*synchronousToolCallingEngine)(nil)

func (e *synchronousToolCallingEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	callNum := e.calls.Add(1)
	out := &turns.Turn{}
	if t != nil {
		out = t.Clone()
	}
	if callNum == 1 {
		turns.AppendBlock(out, turns.NewToolCallBlock("call-1", "echo", map[string]any{"text": "hello"}))
		return out, nil
	}
	turns.AppendBlock(out, turns.NewAssistantTextBlock("done"))
	return out, nil
}

func TestAgentRunWithJSToolRegistryDoesNotDeadlockOwner(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	_, err := rt.runtimeOwner.Call(context.Background(), "test.syncJSToolRun", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("fakeEngine", &synchronousToolCallingEngine{}); setErr != nil {
			return nil, setErr
		}
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			globalThis.toolCalls = 0;
			const registry = gp.toolRegistry().add({
				name: "echo",
				description: "Echo input text",
				parameters: { type: "object" },
				handler: (args, ctx) => {
					globalThis.toolCalls++;
					globalThis.toolCtxName = ctx.toolName;
					return { echoed: args.text };
				},
			});
			const agent = gp.agent()
				.engine(globalThis.fakeEngine)
				.tool(registry)
				.toolLoop({ maxIterations: 3 })
				.build();
			const session = agent.session().id("sync-tool-test").build();
			const result = session.next().user("please call echo").run({ timeoutMs: 1000 });
			globalThis.finalText = result.text();
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("sync JS tool run failed: %v", err)
	}
	got, err := rt.runtimeOwner.Call(context.Background(), "test.readSyncJSToolRun", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`JSON.stringify({toolCalls: globalThis.toolCalls, toolCtxName: globalThis.toolCtxName, finalText: globalThis.finalText})`)
	})
	if err != nil {
		t.Fatalf("read sync JS tool run result failed: %v", err)
	}
	want := `{"toolCalls":1,"toolCtxName":"echo","finalText":"done"}`
	if got.(goja.Value).String() != want {
		t.Fatalf("sync JS tool run result = %s, want %s", got.(goja.Value).String(), want)
	}
}
