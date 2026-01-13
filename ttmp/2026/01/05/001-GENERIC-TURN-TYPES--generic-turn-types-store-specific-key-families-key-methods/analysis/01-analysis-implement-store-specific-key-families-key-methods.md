---
Title: 'Analysis: implement store-specific key families + key methods'
Ticket: 001-GENERIC-TURN-TYPES
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - turns
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/turnsrefactor/main.go
      Note: CLI entrypoint for turnsrefactor used during migration
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Linter that must be reviewed/updated to enforce canonical constructors and prevent ad-hoc key creation
    - Path: geppetto/pkg/analysis/turnsrefactor/refactor.go
      Note: One-shot refactor tool (DataGet/DataSet -> key.Get/key.Set) used for safe migration
    - Path: geppetto/pkg/inference/engine/turnkeys.go
      Note: Import-cycle escape hatch; KeyToolConfig must switch to turns.DataK
    - Path: geppetto/pkg/turns/keys.go
      Note: Canonical key definitions to migrate from turns.K to turns.DataK/TurnMetaK/BlockMetaK
    - Path: geppetto/pkg/turns/poc_split_key_types_test.go
      Note: Existing POC proving split key families + key receiver methods are viable
    - Path: geppetto/pkg/turns/types.go
      Note: Current production Key[T] + DataGet/Set; will be replaced by DataKey/TurnMetaKey/BlockMetaKey + key methods
ExternalSources: []
Summary: "Plan and codebase analysis for migrating Geppetto turns to store-specific key families (DataKey/TurnMetaKey/BlockMetaKey) with key receiver methods."
LastUpdated: 2026-01-05T17:15:23.890400581-05:00
WhatFor: "Drive implementation and migration of the new turns production API (key families + key methods), including downstream migration in moments/pinocchio and turnsrefactor usage."
WhenToUse: "Use when implementing key-family types/methods, migrating canonical keys, running turnsrefactor, and deleting the old function API."
---


# Analysis: implement store-specific key families + key methods

## Goal

Implement a new production `turns` API that:

1) Prevents mixing keys across stores at compile time (Turn.Data vs Turn.Metadata vs Block.Metadata).
2) Improves ergonomics by using **methods on keys** (not on stores), so Go generics allow it.
3) Enables a safe, deterministic migration with the existing one-shot refactor tool (`geppetto/cmd/turnsrefactor`), then removes the old API.

## Current State (what exists today)

### `turns` public surface

Production `geppetto/pkg/turns` currently exposes:

- One generic key type: `Key[T]` with a single underlying `TurnDataKey` id (`geppetto/pkg/turns/types.go`).
- Three generic function families:
  - `DataGet/DataSet`
  - `MetadataGet/MetadataSet`
  - `BlockMetadataGet/BlockMetadataSet`
- Three opaque wrappers used in structs:
  - `Turn.Data` is `turns.Data` (backed by `map[TurnDataKey]any`)
  - `Turn.Metadata` is `turns.Metadata` (backed by `map[TurnMetadataKey]any`)
  - `Block.Metadata` is `turns.BlockMetadata` (backed by `map[BlockMetadataKey]any`)

### The core problem

Because `Key[T]` is shared, *any* `Key[T]` can be passed to any store function. This allows accidental cross-store mixing (a Turn.Data key can be used against Turn.Metadata, etc.) without a compile-time error.

### Existing POC (already in the repo)

`geppetto/pkg/turns/poc_split_key_types_test.go` already proves the API shape we want:

- `DataKey[T]`, `TurnMetaKey[T]`, `BlockMetaKey[T]`
- constructors: `DataK`, `TurnMetaK`, `BlockMetaK`
- methods: `key.Get(store)` and `key.Set(&store, value)`

It currently delegates to the old function API, but it validates the language constraints and the intended call-site ergonomics.

### Migration tooling (already built)

`geppetto/pkg/analysis/turnsrefactor` and `geppetto/cmd/turnsrefactor` already implement Step A of migration:

- `turns.DataGet(store, key)` → `key.Get(store)`
- `turns.DataSet(&store, key, value)` → `key.Set(&store, value)`

The tool identifies targets by Go type information (`go/packages` + `types.Info`) and the `turns` package path.

## Target API (what we want)

### Store-specific key families

In `turns`:

- `type DataKey[T any] struct { id TurnDataKey }`
- `type TurnMetaKey[T any] struct { id TurnMetadataKey }`
- `type BlockMetaKey[T any] struct { id BlockMetadataKey }`

Each key type provides:

- `String() string`
- `Get(store)` and `Set(&store, value)` receiver methods

### Store-specific constructors

