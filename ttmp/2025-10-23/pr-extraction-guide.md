# Guide: Carving Atomic PRs from `task/add-gpt5-responses-to-geppetto`

This playbook explains how to turn the current feature pile (responses API work, debug taps, llm-runner, event registry, generic tool executor) into a staircase of reviewable pull requests. It assumes you will clone a clean copy of `github.com/go-go-golems/geppetto`, then selectively replay the necessary pieces.

## 1. Preparation

1. **Fresh clone:**
   ```bash
   git clone git@github.com:go-go-golems/geppetto.git geppetto-clean
   cd geppetto-clean
   ```
2. **Reference materials:**
   - `ttmp/2025-10-23/git-history-and-code-index.db` — query with `sqlite3` to locate commits, changed files, and symbols.
   - `ttmp/2025-10-23/feature-history-timeline.md` — chronological narrative of the messy branch.
   - `ttmp/2025-10-23/git-history-index-guide.md` — reminders on regenerating/querying the DB.
3. **Decide extraction mode:** prefer file-scoped cherry-picks to avoid dragging unrelated edits and to minimise merge conflicts:
   - `git cherry-pick <commit> -- <path1> <path2>` applies a commit to specific files only.
   - `git show <commit> -- <path> | git apply` lets you handpick hunks when commits are noisy.
   - `git filter-repo` or `git filter-branch --subdirectory-filter` can isolate directories if you need a broader slice (e.g., everything under `pkg/inference/engine/`).
4. **Keep branches linear:** create a branch per PR (`pr/01-debug-taps`, `pr/02-event-registry`, etc.) and base each downstream branch on the previous PR branch so the stack lines up.

## 2. Mining the Index

For each PR:

- Use the DB to identify which commits introduce the files of interest:
  ```bash
  sqlite3 -readonly ttmp/2025-10-23/git-history-and-code-index.db <<'SQL'
  SELECT substr(c.hash,1,7) AS hash, c.subject
  FROM commit_files cf
  JOIN files f ON cf.file_id = f.id
  JOIN commits c ON cf.commit_id = c.id
  WHERE f.path LIKE 'pkg/inference/engine/%'
    AND c.committed_at BETWEEN '2025-10-21' AND '2025-10-23'
  ORDER BY c.committed_at;
  SQL
  ```
- Inspect `document_summary` for each candidate commit to confirm you are not pulling in unrelated assets (UI, docs, etc.).
- Drill into symbol-level data to spot refactors that must move together:
  ```bash
  sqlite3 -readonly ttmp/2025-10-23/git-history-and-code-index.db <<'SQL'
  SELECT substr(c.hash,1,7), f.path, cs.symbol_name, cs.symbol_kind
  FROM commit_symbols cs
  JOIN commits c ON cs.commit_id = c.id
  JOIN files f ON cs.file_id = f.id
  WHERE cs.symbol_name IN ('DebugTap', 'RegisterEventSource');
  SQL
  ```

## 3. Proposed PR Stack

Below is a suggested sequence that starts with debugging instrumentation (as requested) and layers the other features on top. Adjust as needed if reviewers prefer a different slicing.

### PR01 – Debug Tap Infrastructure
- **Scope:** Introduce `pkg/inference/engine/debugtap.go`, raw tap fixtures (`pkg/inference/fixtures/rawtap.go`), updates to `pkg/inference/fixtures/fixtures.go`, and the new `cmd/llm-runner` scaffold (minimal CLI + fixtures + README) without the full web UI.
- **Source commits:** Primarily `696aaa6` (debug taps) plus the prerequisite fixture commits (`e5cdac0`, `057410f`) for shared helpers. Verify each pulled change is required; omit e2e-runner deletions to keep history clean.
- **Extraction tips:**
  - Cherry-pick `e5cdac0` and `057410f` but restrict to the fixture paths (`git cherry-pick e5cdac0 -- pkg/inference/fixtures/fixtures.go`).
  - For `696aaa6`, cherry-pick individual directories (`-- pkg/inference/engine/debugtap.go pkg/inference/fixtures/rawtap.go cmd/llm-runner`) and ignore deletions of the old runner. Handle remaining hunks manually with `git apply`.

