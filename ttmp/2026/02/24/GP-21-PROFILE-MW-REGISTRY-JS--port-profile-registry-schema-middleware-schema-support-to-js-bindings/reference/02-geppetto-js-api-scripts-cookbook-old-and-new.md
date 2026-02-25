---
Title: Geppetto JS API scripts cookbook old and new
Ticket: GP-21-PROFILE-MW-REGISTRY-JS
Status: active
Topics:
    - profile-registry
    - js-bindings
    - go-api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/examples/geppetto-js-lab/main.go
      Note: |-
        Script runner for copy-paste execution of examples
        Runner command for executing scripts
    - Path: pkg/js/modules/geppetto/api_builder_options.go
      Note: |-
        Tool-loop and builder option behaviors used by current examples
        Builder/tool loop options used by examples
    - Path: pkg/js/modules/geppetto/api_engines.go
      Note: |-
        fromProfile and fromConfig current behavior
        fromProfile/fromConfig script behavior references
    - Path: pkg/js/modules/geppetto/api_middlewares.go
      Note: |-
        Middleware callback and context contract
        Middleware callback and context usage in examples
    - Path: pkg/js/modules/geppetto/api_sessions.go
      Note: |-
        Session lifecycle behaviors used by current examples
        Session run/start/runAsync examples align with these contracts
    - Path: pkg/js/modules/geppetto/module.go
      Note: |-
        Current require("geppetto") export surface
        Current exported API namespaces referenced by runnable scripts
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/design-doc/02-unified-final-js-api-design-inference-first.md
      Note: |-
        Hard-cutover final API reference for new examples
        New/hard-cutover script section follows final API design
ExternalSources: []
Summary: Large script cookbook for old and new Geppetto JS APIs, including runnable current examples and planned hard-cutover examples.
LastUpdated: 2026-02-25T00:00:00Z
WhatFor: Provide a broad copy/paste script catalog to demonstrate and validate Geppetto JS API functionality.
WhenToUse: Use when onboarding, validating behavior in geppetto-js-lab, or planning migration from old APIs to hard-cutover APIs.
---


# Geppetto JS API scripts cookbook old and new

## Goal

Provide a large set of copy/paste JavaScript scripts that showcase Geppetto JS API functionality:

1. currently available functionality (old/current API), and
2. planned hard-cutover functionality (new/final API direction).

## Context

This cookbook intentionally has two tracks:

1. `Current (runnable today)` scripts align with the API currently exported by `require("geppetto")`.
2. `New (hard-cutover target)` scripts align with the final API design in `design-doc/02-unified-final-js-api-design-inference-first.md` and are expected to run after implementation.

Runner used for current scripts:

```bash
cd /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto
go run ./cmd/examples/geppetto-js-lab --script /abs/path/to/script.js
```

## Quick Reference

### Availability legend

- `CURRENT`: runnable against repository head today.
- `NEW`: design-target example for hard cutover; not yet implemented at repository head.

### CURRENT scripts (runnable today)

#### Script 01: Export inventory

```javascript
// current-01-export-inventory.js
const gp = require("geppetto");
console.log({
  keys: Object.keys(gp).sort(),
  hasProfiles: typeof gp.profiles !== "undefined",
  hasSchemas: typeof gp.schemas !== "undefined",
});
```

#### Script 02: Basic turn constructors

```javascript
// current-02-turn-builders.js
const gp = require("geppetto");

const t = gp.turns.newTurn({
  blocks: [
    gp.turns.newSystemBlock("You are concise."),
    gp.turns.newUserBlock("Say READY"),
  ],
});

console.log({ id: t.id, blockKinds: t.blocks.map((b) => b.kind) });
```

#### Script 03: Echo engine sync run

```javascript
// current-03-echo-run.js
const gp = require("geppetto");

const s = gp.createSession({
  engine: gp.engines.echo({ reply: "READY" }),
});

s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("hello")] }));
const out = s.run();
console.log({ turnCount: s.turnCount(), lastKind: out.blocks[out.blocks.length - 1].kind });
```

#### Script 04: Engine from JS function

