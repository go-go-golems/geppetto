---
Title: Historical Sessions and Reports Spec
Ticket: RDX-002-HIST-REPORTS
Status: active
Topics: []
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../../../tmp/remotedev-server/package/README.md
      Note: Report store and GraphQL notes
    - Path: ../../../../../../../../../../../../tmp/remotedev-server/package/lib/api/schema_def.graphql
      Note: Canonical GraphQL schema for report fields
    - Path: geppetto/ttmp/2026/01/25/RDX-001-RTK-CLI--rdx-rtk-cli/analysis/03-rtk-devtools-cli-design-socketcluster.md
      Note: Base SocketCluster CLI design
    - Path: rdx/cmd/rdx/commands.go
      Note: Glazed command registration pattern
    - Path: rdx/cmd/rdx/socketcluster_commands.go
      Note: Existing live stream command patterns
    - Path: rdx/pkg/rtk/socketcluster.go
      Note: Message decoding utilities to mirror for reports
ExternalSources: []
Summary: "Define the report GraphQL integration, commands, and replay strategy for historical sessions."
LastUpdated: 2026-01-26T11:30:00-05:00
WhatFor: "Guide implementation of historical reports in RDX CLI"
WhenToUse: "When adding report listing, export, or replay functionality"
---

# Historical Sessions and Reports Spec

## Goal

Add CLI support for **historical sessions and reports** backed by the remotedev server’s report store (GraphQL + DB). This enables retrieving prior action histories, sharing session snapshots, and inspecting sessions without live traffic.

## Context and Current State

- The current CLI only streams **live** SocketCluster traffic (`log` channel).
- The remotedev server can persist **reports** and exposes a GraphQL endpoint (`/graphql`, with `/graphiql` UI) for stored data.
- Reports include action history, payload, metadata, and timestamps. The server schema is driven by the report store fields.

Related tickets:
- RDX-001-RTK-CLI (base SocketCluster CLI)
- RDX-003-DISPATCH-SYNC (control channel and time travel) — may reuse report payloads

## Existing Codebase Patterns to Reuse

- **Glazed command structure**: Follow the `NewXCommand` + `RunIntoGlazeProcessor` pattern in `rdx/cmd/rdx/commands.go`.
- **Parameter parsing**: Always use `parsedLayers.InitializeStruct(layers.DefaultSlug, &Settings{})` (per Glazed tutorial) instead of reading Cobra flags directly.
- **Output rows**: Use `types.NewRow` and `types.MRP` like `socketcluster_commands.go` for consistent column naming.
- **Base flags**: Mirror `baseFlags()` for shared server + token flags, but introduce a separate HTTP base URL for reports to avoid mixing with SocketCluster URLs.
- **Error wrapping**: Use `github.com/pkg/errors` when returning contextual errors (as in SocketCluster code).

## Requirements

### Functional

1. **List reports** with filters (app, instanceId, date range, type).
2. **Fetch report details** by id.
3. **Export report** to JSON or JSONL.
4. **Replay report** into a structured stream compatible with existing CLI commands.
5. **Optional**: open GraphiQL URL in browser (non-default).

### Non-Functional

- No performance constraints: in-memory processing is fine.
- No backward compatibility with old custom protocol.
- Output must be structured via Glazed.

## Data Model (Report Store)

The report store persists fields similar to:

- `id`, `type`, `title`, `description`
- `action`, `payload`, `preloadedState`
- `version`, `userAgent`, `user`, `userId`
- `instanceId`, `meta`, `exception`, `screenshot`
- `added`, `appId`

These fields should be surfaced as CLI columns (with selectable subsets).

## GraphQL Schema (Confirmed)

From `remotedev-server`:

```graphql
enum ReportType {
  STATE
  ACTION
  STATES
  ACTIONS
}

type Report {
  id: ID!
  type: ReportType
  title: String
  description: String
  action: String
  payload: String
  preloadedState: String
  screenshot: String
  userAgent: String
  version: String
  userId: String
  user: String
  meta: String
  exception: String
  instanceId: String
  added: String
  appId: ID
}

type Query {
  reports: [Report]
  report(id: ID!): Report
}
```

