package geppetto

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	gepprofiles "github.com/go-go-golems/geppetto/pkg/profiles"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

type jsRuntime struct {
	vm     *goja.Runtime
	runner runtimeowner.Runner
}

func newJSRuntime(t *testing.T, opts Options) *jsRuntime {
	t.Helper()
	loop := eventloop.NewEventLoop()
	go loop.Start()
	t.Cleanup(func() {
		_ = loop.Stop()
	})

	vm := goja.New()
	opts.Runner = runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
		Name:          "geppetto-js-module-test",
		RecoverPanics: true,
	})
	reg := require.NewRegistry()
	Register(reg, opts)
	reg.Enable(vm)
	return &jsRuntime{
		vm:     vm,
		runner: opts.Runner,
	}
}

func mustRunJS(t *testing.T, rt *jsRuntime, src string) goja.Value {
	t.Helper()
	v, err := rt.vm.RunString(src)
	if err != nil {
		t.Fatalf("js execution failed: %v\nscript:\n%s", err, src)
	}
	return v
}

type promiseSnapshot struct {
	State  goja.PromiseState
	Result any
}

func waitForPromiseExpr(t *testing.T, rt *jsRuntime, expr string, timeout time.Duration) promiseSnapshot {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		ret, err := rt.runner.Call(context.Background(), "module_test.PromiseState", func(_ context.Context, vm *goja.Runtime) (any, error) {
			v, runErr := vm.RunString(expr)
			if runErr != nil {
				return nil, runErr
			}
			if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
				return nil, fmt.Errorf("promise expression %q returned null/undefined", expr)
			}
			p, ok := v.Export().(*goja.Promise)
			if !ok {
				return nil, fmt.Errorf("promise expression %q returned %T", expr, v.Export())
			}
			snap := promiseSnapshot{State: p.State()}
			if p.Result() != nil {
				snap.Result = p.Result().Export()
			}
			return snap, nil
		})
		if err != nil {
			t.Fatalf("failed to inspect promise %q: %v", expr, err)
		}
		snap, ok := ret.(promiseSnapshot)
		if !ok {
			t.Fatalf("unexpected promise snapshot type %T for %q", ret, expr)
		}
		if snap.State == goja.PromiseStateFulfilled {
			return snap
		}
		if snap.State == goja.PromiseStateRejected {
			t.Fatalf("promise %q rejected: %v", expr, snap.Result)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for promise %q to settle", expr)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func mustEvalExprExport(t *testing.T, rt *jsRuntime, expr string) any {
	t.Helper()
	ret, err := rt.runner.Call(context.Background(), "module_test.EvalExpr", func(_ context.Context, vm *goja.Runtime) (any, error) {
		v, runErr := vm.RunString(expr)
		if runErr != nil {
			return nil, runErr
		}
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			return nil, nil
		}
		return v.Export(), nil
	})
	if err != nil {
		t.Fatalf("failed evaluating %q: %v", expr, err)
	}
	return ret
}

func TestTurnsCodecAndHelpers(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
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
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		if (!gp.consts) throw new Error("missing consts export");
		if (gp.consts.ToolChoice.AUTO !== "auto") throw new Error("ToolChoice.AUTO mismatch");
		if (gp.consts.ToolChoice.NONE !== "none") throw new Error("ToolChoice.NONE mismatch");
		if (gp.consts.ToolErrorHandling.RETRY !== "retry") throw new Error("ToolErrorHandling.RETRY mismatch");
		if (gp.consts.BlockKind.TOOL_USE !== "tool_use") throw new Error("BlockKind.TOOL_USE mismatch");
		if (gp.consts.TurnMetadataKeys.SESSION_ID !== "session_id") throw new Error("TurnMetadataKeys.SESSION_ID mismatch");
		if (gp.consts.TurnDataKeys.TOOL_CONFIG !== "tool_config") throw new Error("TurnDataKeys.TOOL_CONFIG mismatch");
		if (gp.consts.BlockMetadataKeys.CLAUDE_ORIGINAL_CONTENT !== "claude_original_content") throw new Error("BlockMetadataKeys.CLAUDE_ORIGINAL_CONTENT mismatch");
		if (gp.consts.RunMetadataKeys.TRACE_ID !== "trace_id") throw new Error("RunMetadataKeys.TRACE_ID mismatch");
		if (gp.consts.PayloadKeys.ENCRYPTED_CONTENT !== "encrypted_content") throw new Error("PayloadKeys.ENCRYPTED_CONTENT mismatch");
		if (gp.consts.EventType.TOOL_RESULT !== "tool-result") throw new Error("EventType.TOOL_RESULT mismatch");
	`)
}

func TestPluginContractHelpersModule(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const plugins = require("geppetto/plugins");
		if (!plugins) throw new Error("missing geppetto/plugins");
		if (plugins.EXTRACTOR_PLUGIN_API_VERSION !== "cozo.extractor/v1") {
			throw new Error("unexpected EXTRACTOR_PLUGIN_API_VERSION");
		}

		const descriptor = plugins.defineExtractorPlugin({
			id: "test.extractor",
			name: "Test Extractor",
			create() {
				return {
					run: plugins.wrapExtractorRun((input) => ({ ok: true, transcript: input.transcript, timeoutMs: input.timeoutMs }))
				};
			}
		});

		const instance = descriptor.create({});
		const result = instance.run({ transcript: "hello world" });
		if (!result || result.ok !== true || result.timeoutMs !== 120000) {
			throw new Error("wrapped run did not normalize timeout");
		}

		let threw = false;
		try {
			instance.run({ transcript: "" });
		} catch (e) {
			threw = /input\.transcript/i.test(String(e));
		}
		if (!threw) {
			throw new Error("expected transcript validation error");
		}
	`)
}

