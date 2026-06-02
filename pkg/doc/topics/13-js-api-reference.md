---
Title: Geppetto JavaScript API Reference
Slug: geppetto-js-api-reference
Short: Reference for the native `require("geppetto")` API exposed through goja.
Topics:
- geppetto
- javascript
- goja
- api-reference
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page documents the hard-cut JavaScript API exposed by:

```js
const gp = require("geppetto");
```

The API is Go-wrapper-first: scripts receive Go-owned wrapper objects and ask for explicit snapshots with `toJSON()`. Legacy map/session/runner exports were removed from the default public surface.

## Running Scripts

For local examples that need profile registry flags, use:

```bash
go run ./cmd/examples/geppetto-js-run run \
  --script examples/js/geppetto/30_real_provider_multiturn.js \
  --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
  --profile default \
  --timeout-ms 120000
```

## `gp.inferenceProfiles`

`gp.inferenceProfiles` loads and resolves Geppetto engine profile registries. It does **not** load Pinocchio unified app config documents with `app:` blocks.

```js
const registry = gp.inferenceProfiles.load("./profiles.yaml");
const settings = registry.resolve("assistant");
```

Supported source forms:

- YAML path
- `yaml:PATH`
- `yaml://PATH`
- SQLite path (`.db`, `.sqlite`, `.sqlite3`)
- `sqlite:PATH`
- `sqlite-dsn:DSN`

Namespace methods:

- `gp.inferenceProfiles.load(source: string | string[]): InferenceRegistry`
- `gp.inferenceProfiles.resolve(input?: string | { registry?, registrySlug?, profile?, profileSlug? }): InferenceSettings`
- `gp.inferenceProfiles.default(): InferenceRegistry`

Registry methods:

- `listRegistries()`
- `listProfiles(registrySlug?)`
- `resolve(input?)`
- `close()`
- `sources`

## Geppetto Registry YAML

Current runtime YAML is one file = one registry:

```yaml
slug: local
profiles:
  assistant:
    display_name: Assistant
    stack:
      - profile_slug: openai-base
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4o-mini
```

Important fields:

- `slug`: registry identifier.
- `profiles.<slug>`: named inference profile.
- `profiles.<slug>.stack`: optional list of base profiles to merge before the leaf profile.
- `profiles.<slug>.inference_settings`: Geppetto inference settings for provider/model/client/model info.
- `default_profile_slug`: present in Go registry structs and some design docs, but the current runtime YAML loader rejects it; use a `default` profile or resolve by explicit profile slug.

## `InferenceSettings`

`InferenceSettings` objects are Go-owned wrappers returned by registry resolution. There is no public `gp.inferenceSettings()` builder.

Methods:

- `toJSON()` — detached redacted snapshot.
- `clone()` — another Go-owned wrapper.
- `debug()` — redacted diagnostic view.

Provider/model/temperature/token changes belong in registry files, not JavaScript setters.

## `gp.engine()`

Build engines from registry-resolved settings:

```js
const engine = gp.engine().inference(settings).build();
```

`engine().inference(...)` rejects plain JavaScript objects. It accepts Go-owned `InferenceSettings` wrappers only.

## `gp.turn()`

Build explicit Go-owned turns:

```js
const turn = gp.turn()
  .system("Answer briefly.")
  .user("Hello")
  .build();
```

Methods:

- `system(text)`
- `user(text)`
- `user(messageBuilderFn)`
- `assistant(text)`
- `metadata(key, value)`
- `build()`

The returned turn wrapper exposes:

- `toJSON()`
- `clone()`

### Message Builder

```js
const turn = gp.turn()
  .user(m => m
    .text("What is in this image?")
    .imageURL("https://example.invalid/screenshot.png"))
  .build();
```

Methods:

- `text(text)`
- `imageURL(url, options?)`
- `imageFile(path)`
- `imageBytes(base64Content, mediaType?)`

## `gp.agent()`

Agents own runtime behavior, not prompt strings. Execution requires explicit turns.

```js
const agent = gp.agent()
  .name("assistant")
  .inference(settings)
  .build();

const result = agent.run(turn);
console.log(result.text());

const EventEmitter = require("events");
const events = new EventEmitter();
events.on("text-delta", ev => process.stdout.write(ev.delta));

const asyncAgent = gp.agent()
  .inference(settings)
  .events(events)
  .build();
const handle = asyncAgent.runAsync(turn);
const asyncResult = await handle.promise;
```

