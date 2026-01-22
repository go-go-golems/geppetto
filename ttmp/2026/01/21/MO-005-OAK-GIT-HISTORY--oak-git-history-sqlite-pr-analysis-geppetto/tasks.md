# Tasks

## TODO

- [x] Draft integration + schema design doc
- [x] Implement MVP generator CLI (`cmd/oakgitdb`)
- [x] Generate PR DB for `geppetto/` vs `origin/main`
- [ ] Add “symbol delta” materialization (base vs head) for Go symbols
- [ ] Add non-call references (type refs, func-as-value, const/var refs) with size controls
- [ ] Add interface implementation edges (best-effort; likely cached)
- [ ] Add PR hunk ingestion + optional blame for changed lines

## Multi-repo DB (geppetto + pinocchio)

- [x] Extend `oakgitdb build` to accept multiple `--repo` roots (repeatable flag or CSV)
- [x] Schema: namespace all facts by repo (add `repo_id` everywhere or per-repo table namespaces)
- [ ] Decide ID strategy for `path`:
  - [x] Option A: `(repo_id, path)` unique (recommended)
  - [ ] Option B: global `path` with `repo_id` column on all referencing tables
- [ ] Decide ID strategy for snapshots/PRs:
  - [x] One “PR per repo” (multiple `pr` rows in one DB)
  - [ ] Or a “multi-repo PR bundle” row that links multiple per-repo PRs
- [ ] Update symbol identity to be cross-repo safe:
  - [ ] Add `repo_id` dimension to `go_symbol`
  - [ ] Ensure `symbol_key` uniqueness is `(repo_id, snapshot_id, symbol_key)`
- [ ] Update ingestion pipeline:
  - [x] Loop repos: resolve refs, compute merge-base, ingest git/oak/go per repo
  - [ ] Store per-repo metadata (name, root path, origin URL)
- [ ] Add example cross-repo queries:
  - [ ] “Find callers in pinocchio for symbols implemented/changed in geppetto”
  - [ ] “Which packages across repos import/use shared module paths?”
- [ ] Backward compatibility:
  - [x] Confirm single-repo mode still works unchanged
  - [x] Add migration/version bump (`meta.schema_version`)
