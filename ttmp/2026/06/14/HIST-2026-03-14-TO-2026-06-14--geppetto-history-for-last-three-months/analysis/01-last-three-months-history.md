---
Title: Last Three Months History
Ticket: HIST-2026-03-14-TO-2026-06-14
Status: active
Topics:
    - git-history
    - docmgr
    - research
DocType: analysis
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - path: /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/06/14/HIST-2026-03-14-TO-2026-06-14--geppetto-history-for-last-three-months/scripts/01_build_history_db.py
      why: Builds the SQLite history database from git and docmgr sources
    - path: /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/06/14/HIST-2026-03-14-TO-2026-06-14--geppetto-history-for-last-three-months/scripts/02_summary_queries.sql
      why: Reusable SQL summaries for the history database
    - path: /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/06/14/HIST-2026-03-14-TO-2026-06-14--geppetto-history-for-last-three-months/scripts/03_render_history_report.py
      why: Renders the Markdown history report from SQLite
    - path: /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/06/14/HIST-2026-03-14-TO-2026-06-14--geppetto-history-for-last-three-months/various/history.sqlite
      why: Generated SQLite database backing the report
    - path: /home/manuel/code/wesen/go-go-golems/geppetto/ttmp/2026/06/14/HIST-2026-03-14-TO-2026-06-14--geppetto-history-for-last-three-months/various/01-history-summary.md
      why: Raw query output from the summary SQL
ExternalSources: []
Summary: "Textbook-style SQLite-backed report on Geppetto work from 2026-03-14 through 2026-06-14."
LastUpdated: 2026-06-14T13:10:00-04:00
WhatFor: "Use this report to understand what changed in Geppetto over the last three months and how the history was derived from git and docmgr data."
WhenToUse: "Use when onboarding to recent project direction, planning follow-up work, or auditing the evidence behind the repository history."
---

# Last Three Months of Geppetto Work

## 1. What this report is for

A useful project history should do more than list commits. A commit stream tells us what changed, but it often leaves out why the work happened, how the work clustered, and which parts of the system absorbed the most design pressure. Docmgr provides the missing layer: tickets, diaries, design notes, changelogs, and research documents. This report combines both sources into a single SQLite database, then uses SQL to ask repeatable questions about the last three months of Geppetto work.

The period covered here is **2026-03-14 through 2026-06-14**. In that window, Geppetto moved through a dense sequence of provider work, runtime integration, JavaScript API expansion, multimodal input support, embeddings exposure, and documentation-heavy design work. The repository did not merely accumulate features. It accumulated operating knowledge: provider compatibility notes, failure diaries, smoke tests, design records, and migration plans.

By the end of this report, the reader should understand three things:

- **What changed:** the main areas of implementation and documentation work.
- **Where the work concentrated:** the files, tickets, and subsystems touched most often.
- **How to reproduce the history:** the SQLite schema and SQL queries that produced the evidence.

## 2. The data model

The history database is deliberately small. It is not a general analytics warehouse; it is a local evidence store for one report. The builder script reads git for commit-level facts and scans `ttmp/` for docmgr ticket facts. The two sources remain separate tables because they describe different kinds of activity.

```text
git log ────────┐
                ├── commits ────────┐
git numstat ────┘                    ├── SQL summaries ─── Markdown report
docmgr ttmp/ ───── docmgr_tickets ───┤
markdown docs ──── docmgr_docs ──────┘
changelogs ─────── changelog_entries
```

The key design choice is to preserve the original grain of the evidence. A commit remains a commit. A touched path remains a touched path. A ticket remains a ticket. A Markdown file remains a document. This matters because aggregation is then an explicit SQL operation rather than an irreversible preprocessing decision.

The SQLite database contains these tables:

| Table | Grain | Purpose |
| --- | --- | --- |
| `commits` | One row per git commit | Stores commit date, author, subject, change size, merge flag, and heuristic category. |
| `commit_files` | One row per commit/path pair | Stores per-file insertions and deletions from `git show --numstat`. |
| `docmgr_tickets` | One row per ticket directory | Stores ticket id, path date, title, status, topics, and summary. |
| `docmgr_docs` | One row per Markdown document | Stores title, doc type, status, topics, last-updated timestamp, and word count. |
| `changelog_entries` | One row per bullet entry | Stores ticket changelog bullets for later narrative reconstruction. |

