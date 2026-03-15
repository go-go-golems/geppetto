---
Title: Scoped database tools extraction analysis, design, and implementation guide
Ticket: GP-33
Status: active
Topics:
    - geppetto
    - tooldb
    - sqlite
    - architecture
    - backend
    - refactor
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: Standard session runner integration point for tool registries
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Tool loop registry attachment and execution path
    - Path: geppetto/pkg/inference/tools/definition.go
      Note: Canonical Geppetto tool definition and schema generation surface
    - Path: geppetto/pkg/inference/tools/scopeddb/query.go
      Note: Implemented shared read-only query runner and SQLite authorizer policy
    - Path: geppetto/pkg/inference/tools/scopeddb/schema.go
      Note: Implemented generic DatasetSpec and BuildResult[Meta] API
    - Path: geppetto/pkg/inference/tools/scopeddb/tool.go
      Note: Implemented shared prebuilt and lazy registration helpers
    - Path: pinocchio/pkg/middlewares/sqlitetool/middleware.go
      Note: Related generic SQLite tool precedent and contrast point
    - Path: pinocchio/pkg/webchat/router.go
      Note: Application-level tool factory registration hook
    - Path: temporal-relationships/internal/extractor/entityhistory/query.go
      Note: Strong readonly SQL safety model to extract
    - Path: temporal-relationships/internal/extractor/entityhistory/spec.go
      Note: Concrete app-owned dataset spec over the shared package
    - Path: temporal-relationships/internal/extractor/gorunner/tools_persistence.go
      Note: Prebuilt scoped DB integration example
    - Path: temporal-relationships/internal/extractor/httpapi/run_chat_transport.go
      Note: |-
        Lazy request-scoped scoped DB integration example
        Run-chat lazy registrar integration
    - Path: temporal-relationships/internal/extractor/scopeddb/schema.go
      Note: Current repo-private scoped SQLite helper layer
ExternalSources: []
Summary: Evidence-backed design for extracting the scoped SQLite query tool pattern from temporal-relationships into a reusable Geppetto package.
LastUpdated: 2026-03-15T15:44:45.805465744-04:00
WhatFor: Give implementers a complete system map and file-level plan for moving scoped database tool mechanics into Geppetto without moving application-specific schema logic.
WhenToUse: Use when implementing GP-33 or when building a new Geppetto application that needs bounded read-only database tools backed by scoped SQLite snapshots.
---



# Scoped database tools extraction analysis, design, and implementation guide

## Executive Summary

`temporal-relationships` currently contains a strong reusable pattern for LLM-safe database tools, but the pattern is split across app-internal packages and repeated in multiple places. The pattern is: build a bounded SQLite database for one logical scope, expose a read-only query tool through the Geppetto tool registry, and let the model inspect only the rows and tables that are intentionally preloaded for that scope. The reusable parts exist today in `internal/extractor/scopeddb`, `transcripthistory`, `entityhistory`, and `runturnhistory`, but they are not packaged in a way other Geppetto applications can import or configure.

Geppetto already has the right lower-level primitives for a reusable solution. A tool is represented by `ToolDefinition` plus an executable function, registered in a `ToolRegistry`, attached to `context.Context`, advertised to providers from the live registry, and executed by the tool loop. Those primitives are visible in `geppetto/pkg/inference/tools/definition.go:14-95`, `geppetto/pkg/inference/tools/registry.go:8-125`, `geppetto/pkg/inference/tools/context.go:8-31`, and `geppetto/pkg/inference/toolloop/loop.go:113-125`. Pinocchio already provides an application-level hook for app-owned tool factories through `ToolRegistrar` and `webchat.Router.RegisterTool`, visible in `pinocchio/pkg/inference/runtime/engine.go:16-17` and `pinocchio/pkg/webchat/router.go:143-149`.

The proposed direction is to extract the generic pattern into a new Geppetto package, tentatively `geppetto/pkg/inference/tools/scopeddb`, and leave the domain-specific pieces inside application repositories. After the extraction, an app should define:

- a schema for its scoped SQLite snapshot,
- a scope request type,
- a preload/materialization function that copies bounded data into the snapshot,
- a description and starter queries for the tool,
- and, optionally, a lazy resolver that derives scope from request or session context.

That separation gives Geppetto a general-purpose tool-building package while keeping application schemas and preload logic local to the app.

## Implementation Update (2026-03-15)

The package proposed in this document was implemented as [`geppetto/pkg/inference/tools/scopeddb`](/home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/inference/tools/scopeddb). The extraction commit in `geppetto` is `f79f77b` (`Add reusable scoped SQLite tool package`). The first `temporal-relationships` migration commit is `ba7cfcb` (`Adopt geppetto scopeddb package`), followed by `eaad1be` (`Use lazy scopeddb registrars for run chat tools`).

The implemented public surface matches the proposal in the important places:

- `ToolDescription`, `ToolDefinitionSpec`, `DatasetSpec[Scope, Meta]`, and `BuildResult[Meta]` exist in the shared package.
- `Meta` stayed in the build result and is preserved by both `BuildInMemory` and `BuildFile`.
- Read-only query execution, object allowlisting, description rendering, prebuilt registration, and lazy registration are all shared in Geppetto.

The main implementation deviations from the earlier proposal are:

- `temporal-relationships` keeps thin package-local wrappers such as [`entityhistory/spec.go`](/home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/entityhistory/spec.go), [`transcripthistory/spec.go`](/home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/transcripthistory/spec.go), and [`runturnhistory/spec.go`](/home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/runturnhistory/spec.go) so each app-owned schema can define its own materializer without moving preload SQL into Geppetto.
- `gorunner` continues calling package-local `RegisterQueryTool(...)`, but those wrappers now delegate to shared `RegisterPrebuilt(...)`, so the duplication moved out of the runtime call site without changing the caller contract.
- `httpapi/run_chat_transport.go` now registers run-chat tools through the shared lazy registrar path, but the existing direct query helpers were kept because tests already exercise them directly and they remain useful as explicit integration seams.

## Problem Statement

