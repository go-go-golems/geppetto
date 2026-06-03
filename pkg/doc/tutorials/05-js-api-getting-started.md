---
Title: Getting Started with the Geppetto JavaScript API
Slug: geppetto-js-api-getting-started
Short: Build and run a session-centered Geppetto JavaScript script.
Topics:
- geppetto
- javascript
- goja
Commands: []
Flags: []
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial shows the hard-cut wrapper-first Geppetto JavaScript API:

```js
const gp = require("geppetto");
```

The current public execution path is session-centered:

1. Resolve registry settings.
2. Build an agent.
3. Build a session.
4. Execute `session.next().run()` or `session.next().runAsync()`.

## Resolve a profile

```js
const settings = gp.inferenceProfiles.resolve("default");
console.log(settings.toJSON().chat.engine);
```

Hosts can configure the default registry. The example runner also accepts registry flags:

```bash
go run ./cmd/examples/geppetto-js-run run \
  --script examples/js/geppetto/30_real_provider_multiturn.js \
  --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
  --profile default \
  --timeout-ms 120000
```

## Build an agent and session

```js
const agent = gp.agent()
  .name("getting-started")
  .inference(settings)
  .runDefaults({ timeoutMs: 120000 })
  .build();

const session = agent.session()
  .id("getting-started-chat")
  .build();
```

Agents own provider/tool/event configuration. Sessions own conversational progression.

## Run a first step

```js
const result = session.next()
  .system("Answer in one sentence.")
  .user("What is a session-centered API?")
  .run();

console.log(result.text());
```

## Continue the same session

```js
const second = session.next()
  .user("Summarize your previous answer in five words.")
  .run();

console.log(second.text());
```

No public turn builder is required. `session.next()` uses the latest final turn as context and produces a new final turn.

## Use async execution

```js
const EventEmitter = require("events");
const events = new EventEmitter();
events.on("text-delta", ev => { if (ev.delta) process.stdout.write(ev.delta); });

const asyncAgent = gp.agent()
  .inference(settings)
  .events(events)
  .build();

const asyncSession = asyncAgent.session().id("async-chat").build();
const handle = asyncSession.next()
  .user("Write a short answer.")
  .runAsync();

const asyncResult = await handle.promise;
```

`handle.cancel()` cancels the in-flight run. `handle.close()` releases the handle. Event listeners belong on the builder-level EventEmitter.

## Persist and resume

If the host provides storage, use `gp.turnStores`:

```js
const store = gp.turnStores.default();

const durableAgent = gp.agent()
  .inference(settings)
  .store(store)
  .build();

const durableSession = durableAgent.session()
  .id("durable-chat")
  .store(store)
  .resumeLatest()
  .build();

const durableResult = durableSession.next()
  .user("Continue this durable conversation.")
  .run();
```

`resumeLatest()` starts empty when there is no previous stored final turn. Use `resumeLatest({ required: true })` for strict resume.

## Fork a conversation

```js
const fork = session.fork()
  .id("getting-started-chat-fork")
  .build();

const forkResult = fork.next()
  .user("Give an alternative answer.")
  .run();
```

Forks preserve imported history as base context and create new turn identities for derived runs.

## What is intentionally not public

Older scripts may refer to APIs that are no longer public:

- `gp.turn(...)`
- `agent.run(turn)`
- `agent.runAsync(turn)`
- `gp.turns.newTurn(...)`
- `gp.events.collector()`
- legacy `createBuilder`, `createSession`, and `runInference`

Use `agent.session()`, `session.next()`, `session.fork()`, and storage-backed `resumeLatest()` instead.
