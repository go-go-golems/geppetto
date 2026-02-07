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
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/README.md
      Note: Style contract recommendation implemented in frontend docs
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components
      Note: |-
        Main React component surface with helper and style duplication to clean up
        Current duplication hotspots across UI components
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/AnomalyPanel.stories.tsx
      Note: P3.13 anomaly panel scenario migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/AppShell.stories.tsx
      Note: |-
        P3.9 story-level handler deduplication evidence
        P3.13 AppShell anomaly factory migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/ConversationCard.stories.tsx
      Note: P3.14 fixture import migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/EventCard.stories.tsx
      Note: P3.14 fixture import migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/EventInspector.stories.tsx
      Note: P3.12 event-inspector story scenario migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/EventTrackLane.stories.tsx
      Note: P3.14 fixture/factory import migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/MiddlewareChainView.stories.tsx
      Note: P3.14 fixture import migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/ProjectionLane.stories.tsx
      Note: P3.14 fixture import migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/SessionList.stories.tsx
      Note: |-
        P3.10 story-level handler deduplication evidence
        P3.14 fixture/factory import migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/StateTrackLane.stories.tsx
      Note: P3.14 fixture import migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/TimelineEntityCard.stories.tsx
      Note: P3.14 fixture import migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/TimelineLanes.stories.tsx
      Note: P3.11 timeline story scenario migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/TurnInspector.stories.tsx
      Note: P3.14 fixture import migration evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/index.css
      Note: |-
        Current global tokens/utilities baseline to evolve into reusable design system layers
        Current token/utility baseline for style extraction planning
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/README.md
      Note: P3.15 documented mock architecture contract
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/data.ts
      Note: |-
        Central fixture file to split into factories/scenarios
        Current centralized fixture file to split into fixtures/factories/scenarios
        P3.5 finalized as compatibility shim pending story migration
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/anomalyFactory.ts
      Note: P3.2 anomaly deterministic builders
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/deterministic.test.ts
      Note: P3.4 helper validation evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/deterministic.ts
      Note: P3.4 deterministic helper utility design evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/eventFactory.ts
      Note: P3.2 event and middleware-trace deterministic builders
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/timelineFactory.ts
      Note: P3.2 timeline deterministic builders
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/turnFactory.ts
      Note: P3.2 turn and turn-detail deterministic builders
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/conversations.ts
      Note: P3.1 conversation/session fixture module
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/events.ts
      Note: P3.1 event and middleware-trace fixture module
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/timeline.ts
      Note: P3.1 timeline fixture module
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/handlers.ts
      Note: |-
        Current MSW handlers to refactor into reusable handler builders
        Current MSW handlers to refactor into reusable handler builder
        P3.8 now delegates to centralized default handler bundle
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/msw/createDebugHandlers.ts
      Note: |-
        P3.6 reusable MSW handler builder design evidence
        P3.9 delay override enhancement evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/msw/defaultHandlers.ts
      Note: |-
        P3.7 default handler bundle design evidence
        P3.9 default handler override wiring evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios
      Note: P3.3 reusable scenario layer snapshot evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/anomalyScenarios.ts
      Note: P3.13 anomaly scenario catalog evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/eventInspectorScenarios.ts
      Note: P3.12 event-inspector scenario catalog evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/timelineScenarios.ts
      Note: P3.11 timeline scenario catalog expansion evidence
ExternalSources: []
Summary: Detailed execution plan for reducing PI-013 frontend code size by consolidating duplicate helpers, extracting reusable CSS design system layers, and centralizing Storybook mock data generation.
LastUpdated: 2026-02-07T14:20:00-05:00
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

Use a consistent hybrid approach:

- namespaced class names for runtime styling
- optional `data-part` attributes for external theming/extension points

### Recommendation

Adopt **namespaced classes + optional data-part hooks**:

- each component/page has a stable root class (`app-shell`, `timeline-lanes`, `event-inspector`, etc.)
- internal parts use component-prefixed names (`app-header-nav`, `timeline-lane-header`, `anomaly-detail-row`, etc.)
- `data-part` is only added where third-party composition/theming needs semantic hooks
- utility classes from `primitives.css` remain minimal and reusable

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

### 13.8 Progress snapshot (after P3.8 handlers migration to centralized builder)

Captured after commit `00d9a9c` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Inline endpoint definitions in `src/mocks/handlers.ts` | 8 | 0 | replaced by centralized default bundle import |
| `src/mocks/handlers.ts` role | monolithic endpoint implementation | compatibility re-export | now delegates to `defaultHandlers` |

P3.8 details:

- Migrated:
  - `src/mocks/handlers.ts`
