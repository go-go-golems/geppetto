---
Title: UX Frontend React Code Review Report
Ticket: PI-013-TURN-MW-DEBUG-UI
Status: active
Topics:
    - websocket
    - middleware
    - turns
    - events
    - frontend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: |-
        Current backend route reality for /timeline and /turns
        Backend route and response contract cross-check
    - Path: ../../../../../../../pinocchio/pkg/webchat/turn_store.go
      Note: |-
        Current turn snapshot payload schema (serialized payload)
        Backend turn snapshot payload schema cross-check
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/api/debugApi.ts
      Note: |-
        Frontend API contract and endpoint assumptions
        Frontend endpoint and response contract audit
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/AppShell.tsx
      Note: |-
        App-level state/routing shell
        Shell state and routing integration review
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/EventInspector.tsx
      Note: |-
        Event detail rendering and trust/correlation UI
        Event semantic/SEM/raw inspector review
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/SnapshotDiff.tsx
      Note: |-
        Diff algorithm and performance/identity logic
        Diff algorithm and complexity review
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/TurnInspector.tsx
      Note: |-
        Turn inspection UI and compare flow
        Phase compare and detail flow review
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/main.tsx
      Note: |-
        Dev bootstrap and MSW wiring behavior
        MSW dev bootstrap behavior reviewed
ExternalSources: []
Summary: Exhaustive frontend code review of PI-013 React implementation commits, including architecture mapping, integration readiness, strengths, failures, and prioritized remediation plan.
LastUpdated: 2026-02-07T12:34:00-05:00
WhatFor: Give engineering + UX leadership a full quality assessment of the delivered React debug UI code and a practical path to production readiness.
WhenToUse: Use when deciding whether to continue from this frontend baseline, prioritizing PI-014/015 backend work, and planning cleanup/refactor tasks before wider team onboarding.
---


# PI-013 UX Frontend React Code Review (Exhaustive)

## 0) Executive Summary

### Verdict

The UX frontend programmer delivered a **very strong UI exploration/prototyping pass** in a very short burst of commits, with broad surface-area coverage (scaffold, data layer, state layer, 18 non-story components, 18 stories, route shell).

However, as currently committed, this work is **prototype-grade, not integration-ready**.

### What is genuinely good

- Storybook-first development discipline is visible across nearly all components.
- Visual semantics for turns/events/timeline entities are thoughtful and consistent.
- A lot of important conceptual UI has been made concrete quickly (screen lanes, diff model, event inspector modes, anomaly panel, filter panel, shell + routing).
- Build quality for the frontend package itself is currently healthy (`npm run build` and `npm run build-storybook` pass).

### What blocks production use today

1. **API contract drift** between frontend assumptions and real backend routes/schema.
2. **MSW is always on in dev**, so live backend integration is effectively masked.
3. **Core interaction wiring is incomplete** (selection/diff/filter/anomaly state mostly not connected end-to-end).
4. **High style/runtime duplication** from `<style>{`...`}</style>` in many component render paths.
5. **Large-component complexity + duplicate helpers + no tests/lint gates** create medium-term maintainability risk.

### Final recommendation

- Keep this as the **UI reference baseline**.
- Do **not** treat it as production-ready debug UI yet.
- Execute a short stabilization phase immediately after PI-014 contract work:
  - contract alignment,
  - runtime wiring completion,
  - style extraction,
  - minimal automated tests,
  - accessibility pass.

---

## 1) Review Scope and Method

## 1.1 Commits reviewed (frontend burst)

Reviewed the full React burst in `web-agent-example`:

1. `465663a` — scaffold
2. `9ad90bc` — core components + Storybook + MSW + state/api/types
3. `aaadbc5` — Screen 1 lanes
4. `d103241` — Snapshot Diff
5. `f0ce933` — Event Inspector
6. `09951e0` — FilterBar + AnomalyPanel
7. `ab6d31d` — AppShell + routes

### Delta size

- `git diff --shortstat 0a8be5b..ab6d31d`
- **62 files changed, 14,230 insertions**

### Current frontend surface

- TS/TSX files under `src`: **52**
- Non-story components: **18**
- Story files: **18**
- `src` LOC total (ts/tsx/css): **8,215**
- Story LOC: **1,750**
- Non-story LOC: **6,465**

## 1.2 Runtime validation performed

Executed:

- `npm run build` ✅
- `npm run build-storybook` ✅ (with expected chunk-size warnings)

Interpretation:

- TypeScript + Vite compile state is clean.
- Storybook bundle builds, but this does **not** validate backend integration correctness.

## 1.3 Review standard

This review evaluated:

- architecture and flow cohesion,
- backend contract fit,
- state model coherence,
- rendering correctness/perf risk,
- accessibility and UX operability,
- maintainability/refactor cost,
- delivery value vs technical debt introduced.

---

## 2) What Exists Today (Inventory by Capability)

## 2.1 Infrastructure