```javascript
// current-04-from-function.js
const gp = require("geppetto");

const eng = gp.engines.fromFunction((turn) => {
  turn.blocks.push(gp.turns.newAssistantBlock("FUNCTION_ENGINE_OK"));
  return turn;
});

const s = gp.createSession({ engine: eng });
s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("ping")] }));
console.log(s.run().blocks.map((b) => ({ kind: b.kind, text: b.text })));
```

#### Script 05: Builder chaining

```javascript
// current-05-builder-chain.js
const gp = require("geppetto");

const s = gp.createBuilder()
  .withEngine(gp.engines.echo({ reply: "CHAIN_OK" }))
  .useMiddleware(gp.middlewares.fromJS((turn, next) => next(turn), "noop"))
  .buildSession();

s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("go")] }));
console.log({ ok: !!s.run() });
```

#### Script 06: Middleware context (sessionId/inferenceId/tags/deadline)

```javascript
// current-06-middleware-context.js
const gp = require("geppetto");
let seen = null;

const s = gp.createBuilder()
  .withEngine(gp.engines.echo({ reply: "CTX_OK" }))
  .useMiddleware(
    gp.middlewares.fromJS((turn, next, ctx) => {
      seen = ctx;
      return next(turn);
    }, "ctx-mw"),
  )
  .buildSession();

s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("hello")] }));
s.run(undefined, { timeoutMs: 500, tags: { requestId: "r-1" } });

console.log({
  hasSessionId: !!seen.sessionId,
  hasInferenceId: !!seen.inferenceId,
  hasDeadlineMs: !!seen.deadlineMs,
  requestId: seen.tags && seen.tags.requestId,
});
```

#### Script 07: Go middleware reference

```javascript
// current-07-go-middleware.js
const gp = require("geppetto");

const s = gp.createBuilder()
  .withEngine(gp.engines.echo({ reply: "GO_MW_OK" }))
  .useGoMiddleware("systemPrompt", { prompt: "System prompt from Go middleware" })
  .buildSession();

s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("start")] }));
const out = s.run();
console.log({ blockKinds: out.blocks.map((b) => b.kind) });
```

#### Script 08: Tool registry direct call

```javascript
// current-08-tool-registry-direct.js
const gp = require("geppetto");

const reg = gp.tools.createRegistry();
reg.register({
  name: "sum",
  description: "add two numbers",
  handler: (args) => ({ value: Number(args.a || 0) + Number(args.b || 0) }),
});

console.log(reg.call("sum", { a: 2, b: 3 }));
```

#### Script 09: Tool loop basic

```javascript
// current-09-tool-loop-basic.js
const gp = require("geppetto");

const reg = gp.tools.createRegistry();
reg.register({
  name: "echo_tool",
  description: "echo",
  handler: (args) => ({ echoed: args }),
});

const eng = gp.engines.fromFunction((turn) => {
  const hasToolUse = turn.blocks.some((b) => b.kind === "tool_use");
  if (!hasToolUse) {
    turn.blocks.push(gp.turns.newToolCallBlock("call-1", "echo_tool", { x: 1 }));
    return turn;
  }
  turn.blocks.push(gp.turns.newAssistantBlock("done"));
  return turn;
});

const s = gp.createBuilder()
  .withEngine(eng)
  .withTools(reg, { enabled: true, maxIterations: 3, toolChoice: gp.consts.ToolChoice.AUTO })
  .buildSession();

s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("go")]}));
console.log({ blocks: s.run().blocks.map((b) => b.kind) });
```

#### Script 10: Tool hooks

```javascript
// current-10-tool-hooks.js
const gp = require("geppetto");
let beforeSeen = 0;

const reg = gp.tools.createRegistry();
reg.register({
  name: "t",
  description: "tool",
  handler: () => ({ ok: true }),
});

const eng = gp.engines.fromFunction((turn) => {
  const hasToolUse = turn.blocks.some((b) => b.kind === "tool_use");
  if (!hasToolUse) {
    turn.blocks.push(gp.turns.newToolCallBlock("c1", "t", {}));
    return turn;
  }
  turn.blocks.push(gp.turns.newAssistantBlock("done"));
  return turn;
});

const s = gp.createBuilder()
  .withEngine(eng)
  .withTools(reg, { enabled: true, maxIterations: 3 })
  .withToolHooks({
    beforeToolCall: () => {
      beforeSeen += 1;
    },
  })
  .buildSession();

s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("go")]}));
s.run();
console.log({ beforeSeen });
```

