# Tasks

## Completed

- [x] Create ticket workspace `PI-019-CLEANUP-UI`
- [x] Analyze current debug UI helper/style/mock duplication hotspots
- [x] Analyze reusable styling architecture patterns in `pinocchio` webchat
- [x] Author detailed implementation plan for helper, CSS, and Storybook mock cleanup
- [x] Create and maintain ticket diary while planning
- [x] Upload design/implementation documentation to reMarkable
- [x] Expand ticket tasks into detailed execution checklist

## Phase 0 — Baseline + Guardrails (prep)

- [x] P0.1 Record baseline file metrics (LOC per major component, total TSX/CSS LOC)
- [x] P0.2 Record baseline duplication metrics (helper duplicates, inline `<style>` block count)
- [x] P0.3 Capture baseline build metrics (`npm run build`, `npm run build-storybook` chunk outputs)
- [x] P0.4 Add temporary tracking section in `analysis/01` for before/after numbers
- [x] P0.5 Confirm branch strategy and PR slicing plan with maintainers

## Phase 1 — Helper Unification

### 1A. Create shared helper modules

- [x] P1.1 Create `src/ui/format/phase.ts` (`formatPhaseLabel`, `formatPhaseShort`)
- [x] P1.2 Create `src/ui/format/time.ts` (`formatTimeShort`, safe date formatting)
- [x] P1.3 Create `src/ui/format/text.ts` (`truncateText`, safe stringify helpers)
- [x] P1.4 Create `src/ui/presentation/events.ts` (`getEventPresentation`)
- [x] P1.5 Create `src/ui/presentation/blocks.ts` (`getBlockPresentation`)
- [x] P1.6 Create `src/ui/presentation/timeline.ts` (`getTimelineKindPresentation`)

### 1B. Migrate components to shared helpers

- [x] P1.7 Migrate `EventCard.tsx` to shared event/text helpers
- [x] P1.8 Migrate `EventTrackLane.tsx` to shared event helpers
- [x] P1.9 Migrate `EventInspector.tsx` to shared event/time helpers
- [x] P1.10 Migrate `BlockCard.tsx` to shared block/text helpers
- [x] P1.11 Migrate `StateTrackLane.tsx` to shared block/phase helpers
- [x] P1.12 Migrate `FilterBar.tsx` to shared block presentation helpers
- [x] P1.13 Migrate `ProjectionLane.tsx` to shared timeline presentation helpers
- [x] P1.14 Migrate `TimelineEntityCard.tsx` to shared timeline/text helpers
- [x] P1.15 Migrate `SnapshotDiff.tsx` to shared phase helpers
- [x] P1.16 Migrate `TurnInspector.tsx` to shared phase/time helpers

### 1C. Cleanup + tests

- [x] P1.17 Remove all now-duplicate local helper functions
- [x] P1.18 Add helper unit tests for mapping fallback behavior
- [x] P1.19 Add helper unit tests for truncation/format edge cases
- [x] P1.20 Verify no duplicate helper signatures remain via grep check

## Phase 2 — CSS Dedup + Reusable Design System

### 2A. Establish style architecture

- [x] P2.1 Create `src/styles/tokens.css` (color, spacing, type, radius, shadow tokens)
- [x] P2.2 Create `src/styles/reset.css` (minimal global reset)
- [x] P2.3 Create `src/styles/primitives.css` (buttons, cards, badges, inputs)
- [x] P2.4 Create `src/styles/layout.css` (app shell + page/lane layout)
- [x] P2.5 Create `src/styles/components/` directory and per-component CSS files
- [x] P2.6 Convert `src/index.css` into import orchestrator for style layers

### 2B. Extract inline style blocks from runtime files