- Vite + React + TS scaffold
- Storybook 8 config
- MSW handlers + large mock data corpus
- RTK Query API slice
- Redux UI slice and store

## 2.2 Implemented UI modules

- Session/Conversation listing
- Turn inspector + phase tabs
- Timeline lanes (state/events/projection)
- Snapshot diff screen component
- Event inspector (semantic/SEM/raw modes)
- Filter bar
- Anomaly panel
- App shell + route scaffold

## 2.3 Storybook coverage

Nearly all primary components have stories, often with multiple states.

This is a real strength and reduces onboarding friction for future UI collaborators.

---

## 3) What Works Well (High-Value Positives)

## 3.1 Breadth-first UI delivery was excellent

Given the scope, the programmer shipped a broad and coherent first cut quickly. This is exactly what PI-013 needed for concept proving.

## 3.2 Visual language consistency

Badge/chip color coding across block kinds, event families, and entity kinds is mostly consistent and readable.

## 3.3 Storybook-first discipline

The presence of stories for core and advanced components gives the project a strong future testability/documentation base.

## 3.4 Separation of domain types from component props (partially)

A central type file exists and is used by many components. This can be evolved into a stable contract layer once backend alignment is done.

## 3.5 Diff and inspector UX intent is strong

Even where wiring is incomplete, the conceptual model is right:

- side-by-side diff
- trust checks
- correlated nodes
- phase-aware turn exploration

This is the right product direction.

---

## 4) Critical Findings (must-fix before real use)

### 4.1 Frontend contract assumes `/debug/*` routes that current backend does not expose

**Problem**  
Frontend API base is hard-coded to `/debug/`, but current backend routes are `/turns`, `/timeline`, `/chat`, `/ws`, etc.

**Where**
- `web-agent-example/.../src/api/debugApi.ts:47` (`baseUrl: '/debug/'`)
- `pinocchio/pkg/webchat/router.go:666-667` (`/timeline`)
- `pinocchio/pkg/webchat/router.go:727-728` (`/turns`)
- `pinocchio/pkg/webchat/router.go:818-819` (`/chat`)

**Why it matters**  
The app cannot integrate with current backend routes without API adaptation/migration.

**Cleanup sketch**
```ts
// Temporary contract adapter layer (until PI-014 lands)
const DEBUG_PREFIX_ENABLED = import.meta.env.VITE_USE_DEBUG_PREFIX === '1'
const BASE = DEBUG_PREFIX_ENABLED ? '/debug/' : '/'

baseQuery: fetchBaseQuery({ baseUrl: BASE })
```
And provide per-endpoint translators.

---

### 4.2 `/turns` response-shape mismatch (frontend expects array, backend returns envelope with `items`)

**Problem**  
`getTurns` is typed as `TurnSnapshot[]` and returns raw response, while backend returns:
`{ conv_id, session_id, phase, since_ms, items: [...] }`

**Where**
- `src/api/debugApi.ts:72-84`
- `pinocchio/pkg/webchat/router.go:717-722`

**Why it matters**  
App code consuming `turns` as an array will break against real backend payloads.

**Cleanup sketch**
```ts
getTurns: builder.query<TurnSnapshot[], TurnQuery>({
  query: (...) => `turns?...`,
  transformResponse: (r: { items: TurnSnapshot[] }) => r.items,
})
```

---

### 4.3 Turn payload schema mismatch (backend currently stores serialized payload string)

**Problem**  
Frontend expects `turn: ParsedTurn` on each snapshot. Backend `TurnSnapshot` currently exposes `payload: string`.

**Where**
- `src/types/index.ts` (`TurnSnapshot.turn`)
- `pinocchio/pkg/webchat/turn_store.go:6-12` (`Payload string`)

**Why it matters**  
Without parsing/contract migration, turn views cannot render real data.

**Cleanup sketch**
- Either:
  1) backend upgrades `/debug/turns` contract to parsed shape (PI-014 path), or
  2) frontend introduces decode adapter from serialized payload.

---

### 4.4 Timeline schema mismatch risk (protojson vs expected flat entity props)

**Problem**  
Frontend expects `TimelineEntity { id, kind, created_at, props }`, while current timeline responses are protojson-driven and represented differently in CLI tooling (`convId`, `entityId`, etc.).

**Where**
- `src/types/index.ts:124+`
- `web-agent-example/cmd/web-agent-debug/timeline.go:23-28`
- `pinocchio/pkg/webchat/router.go:666+` (protojson timeline output)

**Why it matters**  
Projection lane is likely to fail or silently mis-render with real backend payloads.

**Cleanup sketch**
Create a strict adapter layer:
```ts
function fromTimelineSnapshotV1(raw: unknown): TimelineSnapshot {
  // normalize ids, timestamps, props payload into UI model
}
```

---

### 4.5 Dev mode always starts MSW, masking backend integration

**Problem**  
In dev, MSW starts unconditionally.

