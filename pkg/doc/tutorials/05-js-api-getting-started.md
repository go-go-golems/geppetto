---
Title: Getting Started with the Geppetto JavaScript API
Slug: geppetto-js-api-getting-started
Short: Step-by-step tutorial for turns, sessions, engines, middlewares, tools, and toolloop hooks from JavaScript.
Topics:
- geppetto
- javascript
- goja
- tutorial
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial is intentionally deep and script-first. You will learn the model behind the API, then build from a blank script file to a multi-step tool-enabled flow. Every stage is executable.

Reference docs:

- [JS API Reference](../topics/13-js-api-reference.md)
- [JS API User Guide](../topics/14-js-api-user-guide.md)

## How to Read This Guide

Each step follows one structure:

- fundamentals: the core idea and why it exists
- APIs used: the exact JS functions involved
- pseudocode: implementation logic independent of syntax details
- diagram: data/control flow
- runnable script: copy/paste code
- validation checklist: what to confirm after execution

If you already know Geppetto’s internal Go architecture, focus on the API and flow sections. If you are new, read every fundamentals section in order.

## Mental Model First

Before writing code, keep this picture in mind.

```text
+------------------------------ JS Script -------------------------------+
| require("geppetto")                                                 |
|                                                                      |
|  turn -> session -> engine -> middlewares -> tools -> toolloop       |
|                                                                      |
+------------------------------+----------------------------------------+
                               |
                               v
+--------------------------- Goja Boundary -----------------------------+
|  decode JS values -> Go structs -> execute -> encode back to JS      |
+------------------------------+----------------------------------------+
                               |
                               v
+-------------------------- Geppetto Runtime ---------------------------+
| turns, sessions, inference engines, middleware chain, tool execution  |
+----------------------------------------------------------------------+
```

Key entities:

- `Turn`: one inference state container containing blocks plus metadata/data maps.
- `Block`: typed unit inside a turn (`user`, `system`, `llm_text`, `tool_call`, `tool_use`, ...).
- `Engine`: function-like inference component that transforms a turn.
- `Session`: stateful history wrapper around repeated engine execution.
- `Middleware`: interceptors around engine execution.
- `ToolRegistry`: callable tool definitions available to the model and runtime.
- `Toolloop`: iterative cycle that executes tool calls and re-enters inference.

## Prerequisites

- Go toolchain installed
- repository checked out
- run commands from repository root

## Step 0: Validate the Host Runtime

### Fundamentals

Your JS does not run in Node.js in this workflow. It runs inside goja, hosted by `geppetto-js-lab`.

That host is responsible for:

- registering native module `require("geppetto")`
- providing helper globals (`assert`, `console`, `ENV`)
- injecting host-side Go tool registry for `useGoTools`

Without a host, your JS code has no access to Geppetto API surfaces.

### APIs Used

- CLI: `go run ./cmd/examples/geppetto-js-lab --list-go-tools`
- JS: none yet (this step validates host setup)

### Pseudocode

```text
start host
register geppetto native module
register host go tools
if --list-go-tools:
  print tool names
```

### Diagram

```text
Terminal command
    |
    v
geppetto-js-lab
    |
    +--> builds Go tool registry
    +--> configures goja runtime
    +--> registers require("geppetto")
    |
    v
prints host capabilities
```

### Run

```bash
go run ./cmd/examples/geppetto-js-lab --list-go-tools
```

Expected output includes:

- `go_double`
- `go_concat`

### Validation Checklist

- command exits with status 0
- tool names printed
- no JS execution errors

## Step 1: Turns and Blocks from First Principles

### Fundamentals

A `Turn` is the core payload for inference. Think of it as an envelope with two categories of content:

- ordered `blocks`: conversational and tool events
- side-channel maps: `metadata` and `data`

A `Block` has:

- `kind`: semantic type (`user`, `tool_call`, etc.)
- `payload`: type-specific fields

Why start here:

- every later API (session, middleware, tools) reads/writes turns
- shape errors here propagate everywhere

Normalization matters because JS objects are flexible while Go structs are typed. `turns.normalize` round-trips through the canonical mapper and guarantees your turn shape is compatible with runtime expectations.

### APIs Used

- `gp.turns.newTurn(opts)`
- `gp.turns.newUserBlock(text)`
- `gp.turns.newToolCallBlock(id, name, args)`
- `gp.turns.normalize(turn)`

### Pseudocode

```text
build user block
build tool_call block
construct turn with blocks + metadata
assert shape
normalize turn through codec
assert normalized fields preserved
```

