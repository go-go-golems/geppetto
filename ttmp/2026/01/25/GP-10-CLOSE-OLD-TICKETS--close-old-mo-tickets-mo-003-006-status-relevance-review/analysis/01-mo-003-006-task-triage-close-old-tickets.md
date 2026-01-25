---
Title: MO-003..006 task triage (close old tickets)
Ticket: GP-10-CLOSE-OLD-TICKETS
Status: active
Topics:
    - infrastructure
    - geppetto
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/engine/engine.go
      Note: Evidence that sink wiring is context-based (no engine.WithSink API)
    - Path: geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/tasks.md
      Note: MO-003 open-task list being triaged
    - Path: geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/tasks.md
      Note: MO-004 open-task list being triaged
    - Path: geppetto/ttmp/2026/01/20/MO-005-CLEANUP-SINKS--cleanup-engine-withsink-usage-move-sink-wiring-to-context-session/tasks.md
      Note: MO-005 sink-cleanup task list being triaged
    - Path: geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/tasks.md
      Note: MO-006 cancellation/lifecycle task list being triaged
    - Path: geppetto/ttmp/2026/01/21/MO-005-OAK-GIT-HISTORY--oak-git-history-sqlite-pr-analysis-geppetto/tasks.md
      Note: MO-005 oakgitdb task list being triaged
    - Path: oak-git-db/pkg/oakgitdb/builder.go
      Note: Evidence for current oakgitdb schema/multi-repo support
ExternalSources: []
Summary: 'Triage of MO-003..MO-006 open tasks: what is done, what remains relevant, and recommendations for closing/splitting tickets.'
LastUpdated: 2026-01-25T09:09:45.107535795-05:00
WhatFor: ""
WhenToUse: ""
---


# MO-003..006 task triage (close old tickets)

## Goal

Review the open tasks in the MO-003..MO-006 ticket series (plus the second `MO-005-*` ticket) and decide:
- what is already done (even if tasks weren’t checked off),
- what is still relevant and should be finished soon,
- what is obsolete/superseded and should be moved or dropped,
- what to do next (concrete actions + priority),
- whether the ticket should be closed.

## Scope

- Tickets reviewed:
  - `MO-003-UNIFY-INFERENCE`
  - `MO-004-UNIFY-INFERENCE-STATE`
  - `MO-005-CLEANUP-SINKS`
  - `MO-005-OAK-GIT-HISTORY`
  - `MO-006-CLEANUP-CANCELLATION-LIFECYCLE`
- Evidence sources used: each ticket’s `tasks.md`, `changelog.md`, and diary/design docs where relevant, plus lightweight code “existence checks” when a task claims a concrete artifact.

## Executive recommendations (draft; updated per-ticket below)

- Close `MO-003-UNIFY-INFERENCE` as **superseded** by the work completed in `MO-004`/`MO-006`, with any remaining work split into narrower follow-up tickets (prompt-resolution middleware, Moments migration, ordering tests/fixtures).
- Close `MO-004-UNIFY-INFERENCE-STATE` as **mostly done**, keeping a small backlog of optional hardening tasks (tests, persisters, and go-go-mento migration) as explicit follow-ups.
- Keep `MO-005-OAK-GIT-HISTORY` **active** if the DB is still being used, but re-scope the remaining open tasks (they are mostly “v2 features”, not MVP completeness).
- Either close or de-prioritize `MO-005-CLEANUP-SINKS` depending on how much `engine.WithSink` still exists in active codepaths; this is cleanup-only and may not justify a dedicated ticket if architecture moved on.
- Close `MO-006-CLEANUP-CANCELLATION-LIFECYCLE` as **effectively complete** (tests + compendium exist); only keep open if we still want interactive smoke scripts and final naming/invariants doc polish.

---

## Ticket: MO-003-UNIFY-INFERENCE

### Snapshot

- Ticket status: `active`
- Last updated: 2026-01-16 (per `docmgr ticket list`)
- Task list reality check: `tasks.md` shows **16 open / 0 done**, but the ticket changelog/diary indicate multiple substantial implementation steps landed (shared runner, pinocchio migrations, middleware fixups, Moments DebugTap).