The current implementation solves a real product need in `temporal-relationships`: the model must query bounded historical context without direct access to the full application database. The app therefore builds tool-facing SQLite snapshots and exposes them as query tools. The problem is not that the pattern is wrong. The problem is that the pattern is trapped inside one app, duplicated across three tool packages, and only partially aligned with the reusable abstractions already present in Geppetto and Pinocchio.

For a new intern, the main confusion comes from the fact that there are multiple layers involved:

- Geppetto owns generic tool execution, tool advertisement, tool registry transport, and the tool loop.
- Pinocchio/webchat owns app-level router and session wiring for chat-style applications.
- `temporal-relationships` owns the domain-specific schema, scope rules, preload SQL, and session-specific tool configuration.

Today those layers are interleaved instead of clearly separated. The result is:

- no reusable Geppetto package for scoped database tools,
- duplicated query-runner and SQL-safety code,
- app-specific tool registration logic repeated in both batch and webchat paths,
- no documented, ergonomic recipe for other Geppetto applications to define their own scoped databases.

This ticket is therefore a design and migration ticket. It is not implementing the package yet. Its purpose is to create a very explicit blueprint for the extraction.

### In Scope

- Map the current scoped-db pattern in `temporal-relationships`.
- Map the Geppetto and Pinocchio runtime/tool abstractions that the extracted package must fit into.
- Propose a reusable package API and package layout.
- Provide a detailed, file-level implementation plan.
- Provide a migration map from the current `temporal-relationships` packages to the proposed extracted package.
- Record the work in a ticket and upload the document bundle to reMarkable.

### Out of Scope

- Implementing the new package in code.
- Changing provider APIs or the Geppetto tool loop contract.
- Introducing write-capable scoped databases as a first-pass feature.
- Making Pinocchio session-storage generic in the same ticket.

## Proposed Solution

Create a new package in Geppetto, tentatively:

```text
geppetto/pkg/inference/tools/scopeddb
```

The package should own the reusable mechanics of the pattern:

1. SQLite snapshot bootstrapping.
2. Schema application.
3. Shared query limits and read-only result formatting.
4. Strong SQL-safety policy for read-only query tools.
5. Description building using allowed objects and starter queries.
6. Registration helpers for:
   - prebuilt scoped databases,
   - lazily built scoped databases resolved from context.

The package should not own application-specific schemas or preload SQL. Those belong in the application repo.

### High-Level Architecture

```text
Application persistent DB / stores
            |
            v
   app-defined scope resolver
            |
            v
   app-defined materializer/preloader
            |
            v
  Geppetto scopeddb package
  - open sqlite
  - apply schema
  - enforce readonly query policy
  - build tool definition
  - register tool
            |
            v
  Geppetto ToolRegistry / toolloop
            |
            v
       provider tool call
            |
            v
     query runner over scoped DB
```

### Recommended Package Responsibility Split

The new Geppetto package should expose a small number of concepts:

- `DatasetSpec`
  - Static definition of a scoped dataset.
  - Owns snapshot schema, allowed objects, the tool definition presentation, default query options, and the materializer callback.
- `Materializer`
  - App callback that fills the scoped database from source state.
  - Receives a destination `*sql.DB` plus typed scope input.
- `ScopeResolver`
  - Optional app callback that derives a typed scope from `context.Context`.
  - Used for lazy registration in request/session-oriented apps.
- `QueryRunner`
  - Shared read-only query executor with bounds and safety checks.
- `Registrar`
  - Helper that turns a dataset spec into a `ToolDefinition` registration function.

### Tool Definition Schema Versus Snapshot Schema

There are three different schemas in play, and the package should name them clearly.

1. **Tool definition schema**
   - The Geppetto `ToolDefinition` itself, defined in `geppetto/pkg/inference/tools/definition.go:14-23`.
   - Fields: name, description, parameters, function, examples, tags, version.
2. **Tool call input schema**
   - The JSON schema generated from `QueryInput`.
   - This is the provider-visible parameters contract.
3. **Snapshot database schema**
   - The SQLite `SchemaSQL` used to build the scoped database itself.

Those are related, but they should not be collapsed into one field. The right split is:

- group model-facing tool metadata together,
- keep the SQLite snapshot schema separate,
- keep the query input contract fixed and generic.

### Recommended API Shape

The following API is intentionally pseudo-Go. The exact names can change during implementation. The important part is the separation of concerns.

```go
package scopeddb

type QueryOptions struct {
    MaxRows        int
    MaxColumns     int
    MaxCellChars   int
    Timeout        time.Duration
    RequireOrderBy bool
}

type QueryInput struct {
    SQL    string `json:"sql"`
    Params []any  `json:"params,omitempty"`
}

type QueryOutput struct {
    Columns   []string         `json:"columns"`
    Rows      []map[string]any `json:"rows"`
    Count     int              `json:"count"`
    Truncated bool             `json:"truncated,omitempty"`
    Error     string           `json:"error,omitempty"`
}

type ToolDescription struct {
    Summary        string
    StarterQueries []string
    Notes          []string
}

type ToolDefinitionSpec struct {
    Name        string
    Description ToolDescription
    Tags        []string
    Version     string
}

type DatasetSpec[Scope any, Meta any] struct {
    InMemoryPrefix string
    SchemaLabel    string
    SchemaSQL      string
    AllowedObjects []string
    Tool           ToolDefinitionSpec
    DefaultQuery   QueryOptions
    Materialize    func(ctx context.Context, dst *sql.DB, scope Scope) (Meta, error)
}

type ScopeResolver[Scope any] func(ctx context.Context) (Scope, error)

type BuildResult[Meta any] struct {
    DB      *sql.DB
    Meta    Meta
    Cleanup func() error
}

func BuildInMemory[Scope any, Meta any](
    ctx context.Context,
    spec DatasetSpec[Scope, Meta],
    scope Scope,
) (*BuildResult[Meta], error)

func BuildFile[Scope any, Meta any](
    ctx context.Context,
    path string,
    spec DatasetSpec[Scope, Meta],
    scope Scope,
) (*BuildResult[Meta], error)

func RegisterPrebuilt[Scope any, Meta any](
    reg tools.ToolRegistry,
    spec DatasetSpec[Scope, Meta],
    db *sql.DB,
    opts QueryOptions,
) error

func NewLazyRegistrar[Scope any, Meta any](
    spec DatasetSpec[Scope, Meta],
    resolve ScopeResolver[Scope],
    opts QueryOptions,
) func(reg tools.ToolRegistry) error
```