**Where**
- `src/main.tsx:10-12`

```ts
if (import.meta.env.DEV) {
  const { worker } = await import('./mocks/browser');
  return worker.start({ onUnhandledRequest: 'bypass' });
}
```

**Why it matters**  
Developers can think integration works while only exercising mocks.

**Cleanup sketch**
Gate with explicit env flag:
```ts
const USE_MSW = import.meta.env.VITE_USE_MSW === '1'
if (import.meta.env.DEV && USE_MSW) { ... }
```

---

## 5) High-Severity Findings (major quality/maintainability issues)

### 5.1 Core interactions are only partially wired end-to-end

**Problem**  
Selection and inspection flows are incomplete:
- `TimelineLanes` is rendered without selection callbacks in route pages.
- `selectedTurnId` gate exists but there is no route-level dispatch path from lane card click to store.

**Where**
- `src/routes/OverviewPage.tsx:63+` (no `onTurnSelect`)
- `src/routes/TimelinePage.tsx:54+` (no `onEventSelect` / `onEntitySelect`)
- `src/store/uiSlice.ts` has actions, but they are mostly unused outside specific components.

**Why it matters**  
UI looks feature-complete but user workflows stall.

**Cleanup sketch**
Create a controller hook per page:
```ts
const onTurnSelect = (turn) => {
  dispatch(selectTurn(turn.turn_id))
  navigate(`/session/${turn.session_id}/turn/${turn.turn_id}`)
}
```

---

### 5.2 Compare controls in TurnInspector have no rendered diff output

**Problem**  
User can choose phase A/B but `SnapshotDiff` is not integrated into `TurnInspector` rendering.

**Where**
- `src/components/TurnInspector.tsx:17-18, 73-99`
- `src/components/SnapshotDiff.tsx` exists but not imported in runtime route path.

**Why it matters**  
A major UX promise appears present but isn’t functionally delivered.

**Cleanup sketch**
```tsx
{comparePhaseA && comparePhaseB && phases[comparePhaseA] && phases[comparePhaseB] && (
  <SnapshotDiff
    phaseA={comparePhaseA}
    phaseB={comparePhaseB}
    turnA={phases[comparePhaseA]!.turn}
    turnB={phases[comparePhaseB]!.turn}
  />
)}
```

---

### 5.3 App shell state is split across local state and Redux (model drift)

**Problem**  
AppShell uses local `useState` for sidebar/filter/anomaly states while `uiSlice` has canonical fields/actions for the same concerns.

**Where**
- Local: `src/components/AppShell.tsx:21-24`
- Global: `src/store/uiSlice.ts:18-22, 97-113`

**Why it matters**  
Dual sources of truth make future behavior unpredictable and harder to test.

**Cleanup sketch**
- Move all shell state to Redux OR keep all shell state local and remove slice fields.
- Do not keep both.

---

### 5.4 Filter state model is inconsistent across component/store

**Problem**  
`FilterBar.FilterState` and `uiSlice.filters` are different shapes.

**Where**
- `src/components/FilterBar.tsx:4-9`
- `src/store/uiSlice.ts:25-29`

**Why it matters**  
Future connection of filter UI to global state will cause avoidable refactor churn.

**Cleanup sketch**
Define a single shared `UiFilters` type in `src/types` and use it everywhere.

---

### 5.5 Massive inline style-tag injection pattern across the app

**Problem**  
There are many embedded `<style>{`...`}</style>` blocks in component render trees, including repeated per-card/per-row renderers.

**Where**
- 31 style blocks across routes/components (sample: `StateTrackLane.tsx:97`, `EventTrackLane.tsx:78`, `ProjectionLane.tsx:105`, `SnapshotDiff.tsx:63/124/192/275/...`).

**Why it matters**  
- Runtime DOM/style duplication.
- Harder theme refactors.
- Performance cost for large data lists.

**Cleanup sketch**
- Extract to `index.css` sections, CSS modules, or styled-system tokens.
- Keep render functions purely structural.

---

### 5.6 Key UI components are monolithic and overgrown

**Problem**  
Large files bundle state, view logic, helper functions, and styles.

**Where**
- `SnapshotDiff.tsx` (~628 LOC)
- `EventInspector.tsx` (~606 LOC)
- `AnomalyPanel.tsx` (~442 LOC)
- `FilterBar.tsx` (~326 LOC)

**Why it matters**  
High cognitive load and high regression risk when changing one behavior.

**Cleanup sketch**
Split by concern:
- `.../SnapshotDiff/{DiffHeader,DiffRow,MetadataDiff,algo.ts}`
- `.../EventInspector/{ViewTabs,SemanticView,SemView,RawView,TrustSignals}`

---

### 5.7 Accessibility gaps for clickable non-button regions

**Problem**  
Many clickable cards are `div` elements without keyboard semantics.

