---
Title: Static Diff and Snapshot Tools Spec
Ticket: RDX-004-DIFF-SNAPSHOT
Status: active
Topics: []
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: rdx/cmd/rdx/socketcluster_commands.go
      Note: State retrieval logic
    - Path: rdx/pkg/rtk/path.go
      Note: Dot-path utilities
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-26T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Static Diff and Snapshot Tools Spec

## Goal

Provide CLI utilities to **capture**, **store**, and **diff** state snapshots from Redux DevTools sessions. These tools are offline, deterministic, and designed for debugging and regression analysis.

## Context

- The current CLI can read live `STATE`/`ACTION` payloads but does not provide offline snapshots or diffs.
- Developers often need to compare state between two points without re-running the app.

Related tickets:
- RDX-001-RTK-CLI (live state access)
- RDX-003-DISPATCH-SYNC (control channel can trigger state updates)
- RDX-002-HIST-REPORTS (reports can be exported as snapshots)

## Requirements

### Functional

1. Capture a state snapshot and save to file.
2. Diff two snapshots with optional JSON patch output.
3. Provide filtering and path-based selection for diffs.
4. Output in structured format for automation.

### Non-Functional

- Offline processing only; no performance constraints.
- No backward compatibility.

## Command Surface

### `rdx snapshot save <instance-id> --out <path>`

- Connects to the live server, waits for a `STATE`/`ACTION` payload, and writes JSON to a file.
- Supports `--path` to store a sub-tree only.

### `rdx snapshot diff <left> <right>`

- Loads two JSON snapshots and computes differences.
- Supports `--format summary|jsonpatch|raw`.
- Supports `--path` to scope the diff.

### `rdx snapshot list`

- Optional: list locally saved snapshots (if stored in a default directory).

## Data Model

Snapshots are plain JSON files with metadata:

```json
{
  "timestamp": "2026-01-26T12:34:56Z",
  "instanceId": "abc123",
  "path": "root",
  "state": { ... }
}
```

## Diff Algorithms

- Default: deep equality comparison with a structural diff.
- Optional JSON Patch output (RFC 6902) using a Go library.

## Implementation Plan

1. Add snapshot model + serializer (`rdx/pkg/rtk/snapshot`).
2. Implement `snapshot` command group in CLI.
3. Use existing `state` logic to retrieve snapshots from the server.
4. Implement diff calculation and output adapters.

## Files and Symbols to Touch

- `rdx/cmd/rdx/socketcluster_commands.go` (reuse for state fetch)
- New: `rdx/pkg/rtk/snapshot`, `rdx/pkg/rtk/diff`

## Out of Scope

- Live diff streaming (covered by `watch` command).
- Server-side snapshot storage.

## Deliverables

- Snapshot save + diff commands.
- JSON fixture examples for tests.
- Documentation for snapshot metadata and formats.