### Two Supported Usage Modes

The extracted package should support two operational modes because the current system already uses both.

#### Mode A: Prebuilt scoped DB

This matches `temporal-relationships/internal/extractor/gorunner/tools_persistence.go:30-124`.

```text
runner startup
  -> build scope once
  -> materialize in-memory sqlite once
  -> register tool against that db
  -> reuse db for all tool calls in the run
  -> close db at run end
```

This is the right mode for batch or single-run workflows where the scope is stable during the loop.

#### Mode B: Lazy scoped DB from context

This matches `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go:227-398`.

```text
router registers tool factory
  -> each tool call resolves session/request scope from context
  -> build scoped db on demand
  -> run query
  -> close db immediately
```

This is the right mode for webchat and request-scoped systems where the tool factory cannot know the concrete scope until a specific request or session is active.

### Example: What an App Would Write After the Extraction

An application should only need to provide schema, scope, and preload logic.

```go
type ReviewScope struct {
    ProductID string
    ReleaseID string
}

type ReviewMeta struct {
    ReviewCount int
}

var reviewHistorySpec = scopeddb.DatasetSpec[ReviewScope, ReviewMeta]{
    InMemoryPrefix: "review_history",
    SchemaLabel:    "review history scoped db schema",
    SchemaSQL:      reviewHistorySchemaSQL,
    AllowedObjects: []string{"scope", "products", "reviews", "review_tags"},
    Tool: scopeddb.ToolDefinitionSpec{
        Name: "query_review_history",
        Description: scopeddb.ToolDescription{
            Summary: "Query bounded review history for the active product release.",
            StarterQueries: []string{
                "SELECT * FROM scope",
                "SELECT review_id, reviewer, rating, summary FROM reviews ORDER BY created_at_ms DESC LIMIT 20",
            },
        },
        Tags:    []string{"sqlite", "reviews"},
        Version: "v1",
    },
    DefaultQuery: scopeddb.DefaultQueryOptions(),
    Materialize: func(ctx context.Context, dst *sql.DB, scope ReviewScope) (ReviewMeta, error) {
        // App-owned code: copy bounded rows into dst.
        return ReviewMeta{ReviewCount: 42}, nil
    },
}
```

#### Prebuilt registration

```go
handle, err := scopeddb.BuildInMemory(ctx, reviewHistorySpec, ReviewScope{
    ProductID: productID,
    ReleaseID: releaseID,
})
if err != nil {
    return err
}
defer handle.Cleanup()

reg := tools.NewInMemoryToolRegistry()
if err := scopeddb.RegisterPrebuilt(reg, reviewHistorySpec, handle.DB, reviewHistorySpec.DefaultQuery); err != nil {
    return err
}
```

#### Lazy webchat registration

```go
router.RegisterTool(
    "query_review_history",
    infruntime.ToolRegistrar(
        scopeddb.NewLazyRegistrar(
            reviewHistorySpec,
            func(ctx context.Context) (ReviewScope, error) {
                return resolveReviewScopeFromChatSession(ctx)
            },
            reviewHistorySpec.DefaultQuery,
        ),
    ),
)
```

That is the ergonomic target.

## Design Decisions

### 1. Put the extracted package in Geppetto, not Pinocchio

This is the correct home because the reusable abstraction is a tool-building abstraction, not a webchat abstraction.

Evidence:

- `ToolDefinition`, `ToolRegistry`, `NewToolFromFunc`, and tool execution live in `geppetto/pkg/inference/tools/*`.
- Tool advertisement from the live registry is handled in `geppetto/pkg/inference/tools/advertisement.go:10-35`.
- The tool loop consumes the registry and snapshots tool definitions in `geppetto/pkg/inference/toolloop/loop.go:113-125`.
- The standard session runner decides whether to run a single pass or tool loop based on whether a registry exists in `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:45-56` and `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:191-218`.

Pinocchio should remain the place where chat/web server applications adapt application state into tool registrars, but the core scoped-db abstraction belongs one layer lower.

### 2. Keep domain schemas and preload SQL in the application repo

The current domain packages prove that the schemas are application-specific:

- transcript history exposes `scope`, `sessions`, and `utterances` in `temporal-relationships/internal/extractor/transcripthistory/schema.go:11-16`,
- entity history exposes a much richer graph/history surface in `temporal-relationships/internal/extractor/entityhistory/schema.go:11-28`,
- run-turn history builds a conversation-oriented schema from turn snapshots and artifact catalogs in `temporal-relationships/internal/extractor/runturnhistory/preload.go:21-200`.

That domain logic should not be moved to Geppetto. Geppetto should only own the reusable mechanics around it.

### 3. Make read-only, bounded query execution the default

The safest reusable default is the stronger query model already used by `entityhistory` and `runturnhistory`.

Evidence:

- `entityhistory` validates SQL, strips literals/comments for semicolon detection, installs a SQLite authorizer, and checks `stmt.Readonly()` in `temporal-relationships/internal/extractor/entityhistory/query.go:48-76` and `temporal-relationships/internal/extractor/entityhistory/query.go:256-337`.
- `runturnhistory` repeats the same safety model in `temporal-relationships/internal/extractor/runturnhistory/query.go:48-76` and `temporal-relationships/internal/extractor/runturnhistory/query.go:256-337`.
- `transcripthistory` currently uses a lighter validator without authorizer-based enforcement in `temporal-relationships/internal/extractor/transcripthistory/query.go:106-159`.

The extracted package should unify on the stronger model and let lightweight validation be an opt-out, not the baseline.

### 4. Support both prebuilt and lazy registration from day one

Both modes already exist in production code:

- prebuilt-once-per-run in `temporal-relationships/internal/extractor/gorunner/tools_persistence.go:30-124`,
- lazy-per-call in `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go:275-398`.

If the package only supported one mode, one of the existing users would still need custom infrastructure and the extraction would not really be complete.