**Where**
- `ConversationCard.tsx:23`
- `EventCard.tsx:21`
- `TimelineEntityCard.tsx:19`
- `StateTrackLane.tsx:69`
- `EventTrackLane.tsx:63`
- `ProjectionLane.tsx:63`
- `AnomalyPanel.tsx:225`

**Why it matters**  
Keyboard and assistive-tech users cannot reliably operate these interactions.

**Cleanup sketch**
Use `<button>` semantics where possible; otherwise add:
- `role="button"`
- `tabIndex={0}`
- `onKeyDown` for Enter/Space
- visible focus states

---

## 6) Medium-Severity Findings (important, but not immediate blockers)

### 6.1 Diff algorithm has correctness/perf caveats

**Problem**  
`computeBlockDiffs` is O(n²) and uses repeated `JSON.stringify` deep compares. Reorder detection also compares against match array index, not block index semantics.

**Where**
- `src/components/SnapshotDiff.tsx:517+`
- `getBlockStatus(..., newIndex)` at `574+`

**Why it matters**  
Potentially expensive on large turns and can misclassify reorder situations.

**Cleanup sketch**
- Precompute hashes once.
- Match by stable id first, fallback by hash index map.
- Compare `blockA.index` to `blockB.index`, not array position.

---

### 6.2 64-bit numeric identifiers modeled as `number`

**Problem**  
`seq` and potentially timeline version-like values are modeled as JS `number` despite int64/uint64-style data.

**Where**
- `src/types/index.ts:65, 127, 135`
- Mock seq examples > `Number.MAX_SAFE_INTEGER`: `src/mocks/data.ts:556+`

**Why it matters**  
Precision loss risk in sorting, equality checks, and keying.

**Cleanup sketch**
Model IDs/seq/version as strings at API boundary.

---

### 6.3 Duplicate domain helper logic across components

**Problem**  
Similar helpers are reimplemented multiple times (`getEventTypeInfo`, `getKindInfo`, `getKindIcon`, `truncateText`, phase formatting).

**Where**
- `EventCard.tsx`, `EventTrackLane.tsx`, `EventInspector.tsx`
- `ProjectionLane.tsx`, `TimelineEntityCard.tsx`
- `StateTrackLane.tsx`, `BlockCard.tsx`, etc.

**Why it matters**  
Inconsistent behavior/drift over time.

**Cleanup sketch**
Move to shared utilities:
- `src/ui/formatters.ts`
- `src/ui/eventPresentation.ts`
- `src/ui/blockPresentation.ts`

---

### 6.4 Scroll sync implementation may produce feedback churn

**Problem**  
Scroll listeners write `scrollTop` to sibling lanes directly and symmetrically.

**Where**
- `src/components/TimelineLanes.tsx:40-54`

**Why it matters**  
Potential event ping-pong/jank in heavy lane content.

**Cleanup sketch**
Use guard flag or RAF lock to prevent re-entrant updates.

---

### 6.5 Unused and partially integrated components in runtime app

**Problem**  
Some components appear story-only currently (not wired in route runtime):
- `SnapshotDiff`
- `MiddlewareChainView`
- `TimelineEntityCard` (ProjectionLane uses a separate internal card)

**Why it matters**  
Code duplication + false sense of feature completeness.

**Cleanup sketch**
- Either wire now or explicitly mark as phase-2 and remove dead runtime exports until needed.

---

### 6.6 `App.tsx` is stale/dead parallel shell

**Problem**  
`App.tsx` contains old route shell (“Turn Inspector - Coming soon”) but main entry uses `AppRouter`.

**Where**
- `src/App.tsx`
- `src/main.tsx` uses `AppRouter`

**Why it matters**  
Confusing to contributors and increases maintenance noise.

**Cleanup sketch**
Remove `App.tsx` or convert it into a tested fallback/demo route intentionally.

---

### 6.7 Storybook uses shared store instance across stories

**Problem**  
Single store instance provided in global Storybook decorator can leak state between stories.

**Where**
- `.storybook/preview.tsx:16-19`

**Why it matters**  
Story determinism degrades over long sessions.

**Cleanup sketch**
Use a store factory per story/decorator invocation.

---

### 6.8 No lint/test scripts in package scripts

**Problem**  
Package only has `dev/build/preview/storybook/build-storybook`.

**Where**
- `package.json:5-10`

**Why it matters**  
No CI-grade guardrails for regressions in logic-heavy components.

**Cleanup sketch**
Add:
- `lint`
- `test` (vitest)
- `test:ui` for algorithmic components (diff, filters)

---

## 7) Low-Severity Findings / Polish Opportunities

1. Responsive behavior is deferred but fixed widths are widespread (`300/280/320px`).
2. `navigator.clipboard.writeText` in `CorrelationIdBar` has no failure handling.
3. `seq && ...` in correlation chip builder omits zero values.
4. Anomaly model is duplicated (`src/types` vs `AnomalyPanel` local type).
5. `strict` TS is on, but `noUnusedLocals` and `noUnusedParameters` are disabled, hiding cleanup signals.

