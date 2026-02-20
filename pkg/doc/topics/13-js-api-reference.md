---
Title: Geppetto JavaScript API Reference
Slug: geppetto-js-api-reference
Short: Exhaustive reference for the native `require("geppetto")` API exposed through goja.
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

This is the contract reference for `require("geppetto")` in `pkg/js/modules/geppetto`.

## Script-First Development Harness

All examples in this page are meant to be written as JS files and executed directly with:

```bash
go run ./cmd/examples/geppetto-js-lab --script <path-to-script.js>
```

The harness provides:

- `require("geppetto")`
- `assert(cond, message)` global helper
- `console.log` and `console.error`
- `ENV` map for environment variables
- built-in Go tools in the JS registry host:
  - `go_double`
  - `go_concat`

List available host Go tools:

```bash
go run ./cmd/examples/geppetto-js-lab --list-go-tools
```

## Host Wiring Requirement

For async APIs (`runAsync`, `start`), register the module with `Options.Runner`:

```go
loop := eventloop.NewEventLoop()
go loop.Start()
runner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
    Name:          "my-app",
    RecoverPanics: true,
})

gp.Register(reg, gp.Options{
    Runner: runner,
    // GoToolRegistry: ...
})
```

## Top-Level Exports

| Export | Type | Purpose |
|---|---|---|
| `version` | string | Module version marker |
| `createBuilder(opts?)` | function | Create a chainable session builder |
| `createSession(opts)` | function | Build a session directly |
| `runInference(engine, turn, opts?)` | function | One-shot inference helper |
| `consts` | namespace | Generated string constants (tool loop, block kinds, keys, event types) |
| `turns` | namespace | Turn and block helpers |
| `engines` | namespace | Engine constructors |
| `middlewares` | namespace | Middleware adapters |
| `tools` | namespace | Tool registry constructors |

## `consts` Namespace

Generated from `pkg/js/modules/geppetto/spec/js_api_codegen.yaml`.

| Group | Example |
|---|---|
| `ToolChoice` | `gp.consts.ToolChoice.AUTO` |
| `ToolErrorHandling` | `gp.consts.ToolErrorHandling.RETRY` |
| `BlockKind` | `gp.consts.BlockKind.TOOL_USE` |
| `TurnDataKeys` | `gp.consts.TurnDataKeys.TOOL_CONFIG` |
| `HookAction` | `gp.consts.HookAction.ABORT` |
| `MetadataKeys` | `gp.consts.MetadataKeys.SESSION_ID` |
| `TurnMetadataKeys` | `gp.consts.TurnMetadataKeys.SESSION_ID` |
| `BlockMetadataKeys` | `gp.consts.BlockMetadataKeys.CLAUDE_ORIGINAL_CONTENT` |
| `EventType` | `gp.consts.EventType.TOOL_RESULT` |

## `turns` Namespace

| Function | Signature | Notes |
|---|---|---|
| `normalize` | `normalize(turn)` | Canonical round-trip mapping |
| `newTurn` | `newTurn(opts?)` | Create turn object |
| `appendBlock` | `appendBlock(turn, block)` | Append block to turn |
| `newUserBlock` | `newUserBlock(text)` | User block |
| `newSystemBlock` | `newSystemBlock(text)` | System block |
| `newAssistantBlock` | `newAssistantBlock(text)` | Assistant text block |
| `newToolCallBlock` | `newToolCallBlock(id, name, args)` | Tool call block |
| `newToolUseBlock` | `newToolUseBlock(id, result, error?)` | Tool result block |

## `engines` Namespace

| Function | Signature | Notes |
|---|---|---|
| `echo` | `echo({reply?})` | Deterministic local engine |
| `fromFunction` | `fromFunction(fn)` | JS callback-backed engine |
| `fromProfile` | `fromProfile(profile?, opts?)` | Provider-backed engine from profile |
| `fromConfig` | `fromConfig(opts)` | Provider-backed engine from explicit config |

