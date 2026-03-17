---
Title: Manuel investigation diary
Ticket: GP-43-REMOVE-STEPSETTINGSPATCH
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - config
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/service.go
      Note: Main Geppetto profile-resolution implementation currently applying StepSettingsPatch
    - Path: geppetto/pkg/profiles/runtime_settings_patch_resolver.go
      Note: Patch helper file targeted for deletion
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Representative downstream consumer applying profile patch data
    - Path: geppetto/ttmp/2026/03/17/GP-43-REMOVE-STEPSETTINGSPATCH--remove-stepsettingspatch-from-geppetto-profile-runtime-and-move-final-stepsettings-resolution-to-callers/design-doc/01-remove-stepsettingspatch-and-move-final-stepsettings-resolution-to-callers-design-and-implementation-guide.md
      Note: Primary GP-43 design document
    - Path: geppetto/ttmp/2026/03/17/GP-43-REMOVE-STEPSETTINGSPATCH--remove-stepsettingspatch-from-geppetto-profile-runtime-and-move-final-stepsettings-resolution-to-callers/scripts/01-stepsettingspatch-surface-inventory.sh
      Note: Ticket-local inventory helper
ExternalSources: []
Summary: Chronological GP-43 diary covering ticket creation, surface inventory, design decisions, and planning for StepSettingsPatch removal.
LastUpdated: 2026-03-17T15:25:00-04:00
WhatFor: Use this diary to understand how the StepSettingsPatch removal plan was formed and what evidence supports the recommended caller-owned final StepSettings boundary.
WhenToUse: Use when reviewing the GP-43 design, validating its evidence, or continuing implementation later.
---

# Manuel investigation diary

## Goal

Record the analysis behind GP-43, including the reason for deleting `StepSettingsPatch`, the observed surface area, and the implementation planning work.

## Step 1: Create the ticket and inventory the current StepSettingsPatch surface

The first task was to separate this cleanup from GP-41. GP-41 removes request-level overrides. This new ticket removes `StepSettingsPatch` itself and the associated `BaseStepSettings` plus `EffectiveStepSettings` flow. The user’s architecture direction was clear: final engine configuration should be resolved and cached by callers rather than being partially merged inside Geppetto profile resolution.

### Prompt Context

**User prompt (verbatim):** "ok, create a new ticket to do that, we should delete StepSettingsPatch entirely. Add an implementation plan and detailed implementation tasks to the new ticket. Then let's plan out the removal of overrides from the main path"

**Assistant interpretation:** Create a dedicated Geppetto ticket for full `StepSettingsPatch` deletion, add an implementation guide and detailed tasks, and then use that work to inform planning for the GP-41 override-removal path.

**Inferred user intent:** Finish drawing the architectural boundary: profile runtime should no longer participate in final engine settings construction.

**Commit (code):** N/A — documentation-only ticket work

### What I did

- Created the new ticket:

```bash
docmgr ticket create-ticket \
  --ticket GP-43-REMOVE-STEPSETTINGSPATCH \
  --title "Remove StepSettingsPatch from Geppetto profile runtime and move final StepSettings resolution to callers" \
  --topics geppetto,profile-registry,architecture,settings
```

- Added the main design doc and this diary with `docmgr doc add`.
- Ran a focused `rg` inventory for:
  - `StepSettingsPatch`
  - `step_settings_patch`
  - `ApplyRuntimeStepSettingsPatch`
  - `MergeRuntimeStepSettingsPatches`
  - `EffectiveStepSettings`
  - `BaseStepSettings`

### Why

- The surface is larger than GP-41 and deserves its own ticket.
- The `rg` pass was necessary to understand the full removal blast radius before writing tasks.

### What worked

- The inventory clearly showed the expected structure:
  - Geppetto core APIs and helpers,
  - Pinocchio web chat and helper layers,
  - GEC-RAG runtime composition,
  - Temporal Relationships runtime composition,
  - docs/examples/tests/migration code.

### What didn't work

- N/A in this step. The surface was easy to identify once the right search terms were chosen.

