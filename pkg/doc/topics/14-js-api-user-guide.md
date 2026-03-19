---
Title: Geppetto JavaScript API User Guide
Slug: geppetto-js-api-user-guide
Short: Practical guide to composing engines and app-owned runtime behavior from JavaScript.
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

This guide shows the intended split after the engine-profile hard cut:

- use engine profiles to resolve `InferenceSettings`
- use the runner to assemble app-owned runtime behavior

## Run Scripts

```bash
go run ./cmd/examples/geppetto-js-lab --script <your-script.js>
```

## Default Path

For most scripts:

1. resolve or construct an engine
2. build app-owned runtime input
3. run with `gp.runner`

```javascript
const gp = require("geppetto");

const engine = gp.engines.fromConfig({
  apiType: "openai",
  model: "gpt-4.1-mini",
  apiKey: "test-openai-key",
});

const runtime = gp.runner.resolveRuntime({
  systemPrompt: "Answer in one short line.",
  runtimeKey: "demo",
});

const out = gp.runner.run({
  engine,
  runtime,
  prompt: "hello",
});
```

## Using Engine Profiles

Engine profiles now configure engine settings only.

Resolve one:

```javascript
const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
console.log(resolved.inferenceSettings.chat.engine);
```

Build an engine from it:

```javascript
const engine = gp.engines.fromResolvedProfile(resolved);
```

Or skip the explicit resolve step:

```javascript
const engine = gp.engines.fromProfile({ profileSlug: "assistant" });
```

## App-Owned Runtime

The runtime object belongs to the application side.

Use it for:

- `systemPrompt`
- explicit middleware refs
- tool-name filtering
- runtime metadata like `runtimeKey`

Example:

```javascript
const runtime = gp.runner.resolveRuntime({
  systemPrompt: "Be terse.",
  middlewares: [gp.middlewares.go("reorderToolResults")],
  toolNames: ["search_docs"],
  runtimeKey: "assistant-terse",
});
```

Do not expect `gp.runner.resolveRuntime(...)` to resolve engine profiles. That was removed intentionally.

## Combining Both Sides

This is the target shape:

```javascript
const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
const engine = gp.engines.fromResolvedProfile(resolved);

const runtime = gp.runner.resolveRuntime({
  systemPrompt: "App-owned prompt",
  runtimeKey: resolved.profileSlug,
  metadata: {
    profileSlug: resolved.profileSlug,
    profileRegistry: resolved.registrySlug,
  },
});

const handle = gp.runner.start({
  engine,
  runtime,
  prompt: "hello",
});
```

## Registry Stacks from JS

If the host did not provide a registry, connect one from JS:

```javascript
const gp = require("geppetto");

gp.profiles.connectStack([
  "examples/js/geppetto/profiles/10-provider-openai.yaml",
  "examples/js/geppetto/profiles/20-team-agent.yaml",
  "examples/js/geppetto/profiles/30-user-overrides.yaml",
]);

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
console.log(resolved.registrySlug, resolved.inferenceSettings.chat.engine);
```

## What Was Removed

The older mixed model is gone:

- no `effectiveRuntime` on resolved profiles
- no profile-derived `runtimeKey` or `runtimeFingerprint`
- no `resolvedProfile` builder option
- no `useResolvedProfile(...)`
- no `runner.resolveRuntime({ profile: ... })`

If you need those behaviors, implement them explicitly in the host script or application code. That is now the intended architecture.
