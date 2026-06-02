---
Title: Geppetto JavaScript API User Guide
Slug: geppetto-js-api-user-guide
Short: Practical guide to composing registry-backed engines, agents, turns, tools, and multimodal messages from JavaScript.
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

This guide shows the wrapper-first Geppetto JavaScript API. The core rule is simple:

1. Registry files own model/provider settings.
2. JavaScript receives Go-owned wrappers.
3. Agents run explicit turns only.

## Run Scripts

Use the real example runner when you need profile flags:

```bash
go run ./cmd/examples/geppetto-js-run run \
  --script examples/js/geppetto/30_real_provider_multiturn.js \
  --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
  --profile default
```

The runner exposes `require("geppetto")` and configures the host-default profile registry for `gp.inferenceProfiles.resolve(...)`.

## Load a Registry and Resolve Settings

```js
const gp = require("geppetto");

const registry = gp.inferenceProfiles.load("examples/js/geppetto/profiles/50-hardcut-phase123.yaml");
const settings = registry.resolve("assistant");

console.log(settings.toJSON().chat.engine);
registry.close();
```

`settings` is a Go-owned `InferenceSettings` wrapper. Use `toJSON()` for a detached redacted snapshot. Do not mutate model parameters in JavaScript; change the registry profile instead.

## Build an Engine

```js
const engine = gp.engine()
  .inference(settings)
  .build();
```

`engine().inference(...)` accepts only registry-resolved `InferenceSettings` wrappers. Plain JavaScript objects are rejected.

## Build an Agent and Explicit Turn

```js
const agent = gp.agent()
  .name("assistant")
  .inference(settings)
  .build();

const turn = gp.turn()
  .system("Answer in one short sentence.")
  .user("What changed in this repository?")
  .build();

const result = agent.run(turn);
console.log(result.text());
```

Agents intentionally do not expose `ask(prompt)` or `system(prompt)`. System/user content belongs in the turn.

## Live Events with `runAsync` and EventEmitter

Use `agent.run(turn)` when you only need a blocking final result. Use `agent.runAsync(turn)` when JavaScript should receive provider/tool-loop events while inference is still running.

The first-pass event API is builder-level: create an EventEmitter, register listeners, pass it to `.events(emitter)`, and then call `runAsync`.

```js
const EventEmitter = require("events");

const events = new EventEmitter();
const counts = Object.create(null);
const deltas = [];

events.on("event", ev => {
  counts[ev.type] = (counts[ev.type] || 0) + 1;
});

events.on("text-delta", ev => {
  if (ev.delta) deltas.push(ev.delta);
});

events.on("inference-error", ev => {
  console.error(ev.message || ev.error);
});

const agent = gp.agent()
  .name("assistant")
  .inference(settings)
  .events(events)
  .build();

const handle = agent.runAsync(turn, { timeoutMs: 120000 });
const result = await handle.promise;

console.log(result.text());
console.log(counts, deltas.join(""));
```

Important constraints:

- Register listeners before `.events(emitter).build()` or at least before `runAsync(...)` starts.
- `runAsync` returns `{ promise, cancel, close }`.
- `cancel()` and `close()` cancel the active run.
- `handle.on(...)` is intentionally not part of the API.
- `agent.runAsync(turn, { events })` is intentionally deferred; use builder-level `.events(emitter)` for now.
- Canonical Geppetto `error` events emit as `inference-error`, not Node's special `error` event.
- Each single Geppetto event attempts generic `event` delivery first and the type-specific channel second; no global ordering is guaranteed across concurrent publishers.
- `runAsync` lifecycle uses `handle.promise`, `cancel()`, and `close()`; there are intentionally no JS-only `runasync-*` lifecycle events.
- `gp.events.collector()` was removed; use `require("events")` EventEmitter values instead.

See:

- `examples/js/geppetto/31_event_emitter_run_async.js`
- `examples/js/geppetto/32_event_emitter_progress_summary.js`
- `examples/js/geppetto/33_event_emitter_multiturn_run_async.js`

## Multi-Turn Inference

The API is explicit: a second provider call receives history only if the script includes that history in the next turn.

```js
const first = gp.turn()
  .system("Be terse.")
  .user("Remember the token ALPHA.")
  .build();
const firstResult = agent.run(first);

const second = gp.turn()
  .system("Be terse.")
  .user("Remember the token ALPHA.")
  .assistant(firstResult.text())
  .user("What token did you just mention?")
  .build();
const secondResult = agent.run(second);
```

See `examples/js/geppetto/30_real_provider_multiturn.js` for a live provider example using `~/.config/pinocchio/profiles.yaml`.

## Tools and Schemas

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
console.log(registry.call("echo_value", { value: "hello" }));
```

Pass the registry into an agent with `.tool(registry)`.

## Multimodal Turns

```js
const turn = gp.turn()
  .system("You are a careful visual reasoning assistant.")
  .user(m => m
    .text("What is in this screenshot?")
    .imageFile("./screenshot.png"))
  .build();
```

Message builder methods:

- `text(...)`
- `imageURL(...)`
- `imageFile(...)`
- `imageBytes(...)`

## Pinocchio Profiles vs Pinocchio Config Docs

`~/.config/pinocchio/profiles.yaml` is usable here because it is a Geppetto registry-shaped file with top-level `slug` and `profiles` keys.

`gp.inferenceProfiles.load(...)` does not load Pinocchio unified app config documents with `app:`/application settings. Those remain application-side host concerns.

## xgoja Provider Config

Generated xgoja hosts can configure Geppetto like this:

```json
{
  "profileRegistries": ["/home/me/.config/pinocchio/profiles.yaml"],
  "defaultProfile": "default",
  "allowRegistryLoad": true
}
```

`allowRegistryLoad` is false by default.

## Current Gaps

Implemented now:

- registry-resolved `InferenceSettings`
- engine builder
- agent and explicit turns
- schema/tool/toolRegistry wrappers
- text/image message builder
- xgoja provider registry loading

Still deferred:

- symbolic host credential resolver
- additional schema helpers such as min/max/default
- `turn().toolCall(...)` and `turn().toolResult(...)`
- hard-cut `gp.embeddings()` wrapper
