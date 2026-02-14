---
Title: Diary
Ticket: GP-019-DEBUG-UI-SNAPSHOT-DIFF-RUNTIME
Status: active
Topics:
    - frontend
    - debugging
    - webchat
    - timeline
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/debug-ui/components/AppShell.tsx
      Note: Exploration tracked local filter state and URL sync behavior
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/debug-ui/components/SnapshotDiff.tsx
      Note: |-
        Exploration confirmed it is currently unmounted in runtime
        Metadata-first diff rendering and chips
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/debug-ui/components/TurnInspector.tsx
      Note: Runtime SnapshotDiff mount and block selection sync
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/debug-ui/routes/OfflinePage.tsx
      Note: Offline inspector route integration and tabs
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/debug-ui/routes/OverviewPage.tsx
      Note: Exploration confirmed events/entities currently omitted in overview lanes
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/debug-ui/routes/offlineData.ts
      Note: Offline parsing bridge for turns/events/entities
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/debug-ui/store/uiSlice.ts
      Note: Removed dead UI slice state and added selectedEntityId
ExternalSources: []
Summary: Exploration diary for snapshot diff runtime integration planning.
LastUpdated: 2026-02-14T18:21:39-05:00
WhatFor: Capture exploration process and planning choices for GP-019.
WhenToUse: Use to review how implementation scope and sequencing were derived.
---



# Diary

## Goal
Capture the concrete exploration trail and planning rationale used to create the runtime SnapshotDiff integration ticket and implementation plan.

## Step 1: Audit runtime wiring and identify integration gaps
I started by validating where the debug UI currently mounts runtime components versus story-only components. The main objective was to avoid assumptions and create a plan tied to exact files and state wiring.

This step established that SnapshotDiff is present but not in runtime composition, and it also exposed additional UI surfaces that remain partially integrated or unused.

### Prompt Context
**User prompt (verbatim):** "Create a new ticket to integrate it in the UI, and add a detailed implementation guide to the ticket. 

Also make a list of other functionality that might not be integrated in the UI.

Also make a ticket + implementation plan to add realtime websocket connection to the debug UI when selecting a socket, to be able to \"follow\" realtime conversations (happening in another tab).

Upload both implementation plans to the remarkable tablet. Keep a diary while exploring /"

**Assistant interpretation:** Create two docmgr tickets with detailed plans, explicitly list unintegrated UI functionality, keep an exploration diary, and upload both plans to reMarkable.

**Inferred user intent:** Turn exploratory findings into actionable tickets and implementation-ready documentation so execution can continue immediately.

**Commit (code):** N/A (documentation-only step)

### What I did
- Enumerated debug-ui components/routes under `pinocchio/cmd/web-chat/web/src/debug-ui`.
- Verified `SnapshotDiff.tsx` has zero runtime usage.
- Inspected `TurnInspector.tsx`, `OverviewPage.tsx`, `TimelinePage.tsx`, `OfflinePage.tsx`, and `AppShell.tsx` for composition and state flow.
- Audited `uiSlice.ts` fields for partially-wired or unused state.
- Created ticket workspace `GP-019-DEBUG-UI-SNAPSHOT-DIFF-RUNTIME` and authored the implementation plan.

### Why
- Needed an accurate gap inventory before proposing task breakdown and acceptance criteria.
- Needed to separate immediate SnapshotDiff integration from larger realtime work.

### What worked
- `rg` scans quickly surfaced orphan components and unused integration points.
- Route and component reads made it straightforward to map current runtime behavior.

### What didn't work
- One attempted file read targeted non-existent `debug-ui/store/turnsApi.ts` and failed with: `sed: can't read ... No such file or directory`.
- Resolved by switching inspection to `debug-ui/api/debugApi.ts`.

### What I learned
- Compare-phase state is already present, so SnapshotDiff integration is mostly composition and behavior wiring.
- Overview route currently under-uses available timeline/event data even though the APIs exist.