### `fromProfile` precedence

1. explicit `profile` argument
2. `opts.profile`
3. `PINOCCHIO_PROFILE`
4. default `4o-mini`

### `fromProfile` / `fromConfig` options

| Option | Type | Description |
|---|---|---|
| `apiType` / `provider` | string | provider (`openai`, `openai-responses`, `claude`, `gemini`, ...) |
| `model` | string | model override |
| `apiKey` | string | explicit API key |
| `baseURL` | string | provider base URL |
| `temperature` | number | sampling temperature |
| `topP` | number | top-p |
| `maxTokens` | number | max response tokens |
| `timeoutSeconds` / `timeoutMs` | number | timeout override |

## Sessions and Builder

### Session methods

| Method | Signature | Notes |
|---|---|---|
| `append` | `append(turn)` | Append turn |
| `latest` | `latest()` | Latest turn snapshot |
| `turnCount` | `turnCount()` | Number of turns |
| `turns` | `turns()` | Turn snapshot array |
| `getTurn` | `getTurn(index)` | Turn snapshot or `null` |
| `turnsRange` | `turnsRange(start, end)` | Sliced snapshot array |
| `run` | `run(seedTurn?, runOptions?)` | Sync inference with optional per-run options |
| `runAsync` | `runAsync(seedTurn?)` | Promise inference (host must configure `Options.Runner`) |
| `start` | `start(seedTurn?, runOptions?)` | RunHandle-based async inference with cancellation and event subscriptions |
| `isRunning` | `isRunning()` | Run state |
| `cancelActive` | `cancelActive()` | Cancel active run |

### Builder methods

| Method | Signature | Notes |
|---|---|---|
| `withEngine` | `withEngine(engine)` | Set base engine |
| `useMiddleware` | `useMiddleware(mw)` | Attach middleware object |
| `useGoMiddleware` | `useGoMiddleware(name, opts?)` | Attach built-in Go middleware |
| `withTools` | `withTools(registry, toolLoopOpts?)` | Attach tools + loop config |
| `withToolLoop` | `withToolLoop(opts)` | Configure loop directly |
| `withToolHooks` | `withToolHooks(hooks)` | Lifecycle hook registration |
| `buildSession` | `buildSession()` | Materialize session |

## `middlewares` Namespace

| Function | Signature | Notes |
|---|---|---|
| `fromJS` | `fromJS((turn, next, ctx?) => out, name?)` | JS middleware adapter with context payload |
| `go` | `go(name, opts?)` | Build Go middleware reference |

Built-in Go middleware names:

- `systemPrompt`
- `reorderToolResults`
- `turnLogging`

## `tools` Namespace

`tools.createRegistry()` returns a mutable registry object.

### Registry methods

| Method | Signature | Notes |
|---|---|---|
| `register` | `register(spec)` | Register JS tool |
| `useGoTools` | `useGoTools(names?)` | Import host Go tools |
| `list` | `list()` | List tool metadata |
| `call` | `call(name, args?)` | Direct invocation |

JS tool spec fields:

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Unique name |
| `description` | string | no | Human description |
| `handler` | function | yes | `({ ...args }, ctx?) => result` |
| `parameters` | object | no | JSON schema payload |

## Toolloop Options and Hooks

### `withTools(..., toolLoopOpts)` / `withToolLoop(toolLoopOpts)`

| Option | Type | Description |
|---|---|---|
| `enabled` | bool | Enable tool loop |
| `maxIterations` | number | Max loop iterations |
| `toolChoice` | string | `auto` / `none` / `required` (or `gp.consts.ToolChoice.*`) |
| `maxParallelTools` | number | Max parallel tool calls |
| `executionTimeoutMs` | number | Per-tool timeout |
| `allowedTools` | string[] | Tool allowlist |
| `toolErrorHandling` | string | `continue` / `abort` / `retry` (or `gp.consts.ToolErrorHandling.*`) |
| `retryMaxRetries` | number | Retry count |
| `retryBackoffMs` | number | Retry backoff base |
| `retryBackoffFactor` | number | Exponential factor |
| `hooks` | object | Lifecycle hooks object |