func TestSessionRunWithEchoEngine(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
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
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
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
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
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

	rt := newJSRuntime(t, Options{
		GoToolRegistry: goRegistry,
	})

	mustRunJS(t, rt, `
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
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
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

func TestMiddlewareToolHandlerAndHookReceiveContext(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		let middlewareCtx = null;
		let toolHandlerCtx = null;
		let hookCtx = null;

		const reg = gp.tools.createRegistry();
		reg.register({
			name: "ctx_tool",
			description: "returns context details",
			handler: (args, ctx) => {
				toolHandlerCtx = ctx;
				return { ok: true, callId: ctx && ctx.callId };
			}
		});

		const eng = gp.engines.fromFunction((turn) => {
			const hasToolUse = turn.blocks.some(b => b.kind === "tool_use");
			if (!hasToolUse) {
				turn.blocks.push(gp.turns.newToolCallBlock("ctx-call-1", "ctx_tool", { value: "x" }));
				return turn;
			}
			turn.blocks.push(gp.turns.newAssistantBlock("done"));
			return turn;
		});

		const mw = gp.middlewares.fromJS((turn, next, ctx) => {
			middlewareCtx = ctx;
			return next(turn);
		}, "ctx-mw");

		const s = gp.createBuilder()
			.withEngine(eng)
			.useMiddleware(mw)
			.withTools(reg, { enabled: true, maxIterations: 3, toolChoice: gp.consts.ToolChoice.AUTO })
			.withToolHooks({
				beforeToolCall: (payload) => {
					hookCtx = payload;
				}
			})
			.buildSession();

		s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("start")] }));
		const out = s.run();
		if (!out || !Array.isArray(out.blocks)) throw new Error("expected output turn");

		if (!middlewareCtx || !middlewareCtx.sessionId || !middlewareCtx.inferenceId) {
			throw new Error("middleware context missing sessionId/inferenceId: " + JSON.stringify(middlewareCtx));
		}
		if (!middlewareCtx.turnId) {
			throw new Error("middleware context missing turnId");
		}

		if (!toolHandlerCtx || !toolHandlerCtx.callId) {
			throw new Error("tool handler context missing callId: " + JSON.stringify(toolHandlerCtx));
		}
		if (!toolHandlerCtx.sessionId || !toolHandlerCtx.inferenceId) {
			throw new Error("tool handler context missing sessionId/inferenceId: " + JSON.stringify(toolHandlerCtx));
		}
		if (toolHandlerCtx.callId !== "ctx-call-1") {
			throw new Error("tool handler callId mismatch: " + toolHandlerCtx.callId);
		}

		if (!hookCtx || !hookCtx.sessionId || !hookCtx.inferenceId) {
			throw new Error("hook payload missing sessionId/inferenceId: " + JSON.stringify(hookCtx));
		}
	`)
}

