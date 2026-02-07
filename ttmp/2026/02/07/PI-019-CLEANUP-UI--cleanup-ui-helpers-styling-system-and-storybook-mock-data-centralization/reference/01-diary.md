---
Title: Diary
Ticket: PI-019-CLEANUP-UI
Status: active
Topics:
    - frontend
    - architecture
    - middleware
    - websocket
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/styles/webchat.css
      Note: |-
        Reference architecture used for reusable styling strategy
        Reference style architecture evidence
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/README.md
      Note: P2.22 style contract documentation
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/package.json
      Note: Added test:unit script for helper tests
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components
      Note: |-
        Main source of helper/style duplication analyzed
        Duplication audit evidence
        Component migration targets for P1.7-P1.16
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/AppShell.tsx
      Note: P2.7 inline style removal
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/components/TurnInspector.tsx
      Note: P2.13 verified style-clean (no inline <style> block)
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/data.ts
      Note: |-
        Storybook mock centralization baseline analysis
        Mock centralization analysis evidence
        P3.1 compatibility export layer after fixture split
        P3.5 explicit legacy compatibility shim contract
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/common.ts
      Note: P3.2 deterministic fixture selector and clone helpers
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/conversationFactory.ts
      Note: P3.2 conversation/session/detail deterministic builders
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/deterministic.test.ts
      Note: P3.4 deterministic helper unit tests
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/deterministic.ts
      Note: P3.4 deterministic id/time/seq utility helpers
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/index.ts
      Note: P3.2 barrel export for all factory builders
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/anomalies.ts
      Note: P3.1 anomalies fixture domain file
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/turns.ts
      Note: P3.1 turn and turn-detail fixture domain extraction
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/msw/createDebugHandlers.ts
      Note: P3.6 reusable MSW debug handler builder
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/anomalyScenarios.ts
      Note: P3.3 anomaly panel scenario variants
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/eventInspectorScenarios.ts
      Note: P3.3 event inspector scenario variants
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/index.ts
      Note: P3.3 scenario barrel exports
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/overviewScenarios.ts
      Note: P3.3 overview story context scenarios
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/timelineScenarios.ts
      Note: P3.3 timeline lane story context scenarios
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components
      Note: Component CSS target directory for Phase 2B extractions
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/AnomalyPanel.css
      Note: P2.17 extracted anomaly panel styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/AppShell.css
      Note: P2.7 extracted AppShell styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/EventInspector.css
      Note: P2.15 extracted styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/EventTrackLane.css
      Note: P2.10 extracted styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/EventsPage.css
      Note: P2.20 extracted route styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/FilterBar.css
      Note: P2.16 extracted FilterBar + FilterChip styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/NowMarker.css
      Note: P2.12 extracted styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/OverviewPage.css
      Note: P2.18 extracted route styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/ProjectionLane.css
      Note: P2.11 extracted styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/SnapshotDiff.css
      Note: P2.14 extracted styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/StateTrackLane.css
      Note: P2.9 extracted styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/TimelineLanes.css
      Note: P2.8 extracted styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/TimelinePage.css
      Note: P2.19 extracted route styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/components/TurnDetailPage.css
      Note: P2.21 extracted route styles
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/layout.css
      Note: Layout layer created in P2.4
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/primitives.css
      Note: Primitive components layer created in P2.3
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/reset.css
      Note: Global reset layer created in P2.2
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/styles/tokens.css
      Note: Token layer created in P2.1
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/ui/format/format.test.ts
      Note: Unit tests for truncation/stringify/time edge cases (P1.19)
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/ui/format/phase.ts
      Note: Phase formatter module created in P1.1
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/ui/format/text.ts
      Note: Text helper module created in P1.3
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/ui/format/time.ts
      Note: Time formatter module created in P1.2
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/ui/presentation/events.ts
      Note: Event presentation module and mappings
    - Path: ../../../../../../../web-agent-example/cmd/web-agent-debug/web/src/ui/presentation/presentation.test.ts
      Note: Unit tests for presentation mapping fallbacks (P1.18)
    - Path: ttmp/2026/02/07/PI-019-CLEANUP-UI--cleanup-ui-helpers-styling-system-and-storybook-mock-data-centralization/analysis/01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup.md
      Note: |-
        Primary design/implementation plan produced in this ticket
        Primary plan artifact created during diary process
ExternalSources: []
Summary: Step-by-step diary for creating PI-019, analyzing helper/style/mock duplication, drafting the detailed implementation plan, and uploading design documentation to reMarkable.
LastUpdated: 2026-02-07T15:46:00-05:00
WhatFor: Preserve execution trail and rationale for PI-019 planning work.
WhenToUse: Use when reviewing how the plan was assembled, validated, and uploaded.
---


















# Diary

## Goal

Create PI-019 cleanup ticket artifacts and produce a detailed implementation plan for three cleanup streams (helpers, CSS, Storybook mock data), with a full work diary and reMarkable upload trail.

## Step 1: Create ticket workspace and planning docs

I started by creating the new docmgr ticket and immediately scaffolding the two required documents: the implementation plan and diary. This gave me stable paths for all planning and upload work.

This step set the execution frame: ticket first, then analysis, then diary, then upload and verification.

### Prompt Context

**User prompt (verbatim):** "let's start with reducing the size of the code.

Create a new ticket PI-019-CLEANUP-UI to

- unify remove duplicate helpers
- unify css style duplications into a reusable design system (see how the webchat in pinocchio/ is structured for reuse and styling)
- also analyze how th emock data is generated in storybook and if it can be centralized too

After creating the document, write a detailed implementation plan on how to do the css style dedpulication and the moc data and the helper cleanup, in detail.

Keep a diary as you work. Upload the design implementation documentation to remarkable."

**Assistant interpretation:** Create a new ticket and produce a detailed implementation design for reducing frontend code size by consolidating helpers, CSS, and Storybook mock structure, while documenting each step and uploading the plan to reMarkable.

**Inferred user intent:** Turn review findings into an actionable cleanup ticket with concrete architecture and execution steps, not just recommendations.

**Commit (code):** N/A (documentation/planning work)

### What I did

- Ran:
  - `docmgr ticket create-ticket --ticket PI-019-CLEANUP-UI ...`
  - `docmgr doc add --ticket PI-019-CLEANUP-UI --doc-type analysis --title "Implementation Plan for UI Helper, CSS System, and Storybook Mock Data Cleanup"`
  - `docmgr doc add --ticket PI-019-CLEANUP-UI --doc-type reference --title "Diary"`

### Why

- Ensure all planning output and upload references are tied to a dedicated ticket namespace.

### What worked

- Ticket workspace and both docs were created successfully.

### What didn't work

- N/A

### What I learned

- PI-019 uses the same docmgr root (`geppetto/ttmp`) as other PI tickets, so no workspace switching was required.

### What was tricky to build

- Choosing topics likely to pass existing vocabulary while remaining accurate.

### What warrants a second pair of eyes

- Confirm topic tags for this ticket should remain `frontend/react/redux/websocket`.

### What should be done in the future

- Add explicit topic guidance for “cleanup/refactor” style tickets.

### Code review instructions

- Verify ticket paths under:
  - `geppetto/ttmp/2026/02/07/PI-019-CLEANUP-UI--...`

### Technical details

- Created documents:
  - `analysis/01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup.md`
  - `reference/01-diary.md`

---

## Step 2: Analyze duplication hotspots in debug UI frontend

I performed an inventory pass focused on helper duplication, inline style duplication, and Storybook mock duplication patterns in `web-agent-example/cmd/web-agent-debug/web`.

This gave the quantitative and file-level evidence needed for a detailed plan rather than a generic cleanup checklist.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** produce concrete cleanup plan rooted in actual code duplication hotspots.

**Inferred user intent:** prioritize real effort reduction opportunities that materially shrink component complexity.

**Commit (code):** N/A

### What I did