### `withToolHooks(hooks)` fields

| Hook field | Signature | Purpose |
|---|---|---|
| `beforeToolCall` | `(ctx) => mutationOrAction` | Mutate call args/name/id or abort |
| `afterToolCall` | `(ctx) => mutationOrAction` | Transform result/error |
| `onToolError` | `(ctx) => retryDecision` | Retry/abort/backoff decision |
| `failOpen` | bool | Ignore hook failures |
| `hookErrorPolicy` / `onHookError` | string | `fail-open` or `fail-closed` |
| `maxHookRetries` | number | Hard cap for hook retries |

### Callback Context Payloads

- Middleware callback third parameter (`ctx`):
  - `sessionId`, `inferenceId`, `traceId`, `turnId`
  - `middlewareName`, `timestampMs`
  - optional `deadlineMs`
  - optional `tags` from `run(..., { tags })` / `start(..., { tags })`
- Tool handler second parameter (`ctx`):
  - `toolName`, `timestampMs`
  - `sessionId`, `inferenceId`
  - `callId`, optional `callName`
  - optional `deadlineMs`
  - optional `tags`
- Hook payloads (`beforeToolCall`, `afterToolCall`, `onToolError`) include:
  - `sessionId`, `inferenceId`
  - optional `tags`

## Run Options and RunHandle

Per-run options accepted by `session.run()` and `session.start()`:

| Option | Type | Notes |
|---|---|---|
| `timeoutMs` | number | Creates a run-scoped context deadline |
| `tags` | object | Arbitrary run tags propagated to callback context payloads |

`session.start()` returns a `RunHandle`:

| Field/Method | Type | Notes |
|---|---|---|
| `promise` | `Promise<Turn>` | Resolves/rejects when run finishes |
| `cancel()` | function | Cancels the in-flight run |
| `on(eventType, cb)` | function | Subscribe to streamed events; `eventType` supports exact type or `"*"` |

## Metadata and Key Mapping

The codec canonicalizes turn and block fields across JS and Go:

- block kinds use generated mapper in `pkg/turns/block_kind_gen.go`
- key IDs use generated mapper in `pkg/turns/keys_gen.go`
- unknown keys are preserved

## Executable JS Examples

Use the maintained scripts under `examples/js/geppetto`:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/01_turns_and_blocks.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/02_session_echo.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/03_middleware_composition.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/04_tools_and_toolloop.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/05_go_tools_from_js.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/07_context_and_constants.js
```

Optional live inference script:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/06_live_profile_inference.js
```

`06_live_profile_inference.js` self-skips unless `GEMINI_API_KEY` or `GOOGLE_API_KEY` is set.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `createSession requires options object with engine` | Missing engine | pass `{ engine: gp.engines.*(...) }` |
| `no go tool registry configured` | `useGoTools` used in a host without Go tool registry | use `geppetto-js-lab` or register `Options.GoToolRegistry` |
| `builder has no engine configured` | builder missing `withEngine` | set engine before `buildSession()` |
| `runAsync requires module options Runner to be configured` | runtime runner not provided | use sync `run()` or register module with `Options.Runner` |
| `start requires module options Runner to be configured` | runtime runner not provided | use sync `run()` or register module with `Options.Runner` |
| `invalid toolChoice ...` / `invalid toolErrorHandling ...` | unsupported enum value | use allowed values or `gp.consts.*` constants |
| Tool calls not executed | registry not attached | call `.withTools(reg, { enabled: true })` |

## See Also

- [JS API User Guide](14-js-api-user-guide.md)
- [JS API Getting Started Tutorial](../tutorials/05-js-api-getting-started.md)
- [Turns and Blocks](08-turns.md)
- [Middlewares](09-middlewares.md)
- [Sessions](10-sessions.md)