#### Script 11: `start()` run handle and events

```javascript
// current-11-start-handle.js
const gp = require("geppetto");

const s = gp.createSession({ engine: gp.engines.echo({ reply: "START_OK" }) });
s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("go")]}));

const handle = s.start(undefined, { timeoutMs: 1000, tags: { mode: "start" } });
let eventCount = 0;
handle.on("*", () => {
  eventCount += 1;
});

handle.promise.then((out) => {
  console.log({ resolved: !!out, eventCount });
});
```

#### Script 12: `runAsync()`

```javascript
// current-12-run-async.js
const gp = require("geppetto");

const s = gp.createSession({ engine: gp.engines.echo({ reply: "ASYNC_OK" }) });
s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("go")]}));

s.runAsync().then((out) => {
  console.log({ ok: !!out, blocks: out.blocks.length });
});
```

#### Script 13: History and range inspection

```javascript
// current-13-history-range.js
const gp = require("geppetto");

const s = gp.createSession({ engine: gp.engines.echo({ reply: "H" }) });
for (let i = 0; i < 3; i += 1) {
  s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("u" + i)] }));
  s.run();
}

console.log({
  count: s.turnCount(),
  latest: !!s.latest(),
  rangeLen: s.turnsRange(1, 3).length,
});
```

#### Script 14: Cancel active run

```javascript
// current-14-cancel-active.js
const gp = require("geppetto");

const slow = gp.engines.fromFunction((turn) => {
  sleep(2000); // helper provided by geppetto-js-lab
  turn.blocks.push(gp.turns.newAssistantBlock("SLOW_DONE"));
  return turn;
});

const s = gp.createSession({ engine: slow });
s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("run")] }));
const handle = s.start(undefined, { timeoutMs: 5000 });
handle.cancel();
handle.promise.catch((e) => console.log({ cancelled: true, error: String(e) }));
```

#### Script 15: `runInference()` one-shot helper

```javascript
// current-15-run-inference.js
const gp = require("geppetto");

const out = gp.runInference(
  gp.engines.echo({ reply: "ONE_SHOT_OK" }),
  gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("go")] }),
);

console.log({ blocks: out.blocks.map((b) => b.kind) });
```

#### Script 16: `engines.fromConfig()`

```javascript
// current-16-from-config.js
const gp = require("geppetto");

const eng = gp.engines.fromConfig({
  apiType: "openai",
  model: "gpt-4o-mini",
  apiKey: "test-key",
});

console.log({ name: eng.name });
```

#### Script 17: `engines.fromProfile()` old behavior snapshot

```javascript
// current-17-from-profile-old.js
const gp = require("geppetto");

const explicit = gp.engines.fromProfile("explicit-model", {
  profile: "opts-model",
  apiType: "openai",
  apiKey: "k",
});

const viaOptions = gp.engines.fromProfile(undefined, {
  profile: "opts-model",
  apiType: "openai",
  apiKey: "k",
});

console.log({ explicit: explicit.name, viaOptions: viaOptions.name });
```

#### Script 18: Full mini-app (engine + middleware + tools + hooks)

```javascript
// current-18-mini-app.js
const gp = require("geppetto");

const reg = gp.tools.createRegistry();
reg.register({
  name: "double",
  description: "double number",
  handler: (args) => ({ value: Number(args.n || 0) * 2 }),
});

const eng = gp.engines.fromFunction((turn) => {
  const hasToolUse = turn.blocks.some((b) => b.kind === "tool_use");
  if (!hasToolUse) {
    turn.blocks.push(gp.turns.newToolCallBlock("d1", "double", { n: 7 }));
    return turn;
  }
  turn.blocks.push(gp.turns.newAssistantBlock("workflow complete"));
  return turn;
});

let middlewareCount = 0;
const s = gp.createBuilder()
  .withEngine(eng)
  .useMiddleware(gp.middlewares.fromJS((turn, next) => {
    middlewareCount += 1;
    return next(turn);
  }, "count-mw"))
  .withTools(reg, { enabled: true, maxIterations: 3 })
  .withToolHooks({ beforeToolCall: () => console.log({ hook: "before" }) })
  .buildSession();

s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("start")] }));
const out = s.run(undefined, { tags: { app: "mini" } });
console.log({ middlewareCount, blockKinds: out.blocks.map((b) => b.kind) });
```