- Ran targeted searches for duplicated helper functions:
  - `rg -n "function getEventTypeInfo|function getKindInfo|function getKindIcon|function truncateText|function formatPhase" ...`
- Ran targeted search for inline style blocks:
  - `rg -n '<style>\{`' src/components src/routes src/App.tsx`
- Audited Storybook story data usage and repeated mock patterns:
  - `rg -n "from '../mocks/data'|mock[A-Za-z]+|msw|http.get\('/debug" ...`

### Why

- Needed exact cleanup scope and migration order.

### What worked

- Clear duplicate helper hotspots emerged.
- Inline style duplication was easy to enumerate.
- Story-level mock scattering patterns were obvious from grep inventory.

### What didn't work

- N/A

### What I learned

- The app already has partial centralization (`src/mocks/data.ts`), but stories still duplicate scenario logic and handler setup frequently.

### What was tricky to build

- Distinguishing “acceptable variation” from true duplication in stories.

### What warrants a second pair of eyes

- Confirm whether any story-local specialized mocks should remain local by design.

### What should be done in the future

- Introduce fixture/factory/scenario conventions before adding more stories.

### Code review instructions

- Re-run the helper/style/mock grep commands and compare with plan section 2.

### Technical details

- Key duplication categories captured in analysis doc sections 2.1–2.3.

---

## Step 3: Study pinocchio webchat structure for reusable styling and story setup

Per request, I analyzed the pinocchio webchat implementation to extract reusable design patterns for PI-019, especially tokenized styles and component styling contracts.

I focused on style architecture and story scenario generation patterns rather than product behavior.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** use pinocchio webchat as structural reference for reusable styling and mock/scenario patterns.

**Inferred user intent:** avoid inventing a cleanup architecture from scratch when an in-repo reference already exists.

**Commit (code):** N/A

### What I did

- Read key files:
  - `pinocchio/.../webchat/styles/theme-default.css`
  - `pinocchio/.../webchat/styles/webchat.css`
  - `pinocchio/.../webchat/parts.ts`
  - `pinocchio/.../webchat/ChatWidget.tsx`
  - `pinocchio/.../webchat/ChatWidget.stories.tsx`
- Extracted reusable ideas:
  - token layer + structural CSS layer split,
  - data-part styling contract,
  - shared class/style merge helpers,
  - scenario-runner story pattern.

### Why

- Required by user and useful for reducing design risk.

### What worked

- Pinocchio provided clear, copyable architecture patterns for this cleanup domain.

### What didn't work

- N/A

### What I learned

- The `theme-default.css` + `webchat.css` split is a strong model for PI-019 CSS dedup.

### What was tricky to build

- Translating webchat’s part-based approach to debug UI without over-engineering.

### What warrants a second pair of eyes

- Frontend maintainers should confirm whether they prefer data-part selectors, BEM classes, or a hybrid.

### What should be done in the future

- Standardize one styling contract and document it in frontend README.

### Code review instructions

- Compare PI-019 plan section 3 against the pinocchio files listed above.

### Technical details

- Reference patterns are called out explicitly in analysis sections 3 and 4.

---

## Step 4: Write detailed phased implementation plan

I authored the full implementation plan with current-state evidence, target architecture, phased execution details, acceptance criteria, risks, and PR slicing.

The plan is intentionally concrete and implementation-ready, not conceptual.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** produce a detailed execution blueprint for helper/CSS/mock cleanup.

**Inferred user intent:** enable immediate task execution with minimal ambiguity.

**Commit (code):** N/A

### What I did

- Wrote `analysis/01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup.md` including:
  - current-state duplication evidence,
  - pinocchio reference architecture mapping,
  - target file structures,
  - phase-by-phase implementation tasks,
  - validation and DoD.
- Updated ticket `index.md` and `tasks.md` to reflect created plan and execution phases.

### Why

- User requested a detailed plan “in detail” and this ticket is planning-first.

### What worked

- The final plan includes actionable sequencing and explicit acceptance criteria.

### What didn't work

- N/A

### What I learned

- Starting with helper unification (Phase 1) is the safest first move and reduces noise for CSS/mock refactors.

### What was tricky to build

- Balancing detail with phased delivery so the plan is practical for PR slicing.

### What warrants a second pair of eyes

- Validate phase estimates and PR boundaries with frontend owners.

### What should be done in the future

- Convert phase tasks into assignee-level execution tasks once implementation starts.

### Code review instructions

- Review sections 4, 5, 6, and 10 of analysis doc first.

### Technical details

- Plan path:
  - `analysis/01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup.md`

---

## Step 5: Docmgr linking, changelog update, and frontmatter validation

After writing the plan, I related the key source files to the analysis doc, updated ticket changelog entries, and validated frontmatter for ticket docs.

This keeps the ticket discoverable and auditable.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** store the documentation cleanly in ticket with proper bookkeeping.

**Inferred user intent:** make the plan easy to find and trustworthy for future execution.

**Commit (code):** N/A

### What I did

- Ran `docmgr doc relate` on analysis doc with 8 key code references.
- Updated ticket changelog entries with planning completion + upload trail.
- Validated frontmatter for index/analysis/diary docs.

### Why

- Strong file-to-doc traceability is required for real reuse.

### What worked

- Relation updates and frontmatter validation succeeded.

### What didn't work

- N/A

### What I learned

- Pinning reference files in frontmatter materially improves future re-entry speed.

### What was tricky to build

- Choosing a concise but complete RelatedFiles set.

### What warrants a second pair of eyes

- Confirm related file set is not over-broad.

### What should be done in the future

- Add new related files incrementally when implementation PRs begin.

### Code review instructions

- Check analysis doc frontmatter `RelatedFiles` section.

### Technical details

- Related files include both debug UI sources and pinocchio reference architecture files.

---

## Step 6: Upload implementation plan to reMarkable and verify

I executed the reMarkable upload workflow for the design/implementation document as requested.

I used dry-run first, then real upload, then cloud listing verification.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** upload the design implementation documentation to reMarkable after ticket storage.

**Inferred user intent:** make the plan accessible on reMarkable for review.

**Commit (code):** N/A

### What I did

- Ran:
  - `remarquee status`
  - `remarquee upload md --dry-run <analysis/01> --remote-dir "/ai/2026/02/07/PI-019-CLEANUP-UI"`
  - `remarquee upload md <analysis/01> --remote-dir "/ai/2026/02/07/PI-019-CLEANUP-UI"`
  - `remarquee cloud ls /ai/2026/02/07/PI-019-CLEANUP-UI --long --non-interactive`

### Why

- Required deliverable from user.

### What worked

- Upload succeeded:
  - `OK: uploaded 01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup.pdf -> /ai/2026/02/07/PI-019-CLEANUP-UI`
- Cloud listing verification succeeded.

### What didn't work

- N/A

### What I learned

- Single-doc markdown upload is sufficient here since only the implementation plan was requested for upload.

### What was tricky to build

- Ensuring remote folder naming remains consistent with ticket naming conventions.

### What warrants a second pair of eyes

- Confirm whether future revisions should overwrite this file or use explicit versioned names.

### What should be done in the future

- If plan updates significantly, upload a `(... Final)` or `(... v2)` variant and note it in changelog.

### Code review instructions

- Verify cloud folder contains:
  - `01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup`

### Technical details

- Remote path:
  - `/ai/2026/02/07/PI-019-CLEANUP-UI`

## Step 7: Expand ticket tasks into detailed execution checklist

After the planning docs were complete, you requested a more detailed task breakdown directly in the ticket. I expanded `tasks.md` from a phase summary into a granular execution checklist with phase-prefixed task IDs, file-level migration targets, and explicit validation criteria.

This step turns PI-019 from “plan exists” into “implementation-ready checklist” for immediate execution.

### Prompt Context

**User prompt (verbatim):** "add detailed tasks to the ticket."

**Assistant interpretation:** Replace high-level task bullets with a concrete, implementation-level checklist that can be executed and tracked step-by-step.

