---
Title: Build Scoped JavaScript Eval Tools
Slug: geppetto-build-scopedjs-eval-tools
Short: Intern-friendly developer guide to designing, wiring, and debugging `eval_xxx` tools with `pkg/inference/tools/scopedjs`.
Topics:
- geppetto
- tutorial
- javascript
- goja
- tools
- js-bindings
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This guide explains how to create your own scoped JavaScript tool in Geppetto using `pkg/inference/tools/scopedjs`. The audience is a new intern who knows Go, is comfortable reading code, but has not yet internalized how Geppetto turns a prepared goja runtime into one LLM-facing tool such as `eval_fs_demo` or `eval_dbserver`.

The main idea is simple, but the system has several moving parts that matter for correctness. Your application owns the runtime contents: which modules exist, which globals exist, which bootstrap JavaScript files run, and what scope the tool is allowed to see. `scopedjs` owns the reusable parts: booting the runtime, loading those components, building a good tool description, and exposing a consistent `{ code, input } -> { result, console, error }` tool contract.

## What You Will Build

By the end of this tutorial, you should be able to build a tool that feels like this:

- the host application decides what "scope" means, such as one workspace, one account, or one temporary environment
- Go code creates a prepared JavaScript runtime for that scope
- the runtime exposes modules like `require("fs")`, globals like `db` or `workspaceRoot`, and helper functions loaded from bootstrap files
- Geppetto registers that prepared runtime as one tool named `eval_xxx`
- the model sends JavaScript source in `code` and optional structured input in `input`
- the tool returns a structured `EvalOutput`

Concrete examples already exist in the repo:

- `cmd/examples/scopedjs-tool/main.go`
- `cmd/examples/scopedjs-dbserver/main.go`

Read those after this guide, not before. This guide gives you the mental model those examples assume.

## Why Scoped JavaScript Tools Exist

Ordinary tools work best when the host already knows the exact function shape: `get_weather(city)` or `lookup_ticket(id)`. That breaks down when the model needs to compose multiple capabilities in one step. If the task is "query the scoped data, write a file, create a note, and register a route", many tiny tools become awkward because the model has to coordinate state across multiple calls.

A scoped JavaScript tool solves that by moving the composition boundary into the runtime. Instead of advertising five tiny tools, you advertise one bounded tool whose environment is already prepared for a specific scope. The model can then write one small script against that environment.

That gives you:

- flexibility: the model can compose capabilities in one call
- control: the host still decides the exact runtime surface
- reuse: the host app does not need to reimplement runtime bootstrap for every project
- explainability: the generated tool description can tell the model exactly which modules, globals, and helpers exist

## Mental Model

Think about the system as a pipeline, not as one magic function.

```text
application scope
  workspace/account/request/session
        |
        v
EnvironmentSpec[Scope, Meta]
  RuntimeLabel
  Tool metadata
  DefaultEval options
  Configure callback
        |
        v
Builder
  AddNativeModule(...)
  AddGlobal(...)
  AddBootstrapSource(...)
  AddHelper(...)
        |
        v
BuildRuntime(...)
  create goja runtime
  install modules
  install globals
  run bootstrap JS
        |
        v
RegisterPrebuilt(...) or NewLazyRegistrar(...)
        |
        v
LLM-facing tool: eval_xxx
        |
        v
RunEval(...)
  input:  { code, input }
  output: { result, console, error, durationMs }
```

Why this matters: when something breaks, the fix depends on which layer failed. A missing global is a `Configure(...)` problem. A missing tool description detail is a `ToolDescription` or manifest problem. A promise rejection format issue is an eval/runtime problem.

## System Map

These are the files you should understand before you change `scopedjs` behavior:

| File | Why it matters |
|------|----------------|
| `pkg/inference/tools/scopedjs/schema.go` | Public types: `EnvironmentSpec`, `EvalInput`, `EvalOutput`, `EvalOptions`, `Builder` manifest docs |
| `pkg/inference/tools/scopedjs/builder.go` | Builder methods for registering modules, globals, bootstrap sources, and helper docs |
| `pkg/inference/tools/scopedjs/runtime.go` | `BuildRuntime(...)` and the code that converts builder state into a live goja runtime |
| `pkg/inference/tools/scopedjs/eval.go` | `RunEval(...)`, async wrapper execution, console capture, promise handling, timeout behavior |
| `pkg/inference/tools/scopedjs/tool.go` | `RegisterPrebuilt(...)` and `NewLazyRegistrar(...)` |
| `pkg/inference/tools/scopedjs/description.go` | How the model-facing tool description is synthesized from your docs and manifest |
| `cmd/examples/scopedjs-tool/main.go` | Minimal runnable example with `fs` and `workspaceRoot` |
| `cmd/examples/scopedjs-dbserver/main.go` | Composed example with `fs`, fake `webserver`, fake `obsidian`, and a `db` global |

The key design principle is separation of concerns:

- your application decides what the runtime contains
- `scopedjs` decides how to host and expose that runtime consistently

## Prerequisites and Imports

Your Go file will need these imports to build a scopedjs tool:

```go
import (
    "context"

    gojengine "github.com/go-go-golems/go-go-goja/engine"
    ggjmodules "github.com/go-go-golems/go-go-goja/modules"
    _ "github.com/go-go-golems/go-go-goja/modules/fs"  // side-effect import: registers the fs module

    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/inference/tools/scopedjs"
)
```

Important: the blank import `_ "github.com/go-go-golems/go-go-goja/modules/fs"` is required for the `fs` module to be available via `ggjmodules.GetModule("fs")`. Without it, `GetModule` returns nil. Add similar blank imports for any other go-go-goja modules you want to use.

## Step 1: Decide the Runtime Boundary

Before you write code, decide what one eval tool is allowed to mean. This is the most important design step because a good runtime boundary keeps the tool understandable for both the model and the next developer.

Start by answering these questions:

- What scope does one call operate on?
- Which capabilities must be composable in one script?
- Which capabilities should remain separate tools because they are too dangerous or too expensive?
- Should runtime state persist across calls, or should each call start fresh?

Good boundary:

- "This tool operates on one temporary workspace and can read files, query a scoped data facade, and produce note metadata."

Bad boundary:

- "This tool can do whatever the whole application can do."

If you skip this step, the runtime usually becomes vague. That creates two downstream failures:

- the tool description becomes too fuzzy for the model to use well
- the host app quietly exposes too much ambient power to the runtime

## Step 2: Define `Scope` and `Meta`

`scopedjs` is generic over two host-owned types:

```go
type EnvironmentSpec[Scope any, Meta any] struct { ... }
```

`Scope` is the data required to build the runtime. `Meta` is extra information you want back after runtime construction.

Typical `Scope` examples:

- a workspace root path
- a struct containing `AccountID`, `WorkspaceID`, and a few already-opened handles
- a session-local configuration object

Typical `Meta` examples:

- counts of loaded fixtures
- resolved names for status bars or telemetry
- a label you want to show in logs

Pseudocode:

```go
type DemoScope struct {
    WorkspaceRoot string
    AccountID     string
}

type DemoMeta struct {
    ProjectName string
    FileCount   int
}
```

Why this split matters:

- `Scope` is input to runtime construction
- `Meta` is output from runtime construction

Do not overload `Meta` with data the runtime actually needs to function. If JavaScript code needs something, put it in the runtime as a module, global, or bootstrap helper.

## Step 3: Define `EnvironmentSpec`

`EnvironmentSpec` is the main configuration object. It tells `scopedjs` what to build and what the final tool should look like.

Core shape from `pkg/inference/tools/scopedjs/schema.go`:

```go
type EnvironmentSpec[Scope any, Meta any] struct {
    RuntimeLabel string
    Tool         ToolDefinitionSpec
    DefaultEval  EvalOptions
    Describe     func() (EnvironmentManifest, error)   // optional
    Configure    func(ctx context.Context, b *Builder, scope Scope) (Meta, error)
}
```

