# Oak + Git history → SQLite PR analysis (geppetto)

This is the document workspace for ticket MO-005-OAK-GIT-HISTORY.

## Structure

- **design/**: Design documents and architecture notes
- **reference/**: Reference documentation and API contracts
- **playbooks/**: Operational playbooks and procedures
- **scripts/**: Utility scripts and automation
- **sources/**: External sources and imported documents
- **various/**: Scratch or meeting notes, working notes
- **archive/**: Optional space for deprecated or reference-only artifacts

## Getting Started

Use docmgr commands to manage this workspace:

- Add documents: `docmgr doc add --ticket MO-005-OAK-GIT-HISTORY --doc-type design-doc --title "My Design"`
- Import sources: `docmgr import file --ticket MO-005-OAK-GIT-HISTORY --file /path/to/doc.md`
- Update metadata: `docmgr meta update --ticket MO-005-OAK-GIT-HISTORY --field Status --value review`

## Generate / Regenerate the DB

From the `geppetto/` repo root:

```bash
GOCACHE=/tmp/go-build-cache go run ./cmd/oakgitdb build \
  --repo . \
  --base origin/main \
  --head HEAD \
  --out ttmp/2026/01/21/MO-005-OAK-GIT-HISTORY--oak-git-history-sqlite-pr-analysis-geppetto/various/pr-vs-origin-main.db \
  --oak-sources cmd,pkg,misc \
  --oak-glob '*.go' \
  --packages ./...
```

## Quick Queries

```bash
DB=ttmp/2026/01/21/MO-005-OAK-GIT-HISTORY--oak-git-history-sqlite-pr-analysis-geppetto/various/pr-vs-origin-main.db
sqlite3 -readonly "$DB" ".tables"
sqlite3 -readonly "$DB" "select id,name,ref,substr(sha,1,7) from snapshot;"
sqlite3 -readonly "$DB" "select change_type,count(*) from pr_file group by change_type order by change_type;"
sqlite3 -readonly "$DB" "select count(*) from go_symbol;"
sqlite3 -readonly "$DB" "select count(*) from go_ref;"
```