### Recommendation

Close `MO-003-UNIFY-INFERENCE` as **superseded / “completed enough”**, because:
- The executed work (runner + migrations) appears done and already captured in the ticket’s changelog/diary.
- The remaining items are large, multi-repo efforts (Moments migration, prompt-resolution middleware) better tracked as dedicated follow-up tickets (or folded into the currently-active architecture series).

### Open tasks assessment

Interpretation key:
- **Done**: evidence suggests the outcome exists (even if checkbox isn’t ticked).
- **Relevant**: still valuable to do, but not completed.
- **Superseded**: replaced by other tickets/architecture; close here and track elsewhere if needed.

| Task | Text (abridged) | Assessment | Why / evidence | What’s needed next | Priority |
|---:|---|---|---|---|---|
| 2 | Inventory inference loops + state usage | Done | MO-003 has multiple analysis/design docs covering pinocchio/moments/go-go-mento flows. | N/A | — |
| 3 | Define unified inference loop API contract | Done / Superseded | MO-003 design docs + follow-up MO-004 “InferenceState + Session + EngineBuilder” work. | Ensure the “final contract” lives in MO-004 docs; close here. | — |
| 4 | Design prompt-resolution middleware | Relevant | MO-003 has analysis for Moments prompt resolution, but no clear shared middleware implementation tracked here. | Decide target API (middleware vs builder), implement in geppetto, migrate callers. | Medium |
| 5 | Implement shared conversation builder (state snapshot + turn builder) | Relevant | Large scope; MO-004 focuses on state/session, but “conversation builder” is still a concrete gap if we want true unification across repos. | Create a new narrowly-scoped ticket if still desired; define boundaries (what is built where). | Medium–High |
| 6 | Migrate pinocchio TUI to shared inference builder | Done | MO-003 changelog shows “migrate TUI backend” commit `2df3b2c`. | Mark as done in MO-003 tasks (optional cleanup). | — |
| 7 | Migrate pinocchio webchat to shared inference builder | Done | MO-003 changelog shows “migrate webchat” commit `0fdcb56`. | Mark as done in MO-003 tasks (optional cleanup). | — |
| 8 | Author Moments follow-up plan doc | Done | MO-003 design docs include a Moments migration plan section. | If needed, consolidate into one “Moments migration plan” doc. | Low |
| 9 | Add tests/fixtures for ordering + prompt resolution | Relevant | MO-004/MO-006 added unit tests for session/tool loop, but ordering + prompt resolution fixtures/tests are not obviously complete here. | Add targeted tests in the codebase that owns ordering invariants (turn snapshot builder). | Medium |
| 10 | Update docs/designs with final flow diagrams | Relevant but optional | Many docs exist; “final diagrams” are polish. | Create final consolidated diagram and link from MO-004/MO-006. | Low |
| 11 | Add webchat DebugTap (pre-inference snapshots) | Likely done | MO-003 diary references a Moments DebugTap writing to `/tmp/conversations/...`. | Confirm it’s merged/used; if not, commit + wire it. | Low |
| 12 | Define shared inference runner (TUI + webchat) | Done | Runner exists and is used per MO-003 diary/changelog. | N/A | — |
| 13 | Refactor pinocchio TUI backend to use shared runner + sinks | Done | Covered by Step 1 in MO-003 changelog. | N/A | — |
| 14 | Refactor pinocchio webchat router to use shared runner + sinks | Done | Covered by Step 2 in MO-003 changelog. | N/A | — |
| 15 | Align TUI + webchat snapshot/ConversationState handling | Done (as implemented) | Runner unifies orchestration; MO-003 diary indicates removal of duplicate helpers. | N/A | — |
| 16 | Document Moments migration plan (no code) | Done | MO-003 design docs cover Moments migration phases. | N/A | Low |

### Suggested “closure actions” (if we actually close MO-003)

- In `MO-003-UNIFY-INFERENCE/tasks.md`: check off tasks that are clearly done and move remaining relevant items into new tickets with narrower scope.
- Change the ticket status to `complete` after follow-up tickets exist (or explicitly record “superseded by MO-004/MO-006” in `index.md`).

