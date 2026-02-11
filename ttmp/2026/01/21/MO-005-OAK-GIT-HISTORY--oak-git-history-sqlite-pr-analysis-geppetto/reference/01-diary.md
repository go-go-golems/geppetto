---
Title: Diary
Ticket: MO-005-OAK-GIT-HISTORY
Status: active
Topics:
    - infrastructure
    - tools
    - geppetto
    - go
    - persistence
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/oakgitdb/main.go
      Note: MVP CLI implemented during Step 2
    - Path: geppetto/pkg/analysis/oakgitdb/builder.go
      Note: MVP DB builder implementation
    - Path: geppetto/ttmp/2025-10-23/build_history_index.py
      Note: Prior art referenced during investigation
    - Path: geppetto/ttmp/2026/01/21/MO-005-OAK-GIT-HISTORY--oak-git-history-sqlite-pr-analysis-geppetto/planning/01-design-oak-git-history-sqlite-database-for-pr-vs-origin-main.md
      Note: Design doc for the tool+schema being investigated
    - Path: geppetto/ttmp/2026/01/21/MO-005-OAK-GIT-HISTORY--oak-git-history-sqlite-pr-analysis-geppetto/various/pr-vs-origin-main.db
      Note: Step 2 generated DB artifact
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-21T16:32:08.103606297-05:00
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Build and iterate on a reproducible “PR vs `origin/main`” SQLite database for `geppetto/` that combines git history + symbol graphs (oak + Go AST/types), so we can answer “what changed” and “who uses it” questions quickly and precisely.

## Step 1: Scope, baselines, and prior art discovery

This step established the correct scope (“the PR in `geppetto/`”) and validated the baseline refs for comparison. It also surfaced existing in-repo prior art for git→sqlite indexing and clarified how to get machine-readable output from `oak` without log contamination.

The key outcome is a concrete direction: use git for PR shape, use `oak glaze ... --output json` for syntax-level extraction, and add a small Go analyzer for typed references (“who uses what”).

### What I did
- Confirmed the workspace root is not a git repo; the relevant worktree is `geppetto/`.
- Identified PR base/head:
  - `base`: `git merge-base origin/main HEAD` (currently equals `origin/main`)
  - `head`: `git rev-parse HEAD`
- Enumerated PR commit range and changed files:
  - `git log --oneline origin/main..HEAD`
  - `git diff --name-status origin/main..HEAD`
- Verified `oak` and `docmgr` availability and inspected oak’s machine-readable mode:
  - `oak glaze go definitions ... --output json`
  - observed oak prints warnings on **stderr**, JSON on **stdout**
  - validated that redirecting stderr keeps JSON parseable:
    - `oak glaze go definitions cmd/geppetto-lint/main.go --output json 2>/dev/null`
- Found in-repo prior art for git history indexing:
  - `geppetto/ttmp/2025-10-23/build_history_index.py`
  - `geppetto/ttmp/2025-10-23/git-history-index-guide.md`
  - `geppetto/ttmp/2025-10-23/pr-extraction-guide.md`
- Created the docmgr ticket workspace and initial docs:
  - `MO-005-OAK-GIT-HISTORY`
  - planning doc + diary doc

### Why
- The root workspace contains multiple repos; symbol/history indexing must match the PR’s actual repo (`geppetto/`) to avoid invalid conclusions.
- Oak is valuable for syntax-level extraction even when type-checking is flaky, but we need a clean JSON channel for ingestion.
- Prior art gives a concrete example of git→sqlite ingestion patterns (commit metadata, renames, symbol index) that we can reuse or supersede.

### What worked
- `git merge-base origin/main HEAD` gives a reliable base SHA for “PR vs main” comparisons.
- `oak glaze ... --output json` provides match objects including byte offsets and rows/cols (good for stable spans).
- Oak’s warnings are safely isolatable (stderr), so a builder can parse stdout as JSON without heuristics.

### What didn't work
- Attempting to reason about the “PR vs origin/main” at the workspace root failed because there is no `.git/` there (multi-worktree setup).
- Oak still emits deprecation warnings even with `--log-level error`; suppressing them requires handling stderr (not log level flags).

### What I learned
- Oak’s “glaze” mode is the right integration point because it supports structured output (`--output json`) and field selection.
- The repo already contains a “git history index” concept, but it’s commit-history-centric and regex-based; we need a PR-centric, typed symbol graph.