---

## 8) Screen-by-Screen Review

## 8.1 Screen 1: Session Overview / Timeline Lanes

**What works**
- Clear three-lane conceptual layout.
- Good visual separation and lane headers.
- NowMarker concept is useful for live mode.

**What doesn’t**
- Selection callbacks not wired at page level.
- Overview route feeds empty events/entities currently.
- Scroll-sync strategy needs hardening.

**Recommendation**
- Wire lane selection to store + navigation now.
- Share lane card components where possible.

## 8.2 Screen 3: Snapshot Diff

**What works**
- Good summary chips and status coding.
- Identity-aware intent exists.

**What doesn’t**
- Not integrated into turn workflow.
- Diff algo needs optimization/correctness tightening.

**Recommendation**
- Add unit tests for diff edge cases (duplicate payloads, reorder-only, metadata-only changes).

## 8.3 Screen 5: Event Inspector

**What works**
- Nice tri-view mode concept.
- Trust and correlated-node framing is very useful.

**What doesn’t**
- Raw wire mode is placeholder in current implementation.
- Correlation IDs extracted from `event.data`, but actual envelope fields may differ by source.

**Recommendation**
- Move correlation extraction to robust normalizer.

## 8.4 Screen 7: Filter Bar

**What works**
- UX is clean and understandable.
- Select-all and clear-all affordances are present.

**What doesn’t**
- Not applied to lane/event lists in runtime pages.
- State model mismatch with global store.

**Recommendation**
- Connect filter state to selectors and memoized filtered collections.

## 8.5 Screen 8: Anomaly Panel

**What works**
- Good severity filtering and detail panel.

**What doesn’t**
- No data source integration in app path.
- Type model drift from global anomaly type.

**Recommendation**
- Back anomalies with selector/service and unify anomaly DTO type.

## 8.6 AppShell + Routing

**What works**
- Clear shell framing and top nav.
- Good right-side panel concept for filters.

**What doesn’t**
- Shell state duplicated (local + Redux model).
- Filter context passed through `<Outlet context>` but not consumed.
- Deep-linking model still weak (selection mostly in store, not URL).

**Recommendation**
- Decide canonical state ownership and encode key state in route/query params.

---

## 9) Architecture Map and Readiness Assessment

## 9.1 Current architecture quality rating

- **Prototype UX quality:** High
- **Integration readiness:** Low
- **Maintainability readiness:** Medium-Low
- **Testability readiness:** Medium (thanks to stories, but low automated assertions)

## 9.2 Runtime flow reality

Current practical runtime in dev is:

1. App boots
2. MSW starts unconditionally (dev)
3. API calls hit mocks
4. Rich UI appears functional
5. Real backend discrepancies remain hidden

This is useful for UI exploration, but dangerous if mistaken for integrated readiness.

---

## 10) Prioritized Remediation Plan

## Phase A (must do before claiming backend integration)

1. Add explicit `VITE_USE_MSW` toggle and default it off for integration sessions.
2. Build API normalizer/adapters for current backend payloads.
3. Wire selection flows (`turn/event/entity`) end-to-end in routes.
4. Integrate `SnapshotDiff` into turn compare experience.

## Phase B (stabilization)

5. Consolidate AppShell + uiSlice state ownership.
6. Unify filter model and apply filters to data views.
7. Unify anomaly model and data source.
8. Remove stale `App.tsx` and dead branches.

## Phase C (quality hardening)

9. Extract CSS from runtime style tags.
10. Split monolithic components.
11. Add unit tests for diff/correlation/filter logic.
12. Add lint/test scripts and CI checks.

## Phase D (UX + accessibility)

13. Convert clickable divs to accessible controls.
14. Add keyboard support and focus states.
15. Implement responsive layout behavior for narrow widths.

---

## 11) What We Learned

1. The team can move from concept to concrete UI very quickly when story-driven.
2. Fast UI prototyping without contract adapters creates hidden integration debt.
3. Storybook success is not integration success.
4. For PI-013 class features, API normalization is not optional—it is foundational.
5. The fastest path now is not rewriting everything; it is **stabilizing this baseline with explicit adapters and wiring**.

---

## 12) “What works / What doesn’t / Better / Learned” Condensed Matrix

| Area | What works | What doesn’t | Better next step | What we learned |
|---|---|---|---|---|
| Component design | Broad, coherent UI kit | Some components not runtime-wired | Integrate or park deliberately | Prototype momentum is strong |
| Data layer | Clean RTK Query skeleton | Endpoint/schema drift | Add adapters + contract tests | Contract-first avoids rework |
| State layer | uiSlice has needed concepts | Dual source-of-truth with local state | Choose one state owner | Coherence matters more than volume |
| Styling | Visual consistency is good | 31 inline style blocks | Extract to module/global CSS | Runtime style injection scales poorly |
| QA | Build and Storybook pass | No tests/lint and masked backend | Add test/lint + live mode toggle | “Green build” can still hide critical gaps |

