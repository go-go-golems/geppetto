package geppetto

import (
	"os"
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

func TestConstsExported(t *testing.T) {
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");
		if (!gp.consts) throw new Error("missing consts export");
		if (gp.consts.ToolChoice.AUTO !== "auto") throw new Error("ToolChoice.AUTO mismatch");
		if (gp.consts.ToolChoice.NONE !== "none") throw new Error("ToolChoice.NONE mismatch");
		if (gp.consts.ToolErrorHandling.RETRY !== "retry") throw new Error("ToolErrorHandling.RETRY mismatch");
		if (gp.consts.BlockKind.TOOL_USE !== "tool_use") throw new Error("BlockKind.TOOL_USE mismatch");
		if (gp.consts.MetadataKeys.SESSION_ID !== "session_id") throw new Error("MetadataKeys.SESSION_ID mismatch");
		if (gp.consts.EventType.TOOL_RESULT !== "tool-result") throw new Error("EventType.TOOL_RESULT mismatch");
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

func TestSessionHistoryInspectionAndSnapshotImmutability(t *testing.T) {
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");
		const s = gp.createSession({ engine: gp.engines.echo({ reply: "ACK" }) });

		s.append(gp.turns.newTurn({ id: "t1", blocks: [gp.turns.newUserBlock("one")] }));
		s.run();
		s.append(gp.turns.newTurn({ id: "t2", blocks: [gp.turns.newUserBlock("two")] }));
		s.run();

		if (s.turnCount() !== 2) throw new Error("turnCount should be 2");

		const h = s.turns();
		if (!Array.isArray(h) || h.length !== 2) throw new Error("turns() length mismatch");

		const first = s.getTurn(0);
		const missing = s.getTurn(5);
		if (!first || first.id !== "t1") throw new Error("getTurn(0) mismatch");
		if (missing !== null) throw new Error("getTurn out-of-range should be null");

		const range = s.turnsRange(1, 2);
		if (!Array.isArray(range) || range.length !== 1 || range[0].id !== "t2") throw new Error("turnsRange mismatch");

		// Mutating returned snapshots must not mutate session internal history.
		h[0].id = "mutated-id";
		h[0].blocks.push(gp.turns.newAssistantBlock("mutated"));
		const firstAgain = s.getTurn(0);
		if (firstAgain.id !== "t1") throw new Error("history snapshot mutability leak on id");
		if (firstAgain.blocks.length !== 2) throw new Error("history snapshot mutability leak on blocks");
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

func TestToolLoopEnumValidation(t *testing.T) {
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");
		const reg = gp.tools.createRegistry();
		reg.register({
			name: "noop",
			description: "noop",
			handler: () => ({ ok: true })
		});
		const eng = gp.engines.echo({ reply: "OK" });

		let threwChoice = false;
		try {
			gp.createBuilder().withEngine(eng).withTools(reg, {
				enabled: true,
				toolChoice: "bad-choice"
			});
		} catch (e) {
			threwChoice = /invalid toolChoice/i.test(String(e));
		}
		if (!threwChoice) throw new Error("expected invalid toolChoice to throw");

		let threwHandling = false;
		try {
			gp.createBuilder().withEngine(eng).withTools(reg, {
				enabled: true,
				toolErrorHandling: "explode"
			});
		} catch (e) {
			threwHandling = /invalid toolErrorHandling/i.test(String(e));
		}
		if (!threwHandling) throw new Error("expected invalid toolErrorHandling to throw");
	`)
}

func TestEnginesFromProfileAndFromConfigResolution(t *testing.T) {
	t.Setenv("PINOCCHIO_PROFILE", "env-profile-model")
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");

		const explicit = gp.engines.fromProfile("explicit-model", {
			profile: "opts-model",
			apiType: "openai",
			apiKey: "test-openai-key"
		});
		if (explicit.name !== "profile:explicit-model") throw new Error("explicit profile precedence mismatch");

		const optsProfile = gp.engines.fromProfile(undefined, {
			profile: "opts-model",
			apiType: "openai",
			apiKey: "test-openai-key"
		});
		if (optsProfile.name !== "profile:opts-model") throw new Error("options profile precedence mismatch");

		const envProfile = gp.engines.fromProfile(undefined, {
			apiType: "openai",
			apiKey: "test-openai-key"
		});
		if (envProfile.name !== "profile:env-profile-model") throw new Error("env profile precedence mismatch");

		const fromConfig = gp.engines.fromConfig({
			apiType: "openai",
			model: "gpt-4o-mini",
			apiKey: "test-openai-key"
		});
		if (fromConfig.name !== "config") throw new Error("fromConfig name mismatch");

		let threw = false;
		try {
			gp.engines.fromConfig({ apiType: "bogus-provider", model: "x", apiKey: "k" });
		} catch (e) {
			threw = true;
		}
		if (!threw) throw new Error("fromConfig should throw for unknown provider");
	`)
}

func TestEngineFromProfileInferenceIntegration_Gemini(t *testing.T) {
	if os.Getenv("GEPPETTO_LIVE_INFERENCE_TESTS") != "1" {
		t.Skip("skipping live inference integration test (set GEPPETTO_LIVE_INFERENCE_TESTS=1 to enable)")
	}
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
		t.Skip("skipping gemini integration test: GEMINI_API_KEY/GOOGLE_API_KEY not set")
	}
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");
		const s = gp.createSession({
			engine: gp.engines.fromProfile("gemini-2.5-flash-lite", {
				apiType: "gemini"
			})
		});
		s.append(gp.turns.newTurn({
			blocks: [gp.turns.newUserBlock("Reply with exactly READY.")]
		}));
		const out = s.run();
		if (!out || !Array.isArray(out.blocks) || out.blocks.length < 2) {
			throw new Error("expected output turn with model response blocks");
		}
	`)
}

func TestOpaqueRefHidden(t *testing.T) {
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");
		const eng = gp.engines.echo({ reply: "test" });

		// __geppetto_ref must not appear in Object.keys()
		const keys = Object.keys(eng);
		if (keys.includes("__geppetto_ref")) {
			throw new Error("__geppetto_ref is enumerable — found in Object.keys(): " + JSON.stringify(keys));
		}

		// __geppetto_ref must not appear in JSON.stringify()
		const json = JSON.stringify(eng);
		if (json.includes("__geppetto_ref")) {
			throw new Error("__geppetto_ref leaks into JSON.stringify: " + json);
		}

		// Overwriting must silently fail (non-writable)
		eng.__geppetto_ref = 42;
		// Engine must still work after overwrite attempt
		const s = gp.createSession({ engine: eng });
		s.append(gp.turns.newTurn({ blocks: [ gp.turns.newUserBlock("hello") ] }));
		const out = s.run();
		const last = out.blocks[out.blocks.length - 1];
		if (!last || last.kind !== "llm_text") {
			throw new Error("engine broken after overwrite attempt");
		}

		// Also verify on session, builder, and tool registry objects
		const builder = gp.createBuilder();
		if (Object.keys(builder).includes("__geppetto_ref")) {
			throw new Error("builder ref is enumerable");
		}

		const reg = gp.tools.createRegistry();
		if (Object.keys(reg).includes("__geppetto_ref")) {
			throw new Error("tool registry ref is enumerable");
		}
	`)
}

