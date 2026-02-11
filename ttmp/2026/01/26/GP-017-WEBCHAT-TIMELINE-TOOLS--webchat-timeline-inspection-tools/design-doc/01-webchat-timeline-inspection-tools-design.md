---
Title: 'Webchat timeline inspection tools: design'
Ticket: GP-017-WEBCHAT-TIMELINE-TOOLS
Status: active
Topics:
    - webchat
    - backend
    - cli
    - debugging
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: glazed/pkg/doc/tutorials/05-build-first-command.md
      Note: Reference for Glazed command construction
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Integration point for new timeline subcommands
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: Data source for inspection commands
ExternalSources: []
Summary: Design a set of read-only CLI tools under cmd/web-chat for inspecting timeline persistence and hydration snapshots.
LastUpdated: 2026-01-26T12:35:00-05:00
WhatFor: Provide a concrete CLI design and implementation plan for timeline inspection tools.
WhenToUse: When implementing or reviewing the web-chat timeline inspection CLI.
---


# Webchat timeline inspection tools: design

## Executive Summary

Add a `web-chat timeline` command group under `pinocchio/cmd/web-chat` that provides read-only inspection of persisted hydration data. The commands use Glazed output so results are available as tables, JSON, YAML, or CSV. They connect to the same SQLite timeline DB used by the web server (via `--timeline-dsn`/`--timeline-db`) and optionally to the HTTP `/timeline` endpoint for remote snapshots.

## Problem Statement

Hydration debugging relies on manual DB queries, ad-hoc scripts, or the web UI. There is no official CLI for listing conversations, inspecting timeline entities, or exporting snapshots for diffing. This slows down debugging when investigating duplicate entities, ordering issues, and discrepancies between live streaming and persisted state.

## Proposed Solution

### Command group
Introduce a new command group under the existing `web-chat` binary:

```
web-chat timeline <subcommand>
```

### Subcommands (read-only)

1) `timeline list`
- Lists conversations present in the timeline DB.
- Output columns: `conv_id`, `version`, `entity_count`, `min_created_at_ms`, `max_updated_at_ms`.
- Optional filters: `--conv-id-prefix`, `--limit`.

2) `timeline snapshot`
- Reads a `TimelineSnapshotV1` for a conversation from the store (or via HTTP).
- Flags: `--conv-id` (required), `--since-version`, `--limit`, `--base-url` (optional HTTP mode).
- Output options:
  - Structured row with `conv_id`, `version`, `server_time_ms`, `entities` (JSON array).
  - `--raw-json` to emit the raw protojson snapshot.

3) `timeline entities`
- Lists entities for a conversation with filtering.
- Flags: `--conv-id` (required), `--kind`, `--entity-id`, `--version-min`, `--version-max`, `--created-after-ms`, `--updated-before-ms`, `--limit`, `--order`.
- Output columns: `entity_id`, `kind`, `created_at_ms`, `updated_at_ms`, `version`, `summary` (optional derived field for messages/tool calls).

4) `timeline entity`
- Dumps one entity by `entity_id` (and `conv_id`), including raw protojson.
- Intended for deep inspection or debugging a specific duplicate.

5) `timeline stats`
- Aggregated per-conversation stats: counts by kind, earliest/latest timestamps, max version, distinct entity count.
- Helps detect duplication or missing entity categories.

6) `timeline verify` (optional)
- Runs consistency checks (monotonic versioning, missing IDs, unexpected empty kinds).
- Emits rows describing any anomalies.

### Data access

- Primary source: SQLite timeline DB using the same `SQLiteTimelineStore` semantics.
- For list/stats/filters, run SQL queries directly against `timeline_entities` to avoid reading full snapshots.
- For snapshot, reuse `TimelineStore.GetSnapshot` to ensure ordering matches production hydration.
- Optional HTTP mode for `snapshot` via `GET /timeline` when `--base-url` is provided.

### Output strategy (Glazed)

All subcommands implement `cmds.GlazeCommand`:
- Use `types.Row` for structured output.
- Provide `settings.NewGlazedParameterLayers()` and `cli.NewCommandSettingsLayer()` so `--output`, `--fields`, `--sort-columns`, `--print-schema`, etc. are available.
- Parse all flags via `parsedLayers.InitializeStruct(layers.DefaultSlug, &Settings{})`.

### CLI plumbing

- Add new command structs under `pinocchio/cmd/web-chat` (e.g., `timeline_list.go`, `timeline_snapshot.go`, etc.).
- Create a small helper to open the DB:
  - `--timeline-dsn` or `--timeline-db` (same semantics as server).
  - Use `webchat.SQLiteTimelineDSNForFile` when only a file path is provided.
- Add a `timeline` parent command to the root `web-chat` command.

## Design Decisions

1) **Keep tools inside `web-chat` binary**
- Pros: consistent config flags, reuse existing Glazed setup, zero new binaries.
- Cons: larger binary surface area.

2) **Use Glazed output everywhere**
- Pros: uniform output formats for automation.
- Cons: raw JSON dumps need a `--raw-json` option.

3) **Reuse timeline store semantics for snapshots**
- Pros: parity with hydration ordering rules.
- Cons: list/stats still require SQL access.

4) **Read-only by default**
- Keeps tools safe and lowers operational risk.

## Alternatives Considered

- **Manual `sqlite3` queries only**: error-prone and not reusable.
- **Separate `timeline-inspect` binary**: adds build complexity and duplicates Glazed setup.
- **HTTP-only inspection**: requires server running and timeline store enabled; not ideal for offline analysis.
- **Python scripts**: useful for quick checks but not standardized or discoverable.

## Implementation Plan

1) Add a `timeline` parent command under `cmd/web-chat` (Cobra + Glazed).
2) Implement `timeline list` and `timeline snapshot` first (core inspection).
3) Add `timeline entities` and `timeline entity` for focused inspection.
4) Add `timeline stats` and optional `timeline verify` for diagnostics.
5) Document usage in ticket analysis and update help text.

## Open Questions

- Should `timeline snapshot` default to DB or HTTP mode when both are available?
- Do we need a `--conv-id` auto-complete or `--latest` convenience flag?
- How much derived decoding should `timeline entities` do (message content vs raw JSON only)?
- Should `timeline verify` check additional invariants (e.g., created_at monotonicity)?

## References

- Glazed tutorial: `glazed/pkg/doc/tutorials/05-build-first-command.md`
- Timeline store implementation: `pinocchio/pkg/webchat/timeline_store_sqlite.go`
- Timeline endpoint: `pinocchio/pkg/webchat/router.go`
