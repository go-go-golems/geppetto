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
- `events(sink)`
- `runDefaults(options?)`
- `build()`

Agent methods:

- `run(turn, options?)`
- `stream(turn, options?)`

Intentionally absent:

- `agent.ask(prompt)`
- `agent.system(prompt)`
- `agent.profile(name)`
- `agent.inferenceProfile(name)`

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

Use the wrapper-first APIs in this reference instead.
