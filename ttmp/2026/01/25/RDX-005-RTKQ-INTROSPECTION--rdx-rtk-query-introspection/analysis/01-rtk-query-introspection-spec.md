---
Title: RTK Query Introspection Spec
Ticket: RDX-005-RTKQ-INTROSPECTION
Status: active
Topics: []
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: rdx/cmd/rdx/socketcluster_commands.go
      Note: State retrieval for RTK Query slice
    - Path: rdx/pkg/rtk/path.go
      Note: Dot-path utilities
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-26T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# RTK Query Introspection Spec

## Goal

Provide specialized CLI commands to introspect **RTK Query state** for live sessions, including caches, subscriptions, and active requests.

## Context

- RTK Query stores its state under a reducer path (default: `api`).
- The state is large and not human-friendly in raw form.
- A focused CLI view is significantly more usable than full JSON dumps.

Related tickets:
- RDX-001-RTK-CLI (live state access)
- RDX-004-DIFF-SNAPSHOT (useful for comparing RTK Query cache states)

## Requirements

### Functional

1. Detect the RTK Query reducer path automatically (`api` by default, overrideable).
2. List cached queries and their status.
3. Show a single query cache entry with metadata.
4. Show active subscriptions and polling intervals.
5. Show mutations in flight and their results.

### Non-Functional

- No performance constraints.
- No backward compatibility.

## Command Surface

### `rdx rtkq list`\

List all cached query entries:
- `endpointName`
- `queryKey` / `requestId`
- `status`
- `fulfilledTimeStamp`
- `dataSize` (optional)

### `rdx rtkq show <endpoint> <arg>`

Show the cache entry for a specific endpoint+arg.

### `rdx rtkq subscriptions`

List active subscriptions by endpoint and cache key.

### `rdx rtkq mutations`

List ongoing and completed mutations, keyed by requestId.

### `rdx rtkq config`

Show RTK Query slice settings (reducer path, polling, refetch settings if present).

## Data Model

RTK Query state is typically structured as:

- `queries`: map of cache keys -> query state
- `mutations`: map of requestId -> mutation state
- `subscriptions`: map of cache keys -> subscription data
- `provided`: invalidation tags
- `config`: base config values

The CLI should navigate this structure and emit flattened rows.

## Implementation Plan

1. Implement a helper for locating the RTK Query slice (`--slice-path` override).
2. Build query/mutation/subscription iterators producing Glazed rows.
3. Add a `rtkq` command group in CLI.
4. Document default paths and examples.

## Files and Symbols to Touch

- `rdx/cmd/rdx/commands.go` (register group)
- New: `rdx/pkg/rtk/rtkq` (slice parsing + helpers)

## Out of Scope

- Modifying RTK Query configuration at runtime.
- Dispatching invalidation actions.

## Deliverables

- RTK Query inspection commands.
- Examples for common RTK Query setups.