Builder methods:

- `name(name)`
- `inference(settings)`
- `engine(engine)`
- `middleware(middleware)`
- `goMiddleware(name, options?)`
- `tool(registry)`
- `goTool(name)`
- `toolLoop(options?)`
- `events(sink)` accepts a Go `EventSink` wrapper or a go-go-goja `EventEmitter` from `require("events")`.
- `runDefaults(options?)`
- `build()`

Agent methods:

- `run(turn, options?)` — synchronous final-result execution.
- `runAsync(turn, options?)` — non-blocking execution returning `{ promise, cancel, close }`; use builder-level `.events(emitter)` for live events.

Intentionally absent:

- `agent.ask(prompt)`
- `agent.system(prompt)`
- `agent.profile(name)`
- `agent.inferenceProfile(name)`
- `agent.stream(...)`
- `agent.runAsync(turn, { events })`
- `handle.on(...)`

## `RunOptions`

Both `run` and `runAsync` accept the same run options:

```ts
interface RunOptions {
  timeoutMs?: number;
  tags?: Record<string, any>;
}
```

`timeoutMs` cancels the Go run context after the requested number of milliseconds. `tags` are attached to runtime metadata and are useful for tracing examples, tests, and host integrations.

## `runAsync` and EventEmitter Events

`runAsync` is the live-event execution path. It starts inference in Go and immediately returns control to JavaScript so go-go-goja can deliver EventEmitter callbacks on the runtime owner thread.

The first-pass event API is builder-level only:

```js
const EventEmitter = require("events");
const events = new EventEmitter();

const seen = [];
events.on("event", ev => seen.push(ev.type));
events.on("text-delta", ev => process.stdout.write(ev.delta));
events.on("inference-error", ev => console.error(ev.message || ev.error));

const agent = gp.agent()
  .inference(settings)
  .events(events)
  .build();

const handle = agent.runAsync(turn, { timeoutMs: 120000 });
const result = await handle.promise;
```

Handle shape:

```ts
interface AgentAsyncHandle {
  promise: Promise<RunResult>;
  cancel(): void;
  close(): void;
}
```

`cancel()` and `close()` both cancel the active inference run. `runAsync` does not expose `handle.on(...)`; register listeners on the EventEmitter before passing it to `.events(...)`.

Every Geppetto event is emitted twice:

1. as `"event"`, for generic logging and counting;
2. as its type-specific event name, such as `"text-delta"`, `"provider-call-started"`, or `"tool-result-ready"`.

Canonical Geppetto `error` events are emitted as `"inference-error"` for the type-specific channel, not Node's special `"error"` event. They still appear on the generic `"event"` channel with `ev.type === "error"`.

Common payload fields:

- `type`: canonical Geppetto event type.
- `timestampMs`: JavaScript-facing emission timestamp.
- `sessionId`, `inferenceId`, `turnId`: correlation identifiers when available.
- `correlation`: structured run/provider/segment/tool correlation when available.
- `metaExtra`: provider or runtime metadata when available.
- `rawPayload`: raw provider payload string when the event carries one.

Type-specific payload fields:

- `text-delta`: `delta`, `text`, `sequence`.
- `text-segment-finished`: `text`, `finishReason`.
- `reasoning-delta`: `delta`, `text`, `sequence`, `source?`.
- `reasoning-segment-finished`: `text`, `finishReason`, `source?`.
- `tool-call-started`: `toolCall.id`, `toolCall.name`.
- `tool-call-arguments-delta`: `toolCall.id`, `toolCall.delta`, `toolCall.arguments`, `toolCall.sequence`.
- `tool-call-requested`: `toolCall.id`, `toolCall.name`, `toolCall.input`.
- `tool-execution-started`: `toolCall.id`, `toolCall.name`, `toolCall.input`.
- `tool-result-ready`: `toolResult.id`, `toolResult.name`, `toolResult.result`, `toolResult.status`.
- `tool-call-finished`: `toolCall.id`, `toolCall.name`, `toolCall.status`.
- `inference-error`: `error`, `message`.
- `interrupt`: `text`.

Provider support varies. Some providers emit many `text-delta` events; others may emit only lifecycle/final events. Always use `result.text()` as the final answer source.

Ordering semantics:

- For each single Geppetto event, the adapter attempts to emit the generic `"event"` channel first and the type-specific channel second.
- If one channel cannot be scheduled, the adapter still attempts the other channel and logs the failed channel.
- Ordering is not globally guaranteed across concurrent provider/tool publishers.
- The EventEmitter carries Geppetto/provider/tool events only. `runAsync` lifecycle is represented by `handle.promise`, `cancel()`, and `close()`; there are intentionally no adapter-specific `runasync-*` events.
- `gp.events.collector()` was removed. Use `require("events")` and builder-level `.events(emitter)` for JavaScript event handling.

Troubleshooting:

- `geppetto events: jsevents manager is not installed`: the host registered `require("geppetto")` without installing go-go-goja `jsevents.Install()` or without passing an `EventEmitterManagerResolver` to the Geppetto module options.
- Listener throws: Geppetto-owned runtimes log asynchronous EventEmitter listener dispatch failures; provider/xgoja hosts should install `jsevents.Install(jsevents.WithErrorHandler(...))` or equivalent host diagnostics.
- Missing profile: registry-backed examples reject with `GoError: profile not found`; check `--profile-registries`, `--profile`, and default profile configuration.
- No `text-delta`: some providers or adapters do not stream token deltas. Listen to the generic `event` channel to inspect observed event types and use `result.text()` for final output.

Examples:

- `examples/js/geppetto/31_event_emitter_run_async.js`
- `examples/js/geppetto/32_event_emitter_progress_summary.js`
- `examples/js/geppetto/33_event_emitter_multiturn_run_async.js`

## `RunResult`

`agent.run(turn)` returns a Go-owned result wrapper:

- `inputTurn()`
- `effectiveTurn()`
- `outputTurn()`
- `text()`
- `usage()`
- `stopReason()`
- `events()`
- `toJSON()`

`inputTurn` is the caller's turn snapshot. `effectiveTurn` includes runtime metadata before inference. `outputTurn` is the provider/runtime output turn.

## `gp.schema`

Schema builders produce Go-owned schema wrappers for tools.

```js
const input = gp.schema.object()
  .property("value", gp.schema.string().description("Value to echo"))
  .required("value")
  .build();
```

Constructors:

- `string()`
- `integer()`
- `number()`
- `boolean()`
- `array()`
- `object()`
- `enum(...values)`

Builder methods:

- `description(text)`
- `property(name, schema)`
- `items(schema)`
- `required(...names)`
- `build()`
- `toJSON()`

## `gp.tool()` and `gp.toolRegistry()`

```js
const echo = gp.tool("echo_value")
  .description("Echo a value")
  .input(input)
  .handler((args, ctx) => ({ echoed: args.value }))
  .build();

const registry = gp.toolRegistry().add(echo);
console.log(registry.call("echo_value", { value: "hello" }));
```

Tool builder methods:

- `description(text)`
- `input(schema)`
- `handler(fn)`
- `build()`

Tool registry methods:

- `add(tool)`
- `addGo(...names)`
- `list()`
- `call(name, input?)`

## xgoja Provider Configuration

The Geppetto xgoja provider accepts registry configuration:

```json
{
  "profileRegistries": ["/home/me/.config/pinocchio/profiles.yaml"],
  "defaultProfile": "default",
  "allowRegistryLoad": true
}
```

`allowRegistryLoad` defaults to false. This prevents generated hosts from loading arbitrary registry paths unless explicitly allowed.

For EventEmitter support, generated/xgoja hosts must provide the same runtime integration that `pkg/js/runtime.NewRuntime` installs automatically:

```go
opts.RuntimeOwner = ctx.Owner
opts.EventEmitterManagerResolver = func() (*jsevents.Manager, bool) {
    value, ok := ctx.Value(jsevents.RuntimeValueKey)
    if !ok {
        return nil, false
    }
    manager, ok := value.(*jsevents.Manager)
    return manager, ok && manager != nil
}
```

Hosts should also install listener diagnostics with `jsevents.Install(jsevents.WithErrorHandler(...))` or an equivalent runtime initializer. Without the manager resolver, `.events(new EventEmitter())` fails with `geppetto events: jsevents manager is not installed`.

## Removed Legacy Surface

The following old public names are intentionally absent from the hard-cut surface:

- `gp.turns`
- `gp.engines`
- `gp.profiles`
- `gp.runner`
- `gp.schemas`
- `gp.middlewares`
- `gp.tools`
- `gp.createBuilder`
- `gp.createSession`
- `gp.runInference`
- `gp.events`
- `gp.events.collector`

Use the wrapper-first APIs in this reference instead.