### NEW scripts (hard-cutover target)

These scripts reflect the final API direction and are expected to run after implementing `gp.profiles`, `gp.schemas`, `gp.factories`, and cutover `engines.fromProfile` behavior.

#### Script 19: List registries

```javascript
// new-19-list-registries.js
const gp = require("geppetto");
console.log(gp.profiles.listRegistries());
```

#### Script 20: Resolve effective profile

```javascript
// new-20-resolve-profile.js
const gp = require("geppetto");

const resolved = gp.profiles.resolve({
  registrySlug: "default",
  profileSlug: "agent",
  runtimeKeyFallback: "agent",
  requestOverrides: {
    system_prompt: "Override for one run",
  },
});

console.log({
  registry: resolved.registrySlug,
  profile: resolved.profileSlug,
  runtimeKey: resolved.runtimeKey,
  fingerprint: resolved.runtimeFingerprint,
});
```

#### Script 21: Middleware schema catalog

```javascript
// new-21-schemas-middlewares.js
const gp = require("geppetto");
const rows = gp.schemas.listMiddlewares();
console.log(rows.map((r) => ({ name: r.name, hasSchema: !!r.schema })));
```

#### Script 22: Extension schema catalog

```javascript
// new-22-schemas-extensions.js
const gp = require("geppetto");
const rows = gp.schemas.listExtensions();
console.log(rows.map((r) => r.key));
```

#### Script 23: Profile CRUD

```javascript
// new-23-profile-crud.js
const gp = require("geppetto");

gp.profiles.createProfile(
  {
    slug: "ops",
    display_name: "Ops",
    runtime: {
      system_prompt: "You are operations support",
      middlewares: [{ name: "retry", id: "default" }],
    },
  },
  { registrySlug: "default", write: { actor: "js-example", source: "cookbook" } },
);

gp.profiles.updateProfile(
  "ops",
  { runtime: { system_prompt: "You are strict operations support" } },
  { registrySlug: "default", write: { actor: "js-example", source: "cookbook" } },
);

console.log(gp.profiles.getProfile("ops", "default"));

gp.profiles.deleteProfile("ops", {
  registrySlug: "default",
  write: { actor: "js-example", source: "cookbook" },
});
```

#### Script 24: Factory plan and debug

```javascript
// new-24-factory-plan.js
const gp = require("geppetto");

const f = gp.factories.createEngineFactory({
  defaultRegistrySlug: "default",
  defaultProfileSlug: "agent",
});

const plan = f.plan({
  profile: "agent",
  debug: true,
  middlewarePatch: (m) => m.configure("retry", { maxAttempts: 2 }),
});

console.log({
  runtimeKey: plan.runtimeKey,
  fingerprint: plan.runtimeFingerprint,
  middlewares: plan.resolvedRuntime.middlewares,
});
```

#### Script 25: Factory -> session -> run

```javascript
// new-25-factory-session-run.js
const gp = require("geppetto");

const f = gp.factories.createEngineFactory({ defaultRegistrySlug: "default" });
const s = f.createSession({ profile: "agent" });

s.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("hello")]}));
const out = s.run(undefined, { timeoutMs: 1200, tags: { mode: "factory" } });
console.log(out.blocks.map((b) => b.kind));
```

#### Script 26: Middleware patch operations

```javascript
// new-26-middleware-patch.js
const gp = require("geppetto");

const patch = gp.factories.middlewarePatch()
  .prepend({ name: "request-id", id: "req-id" })
  .configure("retry", { maxAttempts: 4 })
  .disable("verbose-logging")
  .append({ name: "telemetry", id: "trace", config: { level: "debug" } })
  .build();

console.log(patch);
```

