---
Title: Implementation Plan for UI Helper, CSS System, and Storybook Mock Data Cleanup
Ticket: PI-019-CLEANUP-UI
Status: active
Topics:
    - frontend
    - architecture
    - middleware
    - websocket
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.stories.tsx
      Note: |-
        Reference scenario-runner approach for story data generation
        Reference scenario-runner pattern for story mock generation
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/parts.ts
      Note: |-
        Reference helper pattern for class/style slot overrides
        Reference class/style merge and part props helper approach
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/styles/theme-default.css
      Note: |-
        Reference token layer pattern for reusable theming
        Reference token architecture
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/styles/webchat.css
      Note: |-
        Reference data-part based styling structure
        Reference part-based style layering
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components
      Note: |-
        Main React component surface with helper and style duplication to clean up
        Current duplication hotspots across UI components
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/index.css
      Note: |-
        Current global tokens/utilities baseline to evolve into reusable design system layers
        Current token/utility baseline for style extraction planning
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/data.ts
      Note: |-
        Central fixture file to split into factories/scenarios
        Current centralized fixture file to split into fixtures/factories/scenarios
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/handlers.ts
      Note: |-
        Current MSW handlers to refactor into reusable handler builders
        Current MSW handlers to refactor into reusable handler builder
ExternalSources: []
Summary: Detailed execution plan for reducing PI-013 frontend code size by consolidating duplicate helpers, extracting reusable CSS design system layers, and centralizing Storybook mock data generation.
LastUpdated: 2026-02-07T12:05:00-05:00
WhatFor: Provide a concrete implementation blueprint for PI-019 cleanup work with phased tasks, target file structure, acceptance criteria, and migration strategy.
WhenToUse: Use when implementing PI-019 cleanup tasks, assigning work, and reviewing whether the frontend has reached the desired maintainability baseline.
---


# PI-019 Implementation Plan: Helper + CSS + Storybook Mock Cleanup

## 1) Goal

Reduce frontend code size and maintenance cost in `web-agent-example/cmd/web-agent-debug/web` by implementing three coordinated cleanups:

1. **Unify duplicate helpers** (event/block/entity presentation, formatting, truncation, phase labels).
2. **Unify CSS/style duplication** into a reusable design system structure inspired by `pinocchio` webchat.
3. **Centralize Storybook mock data generation** (fixtures + factories + scenarios + MSW handler builders).

This plan is intentionally implementation-focused and designed for phased delivery without breaking current feature work.

---

## 2) Current State (Why this cleanup is needed)

## 2.1 Helper duplication (current evidence)

Duplicate/near-duplicate functions are spread across multiple components:

- `getEventTypeInfo` in:
  - `src/components/EventCard.tsx`
  - `src/components/EventTrackLane.tsx`
  - `src/components/EventInspector.tsx`
- `getKindInfo` in:
  - `src/components/ProjectionLane.tsx`
  - `src/components/TimelineEntityCard.tsx`
- `getKindIcon` in:
  - `src/components/BlockCard.tsx`
  - `src/components/StateTrackLane.tsx`
  - `src/components/FilterBar.tsx`
- `truncateText` in:
  - `src/components/EventCard.tsx`
  - `src/components/BlockCard.tsx`
  - `src/components/TimelineEntityCard.tsx`
- phase formatting duplicates in:
  - `src/components/SnapshotDiff.tsx`
  - `src/components/StateTrackLane.tsx`
  - `src/components/TurnInspector.tsx`

### Impact

- Hard to keep labels/icons/colors consistent.
- More code to review for each change.
- Increased bug chance when one copy is updated and others are not.

## 2.2 CSS/style duplication (current evidence)

The current debug UI has many runtime `<style>{`...`}</style>` blocks embedded in components/routes.

Examples include:

- `src/components/EventInspector.tsx` (multiple style blocks)
- `src/components/SnapshotDiff.tsx` (multiple style blocks)
- `src/components/AnomalyPanel.tsx` (multiple style blocks)
- `src/components/AppShell.tsx`
- `src/routes/*Page.tsx`