- Change:
  - replaced local endpoint definitions with:
    - `import { defaultHandlers } from './msw/defaultHandlers'`
    - `export const handlers = defaultHandlers`

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.15 Progress snapshot (after P3.15 mock architecture README)

Captured after commit `74437ff` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Mock architecture documentation files under `src/mocks` | 0 | 1 | `src/mocks/README.md` |
| Documented mock layers (fixtures/factories/scenarios/msw/legacy shim) | ad hoc knowledge | explicit documented contract | reduces onboarding and drift risk |

P3.15 details:

- Added:
  - `src/mocks/README.md`
- Documented:
  - layer roles and contracts
  - story authoring rules
  - legacy `data.ts` shim policy
  - quick usage examples for scenarios and MSW helpers

Validation:

- `npm run build` passed.

### 13.14 Progress snapshot (after P3.14 story fixture/factory import standardization)

Captured after commit `d76e750` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Story imports of legacy `../mocks/data` | 9 | 0 | eliminated from component stories |
| Story files updated to direct fixture/factory imports | 0 | 9 | switched to `mocks/fixtures` and `mocks/factories` |

P3.14 details:

- Updated stories:
  - `ConversationCard.stories.tsx`
  - `EventCard.stories.tsx`
  - `EventTrackLane.stories.tsx`
  - `MiddlewareChainView.stories.tsx`
  - `ProjectionLane.stories.tsx`
  - `SessionList.stories.tsx`
  - `StateTrackLane.stories.tsx`
  - `TimelineEntityCard.stories.tsx`
  - `TurnInspector.stories.tsx`
- Change:
  - removed legacy compatibility-layer imports
  - switched to shared fixture/factory imports
  - reduced local repeated “many item” composition where applicable (`EventTrackLane`, `SessionList`)

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.13 Progress snapshot (after P3.13 anomaly story centralization)

Captured after commit `cb78165` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Local inline anomaly arrays in `AnomalyPanel.stories.tsx` | 1 | 0 | replaced by anomaly scenarios |
| AppShell story-local anomaly array definitions | 1 | 0 | replaced by `makeAnomalies(2)` |

P3.13 details:

- Updated:
  - `src/components/AnomalyPanel.stories.tsx`
  - `src/components/AppShell.stories.tsx`
- Change:
  - `AnomalyPanel` story args now sourced from `makeAnomalyScenario(...)`
  - `AppShell` anomaly args now sourced from `makeAnomalies(2)`

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.12 Progress snapshot (after P3.12 event inspector story scenario migration)

Captured after commit `56843d6` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| `EventInspector.stories.tsx` stories using local mock composition | 9 | 0 | all migrated to `eventInspectorScenarios` |
| Local helper/mock declarations in `EventInspector.stories.tsx` | 1 block + multiple inline arrays/objects | 0 | replaced by centralized scenario args |

P3.12 details:

- Updated:
  - `src/components/EventInspector.stories.tsx`
- Change:
  - story args switched to `makeEventInspectorScenario(...)`
  - removed local `mockEvents`, `mockTimelineEntities`, and `mockBlock` composition

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.11 Progress snapshot (after P3.11 timeline story scenario migration)

Captured after commit `a3a28ad` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| `TimelineLanes.stories.tsx` stories using local inline mock composition | 7 | 0 | all migrated to scenario builder args |
| Timeline scenario variants available in `timelineScenarios` | 5 | 7 | added `turnsOnly` and `eventsOnly` |

P3.11 details:

- Updated:
  - `src/components/TimelineLanes.stories.tsx`
  - `src/mocks/scenarios/timelineScenarios.ts`
- Change:
  - story args switched from local `mocks/data` composition to `makeTimelineScenario(...)`
  - scenario catalog expanded to cover timeline-only variants previously represented inline

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.10 Progress snapshot (after P3.10 SessionList story handler deduplication)

Captured after commit `2c996cc` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Inline `http.get(...)` blocks in `SessionList.stories.tsx` | 1 | 0 | replaced with centralized helper usage |
| SessionList story variants using centralized handler helpers | 0 | 1 | `WithMSW` now uses `createDefaultDebugHandlers()` |

P3.10 details:

- Updated:
  - `src/components/SessionList.stories.tsx`
- Change:
  - removed local inline endpoint setup
  - switched `WithMSW` story to `createDefaultDebugHandlers()`

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.9 Progress snapshot (after P3.9 AppShell story handler deduplication)

Captured after commit `db9569a` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Inline `http.get(...)` blocks in `AppShell.stories.tsx` | 7 | 0 | replaced with centralized handler helpers |
| Storys using `defaultHandlers` / `createDefaultDebugHandlers` in `AppShell.stories.tsx` | 0 | 3 | meta/default + empty + loading overrides |

P3.9 details:

- Updated:
  - `src/components/AppShell.stories.tsx`
  - `src/mocks/msw/createDebugHandlers.ts`
  - `src/mocks/msw/defaultHandlers.ts`
