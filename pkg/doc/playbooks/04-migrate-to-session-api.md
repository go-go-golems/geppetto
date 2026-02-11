---
Title: "Migration playbook: move to Session/EngineBuilder/ExecutionHandle"
Slug: migrate-to-session-api
Short: Step-by-step guide to migrate from the legacy inference lifecycle APIs to geppetto/pkg/inference/session.
Topics:
  - inference
  - architecture
  - events
  - migration
Commands: []
Flags: []
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Playbook
---

# Migration playbook: move to Session/EngineBuilder/ExecutionHandle

## Goal

Migrate code that previously relied on:

- `geppetto/pkg/inference/state` (`InferenceState`, `StartRun/FinishRun`, cancel plumbing),
- `geppetto/pkg/inference/core` (`Session`, `RunInferenceStarted`, lifecycle helpers),
- “engine-config sinks” (previously done via `engine.WithSink` / `engine.Option`),

to the new, unified API:

- `geppetto/pkg/inference/session.Session` (multi-turn state),
- `EngineBuilder` + `InferenceRunner` (blocking inference entrypoint),
- `ExecutionHandle` (cancel + wait for an in-flight inference),
- **context-only sinks** (`events.WithEventSinks`).

No backwards compatibility is assumed.

## What changed (high-level)

### 1) Event sinks

Old:

- you could pass a sink into engine construction (e.g., `engine.WithSink(sink)`)
- provider engines sometimes “bridged” those sinks into context

New:

- provider engines and helpers publish via context only
- attach sinks via `events.WithEventSinks(ctx, sinks...)`
- the canonical place to wire sinks is `enginebuilder.Builder.EventSinks`

### 2) Lifecycle primitives

Old:

- `InferenceState` mixed long-lived state (turn) with in-flight state (running + cancel)
- `core.Session` added additional lifecycle entrypoints (`RunInferenceStarted`) to support “start-but-not-yet-running” patterns

New:

- `Session` owns turn history and enforces “one active inference per session”
- `Session.StartInference(ctx)` starts the run and returns an `ExecutionHandle`
- `ExecutionHandle.Wait()` is the single “blocking join” point

### 3) Engine creation API

New (only):

```go
eng, err := factory.NewEngineFromParsedLayers(parsedLayers)
runCtx := events.WithEventSinks(ctx, sink)
_, err = eng.RunInference(runCtx, seed)
```

## Quick mapping table

| Legacy concept | New concept |
|---|---|
| `engine.WithSink(sink)` | `events.WithEventSinks(ctx, sink)` (or `enginebuilder.Builder.EventSinks`) |
| `InferenceState` | `session.Session` |
| `StartRun/FinishRun` | `Session.IsRunning()` + `Session.StartInference()` |
| `SetCancel/HasCancel` | `ExecutionHandle.Cancel()` / `Session.CancelActive()` |
| `RunInferenceStarted` | not needed (start is immediate; wait is explicit) |

## Step 0: Find call sites

Search for:

- `engine.WithSink`
- `engine.Option`
- `InferenceState`
- `core.Session`
- `RunInferenceStarted`
- `StartRun` / `FinishRun`

## Step 1: Update engine factory calls (remove options)

Replace:

Previously you may have been passing “engine-config” options at construction time (e.g., to attach sinks). Those options are removed.

With:

```go
eng, err := factory.NewEngineFromParsedLayers(parsedLayers)
if err != nil { return err }
```

Then attach sinks at runtime:

```go
runCtx := events.WithEventSinks(ctx, sink)
_, err = eng.RunInference(runCtx, seed)
```

## Step 2: Introduce an enginebuilder Builder (recommended for chat-style apps)

Build a single runner that owns:

- sink wiring (context sinks),
- middleware wrapping,
- tool registry + tool config,
- snapshot hooks / persistence.

Typical shape:

```go
base, _ := factory.NewEngineFromParsedLayers(parsedLayers)

b := enginebuilder.New(
    enginebuilder.WithBase(base),
    enginebuilder.WithMiddlewares(/* system prompt, logging, etc */),
    enginebuilder.WithEventSinks(sink),
    enginebuilder.WithToolRegistry(registry), // optional: enables tool loop
    // enginebuilder.WithToolConfig(*toolCfg), // optional
)
```

## Step 3: Replace “InferenceState” with “Session”

Old (conceptually):

- “current turn” lives on `InferenceState`
- “running flag + cancel func” also live on `InferenceState`

New:

```go
sess := &session.Session{
    SessionID: sessionID,
    Builder:   b,
}

sess.Append(seedTurn) // append the turn you want to run
handle, err := sess.StartInference(ctx)
if err != nil { return err }
out, err := handle.Wait()
```

Notes:

- `Session.StartInference` runs asynchronously; `Wait()` blocks.
- On success, the latest appended turn is mutated in-place (no additional turn is appended).
- For follow-up user prompts, prefer `Session.AppendNewTurnFromUserPrompt(...)` instead of manually
  cloning `sess.Latest()`.

## Step 4: Move cancellation to ExecutionHandle

Replace “store cancel on state” with:

```go
_ = sess.CancelActive()
// or if you hold the handle:
handle.Cancel()
```

## Step 5: Migrate UIs (Bubble Tea and webchat)

### Bubble Tea backend pattern

The `chat.Backend` should:

- start inference synchronously (so “already running” is deterministic),
- return a `tea.Cmd` that blocks on `handle.Wait()` and emits `BackendFinishedMsg`.

Pseudo:

```go
func (b *EngineBackend) Start(ctx context.Context, prompt string) (tea.Cmd, error) {
    _, err := b.sess.AppendNewTurnFromUserPrompt(prompt)
    if err != nil { return nil, err }

    handle, err := b.sess.StartInference(ctx)
    if err != nil { return nil, err }

    return func() tea.Msg {
        _, _ = handle.Wait()
        return chat.BackendFinishedMsg{}
    }, nil
}
```

### Web handlers

Replace `StartRun()/FinishRun()` with:

- check `sess.IsRunning()`
- append the prompt turn (`sess.AppendNewTurnFromUserPrompt(prompt)`)
- call `sess.StartInference(ctx)` and return immediately

## Step 6: Delete (or stop using) legacy APIs

After all callers are migrated:

- remove references to `geppetto/pkg/inference/core`
- remove references to `geppetto/pkg/inference/state`
- ensure docs/snippets no longer mention removed “engine-config sinks” APIs

## Validation checklist

- `go test ./... -count=1` in `geppetto/` and downstream repos
- run real-world examples that stream events + tools:
  - `geppetto/cmd/examples/simple-streaming-inference`
  - `geppetto/cmd/examples/generic-tool-calling`
  - `geppetto/cmd/examples/openai-tools` (Responses thinking)
- if you have a TUI/webchat, do at least a 2-turn interaction and verify:
  - second prompt runs (no “stuck generating”)
  - cancellation returns to input state
