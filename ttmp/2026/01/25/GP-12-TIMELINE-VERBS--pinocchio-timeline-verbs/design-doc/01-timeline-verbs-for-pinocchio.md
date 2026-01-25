---
Title: Timeline verbs for pinocchio
Ticket: GP-12-TIMELINE-VERBS
Status: active
Topics:
    - backend
    - tools
    - architecture
    - persistence
    - go
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/pinocchio/main.go
      Note: Command registration pattern for new CLI groups
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Projection kinds and snapshot generation behavior
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: SQLite schema and snapshot ordering for timeline projections
    - Path: pinocchio/proto/sem/timeline/message.proto
      Note: Message projection fields
    - Path: pinocchio/proto/sem/timeline/tool.proto
      Note: Tool call/result projection fields
    - Path: pinocchio/proto/sem/timeline/transport.proto
      Note: Canonical TimelineEntityV1 and snapshot definitions
ExternalSources: []
Summary: Design a suite of Glazed CLI commands under `pinocchio timeline` to query SQLite timeline projection stores with structured output.
LastUpdated: 2026-01-25T13:36:44-05:00
WhatFor: Provide a consistent, inspectable CLI for timeline projection data stored by Pinocchio's SQLite timeline store.
WhenToUse: Use when debugging or analyzing timeline projections without running the web server.
---


# Timeline verbs for pinocchio

## Executive Summary

Introduce a `pinocchio timeline` command group with Glazed subcommands that read the SQLite timeline projection store and emit structured rows. The commands focus on inspection (list conversations, entities, kinds, stats), data extraction (snapshots, raw entity JSON), and incremental “tail”-style updates. They reuse existing timeline protobuf types and the SQLite schema already produced by the webchat timeline projector, so no new persistence format is introduced.

## Problem Statement

Timeline projections are persisted to SQLite by the webchat server, but there is no direct CLI to inspect or analyze them. Developers currently need to run the web server or write ad-hoc SQL/JSON queries, which is slow and inconsistent. We need a dedicated set of Glazed commands to query the timeline projection database and emit structured output for debugging, audit, and analysis tasks.

## Proposed Solution

### Command group

Add a new top-level command group:

```
pinocchio timeline <subcommand>
```

Each subcommand implements `cmds.GlazeCommand` and yields `types.Row` objects. Parameters are parsed via `parsedLayers.InitializeStruct(layers.DefaultSlug, &Settings{})`, following the Glazed tutorial guidance.

### Common settings layer

Add a shared settings struct and parameter layer for timeline DB access:

- `--timeline-dsn` (string, preferred)
- `--timeline-db` (string, file path; use `SQLiteTimelineDSNForFile`)
- `--read-only` (bool, default true) — wraps DSN with `mode=ro&immutable=1` when possible
- `--conv-id` (string) — for commands that need a specific conversation

This mirrors the `timeline-dsn`/`timeline-db` parameters used by the webchat router for the `/timeline` endpoint.

### Subcommands (initial set)

1. `pinocchio timeline conversations`
   - Lists conversation IDs with current projection version.
   - Source table: `timeline_versions`.
   - Output columns: `conv_id`, `version`, `entity_count`, `latest_updated_at_ms` (derived via subquery).

2. `pinocchio timeline snapshot`
   - Emits the same data as the HTTP `/timeline` endpoint but via CLI.
   - Parameters: `--conv-id` (required), `--since-version`, `--limit`.
   - Uses `TimelineStore.GetSnapshot` for consistent ordering.
   - Output rows: `conv_id`, `version`, `server_time_ms`, plus flattened entity fields (see projections below) unless `--raw` is provided.

3. `pinocchio timeline entities`
   - Query entities directly from `timeline_entities` with filters.
   - Parameters: `--conv-id` (required), `--kind` (repeatable), `--since-version`, `--since-created-ms`, `--order` (created|version), `--limit`.
   - Output rows: standard entity columns + projection-specific fields.

4. `pinocchio timeline entity`
   - Fetch a single entity by `--conv-id` and `--entity-id`.
   - Optionally include `--raw-json` column for debugging.

5. `pinocchio timeline kinds`
   - Lists distinct `kind` values for a conversation with counts.

6. `pinocchio timeline stats`
   - Aggregated metrics for a conversation: counts per kind, total entities, earliest/last timestamps, current version.

7. `pinocchio timeline tail`
   - Polling loop to emit incremental updates (`--since-version` tracked in-memory).
   - Parameters: `--poll-interval`, `--limit` per poll.
   - Uses Glazed output so it can be piped/filtered.

### Projection mapping (entity → row)

All entity rows include:

- `conv_id`
- `entity_id`
- `kind`
- `created_at_ms`
- `updated_at_ms`
- `version`

Additional fields depend on the `kind` and snapshot type:

- `message`: `role`, `content`, `streaming`, `metadata_json`
- `tool_call`: `name`, `status`, `progress`, `done`, `input_json`
- `tool_result`: `tool_call_id`, `custom_kind`, `error`, `result_raw`, `result_json`
- `thinking_mode`: `status`, `mode`, `phase`, `reasoning`, `success`, `error`
- `mode_evaluation`: `status`, `current_mode`, `analysis`, `decision`, `recommended_mode`, `success`, `error`
- `inner_thoughts`: `status`, `text`
- `status`: `text`, `type`
- `team_analysis`: `status`, `team_size`, `depth`, `progress`, `current_pair`, `insights_json`, `visualization_json`
- `planning`: `run_id`, `status`, `provider`, `planner_model`, `max_iterations`, `started_at_unix_ms`, `completed_at_unix_ms`, `final_decision`, `status_reason`, `final_directive`, `iteration_count`, `execution_status`, `execution_error`

Structured fields (`google.protobuf.Struct`, maps, or nested messages) are emitted as JSON strings for Glazed output. This keeps outputs deterministic and avoids schema drift in tabular formats while still allowing JSON/YAML export.

### SQL access pattern

Use a small `TimelineDB` reader that opens SQLite with read-only flags. Two query paths are used:

- `timeline_versions` for conversation list and version state.
- `timeline_entities` for entity enumeration (ordered by `created_at_ms` or `version`).

Entity JSON is decoded with:

```go
protojson.UnmarshalOptions{DiscardUnknown: true}
```

This matches the write-side behavior in `SQLiteTimelineStore` and tolerates unknown fields from future schema versions.

### Integration

- Add `pinocchio/cmd/pinocchio/cmds/timeline` package.
- Provide a `RegisterTimelineCommands(rootCmd *cobra.Command)` similar to `tokens.RegisterCommands`.
- Register the group in `pinocchio/cmd/pinocchio/main.go` during `initAllCommands`.

## Design Decisions

1. **Use Glazed commands for all subcommands.**
   - Ensures consistent output formats and filtering (`--output`, `--fields`, `--sort-columns`).
2. **Read directly from SQLite instead of HTTP `/timeline`.**
   - Avoids server dependency and supports offline analysis.
3. **Keep a minimal shared settings layer.**
   - Mirrors `timeline-dsn`/`timeline-db` already used in webchat configuration.
4. **Emit structured fields as JSON strings.**
   - Keeps row schemas stable while preserving full content.
5. **Prefer read-only SQLite connections.**
   - Protects projection data and allows concurrent server usage.

## Alternatives Considered

- **Use HTTP `/timeline` endpoint only.**
  - Rejected: requires a running server and does not expose aggregation queries (kinds/stats).
- **Direct SQL via `sqlite3` CLI.**
  - Rejected: inconsistent output and no Glazed formatting.
- **Expose a new gRPC/HTTP endpoint for queries.**
  - Rejected: heavier infrastructure and duplicates the need for local inspection.

## Implementation Plan

1. Create `pinocchio/cmd/pinocchio/cmds/timeline` package with shared DB/config helpers.
2. Implement `TimelineDB` reader (open/close, query helpers for versions and entities).
3. Build projection mappers from `TimelineEntityV1` → `types.Row`.
4. Implement subcommands: `conversations`, `snapshot`, `entities`, `entity`, `kinds`, `stats`, `tail`.
5. Wire the command group into `initAllCommands` in `pinocchio/cmd/pinocchio/main.go`.
6. Add unit tests for SQL queries and projection mapping (use temp SQLite DB seeded via `SQLiteTimelineStore`).
7. Add docs or help text examples (include Glazed output flags).

## Open Questions

- Should `timeline snapshot` default to per-kind flattening or return a single JSON column by default?
- Do we need a `--since-created-ms` filter for incremental queries, or is `--since-version` sufficient?
- Should the CLI auto-detect the latest timeline DB used by webchat profiles (via config) as a convenience feature?
- How should we represent large `result_raw` payloads (truncate vs full text) for table output?

## References

- `pinocchio/pkg/webchat/timeline_store_sqlite.go`
- `pinocchio/pkg/webchat/timeline_projector.go`
- `pinocchio/proto/sem/timeline/transport.proto`
- `pinocchio/proto/sem/timeline/message.proto`
- `pinocchio/proto/sem/timeline/tool.proto`
- `glazed/pkg/doc/tutorials/05-build-first-command.md`