This schema keeps the report honest. If a number appears in the report, it can be traced to a table and a query. If the classification is approximate, the report says so.

## 3. Executive summary

The database contains **525 commits** in the covered period: **466 non-merge commits** and **59 merge commits**. Those commits touched **4,702 file entries**, with **513,044 insertions** and **85,013 deletions** reported by git. On the docmgr side, the same date window contains **52 tickets**, **383 Markdown documents**, and roughly **840,705 words** of written project memory.

Those numbers are important, but their interpretation is more important. Geppetto's recent work was not one linear feature branch. It was a set of overlapping efforts:

- Provider correctness and compatibility work across OpenAI, OpenAI Responses, Claude, Gemini, and OpenAI-compatible providers.
- JavaScript and Goja API expansion, including module tests, generated TypeScript definitions, event vocabulary work, and runtime ownership concerns.
- Multimodal and image input support, with normalization helpers, provider mappings, and smoke coverage.
- Reasoning and streaming support, especially around OpenAI Responses reasoning replay, Claude extended thinking, Gemini thought signatures, and streaming event shape.
- Embedding profile and JavaScript exposure work, including later fixes to preserve embedding-local API settings.
- Documentation and research as first-class work products, visible both in commit subjects and in the size of docmgr tickets.

A concise reading is this: **the last three months were about making Geppetto more capable as a provider/runtime platform while making its behavior inspectable enough to survive rapid change.**

## 4. Overall activity

```sql
SELECT
  COUNT(*) AS commits,
  SUM(is_merge) AS merge_commits,
  SUM(CASE WHEN is_merge = 0 THEN 1 ELSE 0 END) AS non_merge_commits,
  SUM(files_changed) AS files_changed,
  SUM(insertions) AS insertions,
  SUM(deletions) AS deletions,
  MIN(author_day) AS first_day,
  MAX(author_day) AS last_day
FROM commits;
```

This first query is intentionally simple. It establishes the boundary of the data set before any interpretation begins.

| commits | merge_commits | non_merge_commits | files_changed | insertions | deletions | first_day | last_day |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 525 | 59 | 466 | 4702 | 513044 | 85013 | 2026-03-14 | 2026-06-06 |

The last git commit in the window is 2026-06-06, even though the report window ends on 2026-06-14. That means the final week of the report window contributes docmgr ticket state but no later git commits in this checkout.

## 5. Monthly rhythm

```sql
SELECT
  substr(author_day, 1, 7) AS month,
  COUNT(*) AS commits,
  SUM(files_changed) AS files_changed,
  SUM(insertions) AS insertions,
  SUM(deletions) AS deletions
FROM commits
GROUP BY month
ORDER BY month;
```

| month | commits | files_changed | insertions | deletions |
| --- | --- | --- | --- | --- |
| 2026-03 | 200 | 2027 | 159282 | 51106 |
| 2026-04 | 40 | 179 | 9595 | 1606 |
| 2026-05 | 207 | 1603 | 203020 | 14442 |
| 2026-06 | 78 | 893 | 141147 | 17859 |

March and May are the two heavy implementation months by commit count. April is much quieter. June has fewer commits than March or May, but a large insertion count, which reflects the presence of sizeable documentation and feature work early in the month.

The important lesson is that commit count and changed lines measure different things. A month can have many small corrective commits, or fewer commits that introduce large docs, generated files, or broad refactors. The report therefore keeps both dimensions visible.

## 6. Work categories

The builder assigns each commit a heuristic category from its subject line. This is not a semantic classifier. It is a practical first pass that lets us see the shape of the work without reading all 525 commits one by one.

```sql
SELECT
  category,
  COUNT(*) AS commits,
  SUM(insertions) AS insertions,
  SUM(deletions) AS deletions
FROM commits
GROUP BY category
ORDER BY commits DESC;
```

