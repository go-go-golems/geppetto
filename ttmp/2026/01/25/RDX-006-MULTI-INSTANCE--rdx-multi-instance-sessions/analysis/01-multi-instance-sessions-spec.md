---
Title: Multi-Instance Sessions Spec
Ticket: RDX-006-MULTI-INSTANCE
Status: active
Topics: []
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: rdx/cmd/rdx/commands.go
      Note: Entry points for instance selection
    - Path: rdx/cmd/rdx/socketcluster_commands.go
      Note: Instance tracking hooks
    - Path: rdx/cmd/rdx/socketcluster.go
      Note: SocketCluster monitor and message fanout
    - Path: rdx/pkg/rtk/socketcluster.go
      Note: Relay message parsing helpers
    - Path: rdx/pkg/rtk/scclient/client.go
      Note: Minimal SocketCluster client now powering monitor
ExternalSources: []
Summary: "Define multi-instance registry, selectors, and sessions command group."
LastUpdated: 2026-01-26T12:55:00-05:00
WhatFor: "Guide implementation of multi-instance session discovery and selection"
WhenToUse: "When adding instance selectors or session registry features"
---

# Multi-Instance Sessions Spec

## Goal

Support monitoring and managing **multiple concurrent instances** of Redux DevTools sessions, with grouping, labeling, and selection semantics.

## Context

- SocketCluster streams do not provide a formal list API. Instances are inferred from traffic.
- Teams often run multiple apps or multiple tabs simultaneously.
- We now own a minimal SocketCluster client, so instance tracking should remain independent of transport details.

Related tickets:
- RDX-001-RTK-CLI (base SocketCluster stream)
- RDX-002-HIST-REPORTS (historical sessions may provide alternative instance lists)
- RDX-007-SC-CLIENT (minimal SocketCluster client replacement)

## Requirements

### Functional

1. Maintain a live index of instances with last-seen timestamps.
2. Support filters: by app name, environment, instanceId prefix.
3. Allow `tail`, `state`, `watch` to accept instance selectors (not only ids).
4. Provide a `rdx sessions` command to list and label instances.

### Non-Functional

- No compatibility with the custom protocol.
- Best-effort metadata inference is acceptable.
- In-memory registry is acceptable unless persistence is explicitly requested.

## Data Model

Maintain an in-memory registry:

```
instanceId -> {
  appName,
  firstSeen,
  lastSeen,
  meta,
  label
}
```

Labels can be user-defined or derived from `meta` or relay message fields.

## Command Surface

### `rdx sessions`

List known instances with last-seen and labels.

### `rdx sessions label <instance-id> <label>`

Add a label to an instance for easier selection.

### `rdx tail --instance-select <pattern>`

Select instance by regex or label rather than id.

## Selector Semantics

- `--instance-select` accepts:
  - Exact instance id
  - Label match
  - Regex (if value is wrapped with `/.../`)
- If multiple matches are found, return an error listing matches.
- If no match, return a clear error suggesting `rdx sessions`.

## Implementation Plan

1. Add a session registry component to track active instances.
2. Extend `list` command to output registry entries.
3. Add `sessions` command group.
4. Implement selector resolution for `tail/state/watch`.

## Files and Symbols to Touch

- `rdx/cmd/rdx/socketcluster_commands.go` (instance tracking)
- `rdx/cmd/rdx/commands.go` (new sessions command)
- New: `rdx/pkg/rtk/session_registry`
- `rdx/pkg/rtk/scclient` (unchanged, just used)

## Out of Scope

- Cross-session analytics or aggregation of state diffs.
- Persisting labels across runs (unless requested).

## Deliverables

- Session registry and `sessions` command group.
- Selector logic for existing commands.