Notes:
- No server-side filtering in the schema; apply filters client-side.
- `added` is stored as ISO-8601 via `new Date().toISOString()`.

## Commands

### `rdx report list`

**Purpose:** List stored reports with filters.

**Flags:**
- `--server` (base HTTP URL for GraphQL, default `http://localhost:8000`)
- `--type` (`STATE`, `ACTION`, `STATES`, `ACTIONS`)
- `--instance-id`
- `--since`, `--until` (RFC3339)
- `--limit`, `--offset`

**Output columns:**
- `id`, `type`, `title`, `instance_id`, `added`, `user`, `app_id`, `meta`

### `rdx report show <id>`

**Purpose:** Fetch and display full report details.

**Output:** a single row containing all report fields, with `payload` and `action` decoded if possible.

### `rdx report export <id>`

**Purpose:** Export report into file(s).

**Flags:**
- `--format json|jsonl`
- `--out path`

### `rdx report replay <id>`

**Purpose:** Convert report payload into a stream of `ACTION` / `STATE` rows.

**Flags:**
- `--speed` (replay speed multiplier)
- `--start-at` (action index)

### Optional: `rdx report graphiql`

**Purpose:** Open the GraphiQL UI for the report server.

**Flags:**
- `--server` (base HTTP URL)

## Replay Strategy

1. Parse `payload` as JSON. If parsing fails, emit a single row with `payload_raw` and stop.
2. If payload is a lifted state object (contains `actionsById` + `computedStates`):
   - Sort `stagedActionIds` (or fall back to `actionsById` keys).
   - Emit `ACTION` rows with `action_type`, `action_id`, and `action` fields.
   - Emit `STATE` rows with `state_index` and `state` derived from `computedStates`.
3. If payload is an array: treat as ordered actions and emit `ACTION` rows.
4. Include `report_id`, `instance_id`, `added`, and `source="report"` metadata on each row.

## Clean Expansion Plan

- **New package**: `rdx/pkg/rtk/reports`
  - `Client` with `ListReports(ctx)` and `GetReport(ctx, id)`.
  - GraphQL request/response structs, error handling, and `ParseReportPayload` helper.
- **Command files**: `rdx/cmd/rdx/report_commands.go` + `report_runtime.go`.
  - Mirror patterns from `socketcluster_commands.go` for row creation.
  - Keep HTTP/GraphQL logic out of `cmd` and in `pkg/rtk/reports`.
- **Flag consistency**: Use `--server` but ensure it is HTTP (not SocketCluster). Provide clear help text.
- **Column naming**: Use `snake_case` keys like existing commands.
- **Parsing logic**: Keep JSON decoding tolerant; avoid failing the entire command if one report has malformed payload.

## Implementation Plan

1. Implement HTTP GraphQL client in `rdx/pkg/rtk/reports` with schema-aligned types.
2. Add `report` command group with subcommands (`list`, `show`, `export`, `replay`, optional `graphiql`).
3. Map GraphQL results to Glazed rows with selectable fields.
4. Add replay functionality: parse payload history into `ACTION` and `STATE` rows.
5. Add JSON fixtures + unit tests for payload parsing and replay mapping.
6. Document usage and edge cases in the ticket workspace.

## Files and Symbols to Touch

- `rdx/cmd/rdx/commands.go` (register new commands)
- `rdx/cmd/rdx/report_commands.go` (new: command definitions)
- `rdx/cmd/rdx/report_runtime.go` (new: Glazed row mapping)
- `rdx/pkg/rtk/reports` (new package for GraphQL queries + types)
- `rdx/pkg/rtk/reports/replay.go` (new: payload parsing)

## Open Questions

- Whether `payload` is JSAN-encoded; if so, do we need JSAN decoding or accept JSON only?
- Whether `report.type` always aligns with payload shape (`ACTIONS` vs `STATE`).
- Should `report list` support sorting by `added`? (Default likely newest-first.)

## Out of Scope

- Modifying the server schema or migrations.
- Live stream behaviors (covered by RDX-001).
- Writing reports back to the server.

## Deliverables

- New CLI commands with structured output.
- Documentation and examples in ticket.
- JSON fixtures for testing GraphQL parsing.