---

## Ticket: MO-004-UNIFY-INFERENCE-STATE

### Snapshot

- Ticket status: `active`
- Last updated: 2026-01-20
- `tasks.md`: 6 open / 7 done
- Changelog indicates substantial implementation work landed (InferenceState/Session/EngineBuilder extraction, pinocchio migrations, sink wiring changes, postmortem + testing playbook).

### Recommendation

Close `MO-004-UNIFY-INFERENCE-STATE` as **complete** with two explicit follow-ups:
- Optional: provide concrete `TurnPersister` implementations (no-op + filesystem/debug).
- Optional: decide whether `go-go-mento` needs migration at all (it appears to be a deprecated ancestor; if so, close the “update go-go-mento” task as “won’t do”).

### Open tasks assessment (MO-004)

| Task | Text (abridged) | Assessment | Why / evidence | What’s needed next | Priority |
|---:|---|---|---|---|---|
| 4 | Move ToolCallingLoop into geppetto inference core | Done (but task list stale) | Current tree has `geppetto/pkg/inference/toolloop/*` with tests; tool-loop is clearly geppetto-owned now. | Check off task 4 and reference the newer package path. | — |
| 6 | Update go-go-mento webchat to use geppetto types | Likely obsolete | `go-go-mento` is described as deprecated; updating it may not be worth the effort unless it’s still executed anywhere. | Decide “is go-go-mento still run?” If no: close as “won’t do”. If yes: create a focused migration ticket. | Low |
| 8 | Add reference persister implementations | Not done | `TurnPersister` exists (interface + tests), but no concrete implementations found in `geppetto/pkg/inference`. | Add `NoopPersister` and a filesystem persister (write turns under a debug dir). | Medium |
| 9 | Add targeted tests for Session.RunInference paths | Done (but task list stale) | `geppetto/pkg/inference/session/session_test.go` exists and covers success + cancellation + invariants. Tool-loop has its own tests. | Check off task 9. | — |
| 10 | Document migration notes / breaking API changes | Mostly done | MO-004 has a postmortem and an inference testing playbook; those serve as migration notes. | Optional: add a short “Breaking changes” section in `index.md` linking the postmortem. | Low |

### Suggested “closure actions” (MO-004)

- Check off tasks 4 and 9 in `MO-004-UNIFY-INFERENCE-STATE/tasks.md`.
- Mark task 6 as “won’t do” if `go-go-mento` is deprecated and not executed.
- Either implement persisters (task 8) or move that requirement into a new small ticket and close MO-004.

---

## Ticket: MO-005-CLEANUP-SINKS

### Snapshot

- Ticket status: `active`
- Last updated: 2026-01-20
- `tasks.md`: 5 open / 0 done
- Ticket contains a thorough migration design doc, but the task list was never updated after subsequent refactors.

### Recommendation

Close `MO-005-CLEANUP-SINKS` as **done / superseded**, because the main goal (“remove engine-config sinks like `engine.WithSink` and standardize on context/session sinks”) appears to already be true in the current codebase:
- No `engine.WithSink` call sites exist in `*.go` (search across the workspace).
- `geppetto/pkg/inference/engine/engine.go` defines only the `Engine` interface (no `engine.Config` with sink fields).
- Event sinks are wired via `events.WithEventSinks(ctx, ...)` and higher-level builders (e.g. tool-loop enginebuilder has `WithEventSinks(...)`).

### Open tasks assessment (MO-005-CLEANUP-SINKS)

| Task | Text (abridged) | Assessment | Why / evidence | What’s needed next | Priority |
|---:|---|---|---|---|---|
| 2 | Inventory `engine.WithSink` usages | Done | Inventory exists in ticket analysis doc; current code has no call sites. | N/A | — |
| 3 | Write migration design | Done | Ticket analysis doc is the migration design. | N/A | — |
| 4 | Implement cleanup + remove WithSink / Config.EventSinks | Done | WithSink/Config no longer appear in Go code; sinks are context-based. | Optional: update MO-005 docs noting completion. | — |
| 5 | Update examples + docs; add smoke tests | Mostly done | Examples now use context sinks (e.g., `enginebuilder.WithEventSinks`). Tool-loop tests exist elsewhere. | Optional: add a brief “how to wire sinks” snippet to a central playbook. | Low |

