---
Title: Adopt Scoped JavaScript Eval Tools
Slug: geppetto-playbook-adopt-scopedjs-eval-tools
Short: Step-by-step guide to package a goja runtime as one `eval_xxx` tool using `pkg/inference/tools/scopedjs`.
Topics:
- geppetto
- javascript
- goja
- tools
- playbook
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Adopt Scoped JavaScript Eval Tools

This playbook shows how to expose a prepared JavaScript runtime as one Geppetto tool using `pkg/inference/tools/scopedjs`.

Use this when:

- the model should be able to compose multiple runtime capabilities in one script,
- your application can describe a bounded JS environment more clearly than many separate atomic tools,
- and you want the same reusable registration pattern as the scoped DB package: prebuilt runtime or lazy build from request context.

## Mental Model

```text
app scope/context
    |
    v
EnvironmentSpec + Builder
    |
    v
BuildRuntime(...)
    |
    v
RegisterPrebuilt(...) or NewLazyRegistrar(...)
    |
    v
tool registry entry: eval_xxx
    |
    v
model sends { code, input }
    |
    v
RunEval(...) executes inside owned goja runtime
```

## Step 1: Define the environment spec

The core API shape is:

```go
spec := scopedjs.EnvironmentSpec[Scope, Meta]{
    RuntimeLabel: "dbserver",
    Tool: scopedjs.ToolDefinitionSpec{
        Name: "eval_dbserver",
        Description: scopedjs.ToolDescription{
            Summary: "Execute JavaScript inside the scoped dbserver runtime.",
            Notes: []string{
                "Use return to provide the final result.",
            },
            StarterSnippets: []string{
                `const rows = await db.query("SELECT * FROM users"); return rows;`,
            },
        },
        Tags: []string{"javascript", "scopedjs"},
    },
    DefaultEval: scopedjs.DefaultEvalOptions(),
    Configure: func(ctx context.Context, b *scopedjs.Builder, scope Scope) (Meta, error) {
        // add modules, globals, bootstrap scripts, docs
        return meta, nil
    },
}
```

The important split is:

- `ToolDefinitionSpec` describes the LLM-facing tool.
- `Configure(...)` builds the actual runtime environment.
- `Scope` is application-owned data.
- `Meta` is optional application-owned build metadata.

## Step 2: Add modules, globals, and bootstrap code

Inside `Configure(...)`, use the builder:

```go
Configure: func(ctx context.Context, b *scopedjs.Builder, scope Scope) (Meta, error) {
    if err := b.AddNativeModule(myModule); err != nil {
        return Meta{}, err
    }

    if err := b.AddGlobal("db", func(ctx *gojengine.RuntimeContext) error {
        return ctx.VM.Set("db", newDBFacade(scope))
    }, scopedjs.GlobalDoc{
        Type:        "DatabaseFacade",
        Description: "Scoped database helper for the current request.",
    }); err != nil {
        return Meta{}, err
    }

    if err := b.AddBootstrapSource("helpers.js", `
async function fetchUsers() {
  return await db.query("SELECT * FROM users ORDER BY id");
}
`); err != nil {
        return Meta{}, err
    }

    return Meta{}, nil
}
```

Use:

- `AddNativeModule(...)` for go-go-goja native modules.
- `AddModule(...)` if you want to register a custom require module manually.
- `AddGlobal(...)` for runtime-bound globals installed during runtime initialization.
- `AddBootstrapSource(...)` or `AddBootstrapFile(...)` for helper JS loaded before eval begins.

## Step 3: Choose prebuilt or lazy registration

### Prebuilt runtime

Use this when the runtime can be constructed ahead of time and safely reused.

```go
handle, err := scopedjs.BuildRuntime(ctx, spec, scope)
if err != nil {
    return err
}
defer handle.Cleanup()

registry := tools.NewInMemoryToolRegistry()
if err := scopedjs.RegisterPrebuilt(registry, spec, handle, scopedjs.EvalOptions{}); err != nil {
    return err
}
```

### Lazy runtime

Use this when the runtime depends on request/session context.

```go
registrar := scopedjs.NewLazyRegistrar(spec, func(ctx context.Context) (Scope, error) {
    scope, ok := ctx.Value(scopeKey{}).(Scope)
    if !ok {
        return Scope{}, fmt.Errorf("missing scope")
    }
    return scope, nil
}, scopedjs.EvalOptions{})

registry := tools.NewInMemoryToolRegistry()
if err := registrar(registry); err != nil {
    return err
}
```

## Step 4: Understand the tool payload

The model sends:

```json
{
  "code": "const rows = await db.query(\"SELECT * FROM users\"); return rows;",
  "input": {
    "limit": 10
  }
}
```

The tool returns:

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

## Step 5: Start with the smallest real example

See the runnable examples:

```text
cmd/examples/scopedjs-tool/main.go
cmd/examples/scopedjs-dbserver/main.go
```

The smaller example demonstrates:

- registering the `fs` native module,
- injecting a `workspaceRoot` global,
- loading one bootstrap helper file,
- registering a prebuilt `eval_fs_demo` tool,
- and executing it through the normal Geppetto tool-definition path.

The composed example demonstrates:

- registering `fs`, `webserver`, and `obsidian` modules,
- injecting a scoped `db` global,
- loading a bootstrap helper,
- composing multiple capabilities in one eval call,
- and exposing the whole environment as `eval_dbserver_demo`.

Run it from the repo root with:

```bash
env GOWORK=off GOCACHE=/tmp/geppetto-go-build go run ./cmd/examples/scopedjs-tool
env GOWORK=off GOCACHE=/tmp/geppetto-go-build go run ./cmd/examples/scopedjs-dbserver
```

## Step 6: Scale up to the motivating dbserver shape

The intended larger composition looks like this:

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

## Operational Notes

- Prefer `StatePerCall` first. It keeps runtime state bounded and makes debugging simpler.
- Treat every module/global as a capability grant. If you expose a powerful object, the model has that power.
- Keep helper bootstrap files small and well-documented. They become part of the tool contract.
- Use the builder docs intentionally. The tool description is the model's API reference.

## Recommended Adoption Order

1. Start with one prebuilt runtime and one small module/global pair.
2. Verify `RunEval(...)` directly in tests.
3. Register the tool through `RegisterPrebuilt(...)`.
4. Only then add lazy context-derived scope and more modules.
5. Only then introduce larger app-specific compositions such as dbserver or obsidian automation.
