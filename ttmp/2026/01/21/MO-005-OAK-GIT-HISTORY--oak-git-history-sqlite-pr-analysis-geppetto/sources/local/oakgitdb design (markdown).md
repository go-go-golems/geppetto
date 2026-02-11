# Design: Oak + Git history → SQLite database for PR vs `origin/main`

## Executive summary

We want a **single SQLite database** that lets us answer questions like:

- “What symbols changed in this PR compared to `origin/main`?”
- “Who *uses* those symbols (calls, references, implements)?”
- “Which packages/files are impacted, and who last touched them?”
- “Show me the call graph neighborhood around a changed function.”

The database is built from **two ingredients**:

1) **Git history / diff facts** (PR base → PR head): commits, changed files, hunks, line stats, optional blame.
2) **Code facts** for two snapshots (base + head): symbols and relationships, from:
   - `oak` (tree-sitter) for fast, robust *syntax-level* extraction (definitions + spans)
   - a **custom Go analyzer** for richer *typed* graph edges (call sites, type refs, interface impls)

This doc specifies:

- the integration approach (git + oak + Go AST/types)
- a concrete SQLite schema (tables + key indexes)
- a query playbook (practical SQL recipes)
- an implementation plan + CLI design

## Context: scope and baseline

Target repo/worktree: any git repo. This design was initially validated against a Go monorepo-style setup, but the concepts generalize.

PR comparison:

- **Base ref:** `origin/main` (or more generally: `git merge-base origin/main HEAD`)
- **Head ref:** `HEAD` (current worktree state)

We treat the result as “PR vs `origin/main`” even when the merge-base differs, because PRs can be stacked/rebased.

## Goals and non-goals

### Goals

- Build a **rich, queryable** SQLite DB for PR analysis:
  - symbol inventory (base + head)
  - symbol relationships (at least for head; base optional in MVP)
  - PR file changes (status, rename/copy, insert/delete counts)
  - commit metadata for the PR range
- Make it reproducible and scriptable:
  - one command to generate DB (no manual steps)
  - deterministic schema and versioning
- Enable “who uses this symbol?” queries in seconds:
  - call sites for funcs/methods
  - type references
  - interface implementations (best-effort)

### Non-goals (initially)

- Perfect cross-module resolution for every external dependency symbol
- Full history indexing (all commits since repo genesis)
  - this repo focuses on PR/base-vs-head analysis (not “index everything since forever”)
- Whole-program interprocedural call graph accuracy (SSA/callgraph) in MVP
  - we start with “syntactic calls resolved by go/types” (good enough for most navigation)

## Tooling stack and responsibilities

### Git (authoritative for PR shape)

We use git to determine:

- base/head SHA and merge-base
- commit list in the PR range
- file-level change list with rename detection and line stats
- optional: diff hunks, blame for changed lines

### Oak (syntax-level extraction, fast and dependency-free)

We use `oak` for:

- extracting definitions with file spans (bytes + rows/cols)
- capturing raw definition text (signature snippets, optional body)
- being resilient even when the repo doesn’t type-check

Important operational detail:

- `oak glaze ... --output json` produces machine-readable JSON on **stdout**
- current oak emits deprecation warnings on **stderr** (must not be treated as JSON)

### Custom Go analyzer (typed symbol graph; “who uses what”)

We add a small Go CLI (under `cmd/oakgitdb/`) that uses:

- `golang.org/x/tools/go/packages` to load `./...`
- `go/ast` traversal + `types.Info` to:
  - enumerate declared symbols (defs)
  - record references (`Uses`) inside an “enclosing symbol” context
  - mark call edges from `*ast.CallExpr` to `types.Object`
  - (optional) compute interface implementation edges

Constraints:

- The geppetto repo is used in a larger `go.work` workspace; the analyzer should run in a way that respects that setup (avoid network fetches).

## Data model: snapshot-first

Everything in the DB is keyed by a **snapshot**:

- `base` snapshot = merge-base SHA (typically `origin/main`)
- `head` snapshot = PR head SHA (`HEAD`)

PR metadata (commits/diff) is keyed by a **PR record** that links the two snapshots.

This makes it straightforward to:

- compare symbol sets between snapshots
- add future snapshots (e.g., `HEAD^`, `main@{yesterday}`)
- compute deltas as materialized tables or views

## SQLite schema (proposed)

### Conventions

- Paths are stored **repo-relative** (`cmd/foo/main.go`), not absolute.
- Positions are stored with both:
  - `*_byte` offsets (oak-native)
  - `*_row`/`*_col` (oak-native, 0-based from oak; we store as-is and document it)
  - `*_line`/`*_col` (Go `token.Position`, 1-based)