### Diagram

```text
JS object literal
   |
   v
turns.newTurn(...)
   |
   +--> blocks array
   +--> metadata map
   +--> data map
   |
   v
turns.normalize(turn)
   |
   +--> decode JS -> Go Turn
   +--> encode Go Turn -> JS
   |
   v
canonical turn object
```

### Script

Create `scratch/js-lab/01_turns.js`:

```js
const gp = require("geppetto");

const turn = gp.turns.newTurn({
  id: "turn-1",
  blocks: [
    gp.turns.newUserBlock("hello"),
    gp.turns.newToolCallBlock("call-1", "js_add", { a: 2, b: 3 })
  ],
  metadata: { session_id: "s-1" }
});

assert(turn.blocks.length === 2, "expected two blocks");
assert(turn.blocks[0].kind === "user", "first block kind mismatch");
assert(turn.blocks[1].kind === "tool_call", "second block kind mismatch");

const normalized = gp.turns.normalize(turn);
assert(normalized.metadata.session_id === "s-1", "metadata mismatch");

console.log("PASS step 1");
```

Run it:

```bash
go run ./cmd/examples/geppetto-js-lab --script scratch/js-lab/01_turns.js
```

### Validation Checklist

- two blocks exist in the order created
- `kind` values are exactly expected
- metadata survives normalization

## Step 2: Session Lifecycle and Deterministic Engine

### Fundamentals

A `Session` provides conversation state and execution lifecycle:

- append a seed turn
- run inference
- inspect history later

`engines.echo` is deterministic and should be your first engine in any new flow because it removes provider variability. You can test state wiring before involving real model APIs.

Conceptually:

- `Session` is your state container
- `Engine` is your transformation function
- `run()` executes engine over current turn context and stores result

### APIs Used

- `gp.engines.echo({ reply })`
- `gp.createSession({ engine })`
- `session.append(turn)`
- `session.run()`
- `session.turnCount()`

### Pseudocode

```text
engine := echo("READY")
session := createSession(engine)
append user turn
out := session.run()
assert last block is llm_text READY
assert history length == 1
```

### Diagram

```text
user turn
   |
   v
session.append
   |
   v
session.run
   |
   v
echo engine appends llm_text("READY")
   |
   v
updated turn returned + stored in session history
```

### Script

Create `scratch/js-lab/02_session.js`:

```js
const gp = require("geppetto");

const session = gp.createSession({
  engine: gp.engines.echo({ reply: "READY" })
});

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("reply READY")] }));
const out = session.run();

const last = out.blocks[out.blocks.length - 1];
assert(last.kind === "llm_text", "missing llm_text output");
assert(last.payload.text === "READY", "assistant text mismatch");
assert(session.turnCount() === 1, "turnCount mismatch");

console.log("PASS step 2");
```

Run it:

```bash
go run ./cmd/examples/geppetto-js-lab --script scratch/js-lab/02_session.js
```

### Validation Checklist

- `run()` returns updated turn
- assistant block is present and deterministic
- history count increments as expected

## Step 3: Middleware Composition (Go + JS)

### Fundamentals

Middleware is a chain-of-responsibility around engine execution. It is used to enforce policies, inject prompts, log traces, or transform turn content.

A middleware receives:

- current turn
- `next(turn)` callback

It can:

- mutate turn before execution
- call `next`
- mutate output after execution

Ordering is critical. If middleware A must run before B, register A first.

In this step:

- Go middleware `systemPrompt` adds a system block
- JS middleware adds metadata after `next`

### APIs Used

- `gp.engines.fromFunction(fn)`
- `gp.createBuilder()`
- `builder.withEngine(engine)`
- `builder.useGoMiddleware("systemPrompt", opts)`
- `gp.middlewares.fromJS(fn, name?)`
- `builder.useMiddleware(middleware)`
- `builder.buildSession()`

### Pseudocode

```text
engine := fromFunction(append "ok")
builder := createBuilder
builder.withEngine(engine)
builder.useGoMiddleware(systemPrompt, {prompt:"SYSTEM"})
builder.useMiddleware(jsMiddleware(add trace_id))
session := builder.buildSession
append user turn
out := session.run
assert system block inserted first
assert trace_id exists
```

### Diagram

```text
Input Turn
   |
   v
[Go systemPrompt middleware]
   |
   v
[JS trace middleware pre]
   |
   v
Engine (append assistant "ok")
   |
   v
[JS trace middleware post -> metadata.trace_id]
   |
   v
Output Turn
```

### Script

Create `scratch/js-lab/03_middleware.js`:

```js
const gp = require("geppetto");

const engine = gp.engines.fromFunction((turn) => {
  turn.blocks.push(gp.turns.newAssistantBlock("ok"));
  return turn;
});

const session = gp
  .createBuilder()
  .withEngine(engine)
  .useGoMiddleware("systemPrompt", { prompt: "SYSTEM" })
  .useMiddleware(gp.middlewares.fromJS((turn, next) => {
    const out = next(turn);
    out.metadata = out.metadata || {};
    out.metadata.trace_id = "js-mw";
    return out;
  }, "trace"))
  .buildSession();

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("ping")] }));
const out = session.run();

assert(out.blocks[0].kind === "system", "system prompt missing");
assert(out.metadata.trace_id === "js-mw", "trace metadata missing");

console.log("PASS step 3");
```

Run it:

```bash
go run ./cmd/examples/geppetto-js-lab --script scratch/js-lab/03_middleware.js
```

### Validation Checklist

- system block appears at index 0
- metadata contains `trace_id`
- no middleware ordering surprises

## Step 4: Tool Registry and Toolloop Mechanics

### Fundamentals

Tooling has two layers:

- registry layer: what tools exist and how they execute
- orchestration layer (toolloop): when and how tool calls are executed during inference

Core loop idea:

1. engine emits `tool_call`
2. runtime executes tool
3. runtime appends `tool_use` result block
4. engine runs again with new context
5. loop stops when no tool calls or limits reached

This means your engine can act as orchestrator logic while toolloop handles reliable execution and retries/policies.

### APIs Used

- `gp.tools.createRegistry()`
- `registry.register(spec)`
- `gp.turns.newToolCallBlock(id, name, args)`
- `builder.withTools(registry, opts)`
- tool options: `enabled`, `maxIterations`, `toolChoice`, `maxParallelTools`

### Pseudocode

```text
register js_add tool
engine:
  if no tool_use yet:
    emit tool_call(js_add)
  else:
    emit assistant done
builder.withTools(reg, loopOpts)
run session
assert tool_use block exists with sum result
```

### Diagram

```text
Iteration 1:
  Engine -> tool_call(js_add)
  Toolloop executes js_add -> tool_use({sum:5})

Iteration 2:
  Engine sees tool_use -> assistant("done")
  No new tool_call -> stop
```

### Script

Create `scratch/js-lab/04_tools.js`:

```js
const gp = require("geppetto");

const reg = gp.tools.createRegistry();
reg.register({
  name: "js_add",
  description: "Add numbers",
  handler: ({ a, b }) => ({ sum: a + b })
});

const engine = gp.engines.fromFunction((turn) => {
  const hasToolUse = turn.blocks.some((b) => b.kind === "tool_use");
  if (!hasToolUse) {
    turn.blocks.push(gp.turns.newToolCallBlock("call-1", "js_add", { a: 2, b: 3 }));
    return turn;
  }
  turn.blocks.push(gp.turns.newAssistantBlock("done"));
  return turn;
});

const session = gp
  .createBuilder()
  .withEngine(engine)
  .withTools(reg, { enabled: true, maxIterations: 3, toolChoice: "auto" })
  .buildSession();

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("compute")] }));
const out = session.run();

const toolUse = out.blocks.find((b) => b.kind === "tool_use");
assert(!!toolUse, "missing tool_use block");
assert(String(toolUse.payload.result).includes("sum"), "tool result missing sum");

console.log("PASS step 4");
```

Run it:

```bash
go run ./cmd/examples/geppetto-js-lab --script scratch/js-lab/04_tools.js
```

### Validation Checklist

- at least one `tool_call` generated internally by engine
- corresponding `tool_use` appended by runtime
- final assistant block appears after tool result

## Step 5: Hybrid Tooling (Import Go Tools into JS)

### Fundamentals

In production, you often need mixed tools:

- quick script-local JS tools
- mature Go tools from backend systems

`useGoTools` imports host-exposed Go tools into your JS registry. This keeps one unified invocation model in JS while still using typed Go implementations.

Important constraint:

- host must provide a Go tool registry
- `geppetto-js-lab` does this automatically

### APIs Used

- `registry.useGoTools([names])`
- `registry.call(name, args)`
- `builder.withTools(registry, opts)` for loop-driven usage

### Pseudocode

```text
reg := createRegistry
reg.useGoTools(["go_double"])
result := reg.call("go_double", {n:21})
assert result.value == 42
```

### Diagram