### Suggested “closure actions” (MO-005-CLEANUP-SINKS)

- Mark the ticket status `complete` and add a short note in `index.md` like “Completed as part of inference/session/toolloop refactor; engine-config sinks removed.”

---

## Ticket: MO-005-OAK-GIT-HISTORY

### Snapshot

- Ticket status: `active`
- Last updated: 2026-01-21
- `tasks.md`: 17 open / 10 done
- Core deliverable (MVP) was completed and then moved into a standalone repo folder: `oak-git-db/` (with its own docs and `go.mod`).

### Recommendation

Close `MO-005-OAK-GIT-HISTORY` as **“MVP complete; future enhancements deferred”** unless there is an active, near-term need for the remaining v2 features.

Rationale:
- The original goal (“generate a PR-focused SQLite DB and document usage”) appears achieved and documented.
- The remaining open tasks are mostly “nice-to-have” enrichment features (symbol deltas, extra edge types, PR hunks/blame, query examples, metadata polish).

If the tool is actively used right now for PR reviews, keep the ticket open but re-scope it explicitly as “oakgitdb v2 enhancements” and prune/prioritize the list.

### Open tasks assessment (MO-005-OAK-GIT-HISTORY)

| Task | Text (abridged) | Assessment | Why / evidence | What’s needed next | Priority |
|---:|---|---|---|---|---|
| 4 | “Symbol delta” materialization (base vs head) | Relevant (v2) | Useful for “what changed” queries; not required for MVP DB existence. | Implement delta tables/materialized views in `oak-git-db/pkg/oakgitdb/builder.go`. | Medium |
| 5 | Non-call references (type refs, const/var refs, func-as-value) | Relevant (v2) | Increases recall for code-impact analysis; may increase DB size significantly. | Extend extractor + add size controls (sampling/limits). | Low–Medium |
| 6 | Interface implementation edges | Relevant (v2) | High leverage for Go analysis, but complex/cached. | Add best-effort extraction + caching strategy. | Medium |
| 7 | PR hunk ingestion + optional blame | Relevant (v2) | Useful for “line-level” context; more expensive and optional. | Implement git diff/hunk table + optional blame mode. | Low–Medium |
| 10–19 | Multi-repo ID strategy + cross-repo symbol identity | Partially done | Multi-repo support landed (per ticket changelog); remaining items are design cleanup / identity normalization. | If multi-repo queries are used: finalize identity choices and document them in `oak-git-db/docs/design.md`. | Medium |
| 21 | Store per-repo metadata | Relevant but optional | Nice for UX and reproducibility. | Ingest repo name/root/origin URL at build time. | Low |
| 22–24 | Example cross-repo queries | Relevant but optional | Improves usability; low engineering risk. | Add a few reference queries to `oak-git-db/docs/usage.md`. | Low |

### Suggested “closure actions” (MO-005-OAK-GIT-HISTORY)

- Decide whether “v2 enhancements” should be:
  - A new ticket (recommended), keeping this ticket marked `complete`, or
  - Continued in this ticket, but with a refreshed prioritized task list and an explicit scope statement.

---

## Ticket: MO-006-CLEANUP-CANCELLATION-LIFECYCLE

### Snapshot

- Ticket status: `active`
- Last updated: 2026-01-20 / 2026-01-21
- `tasks.md`: 4 open / 3 done
- The ticket already contains:
  - A lifecycle/cancellation analysis doc (“Run vs Conversation vs Inference”).
  - A compendium doc consolidating sinks/session/state/tool-loop/cancellation.
  - Unit tests (though some file paths referenced in the ticket have since moved as part of later refactors).

### Recommendation

Close `MO-006-CLEANUP-CANCELLATION-LIFECYCLE` as **complete**.

The only remaining open task that’s meaningfully “new work” is interactive smoke testing (tmux + webchat cancel behavior). Everything else (documenting current lifecycle and proposing a cleaned model) appears to have been delivered as written docs in the ticket itself.

### Open tasks assessment (MO-006)