- Symbols have **two IDs**:
  - `symbol_id` (INTEGER PK, internal)
  - `symbol_key` (TEXT UNIQUE, stable-ish key like `pkgPath::Kind::FullName`)

### Core tables

```sql
-- Schema versioning (so we can migrate safely)
CREATE TABLE meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE repo (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  root_path TEXT NOT NULL,
  remote_origin TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE snapshot (
  id INTEGER PRIMARY KEY,
  repo_id INTEGER NOT NULL,
  name TEXT NOT NULL,              -- 'base' | 'head'
  ref TEXT NOT NULL,               -- 'origin/main' | 'HEAD' (input)
  sha TEXT NOT NULL,               -- resolved commit sha
  created_at TEXT NOT NULL,
  FOREIGN KEY(repo_id) REFERENCES repo(id)
);

CREATE UNIQUE INDEX idx_snapshot_unique ON snapshot(repo_id, name);
CREATE INDEX idx_snapshot_sha ON snapshot(sha);
```

### PR tables (range + diff facts)

```sql
CREATE TABLE pr (
  id INTEGER PRIMARY KEY,
  repo_id INTEGER NOT NULL,
  base_snapshot_id INTEGER NOT NULL,
  head_snapshot_id INTEGER NOT NULL,
  merge_base_sha TEXT NOT NULL,
  base_ref TEXT NOT NULL,
  head_ref TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(repo_id) REFERENCES repo(id),
  FOREIGN KEY(base_snapshot_id) REFERENCES snapshot(id),
  FOREIGN KEY(head_snapshot_id) REFERENCES snapshot(id)
);

CREATE TABLE git_commit (
  sha TEXT PRIMARY KEY,
  parents TEXT,
  author_name TEXT,
  author_email TEXT,
  authored_at TEXT,
  committer_name TEXT,
  committer_email TEXT,
  committed_at TEXT,
  subject TEXT,
  body TEXT
);

CREATE TABLE pr_commit (
  pr_id INTEGER NOT NULL,
  sha TEXT NOT NULL,
  ord INTEGER NOT NULL,            -- 0..N in topo/chronological order
  PRIMARY KEY (pr_id, sha),
  FOREIGN KEY(pr_id) REFERENCES pr(id),
  FOREIGN KEY(sha) REFERENCES git_commit(sha)
);

CREATE TABLE path (
  id INTEGER PRIMARY KEY,
  path TEXT UNIQUE NOT NULL
);

CREATE TABLE pr_file (
  pr_id INTEGER NOT NULL,
  path_id INTEGER NOT NULL,        -- new/current path
  change_type TEXT NOT NULL,       -- A/M/D/R/C
  old_path_id INTEGER,             -- for renames/copies
  rename_score INTEGER,            -- for R/C entries, if available
  additions INTEGER,
  deletions INTEGER,
  PRIMARY KEY (pr_id, path_id),
  FOREIGN KEY(pr_id) REFERENCES pr(id),
  FOREIGN KEY(path_id) REFERENCES path(id),
  FOREIGN KEY(old_path_id) REFERENCES path(id)
);

CREATE INDEX idx_pr_file_change_type ON pr_file(pr_id, change_type);
```

### Oak extraction tables (raw, lossless)

We store raw `oak glaze ... --output json` matches (per capture).

```sql
CREATE TABLE oak_match (
  snapshot_id INTEGER NOT NULL,
  path_id INTEGER NOT NULL,
  query TEXT NOT NULL,
  capture TEXT NOT NULL,
  node_type TEXT,
  text TEXT,
  start_byte INTEGER,
  end_byte INTEGER,
  start_row INTEGER,
  start_col INTEGER,
  end_row INTEGER,
  end_col INTEGER,
  PRIMARY KEY (snapshot_id, path_id, query, capture, start_byte, end_byte),
  FOREIGN KEY(snapshot_id) REFERENCES snapshot(id),
  FOREIGN KEY(path_id) REFERENCES path(id)
);

CREATE INDEX idx_oak_match_lookup ON oak_match(snapshot_id, path_id, query, capture);
CREATE INDEX idx_oak_match_text ON oak_match(snapshot_id, text);
```

Notes:

- We keep this raw form because it’s stable and easy to re-derive higher-level “definition” records later via SQL views or a post-processing pass.

### Go analyzer tables (typed symbols + refs)