---

## 13) Final Recommendation to Leadership

Keep this code. Do not discard it.

But treat it as **Phase-1 UI prototype implementation** and schedule a short hardening sprint before wider rollout.

If we execute the remediation plan above, this can become a strong production debug UI foundation with far less effort than a restart.

---

## 14) Appendix A — Commands run during review

```bash
cd web-agent-example

git log --oneline --decorate --graph --max-count=30
git diff --name-status 0a8be5b..ab6d31d
git show --stat --oneline 465663a 9ad90bc aaadbc5 d103241 f0ce933 09951e0 ab6d31d

cd cmd/web-agent-debug/web
npm run build
npm run build-storybook

rg -n "import.meta.env.DEV|worker.start" src/main.tsx
rg -n "baseUrl: '/debug/'|getTurns" src/api/debugApi.ts
rg -n "Outlet context=\{\{ filters \}\}" src/components/AppShell.tsx
rg -n "useOutletContext" src

cd ../../../../pinocchio
rg -n "mux.HandleFunc\(\"/timeline|mux.HandleFunc\(\"/turns" pkg/webchat/router.go
rg -n '"items":\s*items' pkg/webchat/router.go
rg -n "type TurnSnapshot|Payload\s+string" pkg/webchat/turn_store.go
```

---

## 15) Appendix B — Suggested immediate task tickets

1. **PI-013-FE-001**: Dev runtime toggle for MSW + integration mode docs.
2. **PI-013-FE-002**: API adapter layer for turns/timeline/event envelopes.
3. **PI-013-FE-003**: Turn selection and compare flow completion.
4. **PI-013-FE-004**: CSS extraction from inline style blocks.
5. **PI-013-FE-005**: Accessibility pass for clickable cards.
6. **PI-013-FE-006**: Diff algorithm hardening + tests.
7. **PI-013-FE-007**: State model consolidation (AppShell + uiSlice).

---

## 16) Commit-by-Commit Forensic Review

This section documents what each commit contributed, what was good in that commit, and what debt was introduced at that same step.

### 16.1 `465663a` — Frontend scaffold

**Added**
- `package.json`
- `tsconfig.json`
- `vite.config.ts`
- initial README

**What worked**
- Good immediate scaffold choice (Vite + TS + React).
- Storybook planned from day one.

**What didn’t**
- No testing/lint scripts established at scaffold time, which later allowed logic-heavy code to grow without guardrails.

**Lesson**
- Scaffold is the best moment to bake quality gates.

### 16.2 `9ad90bc` — Core baseline (huge foundational commit)

**Added**
- Store, API, types, initial components, styles, mocks, stories, Storybook setup.

**What worked**
- Extremely high velocity with broad feature baseline.
- Good UX semantics and story coverage pattern.

**What didn’t**
- High amount of coupled code landed at once; no incremental integration checkpoints.
- Contract assumptions were encoded directly into API layer without adapter boundary.

**Lesson**
- Large burst commits are acceptable for prototypes but should be followed by a dedicated stabilization checkpoint.

### 16.3 `aaadbc5` — Screen 1 lanes

**Added**
- Timeline lanes + per-lane cards + now marker.

**What worked**
- Excellent structural decomposition of lane concept.

**What didn’t**
- Scroll sync implemented fast but without anti-feedback guard.
- Repeated per-item style blocks started to accumulate technical debt.

### 16.4 `d103241` — SnapshotDiff

**Added**
- Complete diff UI and algorithm.

**What worked**
- Diff feature is thoughtfully designed at UX level.

**What didn’t**
- Algorithmic complexity and edge-case correctness were not backed by tests.
- Integration into actual turn inspector flow remained pending.

### 16.5 `f0ce933` — EventInspector

**Added**
- Semantic/SEM/raw modes, trust checks, correlated nodes.

**What worked**
- Very strong exploratory UX for debugging semantics.

**What didn’t**
- Raw mode remains placeholder for real wire payload shape.
- Component became large and multi-responsibility quickly.

### 16.6 `09951e0` — FilterBar + AnomalyPanel

**Added**
- Filtering UI and anomaly UI.

**What worked**
- Good operator affordances and discoverability.

**What didn’t**
- Data path integration not completed (UI mostly shell-level with mock usage).
- Type drift introduced (`Anomaly` in multiple places with different shape assumptions).

### 16.7 `ab6d31d` — AppShell + route integration

**Added**
- Router + app shell + route pages.

**What worked**
- Big move from component library to app skeleton.

**What didn’t**
- AppShell local state and uiSlice global state diverged.
- Several workflows still stop at route-level glue boundaries.

---

## 17) File-by-File Runtime Review Matrix (Non-Story Components)

This section covers every runtime component introduced, with focused review notes.

### 17.1 `src/main.tsx`

**Status:** Compiles, but integration-risky.