| Task | Text (abridged) | Assessment | Why / evidence | What’s needed next | Priority |
|---:|---|---|---|---|---|
| 2 | Document current lifecycle + cancellation across TUI/webchat | Done (task list stale) | Ticket contains `analysis/01-run-vs-conversation-vs-inference-...md`. | Check off task 2. | — |
| 3 | Propose renamed/cleaned model; ownership + invariants | Done (task list stale) | The analysis doc + compendium propose the cleaned mental model and invariants. | Check off task 3. | — |
| 4 | Add tests/smoke scripts for cancel behavior (tmux + webchat) | Partially done | Minimal unit tests were added (per changelog), but interactive smoke scripts are not clearly present. | Add a short playbook/script to reproduce cancel behavior in TUI + webchat. | Low–Medium |

### Suggested “closure actions” (MO-006)

- Check off tasks 2 and 3 in `MO-006-CLEANUP-CANCELLATION-LIFECYCLE/tasks.md`.
- Either implement a short smoke-testing playbook/script (task 4) or explicitly defer it and close the ticket as complete.

---

## Closure playbook (copy/paste)

These commands are intentionally “operator-driven”: skim the ticket once, then run the closure steps.

### MO-003-UNIFY-INFERENCE

```bash
docmgr task list --ticket MO-003-UNIFY-INFERENCE
# Suggested “safe” check-offs (after confirming): 2,3,6,7,8,12,13,14,15,16
docmgr task check --ticket MO-003-UNIFY-INFERENCE --id 2,3,6,7,8,12,13,14,15,16
docmgr ticket close --ticket MO-003-UNIFY-INFERENCE --changelog-entry "Closed as superseded by MO-004/MO-006; remaining work split into narrower follow-ups"
```

### MO-004-UNIFY-INFERENCE-STATE

```bash
docmgr task list --ticket MO-004-UNIFY-INFERENCE-STATE
# Suggested check-offs (after confirming): 4,9,10
docmgr task check --ticket MO-004-UNIFY-INFERENCE-STATE --id 4,9,10
docmgr ticket close --ticket MO-004-UNIFY-INFERENCE-STATE --changelog-entry "Core unification complete; persister helpers and any go-go-mento migration deferred"
```

### MO-005-CLEANUP-SINKS

```bash
docmgr task list --ticket MO-005-CLEANUP-SINKS
# Suggested check-offs (after confirming): 2,3,4,5
docmgr task check --ticket MO-005-CLEANUP-SINKS --id 2,3,4,5
docmgr ticket close --ticket MO-005-CLEANUP-SINKS --changelog-entry "Closed: engine.WithSink no longer exists; sinks standardized on context/session"
```

### MO-005-OAK-GIT-HISTORY

```bash
docmgr task list --ticket MO-005-OAK-GIT-HISTORY
# Option A (recommended): close as MVP complete; keep v2 ideas for a new ticket.
docmgr ticket close --ticket MO-005-OAK-GIT-HISTORY --changelog-entry "Closed as MVP complete; oakgitdb moved to oak-git-db/; deferred v2 enrichment tasks"
```

### MO-006-CLEANUP-CANCELLATION-LIFECYCLE

```bash
docmgr task list --ticket MO-006-CLEANUP-CANCELLATION-LIFECYCLE
# Suggested check-offs (after confirming): 2,3
docmgr task check --ticket MO-006-CLEANUP-CANCELLATION-LIFECYCLE --id 2,3
docmgr ticket close --ticket MO-006-CLEANUP-CANCELLATION-LIFECYCLE --changelog-entry "Closed: lifecycle/cancellation model documented + minimal regression tests exist; smoke scripts deferred"
```

## Suggested follow-up tickets (only if the work is still wanted)

- Prompt-resolution middleware (shared, reusable) — medium priority if Moments migration is still planned.
- Turn persister helpers (no-op + filesystem/debug) — medium priority (developer ergonomics).
- oakgitdb v2 enrichment (symbol delta, interface edges, hunks/blame, query examples) — low–medium priority depending on active usage.
- Cancellation smoke-testing playbook/scripts (tmux + webchat) — low–medium priority if cancellation regressions recur.