```sql
CREATE TABLE go_symbol (
  id INTEGER PRIMARY KEY,
  snapshot_id INTEGER NOT NULL,
  symbol_key TEXT NOT NULL,        -- stable-ish unique key
  kind TEXT NOT NULL,              -- func|method|type|var|const|field|pkg|...
  pkg_path TEXT,                   -- import path when known
  name TEXT NOT NULL,
  recv TEXT,                       -- receiver type for methods (optional)
  signature TEXT,                  -- best-effort human signature/type string
  path_id INTEGER,                 -- where it is declared (if in-repo)
  start_line INTEGER,
  start_col INTEGER,
  end_line INTEGER,
  end_col INTEGER,
  doc TEXT,
  is_exported INTEGER NOT NULL,
  is_external INTEGER NOT NULL,
  FOREIGN KEY(snapshot_id) REFERENCES snapshot(id),
  FOREIGN KEY(path_id) REFERENCES path(id)
);

CREATE UNIQUE INDEX idx_go_symbol_key ON go_symbol(snapshot_id, symbol_key);
CREATE INDEX idx_go_symbol_name ON go_symbol(snapshot_id, name);
CREATE INDEX idx_go_symbol_pkg ON go_symbol(snapshot_id, pkg_path);

CREATE TABLE go_ref (
  snapshot_id INTEGER NOT NULL,
  from_symbol_id INTEGER NOT NULL, -- enclosing symbol (usually a func/method)
  to_symbol_id INTEGER NOT NULL,   -- referenced symbol
  kind TEXT NOT NULL,              -- call|use|type|implements|embed|import
  path_id INTEGER,                 -- file where reference occurs
  line INTEGER,
  col INTEGER,
  PRIMARY KEY (snapshot_id, from_symbol_id, to_symbol_id, kind, path_id, line, col),
  FOREIGN KEY(snapshot_id) REFERENCES snapshot(id),
  FOREIGN KEY(from_symbol_id) REFERENCES go_symbol(id),
  FOREIGN KEY(to_symbol_id) REFERENCES go_symbol(id),
  FOREIGN KEY(path_id) REFERENCES path(id)
);

CREATE INDEX idx_go_ref_to ON go_ref(snapshot_id, to_symbol_id, kind);
CREATE INDEX idx_go_ref_from ON go_ref(snapshot_id, from_symbol_id, kind);
```

### Delta tables (computed)

We can compute deltas lazily via views, but for performance we can materialize:

```sql
CREATE TABLE symbol_delta (
  pr_id INTEGER NOT NULL,
  symbol_key TEXT NOT NULL,
  status TEXT NOT NULL,            -- added|removed|modified|unchanged
  base_symbol_id INTEGER,
  head_symbol_id INTEGER,
  PRIMARY KEY (pr_id, symbol_key),
  FOREIGN KEY(pr_id) REFERENCES pr(id),
  FOREIGN KEY(base_symbol_id) REFERENCES go_symbol(id),
  FOREIGN KEY(head_symbol_id) REFERENCES go_symbol(id)
);
```

In MVP, we can compute `symbol_delta` using the **Go analyzer only for head** and derive “added symbols” from:

- new paths in `pr_file` joined with `oak_match`/`go_symbol` in head

Later, we can fill `base_symbol_id` by running Go analyzer against base snapshot too.

## Example queries (the “why” of the schema)

### 1) What files changed in the PR?

```sql
SELECT p.path, pf.change_type, pf.additions, pf.deletions
FROM pr_file pf
JOIN path p ON p.id = pf.path_id
WHERE pf.pr_id = 1
ORDER BY pf.change_type, p.path;
```

### 2) List newly added Go symbols (head snapshot) in added files

```sql
SELECT gs.symbol_key, gs.kind, gs.pkg_path, gs.name, p.path
FROM pr_file pf
JOIN path p ON p.id = pf.path_id
JOIN go_symbol gs ON gs.path_id = p.id
WHERE pf.pr_id = 1
  AND pf.change_type = 'A'
  AND gs.snapshot_id = (SELECT head_snapshot_id FROM pr WHERE id = 1)
ORDER BY p.path, gs.kind, gs.name;
```

### 3) “Who calls function X?” (callers in head snapshot)

```sql
WITH target AS (
  SELECT id
  FROM go_symbol
  WHERE snapshot_id = (SELECT head_snapshot_id FROM pr WHERE id = 1)
    AND symbol_key = 'github.com/example/project/pkg/foo::func::NewThing'
)
SELECT caller.symbol_key AS caller, p.path, r.line, r.col
FROM go_ref r
JOIN go_symbol caller ON caller.id = r.from_symbol_id
JOIN path p ON p.id = r.path_id
WHERE r.snapshot_id = (SELECT head_snapshot_id FROM pr WHERE id = 1)
  AND r.kind = 'call'
  AND r.to_symbol_id = (SELECT id FROM target)
ORDER BY caller.symbol_key, p.path, r.line;
```

### 4) “What are the most-used changed symbols?”

