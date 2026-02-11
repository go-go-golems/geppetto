---
Title: Diary
Ticket: MO-005-CLEANUP-SINKS
Status: active
Topics:
    - inference
    - architecture
    - events
    - webchat
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
  - Path: geppetto/ttmp/2026/01/20/MO-005-CLEANUP-SINKS--cleanup-engine-withsink-usage-move-sink-wiring-to-context-session/analysis/01-sink-cleanup-removing-engine-withsink-and-standardizing-on-context-session-sinks.md
    Note: Primary sink-cleanup analysis and migration plan
  - Path: geppetto/pkg/events/context.go
    Note: Context sink plumbing (append-only semantics)
  - Path: geppetto/pkg/inference/core/session.go
    Note: Session-owned EventSinks injection per run
  - Path: geppetto/pkg/inference/engine/options.go
    Note: engine.WithSink implementation targeted for removal
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-20T16:05:24.97607979-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Remove `engine.WithSink` and standardize event sink wiring on run-context sinks (preferably injected by `core.Session`), while keeping streaming/tool-loop behavior correct and avoiding duplicate event emission.

## Step 1: Create MO-005 ticket workspace and docs

This step created a clean ticket workspace for the sink cleanup and set up the docs needed to track work (analysis + diary + tasks). The scope is intentionally narrow: sink wiring only, not broader inference refactors.

**Commit (code):** N/A

### What I did
- Created ticket `MO-005-CLEANUP-SINKS` and initial documents (`Diary`, `Sink cleanup...` analysis).
- Added tasks for inventory, design, implementation, and smoke tests.

### Why
- Sink wiring currently has two injection points (engine-config vs ctx), which increases the risk of duplicate/missing events. We want one clear pattern.

### What worked
- Ticket scaffold created successfully.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Review ticket scaffold under:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-005-CLEANUP-SINKS--cleanup-engine-withsink-usage-move-sink-wiring-to-context-session/`

### Technical details
- Commands:
  - `docmgr ticket create-ticket --ticket MO-005-CLEANUP-SINKS ...`
  - `docmgr doc add --ticket MO-005-CLEANUP-SINKS ...`
  - `docmgr task add --ticket MO-005-CLEANUP-SINKS ...`

## Step 2: Inventory `engine.WithSink` usage across repos (incl. moments)

This step established the concrete list of call sites that must be migrated before we can delete `engine.WithSink`. Importantly, it also verified that moments and go-go-mento already prefer context sinks for their webchat router runs, so the main cleanup work is in pinocchio + geppetto tests/fixtures.

**Commit (code):** N/A

### What I did
- Ran ripgrep searches for `engine.WithSink(` across the workspace, excluding docs for the “code inventory” view.
- Recorded all matches and verified there are no production uses in `moments/**` or `go-go-mento/**` outside docs.

### Why
- We need a complete list of call sites to avoid leaving a straggler that blocks deletion.

### What worked
- Found a small, finite set of production call sites (pinocchio CLI + pinocchio TUI builder) and a handful of geppetto fixtures/tests.

### What didn't work
- N/A

### What I learned
- The biggest “real” dependency on engine-config sinks is pinocchio’s TUI path: `runtime/builder.go` injects `engine.WithSink(uiSink)` and the backend Session currently doesn’t supply `EventSinks`, so the UI relies on engine-config sinks today.

### What was tricky to build
- Distinguishing “docs mention WithSink” vs “code depends on WithSink”.

### What warrants a second pair of eyes
- Confirm no other alias exists (e.g. `inference.WithSink`) in production code that should also be migrated.

### What should be done in the future
- Update docs after code migration so they don’t continue recommending `engine.WithSink`.

### Code review instructions
- Review the inventory section in the analysis doc:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-005-CLEANUP-SINKS--cleanup-engine-withsink-usage-move-sink-wiring-to-context-session/analysis/01-sink-cleanup-removing-engine-withsink-and-standardizing-on-context-session-sinks.md`

### Technical details
- Commands:
  - `rg -n "\\bengine\\.WithSink\\(" -S --glob '!**/ttmp/**' --glob '!**/pkg/doc/**'`
