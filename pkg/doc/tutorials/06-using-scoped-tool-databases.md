---
Title: Using Scoped Tool Databases
Slug: geppetto-scoped-tool-databases
Short: Build a read-only scoped SQLite tool with `geppetto/pkg/inference/tools/scopeddb`, choose prebuilt or lazy registration, and keep queries safe.
Topics:
- geppetto
- tutorial
- tools
- sqlite
- scopeddb
- turns
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Using Scoped Tool Databases

This tutorial explains how to expose a scoped, read-only SQLite snapshot as a Geppetto tool using `github.com/go-go-golems/geppetto/pkg/inference/tools/scopeddb`.

The core idea is simple:

- your application defines a dataset schema
- your application materializes only the rows for one scope
- `scopeddb` wraps that SQLite database as a safe query tool
- the model can ask SQL questions, but only against the tables and views you allow

This pattern is useful when the model needs flexible read access to structured data, but you do not want to give it a live application database, write access, or unrestricted SQL.

## What Problem Scoped Tool Databases Solve

Ordinary Geppetto tools are great when you already know the exact function shape: `get_weather(location)` or `lookup_customer(id)`. They become awkward when the model needs exploratory access to a small, changing, structured dataset.

Examples:

- "Show the last 10 support tickets for this account."
- "Which transcript turns mention deployment failures?"
- "List commitments that changed status in this run."

You could build a separate tool for every query shape, but that scales badly. A scoped tool database gives the model one query tool backed by a temporary SQLite snapshot that only contains the data for the current scope.

That gives you:

- flexibility: the model can answer more than one canned question
- isolation: the tool only sees the current account, run, transcript, or request scope
- safety: only `SELECT` and `WITH` queries are allowed
- controllability: you define the schema, allowed objects, truncation rules, and timeout

## Mental Model

Think of the system as a three-layer stack:

```text
Application scope
  account_id="northwind"
  run_id="run-123"
  transcript_id="tr-42"
        |
        v
Scoped snapshot builder
  create SQLite schema
  copy only rows for that scope
  return DB handle + Meta
        |
        v
Geppetto tool
  query_support_history(sql, params)
  validates SQL
  enforces readonly access
  returns rows as JSON
```

Another way to say it:

- your app owns scope resolution and data loading
- `scopeddb` owns SQLite bootstrapping, query validation, and tool definition
- the tool loop owns model interaction and tool execution

## The Package Surface

The main files live in `pkg/inference/tools/scopeddb`:

- `schema.go`
  - schema/bootstrap types
  - `DatasetSpec`
  - `BuildInMemory`
  - `BuildFile`
- `tool.go`
  - `RegisterPrebuilt`
  - `NewLazyRegistrar`
- `query.go`
  - `QueryInput`
  - `QueryOutput`
  - `QueryOptions`
  - `QueryRunner`
- `description.go`
  - model-facing tool description assembly
- `helpers.go`
  - allowed-object normalization helpers

The package is intentionally small. Your application is expected to bring the domain-specific parts:

- the scope type
- the schema SQL
- the data materialization callback
- the registration point for the tool

## Core Types

### `DatasetSpec`

`DatasetSpec[Scope, Meta]` is the central configuration object.

```go
type DatasetSpec[Scope any, Meta any] struct {
    InMemoryPrefix string
    SchemaLabel    string
    SchemaSQL      string
    AllowedObjects []string
    Tool           ToolDefinitionSpec
    DefaultQuery   QueryOptions
    Materialize    func(ctx context.Context, dst *sql.DB, scope Scope) (Meta, error)
}
```

What each field means:

- `InMemoryPrefix`
  - naming prefix for generated in-memory SQLite DSNs
- `SchemaLabel`
  - human-readable label used in schema/bootstrap errors
- `SchemaSQL`
  - SQLite DDL for the scoped snapshot
- `AllowedObjects`
  - the tables and views the query validator will allow the model to reference
- `Tool`
  - model-facing tool definition metadata
- `DefaultQuery`
  - limits such as row count, column count, timeout, and `ORDER BY` policy
- `Materialize`
  - your callback that inserts scope-specific rows and returns typed `Meta`

### `Meta`

`Meta` exists for application-owned information produced while materializing the scoped database. `scopeddb` does not interpret it. Your code can use it for diagnostics, status messages, telemetry, or rendering.

Good `Meta` examples:

- number of rows copied per table
- the resolved scope id after alias lookup
- a cache key or dataset hash
- a note that a result was truncated before materialization

Bad `Meta` examples:

- anything required for the SQL tool to function
- large duplicate copies of the actual dataset

If the tool itself needs a table, put that information in SQLite, not in `Meta`.

### Tool definition metadata

The `Tool` field on `DatasetSpec` has this shape:

```go
type ToolDefinitionSpec struct {
    Name        string
    Description ToolDescription
    Tags        []string
    Version     string
}

type ToolDescription struct {
    Summary        string
    StarterQueries []string
    Notes          []string
}
```

This is application-authored input used to build the final Geppetto `tools.ToolDefinition`.

Important separation:

- `ToolDescription` is model-facing prose
- `SchemaSQL` is machine-consumed schema
- `QueryInput` is the callable tool argument schema

Do not collapse these into one field. They serve different consumers.

### Query contract

The generated tool takes a small input schema:

```go
type QueryInput struct {
    SQL    string   `json:"sql"`
    Params []string `json:"params,omitempty"`
}
```

And returns:

```go
type QueryOutput struct {
    Columns   []string
    Rows      []map[string]any
    Count     int
    Truncated bool
    Error     string
}
```

Why are `Params` strings instead of `[]any`?

- Geppetto reflects tool inputs to JSON Schema for providers
- provider-side function schema validation rejected the looser `[]any` shape
- `[]string` produces a provider-safe schema with `params.items.type = "string"`
- the query runner converts those strings into SQLite bind arguments at execution time

## Safe Querying Rules

`scopeddb` is not "SQLite access for the model". It is a constrained query surface.

The runner in `query.go` enforces several rules:

- only a single statement is allowed
- only `SELECT` or `WITH` queries are allowed
- referenced tables/views must be in `AllowedObjects`
- SQLite authorizer hooks prevent unsafe access patterns at execution time
- query execution uses a timeout
- row, column, and cell-size limits are enforced
- optional `RequireOrderBy` forces deterministic result ordering

That means this is the intended SQL style:

```sql
SELECT ticket_id, opened_at, status
FROM tickets
WHERE account_id = ?
ORDER BY opened_at DESC
LIMIT 10
```

And these are intentionally rejected:

- `DELETE FROM tickets`
- `SELECT * FROM tickets; SELECT * FROM comments`
- `SELECT * FROM sqlite_master`

## Two Registration Modes

There are two main ways to expose a scoped tool.

### 1. Prebuilt registration

Use `RegisterPrebuilt` when your application already built the scoped database and wants the tool to query that specific handle.

```text
request starts
  -> app resolves scope
  -> app builds snapshot once
  -> app registers tool against that DB
  -> model queries it many times during this run
```

This is the best option when:

- building the snapshot is expensive and should happen once
- you want to render or inspect `Meta` before inference starts
- the app already has a request-local lifecycle around the DB handle

### 2. Lazy registration

Use `NewLazyRegistrar` when you want the tool registration to be cheap and defer snapshot construction until the tool is actually called.

```text
request starts
  -> app registers lazy tool
  -> model may or may not call it
  -> on tool call:
       resolve scope from context
       build in-memory snapshot
       run query
       cleanup
```

This is the best option when:

- the tool is optional and often unused
- scope can only be resolved at tool-call time from context
- you want the simplest registration hook for app-owned registrars

Tradeoff:

- `NewLazyRegistrar` rebuilds the in-memory DB for each tool call
- `RegisterPrebuilt` lets one snapshot support multiple SQL calls

## Step-by-Step: Define a Scoped Dataset

Start by defining a scope type and a metadata type.

```go
type SupportScope struct {
    AccountID string
}

type SupportMeta struct {
    AccountID   string
    TicketCount int
    CommentCount int
}
```

Next, define the dataset spec.

```go
var supportHistorySpec = scopeddb.DatasetSpec[SupportScope, SupportMeta]{
    InMemoryPrefix: "support_history",
    SchemaLabel:    "support history schema",
    SchemaSQL: `
CREATE TABLE scope(
    account_id TEXT PRIMARY KEY
);

CREATE TABLE tickets(
    ticket_id   TEXT PRIMARY KEY,
    account_id  TEXT NOT NULL,
    subject     TEXT NOT NULL,
    status      TEXT NOT NULL,
    opened_at   TEXT NOT NULL
);