### PR02 – Event Registry Extensibility
- **Scope:** Add `pkg/events/registry.go`, extend `pkg/events/context.go` and `pkg/events/chat-events.go`, plus the necessary wiring in `pkg/inference/engine/debugtap.go` (now available from PR01).
- **Source commits:** `e130785` (event extensibility) but scrub out unrelated docs under `ttmp/2025-10-21/...` unless they are vital for reviewers.
- **Extraction tips:** Use `git cherry-pick e130785 -- pkg/events pkg/inference/engine/debugtap.go` to stay focused on runtime files. If docs are needed, place them in a separate commit/PR to reduce review noise.

### PR03 – Tool Executor Abstraction
- **Scope:** Make the tool executor generic (`pkg/inference/tools/base_executor.go`, updates to `pkg/inference/tools/executor.go`). Depends on event registry additions only at the API level.
- **Source commits:** `b21e6f9` (plus any lint/format adjustments from `eaa263f` limited to these files).
- **Extraction tips:** Cherry-pick with path filters (`git cherry-pick b21e6f9 -- pkg/inference/tools`) then run `gofmt`. If lint commits touched these files, reapply only the relevant hunks manually.

### PR04 – LLM Runner Web UI & Debugging Consoles
- **Scope:** Add `cmd/llm-runner/api.go`, `serve.go`, templates, and the TypeScript front-end under `cmd/llm-runner/web/…`.
- **Source commits:** `5908d75` (UI), `facd1b8` (correlation improvements). Combine into cohesive commits: one for backend API, one for front-end assets, one for fixtures/cassettes.
- **Extraction tips:**
  - Use `git cherry-pick <commit> -- cmd/llm-runner/api.go cmd/llm-runner/serve.go` first, then a second cherry-pick for `cmd/llm-runner/web`.
  - Handle cassette YAML separately to avoid churn. If you need everything under `cmd/llm-runner`, `git filter-repo --path cmd/llm-runner` (on a copy) can help craft patches without dragging the rest of the repo.

### PR05 – Cleanups and Lint Fixes
- **Scope:** Apply formatting and lint corrections that were bundled later (`eaa263f`, `fd0f919`), but only for paths touched in earlier PRs.
- **Approach:** After the stack is ready, run `make lint`. If additional adjustments are needed, add a dedicated “lint fixes” commit per PR or a single top-of-stack commit.

### Optional Documentation PRs
- Extract design docs from `ttmp/2025-10-17/**` and `ttmp/2025-10-21/**` into a separate “notes” PR so code reviewers are not overwhelmed.

## 4. Workflow per PR

1. **Create feature branch:** `git checkout -b pr/01-debug-taps main`
2. **Apply patches with filters:** rely on `git cherry-pick -- <paths>` or `git show <commit> -- <paths> | git apply` to keep each commit narrow.
3. **Format & lint:** run `make lint` (or `golangci-lint run ./...`) after each PR to ensure cleanliness.
4. **Author focused commit message:** summarise what the PR introduces; reference any design doc PRs for context.
5. **Push & open PR:** base each new PR branch on the previous PR branch to form a reviewable stack.

## 5. Double-Checking Dependencies

- Before finalising each PR, query the DB to ensure no missing pieces:
  ```bash
  sqlite3 -readonly ttmp/2025-10-23/git-history-and-code-index.db <<'SQL'
  SELECT DISTINCT f.path
  FROM commit_files cf
  JOIN files f ON cf.file_id = f.id
  JOIN commits c ON c.id = cf.commit_id
  WHERE c.subject LIKE '%debug%'
    AND c.committed_at BETWEEN '2025-10-21' AND '2025-10-22'
  ORDER BY f.path;
  SQL
  ```
- Compare against your staged changes in the clean repo; reconcile any intentional omissions (e.g., docs).

Following this guide gives you a crisp PR ladder that starts with low-level debug taps/events, builds developer tooling on top (LLM runner), and finishes with generic execution and cleanups—while avoiding merge conflicts by cherry-picking only the files you need.