What each field does:

- `RuntimeLabel` — human-readable label for logs and error messages
- `Tool` — model-facing tool metadata (name, description, starter snippets)
- `DefaultEval` — default eval options (timeout, etc.); use `DefaultEvalOptions()` for sensible defaults
- `Describe` — optional callback that returns a static `EnvironmentManifest` describing available modules, globals, helpers, and bootstrap files. When provided, this manifest is used for the tool description instead of (or merged with) what the builder collects during `Configure`. Useful when the manifest is known statically and you want to separate description from runtime wiring. If omitted, the manifest is built entirely from builder method calls inside `Configure`.
- `Configure` — callback that receives a `*Builder` and populates it with modules, globals, bootstrap code, and helper docs. This is where the runtime is actually wired.

A minimal spec looks like this:

```go
spec := scopedjs.EnvironmentSpec[DemoScope, DemoMeta]{
    RuntimeLabel: "project-ops",
    Tool: scopedjs.ToolDefinitionSpec{
        Name: "eval_project_ops",
        Description: scopedjs.ToolDescription{
            Summary: "Execute JavaScript in the scoped project runtime.",
            Notes: []string{
                "Use return to provide the final result.",
            },
            StarterSnippets: []string{
                `const rows = db.query("SELECT * FROM notes"); return rows;`,
            },
        },
        Tags:    []string{"javascript", "tools"},
        Version: "1.0.0",
    },
    DefaultEval: scopedjs.DefaultEvalOptions(),
    Configure: func(ctx context.Context, b *scopedjs.Builder, scope DemoScope) (DemoMeta, error) {
        // runtime wiring happens here
        return DemoMeta{}, nil
    },
}
```

Read that in two halves:

- `Tool` describes what the model sees
- `Configure(...)` describes what the runtime actually contains

If you forget to document the tool well, the runtime may work technically but still perform badly with the model because the generated description will not teach the model what it can call.

## Step 4: Populate the Runtime with `Builder`

`Configure(...)` receives a `*scopedjs.Builder`. This is where you register the runtime contents. Most work happens here.

The main builder methods live in `pkg/inference/tools/scopedjs/builder.go`:

| Method | Use it for | Typical example |
|--------|------------|-----------------|
| `AddNativeModule(...)` | Existing go-go-goja native modules | `fs`, a custom native module |
| `AddModule(...)` | Manual `require(...)` registration | A one-off module not implemented as `NativeModule` |
| `AddGlobal(...)` | Scope-bound globals installed at runtime init | `db`, `workspaceRoot`, `config` |
| `AddInitializer(...)` | Extra runtime init logic | advanced setup when globals are not enough |
| `AddBootstrapSource(...)` | Inline helper JS | `joinPath(...)`, helper wrappers |
| `AddBootstrapFile(...)` | Preload a JS file from disk | `bootstrap/routes.js` |
| `AddHelper(...)` | Documentation for helper functions | `joinPath(a, b)` |

### 4a. Add native modules

This is how you expose modules that are loaded via `require("...")`.

```go
if err := b.AddNativeModule(fsModule); err != nil {
    return DemoMeta{}, err
}
```

How it works in practice:

- you bring a `go-go-goja` native module
- `scopedjs` records it in the manifest
- `BuildRuntime(...)` installs it into the goja `require` registry

Why it matters: native modules are the cleanest way to expose reusable JS-facing APIs backed by Go.

### 4b. Add globals

Globals are installed during runtime initialization, not through `require(...)`.

```go
if err := b.AddGlobal("workspaceRoot", func(ctx *gojengine.RuntimeContext) error {
    return ctx.VM.Set("workspaceRoot", scope.WorkspaceRoot)
}, scopedjs.GlobalDoc{
    Type:        "string",
    Description: "Scoped root directory for this workspace.",
}); err != nil {
    return DemoMeta{}, err
}
```