- ✅ Clean app bootstrap.
- ❌ MSW starts unconditionally in dev (`import.meta.env.DEV`), masking live backend integration by default.
- ✅ Recommendation: add explicit `VITE_USE_MSW` guard.

### 17.2 `src/api/debugApi.ts`

**Status:** Strong skeleton, contract-risky.

- ✅ Good RTK Query structure and endpoint grouping.
- ❌ Hard-coded `/debug/` base path does not match current router reality.
- ❌ `getTurns` transform missing for envelope payload (`items`).
- ❌ Turn/timeline/event DTO assumptions not protected via adapters.

### 17.3 `src/store/uiSlice.ts`

**Status:** Rich state model, partially unused.

- ✅ Captures intended UX state domain.
- ❌ Multiple fields/actions currently unused in runtime path.
- ❌ Filter and panel state drift from AppShell local state.

### 17.4 `src/routes/index.tsx`

**Status:** Functional route skeleton.

- ✅ App shell nesting and route hierarchy are clear.
- ⚠️ Deep-link semantics still weak because selection source remains mostly store-driven.

### 17.5 `src/routes/OverviewPage.tsx`

**Status:** Partially wired.

- ✅ Good empty/loading states.
- ❌ Timeline lane callbacks not wired.
- ❌ Turn detail is gated on `selectedTurnId`, but selection path is incomplete.

### 17.6 `src/routes/TimelinePage.tsx`

**Status:** Read-only presentational currently.

- ✅ Loads turns/events/timeline in one place.
- ❌ No interactive selection wiring to inspector flows.

### 17.7 `src/routes/EventsPage.tsx`

**Status:** Functional within mock contract.

- ✅ Event list + inspector split works locally.
- ⚠️ Depends on endpoint shape currently not present in backend.

### 17.8 `src/routes/TurnDetailPage.tsx`

**Status:** Defensive UI, backend contract pending.

- ✅ Handles missing IDs and loading/error states.
- ❌ Depends on `/turn/:conv/:session/:turn` endpoint not currently in backend.

### 17.9 `src/components/AppShell.tsx`

**Status:** UX shell is good; architecture needs consolidation.

- ✅ Good navigation and panel affordances.
- ❌ Local-state duplication for sidebar/filter/anomaly.
- ❌ Outlet context carries filters but route consumers do not use it.

### 17.10 `src/components/SessionList.tsx`

**Status:** Good base, minor behavior debt.

- ✅ Handles loading/error/empty states.
- ⚠️ API hook still runs even when override data is provided (storybook use-case), producing avoidable background work.

### 17.11 `src/components/ConversationCard.tsx`

**Status:** Good summary card.

- ✅ Useful compact metadata.
- ⚠️ Clickable div needs keyboard accessibility support.

### 17.12 `src/components/TurnInspector.tsx`

**Status:** Strong intent, partially completed flow.

- ✅ Good phase tabs and metadata view.
- ❌ Compare controls not connected to actual diff display.
- ⚠️ Keying by `kind-index` can be unstable for reorder-heavy sequences.

### 17.13 `src/components/BlockCard.tsx`

**Status:** Useful block renderer, could be split.

- ✅ Handles text/tool call/tool result cases.
- ✅ Metadata expand behavior is practical.
- ⚠️ Inline style and helper duplication increase maintenance cost.

### 17.14 `src/components/TimelineLanes.tsx`

**Status:** Conceptually strong.

- ✅ Lane architecture is clear and extensible.
- ❌ Scroll sync implementation needs loop prevention guard.

### 17.15 `src/components/StateTrackLane.tsx`

**Status:** Good visual lane carding.

- ✅ Fast readability for snapshot phases.
- ⚠️ Repeated inline style tag in per-card renderer.

### 17.16 `src/components/EventTrackLane.tsx`

**Status:** Good event lane summary.

- ✅ Type badge semantics are helpful.
- ⚠️ Duplicated event-presentation helper logic vs EventCard/EventInspector.

### 17.17 `src/components/ProjectionLane.tsx`

**Status:** Works, but duplicates another card implementation.

- ✅ Projection summary logic is good.
- ❌ Internal `EntityCard` duplicates `TimelineEntityCard` purpose with separate helper logic.

### 17.18 `src/components/SnapshotDiff.tsx`

**Status:** High-value feature with algorithm debt.

- ✅ UI and statuses are strong.
- ❌ O(n²) and stringification-heavy matching.
- ❌ Reorder detection subtle bug risk (`newIndex` positional compare).

### 17.19 `src/components/EventInspector.tsx`

**Status:** Excellent prototype value, high complexity.

- ✅ Useful multi-mode event analysis model.
- ❌ Large file and mixed responsibilities.
- ❌ Raw-wire mode remains mostly placeholder behavior.

### 17.20 `src/components/FilterBar.tsx`

**Status:** Usable UI, disconnected data path.

- ✅ Good filter ergonomics.
- ❌ Not consistently connected to route-level selectors.