func TestRunOptionsTimeoutAndTags(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		let seenCtx = null;

		const s = gp.createBuilder()
			.withEngine(gp.engines.echo({ reply: "OK" }))
			.useMiddleware(gp.middlewares.fromJS((turn, next, ctx) => {
				seenCtx = ctx;
				return next(turn);
			}, "opts-mw"))
			.buildSession();

		s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("hello")] }));
		const out = s.run(undefined, {
			timeoutMs: 1000,
			tags: { requestId: "req-1", attempt: 2 }
		});
		if (!out || !Array.isArray(out.blocks)) throw new Error("expected output turn");
		if (!seenCtx || !seenCtx.deadlineMs) throw new Error("expected deadlineMs in middleware ctx");
		if (!seenCtx.tags || seenCtx.tags.requestId !== "req-1" || seenCtx.tags.attempt !== 2) {
			throw new Error("expected run tags in middleware ctx: " + JSON.stringify(seenCtx));
		}

		let badTimeout = false;
		try {
			s.run(undefined, { timeoutMs: -1 });
		} catch (e) {
			badTimeout = /timeoutMs/i.test(String(e));
		}
		if (!badTimeout) throw new Error("expected negative timeoutMs to throw");
	`)
}

func TestSessionStartReturnsRunHandle(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const s = gp.createSession({ engine: gp.engines.echo({ reply: "OK" }) });

		const handle = s.start(undefined, {
			timeoutMs: 1000,
			tags: { mode: "start-test" }
		});
		if (!handle || typeof handle !== "object") throw new Error("start() should return object");
		if (!handle.promise || typeof handle.promise.then !== "function") throw new Error("missing promise");
		if (typeof handle.cancel !== "function") throw new Error("missing cancel()");
		if (typeof handle.on !== "function") throw new Error("missing on()");

		const chained = handle.on("*", () => {});
		if (chained !== handle) throw new Error("on() should return run handle for chaining");
	`)
}

func TestRunAsyncWithJSEngineAndMiddleware(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		globalThis.__seenAsync = { engine: false, middleware: false };

		const eng = gp.engines.fromFunction((turn) => {
			globalThis.__seenAsync.engine = true;
			turn.blocks.push(gp.turns.newAssistantBlock("ASYNC-OK"));
			return turn;
		});
		const mw = gp.middlewares.fromJS((turn, next) => {
			globalThis.__seenAsync.middleware = true;
			return next(turn);
		}, "async-mw");

		const s = gp.createBuilder()
			.withEngine(eng)
			.useMiddleware(mw)
			.buildSession();
		s.append(gp.turns.newTurn({ blocks: [ gp.turns.newUserBlock("hello") ] }));
		globalThis.__runAsyncPromise = s.runAsync();
	`)

	snap := waitForPromiseExpr(t, rt, "__runAsyncPromise", 2*time.Second)
	if snap.State != goja.PromiseStateFulfilled {
		t.Fatalf("expected fulfilled promise, got %v", snap.State)
	}
	if turn, ok := snap.Result.(map[string]any); !ok || len(turn) == 0 {
		t.Fatalf("expected non-empty turn result map, got %T (%v)", snap.Result, snap.Result)
	}

	seenRaw := mustEvalExprExport(t, rt, "__seenAsync")
	seen, ok := seenRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected __seenAsync object, got %T (%v)", seenRaw, seenRaw)
	}
	if seen["engine"] != true {
		t.Fatalf("expected JS engine callback to run, got %v", seen["engine"])
	}
	if seen["middleware"] != true {
		t.Fatalf("expected JS middleware callback to run, got %v", seen["middleware"])
	}
}

func TestStartWithJSEngineAndMiddleware(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		globalThis.__seenStart = { engine: false, middleware: false, events: 0 };

		const eng = gp.engines.fromFunction((turn) => {
			globalThis.__seenStart.engine = true;
			turn.blocks.push(gp.turns.newAssistantBlock("START-OK"));
			return turn;
		});
		const mw = gp.middlewares.fromJS((turn, next) => {
			globalThis.__seenStart.middleware = true;
			return next(turn);
		}, "start-mw");

		const s = gp.createBuilder()
			.withEngine(eng)
			.useMiddleware(mw)
			.buildSession();
		s.append(gp.turns.newTurn({ blocks: [ gp.turns.newUserBlock("hello") ] }));

		const handle = s.start(undefined, {
			timeoutMs: 1000,
			tags: { mode: "start-async-test" }
		});
		handle.on("*", () => {
			globalThis.__seenStart.events++;
		});
		globalThis.__startHandle = handle;
	`)

	snap := waitForPromiseExpr(t, rt, "__startHandle.promise", 2*time.Second)
	if snap.State != goja.PromiseStateFulfilled {
		t.Fatalf("expected fulfilled start promise, got %v", snap.State)
	}
	if turn, ok := snap.Result.(map[string]any); !ok || len(turn) == 0 {
		t.Fatalf("expected non-empty turn result map, got %T (%v)", snap.Result, snap.Result)
	}

	seenRaw := mustEvalExprExport(t, rt, "__seenStart")
	seen, ok := seenRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected __seenStart object, got %T (%v)", seenRaw, seenRaw)
	}
	if seen["engine"] != true {
		t.Fatalf("expected JS engine callback to run in start(), got %v", seen["engine"])
	}
	if seen["middleware"] != true {
		t.Fatalf("expected JS middleware callback to run in start(), got %v", seen["middleware"])
	}
}

