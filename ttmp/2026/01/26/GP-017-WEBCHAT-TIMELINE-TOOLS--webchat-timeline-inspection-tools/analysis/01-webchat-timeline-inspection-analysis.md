---
Title: 'Webchat timeline inspection: analysis'
Ticket: GP-017-WEBCHAT-TIMELINE-TOOLS
Status: active
Topics:
    - webchat
    - backend
    - cli
    - debugging
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: glazed/pkg/doc/tutorials/05-build-first-command.md
      Note: Glazed command patterns and layers
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Current web-chat CLI command structure
    - Path: pinocchio/pkg/webchat/router.go
      Note: GET /timeline handler and query params
    - Path: pinocchio/pkg/webchat/timeline_store_memory.go
      Note: In-memory timeline semantics and eviction
    - Path: pinocchio/pkg/webchat/timeline_store_sqlite.go
      Note: SQLite timeline schema and snapshot ordering
ExternalSources: []
Summary: Analyze current timeline persistence + hydration surfaces and define CLI inspection requirements.
LastUpdated: 2026-01-26T12:35:00-05:00
WhatFor: Scope CLI inspection tools for persisted conversations and hydration state in webchat.
WhenToUse: When debugging hydration or building read-only tools against the timeline store.
---


# Webchat timeline inspection: analysis

## Overview

The webchat server persists hydration state via the timeline projection store. Today, inspection happens indirectly via the web UI, Redux DevTools, or manual SQLite queries. There are no purpose-built CLI tools under `pinocchio/cmd/web-chat` for reading persisted timeline data. This analysis describes the persistence surfaces, semantics, and debugging gaps that the new inspection tools should address.

## Persistence surfaces today

### 1) Timeline store (canonical hydration source)
- Optional, enabled by `--timeline-dsn` or `--timeline-db` when running the webchat server.
- Implementations:
  - SQLite (`SQLiteTimelineStore`) for durable storage.
  - In-memory (`InMemoryTimelineStore`) when no SQLite DB is configured.
- The timeline projector (`TimelineProjector`) updates the store whenever SEM frames are processed during a conversation.

### 2) HTTP snapshot endpoint
- `GET /timeline?conv_id=...&since_version=...&limit=...`
- Returns a `TimelineSnapshotV1` JSON payload used for hydration.
- Built from the timeline store; not available when the store is disabled.

### 3) Client-side hydration
- The browser calls `GET /timeline` to hydrate the Redux timeline state.
- Streaming WebSocket events update the UI during live sessions.

## SQLite schema and semantics

SQLite store tables (in `timeline_store_sqlite.go`):

- `timeline_versions`:
  - `conv_id TEXT PRIMARY KEY`
  - `version INTEGER NOT NULL`

- `timeline_entities`:
  - `conv_id TEXT NOT NULL`
  - `entity_id TEXT NOT NULL`
  - `kind TEXT NOT NULL`
  - `created_at_ms INTEGER NOT NULL`
  - `updated_at_ms INTEGER NOT NULL`
  - `version INTEGER NOT NULL`
  - `entity_json TEXT NOT NULL`
  - primary key `(conv_id, entity_id)`

Ordering / versioning behavior:
- Each upsert increments `version` for the conversation (monotonic per `conv_id`).
- The `created_at_ms` of an entity is preserved across upserts; `updated_at_ms` is set to now.
- Snapshots are ordered by `version ASC, entity_id ASC` for full hydration, and by `version ASC` for incremental updates.

JSON format:
- `entity_json` is protojson with lowerCamelCase field names.
- Payload matches the `sem.timeline.TimelineEntityV1` structure.

In-memory store:
- Mirrors ordering semantics of SQLite.
- Evicts oldest entities when `maxEntitiesPerConv` is exceeded.

## Current observability gaps

- No CLI to list conversations or inspect timeline entities in bulk.
- No read-only tool to compare persisted snapshots with the hydrated client state.
- Manual SQLite queries are error-prone and require knowledge of JSON structure.
- Remote inspection is difficult without spinning up the web server and calling `/timeline`.

## Debugging needs and user stories

Common investigation needs:
- List conversation IDs present in the timeline DB.
- Inspect entity counts and kinds per conversation (messages vs tool calls vs planning, etc.).
- Fetch a full snapshot for a conversation and export it to JSON for comparison.
- Inspect a specific entity ID, see creation/update times, and dump raw protojson.
- Filter entities by kind, version range, or created/updated time to spot duplicates or ordering anomalies.

Hydration-specific needs:
- Verify that optimistic entities are reconciled with canonical IDs (`user-<turn_id>`).
- Inspect version gaps or unexpected ordering when comparing live WS events to persisted data.
- Confirm that `created_at_ms` and `updated_at_ms` behavior is consistent across upserts.

## Constraints from Glazed command patterns

The new tools should follow the Glazed command conventions used in `cmd/web-chat`:
- Use `cmds.CommandDescription` and `cli.BuildCobraCommand` for subcommands.
- Parse settings via `parsedLayers.InitializeStruct(...)` (not direct Cobra flags).
- Add `settings.NewGlazedParameterLayers()` and `cli.NewCommandSettingsLayer()` so commands inherit `--output`, `--fields`, `--print-schema`, etc.
- Prefer `GlazeCommand` (structured output) for inspection commands.

## Scope boundary

In scope:
- Read-only inspection tools for timeline persistence and hydration snapshots.
- Local SQLite DB access plus optional HTTP snapshot access.

Out of scope (for now):
- Mutating or pruning timeline data.
- Editing conversations or replaying events.
- New persistence formats beyond SQLite.