### What was tricky to build
- Distinguishing truly unused state/components from those used only in Storybook required checking route composition and runtime imports together.
- The subtle gap is not missing components, but missing runtime composition links.

### What warrants a second pair of eyes
- Whether to keep or remove `MiddlewareChainView`/`TimelineEntityCard` in this ticket versus a cleanup-only follow-up.
- Whether filter state should remain local to `AppShell` or become slice-driven before adding more route coupling.

### What should be done in the future
1. Execute ticket tasks in small increments with route/component tests per phase.
2. Align offline inspector composition with runtime inspector components to minimize duplicated UI logic.

### Code review instructions
- Start with `pinocchio/cmd/web-chat/web/src/debug-ui/components/TurnInspector.tsx` and `pinocchio/cmd/web-chat/web/src/debug-ui/components/SnapshotDiff.tsx`.
- Then review route composition in `pinocchio/cmd/web-chat/web/src/debug-ui/routes/OverviewPage.tsx` and `pinocchio/cmd/web-chat/web/src/debug-ui/routes/OfflinePage.tsx`.
- Validate by running debug UI and confirming compare phases render diff rows in runtime.

### Technical details
- Primary discovery commands used:
  - `rg --files .../debug-ui/components`
  - `rg -n "<SnapshotDiff|SnapshotDiff\(" .../debug-ui`
  - `rg -n "filterBarOpen|setFilters|inspectorPanel|liveStreamEnabled" .../debug-ui`

## Step 2: Implement runtime integration, harden lint/type behavior, and close tasks
I implemented the GP-019 runtime wiring in `pinocchio` as one cohesive pass: SnapshotDiff now mounts in runtime TurnInspector, diff row clicks move phase/block selection, overview lanes consume events/entities, and offline viewer was upgraded into the same inspector composition pattern.

The second part of this step was stabilization and cleanup. I resolved strict TypeScript and Biome issues introduced during integration, added regression tests for phase/selection and offline parsing behavior, and committed the full change as one focused GP-019 code commit.

### Prompt Context
**User prompt (verbatim):** "Ok, do 019 now"

**Assistant interpretation:** Execute GP-019 end-to-end immediately: implement, validate, and close ticket tasks.

**Inferred user intent:** Move from planning to shipped runtime behavior and ticket closure-ready documentation without additional design iteration.

**Commit (code):** 58ebcef — "feat(debug-ui): wire snapshot diff and offline inspector runtime"

### What I did
- Implemented runtime compare normalization and diff mounting in `TurnInspector.tsx` using new helper state logic in `turnInspectorState.ts`.
- Wired diff-row click handling back to phase/block inspector selection (`resolveBlockSelectionIndex`).
- Extended `SnapshotDiff.tsx` metadata rendering to show middleware/trace chips and changed values; updated `SnapshotDiff.css`.
- Added shared lane hook `useLaneData.ts` and switched `OverviewPage.tsx` + `TimelinePage.tsx` to use real turns/events/entities.
- Reworked `OfflinePage.tsx` into inspector layout with tabs (`turns/events/timeline/raw`) and integrated `TimelineLanes`, `TurnInspector`, `EventInspector`, `TimelineEntityCard`.
- Added offline parser/model bridge in `offlineData.ts` with tests in `offlineData.test.ts`.
- Removed orphan `MiddlewareChainView.tsx` + `.stories.tsx`; reused `TimelineEntityCard` inside `ProjectionLane.tsx`.
- Removed dead UI slice fields/actions and introduced `selectedEntityId` selection in `uiSlice.ts`.
- Added tests for compare selection and diff block mapping in `turnInspectorState.test.ts`.
- Ran validation commands:
  - `npm run typecheck`
  - `npm run check`
  - `npx vitest run src/debug-ui/components/turnInspectorState.test.ts src/debug-ui/routes/offlineData.test.ts`
  - `npm run build`