func TestEnginesFromProfileAndFromConfigResolution(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-openai-key")
	rt := newJSRuntime(t, Options{
		ProfileRegistry: mustNewJSProfileRegistry(t),
	})
	mustRunJS(t, rt, `
		const gp = require("geppetto");

		const explicit = gp.engines.fromProfile("explicit-model");
		if (explicit.name !== "profile:default/explicit-model") throw new Error("explicit profile resolve mismatch");

		const defaultProfile = gp.engines.fromProfile(undefined);
		if (defaultProfile.name !== "profile:default/default-model") throw new Error("default profile resolve mismatch");

		let threwLegacyRegistry = false;
		try {
			gp.engines.fromProfile("default-model", { registrySlug: "shared" });
		} catch (e) {
			threwLegacyRegistry = /registryslug has been removed/i.test(String(e));
		}
		if (!threwLegacyRegistry) throw new Error("legacy registrySlug option should fail with migration error");

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

func TestEnginesFromProfileRequiresProfileRegistry(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		let threw = false;
		try {
			gp.engines.fromProfile("any");
		} catch (e) {
			threw = /profile registry/i.test(String(e));
		}
		if (!threw) throw new Error("fromProfile should throw when no profile registry is configured");
	`)
}

func TestEngineFromProfileInferenceIntegration_Gemini(t *testing.T) {
	if os.Getenv("GEPPETTO_LIVE_INFERENCE_TESTS") != "1" {
		t.Skip("skipping live inference integration test (set GEPPETTO_LIVE_INFERENCE_TESTS=1 to enable)")
	}
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_API_KEY") == "" {
		t.Skip("skipping gemini integration test: GEMINI_API_KEY/GOOGLE_API_KEY not set")
	}
	rt := newJSRuntime(t, Options{
		ProfileRegistry: mustNewJSGeminiProfileRegistry(t),
	})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const s = gp.createSession({
			engine: gp.engines.fromProfile("gemini-2.5-flash-lite")
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
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
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
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
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

func mustNewJSProfileRegistry(t *testing.T) gepprofiles.RegistryReader {
	t.Helper()
	store := gepprofiles.NewInMemoryProfileStore()
	if err := store.UpsertRegistry(context.Background(), &gepprofiles.ProfileRegistry{
		Slug:               gepprofiles.MustRegistrySlug("default"),
		DefaultProfileSlug: gepprofiles.MustProfileSlug("default-model"),
		Profiles: map[gepprofiles.ProfileSlug]*gepprofiles.Profile{
			gepprofiles.MustProfileSlug("default-model"): {
				Slug: gepprofiles.MustProfileSlug("default-model"),
				Runtime: gepprofiles.RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{
							"ai-engine":   "gpt-4o-mini",
							"ai-api-type": "openai",
						},
						"api": map[string]any{
							"openai-api-key": "test-openai-key",
						},
					},
				},
			},
			gepprofiles.MustProfileSlug("explicit-model"): {
				Slug: gepprofiles.MustProfileSlug("explicit-model"),
				Runtime: gepprofiles.RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{
							"ai-engine":   "gpt-4o",
							"ai-api-type": "openai",
						},
						"api": map[string]any{
							"openai-api-key": "test-openai-key",
						},
					},
				},
			},
		},
	}, gepprofiles.SaveOptions{Actor: "test", Source: "test"}); err != nil {
		t.Fatalf("UpsertRegistry(default) failed: %v", err)
	}
	registry, err := gepprofiles.NewStoreRegistry(store, gepprofiles.MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewStoreRegistry failed: %v", err)
	}
	return registry
}

func mustNewJSGeminiProfileRegistry(t *testing.T) gepprofiles.RegistryReader {
	t.Helper()
	store := gepprofiles.NewInMemoryProfileStore()
	if err := store.UpsertRegistry(context.Background(), &gepprofiles.ProfileRegistry{
		Slug:               gepprofiles.MustRegistrySlug("default"),
		DefaultProfileSlug: gepprofiles.MustProfileSlug("gemini-2.5-flash-lite"),
		Profiles: map[gepprofiles.ProfileSlug]*gepprofiles.Profile{
			gepprofiles.MustProfileSlug("gemini-2.5-flash-lite"): {
				Slug: gepprofiles.MustProfileSlug("gemini-2.5-flash-lite"),
				Runtime: gepprofiles.RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{
							"ai-engine":   "gemini-2.5-flash-lite",
							"ai-api-type": "gemini",
						},
					},
				},
			},
		},
	}, gepprofiles.SaveOptions{Actor: "test", Source: "test"}); err != nil {
		t.Fatalf("UpsertRegistry(default gemini) failed: %v", err)
	}
	registry, err := gepprofiles.NewStoreRegistry(store, gepprofiles.MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewStoreRegistry failed: %v", err)
	}
	return registry
}