| category | commits | insertions | deletions |
| --- | --- | --- | --- |
| docs/research | 152 | 137517 | 4008 |
| maintenance | 151 | 48034 | 41398 |
| features | 69 | 62473 | 975 |
| merge | 59 | 250542 | 37204 |
| fixes | 47 | 6292 | 535 |
| dependencies | 30 | 457 | 360 |
| tests | 17 | 7729 | 533 |

Two categories dominate by commit count: `docs/research` and `maintenance`. That pairing is revealing. The project was not only adding new capabilities; it was explaining, stabilizing, and reshaping them. The `features` category is smaller by commit count but still substantial by insertions. The `fixes` category is also meaningful: many feature and provider changes were followed by targeted correctness patches.

Merge commits have large line totals because git reports a merge's effective diff representation. For interpretation, they should be read as integration events rather than as direct authored changes of the same kind as normal commits.

## 7. Monthly category breakdown

```sql
SELECT
  substr(author_day, 1, 7) AS month,
  category,
  COUNT(*) AS commits,
  SUM(files_changed) AS files_changed,
  SUM(insertions) AS insertions,
  SUM(deletions) AS deletions
FROM commits
GROUP BY month, category
ORDER BY month, commits DESC;
```

| month | category | commits | insertions | deletions |
| --- | --- | --- | --- | --- |
| 2026-03 | maintenance | 71 | 22064 | 24507 |
| 2026-03 | docs/research | 56 | 28402 | 2910 |
| 2026-03 | features | 31 | 29832 | 230 |
| 2026-03 | merge | 18 | 76466 | 23246 |
| 2026-03 | fixes | 15 | 2425 | 110 |
| 2026-03 | dependencies | 9 | 93 | 103 |
| 2026-04 | merge | 9 | 4658 | 727 |
| 2026-04 | maintenance | 8 | 2102 | 627 |
| 2026-04 | docs/research | 8 | 476 | 155 |
| 2026-04 | dependencies | 7 | 121 | 53 |
| 2026-04 | fixes | 4 | 293 | 21 |
| 2026-04 | features | 3 | 1839 | 12 |
| 2026-04 | tests | 1 | 106 | 11 |
| 2026-05 | docs/research | 60 | 63604 | 616 |
| 2026-05 | maintenance | 59 | 15616 | 7207 |
| 2026-05 | merge | 26 | 100019 | 5483 |
| 2026-05 | fixes | 22 | 3016 | 374 |
| 2026-05 | features | 19 | 18405 | 414 |
| 2026-05 | tests | 11 | 2138 | 164 |
| 2026-05 | dependencies | 10 | 222 | 184 |
| 2026-06 | docs/research | 28 | 45035 | 327 |
| 2026-06 | features | 16 | 12397 | 319 |
| 2026-06 | maintenance | 13 | 8252 | 9057 |
| 2026-06 | merge | 6 | 69399 | 7748 |
| 2026-06 | fixes | 6 | 558 | 30 |
| 2026-06 | tests | 5 | 5485 | 358 |
| 2026-06 | dependencies | 4 | 21 | 20 |

The monthly breakdown gives the history a visible rhythm. March looks like broad restructuring and feature work. May looks like a second surge, with documentation and research as a central part of the implementation process. June is narrower but still intense: provider polish, image input, Claude/Gemini reasoning behavior, Goja runtime flags, embeddings exposure, and JavaScript-facing integration all appear in the recent commits.

## 8. Hot paths in the codebase

A frequently touched file is not automatically a problematic file. It may be a stable integration point, a generated artifact, or the natural place where an API boundary is evolving. Still, hot paths deserve attention because they show where change repeatedly lands.

```sql
SELECT
  path,
  COUNT(*) AS commits,
  SUM(COALESCE(insertions, 0)) AS insertions,
  SUM(COALESCE(deletions, 0)) AS deletions
FROM commit_files
GROUP BY path
ORDER BY commits DESC, insertions + deletions DESC
LIMIT 30;
```

