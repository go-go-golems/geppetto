---
Title: Geppetto JavaScript API User Guide
Slug: geppetto-js-api-user-guide
Short: Practical guide to composing engines, middlewares, tools, and sessions from JavaScript.
Topics:
- geppetto
- javascript
- goja
- user-guide
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

This guide is focused on writing JS files and validating behavior by executing those JS files directly.

If you need exact signatures for every method, use [JS API Reference](13-js-api-reference.md).

## JS-First Workflow

1. Write a JS script file.
2. Put assertions in the script (`assert(...)`).
3. Run it with:

```bash
go run ./cmd/examples/geppetto-js-lab --script <your-script.js>
```

4. Treat non-zero exit as failure and fix the script or pipeline setup.

## Suggested Project Layout

```text
examples/js/geppetto/
  01-07 core turns/session/tools scripts
  08-12 profile registry read/resolve scripts
  13-14 schema catalog scripts
  15-16 sqlite and mixed-stack scripts
  17-18 hard-cutover error-contract scripts
  19_profiles_connect_stack_runtime.js
```

You can copy these and branch them into your own scenario files.

## Workflow 1: Start with Turns and Blocks

Goal: verify payload and metadata shape before adding engines.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/01_turns_and_blocks.js
```

Use this stage to lock your turn schema and metadata keys.

## Workflow 2: Add Deterministic Session Inference

Goal: verify session lifecycle and history with no provider variability.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/02_session_echo.js
```

Keep this deterministic until basic state flow is stable.

## Workflow 3: Compose JS and Go Middlewares

Goal: verify middleware order and turn mutation behavior.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/03_middleware_composition.js
```

Default Go middleware names available here:

- `systemPrompt`
- `reorderToolResults`
- `turnLogging`

## Workflow 4: Register JS Tools and Enable Toolloop

Goal: verify tool-call to tool-use lifecycle using pure JS tools.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/04_tools_and_toolloop.js
```

## Workflow 5: Import and Call Go Tools from JS

Goal: confirm hybrid registry behavior (`JS tools + Go tools`).

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --list-go-tools
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/05_go_tools_from_js.js
```

## Workflow 6: Optional Live Provider Inference

Goal: final external smoke check after deterministic scripts pass.

Run:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/06_live_profile_inference.js
```

The script skips cleanly if no Gemini key is set (`GEMINI_API_KEY` or `GOOGLE_API_KEY`).

## Registry-Backed `fromProfile` (Hard Cutover)

`gp.engines.fromProfile(...)` is now registry-backed and no longer uses model/env fallback semantics.

Host requirement:

- module registration must include `Options.ProfileRegistry`.

Example:

```javascript
const gp = require("geppetto");

const engine = gp.engines.fromProfile("assistant", {
  runtimeKey: "chat",
  requestOverrides: {
    system_prompt: "Answer tersely."
  }
});

assert(engine.metadata && engine.metadata.runtimeFingerprint, "missing profile runtime fingerprint");
```

`requestOverrides` are still policy-gated by merged profile policy:

- disabled when `allow_overrides` is false,
- denied for keys listed in `denied_override_keys`,
- restricted to listed keys when `allowed_override_keys` is set.

Note: runtime `registrySlug` selection in `engines.fromProfile(...)` is removed. Registry resolution comes from the loaded registry stack.

## Runtime Stack Binding from JS

If the host did not inject `Options.ProfileRegistry`, scripts can bind registry sources directly:

```javascript
const gp = require("geppetto");

const connected = gp.profiles.connectStack([
  "examples/js/geppetto/profiles/10-provider-openai.yaml",
  "examples/js/geppetto/profiles/20-team-agent.yaml",
]);

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
console.log(connected.sources, resolved.registrySlug);

gp.profiles.disconnectStack();
```

`disconnectStack()` restores host baseline profile-registry wiring when one exists; otherwise it clears runtime-connected sources.

## Working Directly with `gp.profiles`

Use `gp.profiles` when you need registry inspection or CRUD from JS:

```javascript
const gp = require("geppetto");

const registries = gp.profiles.listRegistries();
const resolved = gp.profiles.resolve({ profileSlug: "assistant" });

console.log(registries.map((r) => r.slug));
console.log(resolved.runtimeFingerprint);
```

For mutations, host wiring must provide writable registry support.

## Working with `gp.schemas`

Use `gp.schemas` to inspect middleware and extension schema catalogs:

```javascript
const gp = require("geppetto");

const middlewareSchemas = gp.schemas.listMiddlewares();
const extensionSchemas = gp.schemas.listExtensions();

console.log(middlewareSchemas.map((x) => x.name));
console.log(extensionSchemas.map((x) => x.key));
```

Host requirements:

- `listMiddlewares()` requires `Options.MiddlewareSchemas`.
- `listExtensions()` requires `Options.ExtensionCodecs` and/or `Options.ExtensionSchemas`.

## Recommended Iteration Loop

1. Keep one script per behavior slice.
2. Use assertions for observable outcomes (block kinds, payload content, metadata changes).
3. Commit only scripts that are executable without manual edits.
4. Add live-provider scripts only after deterministic script set is green.

## Recording/Storage Hook Pattern

When the host wants to persist runs, pass hook references through builder
options or the chainable methods:

```javascript
const session = gp
  .createBuilder({
    engine,
    persister,
    eventSinks: [eventSink],
    snapshotHook,
  })
  .buildSession();
```

These map to the same runtime builder hooks:

- `withPersister(...)`
- `withEventSink(...)`
- `withSnapshotHook(...)`


## Writing Plugin Descriptors

For extractor and optimizer plugin scripts, import shared helpers from:

```javascript
const plugins = require("geppetto/plugins");
```

Use:

1. `plugins.defineExtractorPlugin(...)` for extractor descriptor scripts.
2. `plugins.wrapExtractorRun(...)` to normalize extractor `run` input.
3. `plugins.defineOptimizerPlugin(...)` for optimizer evaluator scripts.

Reference optimizer script:

1. `cmd/gepa-runner/scripts/toy_math_optimizer.js`

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `module geppetto not found` | host runtime did not register module | use `geppetto-js-lab` or register via `gp.Register(reg, opts)` |
| `engines.fromProfile requires a configured profile registry` | host module options omitted profile registry | pass `Options.ProfileRegistry` at module registration |
| `options.registrySlug has been removed` | script still passes `registrySlug` into `fromProfile` | remove runtime selector; load registries in stack order and resolve by profile slug |
| `no go tool registry configured` | script calls `useGoTools` in host without Go registry | run with `geppetto-js-lab` or configure `Options.GoToolRegistry` |
| tool loop does not execute | registry not bound to builder | call `.withTools(reg, { enabled: true })` |
| recording hooks ignored | non-hook values passed into builder options | pass Go `TurnPersister` / `EventSink` / `SnapshotHook` references |
| unstable output in live script | provider variability | keep deterministic checks in `echo`/`fromFunction` scripts |

## See Also

- [JS API Reference](13-js-api-reference.md)
- [JS API Getting Started Tutorial](../tutorials/05-js-api-getting-started.md)
- [Tools](07-tools.md)
- [Middlewares](09-middlewares.md)
- [Sessions](10-sessions.md)