### Impact

- Repeated CSS snippets in render paths.
- Larger component files (mixed behavior + structure + CSS text).
- Hard to theme or restyle globally.
- Higher long-term bundle/maintenance overhead.

## 2.3 Storybook mock data is partly centralized, partly scattered

Current situation:

- Good base fixtures already exist in:
  - `src/mocks/data.ts`
  - `src/mocks/handlers.ts`
- But story files also define/duplicate local mock structures and ad-hoc variants, for example:
  - repeated `mockAnomalies` arrays
  - repeated event/timeline duplication logic
  - repeated MSW handlers in individual stories (`AppShell.stories.tsx`, `SessionList.stories.tsx`)

### Impact

- Story setup drift.
- Hard to build coherent scenario sets.
- Harder to reason about “canonical” fixture contracts.

---

## 3) Reference Pattern from `pinocchio` Webchat (what to copy)

Use these structural ideas from `pinocchio/cmd/web-chat/web/src/webchat`:

## 3.1 Token layer + structural CSS layer split

- `styles/theme-default.css` defines token variables (`--pwchat-*`).
- `styles/webchat.css` applies component/layout styles using those tokens.

### Why it helps

- Theme overrides become easy.
- Component CSS becomes predictable and reusable.

## 3.2 Data-part based styling contracts

`webchat.css` uses selectors like:

- `[data-part="header"]`
- `[data-part="card"]`
- `[data-part="composer"]`

Components attach semantic part names rather than embedding local style strings.

### Why it helps

- Strong styling API between structure and theme.
- Enables partial override strategy without rewriting components.

## 3.3 Shared style/props merge helpers

`webchat/parts.ts` includes reusable helpers:

- `mergeClassName`
- `mergeStyle`
- `getPartProps`

### Why it helps

- Eliminates repetitive class/style merge logic.
- Supports extension points cleanly.

## 3.4 Storybook scenario runner pattern

`ChatWidget.stories.tsx` uses a `ScenarioRunner` and frame sequences rather than hand-writing large bespoke fixture objects per story.

### Why it helps

- Reusable story scenarios.
- Easier story maintenance.
- Better alignment with runtime event flow.

---

## 4) Target Architecture for PI-019

## 4.1 Helper architecture target

Create a shared helper layer:

```text
src/ui/
  format/
    phase.ts
    time.ts
    text.ts
  presentation/
    events.ts
    blocks.ts
    timeline.ts
  identity/
    keys.ts
```

### Proposed exports

- `formatPhaseLabel(phase)`
- `formatTimeShort(ts)`
- `truncateText(text, max)`
- `eventPresentation(type)` → `{ icon, colorVar, badgeClass }`
- `blockPresentation(kind)` → `{ icon, badgeClass, borderClass }`
- `timelineKindPresentation(kind)`

All components consume these helpers; no per-component copies.

## 4.2 CSS architecture target

Create layered style files inspired by webchat:

```text
src/styles/
  tokens.css            # palette, spacing, typography, radii, shadows
  reset.css             # app-level base resets
  primitives.css        # button/card/badge/input/list primitives
  layout.css            # app shell, page layouts, panels, lanes
  components/
    app-shell.css
    timeline-lanes.css
    turn-inspector.css
    snapshot-diff.css
    event-inspector.css
    filter-bar.css
    anomaly-panel.css
```

Use either:

- data-part attributes, or
- stable BEM-like class names

but standardize on one approach consistently.

### Recommendation

Adopt **data-part + utility class hybrid**:

- data-part for major semantic regions (`data-part="timeline-lane"` etc.)
- utility classes from `index.css` kept minimal

## 4.3 Storybook mock architecture target

