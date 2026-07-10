---
Title: Investigation Diary
Ticket: HIST-2026-03-14-TO-2026-06-14
Status: active
Topics:
    - git-history
    - docmgr
    - research
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: ttmp/2026/06/14/HIST-2026-03-14-TO-2026-06-14--geppetto-history-for-last-three-months/scripts/01_build_history_db.py
      Note: Diary records implementation and validation of the database builder
    - Path: ttmp/2026/06/14/HIST-2026-03-14-TO-2026-06-14--geppetto-history-for-last-three-months/scripts/03_render_history_report.py
      Note: Diary records the report aggregation fix
ExternalSources: []
Summary: Chronological diary for building the git/docmgr SQLite history ticket.
LastUpdated: 2026-06-14T12:55:00-04:00
WhatFor: Records the commands, scripts, failures, and validation steps used to create the last-three-months history.
WhenToUse: Use when regenerating, reviewing, or extending the history report.
---


# Diary

## Goal

Capture how the last-three-months Geppetto history was built from git and docmgr evidence, with reusable scripts and SQL stored in the ticket workspace.

## Step 1: Create a docmgr-backed SQLite history package

I created a new docmgr ticket for the requested history, then built a reproducible SQLite-based pipeline inside the ticket. The pipeline extracts git commits, commit file stats, docmgr ticket metadata, document metadata, and changelog entries for the window beginning 2026-03-14.

The result is a ticket-local history bundle: scripts and SQL live under `scripts/`, the SQLite database lives under `various/history.sqlite`, raw SQL output lives in `various/01-history-summary.md`, and the final narrative report lives in `analysis/01-last-three-months-history.md`.

### Prompt Context

**User prompt (verbatim):** "Using git and docmgr, storing in sqlite, make me a history of the last 3 months. Store in a new docmgr ticket. 

Store all scripts and sql queries in the scripts/ folder. "

**Assistant interpretation:** Create a new docmgr ticket containing a reproducible history report for the last three months, backed by SQLite data built from git and docmgr sources.

**Inferred user intent:** The user wants a durable, inspectable project history that can be regenerated and queried, not just an ad-hoc prose summary.

**Commit (code):** N/A — no git commit was created during this turn.

### What I did
- Ran `docmgr status --summary-only` to confirm the docmgr root and workspace state.
- Created ticket `HIST-2026-03-14-TO-2026-06-14` with title `Geppetto history for last three months`.
- Added `analysis/01-last-three-months-history.md` and `reference/01-investigation-diary.md`.
- Wrote `scripts/01_build_history_db.py` to build `various/history.sqlite` from git and docmgr files.
- Wrote `scripts/02_summary_queries.sql` with reusable SQLite aggregate queries.
- Wrote `scripts/03_render_history_report.py` to render the final Markdown report from SQLite.
- Ran the scripts to generate the SQLite database, raw query output, and final report.
- Added missing vocabulary topics with `docmgr vocab add` after `docmgr doctor` reported unknown topic values.
- Renamed raw query output from `history-summary.md` to `01-history-summary.md` and added frontmatter so docmgr validation can treat it as a document.

### Why
- Git alone gives commit chronology but misses the rationale captured in docmgr tickets and diaries.
- Docmgr alone gives planning and investigation context but misses concrete commit-level activity.
- SQLite provides a stable intermediate representation that can be queried, audited, and regenerated.
- Keeping scripts and SQL in `scripts/` makes the report reproducible and follows the user's storage constraint.

### What worked
- Git extraction successfully loaded 525 commits for 2026-03-14 through 2026-06-14.
- Commit file statistics loaded into a separate `commit_files` table, enabling path hot-spot queries.
- Docmgr ticket scanning loaded 52 tickets and 383 markdown documents dated in the same window.
- The rendered report now includes monthly git activity, category breakdowns, top touched paths, docmgr activity, large tickets, recent commits, interpretation, and reproduction commands.

### What didn't work
- First `docmgr doctor --ticket HIST-2026-03-14-TO-2026-06-14 --stale-after 30` reported unknown topics:
  - `docmgr`
  - `git-history`
  - `research`
- The same doctor run also reported that `various/history-summary.md` had no frontmatter and no numeric prefix:
  - `frontmatter delimiters '---' not found`
  - `missing_numeric_prefix`
- The first rendered docmgr activity-by-month table accidentally counted joined document rows as tickets. I fixed `scripts/03_render_history_report.py` to aggregate per ticket first, then month.

### What I learned
- Docmgr validation scans markdown outputs in ticket subdirectories, so generated Markdown artifacts need either proper frontmatter or a non-doc extension/location.
- Counting tickets over a `LEFT JOIN` requires a per-ticket subquery or `COUNT(DISTINCT t.ticket)`; otherwise document counts inflate ticket counts.
- The last-three-month history is strongly documentation-heavy: the extracted docmgr corpus has roughly 840k words in the same period as the git commits.

### What was tricky to build
- The main sharp edge was reconciling filesystem-based docmgr ticket dates with git commit dates. The script uses the dated ticket path (`ttmp/YYYY/MM/DD/...`) as the ticket date and git author dates for commit chronology.
- YAML frontmatter parsing was intentionally kept simple and dependency-free. The script extracts common scalar and list fields well enough for report metadata, but it is not a full YAML parser.
- Merge commits can make insertion/deletion totals look large because `git show --numstat` reports combined changes for the merge commit representation. The report keeps merge commits visible as their own category rather than hiding them.

### What warrants a second pair of eyes
- The heuristic commit categorizer in `classify()` is subject-line based and intentionally approximate.
- The report narrative is generated from aggregate evidence, but finer historical claims should be checked against the linked commits and large docmgr tickets.
- The docmgr scanner uses path dates, not `LastUpdated`, so it answers “tickets created in the last three months” rather than “all docs edited in the last three months.”

### What should be done in the future
- Optionally add a second SQLite table for `git log --follow` or file rename tracking if path continuity matters.
- Optionally enrich commit categories with ticket references, PR numbers, or conventional-commit parsing.
- Optionally add a query that joins commits touching `ttmp/` paths back to the corresponding docmgr ticket.

### Code review instructions
- Start with `scripts/01_build_history_db.py` and inspect the schema plus `load_git()` / `load_docmgr()`.
- Then inspect `scripts/02_summary_queries.sql` for the reusable SQL summaries.
- Finally inspect `scripts/03_render_history_report.py` and the generated `analysis/01-last-three-months-history.md`.
- Validate with:
  - `python3 scripts/01_build_history_db.py --repo ../../../../.. --since 2026-03-14 --until 2026-06-14 --db various/history.sqlite`
  - `sqlite3 various/history.sqlite < scripts/02_summary_queries.sql > various/01-history-summary.md`
  - `python3 scripts/03_render_history_report.py --db various/history.sqlite --out analysis/01-last-three-months-history.md --since 2026-03-14 --until 2026-06-14`
  - `docmgr doctor --ticket HIST-2026-03-14-TO-2026-06-14 --stale-after 30`

### Technical details
- SQLite tables created: `commits`, `commit_files`, `docmgr_tickets`, `docmgr_docs`, `changelog_entries`.
- Main database path: `various/history.sqlite`.
- Raw SQL output path: `various/01-history-summary.md`.
- Final report path: `analysis/01-last-three-months-history.md`.