### Why
- GP-019’s scope was runtime integration, not Storybook-only behavior. The core value was making existing components participate in real navigation/state flows.
- Cleanup of dead slice state and orphans was necessary to avoid keeping stale architecture around the new integrated paths.

### What worked
- Compare-phase normalization helpers made runtime and tests deterministic.
- Shared lane data hook removed route duplication and enabled overview parity quickly.
- Offline inspector composition with existing components reduced custom one-off rendering logic.

### What didn't work
- Initial strict typecheck failure:
  - Command: `npm run typecheck`
  - Error: `turnInspectorState.test.ts(64,10): error TS2367: This comparison appears to be unintentional because the types '"draft"' and '"final"' have no overlap.`
  - Fix: replaced tautological test expectation with concrete phase expectation.
- Initial lint pass surfaced hook-order and dependency issues in `OfflinePage.tsx`:
  - Command: `npm run lint`
  - Errors included `lint/correctness/useHookAtTopLevel` and `lint/correctness/noUnusedVariables`.
  - Fix: made hooks unconditional, removed unused memo, and moved early returns after hook declarations.
- One follow-up lint failure remained (`assist/source/organizeImports`) in `OfflinePage.tsx`.
  - Fix command: `npx --yes @biomejs/biome@2.3.8 check --write src/debug-ui/routes/OfflinePage.tsx`.

### What I learned
- The existing debug-ui primitives were mostly complete; most missing behavior was in route composition and state synchronization.
- Offline run detail requires explicit normalization and fallback mapping for `turns`, `artifact`, and `timeline` kinds before inspector components can be reused safely.

### What was tricky to build
- The biggest sharp edge was offline hook ordering versus conditional load states. The page initially followed a common “early-return loading/error states” style, but Biome’s hook rules correctly flagged that as conditional hook execution once inspector hooks were introduced.  
  The resolution was to compute stable fallback inspector data first, run hooks unconditionally, and then branch rendering for source/run/loading/error states.
- Diff-row block linking required phase-aware lookup semantics (ID, then index, then payload+metadata signature) to avoid selecting the wrong block when shapes diverged across snapshots.

### What warrants a second pair of eyes
- `offlineData.ts` flattening/parsing heuristics for timeline props and mixed artifact payload formats.
- `uiSlice.ts` cleanup correctness in case hidden consumers still depended on removed fields/actions outside current debug-ui routes.
- Runtime UX behavior when compare phases are modified rapidly while selected phase has no matching block index.

### What should be done in the future
1. Add component-level route rendering tests (with React test renderer/RTL) once test harness dependencies are introduced.
2. Consider code-splitting debug routes/components to reduce the current large production bundle warning.

### Code review instructions
- Start with runtime behavior:
  - `pinocchio/cmd/web-chat/web/src/debug-ui/components/TurnInspector.tsx`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/components/SnapshotDiff.tsx`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/components/turnInspectorState.ts`
- Then review route composition:
  - `pinocchio/cmd/web-chat/web/src/debug-ui/routes/OverviewPage.tsx`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/routes/TimelinePage.tsx`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/routes/OfflinePage.tsx`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/routes/offlineData.ts`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/routes/useLaneData.ts`
- Validate with:
  - `cd pinocchio/cmd/web-chat/web && npm run check`
  - `cd pinocchio/cmd/web-chat/web && npx vitest run src/debug-ui/components/turnInspectorState.test.ts src/debug-ui/routes/offlineData.test.ts`
  - `cd pinocchio/cmd/web-chat/web && npm run build`

### Technical details
- Commit:
  - `58ebcef feat(debug-ui): wire snapshot diff and offline inspector runtime`
- Key added files:
  - `pinocchio/cmd/web-chat/web/src/debug-ui/components/turnInspectorState.ts`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/components/turnInspectorState.test.ts`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/routes/offlineData.ts`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/routes/offlineData.test.ts`
  - `pinocchio/cmd/web-chat/web/src/debug-ui/routes/useLaneData.ts`