```text
src/mocks/
  fixtures/
    conversations.ts
    turns.ts
    events.ts
    timeline.ts
    anomalies.ts
  factories/
    conversationFactory.ts
    turnFactory.ts
    eventFactory.ts
    timelineFactory.ts
    anomalyFactory.ts
  scenarios/
    overviewScenarios.ts
    eventInspectorScenarios.ts
    timelineScenarios.ts
    anomalyScenarios.ts
  msw/
    createDebugHandlers.ts
    defaultHandlers.ts
  index.ts
```

### Principles

- Fixtures are static canonical examples.
- Factories generate variants deterministically.
- Scenarios define reusable story contexts.
- Story files only compose scenarios (no large local arrays).

---

## 5) Detailed Execution Plan (phased)

## Phase 0 — Baseline and guardrails (1 day)

1. Capture baseline metrics:
   - total LOC
   - number of `<style>` blocks
   - bundle sizes (`npm run build`, `npm run build-storybook`)
2. Add temporary tracking checklist to PI-019 tasks.
3. Add no-op tests for helper modules (scaffold test harness if missing).

**Deliverable:** measurable before/after baseline.

---

## Phase 1 — Helper unification (1–2 days)

## 5.1 Create shared helper modules

Create:

- `src/ui/presentation/events.ts`
- `src/ui/presentation/blocks.ts`
- `src/ui/presentation/timeline.ts`
- `src/ui/format/phase.ts`
- `src/ui/format/text.ts`
- `src/ui/format/time.ts`

## 5.2 Migrate components incrementally

Migrate in this order (highest duplication first):

1. `EventCard.tsx`
2. `EventTrackLane.tsx`
3. `EventInspector.tsx`
4. `BlockCard.tsx`
5. `StateTrackLane.tsx`
6. `FilterBar.tsx`
7. `ProjectionLane.tsx`
8. `TimelineEntityCard.tsx`
9. `SnapshotDiff.tsx`
10. `TurnInspector.tsx`

## 5.3 Remove duplicate local helpers

After each migration, delete local copies and update imports.

## 5.4 Add helper tests

Add focused tests for:

- event type mapping fallback behavior
- block kind mapping fallback behavior
- phase label formatting
- truncation edge cases (undefined, empty, long text)

**Phase 1 acceptance criteria**

- No duplicated presentation helpers remain in component files.
- All helper tests pass.
- Existing stories still render correctly.

---

## Phase 2 — CSS/style dedup into design system layers (2–3 days)

## 5.5 Establish style layers

1. Add `src/styles/tokens.css` and move theme variables from `src/index.css`.
2. Add `src/styles/primitives.css` for card/badge/button/input primitives.
3. Add `src/styles/layout.css` for shell/page/lane layouts.
4. Keep `src/index.css` as thin import orchestrator:

```css
@import './styles/tokens.css';
@import './styles/reset.css';
@import './styles/primitives.css';
@import './styles/layout.css';
@import './styles/components/app-shell.css';
...;
```

## 5.6 Extract inline style blocks component-by-component

Priority order:

1. `AppShell.tsx`
2. `TimelineLanes.tsx`
3. `StateTrackLane.tsx`
4. `EventTrackLane.tsx`
5. `ProjectionLane.tsx`
6. `TurnInspector.tsx`
7. `SnapshotDiff.tsx`
8. `EventInspector.tsx`
9. `FilterBar.tsx`
10. `AnomalyPanel.tsx`
11. route page style blocks (`OverviewPage`, `TimelinePage`, `EventsPage`, `TurnDetailPage`)

## 5.7 Optional (recommended): adopt part-style API for key components

For major reusable shells (e.g., `AppShell`, `TimelineLanes`), add `partProps`-style extension capability similar to `pinocchio` webchat if needed for future theming.

## 5.8 Visual regression check

After each extracted file:

- run Storybook for affected component
- compare before/after screenshots manually (or with visual snapshots if available)

**Phase 2 acceptance criteria**

- Zero runtime `<style>{` blocks in components/routes.
- Tokens and primitives are centralized.
- No visual regressions in key stories.

---

## Phase 3 — Storybook mock centralization (1–2 days)