### 5. Do not fold app-specific session-scope persistence into the first package cut

`temporal-relationships` persists chat tool setups using app-specific tables and tool keys in:

- `temporal-relationships/internal/extractor/runchat/session_store.go:13-35`,
- `temporal-relationships/internal/extractor/runchat/session_store.go:85-132`,
- `temporal-relationships/internal/extractor/runchat/session_store.go:410-489`,
- `temporal-relationships/internal/extractor/httpapi/run_chat_handlers.go:362-500`.

That persistence model reflects one product’s chat UX. It should remain app-owned in the first extraction. The reusable package should accept a resolved scope, not define how UIs store that scope.

## Current-State Analysis

### 1. Geppetto tool model

The Geppetto tool model is already solid and should not be replaced.

- `geppetto/pkg/inference/tools/definition.go:14-23` defines the serializable tool metadata plus the executable `ToolFunc`.
- `geppetto/pkg/inference/tools/definition.go:34-95` shows that `NewToolFromFunc` is the canonical way to turn a Go function into a tool definition with generated JSON schema.
- `geppetto/pkg/inference/tools/registry.go:8-18` defines the registry contract; `registry.go:20-125` implements the in-memory registry used by examples and apps.
- `geppetto/pkg/inference/tools/context.go:8-31` carries the live runtime registry via `context.Context`.
- `geppetto/pkg/inference/tools/advertisement.go:10-35` converts that live registry into provider-facing tool definitions.
- `geppetto/pkg/inference/tools/config.go:5-33` and `config.go:100-129` define runtime policy such as allowed tools and execution limits.

This means the new scoped-db package should produce ordinary Geppetto tools. It should not invent a new side channel for tool exposure.

### 2. Tool loop and session runner integration

The execution path matters because the extracted package must fit into it with minimal glue.

- `geppetto/pkg/inference/toolloop/loop.go:113-125` attaches the registry to context and persists tool config plus tool definitions on the turn.
- `geppetto/pkg/inference/toolloop/loop.go:137-168` extracts pending tool calls, executes them through the registry, and appends tool results.
- `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:158-240` is the standard runner that applications use to invoke either a single engine pass or the tool loop.
- `pinocchio/pkg/inference/runtime/engine.go:16-17` defines `ToolRegistrar`, which is the app-friendly signature for “register this tool into a registry”.
- `pinocchio/pkg/webchat/router.go:74-75`, `router.go:123-145`, and `pinocchio/pkg/webchat/llm_loop_runner.go:109-129` show how webchat apps keep a map of tool factories, rebuild a registry for each session start, and filter by allowed tools.

This tells us the extracted package should expose registration helpers that are compatible with both:

- direct registry mutation in Geppetto-only programs,
- `ToolRegistrar` usage in Pinocchio/webchat apps.

### 3. The current reusable kernel hidden inside temporal-relationships

`temporal-relationships/internal/extractor/scopeddb` already contains a generic core:

- schema bootstrap and in-memory/file opening in `schema.go:14-80`,
- schema statement splitting in `schema.go:45-55`,
- query limits and cell normalization in `query.go:5-59`,
- string normalization and helper ids in `helpers.go:10-43`,
- trailing semicolon normalization in `sql.go:5-12`.

The major problem is location, not just code quality. Because it sits under `internal/extractor/scopeddb`, no other repository can import it. That alone is enough reason to extract it.

### 4. The current domain package structure

Each scoped tool in `temporal-relationships` uses the same broad layout:

```text
<domain>/
  schema.go   -> embed schema and allowed objects
  preload.go  -> populate scoped sqlite db from source state
  query.go    -> enforce read-only queries and return typed row output
  tool.go     -> register geppetto tool
```

Examples:

- `transcripthistory/schema.go:11-30`, `preload.go:13-189`, `query.go:13-178`, `tool.go:12-49`
- `entityhistory/schema.go:11-43`, `preload.go`, `query.go:13-337`, `tool.go:12-50`
- `runturnhistory/schema.go`, `preload.go:21-200`, `query.go:13-337`, `tool.go:7-18`

The package shape is already a strong hint for the eventual extraction boundary: schema/query/tool mechanics are mostly generic; preload logic is domain-specific.

### 5. Concrete duplication

There is significant duplicated code today.

- `entityhistory/query.go` is 337 lines.
- `runturnhistory/query.go` is 337 lines.
- A direct `diff -u` shows they are effectively the same file except for package name and domain allowlist usage.
- `transcripthistory/tool.go` and `entityhistory/tool.go` are structurally the same registration wrapper around `NewQueryRunner` and `NewToolFromFunc`.
- All three packages carry their own `AllowedObjects`, schema embedding, tool description assembly, and default query-option plumbing.

The line counts captured during this investigation were:

```text
scopeddb/*.go                 204 lines
transcripthistory/*.go       1037 lines
entityhistory/*.go           1957 lines
runturnhistory/*.go          1562 lines
```

Not all of that is duplication. The preload files are legitimately domain-heavy. The duplication worth extracting is concentrated in the schema/query/tool layers.

### 6. Current runtime wiring in temporal-relationships

There are two main integration patterns today.

#### Extraction runner path

`temporal-relationships/internal/extractor/gorunner/tools_persistence.go:30-124`:

- checks config toggles,
- opens an in-memory scoped DB for each enabled tool,
- preloads it,
- registers the query tool in a new registry,
- returns cleanup functions.

This is the best example of “build once, reuse for the run”.

#### Run-chat path

`temporal-relationships/internal/extractor/httpapi/run_chat_transport.go:227-398`:

- registers tool factories on the webchat router,
- resolves the active session from tool-call context,
- reopens the main app DB,
- rebuilds the scoped DB on demand,
- runs the query,
- closes the scoped DB.

This is the best example of “lazy, request-scoped materialization”.

### 7. Session-scoped tool selection in temporal-relationships

The app also persists which scopes are enabled for a run-chat session.

- Tool keys are hard-coded in `runchat/session_store.go:13-17`.
- Tool setup records are persisted in chat-specific tables in `runchat/session_store.go:100-132`.
- Validation rules are hard-coded per tool key in `runchat/session_store.go:410-489`.
- Request normalization for chat session creation is hard-coded in `httpapi/run_chat_handlers.go:362-500`.