### 17.21 `src/components/AnomalyPanel.tsx`

**Status:** Solid shell behavior.

- ✅ Good severity filtering and detail expansion.
- ❌ No runtime anomaly pipeline integration yet.
- ❌ Separate anomaly type model from global types.

### 17.22 `src/components/NowMarker.tsx`

**Status:** Fine visual utility.

- ✅ Lightweight and clear.
- ⚠️ Should be tied to real live-state and frame timing hooks once websocket flow lands.

---

## 18) Deep Issue Catalog with Concrete Fix Sketches

### 18.1 Contract Adapter Layer (recommended skeleton)

```ts
// src/api/adapters.ts
export function normalizeTurnsResponse(raw: unknown): TurnSnapshot[] {
  const r = raw as { items?: unknown[] }
  const items = Array.isArray(r?.items) ? r.items : []
  return items.map(normalizeTurnSnapshot)
}

function normalizeTurnSnapshot(x: unknown): TurnSnapshot {
  // support current payload:string and future parsed object
  // parse defensively with validation
}
```

### 18.2 One-state-owner AppShell refactor

```tsx
// AppShell.tsx
const sidebarCollapsed = useAppSelector(s => s.ui.sidebarCollapsed)
const filterOpen = useAppSelector(s => s.ui.filterBarOpen)
const anomalyOpen = useAppSelector(s => s.ui.anomalyPanelOpen)
const dispatch = useAppDispatch()

<button onClick={() => dispatch(toggleSidebar())} />
```

### 18.3 Diff algorithm hardening sketch

```ts
// SnapshotDiff/algo.ts
const byIdB = new Map(blocksB.filter(b => b.id).map(b => [b.id!, b]))
const byHashB = multimap(blocksB, b => stableHash({kind: b.kind, payload: b.payload}))

// match by id -> hash -> added/removed
// compare index with blockB.index (not array position)
```

### 18.4 Styling extraction migration sketch

1. Move all local `<style>` blocks into module files:
   - `AppShell.css`
   - `TimelineLanes.css`
   - `SnapshotDiff.css`
2. Keep token usage through `:root` variables in `index.css`.
3. Delete inline style tags from render paths.
4. Add visual regression snapshots in Storybook for parity.

### 18.5 Accessibility retrofit sketch

```tsx
<div
  role="button"
  tabIndex={0}
  onKeyDown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') onClick?.()
  }}
  onClick={onClick}
/>
```

(Prefer `<button>` where semantics allow.)

---

## 19) Test Strategy Needed for This Codebase

### 19.1 Unit tests (highest ROI)

1. `computeBlockDiffs`:
   - duplicate blocks with same payload
   - reorder-only changes
   - metadata-only changes
   - id-present vs id-absent matching
2. Event presentation mappers:
   - unknown event types
   - missing fields
3. Adapter normalization tests:
   - current backend shape
   - planned `/debug/*` shape

### 19.2 Component behavior tests

- TurnInspector phase selection + compare integration.
- FilterBar state changes reflected in filtered lists.
- AnomalyPanel severity filters and detail toggles.

### 19.3 Integration tests (lightweight)

- “No MSW” integration profile against real backend fixture server.
- Smoke route test for:
  - select conversation,
  - open turn detail,
  - open event inspector,
  - apply filters.

---

## 20) Operational Risk Scenarios if Shipped As-Is

1. **False confidence risk:** Dev appears healthy due mocks; production integration fails on first real payload.
2. **Debugging blind spot risk:** Selection/compare/filter workflows may appear present but be operationally incomplete.
3. **Performance drift risk:** Large sessions amplify style-tag and diff-compute overhead.
4. **Maintainability drag risk:** Monolithic files slow onboarding and increase bug-introducing edit scope.

---

## 21) Suggested Ownership Plan

- **Frontend lead:** state consolidation + wiring completion + style extraction.
- **Backend lead (PI-014):** route/contract migration and normalized debug API delivery.
- **QA/tooling:** test harness + lint gates + integration profile.
- **UX lead:** validate behavior after wiring (especially compare/filter/anomaly flows).

---

## 22) Exit Criteria for “Review Findings Resolved”

The review should be considered resolved only when all criteria below are true:

1. App runs with live backend in dev **without** MSW and key screens load real data.
2. `SnapshotDiff` is integrated and tested via compare controls.
3. Filter controls alter visible lane/event datasets.
4. Anomaly panel is fed by a real or deterministic analysis source.
5. Inline style-tag pattern is removed from per-item render paths.
6. Unit tests cover diff + adapter logic and pass in CI.
7. Accessibility baseline met for interactive cards.

---

## 23) Closing Note

This frontend work should be treated as a successful and valuable **prototype acceleration milestone**. The right next step is disciplined hardening, not reimplementation.

If we preserve the current UX momentum and immediately apply the remediation plan, this can become one of the most useful debugging surfaces in the stack.

---

End of report.