## 5.9 Split fixtures from generation

- Move large static objects from `mocks/data.ts` into `fixtures/*` grouped by domain.
- Introduce factories for generated variants:
  - `makeEvent(...)`
  - `makeTurnSnapshot(...)`
  - `makeTimelineEntity(...)`
  - `makeAnomaly(...)`

## 5.10 Create scenario library

Create reusable scenario builders:

- `makeOverviewScenario({ density, live })`
- `makeEventInspectorScenario({ withCorrelation, withTrustFailures })`
- `makeAnomalyScenario({ severityMix })`

## 5.11 Centralize MSW handler composition

Replace per-story repeated handlers with:

- `createDebugHandlers(overrides?)`

Stories then do:

```ts
parameters: {
  msw: { handlers: createDebugHandlers({ conversations: [], delayMs: 1000 }) }
}
```

## 5.12 Story migration sweep

Update stories to consume scenario/factory APIs, removing large local inline arrays where possible.

**Phase 3 acceptance criteria**

- Story files are mostly declarative.
- `mocks` layer has clear ownership boundaries.
- Repeated MSW handler blocks are eliminated.

---

## Phase 4 — Final cleanup + enforcement (1 day)

## 5.13 Remove stale/dead code

- Remove or repurpose stale `src/App.tsx` if still unused.
- Remove unused helper exports after refactor.

## 5.14 Add lightweight guardrails

- Add lint rules/checks that catch repeated local utility patterns when shared helpers exist.
- Add a simple grep-based CI check for inline style blocks in TSX render returns (if desired).

## 5.15 Documentation update

- Update frontend README with:
  - style layering rules
  - helper module usage rules
  - story fixture/factory/scenario conventions

**Phase 4 acceptance criteria**

- Cleanup conventions documented and enforceable.

---

## 6) Detailed Task Breakdown (ready to convert into checklist)

## Helper cleanup tasks

1. Create `src/ui/presentation/*` + `src/ui/format/*` modules.
2. Migrate each component import-by-import.
3. Delete local duplicated helpers.
4. Add helper unit tests.

## CSS cleanup tasks

1. Create `src/styles/*` layered structure.
2. Move variables and primitives out of `index.css`.
3. Extract all inline style blocks to CSS files.
4. Replace ad-hoc classes with standardized primitives/parts.
5. Validate story rendering after each migration.

## Storybook mock cleanup tasks

1. Split fixtures by domain.
2. Add factories and deterministic ID/timestamp helpers.
3. Add scenario builders.
4. Introduce `createDebugHandlers` for MSW.
5. Migrate stories to scenario/factory API.

---

## 7) Validation Plan

## 7.1 Functional validation

- `npm run build`
- `npm run storybook` (manual spot checks)
- `npm run build-storybook`

## 7.2 Contract validation

- Ensure no story relies on stale endpoint shape from ad-hoc handlers.
- Ensure generated mock seq/version values remain stable and deterministic.

## 7.3 Quality validation

- LOC reduction in large component files (`SnapshotDiff`, `EventInspector`, `AnomalyPanel`, `FilterBar`).
- Inline style block count reduced to zero.
- Duplicate helper function count reduced to zero.

---

## 8) Risk and Mitigation

| Risk | Impact | Mitigation |
|---|---|---|
| Visual regressions during CSS extraction | Medium | Migrate one component at a time and verify in Storybook immediately |
| Refactor churn with ongoing feature work | Medium | Use phased PRs with narrow blast radius |
| Factory over-engineering | Low-Med | Keep factory API minimal and focused on existing story needs |
| Breaking story expectations | Medium | Maintain backward-compatible fixture names during migration, then remove aliases |

---

## 9) Suggested PR slicing

1. **PR-1:** Helper modules + migration + tests.
2. **PR-2:** Style system scaffold + first 3 component extractions.
3. **PR-3:** Remaining style extraction and cleanup.
4. **PR-4:** Mock fixtures/factories/scenarios + story migrations.
5. **PR-5:** Final cleanup, docs, and guardrails.