### What was tricky to build
- Snapshot fidelity vs workspace ergonomics:
  - base snapshot analysis must not require network dependency fetches (go.work matters)
  - oak can analyze extracted trees cheaply; Go typed analysis needs module/workspace correctness

### What warrants a second pair of eyes
- The proposed “symbol_key” stability strategy for Go symbols (receiver formatting, type string normalization) will affect delta quality and query UX.
- Deciding whether/when to store large blobs (full function bodies, full patches) in SQLite vs store hashes + paths.

### What should be done in the future
- Implement MVP DB generator:
  - git PR facts (commits, file changes)
  - oak matches for base + head snapshots
  - Go typed symbols + refs for head snapshot
- Add base snapshot typed analysis if/when module/workspace reproducibility is solved cleanly (likely via worktree + go.work alignment or a stronger overlay strategy).

### Code review instructions
- Start with the planning doc:
  - `geppetto/ttmp/2026/01/21/MO-005-OAK-GIT-HISTORY--oak-git-history-sqlite-pr-analysis-geppetto/planning/01-design-oak-git-history-sqlite-database-for-pr-vs-origin-main.md`
- Validate the foundational assumptions:
  - `cd geppetto && git merge-base origin/main HEAD`
  - `cd geppetto && oak glaze go definitions cmd/geppetto-lint/main.go --output json 2>/dev/null | head`

### Technical details

Commands used (representative):

```bash
cd geppetto
git rev-parse origin/main
git merge-base origin/main HEAD
git log --oneline origin/main..HEAD
git diff --name-status origin/main..HEAD

oak glaze go definitions cmd/geppetto-lint/main.go --output json 2>/dev/null
```

## Step 2: MVP generator CLI + first PR database

This step implemented an MVP Go CLI (`cmd/oakgitdb`) that generates a PR-focused SQLite database by combining (1) git range metadata, (2) oak-extracted definition matches for both base+head snapshots, and (3) Go typed symbol + call-edge extraction for the head snapshot.

The key outcome is a concrete DB artifact stored in the ticket workspace (`various/pr-vs-origin-main.db`) that can already answer practical questions like “what files changed?”, “what are the call edges we can discover in head?”, and “show me oak spans for definitions in base vs head”.

### What I did
- Implemented a Go package to build the DB:
  - `pkg/analysis/oakgitdb/*`
- Added a cobra CLI entrypoint:
  - `cmd/oakgitdb/main.go`
- Implemented DB ingestion steps:
  - Git: merge-base, commit list, name-status diff (renames/copies)
  - Oak: extract a snapshot with `git archive`, then run `oak glaze ...` on the extracted tree
  - Go: `go/packages` load and record declared symbols + call edges (head snapshot)
- Discovered an oak integration sharp edge and implemented a workaround:
  - `oak glaze ...` emits JSON for *file arguments*, but produces no output when given directories (even with `--recurse`)
  - Workaround: expand directories to a list of matching files in Go, then call oak in chunks (200 files per invocation)
- Generated the first full PR DB:
  - `geppetto/ttmp/.../MO-005-OAK-GIT-HISTORY.../various/pr-vs-origin-main.db`

### Why
- We need a database we can query repeatedly while iterating on schema and analysis logic, without re-deriving everything mentally or via ad-hoc grep.
- Using oak for base snapshot avoids needing a type-checking environment for historical code.
- Using Go typed analysis for head snapshot enables “who calls what?” queries that are hard to answer reliably with regex or tree-sitter alone.

### What worked
- `git archive`-based snapshot extraction keeps the analysis reproducible without mutating the worktree.
- The “expand directories to files” workaround makes oak ingestion stable and parseable (JSON on stdout; warnings on stderr).
- The MVP DB stays reasonably small (single-digit MB) while still being useful:
  - includes ~1.5k Go symbols and ~2k call edges for head snapshot with `./...`.

### What didn't work
- `oak glaze <command> <dir> --recurse ... --output json` produced *no stdout output* in practice.
  - This broke the initial “just let oak recurse” plan.
  - The workaround is implemented, but this seems like an oak/glaze limitation/bug worth upstream investigation.
- While updating the ticket changelog, including backticks in the shell command caused zsh command-substitution (the referenced paths were executed as commands). Fix: avoid backticks in shell arguments.