This is useful context for the migration, but it is not the right extraction boundary for Geppetto core.

### 8. Existing related precedent: Pinocchio SQLite tool middleware

There is already a generic SQLite-oriented tool in `pinocchio/pkg/middlewares/sqlitetool/middleware.go:30-210`. It proves that the stack already supports generic database-backed tools. It also clarifies what the new package should not be.

That middleware:

- attaches or opens one live SQLite database,
- precomputes a schema-heavy description,
- registers a `sql_query` tool,
- returns text output,
- and can be used in read-only or read-write modes.

This is documented in `geppetto/pkg/inference/middleware/sqlitetool/sqlite-tool-middleware.md:17-26`.

It is related, but it is not the same pattern as the temporal scoped DB tools because:

- it is middleware-oriented rather than dataset-spec-oriented,
- it attaches a live DB rather than building a bounded snapshot,
- it returns text rather than the typed rows/columns structure used by temporal tools,
- it is intentionally more generic and looser than the bounded-snapshot model needed here.

The right relationship is: keep `sqlitetool` for “attach a DB and let the agent query it”, add `scopeddb` for “build a bounded read-only DB for a specific scope and register a structured query tool”.

## What The Pattern Really Contains

The phrase “scoped db tool pattern” is easy to underspecify. In practice it contains seven separate responsibilities:

1. **Scope selection**
   - Choose the logical slice of source data to expose.
   - Example: prior transcript sessions, selected extraction runs, or one anchor run’s turns.
2. **Snapshot schema**
   - Define the tables and views the model is allowed to query.
3. **Materialization**
   - Copy bounded data into the snapshot DB.
4. **Query policy**
   - Reject writes, multi-statement SQL, and disallowed object access.
5. **Tool description**
   - Tell the model what the tool is, what tables exist, and which starter queries are useful.
6. **Tool registration**
   - Register a Geppetto tool that calls the query runner.
7. **Lifecycle**
   - Build once and reuse, or build lazily and close per call.

If any one of these is missing, the pattern is incomplete. The extracted package needs to make all seven explicit.

## Gap Analysis

### Gap 1: The reusable kernel is not reusable

Observed state:

- generic helpers live in `temporal-relationships/internal/extractor/scopeddb/*`.

Why this matters:

- anything under `internal/` is repo-private,
- another Geppetto app cannot import the package,
- the next app will copy the pattern instead of reusing it.

### Gap 2: SQL-safety logic is duplicated and inconsistent

Observed state:

- `entityhistory/query.go` and `runturnhistory/query.go` carry the same strong SQLite safety logic.
- `transcripthistory/query.go` uses a lighter regex/object-reference strategy.

Why this matters:

- bug fixes will need three edits,
- safety improvements can drift,
- intern readers cannot tell which behavior is canonical.

### Gap 3: Tool registration is domain-packaged instead of framework-packaged

Observed state:

- each domain package has its own `RegisterQueryTool` wrapper,
- webchat registration recreates that logic inline in `run_chat_transport.go:232-272`.

Why this matters:

- Geppetto already has a canonical tool definition path via `NewToolFromFunc`,
- the “turn this scoped DB into a tool” step should be reusable and centralized.

### Gap 4: There is no first-class app recipe for defining custom scoped databases

Observed state:

- the Geppetto “Add a New Tool” playbook covers ordinary tools in `geppetto/pkg/doc/playbooks/01-add-a-new-tool.md:89-170`,
- but there is no equivalent “add a scoped database tool” guide.

Why this matters:

- app authors have to reverse-engineer `temporal-relationships`,
- the extraction would still be hard to adopt if it ships without docs and examples.

### Gap 5: App-owned scope persistence is currently hard-coded to one app

Observed state:

- `temporal-relationships` hardcodes `entity_db`, `transcript_db`, and `run_turns_db`.

Why this matters:

- this is useful evidence for the package design,
- but it cannot become Geppetto core as-is because it encodes product-specific semantics.

## Proposed Package Layout

The package should be small, explicit, and boring.

```text
geppetto/pkg/inference/tools/scopeddb/
  build.go          // open sqlite db and apply schema
  query.go          // QueryInput, QueryOutput, QueryOptions, QueryRunner
  readonly.go       // readonly validation, authorizer, prepared statement checks
  spec.go           // DatasetSpec and related types
  description.go    // build descriptions from allowed objects + starter queries
  registrar.go      // RegisterPrebuilt and NewLazyRegistrar
  helpers.go        // string normalization, ids, object sorting
  testing.go        // test helpers for fixture materialization
```

### Why This Split

- `build.go` isolates database opening and schema application.
- `query.go` isolates the public tool contract.
- `readonly.go` isolates the riskiest logic and makes it unit-testable.
- `spec.go` defines the surface application authors use.
- `registrar.go` is the bridge into Geppetto tools and Pinocchio tool factories.
- `description.go` avoids repeating description concatenation logic across apps.

## Detailed API Reference

### DatasetSpec

`DatasetSpec` is the top-level app-authored contract.

Recommended fields:

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

Field guidance:

- `InMemoryPrefix`
  - Stable prefix used when generating unique in-memory SQLite DSNs.
- `SchemaLabel`
  - Used in error messages when schema application fails.
- `SchemaSQL`
  - Application-owned schema for the snapshot.
- `AllowedObjects`
  - Read allowlist for the query policy and description builder.
- `Tool`
  - The actual tool definition metadata that will be converted into a Geppetto `ToolDefinition`.
- `DefaultQuery`
  - Baseline query limits for the dataset.
- `Materialize`
  - The only mandatory app callback. It fills the destination DB.

### ToolDefinitionSpec

`ToolDefinitionSpec` is the app-facing subset of the eventual Geppetto `ToolDefinition`.

```go
type ToolDefinitionSpec struct {
    Name        string
    Description ToolDescription
    Tags        []string
    Version     string
}
```

Recommended behavior:

- `Name` becomes the Geppetto tool name.
- `Description` is rendered into the final provider-visible description string.
- `Tags` and `Version` are copied into the registered `ToolDefinition`.