| path | commits | insertions | deletions |
| --- | --- | --- | --- |
| go.mod | 60 | 239 | 203 |
| go.sum | 57 | 551 | 431 |
| pkg/steps/ai/openai/engine_openai.go | 37 | 804 | 736 |
| pkg/steps/ai/openai_responses/engine.go | 35 | 551 | 2411 |
| pkg/js/modules/geppetto/module_test.go | 32 | 1702 | 4844 |
| ttmp/vocabulary.yaml | 31 | 124 | 4 |
| pkg/steps/ai/openai_responses/engine_test.go | 30 | 2160 | 212 |
| pkg/doc/types/geppetto.d.ts | 29 | 725 | 969 |
| pkg/js/modules/geppetto/module.go | 29 | 359 | 343 |
| pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl | 27 | 723 | 811 |
| pkg/steps/ai/openai_responses/streaming.go | 26 | 2982 | 2506 |
| Makefile | 25 | 170 | 93 |
| pkg/steps/ai/gemini/engine_gemini.go | 24 | 415 | 997 |
| pkg/js/modules/geppetto/api_engines.go | 24 | 393 | 901 |
| pkg/steps/ai/openai_responses/helpers.go | 23 | 701 | 253 |
| pkg/doc/topics/13-js-api-reference.md | 22 | 1101 | 1443 |
| pkg/steps/ai/claude/engine_claude.go | 21 | 227 | 95 |

The hot paths identify the main technical centers of gravity:

- `pkg/steps/ai/openai_responses/*` shows that OpenAI Responses support, reasoning, helpers, streaming, and tests were heavily revised.
- `pkg/js/modules/geppetto/*` and the generated TypeScript definitions show an active JavaScript API surface.
- `pkg/steps/ai/gemini/engine_gemini.go` and `pkg/steps/ai/claude/engine_claude.go` show targeted provider-specific work.
- `ttmp/vocabulary.yaml` appears because docmgr was not passive; the documentation system itself evolved as new ticket topics were introduced.
- `go.mod` and `go.sum` show frequent dependency movement, though each dependency commit is usually small.

## 9. The docmgr layer

Docmgr gives the commit history its second dimension. A commit subject can say `Gemini: preserve function call thought signatures`; a docmgr ticket can explain why thought signatures matter, how provider behavior differs, what tests were run, and what future review should inspect.

```sql
SELECT
  substr(ticket_day, 1, 7) AS month,
  COUNT(*) AS tickets,
  SUM((SELECT COUNT(*) FROM docmgr_docs d WHERE d.ticket = t.ticket)) AS docs,
  SUM((SELECT COALESCE(SUM(word_count), 0) FROM docmgr_docs d WHERE d.ticket = t.ticket)) AS words
FROM docmgr_tickets t
GROUP BY month
ORDER BY month;
```

| month | tickets | docs | words |
| --- | --- | --- | --- |
| 2026-03 | 24 | 163 | 262517 |
| 2026-04 | 1 | 7 | 11395 |
| 2026-05 | 17 | 123 | 285173 |
| 2026-06 | 10 | 90 | 281620 |

The most striking number is not the number of tickets. It is the amount of written material. March, May, and June each contain hundreds of thousands of words of docmgr material. This changes how the history should be read. Geppetto was not maintained by code changes alone; it was maintained by a written process that records analysis, decisions, failures, validation commands, and follow-up risks.

## 10. Largest research and implementation tickets

```sql
SELECT
  t.ticket,
  t.ticket_day,
  t.title,
  COUNT(d.path) AS docs,
  SUM(d.word_count) AS words
FROM docmgr_tickets t
LEFT JOIN docmgr_docs d ON d.ticket = t.ticket
GROUP BY t.ticket
ORDER BY words DESC
LIMIT 20;
```

