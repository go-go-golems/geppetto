package geppetto

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

func newJSRuntime(t *testing.T, opts Options) *goja.Runtime {
	t.Helper()
	vm := goja.New()
	reg := require.NewRegistry()
	Register(reg, opts)
	reg.Enable(vm)
	return vm
}

func mustRunJS(t *testing.T, vm *goja.Runtime, src string) goja.Value {
	t.Helper()
	v, err := vm.RunString(src)
	if err != nil {
		t.Fatalf("js execution failed: %v\nscript:\n%s", err, src)
	}
	return v
}

func TestTurnsCodecAndHelpers(t *testing.T) {
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");
		const t = gp.turns.newTurn({
			id: "turn-1",
			blocks: [
				gp.turns.newUserBlock("hello"),
				gp.turns.newToolCallBlock("call-1", "echo", {text: "x"})
			],
			metadata: {
				session_id: "s-1",
				trace_id: "trace-1"
			},
			data: {
				tool_config: { enabled: true }
			}
		});
		if (t.id !== "turn-1") throw new Error("turn id mismatch");
		if (!Array.isArray(t.blocks) || t.blocks.length !== 2) throw new Error("block count mismatch");
		if (t.blocks[0].kind !== "user") throw new Error("expected first block kind user");
		if (!t.metadata || t.metadata.session_id !== "s-1") throw new Error("session_id metadata missing");
		if (!t.data || !t.data.tool_config) throw new Error("tool_config data missing");
	`)
}

func TestSessionRunWithEchoEngine(t *testing.T) {
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");
		const eng = gp.engines.echo({ reply: "HELLO-OUT" });
		const s = gp.createSession({ engine: eng });
		s.append(gp.turns.newTurn({ blocks: [ gp.turns.newUserBlock("hello") ] }));
		const out = s.run();
		const last = out.blocks[out.blocks.length - 1];
		if (!last || last.kind !== "llm_text") throw new Error("missing llm_text output");
		if (!last.payload || last.payload.text !== "HELLO-OUT") throw new Error("unexpected llm_text output");
	`)
}

func TestMiddlewareCompositionJSAndGo(t *testing.T) {
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");
		const eng = gp.engines.fromFunction((turn) => {
			turn.blocks.push(gp.turns.newAssistantBlock("ok"));
			return turn;
		});

		const b = gp.createBuilder()
			.withEngine(eng)
			.useGoMiddleware("systemPrompt", { prompt: "SYS-PROMPT" })
			.useMiddleware(gp.middlewares.fromJS((turn, next) => {
				const out = next(turn);
				if (!out.metadata) out.metadata = {};
				out.metadata.trace_id = "js-mw";
				return out;
			}, "trace-mw"));

		const s = b.buildSession();
		s.append(gp.turns.newTurn({ blocks: [ gp.turns.newUserBlock("hello") ] }));
		const out = s.run();
		if (out.blocks[0].kind !== "system") throw new Error("system prompt middleware did not run");
		if (out.blocks[0].payload.text !== "SYS-PROMPT") throw new Error("system prompt text mismatch");
		if (!out.metadata || out.metadata.trace_id !== "js-mw") throw new Error("js middleware metadata missing");
	`)
}

func TestBuilderToolsAndGoToolInvocationFromJS(t *testing.T) {
	type doubleIn struct {
		N int `json:"n" jsonschema:"required"`
	}
	goDef, err := tools.NewToolFromFunc("go_double", "double a number", func(in doubleIn) (map[string]any, error) {
		return map[string]any{"value": in.N * 2}, nil
	})
	if err != nil {
		t.Fatalf("create go tool: %v", err)
	}
	goRegistry := tools.NewInMemoryToolRegistry()
	if err := goRegistry.RegisterTool("go_double", *goDef); err != nil {
		t.Fatalf("register go tool: %v", err)
	}

	vm := newJSRuntime(t, Options{
		GoToolRegistry: goRegistry,
	})

	mustRunJS(t, vm, `
		const gp = require("geppetto");
		const reg = gp.tools.createRegistry();
		reg.register({
			name: "js_add",
			description: "add a + b",
			handler: ({a, b}) => ({ sum: a + b })
		});
		reg.useGoTools(["go_double"]);
		const goDirect = reg.call("go_double", { n: 21 });
		if (!goDirect || goDirect.value !== 42) throw new Error("go tool direct call failed");

		const eng = gp.engines.fromFunction((turn) => {
			const hasToolUse = turn.blocks.some(b => b.kind === "tool_use");
			if (!hasToolUse) {
				turn.blocks.push(gp.turns.newToolCallBlock("tool-1", "js_add", { a: 2, b: 3 }));
				return turn;
			}
			turn.blocks.push(gp.turns.newAssistantBlock("done"));
			return turn;
		});

		const s = gp.createBuilder()
			.withEngine(eng)
			.withTools(reg, { enabled: true, maxIterations: 3, toolChoice: "auto", maxParallelTools: 1 })
			.buildSession();

		s.append(gp.turns.newTurn({ blocks: [ gp.turns.newUserBlock("calc") ] }));
		const out = s.run();
		const toolUse = out.blocks.find(b => b.kind === "tool_use");
		if (!toolUse) throw new Error("missing tool_use block: " + JSON.stringify(out.blocks));
		const resultText = String(toolUse.payload && toolUse.payload.result || "");
		if (!resultText.includes("sum")) throw new Error("tool_use payload missing js result");
	`)
}