### ToolDescription

`ToolDescription` groups the human/model-facing description fields together without mixing them with snapshot schema or query execution configuration.

```go
type ToolDescription struct {
    Summary        string
    StarterQueries []string
    Notes          []string
}
```

This grouping is deliberate:

- `Summary` is the core prose explanation of what the tool is for.
- `StarterQueries` are example queries for the model.
- `Notes` are optional behavior hints such as “prefer bind params” or “use ORDER BY”.

The SQLite `SchemaSQL` stays outside this struct because it is snapshot structure, not description prose.

### BuildResult

`BuildResult[Meta]` should be kept in the public API. The reason is that the current app code already produces and uses preload metadata such as counts, truncation flags, and resolved scope identifiers.

```go
type BuildResult[Meta any] struct {
    DB      *sql.DB
    Meta    Meta
    Cleanup func() error
}
```

Examples of useful `Meta` values:

- preload counts,
- resolved run ids,
- scope start/end ids,
- truncation indicators,
- cache bookkeeping if a future caller needs it.

The important rule is that `Meta` is optional for the tool runtime itself. It exists for the caller that built the scoped DB, not for provider-facing tool execution.

### QueryRunner

`QueryRunner` should centralize the query tool behavior now spread across three packages.

Recommended responsibilities:

- normalize a trailing semicolon,
- reject multi-statement SQL,
- enforce `SELECT` / `WITH` only,
- optionally require `ORDER BY`,
- enforce allowed-object reads through SQLite authorizer,
- ensure the prepared statement is read-only,
- enforce row, column, and cell-size limits,
- return a structured `QueryOutput`.

### RegisterPrebuilt

Use when the caller already has a built scoped DB and a clear cleanup lifecycle.

Recommended behavior:

- instantiate a shared `QueryRunner`,
- create a tool function with `tools.NewToolFromFunc`,
- build the description using `DatasetSpec`,
- register the tool in the provided registry.

### NewLazyRegistrar

Use when the scope is only available from request/session context.

Recommended behavior:

- return a `func(reg tools.ToolRegistry) error`,
- register a tool whose executor:
  - resolves scope from context,
  - builds an in-memory scoped DB,
  - runs the shared query runner,
  - closes the DB before returning.

This mirrors the current run-chat pattern without duplicating the whole factory implementation in each app.

## Diagrams

### Current State

```text
temporal-relationships
  |
  +-- internal/extractor/scopeddb
  |     generic helpers but private to repo
  |
  +-- transcripthistory
  |     schema + preload + query + tool registration
  |
  +-- entityhistory
  |     schema + preload + query + tool registration
  |
  +-- runturnhistory
  |     schema + preload + query + tool registration
  |
  +-- gorunner/tools_persistence.go
  |     prebuilt scoped registry
  |
  +-- httpapi/run_chat_transport.go
        lazy scoped tool factories
```

### Proposed State

```text
geppetto/pkg/inference/tools/scopeddb
  |
  +-- generic schema/query/registration mechanics
  |
  +-- used by applications
         |
         +-- temporal-relationships/transcripthistory spec
         +-- temporal-relationships/entityhistory spec
         +-- temporal-relationships/runturnhistory spec
         +-- future app-specific scoped datasets
```

### Prebuilt Run Flow

```text
app runtime startup
  -> resolve stable scope
  -> scopeddb.BuildInMemory(...)
  -> scopeddb.RegisterPrebuilt(...)
  -> toolloop executes calls against resident snapshot
  -> cleanup on run end
```

### Lazy Request Flow

```text
router registers lazy registrar
  -> tool call arrives
  -> resolve scope from context
  -> BuildInMemory(...)
  -> QueryRunner.Run(...)
  -> close snapshot db
  -> return rows
```

## Implementation Plan

### Phase 1: Extract the generic kernel into Geppetto

Goal: move the reusable mechanics out of `temporal-relationships/internal/extractor/scopeddb`.

Files to add in Geppetto:

- `geppetto/pkg/inference/tools/scopeddb/build.go`
- `geppetto/pkg/inference/tools/scopeddb/query.go`
- `geppetto/pkg/inference/tools/scopeddb/readonly.go`
- `geppetto/pkg/inference/tools/scopeddb/spec.go`
- `geppetto/pkg/inference/tools/scopeddb/description.go`
- `geppetto/pkg/inference/tools/scopeddb/registrar.go`
- `geppetto/pkg/inference/tools/scopeddb/helpers.go`

Files to use as source material:

- `temporal-relationships/internal/extractor/scopeddb/schema.go`
- `temporal-relationships/internal/extractor/scopeddb/query.go`
- `temporal-relationships/internal/extractor/scopeddb/helpers.go`
- `temporal-relationships/internal/extractor/scopeddb/sql.go`
- `temporal-relationships/internal/extractor/entityhistory/query.go`
- `temporal-relationships/internal/extractor/runturnhistory/query.go`

Phase 1 acceptance criteria:

- new package can open in-memory/file-backed SQLite snapshots,
- new package exposes shared query/result types,
- new package has strong default read-only enforcement,
- package tests port the helper and safety tests from the app repo.

### Phase 2: Introduce reusable registration helpers

Goal: stop repeating `NewQueryRunner` plus `NewToolFromFunc` wrappers.

Files to update in Geppetto:

- `geppetto/pkg/inference/tools/scopeddb/registrar.go`
- add a new doc page or playbook under `geppetto/pkg/doc/playbooks/`

Expected deliverables:

- `RegisterPrebuilt(...)`
- `NewLazyRegistrar(...)`
- description builder that consistently includes:
  - allowed tables/views,
  - bind parameter guidance,
  - starter queries,
  - deterministic-ordering hint when `RequireOrderBy` is enabled.

### Phase 3: Convert temporal-relationships to thin dataset specs

Goal: leave only app-specific schema and materialization logic inside the app repo.

Files to refactor in `temporal-relationships`:

- `internal/extractor/transcripthistory/schema.go`
- `internal/extractor/transcripthistory/preload.go`
- `internal/extractor/transcripthistory/tool.go`
- `internal/extractor/entityhistory/schema.go`
- `internal/extractor/entityhistory/preload.go`
- `internal/extractor/entityhistory/tool.go`
- `internal/extractor/runturnhistory/schema.go`
- `internal/extractor/runturnhistory/preload.go`
- `internal/extractor/runturnhistory/tool.go`

Expected refactor shape:

- keep embedded schema SQL and preload functions in the domain package,
- replace package-local query runner and registration code with `scopeddb`,
- optionally replace `tool.go` with a `spec.go` that exports one dataset spec and one convenience registrar.

### Phase 4: Update runtime integration call sites

Goal: remove runtime-specific duplication in the app’s wiring code.

Files to refactor:

- `temporal-relationships/internal/extractor/gorunner/tools_persistence.go`
- `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go`

Desired result:

- gorunner uses `BuildInMemory + RegisterPrebuilt`,
- run-chat uses `NewLazyRegistrar` with app-specific context resolution,
- query options remain app-configurable from current config sources.

### Phase 5: Add documentation and one generic example

Goal: make the package actually adoptable by a new engineer.

Files to add:

- `geppetto/pkg/doc/playbooks/XX-add-a-scoped-database-tool.md`
- `geppetto/cmd/examples/` sample program showing one trivial scoped dataset

Example scope recommendation:

- a tiny “books + reviews” snapshot example,
- one prebuilt usage sample,
- one lazy registrar sample.

## Pseudocode For The Migration

### A. Shared Geppetto package

```go
func RegisterPrebuilt[Scope any, Meta any](
    reg tools.ToolRegistry,
    spec DatasetSpec[Scope, Meta],
    db *sql.DB,
    opts QueryOptions,
) error {
    runner := NewQueryRunner(db, spec.AllowedObjects, opts)
    desc := BuildDescription(spec.Tool.Description, spec.AllowedObjects, opts)

    def, err := tools.NewToolFromFunc(
        spec.Tool.Name,
        desc,
        func(ctx context.Context, in QueryInput) (QueryOutput, error) {
            return runner.Run(ctx, in)
        },
    )
    if err != nil {
        return err
    }
    return reg.RegisterTool(def.Name, *def)
}
```

### B. Temporal gorunner integration

```go
scope := entityhistory.Scope{
    RunID: currentRunID,
    SessionTranscriptID: sessionTranscriptID,
    SourceRunIDs: ...,
}

handle, err := scopeddb.BuildInMemory(ctx, entityhistory.Spec(), scope)
if err != nil {
    return err
}
defer handle.Cleanup()

if err := scopeddb.RegisterPrebuilt(registry, entityhistory.Spec(), handle.DB, queryOpts); err != nil {
    return err
}
```

### C. Temporal run-chat integration

```go
router.RegisterTool(
    "query_extraction_history",
    infruntime.ToolRegistrar(
        scopeddb.NewLazyRegistrar(
            entityhistory.Spec(),
            func(ctx context.Context) (entityhistory.Scope, error) {
                resolved, err := resolveRunChatSessionFromToolContext(ctx, opts)
                if err != nil {
                    return entityhistory.Scope{}, err
                }
                return entityhistory.ScopeFromResolvedSession(resolved), nil
            },
            entityhistory.DefaultQueryOptions(),
        ),
    ),
)
```

## Test Strategy

The extracted package should ship with a layered test suite.

### Unit Tests

Port and adapt the reusable tests from the current app:

- schema/open helpers from `temporal-relationships/internal/extractor/scopeddb/helpers_test.go`
- query safety and readonly enforcement from:
  - `entityhistory/query_test.go`
  - `runturnhistory/query_test.go`
  - selected transcript tests from `transcripthistory/query_test.go`

Unit test focus:

- schema application,
- file/in-memory db creation,
- semicolon handling,
- comment/literal stripping,
- disallowed writes,
- disallowed tables/views,
- readonly prepared statement checks,
- row truncation and column caps,
- cell normalization.

### Integration Tests

For each application dataset spec:

- build an actual scoped DB from a seeded persistent DB,
- register the tool,
- execute the query tool through `ToolFunc.ExecuteWithContext`,
- assert returned columns/rows.

### Migration Regression Tests

For `temporal-relationships`, keep or add “before/after parity” tests:

- transcript history returns the same rows as before,
- entity history still blocks writes and out-of-scope tables,
- run-turn history still materializes turn ordering and cross-reference tables correctly,
- gorunner and run-chat still register the same tool names.

### Documentation Validation

Add one example or playbook smoke check so future refactors do not silently break the package’s onboarding story.

## Risks, Tradeoffs, And Open Questions

### Risk 1: Over-generalizing the first version

If the package tries to support every possible DB backend, write mode, rich caching layer, and UI persistence abstraction at once, the extraction will stall. The first cut should stay focused on bounded SQLite snapshots for read-only query tools.

### Risk 2: Accidentally coupling Geppetto to Pinocchio

The Geppetto package must not import Pinocchio. The lazy registrar should return a plain registry-mutation function with the same signature shape that Pinocchio can wrap as `ToolRegistrar`.

### Risk 3: Breaking current query behavior during unification

The stronger query safety model should become the default, but the migration needs regression tests so transcript queries do not accidentally lose useful behavior.

### Risk 4: Hiding lifecycle cost

Lazy registration rebuilds scoped DBs per call. That is acceptable for correctness and simplicity, but the package documentation should say clearly that apps with high call frequency may want a caller-owned cache outside the first package cut.

### Open Question 1

Should a second-phase cache helper live in Geppetto or stay application-owned?

Recommendation:

- keep caching out of the first implementation,
- revisit only after two applications need the same lifecycle policy.

### Open Question 2

Should the package expose typed `Meta` to tool descriptions or logging?

Recommendation:

- yes for return values from `BuildInMemory`,
- no as part of the core query-tool contract.

### Open Question 3

Should Geppetto also own a generic persistent “scope manifest” abstraction?

Recommendation:

- not in this extraction,
- current evidence shows that scope persistence is heavily app-specific.

## Alternatives Considered

### Alternative A: Leave the code in temporal-relationships and copy it later