| ticket | ticket_day | title | docs | words |
| --- | --- | --- | --- | --- |
| GP-RESPONSES-REPLAY | 2026-05-06 | Audit Responses API reasoning parsing and replay schema | 10 | 144467 |
| 2026-06-05-geppetto-gemini-api-polish | 2026-06-05 | Geppetto Gemini API Polish for Gemini 3 Flash | 19 | 103895 |
| GP-GOJA-API-2026-06-01 | 2026-06-01 | Review and redesign Geppetto go-go-goja API and JavaScript bindings | 8 | 49417 |
| 2026-06-05-geppetto-provider-gap-audit | 2026-06-05 | Geppetto Provider Gap Audit | 16 | 49050 |
| GP-GOJA-STREAM-EVENTS-2026-06-01 | 2026-06-01 | Design Geppetto JS streaming events via go-go-goja event emitter | 9 | 38206 |
| GP-50-REGISTRY-LOADING-CLEANUP | 2026-03-19 | Clean up registry loading and remove ParseEngineProfileRegistrySourceEntries | 13 | 27616 |
| GP-49-ENGINE-PROFILES | 2026-03-18 | reintroduce engine profiles and separate them from app runtime configuration | 7 | 22516 |
| GP-OBSERVABILITY | 2026-05-07 | Add Geppetto provider and event observability hooks for high-frequency inference debugging | 8 | 21472 |
| GP-40-OPINIONATED-GO-APIS | 2026-03-17 | Opinionated Go APIs for Geppetto Runner Scaffolding | 7 | 19703 |
| GP-EVENT-VOCABULARY | 2026-05-08 | Split provider, run, and text segment event vocabulary | 9 | 19200 |
| GP-55-HTTP-PROXY | 2026-03-27 | Add HTTP proxy flags to Geppetto and Pinocchio | 9 | 18174 |
| GP-41-REMOVE-PROFILE-OVERRIDES | 2026-03-17 | Remove request-level profile override functionality from Geppetto profile resolution | 8 | 17727 |
| 2026-06-05-geppetto-llm-proxy-image-input | 2026-06-05 | Geppetto and llm-proxy Image Input Support | 8 | 14121 |
| GP-33 | 2026-03-15 | Extract scoped DB tool pattern into reusable geppetto package | 6 | 13505 |
| GEP-EMBPROF-001 | 2026-05-23 | Embedding Profiles for Geppetto and Pinocchio Registries | 6 | 13273 |

The largest tickets tell the qualitative story. `GP-RESPONSES-REPLAY` is the largest by a wide margin, which matches the hot-path evidence around OpenAI Responses. Gemini, provider gaps, Goja APIs, streaming events, event vocabulary, and image input follow closely. These are not unrelated topics. They all orbit a central concern: Geppetto is becoming a provider abstraction and runtime integration layer that must preserve provider-specific semantics without leaking accidental complexity to callers.

## 11. Recent commit narrative

```sql
SELECT author_day, short_sha, category, subject
FROM commits
ORDER BY author_date DESC
LIMIT 40;
```