Note: `GlobalDoc` also has a `Name` field, but `AddGlobal(...)` populates it automatically from the first argument. You only need to set `Type` and `Description`.

Use globals for data that is naturally ambient to the runtime:

- a scoped path root
- a prepared host facade like `db`
- a small config object

Failure mode if you misuse globals: if you stuff too much behavior into one giant global object, the runtime becomes hard to document and hard to test. Prefer modules for larger capability surfaces.

### 4c. Add bootstrap JavaScript

Bootstrap code runs before the model's `code` executes. This is a good place for helper functions that are small, predictable, and easier to write in JavaScript than in Go.

```go
if err := b.AddBootstrapSource("helpers.js", `
function joinPath(a, b) {
  return a.replace(/\/$/, "") + "/" + b.replace(/^\//, "");
}
`); err != nil {
    return DemoMeta{}, err
}
```

Then document it:

```go
if err := b.AddHelper("joinPath", "joinPath(a, b)", "Join workspace-relative path segments."); err != nil {
    return DemoMeta{}, err
}
```

Why bootstrap exists:

- it keeps tiny JS helpers out of Go glue code
- it lets you shape the runtime ergonomics without creating a full module

### 4d. Full `Configure(...)` pseudocode

This is the shape you should have in your head:

```go
Configure: func(ctx context.Context, b *scopedjs.Builder, scope DemoScope) (DemoMeta, error) {
    if err := b.AddNativeModule(fsModule); err != nil {
        return DemoMeta{}, err
    }

    if err := b.AddNativeModule(customModule); err != nil {
        return DemoMeta{}, err
    }

    if err := b.AddGlobal("db", func(ctx *gojengine.RuntimeContext) error {
        return ctx.VM.Set("db", newScopedDBFacade(scope))
    }, scopedjs.GlobalDoc{
        Type: "object",
        Description: "Scoped data facade for the current runtime.",
    }); err != nil {
        return DemoMeta{}, err
    }

    if err := b.AddBootstrapFile("bootstrap/helpers.js"); err != nil {
        return DemoMeta{}, err
    }

    if err := b.AddHelper("joinPath", "joinPath(a, b)", "Join scoped paths."); err != nil {
        return DemoMeta{}, err
    }

    return DemoMeta{
        ProjectName: scope.AccountID,
    }, nil
}
```

## Step 5: Build the Runtime

`BuildRuntime(...)` turns your spec and scope into a live goja runtime plus manifest metadata. The implementation lives in `pkg/inference/tools/scopedjs/runtime.go`.

What `BuildRuntime(...)` does:

- calls your `Configure(...)`
- converts builder state into module specs and runtime initializers
- creates the runtime factory
- creates a runtime instance
- loads bootstrap files and inline bootstrap sources
- returns a `BuildResult`

The return type is:

```go
type BuildResult[Meta any] struct {
    Runtime  *gojengine.Runtime
    Meta     Meta
    Manifest EnvironmentManifest
    Cleanup  func() error
}
```

Why `Manifest` matters: this is how your builder docs become part of the generated model-facing description. If you forget to register docs when you register capabilities, the runtime works but the tool description becomes weaker.

## Step 6: Choose Prebuilt vs Lazy Registration

After the runtime exists, you still need to expose it as a Geppetto tool. There are two modes.

### Prebuilt: `RegisterPrebuilt(...)`

Use this when the runtime is safe and sensible to build ahead of time.

```text
build scope now
    |
    v
BuildRuntime(...)
    |
    v
RegisterPrebuilt(...)
    |
    v
tool executes against that already-built runtime
```

Typical cases:

- examples
- one temp workspace for one process
- a demo environment you already materialized

Code shape:

```go
handle, err := scopedjs.BuildRuntime(ctx, spec, scope)
if err != nil {
    return err
}
defer handle.Cleanup()

registry := tools.NewInMemoryToolRegistry()
if err := scopedjs.RegisterPrebuilt(registry, spec, handle, scopedjs.EvalOptionOverrides{}); err != nil {
    return err
}
```

### Lazy: `NewLazyRegistrar(...)`

Use this when the runtime depends on request or session context and should be built on demand.

```text
tool call arrives
    |
    v
resolve scope from context
    |
    v
BuildRuntime(...)
    |
    v
RunEval(...)
    |
    v
cleanup runtime
```

Typical cases:

- per-request account scoping
- session-specific runtime views
- environments that must not be reused across users

Code shape:

```go
registrar := scopedjs.NewLazyRegistrar(spec, func(ctx context.Context) (DemoScope, error) {
    scope, ok := ctx.Value(scopeKey{}).(DemoScope)
    if !ok {
        return DemoScope{}, fmt.Errorf("missing scope")
    }
    return scope, nil
}, scopedjs.EvalOptionOverrides{})

if err := registrar(registry); err != nil {
    return err
}
```

How to choose:

| Use prebuilt when... | Use lazy when... |
|----------------------|------------------|
| the scope is stable for the life of the process | the scope changes per call or per session |
| startup cost is acceptable | runtime creation depends on request context |
| you want simple example code | you need strong isolation between callers |

## Step 7: Understand the Eval Contract

The model-facing input and output types live in `pkg/inference/tools/scopedjs/schema.go`.

Input:

```go
type EvalInput struct {
    Code  string         `json:"code"`
    Input map[string]any `json:"input,omitempty"`
}
```

- `Code` — the JavaScript source the model (or caller) writes. It is wrapped in an async function, so `await` and `return` work naturally.
- `Input` — optional structured data passed alongside the code. Inside the JavaScript execution context, it is available as the global variable `input`. For example, if the tool call includes `"input": {"path": "/tmp/file.txt", "limit": 10}`, the JS code can access `input.path` and `input.limit`. This lets the model separate data from logic: the code is the algorithm, the input is the parameters.

When calling the tool directly from Go (e.g. in tests), you provide `Input` as a `map[string]any`:

```go
args, _ := json.Marshal(scopedjs.EvalInput{
    Code: `const fs = require("fs"); return fs.readFileSync(input.path);`,
    Input: map[string]any{
        "path": "/tmp/hello.txt",
    },
})
result, err := def.Function.ExecuteWithContext(ctx, args)
```

Output:

```go
type EvalOutput struct {
    Result     any           `json:"result,omitempty"`
    Console    []ConsoleLine `json:"console,omitempty"`
    Error      string        `json:"error,omitempty"`
    DurationMs int64         `json:"durationMs,omitempty"`
}
```

- `Result` — the value from `return` in the JavaScript code. Can be any JSON-serializable value.
- `Console` — captured `console.log(...)`, `console.error(...)`, etc. Each entry has `Level` and `Text`.
- `Error` — non-empty when the script threw, rejected a promise, or timed out. The model sees this as a normal tool result, not a crash.
- `DurationMs` — wall-clock execution time in milliseconds.

The JavaScript is wrapped in an async function, so `await` and `return` work naturally:

```js
const rows = db.query("SELECT * FROM notes");
console.log("loaded", rows.length);
return rows;
```

On the wire, the model sends JSON like:

```json
{
  "code": "const rows = await db.query(\"SELECT * FROM users\"); return rows;",
  "input": {
    "limit": 10
  }
}
```

And receives:

```json
{
  "result": [{ "id": 1, "name": "Ada" }],
  "console": [{ "level": "log", "text": "loaded users" }],
  "durationMs": 12
}
```

On rejection or timeout, the tool still returns a structured payload:

```json
{
  "error": "Promise rejected: boom",
  "console": [],
  "durationMs": 4
}
```

Why this contract is useful:

- the final result stays machine-friendly
- console output stays visible but separate
- the host can render the two differently in a UI

## Step 8: Complete Minimal Example

This is the smallest realistic pattern, condensed from `cmd/examples/scopedjs-tool/main.go`.

```go
fsModule := ggjmodules.GetModule("fs")

spec := scopedjs.EnvironmentSpec[string, struct{}]{
    RuntimeLabel: "fs-demo",
    Tool: scopedjs.ToolDefinitionSpec{
        Name: "eval_fs_demo",
        Description: scopedjs.ToolDescription{
            Summary: "Execute JavaScript against a scoped workspace with fs access.",
        },
    },
    DefaultEval: scopedjs.DefaultEvalOptions(),
    Configure: func(ctx context.Context, b *scopedjs.Builder, root string) (struct{}, error) {
        if err := b.AddNativeModule(fsModule); err != nil {
            return struct{}{}, err
        }
        if err := b.AddGlobal("workspaceRoot", func(ctx *gojengine.RuntimeContext) error {
            return ctx.VM.Set("workspaceRoot", root)
        }, scopedjs.GlobalDoc{
            Type: "string",
            Description: "Scoped workspace root.",
        }); err != nil {
            return struct{}{}, err
        }
        if err := b.AddBootstrapSource("helpers.js", `
function joinPath(a, b) { return a + "/" + b; }
`); err != nil {
            return struct{}{}, err
        }
        return struct{}{}, nil
    },
}

handle, err := scopedjs.BuildRuntime(ctx, spec, workspaceDir)
if err != nil {
    return err
}
defer handle.Cleanup()

registry := tools.NewInMemoryToolRegistry()
if err := scopedjs.RegisterPrebuilt(registry, spec, handle, scopedjs.EvalOptionOverrides{}); err != nil {
    return err
}
```

The important lesson is not the `fs` module itself. The lesson is the pattern:

- define scope
- define spec
- populate builder
- build runtime
- register tool

Run the examples from the repo root with:

```bash
env GOWORK=off GOCACHE=/tmp/geppetto-go-build go run ./cmd/examples/scopedjs-tool
env GOWORK=off GOCACHE=/tmp/geppetto-go-build go run ./cmd/examples/scopedjs-dbserver
```

## Step 9: Scaling Up — the Composed dbserver Shape

The minimal example above uses a single module and global. A real application may compose several capabilities into one runtime. The intended pattern for a larger tool looks like this:

```text
eval_dbserver runtime
  - require("fs")
  - require("webserver")
  - require("obsidian")
  - global db
  - bootstrap sql_helpers.js
  - bootstrap routes.js
```

Pseudocode:

```go
Configure: func(ctx context.Context, b *scopedjs.Builder, scope ServerScope) (Meta, error) {
    b.AddNativeModule(fsModule)
    b.AddNativeModule(webserverModule)
    b.AddNativeModule(obsidianModule)

    b.AddGlobal("db", func(ctx *gojengine.RuntimeContext) error {
        return ctx.VM.Set("db", NewScopedDBFacade(scope.DB))
    }, scopedjs.GlobalDoc{
        Type: "ScopedDBFacade",
        Description: "Database facade for the current scoped server environment.",
    })

    b.AddBootstrapFile("bootstrap/sql_helpers.js")
    b.AddBootstrapFile("bootstrap/routes.js")

    return Meta{}, nil
}
```

Then the model can write code like:

```js
const rows = await db.query("SELECT id, title FROM notes ORDER BY id");
const server = require("webserver");

server.get("/notes", () => rows);

return {
  route: "/notes",
  count: rows.length
};
```

See `cmd/examples/scopedjs-dbserver/main.go` for the full runnable version of this pattern.

## Step 10: Know Where Description Text Comes From

Many developers assume the tool description is only whatever they put in `ToolDescription.Summary`. That is not true.

`pkg/inference/tools/scopedjs/description.go` builds the final description from several sources:

- `ToolDescription.Summary`
- module docs gathered through the builder manifest
- global docs gathered through the builder manifest
- helper docs gathered through the builder manifest
- bootstrap file names
- eval state mode notes
- starter snippets

This is why documentation must be added at registration time, not retrofitted later. If you expose a capability but do not document it through the builder or `ToolDescription`, the model gets a runtime it cannot fully understand.

## Step 11: Test the Right Things

A good `scopedjs` tool should have tests at three levels.

### 1. Build tests

Verify the runtime builds and the expected tool is registered.

Questions to answer:

- does `BuildRuntime(...)` succeed?
- does the manifest include the expected modules/globals/helpers?
- does the registry contain the final tool name?

### 2. Direct eval tests

Call `RunEval(...)` directly or execute the tool definition without a full UI or agent loop.

Questions to answer:

- can JS call the expected modules and globals?
- does the returned result shape match expectations?
- is console capture working?

### 3. End-to-end behavior tests

Use a real tool loop or example binary if the runtime is meant for user-facing workflows.

Questions to answer:

- does the model reliably choose the tool?
- is the description strong enough?
- are results rendered in a way humans can understand?

Pseudocode test plan:

```text
test build
  -> runtime builds
  -> tool registered

test composed eval
  -> JS reads scoped data
  -> JS writes file or returns note metadata
  -> JS registers route or similar side effect

test error path
  -> missing file or invalid helper usage
  -> error surfaces in EvalOutput.Error
```

## Common Design Mistakes

These mistakes show up early and cost time later.

| Mistake | Why it hurts | Better approach |
|---------|--------------|-----------------|
| stuffing everything into one giant global | hard to document and reason about | use modules for capability groups, globals for ambient context |
| giving the runtime ambient application access | weak isolation and surprising behavior | pass a narrow scope and expose only bounded facades |
| skipping helper docs | the model sees less than the runtime actually offers | always pair helper bootstrap with `AddHelper(...)` |
| writing examples before choosing state mode | confusing cross-call behavior | decide `StatePerCall`, `StatePerSession`, or `StateShared` up front |
| relying only on UI tests | hard to isolate failures | add direct runtime/eval tests first |

## Troubleshooting

This table covers the failures you are most likely to hit when building your first tool.

| Problem | Cause | Solution |
|---------|-------|----------|
| `runtime is nil` or tool registration fails | `BuildRuntime(...)` did not succeed or `handle.Runtime` was not checked | fix the runtime build first, then register |
| `require("fs")` or another module fails | the module was never registered in `Configure(...)` | call `AddNativeModule(...)` or `AddModule(...)` |
| a global like `db` is undefined | the global binding was not installed or returned an error | verify `AddGlobal(...)` and the runtime initializer path |
| helper function is missing | bootstrap source or file did not load | check `AddBootstrapSource(...)`, `AddBootstrapFile(...)`, and bootstrap errors |
| the model does not use the tool well | the description is too vague or missing manifest docs | improve `ToolDescription`, global docs, module docs, and starter snippets |
| result output is hard to understand | the JS returns ad hoc shapes | return a concise structured object with stable keys |
| error text is generic after promise rejection | rejection formatting may have lost JS error details | inspect `pkg/inference/tools/scopedjs/eval.go` and verify current runtime behavior with a direct eval test |

## Operational Notes

- Treat every module/global as a capability grant. If you expose a powerful object, the model has that power.
- Prefer the lazy registration path first if you need a fresh runtime per request. Use prebuilt registration only when shared runtime reuse is intentional.
- Keep helper bootstrap files small and well-documented. They become part of the tool contract.
- Use the builder docs intentionally. The tool description is the model's API reference.

## Recommended Adoption Order

1. Start with one prebuilt runtime and one small module/global pair.
2. Verify `RunEval(...)` directly in tests.
3. Register the tool through `RegisterPrebuilt(...)`.
4. Only then add lazy context-derived scope and more modules.
5. Only then introduce larger app-specific compositions such as dbserver or obsidian automation.

## See Also

- [Tools](../topics/07-tools.md) for the wider Geppetto tool model
- [Using Scoped Tool Databases](06-using-scoped-tool-databases.md) for the analogous `scopeddb` pattern
- `cmd/examples/scopedjs-tool/main.go` for the smallest runnable example
- `cmd/examples/scopedjs-dbserver/main.go` for the composed multi-capability example