**Inferred user intent:** Increase operational clarity so the cleanup can start without additional planning cycles.

**Commit (code):** N/A

### What I did

- Rewrote `tasks.md` with detailed tasks across:
  - Phase 0 baseline/guardrails
  - Phase 1 helper unification (module creation + per-component migrations + tests)
  - Phase 2 CSS design-system extraction (style layers + per-file extraction list)
  - Phase 3 Storybook mock centralization (fixtures/factories/scenarios/MSW builders)
  - Phase 4 final cleanup/docs/enforcement
  - Validation/exit criteria
- Added task IDs (`P1.7`, `P2.14`, etc.) for easier status referencing in follow-up diary/changelog entries.

### Why

- High-level tasks were not sufficiently actionable for implementation tracking.

### What worked

- Task list now maps directly to file-level work and review checkpoints.

### What didn't work

- N/A

### What I learned

- Converting phase summaries into granular checklists early reduces execution ambiguity significantly.

### What was tricky to build

- Balancing granularity with readability (too many tiny tasks vs too broad tasks).

### What warrants a second pair of eyes

- Confirm if maintainers want to split any very large Phase 2/3 tasks further before coding starts.

### What should be done in the future

- As implementation starts, mark tasks incrementally and add changelog entries keyed to task IDs.

### Code review instructions

- Review updated:
  - `tasks.md`
- Confirm each major concern (helpers, CSS, mocks, validation) has explicit executable subtasks.

### Technical details

- `tasks.md` now contains detailed task IDs and per-phase acceptance checks.

## Step 8: Execute Phase 0 baseline tasks and lock tracking metrics

I started implementation by completing Phase 0 first so the ticket has measurable before/after numbers before refactors begin. I collected LOC and duplication baselines, ran production and Storybook builds, then added the temporary tracking section into the implementation plan.

I also checked off `P0.1` through `P0.5` directly in docmgr so this work is reflected in ticket state before moving into helper module code changes.

### Prompt Context

**User prompt (verbatim):** "Work on docmgr ticket PI-019-CLEANUP-UI and keep a frequent diary as you work.

Check tsks off one by one and commit to git appropriately."

**Assistant interpretation:** Execute PI-019 implementation tasks in sequence, maintain a running diary while working, and commit progress with clean task-aligned commits.

**Inferred user intent:** Drive the ticket from planning into execution with traceable progress, not just design docs.

**Commit (code):** N/A (ticket execution and documentation baseline step)

### What I did

- Collected baseline LOC metrics for `web-agent-debug/web`:
  - total TSX LOC/files, total CSS LOC/files, and top large TSX components by LOC.
- Collected baseline duplication metrics:
  - counts for duplicated helper signatures and runtime inline `<style>{` block count.
- Ran baseline builds:
  - `npm run build`
  - `npm run build-storybook`
- Updated:
  - `analysis/01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup.md` with section `12) Temporary Tracking: Baseline vs After`.
- Checked off tasks one by one:
  - `docmgr task check --ticket PI-019-CLEANUP-UI --id 8`
  - `docmgr task check --ticket PI-019-CLEANUP-UI --id 9`
  - `docmgr task check --ticket PI-019-CLEANUP-UI --id 10`
  - `docmgr task check --ticket PI-019-CLEANUP-UI --id 11`
  - `docmgr task check --ticket PI-019-CLEANUP-UI --id 12`

### Why

- Phase 0 provides the measurement baseline needed to prove cleanup impact in later commits.

### What worked

- Both builds passed, and the metrics are now documented in the implementation plan.
- Task status in `tasks.md` now reflects Phase 0 completion.

### What didn't work

- `npm run build-storybook` emitted existing Vite warnings about `eval` usage from Storybook runtime bundles and chunk-size warnings; builds still completed successfully.

### What I learned

- The runtime app footprint is concentrated in a few large components (`SnapshotDiff`, `EventInspector`, `AnomalyPanel`, `FilterBar`), which confirms the priority order for cleanup.

### What was tricky to build

- Mapping “confirm branch strategy with maintainers” to an execution-ready action in an autonomous run required explicitly documenting the commit/PR slicing strategy in the plan section so it is reviewable and not implicit.

### What warrants a second pair of eyes

- Verify whether maintainers want to adjust PR slice boundaries before Phase 2 CSS extraction starts.

### What should be done in the future

- As each phase lands, update the “After” column in section 12 with the same commands for consistent trend tracking.

### Code review instructions

- Review metric and strategy updates in:
  - `analysis/01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup.md`
- Confirm task checks in:
  - `tasks.md`

### Technical details

- Key baseline values captured:
  - TSX LOC: `6629`
  - CSS LOC: `349`
  - Inline `<style>{` blocks: `31`

## Step 9: Complete Phase 1 helper unification (modules + component migrations)

I implemented the shared helper layer under `src/ui` and migrated all Phase 1B target components to consume those helpers. This removed local duplicate helper implementations across event, block, timeline, phase, time, and text formatting concerns.

After migration, I validated with `npm run build`, `npm run build-storybook`, and helper-signature grep checks, then checked off the completed tasks one by one in docmgr.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Execute implementation tasks in order, keep diary updates frequent, and commit work in focused slices.

**Inferred user intent:** Move PI-019 from planning into concrete code reduction with measurable task progress.

**Commit (code):** `56751d0f6f0a261344597ff35b1ec8cb43434303` — "refactor(debug-ui): unify phase/time/text and presentation helpers"

### What I did

- Added shared helper modules:
  - `src/ui/format/phase.ts`
  - `src/ui/format/time.ts`
  - `src/ui/format/text.ts`
  - `src/ui/presentation/events.ts`
  - `src/ui/presentation/blocks.ts`
  - `src/ui/presentation/timeline.ts`
- Migrated components:
  - `EventCard.tsx`, `EventTrackLane.tsx`, `EventInspector.tsx`
  - `BlockCard.tsx`, `StateTrackLane.tsx`, `FilterBar.tsx`
  - `ProjectionLane.tsx`, `TimelineEntityCard.tsx`
  - `SnapshotDiff.tsx`, `TurnInspector.tsx`
- Removed now-duplicated local helper functions from migrated files.
- Ran validation:
  - `npm run build`
  - `npm run build-storybook`
  - `rg -n "function (getEventTypeInfo|getKindInfo|getKindIcon|truncateText|formatPhase|formatPhaseName)" ...` (all zero in components/routes)
- Checked off tasks one-by-one:
  - `P1.1` through `P1.17`
  - `P1.20`

### Why

- Consolidating helper logic first reduces repeated UI logic and lowers risk for the next CSS and Storybook refactor phases.

### What worked

- Shared helper modules compiled cleanly and were adopted by all intended components.
- Build and Storybook build both passed after migration.
- Duplicate helper function signatures for the targeted set are now zero in runtime components/routes.

### What didn't work

- First build attempt failed due a missing closing brace in `TimelineEntityCard.tsx`:
  - `src/components/TimelineEntityCard.tsx(112,1): error TS1005: '}' expected.`
- Fix: restored the missing `}` after the component return block and re-ran validation successfully.

### What I learned

- A single shared event presentation helper needed small options (e.g., thinking icon behavior) to preserve existing per-component semantics without duplicating maps.

### What was tricky to build

- Preserving existing visual behavior while unifying helpers required careful mapping because previous components had subtle differences in fallback icon/color logic. I handled this by using a canonical mapping plus limited option overrides instead of introducing parallel helper variants.

### What warrants a second pair of eyes

- Review event presentation fallback behavior (`events.ts`) to confirm all desired icon/color defaults are unchanged for edge event types.

### What should be done in the future

- Complete `P1.18` and `P1.19` with dedicated unit tests for mapping fallbacks and truncation/stringify edge cases before closing Phase 1 entirely.

### Code review instructions

- Start with helper modules in:
  - `web-agent-example/cmd/web-agent-debug/web/src/ui/format/`
  - `web-agent-example/cmd/web-agent-debug/web/src/ui/presentation/`
