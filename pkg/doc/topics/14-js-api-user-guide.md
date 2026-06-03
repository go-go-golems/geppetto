---
Title: Geppetto JavaScript API User Guide
Slug: geppetto-js-api-user-guide
Short: Practical guide for writing scripts against the hard-cut Geppetto JavaScript API.
Topics:
- geppetto
- javascript
- goja
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Use `require("geppetto")` from goja/xgoja hosts to access Geppetto's wrapper-first JavaScript API:

```js
const gp = require("geppetto");
```

The public execution model is session-centered. Scripts build an agent, create a session, then run explicit `session.next()` steps. Public `gp.turn(...)`, `agent.run(turn)`, and `agent.runAsync(turn)` are intentionally absent.

## 1. Resolve settings and build an agent

```js
const settings = gp.inferenceProfiles.resolve("default");

const agent = gp.agent()
  .name("assistant")
  .inference(settings)
  .runDefaults({ timeoutMs: 120000 })
  .build();
```

Use registry-resolved `InferenceSettings` wrappers. Plain JavaScript settings maps are rejected.

## 2. Create a session and run the first step

```js
const session = agent.session()
  .id("chat-123")
  .name("Demo chat")
  .build();

const result = session.next()
  .system("Answer in one concise paragraph.")
  .user("What is Geppetto?")
  .run();

console.log(result.text());
```

`session.next()` defines the run boundary. It starts from the latest session context, appends the blocks supplied by the builder, and records the final turn back into the session.

## 3. Continue a conversation

```js
const first = session.next()
  .user("Reply with exactly: ALPHA_GEPPETTO")
  .run();

const second = session.next()
  .user("What exact token did you just return?")
  .run();
```

The second call sees the first call's final turn as context. You do not pass output turns back into `agent.run(...)`; the session owns the progression.

## 4. Run asynchronously with EventEmitter events

Attach an EventEmitter at agent-builder time, then call `runAsync()` on a session turn builder:

```js
const EventEmitter = require("events");
const events = new EventEmitter();

events.on("text-delta", ev => {
  if (ev.delta) console.log(ev.delta);
});

const asyncAgent = gp.agent()
  .inference(settings)
  .events(events)
  .build();

const asyncSession = asyncAgent.session().id("streaming-chat").build();
const handle = asyncSession.next()
  .user("Write a short streaming answer.")
  .runAsync({ timeoutMs: 120000 });

const asyncResult = await handle.promise;
```

`runAsync()` returns `{ promise, cancel, close }`. There is no public `handle.on(...)`; register listeners on the EventEmitter before starting the run.

## 5. Use durable turn stores

Turn storage is host-configured. JavaScript receives Go-owned `TurnStore` wrappers from `gp.turnStores`, then opts agents or sessions into explicit persistence:

```js
const store = gp.turnStores.default();

const durableAgent = gp.agent()
  .inference(settings)
  .store(store)
  .build();

const durableSession = durableAgent.session()
  .id("chat-123")
  .store(store)
  .resumeLatest()
  .build();

const result = durableSession.next()
  .user("Continue from the stored conversation.")
  .run();
```

`resumeLatest()` is non-strict by default: if no stored final turn exists, the session starts empty. Use `resumeLatest({ required: true })` to fail instead. Use `.persist(false)` or agent `.persistTo(null)` to disable inherited host-default persistence.

## 6. Fork a session

Forking creates a new `SessionBuilder` pre-seeded from an existing session turn:

```js
const fork = session.fork()
  .id("chat-123-branch")
  .build();

const forkResult = fork.next()
  .user("Answer from a different angle.")
  .run();
```

The imported base turn remains historical evidence. The first derived `next()` clears the copied turn id before executing a new run.

## 7. Build tools and schemas

```js
const input = gp.schema.object()
  .property("value", gp.schema.string().description("Value to echo"))
  .required("value")
  .build();

const echo = gp.tool("echo_value")
  .description("Echo a value")
  .input(input)
  .handler((args, ctx) => ({ echoed: args.value, toolName: ctx.toolName }))
  .build();

const registry = gp.toolRegistry().add(echo);
const agentWithTools = gp.agent()
  .inference(settings)
  .tool(registry)
  .toolLoop({ maxIterations: 3 })
  .build();
```

If the host configured a Go tool registry, select host tools directly with `goTool(name)`:

```js
const agentWithHostTool = gp.agent()
  .inference(settings)
  .goTool("search")
  .toolLoop({ maxIterations: 3 })
  .build();
```

Use `tool(registry)` for JavaScript-defined tools or explicit wrapper registries. Use `goTool(name)` for tools supplied by the embedding Go application.

## 8. Multimodal user input

Use the session turn builder's message callback:

```js
const result = session.next()
  .user(m => m
    .text("Describe this image.")
    .imageURL("https://example.invalid/screenshot.png"))
  .run();
```

## Migration notes

Use the following replacements when updating older scripts:

- `gp.turn().user(...).build()` + `agent.run(turn)` → `agent.session().build().next().user(...).run()`
- `gp.turn(previousTurn)` → `session.next()` or `session.fork().base(previousTurn)` depending on lifecycle intent
- `agent.runAsync(turn)` → `session.next().user(...).runAsync()`
- `agent.persistTo(store)` → `agent.store(store)` or `session.store(store)`; `persistTo` remains as an alias for host persistence selection

Removed legacy APIs include `gp.turn`, `gp.turns`, `gp.events`, `gp.chat`, `gp.inferenceSettings`, `createBuilder`, `createSession`, `runInference`, and public `agent.run` / `agent.runAsync`.
