# oakgitdb: Implementation guide (pipeline, schema, extraction)

## Executive Summary

`oakgitdb` is a Go CLI that produces a PR-focused SQLite database combining:

- **git**: the PR range as (merge-base of base ref vs head ref), plus commits and file changes
- **oak**: tree-sitter-based extraction of Go definitions for **both** base and head snapshots
- **go/packages + go/types**: typed symbol inventory and **call edges** for the head snapshot (best-effort)

The output DB is intended for interactive exploration with SQL (“what changed?” and “who uses it?”) and for feeding downstream tooling (UIs, reports, agents).

## Problem Statement

Given a PR (or local feature branch), we want to answer:

- What files changed, including renames/copies?
- What symbols were introduced/modified/removed?
- Who calls a changed function, and what does that function call?

Existing approaches (grep, ripgrep, regex symbol inventories) are fast but shallow; purely typed approaches can be brittle if the repo doesn’t type-check. `oakgitdb` intentionally combines both.

## Proposed Solution

### High-level pipeline

The builder (`pkg/oakgitdb`) executes the following phases:

1) Resolve refs:
   - `headSHA = git rev-parse <headRef>`
   - `baseSHA = git merge-base <baseRef> <headSHA>`
2) Create SQLite schema + metadata rows:
   - `repo`, `snapshot(base)`, `snapshot(head)`, `pr`
3) Ingest git PR facts:
   - commits in `<baseSHA>..<headSHA>` into `git_commit` + `pr_commit`
   - file changes with rename detection into `path` + `pr_file`
4) Ingest oak matches for each snapshot:
   - extract snapshot tree via `git archive <sha> | tar` (no checkout mutation)
   - enumerate matching files (`--oak-sources` + `--oak-glob`)
   - run `oak glaze go definitions <file...> --output json` in chunks
   - store match rows into `oak_match`
5) Ingest Go typed facts for head snapshot:
   - load packages via `golang.org/x/tools/go/packages`
   - store declared symbols into `go_symbol`
   - store call edges into `go_ref(kind='call')`

### Code layout

- `cmd/oakgitdb/main.go`
  - Cobra CLI wrapper
  - flag parsing into `oakgitdb.BuildOptions`
- `pkg/oakgitdb/builder.go`
  - schema DDL (`schemaSQL`)
  - `Build(...)` orchestration
  - `ingestGitPR`, `ingestOakSnapshot`, `ingestGoSnapshot`
  - oak file enumeration + chunking (`collectMatchingFiles`, `runOakDefinitions`)
- `pkg/oakgitdb/git.go`
  - `gitString`, `gitBytes`
  - snapshot extraction via `git archive` (`extractGitTree`)
- `pkg/oakgitdb/gomod.go`
  - module path extraction (used to tag/filter external symbols)

## Design Decisions

### Snapshot extraction via `git archive` (not worktrees)

We avoid:

- creating temporary worktrees
- messing with the user’s current checkout

Instead we run:

```bash
git archive --format=tar <sha> | tar -x -C <tmpdir>
```

This is fast and deterministic and works for both base and head snapshots.

### Oak integration: pass file lists, not directories

In practice, `oak glaze ... <dir> --recurse ... --output json` produced no stdout output. To make oak ingestion reliable:

- we enumerate matching files ourselves
- we pass explicit file lists to oak
- we chunk those lists (default 200 files per invocation) to avoid command-line limits

### Typed analysis: “best effort”, head snapshot only (for now)

Typed Go analysis is valuable for “who calls what?” but can fail if the repo doesn’t type-check.

This repo currently:

- runs typed analysis for the **head** snapshot only
- continues even when `packages.PrintErrors(pkgs) > 0` (results may be partial)

Base snapshot typed analysis is intentionally deferred; it requires more careful reproduction of historical module/workspace state.

### Symbol identity: `symbol_key`

We store a normalized “lookup key” (`go_symbol.symbol_key`) to make joins and ad-hoc querying easy:

- functions: `<pkgPath>::func::<FullName>`
- methods: `<pkgPath>::method::<RecvTypeString>.<Name>`
- types/vars/consts: `<pkgPath>::<kind>::<Name>`

This is not perfect but it’s pragmatic and query-friendly.

## Schema and what it enables

The schema lives inline as DDL in `pkg/oakgitdb/builder.go` (`schemaSQL`).

Key tables:

- `snapshot` / `pr`: identify base/head snapshots
- `pr_commit` / `git_commit`: PR commit list and metadata
- `pr_file` + `path`: changed paths with rename/copy metadata
- `oak_match`: raw oak captures with spans for base/head
- `go_symbol`: typed symbol inventory (head)
- `go_ref`: typed edges (currently calls only; head)

Important constraint today:

- `pr_file.additions`/`deletions` are currently not populated (we only ingest name-status); extend `ingestGitPR` with `git diff --numstat -z` to fill this.

## Extending the tool

### Add more reference kinds (beyond calls)

Current edge extraction is:

- `call`: `ast.CallExpr` resolved via `types.Info.Uses`

Recommended next edges:

- `use`: other identifier uses (value refs), potentially very large
- `type`: type references (composites, conversions, embedded fields)
- `implements`: interface implementation edges (best computed per package/type set)

Implementation sketch:

- add new `go_ref.kind` values and produce them in `ingestGoSnapshot`
- add size controls / filters:
  - only for changed files (`pr_file`)
  - only within selected directories
  - or cap edges per “from symbol”

### Add PR hunks + blame

To answer “who last touched the impacted lines”, ingest:

- `git diff --unified=0 --patch -- <paths>` (or parse `--word-diff=porcelain`)
- `git blame --porcelain <baseSHA>..<headSHA> -- <file>`

Store:

- hunk ranges
- line ownership (author email + commit)

## Implementation Plan

Suggested incremental roadmap:

1) Fill `pr_file.additions/deletions` via `--numstat` parsing
2) Add optional base snapshot typed analysis (behind a flag) using:
   - temporary worktree checkout, or
   - `go work use` overlay strategy
3) Add `go_ref.kind='use'` with filters
4) Add `symbol_delta` materialization (base vs head) when base typed symbols are available

## Open Questions

- What module path should this repo use in production (currently `github.com/go-go-golems/oak-git-db`)?
- Should we switch to a pure-Go SQLite driver to avoid CGO requirements?
- What should be the default include/exclude policy for “external” symbols in call edges?

## References

- `docs/design.md`
- `docs/usage.md`

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