- Then review migrations in the ten components listed above.
- Re-run:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- Post-migration targeted helper-signature counts in components/routes:
  - `getEventTypeInfo`: `0`
  - `getKindInfo`: `0`
  - `getKindIcon`: `0`
  - `truncateText`: `0`
  - `formatPhase`: `0`
  - `formatPhaseName`: `0`

## Step 10: Add helper unit tests for fallback and formatting edge cases

I completed the remaining Phase 1 test tasks by introducing a lightweight unit-test runner and adding focused tests for shared helper behavior. The tests cover mapping fallbacks for event/block/timeline presentation and edge cases for text truncation, stringify safety, and date formatting.

After adding tests, I re-ran the production build to ensure test and dependency updates did not affect the normal build path.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue executing ticket tasks sequentially and keep the diary/changelog synchronized with each commit.

**Inferred user intent:** Ensure implementation quality by validating helper behavior while keeping task progress auditable.

**Commit (code):** `aaef9d176f90bcae1eef5cbf9bc9dc16949965ff` — "test(debug-ui): add unit tests for shared helper behavior"

### What I did

- Added `vitest` as a dev dependency and wired:
  - `package.json` script: `test:unit`
- Added tests:
  - `src/ui/presentation/presentation.test.ts`
  - `src/ui/format/format.test.ts`
- Verified:
  - `npm run test:unit` (9 tests passed)
  - `npm run build` (passes)
- Checked off tasks:
  - `P1.18` and `P1.19`

### Why

- Phase 1 required explicit unit coverage for fallback mappings and format/truncation safety to prevent regressions while helpers are centralized.

### What worked

- Tests execute quickly and validate the key behavior contracts for helper modules.
- Build remains green after dependency and test additions.

### What didn't work

- N/A

### What I learned

- Keeping helper tests at the module boundary provides high confidence with low maintenance cost, especially before larger CSS and mock-data refactors.

### What was tricky to build

- Introducing tests into a project without an existing unit-test script required selecting a minimal toolchain change (Vitest only) to avoid broad project churn.

### What warrants a second pair of eyes

- Confirm the lockfile delta from adding Vitest is acceptable for this repo’s dependency policy.

### What should be done in the future

- Add follow-up tests for any new helper branches introduced during Phase 2/3 refactors.

### Code review instructions

- Review test coverage in:
  - `web-agent-example/cmd/web-agent-debug/web/src/ui/presentation/presentation.test.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/ui/format/format.test.ts`
- Validate with:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run test:unit`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`

### Technical details

- Test run summary:
  - `Test Files: 2 passed`
  - `Tests: 9 passed`

## Step 11: Scaffold Phase 2A style architecture and import layers

I moved the base stylesheet structure to the planned layered architecture by splitting `index.css` into token/reset/primitives/layout files and creating the `styles/components` directory with per-component CSS files. The runtime styling is unchanged at this step because inline `<style>{...}` blocks are intentionally deferred to Phase 2B extraction tasks.

This establishes the destination CSS structure so inline-style migrations can be done incrementally without repeatedly reworking global imports.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue task-by-task execution with commits and diary updates, including CSS cleanup groundwork.

**Inferred user intent:** Keep momentum through the planned phases while preserving traceability of each implementation slice.

**Commit (code):** `6efec1b20191c0ed3133fc601cd9e2d2c6c87779` — "refactor(debug-ui): scaffold layered style architecture"

### What I did

- Created style layers:
  - `src/styles/tokens.css`
  - `src/styles/reset.css`
  - `src/styles/primitives.css`
  - `src/styles/layout.css`
- Created `src/styles/components/` with per-component/per-route CSS files for upcoming extraction tasks.
- Moved correlation bar CSS into `src/styles/components/CorrelationIdBar.css`.
- Converted `src/index.css` to a pure import orchestrator for all style layers/files.
- Validated with:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P2.1` to `P2.6`

### Why

- Phase 2B extraction is simpler and safer when file structure and import ordering are stable first.

### What worked

- Build and Storybook both passed after the stylesheet split.
- No immediate UI logic changes were required for this architecture step.

### What didn't work

- N/A

### What I learned

- Setting import orchestration first prevents repetitive merge churn when extracting dozens of inline style blocks.

### What was tricky to build

- Preserving current global class behavior while redistributing CSS required careful grouping to avoid accidental specificity/order changes; I kept broad utility and primitive class definitions unchanged and only moved file boundaries.

### What warrants a second pair of eyes

- Confirm whether placeholder component CSS files should remain minimal until each extraction task lands, or if maintainers prefer pre-populated section stubs.

### What should be done in the future

- Execute Phase 2B tasks (`P2.7+`) by moving inline block styles from each TSX file into its corresponding component CSS file.

### Code review instructions

- Review style-layer split in:
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/`
  - `web-agent-example/cmd/web-agent-debug/web/src/index.css`
- Validate with:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- CSS topology moved from:
  - `1` file (`index.css`)
- to:
  - `21` files (layered + component files)
- Runtime inline `<style>{` block count remains:
  - `31` (to be reduced in Phase 2B)

## Step 12: Complete P2.7 by extracting AppShell inline styles

I migrated the `AppShell.tsx` runtime inline style block into the new component stylesheet location (`styles/components/AppShell.css`) and removed the in-component `<style>{...}` block. This is the first concrete Phase 2B extraction after the scaffold setup.

The extraction reduced runtime inline style blocks by one and validated cleanly with both production and Storybook builds.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue executing and checking off tasks incrementally with dedicated commits and diary updates.

**Inferred user intent:** Maintain steady, auditable progress through the CSS dedup migration instead of batching risky large edits.

**Commit (code):** `41d4e9ce80736b15403d6cc87416fb61cd4f0266` — "refactor(debug-ui): extract AppShell inline styles to css"

### What I did

- Moved AppShell styles:
  - from `src/components/AppShell.tsx` inline `<style>{...}`
  - to `src/styles/components/AppShell.css`