Rejected because it preserves the main problem: other apps still cannot import the pattern.

### Alternative B: Move the entire temporal domain packages into Geppetto

Rejected because the schemas and preload logic are domain-specific, not framework-level.

### Alternative C: Reuse the existing Pinocchio `sqlitetool` middleware as the only solution

Rejected because it solves a different problem. It is a live-DB middleware tool, not a scoped snapshot toolkit. It also returns textual tabular output and is middleware-centered instead of dataset-spec-centered.

### Alternative D: Only support lazy registration

Rejected because the gorunner path already demonstrates a valid build-once lifecycle with better reuse inside one run.

## File-Level Migration Map

### Current file -> future role

- `temporal-relationships/internal/extractor/scopeddb/schema.go`
  - becomes Geppetto `build.go` foundation
- `temporal-relationships/internal/extractor/scopeddb/query.go`
  - becomes Geppetto `query.go` foundation
- `temporal-relationships/internal/extractor/scopeddb/helpers.go`
  - becomes Geppetto `helpers.go`
- `temporal-relationships/internal/extractor/scopeddb/sql.go`
  - becomes Geppetto SQL normalization helper
- `temporal-relationships/internal/extractor/entityhistory/query.go`
  - strong safety logic moved into Geppetto `readonly.go`
- `temporal-relationships/internal/extractor/runturnhistory/query.go`
  - same as above; delete duplicate after migration
- `temporal-relationships/internal/extractor/transcripthistory/tool.go`
  - replaced by thin wrapper over `scopeddb.RegisterPrebuilt`
- `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go`
  - replaced by thin wrappers over `scopeddb.NewLazyRegistrar`

## Intern Checklist

If an intern starts implementing this ticket, the recommended order is:

1. Read the Geppetto tool primitives first.
   - `geppetto/pkg/inference/tools/definition.go`
   - `geppetto/pkg/inference/tools/registry.go`
   - `geppetto/pkg/inference/toolloop/loop.go`
   - `geppetto/pkg/inference/toolloop/enginebuilder/builder.go`
2. Read the Pinocchio registration hooks.
   - `pinocchio/pkg/inference/runtime/engine.go`
   - `pinocchio/pkg/webchat/router.go`
   - `pinocchio/pkg/webchat/llm_loop_runner.go`
3. Read the current temporal generic kernel.
   - `temporal-relationships/internal/extractor/scopeddb/*`
4. Read one simple domain package first.
   - `temporal-relationships/internal/extractor/transcripthistory/*`
5. Read one strict package second.
   - `temporal-relationships/internal/extractor/entityhistory/query.go`
6. Read the runtime call sites last.
   - `gorunner/tools_persistence.go`
   - `httpapi/run_chat_transport.go`

That reading order reduces confusion because it moves from framework primitives to app-specific specialization instead of the other way around.

## References

- `geppetto/pkg/inference/tools/definition.go:14-95` — canonical tool definition and `NewToolFromFunc`
- `geppetto/pkg/inference/tools/registry.go:8-125` — tool registry contract and implementation
- `geppetto/pkg/inference/tools/context.go:8-31` — registry transport via context
- `geppetto/pkg/inference/tools/advertisement.go:10-35` — provider-facing tool advertisement from live registry
- `geppetto/pkg/inference/tools/config.go:5-129` — tool runtime configuration
- `geppetto/pkg/inference/toolloop/loop.go:113-168` — registry attachment, tool snapshotting, and tool execution path
- `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:31-74` — standard runner inputs
- `geppetto/pkg/inference/toolloop/enginebuilder/builder.go:191-218` — switch into tool loop when registry exists
- `geppetto/pkg/doc/playbooks/01-add-a-new-tool.md:89-170` — current public tool authoring guidance
- `pinocchio/pkg/inference/runtime/engine.go:16-17` — `ToolRegistrar`
- `pinocchio/pkg/webchat/router.go:143-149` — app-owned tool registration
- `pinocchio/pkg/webchat/llm_loop_runner.go:109-129` — runtime registry rebuild and allowed-tool filtering
- `pinocchio/pkg/middlewares/sqlitetool/middleware.go:30-210` — existing generic sqlite tool precedent
- `geppetto/pkg/inference/middleware/sqlitetool/sqlite-tool-middleware.md:17-26` — documented behavior of that precedent
- `temporal-relationships/internal/extractor/scopeddb/schema.go:14-80` — current generic SQLite snapshot helpers
- `temporal-relationships/internal/extractor/scopeddb/query.go:5-59` — current generic query limits and cell normalization
- `temporal-relationships/internal/extractor/scopeddb/helpers.go:10-43` — normalization and allowlist helpers
- `temporal-relationships/internal/extractor/transcripthistory/schema.go:11-30` — simple domain schema/allowlist example
- `temporal-relationships/internal/extractor/transcripthistory/preload.go:13-189` — transcript scope materialization
- `temporal-relationships/internal/extractor/transcripthistory/query.go:13-178` — lightweight current query guard
- `temporal-relationships/internal/extractor/transcripthistory/tool.go:12-49` — current domain-specific tool registration wrapper
- `temporal-relationships/internal/extractor/entityhistory/schema.go:11-43` — complex domain schema/allowlist example
- `temporal-relationships/internal/extractor/entityhistory/query.go:13-337` — strong current readonly query enforcement
- `temporal-relationships/internal/extractor/runturnhistory/preload.go:21-200` — turn/history snapshot materialization
- `temporal-relationships/internal/extractor/runturnhistory/query.go:13-337` — duplicated strong readonly query enforcement
- `temporal-relationships/internal/extractor/gorunner/tools_persistence.go:30-124` — build-once runtime integration
- `temporal-relationships/internal/extractor/httpapi/run_chat_transport.go:227-398` — lazy per-call runtime integration
- `temporal-relationships/internal/extractor/runchat/session_store.go:13-35` — app-specific tool keys and tool setup model
- `temporal-relationships/internal/extractor/runchat/session_store.go:410-489` — app-specific validation rules for persisted tool scopes
- `temporal-relationships/internal/extractor/httpapi/run_chat_handlers.go:362-500` — normalization of chat-session tool scopes

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
