---
Title: 'Implementation Plan: Runtime Snapshot Diff Integration'
Ticket: GP-019-DEBUG-UI-SNAPSHOT-DIFF-RUNTIME
Status: active
Topics:
    - frontend
    - debugging
    - webchat
    - timeline
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/components/SnapshotDiff.tsx
      Note: Existing diff component to wire into runtime
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/components/TurnInspector.tsx
      Note: Primary integration point for compare phases and diff rendering
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/routes/OfflinePage.tsx
      Note: Needs visual inspector integration over raw payload
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/routes/OverviewPage.tsx
      Note: Needs full-lane data wiring
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/routes/TimelinePage.tsx
    - Path: pinocchio/cmd/web-chat/web/src/debug-ui/store/uiSlice.ts
ExternalSources: []
Summary: Detailed implementation guide to wire SnapshotDiff into runtime debug UI and close remaining integration gaps.
LastUpdated: 2026-02-14T12:12:00-05:00
WhatFor: Guide execution work for runtime debug-ui integration with no compatibility shims.
WhenToUse: Use when implementing remaining debug-ui runtime integration and deciding what to remove or wire.
---


# Implementation Plan: Runtime Snapshot Diff Integration

## Goal
Move SnapshotDiff and remaining debug-ui surfaces from partial/story-only state into the real runtime debug experience in `pinocchio/cmd/web-chat/web/src/debug-ui`, with read-only behavior and no legacy backfill/compatibility layers.

## Current Integration Status (Code Facts)
1. `SnapshotDiff` exists but is not mounted anywhere in runtime routes/components.
2. `TurnInspector` already stores compare-phase state (`comparePhaseA`, `comparePhaseB`) but only renders block list and metadata card.
3. `OverviewPage` renders `TimelineLanes` with `events=[]` and `entities=[]`, so the 3-lane model is not actually exercised there.
4. `OfflinePage` renders raw run detail JSON and does not reuse `TurnInspector`, `EventInspector`, or `TimelineLanes`.
5. `MiddlewareChainView` and `TimelineEntityCard` are currently orphaned from runtime composition.
6. `uiSlice` includes `liveStreamEnabled`, `inspectorPanel`, and filter fields that are only partially wired to rendered behavior.

## Other Functionality Potentially Not Integrated in UI
1. Middleware chain inspector flow (`inspectorPanel = 'mw'`) has no runtime entry point.
2. Dedicated timeline entity card component is not used (`TimelineEntityCard.tsx` orphaned).
3. `FilterBar` state is local to `AppShell` and not applied to RTK Query params or lane selectors.
4. Overview route does not show real event/projection lanes even when data exists.
5. Offline viewer is mostly a payload dump, not inspector-grade visualization.
6. `NowMarker`/live timeline behavior is disabled in runtime (`isLive={false}` in routes).
7. Diff view in runtime does not currently receive block selection callbacks or metadata-first rendering from compare mode.

## Implementation Guide

### Phase 1: Wire SnapshotDiff into TurnInspector
1. In `TurnInspector.tsx`, derive default compare phases from available phases when state is empty.
2. Mount `SnapshotDiff` below compare selectors when both selected phases exist.
3. Pass `turnA` and `turnB` from `turnDetail.phases[phase]` directly.
4. Keep metadata visible: render metadata diff summary and continue exposing full metadata in expandable cards.
5. Keep current block list for selected phase, but add "jump from diff row" behavior by setting selected block index when `onBlockSelect` fires.

### Phase 2: Make Overview and Timeline routes use full lane data
1. Update `OverviewPage.tsx` to query timeline/events like `TimelinePage` (or share a common data hook).
2. Feed full `turns/events/entities` into `TimelineLanes` on overview.
3. Keep lane selection to drive `selectTurn`, `selectEvent`, and selected entity state.
4. Avoid duplicate fetches by normalizing query args and `skip` logic.

### Phase 3: Upgrade Offline viewer from JSON dump to inspector shell
1. Keep source selection in sidebar as-is.
2. Add read-only visualization sections for offline run detail:
   - turns list + turn detail + snapshot diff
   - event list + event inspector
   - timeline entities lane/inspection
3. Retain raw payload panel as a fallback tab only.
4. Prefer metadata fields in UI rendering over ad-hoc payload flattening where both are available.

### Phase 4: Remove or integrate orphan pieces
1. Decide per component:
   - If required now: wire `MiddlewareChainView`, `TimelineEntityCard`.
   - If not in scope: delete component + stories and remove dead styles.
2. Remove stale UI slice fields that are not used after route wiring.
3. Keep no backwards compatibility shim for removed runtime paths.

### Phase 5: Tests and verification
1. `SnapshotDiff` integration tests in `TurnInspector`:
   - compare phase selection
   - diff rows count
   - metadata change chips
   - block select callback wiring
2. Route-level tests:
   - overview receives and renders non-empty event/projection lanes
   - offline route renders inspector sections for selected run
3. Manual QA:
   - run Storybook and app side by side
   - verify selected turn with known phase differences shows non-zero diff blocks

## Detailed Task Breakdown
1. Add compare-phase defaulting + mounted `SnapshotDiff` in `TurnInspector`.
2. Add selected-entity state and projection interaction parity in overview/timeline pages.
3. Refactor overview data loading to include events/timeline.
4. Introduce offline inspector layout (summary tabs: turns/events/timeline/raw).
5. Integrate metadata-first rendering for diff context (turn metadata + block metadata).
6. Resolve orphan components by wiring or deletion.
7. Remove dead slice fields and selectors after integration.
8. Add/adjust tests for component and route behavior.
9. Run `pnpm test` (or targeted tests) and `pnpm typecheck` in `pinocchio/cmd/web-chat/web`.

## Acceptance Criteria
1. Selecting two phases in a turn renders SnapshotDiff in runtime app.
2. Diff row click selects corresponding block details in current inspector panel.
3. Overview route displays all three lanes with real data when available.
4. Offline viewer supports visual inspection flows, not only JSON dump.
5. No dead UI state fields/components remain for removed behavior.
6. Tests cover the new diff wiring and pass in CI.

## Risks and Mitigations
1. Risk: compare-phase state becomes invalid when changing turns.
   Mitigation: reset compare phases when selected turn changes to unavailable values.
2. Risk: extra RTK Query calls from overview/timeline duplication.
   Mitigation: shared query hook and strict `skip` guards.
3. Risk: metadata rendering leaks huge payloads.
   Mitigation: expandable sections + truncated preview + explicit raw tab.

## Out of Scope
1. Backwards compatibility for old debug UI routes/shape.
2. Realtime websocket follow mode (covered by GP-020).