This keeps review size manageable and reduces merge risk.

---

## 10) Definition of Done for PI-019

PI-019 is done when:

1. Shared helper modules replace all duplicated local helper functions.
2. All inline TSX `<style>{` blocks are removed.
3. Style layers are centralized and documented.
4. Storybook mocks are scenario/factory driven with centralized MSW handler builders.
5. Build + Storybook build pass and key stories look unchanged functionally.
6. Ticket diary/changelog/index/tasks reflect the delivered plan and upload artifacts.

---

## 11) Immediate next action

Start with **Phase 1 (helper unification)** first. It is lowest risk, gives immediate code-size reduction, and simplifies later CSS/mocks migration by reducing cross-component noise.

---

## 12) Temporary Tracking: Baseline vs After

The following baseline was captured on **2026-02-07** before implementation work.

### 12.1 Baseline metrics (before)

| Metric | Before | After | Notes |
|---|---:|---:|---|
| TSX LOC (total) | 6,629 | TBD | `src/**/*.tsx` |
| TSX files | 43 | TBD | `src/**/*.tsx` |
| CSS LOC (total) | 349 | TBD | `src/**/*.css` |
| CSS files | 1 | TBD | `src/**/*.css` |
| Inline `<style>{` blocks | 31 | TBD | Runtime components + routes |
| `getEventTypeInfo` definitions | 3 | TBD | Target: 0 local duplicates |
| `getKindInfo` definitions | 2 | TBD | Target: 0 local duplicates |
| `getKindIcon` definitions | 3 | TBD | Target: 0 local duplicates |
| `truncateText` definitions | 3 | TBD | Target: 0 local duplicates |
| `formatPhase` definitions | 2 | TBD | Target: 0 local duplicates |
| `formatPhaseName` definitions | 1 | TBD | Will be consolidated under shared phase formatter |

### 12.2 Largest TSX files (baseline)

| File | LOC |
|---|---:|
| `src/components/SnapshotDiff.tsx` | 628 |
| `src/components/EventInspector.tsx` | 606 |
| `src/components/AnomalyPanel.tsx` | 442 |
| `src/components/FilterBar.tsx` | 326 |
| `src/components/AppShell.tsx` | 280 |
| `src/components/TurnInspector.tsx` | 266 |
| `src/components/ProjectionLane.tsx` | 234 |
| `src/components/BlockCard.tsx` | 226 |
| `src/components/StateTrackLane.tsx` | 211 |
| `src/components/TimelineLanes.tsx` | 185 |

### 12.3 Build output baseline

- `npm run build`
  - `dist/assets/index-D_0hpNoR.css`: **5.14 kB** (gzip **1.57 kB**)
  - `dist/assets/index-0nh_1Zxj.js`: **364.96 kB** (gzip **109.22 kB**)
  - Build time: **1.16s**
- `npm run build-storybook`
  - Preview build time: **~11s** (`vite` reported **9.82s**)
  - Largest chunks included:
    - `storybook-static/assets/DocsRenderer-...js`: **890.05 kB** (gzip **276.15 kB**)
    - `storybook-static/assets/index-...js`: **662.40 kB** (gzip **157.06 kB**)
    - `storybook-static/assets/preview-...js`: **660.64 kB** (gzip **156.58 kB**)

### 12.4 Branch strategy and PR slicing confirmation

- Working branch for this execution: `task/implement-openai-responses-api`.
- Commit strategy: task-aligned, focused commits (baseline, helper modules, component migrations, tests/cleanup).
- PR slicing remains aligned with section 9 (PR-1 helpers, PR-2/3 CSS, PR-4 mocks, PR-5 cleanup/docs).

### 12.5 Progress snapshot (after Phase 1 helper extraction/migration)