func TestToolLoopHooksMutationRetryAbortAndHookPolicy(t *testing.T) {
	vm := newJSRuntime(t, Options{})
	mustRunJS(t, vm, `
		const gp = require("geppetto");

		function makeEngine(toolName) {
			return gp.engines.fromFunction((turn) => {
				const hasToolUse = turn.blocks.some(b => b.kind === "tool_use");
				if (!hasToolUse) {
					turn.blocks.push(gp.turns.newToolCallBlock("tool-1", toolName, { value: "orig" }));
					return turn;
				}
				turn.blocks.push(gp.turns.newAssistantBlock("done"));
				return turn;
			});
		}

		// Scenario A: before + after hooks mutate args/result.
		const regA = gp.tools.createRegistry();
		regA.register({
			name: "echo_args",
			description: "echo args",
			handler: ({ value }) => ({ seen: value })
		});
		const sA = gp.createBuilder()
			.withEngine(makeEngine("echo_args"))
			.withTools(regA, { enabled: true, maxIterations: 3, toolErrorHandling: "continue" })
			.withToolHooks({
				beforeToolCall: (ctx) => ({ args: { value: "rewritten" } }),
				afterToolCall: (ctx) => ({ result: { post: true, seen: ctx.call.args.value } })
			})
			.buildSession();
		sA.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("calc")] }));
		const outA = sA.run();
		const useA = outA.blocks.find(b => b.kind === "tool_use");
		if (!useA) throw new Error("scenario A: missing tool_use");
		const txtA = String(useA.payload && useA.payload.result || "");
		if (!txtA.includes("rewritten")) throw new Error("scenario A: expected rewritten arg in result");
		if (!txtA.includes("post")) throw new Error("scenario A: expected post-processed result");

		// Scenario B: retry hook drives a second attempt.
		let attemptsB = 0;
		let errorsSeenB = 0;
		const regB = gp.tools.createRegistry();
		regB.register({
			name: "flaky_tool",
			description: "fails once then succeeds",
			handler: ({ value }) => {
				attemptsB++;
				if (attemptsB < 2) throw new Error("transient failure");
				return { ok: true, value, attempts: attemptsB };
			}
		});
		const sB = gp.createBuilder()
			.withEngine(makeEngine("flaky_tool"))
			.withTools(regB, {
				enabled: true,
				maxIterations: 3,
				toolErrorHandling: "retry",
				retryMaxRetries: 5,
				retryBackoffMs: 1
			})
			.withToolHooks({
				onToolError: (ctx) => {
					errorsSeenB++;
					return { retry: true, backoffMs: 0 };
				}
			})
			.buildSession();
		sB.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("retry")] }));
		const outB = sB.run();
		const useB = outB.blocks.find(b => b.kind === "tool_use");
		if (!useB) throw new Error("scenario B: missing tool_use");
		const txtB = String(useB.payload && useB.payload.result || "");
		if (attemptsB !== 2) throw new Error("scenario B: expected exactly 2 attempts, got " + attemptsB);
		if (errorsSeenB < 1) throw new Error("scenario B: expected onToolError hook call");
		if (!txtB.includes("\"attempts\":2")) throw new Error("scenario B: expected successful second attempt");

		// Scenario C: abort action in onToolError disables retries.
		let attemptsC = 0;
		const regC = gp.tools.createRegistry();
		regC.register({
			name: "always_fail",
			description: "always fails",
			handler: () => {
				attemptsC++;
				throw new Error("fail hard");
			}
		});
		const sC = gp.createBuilder()
			.withEngine(makeEngine("always_fail"))
			.withTools(regC, {
				enabled: true,
				maxIterations: 3,
				toolErrorHandling: "retry",
				retryMaxRetries: 5,
				retryBackoffMs: 1
			})
			.withToolHooks({
				onToolError: () => ({ action: "abort" })
			})
			.buildSession();
		sC.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("abort")] }));
		const outC = sC.run();
		const useC = outC.blocks.find(b => b.kind === "tool_use");
		if (!useC) throw new Error("scenario C: missing tool_use");
		if (attemptsC !== 1) throw new Error("scenario C: expected one attempt after abort, got " + attemptsC);

		// Scenario D: hook callback error policy fail-open.
		let attemptsD = 0;
		const regD = gp.tools.createRegistry();
		regD.register({
			name: "ok_tool",
			description: "returns success",
			handler: ({ value }) => {
				attemptsD++;
				return { ok: true, value };
			}
		});
		const sD = gp.createBuilder()
			.withEngine(makeEngine("ok_tool"))
			.withTools(regD, { enabled: true, maxIterations: 3 })
			.withToolHooks({
				failOpen: true,
				beforeToolCall: () => { throw new Error("hook boom"); }
			})
			.buildSession();
		sD.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("fail-open")] }));
		const outD = sD.run();
		const useD = outD.blocks.find(b => b.kind === "tool_use");
		if (!useD) throw new Error("scenario D: missing tool_use");
		if (attemptsD !== 1) throw new Error("scenario D: expected tool to execute in fail-open mode");

		// Scenario E: hook callback error policy fail-closed.
		let attemptsE = 0;
		const regE = gp.tools.createRegistry();
		regE.register({
			name: "blocked_tool",
			description: "would return success",
			handler: ({ value }) => {
				attemptsE++;
				return { ok: true, value };
			}
		});
		const sE = gp.createBuilder()
			.withEngine(makeEngine("blocked_tool"))
			.withTools(regE, { enabled: true, maxIterations: 3 })
			.withToolHooks({
				failOpen: false,
				beforeToolCall: () => { throw new Error("hook closed"); }
			})
			.buildSession();
		sE.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("fail-closed")] }));
		const outE = sE.run();
		const useE = outE.blocks.find(b => b.kind === "tool_use");
		if (!useE) throw new Error("scenario E: missing tool_use");
		if (attemptsE !== 0) throw new Error("scenario E: tool should not execute in fail-closed mode");
		const errE = String(useE.payload && useE.payload.error || "");
		if (!/beforeToolCall hook/i.test(errE)) throw new Error("scenario E: expected beforeToolCall hook error");
	`)
}