### What I learned
- For oak integration, “pass files, not directories” is the reliable path today when using glaze JSON output.
- Even a head-only typed graph is useful if combined with PR file change facts; base typed analysis can be deferred until module/workspace reproduction is solved cleanly.

### What was tricky to build
- Making oak output ingestion robust:
  - avoid stderr contamination
  - handle directory recursion not working in glaze mode
  - chunk file lists to avoid command line limits
- Choosing a symbol identity scheme that’s stable enough for queries but not overly complicated.

### What warrants a second pair of eyes
- Call-edge attribution:
  - currently attributes calls to the enclosing `*ast.FuncDecl` only (package-level otherwise)
  - nested function literals aren’t represented as distinct symbols yet
- Symbol key normalization for methods:
  - receiver formatting (`types.TypeString`) can affect “diffability” across snapshots and query ergonomics

### What should be done in the future
- Add additional edge kinds (type refs, func-as-value refs, const/var refs) with size controls.
- Add a “symbol delta” materialization pass (base vs head) for Go symbols.
- Consider adding optional PR hunks + blame ingestion for “who last touched the impacted lines?” queries.

### Code review instructions
- Start with:
  - `geppetto/pkg/analysis/oakgitdb/builder.go` (schema + pipeline)
  - `geppetto/cmd/oakgitdb/main.go` (CLI flags and usage)
- Smoke-run locally:
  - `cd geppetto`
  - `GOCACHE=/tmp/go-build-cache go run ./cmd/oakgitdb build --out /tmp/pr.db --oak-sources cmd,pkg,misc --packages ./...`
- Inspect DB quickly:
  - `sqlite3 -readonly /tmp/pr.db ".tables"`
  - `sqlite3 -readonly /tmp/pr.db "select count(*) from go_ref;"`

### Technical details

Representative commands used:

```bash
cd geppetto
GOCACHE=/tmp/go-build-cache go run ./cmd/oakgitdb build \
  --repo . \
  --base origin/main \
  --head HEAD \
  --out /tmp/pr.db \
  --oak-sources cmd,pkg,misc \
  --oak-glob '*.go' \
  --packages ./...

sqlite3 -readonly /tmp/pr.db ".tables"
sqlite3 -readonly /tmp/pr.db "select count(*) from oak_match;"
sqlite3 -readonly /tmp/pr.db "select count(*) from go_symbol;"
sqlite3 -readonly /tmp/pr.db "select count(*) from go_ref;"
```

## Step 3: Extract oakgitdb into standalone repo `oak-git-db/`

This step moved the MVP `oakgitdb` code and its detailed documentation out of `geppetto/` into a standalone repo directory (`oak-git-db/`) so it can evolve independently (own module, own docs, reusable across repos).

The ticket workspace remains as a “home base” for the investigation, but the source of truth for the tool is now the `oak-git-db/` repo.

### What I did
- Moved code with `mv`:
  - `geppetto/cmd/oakgitdb/` → `oak-git-db/cmd/oakgitdb/`
  - `geppetto/pkg/analysis/oakgitdb/` → `oak-git-db/pkg/oakgitdb/`
- Moved docs with `mv` into `oak-git-db/docs/` and left stubs in the ticket:
  - `oak-git-db/docs/usage.md`
  - `oak-git-db/docs/implementation.md`
  - `oak-git-db/docs/design.md`
- Added `oak-git-db/go.mod` and ran `go mod tidy`.
- Updated workspace `go.work` to include `./oak-git-db`.
- Updated the ticket index to point to the new repo location.

### Why
- Keeping the tool inside `geppetto/` was convenient for bootstrapping but makes reuse and iteration harder.
- A standalone repo is a better long-term home for a code-navigation/indexing tool.

### What worked
- `mv`-based extraction preserved history within the workspace and made the move mechanical.
- Adding `./oak-git-db` to `go.work` keeps local `go test`/`go run` ergonomics.

### What didn't work
- N/A

### What I learned
- In zsh, unquoted backticks in shell arguments will trigger command substitution; avoid backticks in docmgr CLI invocations.

### What was tricky to build
- Keeping docmgr ticket integrity after moving docs:
  - solved by leaving stub docs in the ticket that point to the new repo files

### What warrants a second pair of eyes
- Module path choice for `oak-git-db` (`go.mod`): confirm the intended canonical import path.

