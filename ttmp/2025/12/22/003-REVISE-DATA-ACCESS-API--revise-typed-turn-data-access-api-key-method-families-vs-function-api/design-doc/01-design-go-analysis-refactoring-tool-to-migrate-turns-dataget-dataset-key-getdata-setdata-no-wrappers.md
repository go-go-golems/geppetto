---
Title: 'Design: refactoring tool to migrate turns.DataGet/DataSet/... -> key-method API (no wrappers)'
Ticket: 003-REVISE-DATA-ACCESS-API
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/turns/types.go
      Note: Current API that callsites use (`DataGet/DataSet/...`) and that the tool targets.
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Existing analyzer pattern (x/tools) we will mirror for packaging/entrypoint style.
    - Path: geppetto/cmd/turnsdatalint/main.go
      Note: singlechecker entrypoint pattern we may reuse if we choose analyzer+driver.
    - Path: geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md
      Note: Updated final design describing the new key-method API.
ExternalSources: []
Summary: "CLI refactoring tool that rewrites turns.{Data,Metadata,BlockMetadata}{Get,Set} calls into key-method calls in one pass (no transitional wrappers)."
LastUpdated: 2025-12-22T00:00:00-05:00
WhatFor: "Provide a safe, repeatable, one-shot migration path from function-based typed accessors to the new key-method API across geppetto/moments/pinocchio."
WhenToUse: "Use right after the new key-method API lands in code; run once to rewrite call sites + fix imports, then remove the old accessor functions."
---

# Design: refactoring tool to migrate turns.DataGet/DataSet/... -> key-method API (no wrappers)

## Executive Summary

We need an *all-at-once* refactor that replaces every call site of:

- `turns.DataGet/DataSet`
- `turns.MetadataGet/MetadataSet`
- `turns.BlockMetadataGet/BlockMetadataSet`

with the new **key-method API**:

- `key.Get(store)` / `key.Set(&store, value)`

Because the desired end state has **no transitional wrappers**, we cannot migrate incrementally in code review by keeping both APIs for long. Instead, we will build a CLI refactoring tool that:

1. Loads packages using `golang.org/x/tools/go/packages`
2. Identifies these function calls by **type resolution** (not string matching)
3. Rewrites the AST to key-method calls
4. Runs import+format cleanup (via `golang.org/x/tools/imports`)
5. Optionally verifies that no targeted calls remain (fail fast)

This keeps the migration mechanical, reviewable, and repeatable.

## Problem Statement

The 002 work introduced typed accessors as generic functions (`DataGet/DataSet/...`) due to Go’s restriction that methods cannot declare their own type parameters.

We now want the ergonomic, store-specific key API described in the updated final design:

- `DataKey[T]` / `TurnMetaKey[T]` / `BlockMetaKey[T]`
- `key.Get(store)` and `key.Set(&store, value)` methods

We want to migrate *all call sites at once* with:

- minimal human error
- minimal style drift
- no intermediate API shims that linger

Hand-editing hundreds of call sites across `geppetto/`, `moments/`, and `pinocchio/` is high risk and high cost. Pure regex-based rewriting is brittle because of alias imports, formatting, and edge cases.

## Proposed Solution

### Tool form

Add a new CLI tool:

- `geppetto/cmd/turnsrefactor` (name bikesheddable)

It runs against one or more package patterns (default `./...`) and rewrites files in-place (`-w`) or prints a diff summary (`--dry-run`).

### Rewrite rules

All rewrites are driven by *resolved* function identity from `types.Info`, not by spellings.

#### Data

- `turns.DataGet(dataExpr, keyExpr)` → `keyExpr.Get(dataExpr)`
- `turns.DataSet(dataPtrExpr, keyExpr, valueExpr)` → `keyExpr.Set(dataPtrExpr, valueExpr)`

#### Turn.Metadata

- `turns.MetadataGet(metaExpr, keyExpr)` → `keyExpr.Get(metaExpr)`
- `turns.MetadataSet(metaPtrExpr, keyExpr, valueExpr)` → `keyExpr.Set(metaPtrExpr, valueExpr)`

#### Block.Metadata

- `turns.BlockMetadataGet(blockMetaExpr, keyExpr)` → `keyExpr.Get(blockMetaExpr)`
- `turns.BlockMetadataSet(blockMetaPtrExpr, keyExpr, valueExpr)` → `keyExpr.Set(blockMetaPtrExpr, valueExpr)`

