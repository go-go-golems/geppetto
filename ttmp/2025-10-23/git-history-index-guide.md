# Git History & Code Index Guide

This repository snapshot includes a reusable SQLite database that mirrors the git history of the current branch (`task/add-gpt5-responses-to-geppetto`). The database lives at `ttmp/2025-10-23/git-history-and-code-index.db` and captures commit metadata, per-file change details, and lightweight symbol extraction for Go/Python/TS/JS files. This document explains how the database was produced and how to query it effectively.

## How the Database Is Produced

- **Script location:** `ttmp/2025-10-23/build_history_index.py`
- **Invocation:** From the repo root, run `python ttmp/2025-10-23/build_history_index.py`
- **Runtime behaviour:**
  - Deletes any existing DB at `ttmp/2025-10-23/git-history-and-code-index.db`.
  - Walks every commit reachable from `HEAD` in chronological order (`git rev-list --reverse HEAD`).
  - For each commit, records metadata (hash, parents, author/committer info, subject/body, timestamps).
  - Collects file change details via `git diff-tree` (`--name-status` and `--numstat -M`) so renames/copies are tracked with scores and old paths.
  - Extracts symbols by parsing the committed file content. Supported suffixes: `.go`, `.py`, `.ts`, `.tsx`, `.js`.
  - Stores a JSON “document summary” blob describing added/modified/deleted/renamed/copied paths per commit.
  - Prints progress every 25 commits processed.
- **Dependencies:** Python 3.11+, standard library (`sqlite3`, `subprocess`, `re`, etc.), and Git available on PATH. No third-party packages required.

If future commits land, rerun the script to regenerate the DB in place. Because the script always rebuilds from scratch, no manual cleanup is necessary.

## Database Schema Overview

The script initialises the following tables:

- `commits` — one row per commit, including metadata and `document_summary` (JSON text).
- `files` — deduplicated file paths touched in history.
- `commit_files` — join table describing how each file changed in a commit (`change_type`, `old_path`, `additions`, `deletions`).
- `commit_symbols` — symbol names/“kinds” discovered in the file snapshot for a given commit.

Indexes are present on commit/file foreign keys and symbol names to keep lookups fast.

## Querying the Database

The examples below use `sqlite3` from the repo root. Feel free to swap in any SQLite client.

### Inspect a Commit Summary
```bash
sqlite3 -readonly -cmd '.mode column' -cmd '.headers on' \
  ttmp/2025-10-23/git-history-and-code-index.db \
  "SELECT substr(hash,1,7) AS hash7, committed_at, subject, document_summary FROM commits WHERE hash LIKE '696aaa6%';"
```
Returns the timestamp, subject, and JSON summary (grouped by added/modified/deleted/etc.) for the commit introducing debug taps.

### List Files Touched by a Commit
```bash
sqlite3 -readonly ttmp/2025-10-23/git-history-and-code-index.db <<'SQL'
SELECT f.path, cf.change_type, cf.old_path, cf.additions, cf.deletions
FROM commit_files cf
JOIN files f ON cf.file_id = f.id
WHERE cf.commit_id = (
  SELECT id FROM commits WHERE hash LIKE '696aaa6%'
);
SQL
```
Shows per-file diff stats for the chosen commit, including rename information.

### Search for a Symbol Definition
```bash
sqlite3 -readonly ttmp/2025-10-23/git-history-and-code-index.db <<'SQL'
SELECT substr(c.hash,1,7) AS hash7, c.committed_at, f.path, cs.symbol_kind
FROM commit_symbols cs
JOIN commits c ON cs.commit_id = c.id
JOIN files f ON cs.file_id = f.id
WHERE cs.symbol_name = 'DebugTap'
ORDER BY c.committed_at;
SQL
```
Finds commits where a given symbol (e.g., `DebugTap`) was present and lists the file and symbol kind.

### Walk Commit History Chronologically
```bash
sqlite3 -readonly ttmp/2025-10-23/git-history-and-code-index.db <<'SQL'
SELECT substr(hash,1,7) AS hash7, committed_at, subject
FROM commits
ORDER BY committed_at;
SQL
```
Produces a concise timeline suitable for spotting feature clusters or cherry-picking ranges.

## Extending or Reusing the Index

- **Different branch:** Check out the target branch and rerun the script; the DB will now reflect that branch’s history.
- **Partial histories:** If you want to limit scope, adjust `build_history_index.py` (e.g., swap `git rev-list HEAD` for `git rev-list --max-count=...` or pass an explicit range).
- **Additional symbol types:** Extend `SYMBOL_REGEXES` at the top of the script with new file extensions and regexes.
- **Alternative output path:** Edit `DB_PATH` near the top of the script before running.

This guide, the database, and the generating script live under `ttmp/2025-10-23/` so they can be versioned alongside analysis artifacts.