| author_day | short_sha | category | subject |
| --- | --- | --- | --- |
| 2026-06-06 | 1ad8be2bf14a | merge | Merge pull request #372 from wesen/bug/store-runtime-owner |
| 2026-06-06 | d091f5ff50dc | fixes | embeddings: preserve embedding-local API settings |
| 2026-06-06 | 66b3167bd328 | dependencies | :arrow_up: Bump depenencies |
| 2026-06-06 | a8ef7d85aae6 | merge | Merge pull request #371 from go-go-golems/task/llm-proxy |
| 2026-06-06 | 712318b7fde6 | features | geppetto: expose embeddings to JavaScript |
| 2026-06-06 | dd6f735c0423 | fixes | Gemini: preserve function call thought signatures |
| 2026-06-05 | bd293e52cb7d | maintenance | Claude: wrap unsigned reasoning blocks |
| 2026-06-05 | 5a0da10d5921 | docs/research | Docs: close image input ticket |
| 2026-06-05 | 0bfce53115d9 | tests | Image input: add smoke coverage |
| 2026-06-05 | bc33c233e89b | maintenance | Gemini: map inline image input |
| 2026-06-05 | c63ae387ff93 | maintenance | Image input: normalize provider mappings |
| 2026-06-05 | dcb3fa1746a0 | features | Image input: add normalization helper |
| 2026-06-05 | b46598b9c604 | docs/research | Docs: plan image input support |
| 2026-06-05 | 3ed5a67125e6 | tests | Gemini: modernize SDK path and smoke coverage |
| 2026-06-05 | e4099e5cf061 | docs/research | Docs: create Gemini API polish guide |
| 2026-06-05 | d5412db204c6 | docs/research | Docs: audit provider gap matrix |
| 2026-06-05 | e21863f6ab52 | docs/research | Docs: create provider gap audit guide |
| 2026-06-05 | 1522d7bd881e | features | Support Claude extended thinking streams |
| 2026-06-05 | 7043076d0658 | maintenance | Force Claude engine streaming requests |
| 2026-06-05 | 1f2f057db446 | merge | Merge pull request #370 from wesen/bug/store-runtime-owner |
| 2026-06-05 | 2d35c00bb619 | merge | Merge something? |
| 2026-06-04 | 2535c87421b3 | merge | Merge pull request #369 from wesen/task/goja-runtime-flags |
| 2026-06-04 | 59b0c1266b16 | tests | Preserve session filters in geppetto latest turn lookup |
| 2026-06-04 | de32fe472ee8 | maintenance | Avoid dynamic SQL in geppetto turn listing |
| 2026-06-04 | 1dacfd6edf77 | fixes | Fix geppetto provider lint issues |
| 2026-06-04 | c85b3a7fa0b4 | dependencies | :arrow_up: Bump depenencies |
| 2026-06-04 | 4c975f1bd6a3 | fixes | Fix geppetto profile agent build nil API panic |
| 2026-06-04 | 5aaa8748532e | maintenance | Register geppetto provider resource closers |
| 2026-06-04 | d89b75b23269 | features | Add geppetto host service contributions |
| 2026-06-04 | 67a8571b565d | features | Add geppetto xgoja turn store flags |

The recent commits show the project in a polishing phase after a burst of provider and runtime work. The sequence around June 4-6 is especially dense: Goja turn store flags, host service contributions, provider closers, nil panic fixes, SQL safety, session filters, image input mapping, Claude reasoning blocks, Gemini thought signatures, embeddings exposure, and local API setting preservation. This is the pattern of a system whose boundary surfaces are being exercised together.

## 12. What the work accomplished

### Provider behavior became more precise

The provider commits are not generic integration work. They preserve details that matter at runtime: Claude extended thinking streams, unsigned reasoning blocks, Gemini function-call thought signatures, inline image mapping, OpenAI Responses reasoning replay, and streaming helpers. These details are easy to flatten away in a provider abstraction. The recent work points in the opposite direction: keep a common interface, but do not erase provider semantics that affect correctness.

### JavaScript became a serious surface area

The repeated changes under `pkg/js/modules/geppetto/`, generated TypeScript definitions, module tests, Goja stream events, and embeddings exposure show that JavaScript is not merely a convenience wrapper. It is an API surface that needs tests, generated definitions, runtime flags, event semantics, and host service integration. That explains why the Goja tickets are among the largest docmgr artifacts.

### Documentation functioned as implementation infrastructure

The amount of docmgr writing is not incidental. Large tickets like provider gap audits, Gemini API polish, Responses replay, and Goja API redesign are part of how the implementation moved safely. They record the state of external APIs, local assumptions, known gaps, and validation steps. In this project, documentation is not separate from engineering. It is one of the tools used to reduce uncertainty before and after code changes.

### Fixes followed new capability quickly

Several recent commits are small corrective steps after larger features. `embeddings: preserve embedding-local API settings`, `Gemini: preserve function call thought signatures`, `Fix geppetto profile agent build nil API panic`, and `Avoid dynamic SQL in geppetto turn listing` all point to the same engineering reality: expanding provider/runtime behavior creates new invariants. The project responded by tightening those invariants with focused fixes and tests.

## 13. How to regenerate the report

Run these commands from the ticket directory:

```bash
python3 scripts/01_build_history_db.py \
  --repo ../../../../.. \
  --since 2026-03-14 \
  --until 2026-06-14 \
  --db various/history.sqlite

sqlite3 various/history.sqlite \
  < scripts/02_summary_queries.sql \
  > various/01-history-summary.md

python3 scripts/03_render_history_report.py \
  --db various/history.sqlite \
  --out analysis/01-last-three-months-history.md \
  --since 2026-03-14 \
  --until 2026-06-14
```

