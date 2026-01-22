# oakgitdb: Usage guide (generate + query PR database)

## Goal

Provide copy/paste-ready commands and query recipes for generating and using the PR-focused SQLite database produced by `oakgitdb` for a target git repo (PR head vs base merge-base).

## Context

`oakgitdb` builds a single SQLite database that combines:

- **Git PR facts**: merge-base, commits in range, file changes (including renames/copies)
- **Oak matches** (tree-sitter): definitions extracted for both base+head snapshots
- **Go typed analysis** (head snapshot): declared symbols + call edges (best-effort, depends on type-check)

Default PR comparison:

- base: `git merge-base <base-ref> <head-sha>`
- head: `<head-ref>`

## Quick Reference

### Requirements

- `git` in PATH
- `oak` in PATH
- `sqlite3` CLI (optional, for ad-hoc querying)
- Go toolchain (and a working build environment for `github.com/mattn/go-sqlite3`)

### Build (recommended)

From `oak-git-db/`:

```bash
GOCACHE=/tmp/go-build-cache go run ./cmd/oakgitdb build \
  --repo /path/to/target-repo \
  --base origin/main \
  --head HEAD \
  --out /tmp/pr-vs-origin-main.db \
  --oak-sources cmd,pkg,misc \
  --oak-glob '*.go' \
  --packages ./...
```

Notes:

- `GOCACHE=/tmp/go-build-cache` avoids permission issues writing into `~/.cache/go-build` in some sandboxes.
- Oak warnings are on stderr; the tool handles this by parsing stdout JSON only.

### Build a smaller DB (faster iteration)

```bash
GOCACHE=/tmp/go-build-cache go run ./cmd/oakgitdb build \
  --repo /path/to/target-repo \
  --out /tmp/pr-small.db \
  --oak-sources pkg \
  --packages ./pkg/inference/...
```

### Inspect quickly

```bash
DB=/tmp/pr-vs-origin-main.db
sqlite3 -readonly "$DB" ".tables"
sqlite3 -readonly "$DB" "select key,value from meta;"
sqlite3 -readonly "$DB" "select id,name,ref,substr(sha,1,7) as sha7 from snapshot;"
```

### Common queries (copy/paste)

Use `pr_id=1` for now (current builder creates a single PR row).

#### What commits are in the PR range?

```bash
sqlite3 -readonly "$DB" <<'SQL'
SELECT pc.ord, substr(c.sha,1,7) AS sha7, c.committed_at, c.subject
FROM pr_commit pc
JOIN git_commit c ON c.sha = pc.sha
WHERE pc.pr_id = 1
ORDER BY pc.ord;
SQL
```

#### What files changed (and how)?

```bash
sqlite3 -readonly "$DB" <<'SQL'
SELECT p.path, pf.change_type, old.path AS old_path, pf.rename_score
FROM pr_file pf
JOIN path p ON p.id = pf.path_id
LEFT JOIN path old ON old.id = pf.old_path_id
WHERE pf.pr_id = 1
ORDER BY pf.change_type, p.path;
SQL
```

#### How many symbols/calls did typed analysis find?

```bash
sqlite3 -readonly "$DB" <<'SQL'
SELECT
  (SELECT count(*) FROM go_symbol) AS go_symbols,
  (SELECT count(*) FROM go_ref WHERE kind='call') AS call_edges,
  (SELECT count(*) FROM oak_match) AS oak_matches;
SQL
```

#### Find a symbol by name (head snapshot)

```bash
sqlite3 -readonly "$DB" <<'SQL'
WITH head AS (SELECT head_snapshot_id AS sid FROM pr WHERE id=1)
SELECT gs.symbol_key, gs.kind, gs.pkg_path, gs.recv, p.path, gs.start_line, gs.start_col
FROM go_symbol gs
LEFT JOIN path p ON p.id = gs.path_id
WHERE gs.snapshot_id = (SELECT sid FROM head)
  AND gs.name = 'NewSession'
ORDER BY gs.kind, gs.symbol_key
LIMIT 50;
SQL
```

#### “Who calls this function?” (callers)