- Removed inline style JSX block from `AppShell.tsx`.
- Validated:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P2.7`

### Why

- Phase 2B requires one-file-at-a-time extraction to keep style regressions easy to isolate and review.

### What worked

- Extraction preserved build and Storybook behavior.
- Inline `<style>{` block count dropped from `31` to `30`.

### What didn't work

- N/A

### What I learned

- The new style-layer scaffold enabled this extraction without further `index.css` churn, confirming the Phase 2A sequencing was correct.

### What was tricky to build

- `AppShell` has classes that overlap with global utility names (e.g., `.main-content`), so extraction needed to preserve class names exactly to avoid behavioral drift.

### What warrants a second pair of eyes

- Confirm there is no unintended CSS specificity conflict between `layout.css` and `components/AppShell.css` for `.main-content`.

### What should be done in the future

- Continue Phase 2B extraction sequence with `TimelineLanes.tsx` (`P2.8`) next.

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/components/AppShell.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/AppShell.css`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- Runtime inline style count change:
  - Before: `31`
  - After: `30`

## Step 13: Complete P2.8-P2.12 by extracting lane and marker inline styles

I extracted inline styles for the next five Phase 2B targets in one cohesive batch: timeline lanes, state lane, event lane, projection lane, and now marker. This removed all inline `<style>{...}` blocks from those components and moved styles into the pre-created component CSS files.

The batch reduced runtime inline style blocks from `30` to `22` while keeping both production and Storybook builds passing.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue executing ticket tasks in order, with regular diary entries and focused commits.

**Inferred user intent:** Keep PI-019 moving steadily through Phase 2B style extraction without losing traceability.

**Commit (code):** `9d7ba6cbc0f6a03c74d121467758e45a0ad0b388` — "refactor(debug-ui): extract lane and marker inline styles"

### What I did

- Extracted inline styles from:
  - `TimelineLanes.tsx` (`P2.8`)
  - `StateTrackLane.tsx` (`P2.9`)
  - `EventTrackLane.tsx` (`P2.10`)
  - `ProjectionLane.tsx` (`P2.11`)
  - `NowMarker.tsx` (`P2.12`)
- Moved them into:
  - `src/styles/components/TimelineLanes.css`
  - `src/styles/components/StateTrackLane.css`
  - `src/styles/components/EventTrackLane.css`
  - `src/styles/components/ProjectionLane.css`
  - `src/styles/components/NowMarker.css`
- Validated:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P2.8` through `P2.12`

### Why

- These five files are tightly related lane/marker UI pieces, so batching them made extraction coherent while still keeping commit size reviewable.

### What worked

- All targeted files now have no inline style blocks.
- Build and Storybook build remained green.
- Runtime inline style count dropped by `8` in this step.

### What didn't work

- N/A

### What I learned

- Doing extraction in lane-oriented clusters reduces context switching and helps avoid missing shared class dependencies.

### What was tricky to build

- `StateTrackLane` had `.block-kind-*` rules that overlap global selectors; to preserve 2px lane-specific chips while global rules remain 3px elsewhere, I moved these into more specific selectors (`.turn-card-blocks .block-kind-*`).

### What warrants a second pair of eyes

- Confirm there are no unintended side-effects from shared class names like `.empty-lane` now being defined in multiple component CSS files.

### What should be done in the future

- Continue with remaining Phase 2B extraction tasks: `P2.13` through `P2.21`.

### Code review instructions

- Review extracted components:
  - `web-agent-example/cmd/web-agent-debug/web/src/components/TimelineLanes.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/StateTrackLane.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/EventTrackLane.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/ProjectionLane.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/NowMarker.tsx`
- Review target CSS:
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/TimelineLanes.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/StateTrackLane.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/EventTrackLane.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/ProjectionLane.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/NowMarker.css`

### Technical details

- Runtime inline style count:
  - Before: `30`
  - After: `22`

## Step 14: Complete P2.13 and route-level extractions (P2.18-P2.21)

I finished the route-level extraction tasks by moving inline style blocks from the four route files into their corresponding component CSS files. During this pass, I also validated that `TurnInspector.tsx` already had no inline `<style>{...}` block, so `P2.13` was marked complete as style-clean without additional extraction.

To avoid global selector collisions after moving route styles into globally imported CSS, I introduced route-scoped class names (`overview-*`, `timeline-*`, `events-*`, `turn-detail-*`) where needed.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue the task sequence with frequent diary updates and clean commits per logical change set.

**Inferred user intent:** Progress through Phase 2B extraction with safe, reviewable slices and reliable validation.

**Commit (code):** `ac2f9369a83bec4721be7a76edebb1330dd1f552` — "refactor(debug-ui): extract route-level inline styles"

### What I did

- Verified:
  - `TurnInspector.tsx` has no inline `<style>{...}` block (`P2.13` style-clean).
- Extracted route styles:
  - `OverviewPage.tsx` -> `OverviewPage.css` (`P2.18`)
  - `TimelinePage.tsx` -> `TimelinePage.css` (`P2.19`)
  - `EventsPage.tsx` -> `EventsPage.css` (`P2.20`)
  - `TurnDetailPage.tsx` -> `TurnDetailPage.css` (`P2.21`)
- Added route-specific class names to prevent cross-page CSS conflicts in global imports.
- Validated:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P2.13`, `P2.18`, `P2.19`, `P2.20`, `P2.21`

### Why

- Route-level extractions are low-risk wins that reduce inline style count quickly while keeping remaining complex component extractions isolated for next steps.

### What worked

- All targeted route files are now free of inline style blocks.
- Build and Storybook remained green.
- Runtime inline style blocks dropped from `22` to `18`.

### What didn't work

- N/A

### What I learned

- Once styles are globally imported, generic class names like `.empty-state` and `.page-header` can collide across routes; route-prefixed class naming avoids this cleanly.

### What was tricky to build

- The route extraction required a small naming migration (JSX + CSS) to preserve behavior while preventing future cascade conflicts after centralization.

### What warrants a second pair of eyes

- Confirm the new route-prefixed class names align with frontend naming conventions desired for the rest of Phase 2.

### What should be done in the future

- Continue with remaining heavy component extractions: `P2.14` to `P2.17`.

### Code review instructions

- Review routes:
  - `web-agent-example/cmd/web-agent-debug/web/src/routes/OverviewPage.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/routes/TimelinePage.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/routes/EventsPage.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/routes/TurnDetailPage.tsx`
- Review extracted CSS:
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/OverviewPage.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/TimelinePage.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/EventsPage.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/TurnDetailPage.css`

### Technical details

- Runtime inline style count:
  - Before: `22`
  - After: `18`

## Step 15: Complete P2.16 by extracting FilterBar inline styles

I extracted both inline style blocks from `FilterBar.tsx` into `styles/components/FilterBar.css` and removed the embedded `<style>{...}` usage in that component. This continues the Phase 2B pattern of moving runtime styles into the centralized component CSS layer.

After extraction, both build and Storybook build remained successful and runtime inline style count dropped from `18` to `16`.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue task execution with checkoffs, diary updates, and focused commits.

**Inferred user intent:** Maintain consistent progress through the CSS extraction backlog and keep the ticket state current.

**Commit (code):** `51ab056b8224cf33191f51907f4752c947b30095` — "refactor(debug-ui): extract FilterBar inline styles"

### What I did

- Removed inline style blocks from:
  - `web-agent-example/cmd/web-agent-debug/web/src/components/FilterBar.tsx`
- Added extracted styles to:
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/FilterBar.css`
- Validated:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P2.16`

### Why

- FilterBar contained two distinct inline style regions (container and chip styles), making it a high-impact single-task extraction candidate.

### What worked

- Extraction preserved behavior and all validations passed.
- Runtime inline style count decreased as expected.

### What didn't work

- N/A

### What I learned

- The style-layer scaffold handles component-level style growth cleanly; larger extracted CSS remains manageable in dedicated component files.

### What was tricky to build

- FilterBar combines parent and child (`FilterChip`) style blocks; extraction required combining both in one CSS module while preserving class semantics and active chip state rules.

### What warrants a second pair of eyes

- Confirm filter chip active-state color classes still match the intended block-kind color mapping after extraction.

### What should be done in the future

- Continue with remaining major component extractions:
  - `P2.14` (`SnapshotDiff.tsx`)
  - `P2.15` (`EventInspector.tsx`)
  - `P2.17` (`AnomalyPanel.tsx`)

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/components/FilterBar.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/FilterBar.css`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- Runtime inline style count:
  - Before: `18`
  - After: `16`

## Step 16: Complete P2.17 by extracting AnomalyPanel inline styles

I extracted all remaining inline style blocks from `AnomalyPanel.tsx` into `styles/components/AnomalyPanel.css` and removed the embedded `<style>{...}` sections from the component. This closed out the anomaly panel extraction task and further reduced runtime inline style usage.

Both production and Storybook builds succeeded after the change, and the global inline-style count dropped from `16` to `13`.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue with the next unchecked Phase 2 task and keep the ticket diary/changelog synchronized.

**Inferred user intent:** Keep delivering concrete, test-verified cleanup increments until the phase is complete.

**Commit (code):** `140afccd70bc02dc25f89843d97a4479786b2bab` — "refactor(debug-ui): extract AnomalyPanel inline styles"

### What I did

- Removed inline styles from:
  - `web-agent-example/cmd/web-agent-debug/web/src/components/AnomalyPanel.tsx`
- Added extracted styles to:
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/AnomalyPanel.css`
- Validated:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P2.17`

### Why

- `AnomalyPanel` was one of the larger remaining style-heavy components and represented a high-value extraction target.

### What worked

- Build and Storybook validation remained green.
- Runtime inline style count reduced as expected.

### What didn't work

- N/A

### What I learned

- Even complex multi-section components can be migrated cleanly when styles are grouped by feature sections (panel shell, cards, detail pane) in the target CSS file.

### What was tricky to build

- `AnomalyPanel` reused generic names like `.empty-state`; moving styles to global CSS required safer selector scoping (`.anomaly-list .empty-state`) to avoid cross-component conflicts.

### What warrants a second pair of eyes

- Verify that anomaly severity chips and detail view badges still visually match prior behavior in Storybook.

### What should be done in the future

- Finish remaining high-density extractions:
  - `P2.14` (`SnapshotDiff.tsx`)
  - `P2.15` (`EventInspector.tsx`)

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/components/AnomalyPanel.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/AnomalyPanel.css`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- Runtime inline style count:
  - Before: `16`
  - After: `13`

## Step 17: Complete SnapshotDiff/EventInspector extraction and reach zero runtime inline styles

I extracted all remaining inline style blocks from `SnapshotDiff.tsx` and `EventInspector.tsx` into their component CSS files. This was the final runtime style-block removal step, bringing the project to zero inline `<style>{...}` blocks in runtime TSX files.

Because these two components contain many generic class names, I scoped CSS selectors under `.snapshot-diff` and `.event-inspector` to avoid unintended global collisions.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Keep executing the remaining Phase 2 tasks with commits and diary/changelog updates.

**Inferred user intent:** Finish the CSS extraction phase with measurable completion and clean task-state traceability.

**Commit (code):** `4e93e8571fa98c10fad2a30f1a1180d29e8e5268` — "refactor(debug-ui): extract inspector and diff inline styles"

### What I did

- Removed all inline `<style>{...}` blocks from:
  - `web-agent-example/cmd/web-agent-debug/web/src/components/SnapshotDiff.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/EventInspector.tsx`
- Added extracted CSS to:
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/SnapshotDiff.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/EventInspector.css`
- Used root-scoped selectors (`.snapshot-diff ...`, `.event-inspector ...`) in CSS to reduce selector leakage.
- Validated:
  - `npm run build`
  - `npm run build-storybook`
  - `rg -n \"<style>\\{\" src/components src/routes src/App.tsx` (result count `0`)
- Checked off:
  - `P2.14`, `P2.15`, `P2.25`

### Why

- These two components were the final blockers for zero runtime inline style blocks.

### What worked

- Both builds passed after extraction.
- Runtime inline style count reached zero.

### What didn't work

- N/A

### What I learned

- For large style-heavy components, removing inline blocks first and then centralizing styles with root-scoped selectors is a reliable migration pattern.

### What was tricky to build

- `EventInspector` and `SnapshotDiff` had repeated generic selectors (`.json-view`, `.meta-*`, `.status-*`) that could conflict globally after extraction. I mitigated this by explicit root scoping in CSS.

### What warrants a second pair of eyes

- Verify visual parity for `SnapshotDiff` and `EventInspector` stories after selector scoping.

### What should be done in the future

- Complete the remaining Phase 2 standardization tasks (`P2.22`–`P2.24`) and then proceed to Phase 3 mock-data centralization.

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/components/SnapshotDiff.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/EventInspector.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/SnapshotDiff.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/EventInspector.css`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`
  - `cd web-agent-example/cmd/web-agent-debug/web && rg -n \"<style>\\{\" src/components src/routes src/App.tsx`

### Technical details

- Runtime inline style count:
  - Before: `13`
  - After: `0`

## Step 18: Complete Phase 2C style contract, class naming normalization, and token replacement

I completed the remaining Phase 2 styling standardization tasks by locking in a concrete style contract, normalizing major component class/part naming, and removing repeated hard-coded `rgba(...)` values from runtime style files. This closes out `P2.22`, `P2.23`, and `P2.24` and fully completes Phase 2.

The change was delivered as a focused frontend commit, then validated with production and Storybook builds plus a grep-based tokenization check to confirm no remaining `rgba(...)` usage outside `tokens.css`.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue PI-019 with one-by-one task completion, keep diary updates frequent, and commit changes in task-aligned slices.

**Inferred user intent:** Finish the remaining Phase 2 standardization tasks with auditable docs and validated code commits.

**Commit (code):** `2181eac248906e2313409940bafca7701a298731` — "refactor(debug-ui): standardize style contract naming and color tokens"

### What I did

- Added and documented the style contract in:
  - `web-agent-example/cmd/web-agent-debug/web/README.md`
- Normalized naming in major component TSX/CSS pairs:
  - `web-agent-example/cmd/web-agent-debug/web/src/components/AppShell.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/AppShell.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/TimelineLanes.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/TimelineLanes.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/AnomalyPanel.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/AnomalyPanel.css`
- Added reusable alpha/shadow tokens in:
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/tokens.css`
- Replaced repeated `rgba(...)` values with token variables in:
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/primitives.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/SnapshotDiff.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/ProjectionLane.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/StateTrackLane.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/EventTrackLane.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/EventInspector.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/TimelineLanes.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/components/AnomalyPanel.css`
- Validated:
  - `npm run build`
  - `npm run build-storybook`
  - `rg -n "rgba\\(" src/styles -g '!src/styles/tokens.css'`
- Checked off:
  - `P2.22`
  - `P2.23`
  - `P2.24`

### Why

- Phase 2 had already removed runtime inline style blocks, but still needed explicit styling contract rules and tokenized alpha color usage to reduce selector collisions and future style drift.

### What worked

- Both build pipelines remained green.
- Class naming updates improved selector clarity for major shell/lane/detail components.
- Repeated color alpha values were centralized under token variables.

### What didn't work

- Initial bulk replacement command failed because a multiline file list was passed as a single argument:
  - Error: `Can't open src/styles/tokens.css ... src/styles/components/StateTrackLane.css: No such file or directory.`
- Fix:
  - Re-ran replacement using `rg -l ... | while IFS= read -r f; do ...; done` to process each file path safely.

### What I learned

- For wide style-token migrations, file iteration via `while read` is safer than shell-expanded multiline variables, especially when applying scripted replacements.

### What was tricky to build

- This pass combined two concerns (naming normalization and token replacement) across overlapping files. I had to avoid introducing regressions while renaming classes and swapping color values in the same components. I handled this by validating all touched files via full production and Storybook builds plus targeted grep checks for leftover `rgba(...)`.

### What warrants a second pair of eyes

- Story-level visual parity for:
  - `AppShell`
  - `TimelineLanes`
  - `AnomalyPanel`
- Focus areas:
  - nav/link active states, lane header/count styling, anomaly detail panel spacing.

### What should be done in the future

- Start Phase 3 (`P3.1` onward): Storybook mock-data centralization (fixtures/factories/scenarios + MSW handler builders).

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/README.md`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/AppShell.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/TimelineLanes.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/components/AnomalyPanel.tsx`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/tokens.css`
  - `web-agent-example/cmd/web-agent-debug/web/src/styles/primitives.css`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`
  - `cd web-agent-example/cmd/web-agent-debug/web && rg -n "rgba\\(" src/styles -g '!src/styles/tokens.css'`

### Technical details

- Hard-coded `rgba(...)` usage in `src/styles` excluding `tokens.css`:
  - Before: `43`
  - After: `0`
- Runtime inline style count:
  - Before: `0`
  - After: `0`

## Step 19: Complete P3.1 by splitting mock data into domain fixture files

I started Phase 3 by extracting the monolithic `src/mocks/data.ts` into domain fixtures under `src/mocks/fixtures/` for conversations, turns, events, timeline, and anomalies. I kept a compatibility layer in `data.ts` so existing imports continue to work while later Phase 3 tasks migrate stories and handlers to more structured factories/scenarios.

This delivered `P3.1` without introducing behavior changes and kept build/Storybook output stable.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue into the next unchecked tasks after Phase 2 completion and keep task/diary/changelog state synchronized with commits.

**Inferred user intent:** Advance PI-019 incrementally with low-risk, reviewable slices that preserve runtime stability.

**Commit (code):** `9f1db57a1c86578e397294496a622eac8c0c5bbf` — "refactor(mocks): split mock data into domain fixture files"

### What I did

- Created fixture domain files:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/conversations.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/turns.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/events.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/timeline.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/anomalies.ts`
- Converted legacy file into compatibility export layer:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/data.ts`
- Validated:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P3.1`

### Why

- `data.ts` had grown into a single large fixture source; splitting by domain is the baseline prerequisite for deterministic factories/scenarios and handler composition work in P3.2-P3.10.

### What worked

- Existing stories and handlers remained compatible through `data.ts` re-exports.
- Build and Storybook compilation passed without regressions.

### What didn't work

- While gathering file-level metrics, I initially ran a command with a wrong directory path and got:
  - `zsh:cd:1: no such file or directory: /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-debug/web`
- Fix:
  - Re-ran the command with the correct `web-agent-example/cmd/web-agent-debug/web` path.

### What I learned

- A compatibility export layer is a safe bridge for large fixture refactors because it decouples file-structure migration from downstream import migration.

### What was tricky to build

- `mockTurns`/`mockTurnDetail` depended on shared local helper state (`stdBlockMeta`) that had to remain co-located during extraction. Splitting too aggressively would have forced cross-file helper coupling. I kept the helper in `fixtures/turns.ts` to preserve readability and avoid circular import risk.

### What warrants a second pair of eyes

- Confirm fixture boundaries are suitable for upcoming factory/scenario tasks:
  - whether `mockMwTrace` should remain in `events.ts` or move to a separate middleware fixture module in P3.2.

### What should be done in the future

- Implement `P3.2` by introducing deterministic factories that consume these fixture baselines.

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/data.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/conversations.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/turns.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/events.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/timeline.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/fixtures/anomalies.ts`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- Fixture modules introduced: `5`
- `data.ts` role: compatibility export layer only (no inline mock objects).

## Step 20: Complete P3.2 by creating deterministic mock factory builders

I added a dedicated `src/mocks/factories/` layer that provides deterministic builders for conversation, turn, event, timeline, and anomaly domains. Builders are index-based and fixture-backed, so they produce stable outputs for the same inputs and keep story/test data generation explicit.

This keeps fixture definitions (`P3.1`) separate from generated variants and prepares the next steps for scenario composition and centralized MSW handler builders.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue PI-019 by completing the next unchecked task with an isolated implementation commit and synchronized ticket docs.

**Inferred user intent:** Move through the remaining checklist in small, auditable increments that keep frontend behavior stable.

**Commit (code):** `2e3c954c995b7af3e94e5124c1830761e1685872` — "feat(mocks): add deterministic fixture factory builders"

### What I did

- Created factory modules:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/common.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/conversationFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/turnFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/eventFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/timelineFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/anomalyFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/index.ts`
- Implemented deterministic builder patterns:
  - index-based fixture selection via `pickByIndex`
  - clone-safe object generation via `cloneMock`
  - list builders (`make*` + `make*List`) with optional per-item override callbacks
- Validated:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P3.2`

### Why

- Phase 3 requires reusable generated variants, not only static fixture arrays, to remove duplicated local story data and ad-hoc handler payloads in later tasks.

### What worked

- New factories compile and do not alter current story behavior (no consumption migration yet).
- Existing builds remained green after introducing the factory layer.

### What didn't work

- N/A

### What I learned

- A minimal deterministic builder API (index + override) is sufficient to unlock scenario migration while avoiding premature complexity.

### What was tricky to build

- `makeConversationDetail` needed to preserve detailed engine config defaults while still allowing deterministic variation by conversation index. I resolved this by combining a base detail template with index-selected summary fields, then layering overrides.

### What warrants a second pair of eyes

- Factory API shape consistency across domains (`makeX`, `makeXs`, `mapOverrides`) before Phase 3 story migrations rely on it heavily.

### What should be done in the future

- Complete `P3.3` and `P3.4` next:
  - scenario builders
  - deterministic id/time/seq utility helpers shared by factories.

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/common.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/conversationFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/turnFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/eventFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/timelineFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/anomalyFactory.ts`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- Factory modules introduced: `7`
- Domains covered by deterministic builders: `5` (conversations, turns, events, timeline, anomalies)

## Step 21: Complete P3.3 by adding reusable mock scenario modules

I completed `P3.3` by introducing a dedicated `src/mocks/scenarios/` layer for reusable story contexts. Each scenario family is typed and composed from the deterministic factory layer added in `P3.2`, so story migration can consume stable scenario objects instead of repeating inline arrays and ad-hoc overrides.

This keeps mock architecture progression clear: fixtures (`P3.1`) define static baselines, factories (`P3.2`) generate deterministic variants, and scenarios (`P3.3`) package representative story contexts for downstream Storybook migration tasks.

### Prompt Context

**User prompt (verbatim):** "continue."

**Assistant interpretation:** Proceed with the next unchecked PI-019 task, complete it end-to-end, and keep docs/task state synchronized.

**Inferred user intent:** Advance the checklist incrementally with auditable commits and continuous diary/changelog updates.

**Commit (code):** `fd2efd35646ee9f72db18b78780afb4b17d4eb02` — "feat(mocks): add reusable story scenario definitions"

### What I did

- Added scenario modules:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/overviewScenarios.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/timelineScenarios.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/eventInspectorScenarios.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/anomalyScenarios.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/index.ts`
- Defined reusable scenario records + helpers:
  - `overview`: default/empty/busy
  - `timeline`: default/selection/live/empty/many-items
  - `event inspector`: key event types + correlated/trust-check variants
  - `anomaly panel`: open/closed/empty/errors-only/many
- Validated:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P3.3`

### Why

- Storybook migration tasks (`P3.11` onward) need centralized reusable contexts so story files stop embedding large local data arrays and one-off state setup.

### What worked

- New scenario modules compile cleanly and are factory-backed, so generated data is deterministic and reusable.
- Build and Storybook build remained green after introducing scenario abstractions.

### What didn't work

- N/A

### What I learned

- Using component prop-derived scenario arg types (`Pick<...Props, ...>`) provides a practical type boundary that keeps scenario contracts aligned with UI component expectations.

### What was tricky to build

- Some scenario variants needed richer combinations (e.g., many-items timeline + selections + deterministic IDs/sequences). I kept those overrides local to scenarios for now and deferred shared id/time/seq helper extraction to `P3.4` to avoid mixing two checklist tasks in one commit.

### What warrants a second pair of eyes

- Scenario naming and coverage balance:
  - enough breadth to replace duplicated story fixtures,
  - without adding maintenance-heavy scenario permutations before migration is complete.

### What should be done in the future

- Complete `P3.4` next by introducing deterministic id/time/seq helper utilities and refactoring factories/scenario overrides to use them.

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/overviewScenarios.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/timelineScenarios.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/eventInspectorScenarios.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/anomalyScenarios.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/scenarios/index.ts`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- Scenario modules introduced: `5`
- Scenario families introduced: `4` (overview, timeline, event inspector, anomaly)
- Scenario consumption migration status: pending (`P3.11`-`P3.14`)

## Step 22: Complete P3.4 with deterministic id/time/seq helper utilities

I completed `P3.4` by adding a dedicated deterministic helper utility module for ids, timestamps, and sequence numbers, then wiring those helpers into factory list builders. The builders now synthesize stable unique overrides when list generation exceeds fixture lengths, while preserving baseline fixture realism for in-range indices.

This keeps factory behavior deterministic as scenarios/stories request larger lists and reduces ad-hoc hard-coded id/seq/time overrides in future migration work.

### Prompt Context

**User prompt (verbatim):** (same as Step 21)

**Assistant interpretation:** Continue to the next unchecked task and complete it in a focused, validated, and documented slice.

**Inferred user intent:** Keep progressing through PI-019 tasks with traceable commits and disciplined diary/changelog updates.

**Commit (code):** `b8b43aabe25d1201f4a7165fdabe46ba7dd7ac9b` — "feat(mocks): add deterministic id time seq utilities"

### What I did

- Added deterministic utility module:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/deterministic.ts`
  - helpers: `makeDeterministicId`, `makeDeterministicTimeMs`, `makeDeterministicIsoTime`, `makeDeterministicSeq`, `shouldApplyDeterministicOverrides`
- Added unit tests:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/deterministic.test.ts`
- Refactored factory list builders to apply deterministic synthetic overrides only after fixture-wrap:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/conversationFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/turnFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/eventFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/timelineFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/anomalyFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/index.ts` (exported deterministic helpers)
- Validated:
  - `npm run test:unit`
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P3.4`

### Why

- Without deterministic synthetic overrides, lists larger than fixture sizes wrap and duplicate ids/timestamps/sequences in ways that make story behavior less realistic and harder to reason about.

### What worked

- Factory generation remains fixture-faithful for baseline indices and becomes deterministic/stable for wrapped indices.
- Unit tests and builds all passed after the refactor.

### What didn't work

- N/A

### What I learned

- Applying deterministic overrides only when fixture-wrap occurs is a practical compromise between realistic baseline examples and scalable synthetic data generation.

### What was tricky to build

- I needed to avoid changing baseline fixture output for low indices while still guaranteeing unique synthetic identity fields for high indices. The solution was to gate deterministic overrides behind `shouldApplyDeterministicOverrides(absoluteIndex, fixtureCount)` and merge user `mapOverrides` last so consumers retain explicit control.

### What warrants a second pair of eyes

- Whether synthetic event id strategy (type-based prefix + deterministic ordinal) is sufficient for all upcoming story migration semantics, especially for tool correlation examples.

### What should be done in the future

- Complete `P3.5` next by deciding whether to keep `src/mocks/data.ts` as a temporary compatibility layer for the remaining story migration steps or remove it once all references are migrated.

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/deterministic.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/deterministic.test.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/conversationFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/turnFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/eventFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/timelineFactory.ts`
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/factories/anomalyFactory.ts`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run test:unit`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- New deterministic helper functions: `5`
- Factory modules updated in this step: `6`
- New unit test file: `1`

## Step 23: Complete P3.5 by formalizing `mocks/data.ts` compatibility contract

I completed `P3.5` by finalizing `src/mocks/data.ts` as an explicit legacy compatibility shim. The file already re-exported fixture modules from `P3.1`; this step codified that contract with clear migration guidance so it can safely persist until story migration tasks finish, then be removed intentionally.

This avoids premature removal risk because many current stories still import `../mocks/data`, while still making the deprecation/removal path explicit.

### Prompt Context

**User prompt (verbatim):** (same as Step 21)

**Assistant interpretation:** Continue through unchecked PI-019 tasks one at a time with focused commits and synchronized ticket docs.

**Inferred user intent:** Keep progress linear and reviewable while preventing accidental breakage from broad migrations.

**Commit (code):** `af06ce20723b971cc1cd2067ece79dbae8d2ce8f` — "chore(mocks): finalize legacy data shim contract"

### What I did

- Updated:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/data.ts`
- Added explicit contract comments:
  - this file is a legacy compatibility shim
  - keep until `P3.11`-`P3.14` migration is complete
  - new code should use `./fixtures/*` or `./factories/*`
- Verified current import footprint before deciding to keep shim:
  - `rg` showed `12` story imports of `../mocks/data`
- Validated:
  - `npm run build`
  - `npm run build-storybook`
- Checked off:
  - `P3.5`

### Why

- `P3.5` allows either removing `data.ts` or formalizing it as compatibility layer; removal now would break current stories before migration tasks are done.

### What worked

- Compatibility behavior stayed unchanged while migration intent is now explicit in code.
- Build/Storybook remained green.

### What didn't work

- N/A

### What I learned

- For staged architecture migration, explicit temporary-shim contracts reduce ambiguity and prevent accidental churn during intermediate phases.

### What was tricky to build

- The key decision was whether to remove `data.ts` now or keep it. Import inventory showed substantial active usage in stories, so immediate removal would create large unrelated diffs and violate the “one task at a time” slicing objective. I kept the shim and documented removal timing, which preserves momentum without hiding technical debt.

### What warrants a second pair of eyes

- Confirm that keeping `data.ts` through `P3.11`-`P3.14` aligns with reviewer preference for migration sequencing versus early hard-cutover.

### What should be done in the future

- Start `P3.6` by introducing centralized MSW handler builders (`createDebugHandlers.ts`, then `defaultHandlers.ts`) and then migrate existing handlers/stories off repeated ad-hoc handler blocks.

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/data.ts`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && rg -n \"from '../mocks/data'\" src/components/*.stories.tsx`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build-storybook`

### Technical details

- `src/mocks/data.ts` remains re-export-only (no inline fixture objects).
- Active story imports depending on `../mocks/data` at this step: `12`.

## Step 24: Complete P3.6 by creating reusable MSW debug handler builder

I completed `P3.6` by introducing `src/mocks/msw/createDebugHandlers.ts`, a reusable builder for all debug UI MSW endpoints. Instead of hard-coding handlers directly in one legacy file, this module defines a structured input contract (`DebugHandlerData`) and returns a complete handler set from those inputs.

This creates the foundation for `P3.7` and `P3.8`, where default handler bundles and existing handler files can be migrated onto this shared builder.

### Prompt Context

**User prompt (verbatim):** (same as Step 21)

**Assistant interpretation:** Continue task-by-task through remaining Phase 3 items with focused commits and synced docs.

**Inferred user intent:** Keep each migration slice small, composable, and ready for follow-up integration steps.

**Commit (code):** `1db6da80d57c9b642401c0acaa65a1ce5ad6f7e5` — "feat(mocks): add reusable debug msw handler builder"

### What I did

- Added:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/msw/createDebugHandlers.ts`
- Implemented `createDebugHandlers(options)` with:
  - `DebugHandlerData` input contract
  - optional `nowMs` and `nowIso` function injections for deterministic timeline/turn detail response values
  - centralized handlers for:
    - `/debug/conversations`
    - `/debug/conversation/:id`
    - `/debug/conversation/:id/sessions`
    - `/debug/turns`
    - `/debug/turn/:convId/:sessionId/:turnId`
    - `/debug/events/:convId`
    - `/debug/timeline`
    - `/debug/mw-trace/:convId/:inferenceId`
- Validated:
  - `npm run build`
- Checked off:
  - `P3.6`

### Why

- Handler definitions were previously monolithic and not configurable; a builder abstraction is needed before centralizing defaults and removing per-story duplication.

### What worked

- The new builder compiles cleanly and encapsulates existing handler behavior behind typed inputs.

### What didn't work

- N/A

### What I learned

- Adding `nowMs`/`nowIso` hooks at builder creation time keeps time-dependent responses testable and deterministic without changing endpoint call sites.

### What was tricky to build

- Preserving current behavior while generalizing required careful separation between static fixture data (input contract) and derived response behavior (404 handling, filtered turns, fallback turn detail). I mirrored existing endpoint semantics directly in the builder to minimize regression risk.

### What warrants a second pair of eyes

- Confirm that the `DebugHandlerData` contract is complete enough for upcoming scenario-driven handler composition in `P3.7`+.

### What should be done in the future

- Complete `P3.7` by adding `src/mocks/msw/defaultHandlers.ts` that wires current fixture defaults into `createDebugHandlers`.

### Code review instructions

- Review:
  - `web-agent-example/cmd/web-agent-debug/web/src/mocks/msw/createDebugHandlers.ts`
- Validate:
  - `cd web-agent-example/cmd/web-agent-debug/web && npm run build`

### Technical details

- New MSW builder modules: `1`
- Debug endpoints covered by builder: `8`


## Related

- Implementation plan:
  - `analysis/01-implementation-plan-for-ui-helper-css-system-and-storybook-mock-data-cleanup.md`
- Ticket index:
  - `index.md`
- Tasks:
  - `tasks.md`