### What I learned

- `StepSettingsPatch` is not a minor implementation detail. It is still part of public profile semantics, caller helpers, and documentation.
- The strongest signal for the removal direction is that final engine creation already happens from concrete `*settings.StepSettings`, while profile code only exists to derive those settings indirectly.

### What was tricky to build

The main judgment call was deciding whether this ticket should also propose removing `SystemPrompt`, `Middlewares`, or `Tools` from `RuntimeSpec`. I kept the ticket narrower. The architectural problem here is specifically partial engine configuration through patching.

### What warrants a second pair of eyes

- Whether any external users depend on `effectiveStepSettings` in the JS API or profile-resolution outputs.
- Whether legacy profile migration should be handled in the same implementation series or in a follow-up cleanup ticket.

### What should be done in the future

- Execute GP-43 after GP-41 planning is clear so the two removals can be staged cleanly rather than colliding.

### Code review instructions

- Start with the main design doc.
- Re-run the inventory script to see all live references.

### Technical details

- Key inventory command:

```bash
rg -n "StepSettingsPatch|step_settings_patch|ApplyRuntimeStepSettingsPatch|MergeRuntimeStepSettingsPatches|EffectiveStepSettings|BaseStepSettings" \
  geppetto pinocchio 2026-03-16--gec-rag temporal-relationships \
  --glob '!**/ttmp/**'
```

## Step 2: Turn the inventory into a phased implementation plan

After the inventory, I wrote the design doc and task list around one central principle: Geppetto profile resolution should stop returning final engine settings. Instead, callers should resolve and cache final `*settings.StepSettings`, then use Geppetto only for engine creation and inference execution.

### What I did

- Wrote the main design doc with:
  - a current-state summary,
  - the target architecture,
  - a phased implementation plan,
  - pseudocode for before/after flows,
  - risks and open questions.
- Replaced the placeholder `tasks.md` with a detailed phase-by-phase checklist.
- Added `scripts/01-stepsettingspatch-surface-inventory.sh` for future continuation.

### Why

- The ticket needed to be implementation-ready, not just an opinion dump.
- The task list should be specific enough that another engineer can pick up Phase 2 or Phase 4 without redoing the investigation.

### What worked

- The phases were easy to structure:
  1. contract decisions,
  2. Geppetto core removal,
  3. caller-facing cleanup,
  4. downstream migration,
  5. docs and migration cleanup,
  6. validation.

### What didn't work

- N/A in this step.

### What I learned

- The most important deletion is not just `StepSettingsPatch`; it is the entire trio:
  - `StepSettingsPatch`
  - `BaseStepSettings`
  - `EffectiveStepSettings`

### What was tricky to build

The main tricky point was making the plan detailed without over-prescribing the exact app-side resolver API. Different apps may want different resolver shapes, and the ticket should not force a new over-general abstraction.

### What warrants a second pair of eyes

- Whether a short-lived compatibility adapter outside Geppetto core would reduce migration pain.
- Whether GP-43 should be implemented before or after GP-41’s main-path override removal.

## Step 3: Freeze the ideal API and fold RuntimeKeyFallback into the same hard cut

After the initial GP-43 planning, the next task was to answer the open API question directly: what should the ideal app-facing boundary actually look like if we were not trying to preserve old behavior? The conclusion is that the application should own the entire resolved runtime object, including final `StepSettings` and runtime identity. That makes `RuntimeKeyFallback` part of the same architectural mistake as `StepSettingsPatch`, not a separate cleanup.

### Prompt Context

**User prompt (verbatim):** "alright, so let's write a document for this ideal API and to implement it, and document all the implementation steps necessary to get there. We don't need to keep backwards compatibility so we can do a hard cut. Add the implementation document to the ticket and upload to remarkable. Update the previous document accordingly. You can also cut off runtimekeyfallback in the same go i think (see GP-45)."

**Assistant interpretation:** Add a new GP-43 implementation document that defines the ideal API and a detailed hard-cut implementation plan, update the earlier GP-43 material to match, and treat `RuntimeKeyFallback` removal as part of the same change series.