The first command rebuilds the database. The second command runs the reusable SQL query pack. The third command renders the Markdown report. If the report is edited by hand afterward, rerunning the renderer may overwrite those edits; in that case, either update the renderer or keep a separate hand-authored version.

## 14. Complete SQL query pack

The SQL is included here because the report should be readable as both narrative and method. A future reader should not have to hunt through the repository to understand how the tables were interrogated.

```sql
-- Summary queries for the Geppetto last-three-months history database.
-- Run with:
--   sqlite3 various/history.sqlite < scripts/02_summary_queries.sql

.mode markdown
.headers on

.print '# Overall git activity'
SELECT
  COUNT(*) AS commits,
  SUM(is_merge) AS merge_commits,
  SUM(CASE WHEN is_merge = 0 THEN 1 ELSE 0 END) AS non_merge_commits,
  SUM(files_changed) AS files_changed,
  SUM(insertions) AS insertions,
  SUM(deletions) AS deletions,
  MIN(author_day) AS first_day,
  MAX(author_day) AS last_day
FROM commits;

.print '\n# Commits by month and category'
SELECT
  substr(author_day, 1, 7) AS month,
  category,
  COUNT(*) AS commits,
  SUM(files_changed) AS files_changed,
  SUM(insertions) AS insertions,
  SUM(deletions) AS deletions
FROM commits
GROUP BY month, category
ORDER BY month, commits DESC;

.print '\n# Top touched paths'
SELECT
  path,
  COUNT(*) AS commits,
  SUM(COALESCE(insertions, 0)) AS insertions,
  SUM(COALESCE(deletions, 0)) AS deletions
FROM commit_files
GROUP BY path
ORDER BY commits DESC, insertions + deletions DESC
LIMIT 30;

.print '\n# Docmgr ticket count by month'
SELECT
  substr(ticket_day, 1, 7) AS month,
  COUNT(*) AS tickets,
  SUM((SELECT COUNT(*) FROM docmgr_docs d WHERE d.ticket = t.ticket)) AS docs,
  SUM((SELECT COALESCE(SUM(word_count), 0) FROM docmgr_docs d WHERE d.ticket = t.ticket)) AS words
FROM docmgr_tickets t
GROUP BY month
ORDER BY month;

.print '\n# Largest docmgr tickets by written words'
SELECT
  t.ticket,
  t.ticket_day,
  t.title,
  COUNT(d.path) AS docs,
  SUM(d.word_count) AS words
FROM docmgr_tickets t
LEFT JOIN docmgr_docs d ON d.ticket = t.ticket
GROUP BY t.ticket
ORDER BY words DESC
LIMIT 20;

.print '\n# Recent notable commits'
SELECT author_day, short_sha, category, subject
FROM commits
ORDER BY author_date DESC
LIMIT 40;
```

## 15. Caveats

Every history report has a point of view. This one is explicit about its limits.

- Commit categories are heuristic and subject-line based. They are useful for orientation, not for final accounting.
- Ticket dates come from docmgr path dates under `ttmp/YYYY/MM/DD/...`. That measures ticket creation or placement date, not necessarily last edit date.
- Word counts are approximate and are computed from Markdown text after stripping simple frontmatter.
- Merge commit line counts can be large and should be interpreted as integration events.
- The SQLite database is generated from the current checkout. If branches, tags, or historical files differ in another clone, the exact numbers may differ.

## 16. Key points to carry forward

- Geppetto's last three months were dominated by provider/runtime work, not by a single isolated feature.
- OpenAI Responses, Gemini, Claude, Goja, JavaScript bindings, and embeddings formed the main implementation centers.
- Docmgr was part of the engineering process, not just a record after the fact.
- The most useful history is reproducible: the database, scripts, SQL, raw output, and report all live inside the ticket.
- The next reader should begin with the large docmgr tickets when they need rationale, and with the hot-path files when they need implementation detail.