CREATE TABLE comments(
    comment_id  TEXT PRIMARY KEY,
    ticket_id   TEXT NOT NULL,
    author      TEXT NOT NULL,
    body        TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

CREATE VIEW latest_tickets AS
SELECT ticket_id, subject, status, opened_at
FROM tickets;
`,
    AllowedObjects: []string{
        "scope",
        "tickets",
        "comments",
        "latest_tickets",
    },
    Tool: scopeddb.ToolDefinitionSpec{
        Name: "query_support_history",
        Description: scopeddb.ToolDescription{
            Summary: "Query support ticket history for the currently selected account.",
            StarterQueries: []string{
                "SELECT ticket_id, subject, status FROM latest_tickets ORDER BY opened_at DESC LIMIT 10",
                "SELECT author, body FROM comments WHERE ticket_id = ? ORDER BY created_at ASC",
            },
            Notes: []string{
                "Use ? placeholders with params instead of inline literal values when filtering by ids.",
                "Prefer ORDER BY for deterministic results.",
            },
        },
        Tags:    []string{"sqlite", "support", "scopeddb"},
        Version: "v1",
    },
    DefaultQuery: scopeddb.QueryOptions{
        MaxRows:        100,
        MaxColumns:     32,
        MaxCellChars:   1000,
        Timeout:        5 * time.Second,
        RequireOrderBy: true,
    },
    Materialize: func(ctx context.Context, dst *sql.DB, scope SupportScope) (SupportMeta, error) {
        // 1. Copy only the rows for scope.AccountID
        // 2. Insert them into the scoped tables
        // 3. Return counts or other app-owned metadata
        return SupportMeta{
            AccountID:    scope.AccountID,
            TicketCount:  42,
            CommentCount: 128,
        }, nil
    },
}
```

## Step-by-Step: Build the Snapshot

If you want one prebuilt scoped DB per request, use `BuildInMemory`:

```go
scope := SupportScope{AccountID: "northwind"}

buildResult, err := scopeddb.BuildInMemory(ctx, supportHistorySpec, scope)
if err != nil {
    return err
}
defer func() { _ = buildResult.Cleanup() }()

fmt.Printf("loaded %d tickets for %s\n", buildResult.Meta.TicketCount, buildResult.Meta.AccountID)
```

The return type is:

```go
type BuildResult[Meta any] struct {
    DB      *sql.DB
    Meta    Meta
    Cleanup func() error
}
```

That gives you three things:

- `DB`
  - the ready-to-query SQLite handle
- `Meta`
  - application-owned materialization metadata
- `Cleanup`
  - a lifecycle hook for closing the DB

If you want to inspect the snapshot on disk during development or export it for debugging, use `BuildFile` instead:

```go
buildResult, err := scopeddb.BuildFile(ctx, "/tmp/support-history.sqlite", supportHistorySpec, scope)
```

`BuildFile` is useful for:

- debugging schema shape with external SQLite tools
- golden test fixtures
- ad hoc local investigation

It is usually not the default runtime path for request-scoped tools.

## Step-by-Step: Register a Prebuilt Tool

Once you have a built DB, register the tool into a Geppetto registry.

```go
registry := tools.NewInMemoryToolRegistry()

if err := scopeddb.RegisterPrebuilt(
    registry,
    supportHistorySpec,
    buildResult.DB,
    supportHistorySpec.DefaultQuery,
); err != nil {
    return err
}
```

What `RegisterPrebuilt` does:

- creates a `QueryRunner`
- builds a tool description from `Summary`, `Notes`, `StarterQueries`, and `AllowedObjects`
- constructs a normal Geppetto `ToolDefinition`
- registers it under `spec.Tool.Name`

After that, the tool loop sees a normal Geppetto tool. `scopeddb` is an implementation detail.

## Step-by-Step: Register a Lazy Tool

If you want the DB to be constructed only when the tool is called, use `NewLazyRegistrar`.

```go
type scopeKey struct{}

registrar := scopeddb.NewLazyRegistrar(
    supportHistorySpec,
    func(ctx context.Context) (SupportScope, error) {
        scope, ok := ctx.Value(scopeKey{}).(SupportScope)
        if !ok {
            return SupportScope{}, fmt.Errorf("support scope missing from context")
        }
        return scope, nil
    },
    supportHistorySpec.DefaultQuery,
)

registry := tools.NewInMemoryToolRegistry()
if err := registrar(registry); err != nil {
    return err
}
```

Important behavior:

- the scope resolver runs during tool execution, not during registration
- a fresh in-memory DB is built for that tool call
- the DB is cleaned up after the query finishes
- errors are returned as `QueryOutput.Error` so the model sees a normal tool result payload

This is the same basic pattern used for request-scoped runtime registrars in Geppetto-based applications.

## Putting It Into the Tool Loop

From the model's point of view, a scoped DB tool is just another registered tool.

```go
loop := toolloop.New(
    toolloop.WithEngine(eng),
    toolloop.WithRegistry(registry),
    toolloop.WithLoopConfig(toolloop.NewLoopConfig().WithMaxIterations(5)),
    toolloop.WithToolConfig(tools.DefaultToolConfig()),
)

updatedTurn, err := loop.RunLoop(ctx, seedTurn)
if err != nil {
    return err
}
_ = updatedTurn
```

The flow looks like this:

```text
user asks question
    |
    v
model emits tool_call(query_support_history, {sql, params})
    |
    v
scopeddb validates query and runs it
    |
    v
tool result returns rows/columns/count
    |
    v
model answers using the returned rows
```

## Recommended Application Structure

For most applications, keep the responsibilities split like this:

- package-local dataset spec
  - owns schema and materialization
- package-local scope resolver
  - knows how to extract request scope from context
- central tool registration point
  - decides whether to use prebuilt or lazy registration

Pseudocode:

```go
// package supporthistory
var Spec = scopeddb.DatasetSpec[SupportScope, SupportMeta]{...}

func ResolveScope(ctx context.Context) (SupportScope, error) { ... }

func NewRegistrar() func(tools.ToolRegistry) error {
    return scopeddb.NewLazyRegistrar(Spec, ResolveScope, Spec.DefaultQuery)
}
```

This keeps `scopeddb` generic and your domain logic local to your own package.

## Description Assembly and Prompting Guidance

The final tool description shown to the model is not only the one-line summary. `BuildDescription(...)` composes:

- `Summary`
- allowed tables/views
- `Notes`
- an `ORDER BY` hint when required
- `StarterQueries`

That matters because provider tool descriptions are often the only place the model sees usage hints before generating SQL.

Good starter queries:

- show the intended table names
- demonstrate joins or views the model should prefer
- show `ORDER BY`
- demonstrate bind parameters with `?`

Bad starter queries:

- use tables not in `AllowedObjects`
- omit ordering when deterministic results matter
- encourage literal string interpolation instead of params

## Testing Strategy

You should test three layers separately.

### 1. Schema/bootstrap tests

Test that the schema opens and materializes correctly:

- `BuildInMemory(...)` succeeds
- expected rows appear in the snapshot
- `Meta` is populated correctly

### 2. Query safety tests

Test the runner directly:

- allowed `SELECT` works
- write statements are rejected
- `sqlite_master` and disallowed tables are rejected
- `ORDER BY` enforcement works when enabled
- bind params work

### 3. Registration/schema tests

Test the registered tool definition:

- tool name is correct
- description contains the expected hints
- provider-facing parameter schema is valid

The existing tests in `pkg/inference/tools/scopeddb/query_test.go`, `pkg/inference/tools/scopeddb/schema_test.go`, and `pkg/inference/tools/scopeddb/tool_test.go` are the best starting point.

## A Minimal End-to-End Example

```go
func RegisterSupportHistoryTool(ctx context.Context, reg tools.ToolRegistry, accountID string) (scopeddb.BuildResult[SupportMeta], error) {
    scope := SupportScope{AccountID: accountID}

    buildResult, err := scopeddb.BuildInMemory(ctx, supportHistorySpec, scope)
    if err != nil {
        return scopeddb.BuildResult[SupportMeta]{}, err
    }

    if err := scopeddb.RegisterPrebuilt(reg, supportHistorySpec, buildResult.DB, supportHistorySpec.DefaultQuery); err != nil {
        _ = buildResult.Cleanup()
        return scopeddb.BuildResult[SupportMeta]{}, err
    }

    return *buildResult, nil
}
```

That is often the cleanest first implementation:

- resolve scope once
- build once
- register once
- reuse for all model queries in that inference run

## Common Mistakes

- Putting business logic into SQL generated by the model instead of into the materialized schema
- Forgetting to include a view or table in `AllowedObjects`
- Returning important scope information only in `Meta` when the model actually needs it in a table
- Using lazy registration when repeated queries should share one snapshot
- Forgetting to call `Cleanup`
- Omitting `ORDER BY` while also expecting stable row ordering

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---|---|---|
| `array schema items is not an object` from provider schema validation | Tool input used an unsupported array item schema | Use the current `QueryInput` shape with `Params []string` |
| `query references disallowed table/view` | Missing entry in `AllowedObjects` or wrong table name in SQL | Add the table/view to `AllowedObjects` or fix the query/examples |
| `only SELECT queries are allowed` | Model produced `DELETE`, `INSERT`, or another non-read query | Improve tool description and starter queries; keep validator strict |
| Tool keeps rebuilding the database | Using `NewLazyRegistrar` | Switch to `RegisterPrebuilt` if one request should share one snapshot |
| Model gives unstable answers for "latest" rows | No deterministic ordering | Set `RequireOrderBy: true` and include ordered starter queries |
| Large text fields make results noisy | Cell truncation too loose | Lower `MaxCellChars` in `QueryOptions` |

## Related Example

If you want a full runnable demo, see the Pinocchio example at `pinocchio/cmd/examples/scopeddb-tui-demo`.

That example shows:

- a concrete dataset spec with fake support-ticket data
- `BuildInMemory(...)`
- `Meta` usage in the UI
- `RegisterPrebuilt(...)`
- TUI rendering of SQL tool calls and tabular results

## See Also

- [Tools in Geppetto (Turn-based)](../topics/07-tools.md)
- [Add a New Tool](../playbooks/01-add-a-new-tool.md)
- [Build a Streaming Inference Command with Tool Calling](../tutorials/01-streaming-inference-with-tools.md)
- [Turns and Blocks in Geppetto](../topics/08-turns.md)