#### Script 27: Policy rejection example

```javascript
// new-27-policy-rejection.js
const gp = require("geppetto");

try {
  gp.factories.createEngineFactory({ defaultRegistrySlug: "default" }).createSession({
    profile: "locked-profile",
    requestOverrides: { system_prompt: "attempt override" },
  });
} catch (e) {
  console.log({
    code: e.code,
    message: e.message,
    details: e.details,
  });
}
```

#### Script 28: Strict `engines.fromProfile()` after hard cutover

```javascript
// new-28-from-profile-cutover.js
const gp = require("geppetto");

// After cutover, this is registry-driven and not model/env fallback.
const eng = gp.engines.fromProfile("agent", { registry: "default" });
console.log({ name: eng.name });
```

#### Script 29: Cutover failure without registry wiring

```javascript
// new-29-from-profile-missing-registry.js
const gp = require("geppetto");

try {
  gp.engines.fromProfile("agent", { registry: "default" });
} catch (e) {
  console.log({ code: e.code, message: e.message });
}
```

#### Script 30: Migration examples (not one-to-one)

`fromProfile("gpt-4o-mini", ...)` (old usage) and `createEngineFactory(...).createSession({ profile: "agent" })` are not inherently equivalent.

Use the migration path that matches intent:

1. Direct model/provider intent -> migrate to `engines.fromConfig(...)`.
2. Profile/runtime-policy intent -> migrate to registry/factory APIs.

```javascript
// new-30a-migration-direct-model.js
// Old (model-centric misuse of fromProfile):
const gp = require("geppetto");
const oldEng = gp.engines.fromProfile("gpt-4o-mini", { apiType: "openai", apiKey: "k" });

// New (explicit direct model construction):
const newEng = gp.engines.fromConfig({ model: "gpt-4o-mini", apiType: "openai", apiKey: "k" });
```

```javascript
// new-30b-migration-profile-driven.js
// Old profile-driven shape:
const gp = require("geppetto");
const oldProfileEng = gp.engines.fromProfile("agent", { profile: "agent" });

// New profile-driven cutover:
const gp2 = require("geppetto");
const s = gp2
  .factories
  .createEngineFactory({ defaultRegistrySlug: "default" })
  .createSession({ profile: "agent" });
```

## Usage Examples

### How to run current scripts quickly

1. Save one script block to a file like `/tmp/current-03-echo-run.js`.
2. Run:

```bash
cd /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto
go run ./cmd/examples/geppetto-js-lab --script /tmp/current-03-echo-run.js
```

### Suggested smoke test sequence (current API)

```bash
# 1) basic exports + turns + sync run
go run ./cmd/examples/geppetto-js-lab --script /tmp/current-01-export-inventory.js
go run ./cmd/examples/geppetto-js-lab --script /tmp/current-02-turn-builders.js
go run ./cmd/examples/geppetto-js-lab --script /tmp/current-03-echo-run.js

# 2) middleware + tools + hooks
go run ./cmd/examples/geppetto-js-lab --script /tmp/current-06-middleware-context.js
go run ./cmd/examples/geppetto-js-lab --script /tmp/current-09-tool-loop-basic.js
go run ./cmd/examples/geppetto-js-lab --script /tmp/current-10-tool-hooks.js

# 3) async lifecycle
go run ./cmd/examples/geppetto-js-lab --script /tmp/current-11-start-handle.js
go run ./cmd/examples/geppetto-js-lab --script /tmp/current-12-run-async.js
```

### How to use new scripts during implementation

1. Pick one `new-*` script and convert it into a failing test script in ticket `scripts/`.
2. Implement the matching API surface.
3. Re-run script until behavior matches the cookbook contract.

## Related

1. `design-doc/02-unified-final-js-api-design-inference-first.md`
2. `reference/01-investigation-diary.md`
3. `scripts/inspect_geppetto_exports.js`
4. `scripts/inspect_inference_surface.js`