```text
JS script
   |
   v
registry.useGoTools("go_double")
   |
   v
host Go tool registry lookup
   |
   v
tool becomes callable in JS registry
   |
   v
registry.call("go_double", {n:21}) -> {value:42}
```

### Script

Create `scratch/js-lab/05_go_tools.js`:

```js
const gp = require("geppetto");

const reg = gp.tools.createRegistry();
reg.useGoTools(["go_double"]);

const direct = reg.call("go_double", { n: 21 });
assert(direct.value === 42, "direct go tool call mismatch");

console.log("PASS step 5", JSON.stringify(direct));
```

Run it:

```bash
go run ./cmd/examples/geppetto-js-lab --script scratch/js-lab/05_go_tools.js
```

### Validation Checklist

- import call succeeds
- direct invocation returns typed value payload
- no registry configuration errors

## Step 6: Live Provider Inference (Optional Final Stage)

### Fundamentals

Live inference introduces non-determinism and credential dependency. That is why it appears last.

What changes from deterministic steps:

- real network and provider API
- model behavior variability
- quota and latency constraints

Recommended order:

- validate deterministic scripts first
- then run live script as smoke test

### APIs Used

- `gp.engines.fromConfig({ apiType, model, apiKey })`
- `gp.createSession({ engine })`
- `session.run()`
- environment via `ENV`

### Pseudocode

```text
apiKey := ENV.GEMINI_API_KEY or ENV.GOOGLE_API_KEY
if missing:
  print SKIP
  exit success
create engine fromConfig(gemini)
run one-turn session
assert output blocks exist
log final block
```

### Diagram

```text
Script start
   |
   +--> key exists? -- no --> print SKIP --> success exit
   |
  yes
   |
   v
create gemini engine -> run session -> inspect final block
```

### Run

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/06_live_profile_inference.js
```

If keys are absent, script self-skips and still exits successfully.

### Validation Checklist

- with no key: skip message appears
- with key: output turn contains assistant-like final block
- no configuration panic/errors

## End-to-End Fast Path

If you want all maintained example scripts in sequence:

```bash
go run ./cmd/examples/geppetto-js-lab --list-go-tools
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/01_turns_and_blocks.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/02_session_echo.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/03_middleware_composition.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/04_tools_and_toolloop.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/05_go_tools_from_js.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/06_live_profile_inference.js
```

## API Summary by Step

| Step | Main APIs | Main Concept |
|---|---|---|
| 0 | `geppetto-js-lab` CLI | host runtime wiring |
| 1 | `turns.*` | canonical turn/block schema |
| 2 | `engines.echo`, `createSession`, `run` | deterministic stateful inference |
| 3 | `createBuilder`, `useGoMiddleware`, `middlewares.fromJS` | chain-of-responsibility composition |
| 4 | `tools.createRegistry`, `withTools` | tool execution loop |
| 5 | `useGoTools`, `call` | hybrid JS+Go tool ecosystem |
| 6 | `engines.fromConfig` | real provider smoke validation |

## Common Failure Modes

| Problem | Why it happens | Fix |
|---|---|---|
| `module geppetto not found` | runtime host did not register module | use `geppetto-js-lab` or register via `gp.Register` |
| `builder has no engine configured` | `withEngine` omitted | call `withEngine` before `buildSession` |
| `no go tool registry configured` | `useGoTools` in host without Go registry | run in `geppetto-js-lab` or inject `Options.GoToolRegistry` |
| no tool execution | tool registry not bound to builder | use `.withTools(reg, { enabled: true })` |
| `runAsync requires module options Runner to be configured` | missing runtime runner in host | use `run()` or register module with `Options.Runner` |
| live script fails auth | missing/invalid API key | set `GEMINI_API_KEY` or `GOOGLE_API_KEY` |

## Design Guidance for Real Projects

Use these conventions when building larger JS automation packs:

- one script per behavior slice
- deterministic first, live-provider last
- strict assertions on observable output (`kind`, payload fields, metadata)
- avoid hidden global state between scripts
- keep script names ordered by dependency and complexity

A practical naming pattern:

- `01_...` schema and shapes
- `02_...` basic session behavior
- `03_...` middleware behavior
- `04_...` toolloop behavior
- `05_...` integration with host tools
- `06_...` live smoke

## Next Steps

1. Use [JS API User Guide](../topics/14-js-api-user-guide.md) for composition patterns and production tradeoffs.
2. Use [JS API Reference](../topics/13-js-api-reference.md) for exhaustive method contracts and options.
3. Extend `examples/js/geppetto` with your own domain scripts and keep them executable through `geppetto-js-lab`.
