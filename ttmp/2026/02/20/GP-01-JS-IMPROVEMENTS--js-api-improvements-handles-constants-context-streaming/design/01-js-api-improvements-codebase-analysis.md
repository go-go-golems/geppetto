---
Title: "JS API Improvements: Codebase Analysis"
Ticket: GP-01-JS-IMPROVEMENTS
DocType: design
Topics:
  - geppetto
  - javascript
  - goja
  - api-design
Summary: >
  Detailed codebase analysis mapping each of the four JS API improvement areas
  (opaque handles, string-typed configs, handler context, streaming/run-handle)
  to the concrete files and line ranges that must change, with proposed
  modification strategies and code sketches.
---

# JS API Improvements: Codebase Analysis

## Table of Contents

1. [5.1 Opaque Handle Design Leaks Into Userland](#51-opaque-handle-design-leaks-into-userland)
2. [5.2 Stringly-Typed Configs Everywhere](#52-stringly-typed-configs-everywhere)
3. [5.3 Middleware and Tool Handlers Lack Context](#53-middleware-and-tool-handlers-lack-context)
4. [5.4 No Streaming / No Run-Handle](#54-no-streaming--no-run-handle)
5. [Cross-Cutting: TypeScript Definitions](#cross-cutting-typescript-definitions)
6. [Implementation Order and Dependencies](#implementation-order-and-dependencies)

---

## 5.1 Opaque Handle Design Leaks Into Userland

### Problem Statement

The geppetto JS module stores Go-side references on JavaScript objects using a
plain property named `__geppetto_ref`. Because this property is set via the
ordinary `Object.Set()` path, it is:

- **Enumerable** — visible in `Object.keys()`, `for...in`, `JSON.stringify()`
- **Writable** — can be accidentally overwritten by user code: `engine.__geppetto_ref = 42`
- **Discoverable** — any `console.log(engine)` reveals the internal property
- **Fragile** — passing the wrong object produces confusing errors like `"expected engine reference"` with no hint about what was actually received

### Relevant Code Locations

| File | Lines | What |
|------|-------|------|
| `pkg/js/modules/geppetto/module.go` | 19 | `hiddenRefKey = "__geppetto_ref"` constant |
| `pkg/js/modules/geppetto/module.go` | 124–126 | `attachRef()` — uses plain `o.Set(hiddenRefKey, ref)` |
| `pkg/js/modules/geppetto/module.go` | 128–139 | `getRef()` — reads via `obj.Get(hiddenRefKey)`, falls back to `v.Export()` |
| `pkg/js/modules/geppetto/api.go` | 187 | `attachRef(o, b)` on builder objects |
| `pkg/js/modules/geppetto/api.go` | 308 | `attachRef(o, sr)` on session objects |
| `pkg/js/modules/geppetto/api.go` | 1125 | `attachRef(o, ref)` on engine objects |
| `pkg/js/modules/geppetto/api.go` | 1208 | `attachRef(o, ref)` on JS middleware objects |
| `pkg/js/modules/geppetto/api.go` | 1225 | `attachRef(o, ref)` on Go middleware objects |
| `pkg/js/modules/geppetto/api.go` | 1308 | `attachRef(o, ref)` on tool registry objects |
| `pkg/js/modules/geppetto/api.go` | 911–921 | `requireEngineRef()` — vague error: `"expected engine reference"` |
| `pkg/js/modules/geppetto/api.go` | 923–933 | `requireToolRegistry()` — vague error: `"expected tool registry reference"` |

### Proposed Modification: DefineDataProperty

Replace the plain `o.Set()` call in `attachRef()` with goja's `DefineDataProperty`:

```go
// module.go — BEFORE
func (m *moduleRuntime) attachRef(o *goja.Object, ref any) {
    _ = o.Set(hiddenRefKey, ref)
}

// module.go — AFTER
func (m *moduleRuntime) attachRef(o *goja.Object, ref any) {
    _ = o.DefineDataProperty(hiddenRefKey, m.vm.ToValue(ref),
        goja.FLAG_FALSE, // writable = false
        goja.FLAG_FALSE, // enumerable = false
        goja.FLAG_FALSE, // configurable = false
    )
}
```

**Why this works:**
- `goja.DefineDataProperty` maps to `Object.defineProperty()` in ECMAScript
- Non-enumerable means `Object.keys()`, `for...in`, `JSON.stringify()` all skip it
- Non-writable means `obj.__geppetto_ref = 42` silently fails (or throws in strict mode)
- Non-configurable means it can't be deleted or redefined
- `obj.Get(hiddenRefKey)` still works for reading — `getRef()` needs zero changes

**Call sites unchanged:**
All 6 `attachRef()` call sites (api.go:187, 308, 1125, 1208, 1225, 1308) pass through the same function and require no modification.

### Proposed Modification: Better Error Messages

```go
// api.go — BEFORE
func (m *moduleRuntime) requireEngineRef(v goja.Value) (*engineRef, error) {
    ref := m.getRef(v)
    switch x := ref.(type) {
    case *engineRef:
        return x, nil
    case engine.Engine:
        return &engineRef{Name: "engine", Engine: x}, nil
    default:
        return nil, fmt.Errorf("expected engine reference")
    }
}

// api.go — AFTER
func (m *moduleRuntime) requireEngineRef(v goja.Value) (*engineRef, error) {
    ref := m.getRef(v)
    switch x := ref.(type) {
    case *engineRef:
        return x, nil
    case engine.Engine:
        return &engineRef{Name: "engine", Engine: x}, nil
    default:
        return nil, fmt.Errorf("expected engine reference, got %T (value: %v)", ref, v)
    }
}
```

Apply the same pattern to `requireToolRegistry()`.

### Impact & Backwards Compatibility

- **Fully backwards compatible** — reading the hidden property still works identically
- **Improved safety** — accidental overwrites no longer corrupt internal state
- **Cleaner serialization** — `JSON.stringify(engine)` no longer leaks internal pointers
- No changes needed to any example scripts or test code

---

## 5.2 Stringly-Typed Configs Everywhere

### Problem Statement

Tool loop settings (`toolChoice`, `toolErrorHandling`), block kinds, and engine
options are all accepted as raw strings in JS. Users get runtime failures for
typos (`"auot"` instead of `"auto"`) with no IDE guidance, no autocomplete,
and no early validation.

The Go side already defines proper typed constants, but these are never exposed
to JavaScript.

### Relevant Code Locations

#### Go Enum Definitions (source of truth)

| File | Lines | What |
|------|-------|------|
| `pkg/inference/tools/config.go` | 82–89 | `ToolChoice`: `"auto"`, `"none"`, `"required"` |
| `pkg/inference/tools/config.go` | 91–98 | `ToolErrorHandling`: `"continue"`, `"abort"`, `"retry"` |
| `pkg/turns/block_kind_gen.go` | 10–43 | `BlockKind`: `"user"`, `"llm_text"`, `"tool_call"`, `"tool_use"`, `"system"`, `"reasoning"`, `"other"` |
| `pkg/turns/keys_gen.go` | 9–33 | Metadata key constants: `session_id`, `inference_id`, `trace_id`, etc. |

#### JS API Consuming Raw Strings

| File | Lines | What |
|------|-------|------|
| `pkg/js/modules/geppetto/api.go` | 589 | `tools.ToolChoice(toString(cfg["toolChoice"], ...))` — raw string cast, no validation |
| `pkg/js/modules/geppetto/api.go` | 590 | `tools.ToolErrorHandling(toString(cfg["toolErrorHandling"], ...))` — same |
| `pkg/js/modules/geppetto/module.go` | 84–116 | `installExports()` — where constants would be added |

### Proposed Modification: Code-Generated Constants

Rather than hand-writing the constants export code, we extend the existing
codegen infrastructure (`cmd/gen-turns`) to generate both the Go installer code
and the TypeScript `.d.ts` enum types from a single YAML schema.

#### Existing Codegen Infrastructure

The codebase already has a YAML-driven code generator:

| File | What |
|------|------|
| `cmd/gen-turns/main.go` | Generator: reads YAML, renders Go via `text/template` |
| `pkg/turns/spec/turns_codegen.yaml` | Schema: defines `block_kinds` and `keys` |
| `pkg/turns/generate.go` | `//go:generate` directives |
| `pkg/turns/block_kind_gen.go` | Generated: BlockKind enum, String(), YAML marshal |
| `pkg/turns/keys_gen.go` | Generated: key constants, typed key variables |

#### Step 1: New YAML Schema for JS API Enums

Create `pkg/js/modules/geppetto/spec/js_api_codegen.yaml`:

```yaml
# JS API constants exported to JavaScript via gp.consts.*
# Also used to generate TypeScript .d.ts enum types.
#
# Each enum group becomes:
#   Go:  a function that installs goja objects on the exports.consts namespace
#   TS:  a const object type in the geppetto.d.ts module declaration

enums:
  - name: ToolChoice
    doc: "How the model should choose tools"
    # source: pkg/inference/tools/config.go:82-89
    values:
      - js_key: AUTO
        value: auto
        go_const: tools.ToolChoiceAuto
      - js_key: NONE
        value: none
        go_const: tools.ToolChoiceNone
      - js_key: REQUIRED
        value: required
        go_const: tools.ToolChoiceRequired

  - name: ToolErrorHandling
    doc: "How to handle tool execution errors"
    # source: pkg/inference/tools/config.go:91-98
    values:
      - js_key: CONTINUE
        value: continue
        go_const: tools.ToolErrorContinue
      - js_key: ABORT
        value: abort
        go_const: tools.ToolErrorAbort
      - js_key: RETRY
        value: retry
        go_const: tools.ToolErrorRetry

  - name: BlockKind
    doc: "The kind of a block within a Turn"
    # source: pkg/turns/block_kind_gen.go (generated from turns_codegen.yaml)
    values:
      - js_key: USER
        value: user
      - js_key: LLM_TEXT
        value: llm_text
      - js_key: TOOL_CALL
        value: tool_call
      - js_key: TOOL_USE
        value: tool_use
      - js_key: SYSTEM
        value: system
      - js_key: REASONING
        value: reasoning
      - js_key: OTHER
        value: other

  - name: HookAction
    doc: "Actions returned from tool hook callbacks"
    values:
      - js_key: ABORT
        value: abort
      - js_key: RETRY
        value: retry
      - js_key: CONTINUE
        value: continue

  - name: MetadataKeys
    doc: "Standard turn/block metadata key names"
    # source: pkg/turns/keys_gen.go (generated from turns_codegen.yaml)
    values:
      - js_key: SESSION_ID
        value: session_id
      - js_key: INFERENCE_ID
        value: inference_id
      - js_key: TRACE_ID
        value: trace_id
      - js_key: PROVIDER
        value: provider
      - js_key: MODEL
        value: model
      - js_key: STOP_REASON
        value: stop_reason
      - js_key: USAGE
        value: usage

  - name: EventType
    doc: "Streaming event types for RunHandle.on()"
    values:
      - js_key: START
        value: start
      - js_key: PARTIAL
        value: partial
      - js_key: FINAL
        value: final
      - js_key: TOOL_CALL
        value: tool-call
      - js_key: TOOL_RESULT
        value: tool-result
      - js_key: ERROR
        value: error
```

#### Step 2: New Generator (or extend `cmd/gen-turns`)

Create `cmd/gen-js-api/main.go` — a new generator that reads the YAML schema
above and produces two outputs:

**Output 1: `pkg/js/modules/geppetto/consts_gen.go`**

Generated Go code that installs the constants on the goja exports object:

```go
// Code generated by cmd/gen-js-api. DO NOT EDIT.

package geppetto

import "github.com/dop251/goja"

// installConsts installs the gp.consts namespace on the module exports.
func (m *moduleRuntime) installConsts(exports *goja.Object) {
    constsObj := m.vm.NewObject()

    // ToolChoice — How the model should choose tools
    {
        o := m.vm.NewObject()
        m.mustSet(o, "AUTO", "auto")
        m.mustSet(o, "NONE", "none")
        m.mustSet(o, "REQUIRED", "required")
        m.mustSet(constsObj, "ToolChoice", o)
    }

    // ToolErrorHandling — How to handle tool execution errors
    {
        o := m.vm.NewObject()
        m.mustSet(o, "CONTINUE", "continue")
        m.mustSet(o, "ABORT", "abort")
        m.mustSet(o, "RETRY", "retry")
        m.mustSet(constsObj, "ToolErrorHandling", o)
    }

    // BlockKind — The kind of a block within a Turn
    {
        o := m.vm.NewObject()
        m.mustSet(o, "USER", "user")
        m.mustSet(o, "LLM_TEXT", "llm_text")
        // ... etc
        m.mustSet(constsObj, "BlockKind", o)
    }

    // ... remaining enums ...

    m.mustSet(exports, "consts", constsObj)
}
```

Then `installExports()` in `module.go` just calls `m.installConsts(exports)`.

**Output 2: `pkg/doc/types/geppetto-consts.d.ts`** (or inline into full .d.ts)

Generated TypeScript const type declarations:

```typescript
// Code generated by cmd/gen-js-api. DO NOT EDIT.

/** How the model should choose tools */
export const ToolChoice: {
    readonly AUTO: "auto";
    readonly NONE: "none";
    readonly REQUIRED: "required";
};

/** How to handle tool execution errors */
export const ToolErrorHandling: {
    readonly CONTINUE: "continue";
    readonly ABORT: "abort";
    readonly RETRY: "retry";
};

/** The kind of a block within a Turn */
export const BlockKind: {
    readonly USER: "user";
    readonly LLM_TEXT: "llm_text";
    readonly TOOL_CALL: "tool_call";
    readonly TOOL_USE: "tool_use";
    readonly SYSTEM: "system";
    readonly REASONING: "reasoning";
    readonly OTHER: "other";
};

// ... etc
```

#### Step 3: Go Template for the Generator

The generator uses the same `text/template` approach as `cmd/gen-turns`:

```go
const goConstsTemplate = `// Code generated by cmd/gen-js-api. DO NOT EDIT.

package geppetto

import "github.com/dop251/goja"

// installConsts installs the gp.consts namespace on the module exports.
func (m *moduleRuntime) installConsts(exports *goja.Object) {
    constsObj := m.vm.NewObject()
{{- range .Enums }}

    // {{ .Name }} — {{ .Doc }}
    {
        o := m.vm.NewObject()
{{- range .Values }}
        m.mustSet(o, "{{ .JsKey }}", "{{ .Value }}")
{{- end }}
        m.mustSet(constsObj, "{{ .Name }}", o)
    }
{{- end }}

    m.mustSet(exports, "consts", constsObj)
}
`

const tsConstsTemplate = `// Code generated by cmd/gen-js-api. DO NOT EDIT.
{{- range .Enums }}

/** {{ .Doc }} */
export const {{ .Name }}: {
{{- range .Values }}
    readonly {{ .JsKey }}: "{{ .Value }}";
{{- end }}
};
{{- end }}
`
```

#### Step 4: go:generate Wiring

Create `pkg/js/modules/geppetto/generate.go`:

```go
package geppetto

//go:generate go run ../../../../cmd/gen-js-api \
//    --schema spec/js_api_codegen.yaml \
//    --go-out . \
//    --ts-out ../../../../pkg/doc/types
```

#### Design Note: BlockKind/MetadataKeys Duplication

The `BlockKind` and `MetadataKeys` values already live in
`turns_codegen.yaml`. Rather than duplicating them:

**Option A (simple):** Accept the duplication. The JS API schema is the
authority for what's *exported to JS*, which is a subset. The values rarely
change. Add a comment cross-referencing the source.

**Option B (DRY):** Have `cmd/gen-js-api` accept `--turns-schema` flag and
read block kinds / metadata keys directly from `turns_codegen.yaml`, only
requiring JS-specific enums (ToolChoice, HookAction, EventType) in the JS
schema. More complex but avoids drift.

**Recommendation:** Start with Option A. The values change very rarely, and
the JS schema is intentionally a curated subset (not everything in
turns_codegen.yaml needs to be exported to JS). Add a CI check that compares
the block_kinds values between both schemas if drift is a concern.

**JS usage after change (unchanged from before — just generated instead of hand-written):**

```js
const gp = require("geppetto");

// Before (error-prone):
const session = gp.createSession({
    engine: engine,
    toolLoop: { toolChoice: "auto", toolErrorHandling: "continue" }
});

// After (guided, generated constants):
const session = gp.createSession({
    engine: engine,
    toolLoop: {
        toolChoice: gp.consts.ToolChoice.AUTO,
        toolErrorHandling: gp.consts.ToolErrorHandling.CONTINUE
    }
});
```

### Optional Enhancement: Validate String Values

In `applyToolLoopSettings()`, add validation for known enum values. This
validation code can also be generated from the same schema:

```go
// api.go:589 — add validation (could be generated too)
choice := tools.ToolChoice(toString(cfg["toolChoice"], string(toolCfg.ToolChoice)))
switch choice {
case tools.ToolChoiceAuto, tools.ToolChoiceNone, tools.ToolChoiceRequired:
    toolCfg.ToolChoice = choice
default:
    panic(m.vm.NewTypeError("invalid toolChoice %q, expected one of: auto, none, required", choice))
}
```

### Impact & Backwards Compatibility

- **Fully backwards compatible** — raw strings still work; constants are an additive API
- **No breaking changes** — `"auto"` still accepted, `gp.consts.ToolChoice.AUTO` simply evaluates to `"auto"`
- New `consts` namespace on the exports object
- New files: `cmd/gen-js-api/`, `spec/js_api_codegen.yaml`, `consts_gen.go`, `geppetto-consts.d.ts`

---

## 5.3 Middleware and Tool Handlers Lack Context

### Problem Statement

JavaScript middleware receives only `(turn, next)`. Tool handlers receive only
`(args)`. Neither gets access to:

- Session ID / inference ID / turn ID for correlation
- Timing information
- Structured logger
- Cancellation signals

This forces benchmark and instrumentation scripts to rely on global mutable
state to correlate tool calls with inputs, which is error-prone and racey.

The Go side **already has** all this context — it flows through
`context.Context` and `Turn.Metadata` — but the JS bridge layer discards it.

### Relevant Code Locations

#### Middleware (context dropped at the JS boundary)

| File | Lines | What |
|------|-------|------|
| `pkg/inference/middleware/middleware.go` | 8–14 | Go signature: `func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)` — `ctx` IS available |
| `pkg/js/modules/geppetto/api.go` | 1257–1298 | `jsMiddleware()` implementation |
| `pkg/js/modules/geppetto/api.go` | 1285 | The JS call: `fn(goja.Undefined(), jsTurn, m.vm.ToValue(nextFn))` — only turn + next, **ctx dropped** |

#### Tool Handlers (context dropped)

| File | Lines | What |
|------|-------|------|
| `pkg/js/modules/geppetto/api.go` | 1396–1401 | Tool handler wrapper: `func(_ context.Context, in map[string]any)` — **ctx discarded** (underscore) |
| `pkg/js/modules/geppetto/api.go` | 1397 | The JS call: `handler(goja.Undefined(), r.api.vm.ToValue(in))` — only args passed |

#### Tool Hooks (partial context)

| File | Lines | What |
|------|-------|------|
| `pkg/js/modules/geppetto/api.go` | 750–762 | `PreExecute` payload: `{phase, call, timestampMs}` — no session/turn info |
| `pkg/js/modules/geppetto/api.go` | 797–810 | `PublishResult` payload: `{phase, call, result, timestampMs}` — no session/turn info |
| `pkg/js/modules/geppetto/api.go` | 854–868 | `ShouldRetry` payload: `{phase, attempt, call, error, ...}` — no session/turn info |
| `pkg/js/modules/geppetto/api.go` | 847 | `tools.CurrentToolCallFromContext(ctx)` — **context IS available** but info not forwarded to JS |

#### Available Context Data (in Go, not forwarded to JS)

| Source | Key | Type |
|--------|-----|------|
| `Turn.Metadata` | `session_id` | string |
| `Turn.Metadata` | `inference_id` | string |
| `Turn.Metadata` | `trace_id` | string |
| `Turn.ID` | (field) | string |
| `context.Context` | `tools.CurrentToolCallFromContext()` | `ToolCall` |
| `context.Context` | cancellation / deadline | signals |

### Proposed Modification A: Add `ctx` to Middleware

Modify `jsMiddleware()` in `api.go:1257-1298` to build and pass a context object as the third argument:

```go
func (m *moduleRuntime) jsMiddleware(name string, fn goja.Callable) middleware.Middleware {
    return func(next middleware.HandlerFunc) middleware.HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            jsTurn, err := m.encodeTurnValue(t)
            if err != nil {
                return nil, err
            }

            // === NEW: Build context object ===
            ctxObj := m.vm.NewObject()
            if sid, ok, _ := turns.KeyTurnMetaSessionID.Get(t.Metadata); ok {
                _ = ctxObj.Set("sessionId", sid)
            }
            if iid, ok, _ := turns.KeyTurnMetaInferenceID.Get(t.Metadata); ok {
                _ = ctxObj.Set("inferenceId", iid)
            }
            if tid, ok, _ := turns.KeyTurnMetaTraceID.Get(t.Metadata); ok {
                _ = ctxObj.Set("traceId", tid)
            }
            _ = ctxObj.Set("turnId", t.ID)
            _ = ctxObj.Set("middlewareName", name)
            _ = ctxObj.Set("timestampMs", time.Now().UnixMilli())
            if deadline, ok := ctx.Deadline(); ok {
                _ = ctxObj.Set("deadlineMs", deadline.UnixMilli())
            }

            nextFn := func(call goja.FunctionCall) goja.Value {
                inTurn := t
                if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
                    decoded, err := m.decodeTurnValue(call.Arguments[0])
                    if err != nil {
                        panic(m.vm.NewGoError(err))
                    }
                    inTurn = decoded
                }
                out, err := next(ctx, inTurn)
                if err != nil {
                    panic(m.vm.NewGoError(err))
                }
                v, err := m.encodeTurnValue(out)
                if err != nil {
                    panic(m.vm.NewGoError(err))
                }
                return v
            }

            // CHANGED: pass ctx as third argument: (turn, next, ctx)
            ret, err := fn(goja.Undefined(), jsTurn, m.vm.ToValue(nextFn), ctxObj)
            if err != nil {
                return nil, fmt.Errorf("%s: %w", name, err)
            }
            if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
                return t, nil
            }
            decoded, err := m.decodeTurnValue(ret)
            if err != nil {
                return nil, err
            }
            return decoded, nil
        }
    }
}
```

**JS usage after change:**

```js
const mw = gp.middlewares.fromJS((turn, next, ctx) => {
    console.log("session:", ctx.sessionId);
    console.log("inference:", ctx.inferenceId);
    console.log("turn:", ctx.turnId);
    console.log("at:", ctx.timestampMs);
    return next(turn);
}, "logging-mw");
```

**Backwards compatible:** existing `(turn, next)` middleware simply ignores the third argument.

### Proposed Modification B: Add `ctx` to Tool Handlers

Modify the tool handler wrapper in `api.go:1396-1401`:

```go
// BEFORE
fn := func(_ context.Context, in map[string]any) (any, error) {
    ret, err := handler(goja.Undefined(), r.api.vm.ToValue(in))
    // ...
}

// AFTER
fn := func(goCtx context.Context, in map[string]any) (any, error) {
    // Build context object
    ctxObj := r.api.vm.NewObject()
    _ = ctxObj.Set("timestampMs", time.Now().UnixMilli())
    _ = ctxObj.Set("toolName", name) // name is already in scope from register()

    if call, ok := tools.CurrentToolCallFromContext(goCtx); ok {
        _ = ctxObj.Set("callId", call.ID)
    }
    if deadline, ok := goCtx.Deadline(); ok {
        _ = ctxObj.Set("deadlineMs", deadline.UnixMilli())
    }

    // Pass (args, ctx) to JS handler
    ret, err := handler(goja.Undefined(), r.api.vm.ToValue(in), ctxObj)
    if err != nil {
        return nil, fmt.Errorf("js tool %s: %w", name, err)
    }
    return cloneJSONValue(ret.Export()), nil
}
```

**JS usage after change:**

```js
registry.register({
    name: "get_weather",
    description: "Get weather for a city",
    handler: (args, ctx) => {
        console.log("tool call ID:", ctx.callId);
        console.log("called at:", ctx.timestampMs);
        return { temp: 72, city: args.city };
    }
});
```

**Backwards compatible:** existing `(args)` handlers simply ignore the second argument.

### Proposed Modification C: Enrich Tool Hook Payloads

Add correlation IDs to the payload maps in `PreExecute`, `PublishResult`, and `ShouldRetry`.

The context is already available in all three methods. We need to extract turn metadata from it.

**For `PreExecute` (api.go:750-762):**

```go
// Add after existing payload construction:
payload := map[string]any{
    "phase": "beforeToolCall",
    "call": map[string]any{ /* existing */ },
    "timestampMs": time.Now().UnixMilli(),
    // === NEW ===
    "sessionId":   extractSessionIDFromContext(ctx),
    "inferenceId": extractInferenceIDFromContext(ctx),
}
```

However, these IDs are stored on the Turn, not directly on the context. Two options:

**Option 1:** Store a reference to the current Turn on the `jsToolHookExecutor` when the tool loop starts. This requires wiring through the enginebuilder/Loop.

**Option 2:** Use context values. The tool loop already puts the registry in context via `tools.WithRegistry(ctx, l.registry)` (loop.go:112). We could similarly put session/inference IDs into context. This requires:
- Adding context key+setter in `session/session.go` for sessionID and inferenceID
- Using them in `StartInference()` before spawning the goroutine
- Reading them in the hook executor

**Option 2 is cleaner** — it follows the existing pattern and doesn't require struct mutations:

```go
// session/context.go (new file)
type ctxKey int
const (
    ctxKeySessionID   ctxKey = iota
    ctxKeyInferenceID
)

func WithSessionMeta(ctx context.Context, sessionID, inferenceID string) context.Context {
    ctx = context.WithValue(ctx, ctxKeySessionID, sessionID)
    ctx = context.WithValue(ctx, ctxKeyInferenceID, inferenceID)
    return ctx
}

func SessionIDFromContext(ctx context.Context) string {
    if v, ok := ctx.Value(ctxKeySessionID).(string); ok { return v }
    return ""
}

func InferenceIDFromContext(ctx context.Context) string {
    if v, ok := ctx.Value(ctxKeyInferenceID).(string); ok { return v }
    return ""
}
```

Then in `session.go:228`:
```go
runCtx, cancel := context.WithCancel(ctx)
runCtx = WithSessionMeta(runCtx, s.SessionID, inferenceID)  // NEW
```

And in the hook executor methods:
```go
payload["sessionId"] = session.SessionIDFromContext(ctx)
payload["inferenceId"] = session.InferenceIDFromContext(ctx)
```

### Impact & Backwards Compatibility

- **Fully backwards compatible** for all three changes (A, B, C)
- Extra arguments to JS functions are silently ignored by goja
- Extra fields in hook payloads don't affect existing hook implementations
- New file: `session/context.go` for context key helpers
- Modified files: `api.go` (middleware wrapper, tool handler wrapper, hook payloads), `session.go` (inject metadata into context)

---

## 5.4 No Streaming / No Run-Handle

### Problem Statement

`session.runAsync()` returns a bare `Promise<Turn>` that resolves only after the
entire inference completes. There is:

- **No event stream** — tokens, tool call events, and iteration progress are invisible to JS
- **No per-run cancellation** — cancellation is via `session.cancelActive()`, not tied to the specific run
- **No per-run options** — no way to set a deadline, tags, or trace ID for a specific run
- **No RunHandle** — the Promise doesn't provide `.cancel()`, `.events()`, or correlation IDs

Yet the Go infrastructure **already supports all of this**:

- `ExecutionHandle` (execution.go) has `Cancel()`, `Wait()`, `IsRunning()`, `SessionID`, `InferenceID`
- `events.EventSink` (sink.go) provides a clean interface for receiving events
- `events.WithEventSinks(ctx, ...)` (context.go) attaches sinks to context
- Rich event types exist: `EventPartialCompletion`, `EventToolCall`, `EventToolResult`, etc. (chat-events.go)
- `context.WithTimeout`, `context.WithCancel` for deadline/cancellation

### Relevant Code Locations

#### Current runAsync Implementation

| File | Lines | What |
|------|-------|------|
| `pkg/js/modules/geppetto/api.go` | 416–426 | `runAsync` method registration on session object |
| `pkg/js/modules/geppetto/api.go` | 441–464 | `runAsync()` implementation: Promise + goroutine |
| `pkg/js/modules/geppetto/api.go` | 430–438 | `runSync()`: `session.StartInference(context.Background())` — hardcoded background context |
| `pkg/js/modules/geppetto/api.go` | 397–414 | `run()` method: synchronous, same context issue |

#### Go Infrastructure (already supports what we need)

| File | Lines | What |
|------|-------|------|
| `pkg/inference/session/session.go` | 185–272 | `StartInference()` — creates `ExecutionHandle`, spawns goroutine |
| `pkg/inference/session/session.go` | 228 | `context.WithCancel(ctx)` — cancellation already wired |
| `pkg/inference/session/execution.go` | 16–85 | `ExecutionHandle` — `Cancel()`, `Wait()`, `IsRunning()`, `SessionID`, `InferenceID` |
| `pkg/events/sink.go` | 1–11 | `EventSink` interface: `PublishEvent(event Event) error` |
| `pkg/events/context.go` | 19–27 | `WithEventSinks()` — attaches sinks to context |
| `pkg/events/context.go` | 41–52 | `PublishEventToContext()` — publishes to all context sinks |
| `pkg/events/chat-events.go` | 11–88 | Event types: `start`, `partial`, `tool-call`, `tool-result`, `final`, `error`, etc. |
| `pkg/inference/toolloop/loop.go` | 81–89 | `snapshot()` — emits `pre_inference`, `post_inference`, `post_tools` phases |

### Proposed Modification: RunHandle Object

#### New Go Type: `jsEventCollector`

```go
// api.go — new type

// jsEventCollector implements events.EventSink and buffers events for
// dispatch to JS callbacks on the event loop.
type jsEventCollector struct {
    api       *moduleRuntime
    mu        sync.Mutex
    listeners map[string][]goja.Callable // eventType -> callbacks
}

var _ events.EventSink = (*jsEventCollector)(nil)

func newJSEventCollector(api *moduleRuntime) *jsEventCollector {
    return &jsEventCollector{
        api:       api,
        listeners: make(map[string][]goja.Callable),
    }
}

func (c *jsEventCollector) subscribe(eventType string, fn goja.Callable) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.listeners[eventType] = append(c.listeners[eventType], fn)
}

func (c *jsEventCollector) PublishEvent(ev events.Event) error {
    c.mu.Lock()
    // Collect matching listeners: exact type match + "*" wildcard
    var cbs []goja.Callable
    if fns, ok := c.listeners[string(ev.Type())]; ok {
        cbs = append(cbs, fns...)
    }
    if fns, ok := c.listeners["*"]; ok {
        cbs = append(cbs, fns...)
    }
    c.mu.Unlock()

    if len(cbs) == 0 {
        return nil
    }

    // Dispatch callbacks on the event loop (thread-safe for goja)
    c.api.loop.RunOnLoop(func(vm *goja.Runtime) {
        payload := c.encodeEvent(ev)
        for _, cb := range cbs {
            _, _ = cb(goja.Undefined(), payload)
        }
    })
    return nil
}

func (c *jsEventCollector) encodeEvent(ev events.Event) goja.Value {
    meta := ev.Metadata()
    obj := c.api.vm.NewObject()
    _ = obj.Set("type", string(ev.Type()))
    _ = obj.Set("sessionId", meta.SessionID)
    _ = obj.Set("inferenceId", meta.InferenceID)
    _ = obj.Set("turnId", meta.TurnID)
    _ = obj.Set("timestampMs", time.Now().UnixMilli())

    // Type-specific fields
    switch e := ev.(type) {
    case *events.EventPartialCompletion:
        _ = obj.Set("delta", e.Delta)
        _ = obj.Set("completion", e.Completion)
    case *events.EventToolCall:
        _ = obj.Set("toolCall", map[string]any{
            "id": e.ToolCall.ID, "name": e.ToolCall.Name, "input": e.ToolCall.Input,
        })
    case *events.EventToolResult:
        _ = obj.Set("toolResult", map[string]any{
            "id": e.ToolResult.ID, "result": e.ToolResult.Result,
        })
    case *events.EventFinal:
        _ = obj.Set("text", e.Text)
    case *events.EventError:
        _ = obj.Set("error", e.ErrorString)
    }

    // Also include raw payload if available
    if p := ev.Payload(); len(p) > 0 {
        _ = obj.Set("rawPayload", string(p))
    }

    return obj
}
```

#### Modified `runAsync()` — Returns RunHandle

```go
func (sr *sessionRef) runAsync(seed *turns.Turn) goja.Value {
    if sr.api.loop == nil {
        panic(sr.api.vm.NewTypeError("runAsync requires module options Loop to be configured"))
    }

    // Parse optional run options from second argument (handled by caller)
    // For now, basic implementation

    promise, resolve, reject := sr.api.vm.NewPromise()

    // Create event collector
    collector := newJSEventCollector(sr.api)

    // Build the RunHandle JS object
    handleObj := sr.api.vm.NewObject()
    sr.api.mustSet(handleObj, "promise", promise)

    // Track cancel function (set inside goroutine)
    var cancelMu sync.Mutex
    var cancelFn context.CancelFunc

    sr.api.mustSet(handleObj, "cancel", func(goja.FunctionCall) goja.Value {
        cancelMu.Lock()
        fn := cancelFn
        cancelMu.Unlock()
        if fn != nil {
            fn()
        }
        return goja.Undefined()
    })

    sr.api.mustSet(handleObj, "on", func(call goja.FunctionCall) goja.Value {
        if len(call.Arguments) < 2 {
            panic(sr.api.vm.NewTypeError("on(eventType, callback) requires 2 arguments"))
        }
        eventType := call.Arguments[0].String()
        fn, ok := goja.AssertFunction(call.Arguments[1])
        if !ok {
            panic(sr.api.vm.NewTypeError("on() second argument must be a function"))
        }
        collector.subscribe(eventType, fn)
        return handleObj // chainable
    })

    go func() {
        ctx := context.Background()
        ctx, cancel := context.WithCancel(ctx)
        defer cancel()

        cancelMu.Lock()
        cancelFn = cancel
        cancelMu.Unlock()

        // Attach event sink to context
        ctx = events.WithEventSinks(ctx, collector)

        if seed != nil {
            sr.session.Append(seed)
        }
        handle, err := sr.session.StartInference(ctx)
        if err != nil {
            sr.api.loop.RunOnLoop(func(*goja.Runtime) {
                _ = reject(sr.api.vm.ToValue(err.Error()))
            })
            return
        }
        out, err := handle.Wait()

        sr.api.loop.RunOnLoop(func(*goja.Runtime) {
            if err != nil {
                _ = reject(sr.api.vm.ToValue(err.Error()))
                return
            }
            v, encErr := sr.api.encodeTurnValue(out)
            if encErr != nil {
                _ = reject(sr.api.vm.ToValue(encErr.Error()))
                return
            }
            _ = resolve(v)
        })
    }()

    return handleObj
}
```

#### Per-Run Options

Modify the `runAsync` method registration (api.go:416-426) to accept an options object:

```go
m.mustSet(o, "runAsync", func(call goja.FunctionCall) goja.Value {
    var t *turns.Turn
    var err error
    if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
        t, err = m.decodeTurnValue(call.Arguments[0])
        if err != nil {
            panic(m.vm.NewGoError(err))
        }
    }
    // NEW: parse run options from second argument
    var runOpts map[string]any
    if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
        runOpts = decodeMap(call.Arguments[1].Export())
    }
    return sr.runAsyncWithOpts(t, runOpts)
})
```

Then `runAsyncWithOpts` wraps the existing logic with timeout/tags:

```go
func (sr *sessionRef) runAsyncWithOpts(seed *turns.Turn, opts map[string]any) goja.Value {
    // ...same as modified runAsync above, but also:
    ctx := context.Background()
    if opts != nil {
        if timeoutMs := toInt(opts["timeoutMs"], 0); timeoutMs > 0 {
            var tCancel context.CancelFunc
            ctx, tCancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
            defer tCancel()
        }
        if tags := decodeMap(opts["tags"]); tags != nil {
            // Store tags in turn metadata for downstream access
            // (could also use context values)
        }
    }
    // ...rest as above
}
```

**JS usage after change:**

```js
const handle = session.runAsync(seedTurn, { timeoutMs: 30000 });

// Subscribe to streaming events
handle.on("partial", (ev) => {
    process.stdout.write(ev.delta);
});

handle.on("tool-call", (ev) => {
    console.log("Tool called:", ev.toolCall.name);
});

handle.on("final", (ev) => {
    console.log("Complete:", ev.text);
});

// Cancel if needed
setTimeout(() => handle.cancel(), 5000);

// Await completion
const result = await handle.promise;
```

### Important Design Considerations

1. **Thread safety:** The `jsEventCollector.PublishEvent()` is called from the inference goroutine. All JS callback dispatch MUST go through `loop.RunOnLoop()` to avoid concurrent goja runtime access.

2. **Event ordering:** Events dispatched via `RunOnLoop()` are queued and executed in order on the event loop, preserving causal ordering.

3. **Backpressure:** The current design is fire-and-forget (no backpressure). For high-throughput streaming, we may need a bounded channel. For the initial implementation, this is fine since events are dispatched asynchronously.

4. **Backwards compatibility of `runAsync`:** The return type changes from `Promise` to `RunHandle`. Existing code that does `await session.runAsync()` will break because the RunHandle is not a Promise. Options:
   - **Option A (recommended):** Keep existing `runAsync()` returning Promise. Add new `session.start()` that returns RunHandle.
   - **Option B:** Make RunHandle thenable (has `.then()` that delegates to `.promise.then()`). This makes `await handle` work, but is more complex.

   **Recommendation: Option A.** Add a new `session.start(seed?, opts?)` method that returns a RunHandle, and leave `runAsync()` unchanged.

5. **Event sink lifecycle:** The collector should automatically unsubscribe when the inference completes to avoid leaking callbacks. This can be done by clearing the listeners map after the promise resolves/rejects.

### Impact & Backwards Compatibility

- **Backwards compatible** if we use Option A (new `session.start()` method, `runAsync()` unchanged)
- New type: `jsEventCollector` implementing `events.EventSink`
- New method on session object: `start(seed?, opts?)`
- Existing `run()` and `runAsync()` unmodified
- New context plumbing: `events.WithEventSinks(ctx, collector)` in the run goroutine

---

## Cross-Cutting: TypeScript Definitions

Even though goja is not TypeScript, `.d.ts` files provide enormous IDE value:
- Autocomplete in VS Code / WebStorm when editing `.js` files
- Documentation on hover
- Type checking with `// @ts-check` or `checkJs` in tsconfig

### Hybrid Approach: Generated Enums + Hand-Maintained API Surface

The `.d.ts` is split into two parts:

1. **Generated portion** — enum/const types produced by `cmd/gen-js-api` from `js_api_codegen.yaml`
2. **Hand-maintained portion** — API surface types (interfaces, function signatures) that reflect the Go API structure

This hybrid approach avoids maintaining enum values in three places (Go, YAML, TypeScript) while keeping the complex API surface types — which change rarely and require human judgment — as a hand-authored template.

### Output Structure

The generator produces a single file by combining a **template skeleton** with generated const types:

**Template file:** `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`

This is a Go `text/template` file that the generator fills in:

```typescript
// Code generated by cmd/gen-js-api from geppetto.d.ts.tmpl + js_api_codegen.yaml. DO NOT EDIT.

declare module "geppetto" {
    export const version: string;

    // ==========================================
    // Constants (generated from js_api_codegen.yaml)
    // ==========================================

    export const consts: {
{{- range .Enums }}
        /** {{ .Doc }} */
        {{ .Name }}: {
{{- range .Values }}
            readonly {{ .JsKey }}: "{{ .Value }}";
{{- end }}
        };
{{- end }}
    };

    // ==========================================
    // Turns & Blocks
    // ==========================================

    interface Block {
        kind: string;
        text?: string;
        id?: string;
        name?: string;
        args?: any;
        result?: any;
        error?: string;
        metadata?: Record<string, any>;
    }

    interface Turn {
        id?: string;
        blocks: Block[];
        metadata?: Record<string, any>;
        data?: Record<string, any>;
    }

    export const turns: {
        normalize(turn: Turn): Turn;
        newTurn(data?: Partial<Turn>): Turn;
        appendBlock(turn: Turn, block: Block): Turn;
        newUserBlock(text: string): Block;
        newSystemBlock(text: string): Block;
        newAssistantBlock(text: string): Block;
        newToolCallBlock(id: string, name: string, args: any): Block;
        newToolUseBlock(id: string, result: any, error?: string): Block;
    };

    // ==========================================
    // Engines
    // ==========================================

    interface Engine {
        name: string;
    }

    interface EngineOptions {
        model?: string;
        apiType?: string;
        provider?: string;
        profile?: string;
        temperature?: number;
        topP?: number;
        maxTokens?: number;
        timeoutSeconds?: number;
        timeoutMs?: number;
        apiKey?: string;
        baseURL?: string;
    }

    export const engines: {
        echo(options?: { reply?: string }): Engine;
        fromProfile(profile?: string, options?: EngineOptions): Engine;
        fromConfig(options: EngineOptions): Engine;
        fromFunction(fn: (turn: Turn) => Turn | void): Engine;
    };

    // ==========================================
    // Middleware
    // ==========================================

    interface MiddlewareContext {
        sessionId?: string;
        inferenceId?: string;
        traceId?: string;
        turnId?: string;
        middlewareName: string;
        timestampMs: number;
        deadlineMs?: number;
    }

    type NextFn = (turn: Turn) => Turn;
    type MiddlewareFn = (turn: Turn, next: NextFn, ctx?: MiddlewareContext) => Turn;

    interface MiddlewareRef {
        type: "js" | "go";
        name: string;
    }

    export const middlewares: {
        fromJS(fn: MiddlewareFn, name?: string): MiddlewareRef;
        go(name: string, options?: Record<string, any>): MiddlewareRef;
    };

    // ==========================================
    // Tools
    // ==========================================

    interface ToolHandlerContext {
        callId?: string;
        toolName: string;
        timestampMs: number;
        deadlineMs?: number;
    }

    interface ToolSpec {
        name: string;
        description?: string;
        parameters?: Record<string, any>;
        handler: (args: Record<string, any>, ctx?: ToolHandlerContext) => any;
    }

    interface ToolInfo {
        name: string;
        description: string;
        version?: string;
        tags?: string[];
    }

    interface ToolRegistry {
        register(spec: ToolSpec): ToolRegistry;
        useGoTools(names?: string[]): ToolRegistry;
        list(): ToolInfo[];
        call(name: string, args?: Record<string, any>): any;
    }

    export const tools: {
        createRegistry(): ToolRegistry;
    };

    // ==========================================
    // Tool Hooks
    // ==========================================

    interface ToolHookCallInfo {
        id: string;
        name: string;
        args: any;
    }

    interface BeforeToolCallPayload {
        phase: "beforeToolCall";
        call: ToolHookCallInfo;
        timestampMs: number;
        sessionId?: string;
        inferenceId?: string;
    }

    interface AfterToolCallPayload {
        phase: "afterToolCall";
        call: ToolHookCallInfo;
        result: { value: any; error: string; durationMs: number };
        timestampMs: number;
        sessionId?: string;
        inferenceId?: string;
    }

    interface OnToolErrorPayload {
        phase: "onToolError";
        attempt: number;
        call: ToolHookCallInfo;
        error: string;
        defaultRetry: boolean;
        defaultBackoffMs: number;
        timestampMs: number;
        sessionId?: string;
        inferenceId?: string;
    }

    interface ToolHooks {
        beforeToolCall?: (payload: BeforeToolCallPayload) =>
            void | { action?: "abort"; error?: string; call?: Partial<ToolHookCallInfo> };
        afterToolCall?: (payload: AfterToolCallPayload) =>
            void | { action?: "abort"; result?: any; error?: string };
        onToolError?: (payload: OnToolErrorPayload) =>
            void | { action?: "abort" | "retry" | "continue"; backoffMs?: number };
        hookErrorPolicy?: "fail-open" | "open" | "fail-closed";
        failOpen?: boolean;
        maxHookRetries?: number;
    }

    // ==========================================
    // Tool Loop Settings
    // ==========================================

    interface ToolLoopSettings {
        enabled?: boolean;
        maxIterations?: number;
        maxParallelTools?: number;
        executionTimeoutMs?: number;
        toolChoice?: "auto" | "none" | "required";
        toolErrorHandling?: "continue" | "abort" | "retry";
        retryMaxRetries?: number;
        retryBackoffMs?: number;
        retryBackoffFactor?: number;
        allowedTools?: string[];
        hooks?: ToolHooks;
    }

    // ==========================================
    // Builder
    // ==========================================

    interface BuilderOptions {
        engine?: Engine;
        middlewares?: MiddlewareRef[];
        tools?: ToolRegistry;
        toolLoop?: ToolLoopSettings;
        toolHooks?: ToolHooks;
    }

    interface Builder {
        withEngine(engine: Engine): Builder;
        useMiddleware(middleware: MiddlewareRef | MiddlewareFn): Builder;
        useGoMiddleware(name: string, options?: Record<string, any>): Builder;
        withTools(registry: ToolRegistry, loopSettings?: ToolLoopSettings): Builder;
        withToolLoop(settings: ToolLoopSettings): Builder;
        withToolHooks(hooks: ToolHooks): Builder;
        buildSession(): Session;
    }

    export function createBuilder(options?: BuilderOptions): Builder;

    // ==========================================
    // Session
    // ==========================================

    interface RunOptions {
        timeoutMs?: number;
        tags?: Record<string, any>;
    }

    interface StreamEvent {
        type: string;
        sessionId?: string;
        inferenceId?: string;
        turnId?: string;
        timestampMs: number;
        delta?: string;
        completion?: string;
        text?: string;
        error?: string;
        toolCall?: { id: string; name: string; input: string };
        toolResult?: { id: string; result: string };
    }

    interface RunHandle {
        promise: Promise<Turn>;
        cancel(): void;
        on(eventType: string, callback: (event: StreamEvent) => void): RunHandle;
    }

    interface Session {
        append(turn: Turn): Turn;
        latest(): Turn | null;
        turnCount(): number;
        turns(): Turn[];
        getTurn(index: number): Turn | null;
        turnsRange(start?: number, end?: number): Turn[];
        isRunning(): boolean;
        cancelActive(): void;
        run(seedTurn?: Turn, options?: RunOptions): Turn;
        runAsync(seedTurn?: Turn): Promise<Turn>;
        start(seedTurn?: Turn, options?: RunOptions): RunHandle;
    }

    interface SessionOptions extends BuilderOptions {}

    export function createSession(options: SessionOptions): Session;

    export function runInference(engine: Engine, turn: Turn, options?: BuilderOptions): Turn;
}
```

### How the Generator Uses the Template

The `cmd/gen-js-api` generator:

1. Parses `js_api_codegen.yaml` to get enum definitions
2. Reads `geppetto.d.ts.tmpl` as a Go `text/template`
3. Executes the template with the parsed enum data
4. Writes the result to `pkg/doc/types/geppetto.d.ts`

```go
// In cmd/gen-js-api/main.go

func generateTypeScript(schema *Schema, tmplPath, outPath string) error {
    tmplBytes, err := os.ReadFile(tmplPath)
    if err != nil {
        return fmt.Errorf("read template: %w", err)
    }
    t, err := template.New("dts").Parse(string(tmplBytes))
    if err != nil {
        return fmt.Errorf("parse template: %w", err)
    }
    var buf bytes.Buffer
    if err := t.Execute(&buf, schema); err != nil {
        return fmt.Errorf("execute template: %w", err)
    }
    return os.WriteFile(outPath, buf.Bytes(), 0644)
}
```

The `go:generate` directive (from section 5.2) already covers this:

```go
//go:generate go run ../../../../cmd/gen-js-api \
//    --schema spec/js_api_codegen.yaml \
//    --dts-template spec/geppetto.d.ts.tmpl \
//    --go-out . \
//    --ts-out ../../../../pkg/doc/types
```

### Why This Hybrid Approach Works Well

- **Enum values are single-sourced** in `js_api_codegen.yaml` — no manual sync between Go, JS runtime, and TypeScript
- **API surface types are readable** — the `.d.ts.tmpl` template is mostly plain TypeScript with only a small `{{ range }}` block for the `consts` section
- **Maintainable by non-TypeScript experts** — adding a new function/interface means copying the pattern of an existing one; the template syntax is minimal
- **Build-time validation** — if the YAML schema changes, `go generate` re-renders the `.d.ts`; stale files are caught by CI (diff check on generated files)

### Alternative Considered: Full Codegen from Go Types

We considered generating the entire `.d.ts` from Go reflection or AST analysis. This was rejected because:
- Go and JS have fundamentally different type systems (goja bridges dynamically)
- Many JS-facing types don't have direct Go struct equivalents (e.g., `ToolSpec.handler` is a JS callable)
- The maintenance cost of a full Go→TS generator far exceeds the benefit for ~30 interface definitions that change infrequently

---

## Implementation Order and Dependencies

| Phase | Area | Effort | Dependencies | New/Modified Files |
|-------|------|--------|-------------|-------------------|
| 1 | 5.1 Opaque handles (DefineDataProperty) | Small | None | `module.go`, `api.go` |
| 1 | 5.2 Codegen infrastructure | Medium | None | `cmd/gen-js-api/main.go`, `spec/js_api_codegen.yaml`, `spec/geppetto.d.ts.tmpl`, `generate.go` |
| 1 | 5.2 Run codegen → `consts_gen.go` + `.d.ts` | Small | Needs codegen infra | `consts_gen.go` (generated), `geppetto.d.ts` (generated) |
| 1 | 5.2 Wire `installConsts()` into `installExports()` | Small | Needs `consts_gen.go` | `module.go` |
| 2 | 5.3a Middleware context | Medium | None | `api.go` |
| 2 | 5.3b Tool handler context | Medium | None | `api.go` |
| 2 | 5.3c Session context plumbing | Medium | None | `session/context.go` (new), `session.go` |
| 2 | 5.3c Tool hook enrichment | Medium | Needs session context plumbing | `api.go` |
| 3 | 5.4 RunHandle + streaming | Large | Needs event loop, session context, event sink integration | `api.go` |
| 3 | 5.4 Per-run options | Medium | Pairs with RunHandle | `api.go` |
| 4 | Update `.d.ts.tmpl` with 5.3/5.4 types | Small | Should reflect all API changes from phases 1-3 | `spec/geppetto.d.ts.tmpl`, re-run `go generate` |
| 4 | Add example script | Small | All phases | `examples/js/geppetto/07_context_and_constants.js` |

**Recommended order:**

1. **Phase 1** — Build the codegen infrastructure first (`cmd/gen-js-api` + YAML schema + `.d.ts.tmpl`), then do the `DefineDataProperty` fix. The codegen is a prerequisite for the constants export, and the `.d.ts` template can be written even before the API changes land (it just describes the target API surface).

2. **Phase 2** — Context plumbing is independent of Phase 1. Start with `session/context.go`, then wire middleware/tool handler ctx in parallel, then enrich hook payloads.

3. **Phase 3** — RunHandle is the largest piece. It depends on the event sink infrastructure and benefits from the context plumbing in 5.3c being in place.

4. **Phase 4** — Update the `.d.ts.tmpl` to include any new types from phases 2-3 (e.g., `MiddlewareContext`, `StreamEvent`, `RunHandle`), then re-run `go generate`. Add the example script.

**CI integration:** Add a `go generate ./...` + `git diff --exit-code` check to CI to ensure generated files are up to date.