- [x] P2.7 Extract styles from `AppShell.tsx`
- [x] P2.8 Extract styles from `TimelineLanes.tsx`
- [x] P2.9 Extract styles from `StateTrackLane.tsx`
- [x] P2.10 Extract styles from `EventTrackLane.tsx`
- [x] P2.11 Extract styles from `ProjectionLane.tsx`
- [x] P2.12 Extract styles from `NowMarker.tsx`
- [x] P2.13 Extract styles from `TurnInspector.tsx`
- [x] P2.14 Extract styles from `SnapshotDiff.tsx`
- [x] P2.15 Extract styles from `EventInspector.tsx`
- [x] P2.16 Extract styles from `FilterBar.tsx`
- [x] P2.17 Extract styles from `AnomalyPanel.tsx`
- [x] P2.18 Extract route-level styles from `OverviewPage.tsx`
- [x] P2.19 Extract route-level styles from `TimelinePage.tsx`
- [x] P2.20 Extract route-level styles from `EventsPage.tsx`
- [x] P2.21 Extract route-level styles from `TurnDetailPage.tsx`

### 2C. Styling contract standardization

- [x] P2.22 Define and document style contract approach (data-part, class naming, or hybrid)
- [x] P2.23 Normalize naming of CSS classes/parts across all major components
- [x] P2.24 Ensure theme token usage replaces hard-coded repeated colors where practical
- [x] P2.25 Verify no inline `<style>{` blocks remain in runtime TSX files

## Phase 3 — Storybook Mock Data Centralization

### 3A. Split fixtures/factories/scenarios

- [x] P3.1 Create `src/mocks/fixtures/` domain files (conversations, turns, events, timeline, anomalies)
- [x] P3.2 Create `src/mocks/factories/` for deterministic builders
- [x] P3.3 Create `src/mocks/scenarios/` for reusable story contexts
- [x] P3.4 Create deterministic id/time/seq helper utilities for factories
- [x] P3.5 Refactor legacy `src/mocks/data.ts` into compatibility export layer or remove after migration

### 3B. Centralize MSW handler setup

- [x] P3.6 Create `src/mocks/msw/createDebugHandlers.ts`
- [x] P3.7 Create `src/mocks/msw/defaultHandlers.ts`
- [x] P3.8 Migrate `src/mocks/handlers.ts` to use handler builder(s)
- [x] P3.9 Remove repeated per-story handler blocks in `AppShell.stories.tsx`
- [x] P3.10 Remove repeated per-story handler blocks in `SessionList.stories.tsx`

### 3C. Story migration

- [x] P3.11 Migrate timeline-related stories to scenario builders
- [ ] P3.12 Migrate event inspector stories to scenario builders
- [ ] P3.13 Migrate anomaly stories to centralized anomaly fixtures/factories
- [ ] P3.14 Ensure stories use shared fixture imports rather than local large arrays
- [ ] P3.15 Add mock architecture README (`src/mocks/README.md`)

## Phase 4 — Final Cleanup + Documentation + Enforcement

- [ ] P4.1 Remove stale/unused `src/App.tsx` or repurpose intentionally
- [ ] P4.2 Remove dead exports and orphan helper files after migration
- [ ] P4.3 Add frontend README section: helper usage rules
- [ ] P4.4 Add frontend README section: style layer contract and token policy
- [ ] P4.5 Add frontend README section: Storybook fixture/factory/scenario policy
- [ ] P4.6 Add lightweight lint/check script for duplicate helper pattern regression
- [ ] P4.7 Add lightweight check for forbidden runtime inline `<style>` blocks

## Validation & Exit Criteria

- [ ] V1 `npm run build` passes after each phase
- [ ] V2 `npm run build-storybook` passes after each phase
- [ ] V3 Visual parity spot-check done for key stories (AppShell, TimelineLanes, SnapshotDiff, EventInspector)
- [ ] V4 Duplicate helper count reduced to zero for targeted signatures
- [ ] V5 Runtime inline `<style>` count reduced to zero
- [ ] V6 Storybook mock setup uses centralized fixtures/factories/scenarios
- [ ] V7 Changelog + diary updated with implementation outcomes and follow-ups