### What should be done in the future
- Consider adding CI/build scripts in `oak-git-db/`.
- Add the next schema/features (symbol deltas, hunks/blame) in the standalone repo.

### Code review instructions
- Start here:
  - `oak-git-db/pkg/oakgitdb/builder.go`
  - `oak-git-db/cmd/oakgitdb/main.go`
  - `oak-git-db/docs/usage.md`
  - `oak-git-db/docs/implementation.md`

### Technical details

Representative commands used:

```bash
cd /home/manuel/workspaces/2025-10-30/implement-openai-responses-api
mv geppetto/cmd/oakgitdb oak-git-db/cmd/oakgitdb
mv geppetto/pkg/analysis/oakgitdb oak-git-db/pkg/oakgitdb

cd oak-git-db
GOCACHE=/tmp/go-build-cache go mod tidy
GOCACHE=/tmp/go-build-cache go test ./... -count=1
```

## Step 4: Multi-repo DB support (geppetto + pinocchio in one SQLite file)

This step extended `oakgitdb` so it can index multiple repo roots into a single SQLite database. The goal is to enable cross-repo review/navigation workflows (for example, “this new `geppetto` symbol is used by `pinocchio`”) without manually juggling multiple DB files.

The key changes were: make `--repo` repeatable, namespace commit/path facts by `repo_id`, and generate one `pr` row per repo inside the same DB.

**Commit (oak-git-db):** b6e0313 — "feat: multi-repo build + repo-namespaced schema"

### What I did
- Updated CLI:
  - `--repo` is now a repeatable string-slice flag (`--repo ../geppetto --repo ../pinocchio`).
- Updated schema (schema_version=2):
  - `git_commit` primary key is now `(repo_id, sha)` to avoid cross-repo hash collisions.
  - `pr_commit` now carries `repo_id` and joins to `git_commit` on `(repo_id, sha)`.
  - `path` is now unique on `(repo_id, path)`.
- Updated ingestion:
  - the builder loops over `RepoDirs` and inserts one `repo` + `snapshot(base/head)` + `pr` per repo.
  - all path creation is now scoped by `repo_id`.
- Updated docs to show multi-repo usage and PR selection:
  - `oak-git-db/README.md`
  - `oak-git-db/docs/usage.md`
  - `oak-git-db/docs/design.md`

### Why
- We already have PR work split across repos (e.g., geppetto vs pinocchio). Without multi-repo DB support, reviewers must maintain multiple DBs and can’t run cross-repo queries.

### What worked
- Multi-repo build works in practice and produces multiple PR rows:
  - `pr` row 1 for `geppetto`, `pr` row 2 for `pinocchio` (example run)
- Cross-repo-safe schema approach (repo_id scoping) keeps joins straightforward.

### What didn't work
- N/A

### What I learned
- Git commit SHAs can collide across unrelated repos in theory; scoping commit identity by `repo_id` is the simplest robust fix.

### What was tricky to build
- Making `path` and commit identity multi-repo safe without duplicating every downstream table.
- Keeping single-repo behavior unchanged while adding multi-repo capability.

### What warrants a second pair of eyes
- Schema evolution strategy:
  - currently we gate with `meta.schema_version` and require rebuilding older DBs; confirm this is acceptable long-term.
- Whether `go_symbol` should carry an explicit `repo_id` column (it’s currently derivable via `snapshot_id -> repo_id`).

### What should be done in the future
- Add example cross-repo SQL recipes to the docs (e.g., “pinocchio call edges that target geppetto symbol keys” via imported module paths).
- Consider adding optional “repo alias” flag for nicer display names in queries.

### Code review instructions
- In `oak-git-db/`, start with:
  - `oak-git-db/pkg/oakgitdb/builder.go`
  - `oak-git-db/cmd/oakgitdb/main.go`
  - `oak-git-db/docs/usage.md`

### Technical details

Smoke run (multi-repo):

```bash
cd oak-git-db
rm -f /tmp/multi-pr.db
GOCACHE=/tmp/go-build-cache go run ./cmd/oakgitdb build \
  --repo ../geppetto \
  --repo ../pinocchio \
  --base origin/main \
  --head HEAD \
  --out /tmp/multi-pr.db \
  --oak-sources cmd,pkg,misc \
  --oak-glob '*.go' \
  --packages ./...
```