Captured after commit `56751d0` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| TSX LOC (total) | 6,629 | 6,452 | `src/**/*.tsx` |
| `getEventTypeInfo` definitions | 3 | 0 | moved to `src/ui/presentation/events.ts` |
| `getKindInfo` definitions | 2 | 0 | moved to `src/ui/presentation/timeline.ts` |
| `getKindIcon` definitions | 3 | 0 | moved to `src/ui/presentation/blocks.ts` |
| `truncateText` definitions | 3 | 0 | moved to `src/ui/format/text.ts` |
| `formatPhase` definitions | 2 | 0 | moved to `src/ui/format/phase.ts` |
| `formatPhaseName` definitions | 1 | 0 | moved to `src/ui/format/phase.ts` |
| Runtime inline `<style>{` blocks | 31 | 31 | unchanged until Phase 2 |

Build outputs after Phase 1:

- `npm run build`
  - `dist/assets/index-D_0hpNoR.css`: **5.14 kB** (gzip **1.57 kB**)
  - `dist/assets/index-B4bn0pbi.js`: **363.71 kB** (gzip **109.32 kB**)
  - Build time: **1.81s**
- `npm run build-storybook`
  - Preview build time: **~10s** (`vite` reported **9.41s**)

### 12.6 Progress snapshot (after Phase 2A style scaffold)

Captured after commit `6efec1b` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| CSS files | 1 | 21 | split into layered files + component placeholders |
| CSS LOC (total) | 349 | 473 | includes scaffold placeholders and extracted base layers |
| Runtime inline `<style>{` blocks | 31 | 31 | extraction begins in Phase 2B |

Scaffold status:

- `src/styles/tokens.css`, `reset.css`, `primitives.css`, `layout.css` created.
- `src/styles/components/` created with per-component/route CSS files.
- `src/index.css` converted to import orchestrator.
- Validation:
  - `npm run build` passed.
  - `npm run build-storybook` passed.

### 12.7 Progress snapshot (after P2.7 AppShell extraction)

Captured after commit `41d4e9c` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Runtime inline `<style>{` blocks | 31 | 30 | `AppShell.tsx` inline block extracted |

P2.7 details:

- Inline styles moved from `src/components/AppShell.tsx` into `src/styles/components/AppShell.css`.
- Validation:
  - `npm run build` passed.
  - `npm run build-storybook` passed.

### 12.8 Progress snapshot (after P2.8-P2.12 extraction batch)

Captured after commit `9d7ba6c` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Runtime inline `<style>{` blocks | 30 | 22 | extracted from TimelineLanes, StateTrackLane, EventTrackLane, ProjectionLane, NowMarker |
| CSS LOC (total) | 473 | 1075 | increased as inline styles moved into component CSS files |

Extraction details:

- Moved inline styles from:
  - `src/components/TimelineLanes.tsx`
  - `src/components/StateTrackLane.tsx`
  - `src/components/EventTrackLane.tsx`
  - `src/components/ProjectionLane.tsx`
  - `src/components/NowMarker.tsx`
- Into:
  - `src/styles/components/TimelineLanes.css`
  - `src/styles/components/StateTrackLane.css`
  - `src/styles/components/EventTrackLane.css`
  - `src/styles/components/ProjectionLane.css`
  - `src/styles/components/NowMarker.css`

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 12.9 Progress snapshot (after P2.13 + P2.18-P2.21)

Captured after commit `ac2f936` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Runtime inline `<style>{` blocks | 22 | 18 | route-level inline styles extracted |

Task notes:

- `P2.13` (`TurnInspector.tsx`): no inline `<style>{...}` block present; task treated as already clean.
- Extracted route-level styles:
  - `OverviewPage.tsx` (`P2.18`)
  - `TimelinePage.tsx` (`P2.19`)
  - `EventsPage.tsx` (`P2.20`)
  - `TurnDetailPage.tsx` (`P2.21`)
- Styles moved into:
  - `src/styles/components/OverviewPage.css`
  - `src/styles/components/TimelinePage.css`
  - `src/styles/components/EventsPage.css`
  - `src/styles/components/TurnDetailPage.css`

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.