**Inferred user intent:** Stop treating GP-43 as only a patch-format deletion and instead make it the explicit architecture ticket for the cleaner app-owned runtime boundary.

**Commit (code):** N/A — ticket documentation work

### What I did

- Added a new design document:
  - `design-doc/02-ideal-app-facing-api-and-hard-cut-implementation-plan.md`
- Updated:
  - `index.md`
  - `design-doc/01-remove-stepsettingspatch-and-move-final-stepsettings-resolution-to-callers-design-and-implementation-guide.md`
  - `tasks.md`
  - `changelog.md`
- Reframed GP-43 around one hard cut:
  - remove `StepSettingsPatch`
  - remove `BaseStepSettings`
  - remove `EffectiveStepSettings`
  - remove `RuntimeKeyFallback`
  - no temporary compatibility adapter in Geppetto core

### Why

- The original GP-43 plan still left some architectural questions open, especially around whether a compatibility adapter should exist and whether `RuntimeKeyFallback` should wait for a separate later ticket.
- The user’s direction was clearer than the earlier draft: define the elegant API and optimize the implementation plan for that end state rather than for incremental compatibility.

### What worked

- The current Pinocchio webchat flow made the right target boundary easy to explain:
  - app resolves request and runtime,
  - app builds final `StepSettings`,
  - app builds engine and filtered tool registry,
  - Geppetto runs the concrete engine/session/toolloop.
- Once that boundary was explicit, it became obvious that `RuntimeKeyFallback` belongs on the app side too.

### What didn't work

- The original GP-43 task list was too tentative in Phase 1. It treated the API shape and migration strategy as open questions rather than as a design decision to lock before implementation starts.

### What I learned

- `RuntimeKeyFallback` is not just a convenience field. It is another symptom of Geppetto still owning more of runtime resolution than it should.
- The elegant API is not “Geppetto returns better resolved profile objects.” The elegant API is “the app resolves a concrete runtime and Geppetto only consumes it.”

### What was tricky to build

The tricky part was making the design specific enough to guide implementation without accidentally inventing another shared “universal runtime resolver” abstraction. The right answer is to standardize the boundary, not the exact app-side helper struct. So the new doc specifies the boundary precisely but leaves room for Pinocchio, GEC-RAG, and Temporal Relationships to each have app-specific resolved-runtime structs.

### What warrants a second pair of eyes

- Whether Geppetto should gain a tiny convenience helper like `BuildEngine(ctx, EngineSpec)` or whether existing engine-construction helpers are already sufficient.
- Whether the legacy profile migration path should hard-error on `step_settings_patch` immediately or receive a one-shot migration helper outside core runtime resolution.

### What should be done in the future

- Refresh the GP-43 bundle on reMarkable so the portable ticket copy reflects the new hard-cut plan.
- Use this updated GP-43 plan as the basis for implementation ordering before touching GP-45 separately.

### Code review instructions

- Start with:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/17/GP-43-REMOVE-STEPSETTINGSPATCH--remove-stepsettingspatch-from-geppetto-profile-runtime-and-move-final-stepsettings-resolution-to-callers/design-doc/02-ideal-app-facing-api-and-hard-cut-implementation-plan.md`
- Then compare the updated framing in:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/17/GP-43-REMOVE-STEPSETTINGSPATCH--remove-stepsettingspatch-from-geppetto-profile-runtime-and-move-final-stepsettings-resolution-to-callers/design-doc/01-remove-stepsettingspatch-and-move-final-stepsettings-resolution-to-callers-design-and-implementation-guide.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/17/GP-43-REMOVE-STEPSETTINGSPATCH--remove-stepsettingspatch-from-geppetto-profile-runtime-and-move-final-stepsettings-resolution-to-callers/tasks.md`

### Technical details

- The new document intentionally treats GP-45’s `RuntimeKeyFallback` concern as part of GP-43 implementation, not as an independent API design axis.
