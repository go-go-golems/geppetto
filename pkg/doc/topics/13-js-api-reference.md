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

The API is Go-wrapper-first: scripts receive Go-owned wrapper objects and ask for explicit snapshots with `toJSON()`. Legacy map/session/runner exports and public turn-run execution were removed from the default public surface. Public execution now starts from an `AgentSession`:

```js
const session = gp.agent().inference(settings).build()
  .session().id("chat-123").build();

const result = session.next()
  .system("Be brief.")
  .user("Hello")
  .run();
```

## `gp.inferenceProfiles`

`gp.inferenceProfiles` loads and resolves Geppetto engine profile registries. It does **not** load Pinocchio unified app config documents with `app:` blocks.

```js
const registry = gp.inferenceProfiles.load("./profiles.yaml");
const settings = registry.resolve("assistant");
```

Supported source forms include YAML paths, `yaml:PATH`, `yaml://PATH`, SQLite paths, `sqlite:PATH`, and `sqlite-dsn:DSN`.

Namespace methods:

- `load(source: string | string[]): InferenceRegistry`
- `resolve(input?: string | { registry?, registrySlug?, profile?, profileSlug? }): InferenceSettings`
- `default(): InferenceRegistry`

Registry methods:

- `listRegistries()`
- `listProfiles(registrySlug?)`
- `resolve(input?)`
- `close()`
- `sources`

`InferenceSettings` is read-only and Go-owned:

- `toJSON()` returns a sanitized snapshot.
- `clone()` returns another settings wrapper.

## `gp.engine()`

Build a provider engine from resolved settings:

```js
const engine = gp.engine().inference(settings).build();
```

Plain JavaScript settings objects are rejected; pass a Go-owned `InferenceSettings` wrapper.

## `gp.agent()`

Build an agent, then create sessions from it:

```js
const agent = gp.agent()
  .name("assistant")
  .inference(settings)
  .events(emitter)
  .runDefaults({ timeoutMs: 120000 })
  .build();

const session = agent.session().id("chat-123").build();
```

Agent builder methods include:

- `name(name)`
- `inference(settings)`
- `engine(engine)`
- `events(emitter)` for builder-level EventEmitter delivery
- `tool(toolOrRegistry)` / `goTool(name)` and `toolLoop(options)`
- `store(turnStore)` / `persistTo(turnStore)`
- `defaultStore(enabled?)` / `persistDefault(enabled?)` / `persist(enabled?)`
- `runDefaults(options)`
- `build()`

Built agents expose:

- `name`
- `session(): SessionBuilder`

They intentionally do **not** expose `run(turn)`, `runAsync(turn)`, or `ask(...)`.

## Sessions

`agent.session()` returns a `SessionBuilder`:

```js
const session = agent.session()
  .id("chat-123")
  .name("Support chat")
  .metadata("tenant", "demo")
  .runDefaults({ timeoutMs: 120000 })
  .build();
```

Session builder methods:

- `id(id)` sets the session id.
- `name(name)` sets a human-readable name.
- `base(turn)` imports a Go-owned `TurnWrapper` as historical base context.
- `store(turnStore)` selects a store for persistence/resume.
- `defaultStore()` selects the host default store.
- `persist(enabled?)` enables or disables persistence for this session.
- `resumeLatest(query?)` loads the latest stored final turn into the session base; pass `{ required: true }` to fail if none exists.
- `resumeNone()` disables resume.
- `metadata(key, value)` attaches session metadata.
- `runDefaults(options)` overrides run defaults for this session.
- `build()` returns an `AgentSession`.

`AgentSession` methods:

- `id()` and `name()`
- `next(): SessionTurnBuilder`
- `fork(options?): SessionBuilder`
- `latestTurn(): TurnWrapper | null`
- `turns(): TurnWrapper[]`
- `turn(index): TurnWrapper | null`
- `turnCount(): number`
- `isRunning(): boolean`
- `cancel()`
- `close()`

`session.next()` clones the latest context, clears any copied turn id for the derived run, appends requested blocks, and runs against the agent engine:

```js
const result = session.next()
  .user("Continue from the previous answer.")
  .run({ timeoutMs: 120000 });
```

`session.fork()` returns a pre-seeded `SessionBuilder`:

```js
const fork = session.fork().id("chat-123-fork").build();
```

## `SessionTurnBuilder`

`session.next()` returns a builder with explicit block methods:

- `system(text)`
- `user(text | (messageBuilder) => messageBuilder)`
- `assistant(text)`
- `metadata(key, value)`
- `run(options?)`
- `runAsync(options?)`

Multimodal user input uses the message builder callback:

```js
const result = session.next()
  .user(m => m.text("Describe this image").imageURL("https://example.invalid/image.png"))
  .run();
```

`runAsync()` returns `{ promise, cancel, close }` and uses the agent builder-level EventEmitter for live events:

```js
const handle = session.next().user("Stream a short answer.").runAsync();
const result = await handle.promise;
```

## Results and turn wrappers

`run()` and `runAsync().promise` resolve to a `RunResult` wrapper:

- `text()` returns assistant text.
- `inputTurn()` returns the run input turn.
- `effectiveTurn()` returns the effective turn sent to the engine.
- `outputTurn()` returns the final output turn.
- `toJSON()` returns a snapshot.

`TurnWrapper` objects are still public as snapshots/results/persistence data. They expose `toJSON()` and `clone()`, but JavaScript no longer constructs public turns with `gp.turn(...)`.

## `gp.turnStores`

`gp.turnStores` exposes host-configured durable turn stores as Go-owned wrappers. Geppetto does not open SQLite files directly from JavaScript; xgoja hosts such as Pinocchio provide stores through module/provider configuration.

```js
const store = gp.turnStores.default();
const session = agent.session()
  .id("chat-123")
  .store(store)
  .resumeLatest()
  .build();
```

Methods:

- `gp.turnStores.default(): TurnStore`
- `gp.turnStores.get(name): TurnStore`
- `store.name()`
- `store.list(query?)`
- `store.loadLatest(query?)`
- `store.close()`

## Tools and schema

Tool and JSON schema wrappers remain Go-owned:

```js
const input = gp.schema.object()
  .property("value", gp.schema.string())
  .required("value")
  .build();

const echo = gp.tool("echo_value")
  .description("Echo a value")
  .input(input)
  .handler(args => ({ echoed: args.value }))
  .build();

const registry = gp.toolRegistry().add(echo);
```

Host applications can also expose a Go-owned tool registry. Use `agent.goTool(name)` to select one of those host tools by name without constructing a JavaScript tool registry:

```js
const agent = gp.agent()
  .inference(settings)
  .goTool("search")
  .toolLoop({ maxIterations: 4 })
  .build();
```

`goTool(name)` resolves against an explicit `agent.tool(registry)` registry when one is set; otherwise it resolves against the module's host-provided Go tool registry.

## Removed public exports

The hard-cut surface intentionally omits legacy and turn-run APIs, including:

- `gp.turn`
- `gp.turns`
- `gp.events`
- `gp.chat`
- `gp.inferenceSettings`
- `gp.createBuilder`
- `gp.createSession`
- `gp.runInference`
- `gp.engines`, `gp.profiles`, `gp.runner`, `gp.schemas`, `gp.middlewares`, `gp.tools`
- `agent.run(turn)` and `agent.runAsync(turn)`