```sql
SELECT d.status, callee.symbol_key, callee.kind, COUNT(*) AS call_count
FROM symbol_delta d
JOIN go_symbol callee ON callee.symbol_key = d.symbol_key
JOIN go_ref r ON r.to_symbol_id = callee.id
WHERE d.pr_id = 1
  AND d.status IN ('added','modified')
  AND r.kind = 'call'
GROUP BY d.status, callee.symbol_key, callee.kind
ORDER BY call_count DESC
LIMIT 50;
```

### 5) “Show the immediate call neighborhood around a symbol”

```sql
-- calls OUT of symbol
SELECT 'out' AS dir, callee.symbol_key, p.path, r.line
FROM go_ref r
JOIN go_symbol callee ON callee.id = r.to_symbol_id
JOIN path p ON p.id = r.path_id
WHERE r.snapshot_id = (SELECT head_snapshot_id FROM pr WHERE id = 1)
  AND r.kind = 'call'
  AND r.from_symbol_id = (
    SELECT id FROM go_symbol
    WHERE snapshot_id = (SELECT head_snapshot_id FROM pr WHERE id = 1)
      AND symbol_key = '...'
  )
UNION ALL
-- calls INTO symbol
SELECT 'in' AS dir, caller.symbol_key, p.path, r.line
FROM go_ref r
JOIN go_symbol caller ON caller.id = r.from_symbol_id
JOIN path p ON p.id = r.path_id
WHERE r.snapshot_id = (SELECT head_snapshot_id FROM pr WHERE id = 1)
  AND r.kind = 'call'
  AND r.to_symbol_id = (
    SELECT id FROM go_symbol
    WHERE snapshot_id = (SELECT head_snapshot_id FROM pr WHERE id = 1)
      AND symbol_key = '...'
  );
```

## CLI / pipeline design (implementation)

### Command shape

Proposed CLI (implemented in Go under `cmd/oakgitdb/`):

```bash
go run ./cmd/oakgitdb build \
  --repo . \
  --base origin/main \
  --head HEAD \
  --out /tmp/geppetto-pr.db
```

Optional flags:

- `--oak-sources cmd,pkg,misc` (default)
- `--oak-with-body=false` (default)
- `--packages ./...` (default)
- `--include-external=false` (default)
- `--write-symbol-delta=true` (default, head-only delta in MVP)

### Pipeline pseudocode

```text
resolve(baseRef, headRef):
  headSHA = git rev-parse headRef
  baseSHA = git merge-base baseRef headSHA

init_db(out):
  create schema + indexes
  insert repo, snapshots(base/head), pr row

ingest_git(pr):
  commits = git log baseSHA..headSHA
  diff = git diff --name-status -M --numstat baseSHA..headSHA
  insert git_commit, pr_commit, pr_file (+ path dimension)

ingest_oak(snapshot):
  if snapshot == base:
    extract repo at baseSHA into temp dir (git archive)
    run oak in that dir
  else:
    run oak in repo worktree
  oak_json = exec("oak glaze go definitions <sources> --recurse --glob '*.go' --output json", stderr->discard)
  insert oak_match rows

ingest_go(snapshot=head):
  packages.Load("./...") with go.work context
  for each package:
    for each file AST:
      walk AST with stack(enclosing func/method)
        record defs -> go_symbol
        record Uses + CallExpr -> go_ref(kind=use/call/type)

compute_deltas(pr):
  (MVP) derive symbol_delta for head-only from changed files + symbol presence
  (later) full base vs head diff by symbol_key
```

## Integration notes / sharp edges

- Oak warnings: emitted on stderr; always ignore stderr when parsing JSON output.
- Oak glaze + directories: in practice, `oak glaze ... <dir> --recurse ... --output json` produced no stdout output; pass explicit file lists (chunked) instead.
- Base snapshot extraction:
  - prefer `git archive` into a temp dir (no worktree mutation)
  - oak only needs the filesystem; no module metadata required
- Typed Go analysis:
  - the simplest reliable start is “head snapshot only”
  - base snapshot typed analysis requires reproducing module state at base:
    - either a worktree checkout *plus* go.work alignment
    - or a more advanced overlay strategy that also covers `go.mod`/`go.sum`

## How this relates to existing prior art

Some prior art (outside this repo) builds full-history SQLite DBs with regex-based symbol extraction. That’s useful for “commit timeline” queries, but this repo differs:

- PR-focused (base/head snapshots), not “all commits since forever”
- uses `oak` (tree-sitter) and Go typed analysis, not regex
- supports “who uses what” (call edges), not just “symbol existed in commit”

We can still copy ideas:

- commit metadata extraction format
- rename/copy parsing logic
- pragmatic indexes for symbol name lookups