1) First, look up the `symbol_key`:

```bash
sqlite3 -readonly "$DB" <<'SQL'
WITH head AS (SELECT head_snapshot_id AS sid FROM pr WHERE id=1)
SELECT gs.id, gs.symbol_key, gs.kind, gs.pkg_path, p.path, gs.start_line
FROM go_symbol gs
LEFT JOIN path p ON p.id = gs.path_id
WHERE gs.snapshot_id = (SELECT sid FROM head)
  AND gs.name = 'NewSession'
ORDER BY gs.id
LIMIT 20;
SQL
```

2) Then query callers by `to_symbol_id`:

```bash
sqlite3 -readonly "$DB" <<'SQL'
WITH head AS (SELECT head_snapshot_id AS sid FROM pr WHERE id=1),
target AS (
  SELECT id FROM go_symbol
  WHERE snapshot_id = (SELECT sid FROM head)
    AND symbol_key LIKE '%::func::%NewSession%'
  LIMIT 1
)
SELECT caller.symbol_key AS caller, p.path, r.line, r.col
FROM go_ref r
JOIN go_symbol caller ON caller.id = r.from_symbol_id
LEFT JOIN path p ON p.id = r.path_id
WHERE r.snapshot_id = (SELECT sid FROM head)
  AND r.kind = 'call'
  AND r.to_symbol_id = (SELECT id FROM target)
ORDER BY caller.symbol_key, p.path, r.line;
SQL
```

#### Search oak matches (definitions/snippets) in a file (base vs head)

```bash
sqlite3 -readonly "$DB" <<'SQL'
SELECT s.name AS snapshot, p.path, m.query, m.capture, m.start_row, m.start_col, m.text
FROM oak_match m
JOIN snapshot s ON s.id = m.snapshot_id
JOIN path p ON p.id = m.path_id
WHERE p.path = 'cmd/some-tool/main.go'
ORDER BY s.name, m.start_row, m.start_col
LIMIT 200;
SQL
```

## Usage Examples

### Example: “What changed and who is impacted?”

1) List changed Go files:

```bash
sqlite3 -readonly "$DB" <<'SQL'
SELECT p.path
FROM pr_file pf
JOIN path p ON p.id = pf.path_id
WHERE pf.pr_id = 1
  AND p.path LIKE '%.go'
ORDER BY p.path;
SQL
```

2) For each changed file, inspect oak matches (definition spans):

```bash
FILE='pkg/inference/session/session.go'
sqlite3 -readonly "$DB" "SELECT s.name,m.capture,m.start_row,m.text FROM oak_match m JOIN snapshot s ON s.id=m.snapshot_id JOIN path p ON p.id=m.path_id WHERE p.path='$FILE' ORDER BY s.name,m.start_row LIMIT 200;"
```

3) For key changed functions, find callers via `go_ref`:

- Find `symbol_key` by name, then query callers as shown above.

## Troubleshooting

### “Go typed analysis is empty / very small”

Symptoms:

- `select count(*) from go_symbol;` is unexpectedly low
- `select count(*) from go_ref;` is 0 or tiny

Causes:

- `go/packages` couldn’t fully type-check (missing build tags, platform issues, workspace constraints)

What to do:

- still use `oak_match` for syntax-level extraction (it’s type-check independent)
- narrow the package set:
  - `--packages ./pkg/inference/...`
- run with the target repo’s workspace/build environment set up as expected

### “The tool fails to write go build cache”

Run with:

```bash
GOCACHE=/tmp/go-build-cache
```

### “I passed a directory to oak and got no JSON”

Known sharp edge:

- `oak glaze go definitions <dir> --recurse ... --output json` produced no stdout output in practice.
- `oakgitdb` works around this by enumerating matching files and passing explicit file lists to oak in chunks.

## Related

- Design/spec: `planning/01-design-oak-git-history-sqlite-database-for-pr-vs-origin-main.md`
- Implementation details: `design-doc/01-oakgitdb-implementation-guide-pipeline-schema-extraction.md`
- Diary: `reference/01-diary.md`
