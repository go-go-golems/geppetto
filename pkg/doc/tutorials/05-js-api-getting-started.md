---
Title: Getting Started with the Geppetto JavaScript API
Slug: geppetto-js-api-getting-started
Short: Step-by-step tutorial for registry-backed settings, agents, explicit turns, tools, and multimodal messages from JavaScript.
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

This tutorial introduces the wrapper-first Geppetto JavaScript API.

Reference docs:

- [JS API Reference](../topics/13-js-api-reference.md)
- [JS API User Guide](../topics/14-js-api-user-guide.md)

## Prerequisites

Run commands from the Geppetto repository root.

## Step 1: Resolve a Profile

Create or use an existing Geppetto registry file. A minimal registry looks like:

```yaml
slug: local
profiles:
  assistant:
    inference_settings:
      api:
        api_keys:
          openai-api-key: test-key
      chat:
        api_type: openai
        engine: gpt-4o-mini
```

Resolve it from JavaScript:

```js
const gp = require("geppetto");

const registry = gp.inferenceProfiles.load("examples/js/geppetto/profiles/50-hardcut-phase123.yaml");
const settings = registry.resolve("assistant");

console.log(settings.toJSON().chat.engine);
registry.close();
```

Run the checked-in example:

```bash
go test ./pkg/js/modules/geppetto -run TestPhase123ExampleScripts -count=1 -v
```

## Step 2: Build an Engine

```js
const engine = gp.engine()
  .inference(settings)
  .build();
```

See:

- `examples/js/geppetto/26_engine_builder_from_registry_profile.js`

## Step 3: Build an Agent and Turn

```js
const agent = gp.agent()
  .name("assistant")
  .inference(settings)
  .build();

const turn = gp.turn()
  .system("Answer briefly.")
  .user("Hello")
  .build();

const result = agent.run(turn);
console.log(result.text());
```

For a deterministic local example, see:

- `examples/js/geppetto/28_agent_from_registry_profile.js`

## Step 4: Multi-Turn Context

Geppetto does not hide conversation state in `agent.ask(...)`. To do multi-turn inference, construct the next explicit turn with the history you want the provider to see:

```js
const result1 = agent.run(gp.turn()
  .system("Be concise.")
  .user("Return ALPHA.")
  .build());

const result2 = agent.run(gp.turn()
  .system("Be concise.")
  .user("Return ALPHA.")
  .assistant(result1.text())
  .user("What did you return previously?")
  .build());
```

Run the real provider example:

```bash
./examples/js/geppetto/run_real_provider_multiturn.sh
```

Override profile settings with environment variables:

```bash
GEPPETTO_PROFILE_REGISTRIES="$HOME/.config/pinocchio/profiles.yaml" \
GEPPETTO_PROFILE=default \
GEPPETTO_TIMEOUT_MS=120000 \
./examples/js/geppetto/run_real_provider_multiturn.sh
```

## Step 5: Add Tools and Schemas

```js
const input = gp.schema.object()
  .property("value", gp.schema.string())
  .required("value")
  .build();

const echo = gp.tool("echo_value")
  .description("Echo a value")
  .input(input)
  .handler((args) => ({ echoed: args.value }))
  .build();

const registry = gp.toolRegistry().add(echo);
console.log(registry.call("echo_value", { value: "hi" }));
```

See:

- `examples/js/geppetto/29_tools_schema_multimodal_turn.js`

## Step 6: Add Images to a Turn

```js
const turn = gp.turn()
  .system("You are a visual reasoning assistant.")
  .user(m => m
    .text("What is in this screenshot?")
    .imageFile("./screenshot.png"))
  .build();
```

`imageURL`, `imageFile`, and `imageBytes` all produce image payloads on the Go-owned turn wrapper.

## Step 7: Use xgoja Provider Registry Config

Generated xgoja hosts can configure registry loading with:

```json
{
  "profileRegistries": ["/home/me/.config/pinocchio/profiles.yaml"],
  "defaultProfile": "default",
  "allowRegistryLoad": true
}
```

Registry loading is denied unless `allowRegistryLoad` is true.

## Removed Legacy APIs

The clean cutover removed the old map/session/runner public names from the default JavaScript surface:

- `gp.engines.fromConfig(...)`
- `gp.profiles.resolve(...)`
- `gp.turns.newTurn(...)`
- `gp.createSession(...)`
- `gp.runner.run(...)`

Use `gp.inferenceProfiles`, `gp.engine()`, `gp.agent()`, `gp.turn()`, `gp.schema`, `gp.tool`, and `gp.toolRegistry()` instead.