- Changes:
  - removed story-local endpoint definitions
  - wired base story handlers to `defaultHandlers`
  - wired empty/loading variants to `createDefaultDebugHandlers` with data/delay overrides

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.7 Progress snapshot (after P3.7 default MSW handler bundle)

Captured after commit `b3b0899` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Default MSW handler bundle modules | 0 | 1 | `src/mocks/msw/defaultHandlers.ts` |
| Reusable default handler creation helper exports | 0 | 1 | `createDefaultDebugHandlers` |

P3.7 details:

- Added:
  - `src/mocks/msw/defaultHandlers.ts`
- Introduced:
  - `defaultDebugHandlerData` fixture-backed handler data contract
  - `createDefaultDebugHandlers(dataOverrides?)` for configurable defaults
  - `defaultHandlers` constant for immediate consumer migration in `P3.8`

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.6 Progress snapshot (after P3.6 reusable MSW handler builder)

Captured after commit `1db6da8` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Reusable MSW debug handler builder modules | 0 | 1 | `src/mocks/msw/createDebugHandlers.ts` |
| Debug endpoints represented in centralized builder | 0 | 8 | conversations, conversation detail, sessions, turns, turn detail, events, timeline, mw-trace |

P3.6 details:

- Added:
  - `src/mocks/msw/createDebugHandlers.ts`
- Introduced:
  - `DebugHandlerData` contract for fixture inputs
  - optional deterministic `nowMs` / `nowIso` injection points
  - single function to construct the full debug endpoint handler set

Validation:

- `npm run build` passed.

### 13.4 Progress snapshot (after P3.4 deterministic id/time/seq helpers)

Captured after commit `b8b43aa` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Deterministic id/time/seq utility modules | 0 | 1 | `src/mocks/factories/deterministic.ts` |
| Factory list builders using deterministic synthetic overrides | 0 | 6 | conversation/session, turn, event, timeline, anomaly, app-shell anomaly |
| Unit tests covering deterministic helper behavior | 0 | 1 | `src/mocks/factories/deterministic.test.ts` |

P3.4 details:

- Added utility helpers:
  - `makeDeterministicId`
  - `makeDeterministicTimeMs`
  - `makeDeterministicIsoTime`
  - `makeDeterministicSeq`
  - `shouldApplyDeterministicOverrides`
- Refactored list builders to apply deterministic synthetic overrides only when `absoluteIndex` exceeds fixture list length:
  - `src/mocks/factories/conversationFactory.ts`
  - `src/mocks/factories/turnFactory.ts`
  - `src/mocks/factories/eventFactory.ts`
  - `src/mocks/factories/timelineFactory.ts`
  - `src/mocks/factories/anomalyFactory.ts`
- Added deterministic helper tests:
  - `src/mocks/factories/deterministic.test.ts`

Validation:

- `npm run test:unit` passed (`13` tests).
- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.5 Progress snapshot (after P3.5 legacy `data.ts` compatibility finalization)

Captured after commit `af06ce2` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Legacy `src/mocks/data.ts` status | compatibility re-export file | explicit compatibility shim contract | no inline fixture definitions |
| Remaining imports of `../mocks/data` | 12 | 12 | unchanged intentionally until story migration tasks |

P3.5 details:

- Confirmed `src/mocks/data.ts` remains a pure re-export compatibility layer.
- Added explicit migration guidance comment:
  - retain for current story/handler compatibility
  - remove after `P3.11`-`P3.14` migration completes

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.2 Progress snapshot (after P3.2 deterministic factory layer)

Captured after commit `2e3c954` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Factory modules under `src/mocks/factories` | 0 | 7 | common + domain builders + index export |
| Domains with deterministic builders | 0 | 5 | conversations, turns, events, timeline, anomalies |

P3.2 details:

- Added:
  - `src/mocks/factories/common.ts`
  - `src/mocks/factories/conversationFactory.ts`
  - `src/mocks/factories/turnFactory.ts`
  - `src/mocks/factories/eventFactory.ts`
  - `src/mocks/factories/timelineFactory.ts`
  - `src/mocks/factories/anomalyFactory.ts`
  - `src/mocks/factories/index.ts`
- Pattern:
  - deterministic index-based selection from fixture arrays (`pickByIndex`)
  - clone-safe object generation (`cloneMock`)
  - optional override and list-builder APIs for story/scenario composition

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 12.12 Progress snapshot (after P2.14 + P2.15 + P2.25)

Captured after commit `4e93e85` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Runtime inline `<style>{` blocks | 13 | 0 | all runtime TSX `<style>` blocks removed |

Completed:

- `P2.14` extracted styles from `SnapshotDiff.tsx` to `src/styles/components/SnapshotDiff.css`
- `P2.15` extracted styles from `EventInspector.tsx` to `src/styles/components/EventInspector.css`
- `P2.25` verified no runtime inline `<style>{` blocks remain

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 12.11 Progress snapshot (after P2.17 AnomalyPanel extraction)

Captured after commit `140afcc` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Runtime inline `<style>{` blocks | 16 | 13 | `AnomalyPanel.tsx` inline blocks extracted |

P2.17 details:

- Inline styles moved from:
  - `src/components/AnomalyPanel.tsx`
- Into:
  - `src/styles/components/AnomalyPanel.css`

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 12.10 Progress snapshot (after P2.16 FilterBar extraction)

Captured after commit `51ab056` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Runtime inline `<style>{` blocks | 18 | 16 | `FilterBar.tsx` inline blocks extracted |

P2.16 details:

- Inline styles moved from:
  - `src/components/FilterBar.tsx`
- Into:
  - `src/styles/components/FilterBar.css`

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

### 12.13 Progress snapshot (after P2.22-P2.24 style contract standardization)

Captured after commit `2181eac` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Hard-coded `rgba(...)` in component/primitives style files | 43 | 0 | replaced with token variables; `tokens.css` remains the canonical source |
| Runtime inline `<style>{` blocks | 0 | 0 | preserved |
| Major components with normalized part naming updates | 0 | 3 | `AppShell`, `TimelineLanes`, `AnomalyPanel` class names standardized |

Phase 2C details:

- `P2.22`: documented style contract approach in frontend `README.md` and aligned plan recommendation in this analysis.
- `P2.23`: normalized class/part naming in major components:
  - `src/components/AppShell.tsx` + `src/styles/components/AppShell.css`
  - `src/components/TimelineLanes.tsx` + `src/styles/components/TimelineLanes.css`
  - `src/components/AnomalyPanel.tsx` + `src/styles/components/AnomalyPanel.css`
- `P2.24`: replaced repeated hard-coded alpha colors with shared token variables in:
  - `src/styles/primitives.css`
  - `src/styles/components/SnapshotDiff.css`
  - `src/styles/components/ProjectionLane.css`
  - `src/styles/components/StateTrackLane.css`
  - `src/styles/components/EventTrackLane.css`
  - `src/styles/components/EventInspector.css`
  - `src/styles/components/TimelineLanes.css`
  - `src/styles/components/AnomalyPanel.css`
  - token definitions in `src/styles/tokens.css`

Validation:

- `npm run build` passed:
  - `dist/assets/index-DT1c9Azr.css`: **30.24 kB** (gzip **5.05 kB**)
  - `dist/assets/index-b1WsY2oQ.js`: **330.62 kB** (gzip **105.41 kB**)
- `npm run build-storybook` passed:
  - preview built in **8.67s**
- `rg -n "rgba\\(" src/styles -g '!src/styles/tokens.css'` returned no matches.

### 13.1 Progress snapshot (after P3.1 fixture domain split)

Captured after commit `9f1db57` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Fixture domain files under `src/mocks/fixtures` | 0 | 5 | conversations, turns, events, timeline, anomalies |
| Legacy `src/mocks/data.ts` role | monolithic fixture definitions | compatibility export layer | allows incremental migration |

P3.1 details:

- Added:
  - `src/mocks/fixtures/conversations.ts`
  - `src/mocks/fixtures/turns.ts`
  - `src/mocks/fixtures/events.ts`
  - `src/mocks/fixtures/timeline.ts`
  - `src/mocks/fixtures/anomalies.ts`
- Refactored:
  - `src/mocks/data.ts` to re-export from fixture modules

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.

### 13.3 Progress snapshot (after P3.3 scenario layer)

Captured after commit `fd2efd3` in `web-agent-example`:

| Metric | Before | After | Notes |
|---|---:|---:|---|
| Scenario modules under `src/mocks/scenarios` | 0 | 5 | overview, timeline, event inspector, anomaly, index barrel |
| Reusable scenario families | 0 | 4 | overview/timeline/event-inspector/anomaly contexts |
| Factory-backed scenario builders | 0 | 4 | all scenario families build data from deterministic factories |

P3.3 details:

- Added:
  - `src/mocks/scenarios/overviewScenarios.ts`
  - `src/mocks/scenarios/timelineScenarios.ts`
  - `src/mocks/scenarios/eventInspectorScenarios.ts`
  - `src/mocks/scenarios/anomalyScenarios.ts`
  - `src/mocks/scenarios/index.ts`
- Pattern:
  - typed scenario records plus `make*Scenario` accessors
  - composition from `src/mocks/factories` rather than local inline arrays
  - story-friendly contexts for empty/default/busy/selection/failure cases

Validation:

- `npm run build` passed.
- `npm run build-storybook` passed.