Notes:

- The tool does **not** need to know whether `keyExpr` is a `DataKey` vs `TurnMetaKey` at rewrite time; compilation after landing the new key types will catch category mismatches. (Long-term we can enhance the tool to detect store/key type mismatches and fail early.)
- The rewrite preserves the original store expressions (e.g. `t.Data`, `&t.Data`, `b.Metadata`, `&cloned.Metadata`) and value expressions.

### Import + format cleanup

After rewriting a file, run:

- `imports.Process(filename, newSrc, &imports.Options{FormatOnly: false, Comments: true})`

This auto-fixes unused imports (e.g. if `turns` import becomes unused after removing `turns.DataGet` references) and formats output.

### Safety rails

- **Dry run by default**: only write files when `-w` is provided.
- **Fail if nothing changed** unless `--allow-noop` is passed (useful in CI).
- **Fail if any targeted calls remain** after rewrite (optional `--verify` flag enabled by default in CI mode).
- **Summary report**: number of packages, files touched, replacements done per rule.

### Scope

The first version is narrowly scoped to rewriting `turns.*Get/*Set` calls. It intentionally does not:

- migrate key constructors (`K[...]` → `DataK/TurnMetaK/BlockMetaK`)
- migrate direct map access (already handled by 002 wrappers)
- implement the new key families in production code (separate change)

Those can be separate rewrite passes or extensions later.

## Design Decisions

### Use type resolution, not string matching

Call expressions are matched by resolving the selected function via `types.Info.Uses` to:

- package path `github.com/go-go-golems/geppetto/pkg/turns`
- object name `DataGet`, etc.

This handles:

- aliased imports (`turns2.DataGet`)
- dot imports (we can choose to reject or support)
- local helper functions named `DataGet` (won’t match)

### `go/packages` driver instead of `singlechecker`

`singlechecker` is ideal for analyzers emitting diagnostics, but it doesn’t apply edits. For a rewrite tool we need a driver that:

- loads packages
- rewrites ASTs
- writes files

So we use `packages.Load` directly (still in the x/tools ecosystem).

### No wrappers / no dual API period (migration posture)

This tool assumes we will:

1. Land the new key-method API in code.
2. Run this tool once to rewrite call sites.
3. Immediately delete the old function API (`DataGet/DataSet/...`).

That’s the “no wrappers” posture: we avoid carrying both APIs.

## Alternatives Considered

### Regex / sed rewrite

Rejected: too brittle (alias imports, multiline calls, comments, formatting, generics, etc.).

### `gopls` code action only

Rejected (for now): code actions are great, but we need a CLI that can run in CI and guarantee “all at once”.

### Keep both APIs and migrate by hand over time

Rejected: explicitly against the “no wrappers and all that” requirement.

## Implementation Plan

- [ ] Implement `geppetto/cmd/turnsrefactor` CLI:
  - flags: `-w`, `--dry-run`, `--verify`, `--packages` (default `./...`)
- [ ] Implement package loader and rewrite engine:
  - `packages.Load` with `NeedSyntax|NeedTypes|NeedTypesInfo|NeedFiles|NeedModule`
  - per-file AST traversal, collect replacements, reprint file
- [ ] Apply `imports.Process` on changed files
- [ ] Verify no `turns.*Get/*Set` calls remain (if enabled)
- [ ] Add a small fixture test (optional) using golden files (input → expected output)
- [ ] Document usage in ticket 003 diary + changelog

Example usage:

```bash
cd /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access && \
go run ./geppetto/cmd/turnsrefactor -- -w --packages ./...
```

## Open Questions

- Do we need to support dot-imports of `turns`? (Simplest: detect and error.)
- Should we support rewriting key constructor calls too (`turns.K` → `turns.DataK/TurnMetaK/BlockMetaK`), or keep that a separate tool pass?
- Should verification include a full `go test ./...` run, or keep it as a separate CI step?

## References

- Ticket 002 diary (ground truth for current API): `geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/reference/01-diary.md`
- Updated final design doc (new API): `geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md`
- Ticket 003 analysis (restart vs refactor): `geppetto/ttmp/2025/12/22/003-REVISE-DATA-ACCESS-API--revise-typed-turn-data-access-api-key-method-families-vs-function-api/analysis/01-analysis-revise-turn-data-metadata-access-api-key-methods-migration-strategy.md`