Replace `turns.K` with:

- `turns.DataK[T](namespace, value string, version uint16) DataKey[T]`
- `turns.TurnMetaK[T](...) TurnMetaKey[T]`
- `turns.BlockMetaK[T](...) BlockMetaKey[T]`

This makes “wrong store” usage impossible at compile time.

### Call-site ergonomics

Instead of:

```go
mode, ok, err := turns.DataGet(t.Data, turnkeys.ThinkingMode)
err := turns.DataSet(&t.Data, turnkeys.ThinkingMode, mode)
```

We want:

```go
mode, ok, err := turnkeys.ThinkingMode.Get(t.Data)
err := turnkeys.ThinkingMode.Set(&t.Data, mode)
```

## Implementation Plan (inside `turns`)

### Phase 1: Add key families + methods (while old API still exists)

1) Add the three key types to production code (likely new file: `geppetto/pkg/turns/key_families.go`):
   - `DataKey[T]`, `TurnMetaKey[T]`, `BlockMetaKey[T]`
2) Add constructors:
   - `DataK`, `TurnMetaK`, `BlockMetaK`
3) Implement receiver methods **directly against the wrapper maps** (not by calling the old functions).
   - Keep the current behavior contracts:
     - `Set` validates JSON serializability (`json.Marshal`)
     - `Get` returns `(zero, false, nil)` on missing
     - `Get` returns `(zero, true, err)` on type mismatch

### Phase 2: Migrate canonical keys (constructor rewrite)

Update canonical key definition files so call sites pick up the right key family types:

- `geppetto/pkg/turns/keys.go`
  - Data keys: `DataK[...]`
  - Turn metadata keys: `TurnMetaK[...]`
  - Block metadata keys: `BlockMetaK[...]`
- `geppetto/pkg/inference/engine/turnkeys.go`
  - `KeyToolConfig` becomes `turns.DataK[ToolConfig](...)`

Downstream (tracked here for completeness):
- `moments/backend/pkg/turnkeys/*` → switch to correct family constructors
- pinocchio local keys (e.g. sqlitetool) → switch to correct family constructors

### Phase 3: Rewrite call sites (function calls → key methods)

Run `geppetto/cmd/turnsrefactor` across the workspace (geppetto + downstream repos):

- rewrites `turns.{Data,Metadata,BlockMetadata}{Get,Set}` calls into `key.Get/key.Set`.
- run as dry-run first; then run with `-w`.

### Phase 4: Delete the old API (“no long-lived dual API”)

Once refactor tool is applied and `go test ./...` is green:

- delete `Key[T]` and `K[T]`
- delete `DataGet/DataSet`, `MetadataGet/MetadataSet`, `BlockMetadataGet/BlockMetadataSet`
- update `turnsrefactor` verify list (or accept that it becomes obsolete after migration)

## Lint updates (turnsdatalint)

We need linting rules that match the new world:

- Enforce canonical constructors in key-definition files:
  - allow `DataK/TurnMetaK/BlockMetaK` only in canonical key files (`pkg/turns/keys.go`, `moments/backend/pkg/turnkeys/*.go`, etc.)
- Ban ad-hoc key construction at usage sites:
  - `turns.DataK("mento", "foo", 1)` inline in middleware should be flagged
- Cross-family misuse becomes a compile-time error (lint not required for that).

Current state note: `turnsdatalint` has already been relaxed to typed-key enforcement (it rejects raw string literals/untyped string consts), but it does not yet enforce “canonical constructor-only usage”.

## Tricky / Open Questions (tracked, not solved here)

### Persistence strategy (typed reconstruction after YAML/JSON import)

The wrapper maps store `any`. After YAML import, complex values decode as `map[string]any` / `[]any`.

Options (not decided in this ticket, but affected by API choices):
- Keep `any` and rely on “typed at access time” + validation in setters (current behavior).
- Switch to `json.RawMessage` (or equivalent) for round-trip fidelity and typed decode at read time, with a YAML bridge for readability.
- Introduce a type registry (key → decoder) to reconstruct strong types after import.

## Validation / Review Checklist

- Ensure new key methods preserve existing error/ok semantics.
- Run `geppetto/cmd/turnsrefactor` (dry-run first) and confirm it rewrites only intended calls.
- Confirm canonical key definitions compile and downstream packages import the correct family types.
- Remove old API only after workspace tests are green.

**Suggested commands**

- `cd geppetto && go test ./... -count=1`
- `go run ./geppetto/cmd/turnsrefactor -- -packages ./...` (dry-run)
- `go run ./geppetto/cmd/turnsrefactor -- -w -packages ./...` (write)
