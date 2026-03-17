---
Title: Manuel investigation diary
Ticket: GP-41-REMOVE-PROFILE-OVERRIDES
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - pinocchio
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/service.go
      Note: Key evidence source for override resolution behavior
    - Path: geppetto/ttmp/2026/03/17/GP-41-REMOVE-PROFILE-OVERRIDES--remove-request-level-profile-override-functionality-from-geppetto-profile-resolution/design-doc/01-remove-geppetto-request-level-profile-overrides-design-and-implementation-guide.md
      Note: Primary deliverable produced from this diary
    - Path: geppetto/ttmp/2026/03/17/GP-41-REMOVE-PROFILE-OVERRIDES--remove-request-level-profile-override-functionality-from-geppetto-profile-resolution/design-doc/02-runtime-glossary-across-geppetto-and-pinocchio.md
      Note: Supplementary glossary produced to explain overloaded Runtime terminology before implementation
    - Path: geppetto/ttmp/2026/03/17/GP-41-REMOVE-PROFILE-OVERRIDES--remove-request-level-profile-override-functionality-from-geppetto-profile-resolution/scripts/01-override-surface-inventory.sh
      Note: Ticket-local inventory helper created during investigation
ExternalSources: []
Summary: Chronological investigation diary for GP-41 covering ticket creation, evidence gathering, document authoring, runtime glossary additions, validation, and reMarkable delivery.
LastUpdated: 2026-03-17T17:07:00-04:00
WhatFor: Use this diary to understand how the GP-41 analysis and glossary were produced, what evidence shaped the conclusions, and how to continue the work.
WhenToUse: Use when reviewing the analysis process, validating claims, onboarding into the runtime-related code paths, or continuing implementation later.
---


# Manuel investigation diary

## Goal

Record the analysis work for GP-41, including exact commands, evidence sources, decisions, tricky points, and handoff guidance.

## Step 1: Create the override-removal ticket and map the surface area

The first step was to create a dedicated Geppetto ticket rather than extending GP-40. GP-40 was about opinionated runner APIs. This new request is a distinct architectural cleanup focused on removing request-level profile overrides. I kept the work in a fresh ticket workspace so the scope stayed narrow and the resulting guide could be implementation-oriented.

The main analytical task in this step was to separate two things that looked similar but are not the same: profile-based runtime configuration and request-level runtime overrides. The evidence showed that current downstream apps do rely on profiles and `runtime.step_settings_patch`, but mostly do not rely on per-request override mutation as a primary product workflow.

### Prompt Context

**User prompt (verbatim):** "ok, add a ticket to geppetto to remove the override functionalit, because we actually never use it in pinocchio or gec-rag, and so it just adds complexity. 

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new Geppetto ticket dedicated to removing request-level profile overrides, produce a detailed intern-oriented design and implementation guide with evidence, keep a diary, and publish the docs to reMarkable.

**Inferred user intent:** Simplify the profile subsystem by deleting a feature that is not pulling its weight in actual product usage, and leave behind a clear implementation guide that another engineer can execute confidently.

**Commit (code):** N/A — documentation-only ticket work

### What I did

- Confirmed the relevant skills and workflow:
  - `ticket-research-docmgr-remarkable`
  - `docmgr`
  - `diary`
- Checked the current docmgr root with:

```bash
docmgr status --summary-only
```

- Listed existing tickets to find the next free Geppetto slot:

```bash
docmgr ticket list | sed -n '1,260p'
```

- Created the new ticket:

```bash
docmgr ticket create-ticket --ticket GP-41-REMOVE-PROFILE-OVERRIDES --title "Remove request-level profile override functionality from Geppetto profile resolution" --topics geppetto,profile-registry,cleanup,architecture
```

- Added the primary design doc and diary:

```bash
docmgr doc add --ticket GP-41-REMOVE-PROFILE-OVERRIDES --doc-type design-doc --title "Remove Geppetto request-level profile overrides: design and implementation guide"
docmgr doc add --ticket GP-41-REMOVE-PROFILE-OVERRIDES --doc-type reference --title "Manuel investigation diary"
```

- Mapped the relevant Geppetto and downstream surfaces with `rg`, `nl -ba`, and `sed`.

### Why

- A separate ticket keeps the analysis focused on one cleanup topic.
- The new ticket makes it easier to hand the work to an intern or another engineer without mixing contexts.
- The evidence pass was necessary because the user’s claim was plausible, but the code still exposed several override surfaces that had to be accounted for explicitly.

### What worked

- The new ticket was created cleanly as `GP-41-REMOVE-PROFILE-OVERRIDES`.
- The core evidence was straightforward to find in:
  - `geppetto/pkg/profiles`
  - `geppetto/pkg/js/modules/geppetto`
  - `pinocchio/cmd/web-chat`
  - `2026-03-16--gec-rag/internal/webchat`
  - `temporal-relationships/internal/extractor/httpapi`
- The evidence strongly supported the main distinction:
  - profiles and `step_settings_patch` are real and in active use
  - request-level overrides are mostly exposed capability, not a dominant runtime usage pattern

### What didn't work

- One `rg` command included a non-existent path and returned:

```text
rg: temporal-relationships/web: No such file or directory (os error 2)
```

- `docmgr doc list --ticket GP-41-REMOVE-PROFILE-OVERRIDES` unexpectedly returned:

```text
No documents found.
```

even though `docmgr doc add` had created the files. I worked around that by using the explicit file paths from the `doc add` output and by inspecting the ticket directory directly with `ls -R`.

### What I learned

- The removal argument is stronger when phrased precisely:
  - do not say profiles are unused,
  - say request-level profile overrides are not the primary usage pattern.
- Temporal Relationships is a useful control case because it uses resolved profiles without exposing `request_overrides` at all.
- A lot of the architectural complexity lives in policy merge and validation, not just in the request resolver signatures.

### What was tricky to build

The tricky part was keeping the recommendation narrow. It would have been easy to overreach and propose replacing `step_settings_patch` or redesigning profile storage entirely. That would have made the ticket broader and less actionable. I deliberately kept the ticket focused on removing request-level overrides while preserving the existing profile-based runtime model.

Another subtle point was distinguishing transport-level exposure from product reliance. Pinocchio and GEC-RAG do carry `request_overrides` in their request shapes, but that alone does not mean the feature is central. The important runtime evidence is that both systems resolve a profile and then apply the resolved runtime wholesale.

### What warrants a second pair of eyes

- Whether any external API consumers rely on `request_overrides` for Pinocchio web chat.
- Whether legacy YAML files containing override policy fields should be tolerated temporarily or removed immediately.
- Whether `PolicySpec` should shrink immediately to only `ReadOnly` or pass through a short deprecation phase.

### What should be done in the future

- Execute the removal in Geppetto core.
- Clean up JS bindings and examples.
- Simplify Pinocchio and GEC-RAG request contracts.
- Audit docs and examples for stale override references.

### Code review instructions

- Start with the main guide in `design-doc/01-remove-geppetto-request-level-profile-overrides-design-and-implementation-guide.md`.
- Then review these core evidence files:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/types.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/web-chat/profile_policy.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/resolver.go`
- Re-run the inventory script:

```bash
bash geppetto/ttmp/2026/03/17/GP-41-REMOVE-PROFILE-OVERRIDES--remove-request-level-profile-override-functionality-from-geppetto-profile-resolution/scripts/01-override-surface-inventory.sh
```

### Technical details

- Key commands used:

```bash
rg -n "request_overrides|RequestOverrides|overrideKey|AllowedOverrideKeys|step_settings_patch" geppetto pinocchio 2026-03-16--gec-rag temporal-relationships --glob '!**/node_modules/**'
nl -ba geppetto/pkg/profiles/service.go | sed -n '1,240p'
nl -ba geppetto/pkg/profiles/registry.go | sed -n '1,160p'
nl -ba geppetto/pkg/profiles/types.go | sed -n '1,140p'
nl -ba geppetto/pkg/profiles/stack_merge.go | sed -n '140,210p'
nl -ba geppetto/pkg/profiles/validation.go | sed -n '70,120p'
nl -ba geppetto/pkg/js/modules/geppetto/api_profiles.go | sed -n '220,285p'
nl -ba geppetto/pkg/js/modules/geppetto/api_engines.go | sed -n '240,295p'
nl -ba pinocchio/pkg/webchat/http/api.go | sed -n '1,80p'
nl -ba pinocchio/cmd/web-chat/profile_policy.go | sed -n '230,315p'
nl -ba pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx | sed -n '150,195p'
nl -ba 2026-03-16--gec-rag/web/src/api/chatApi.ts | sed -n '1,80p'
nl -ba 2026-03-16--gec-rag/internal/webchat/resolver.go | sed -n '160,290p'
nl -ba temporal-relationships/internal/extractor/httpapi/run_chat_handlers.go | sed -n '1,120p'
nl -ba temporal-relationships/internal/extractor/httpapi/run_chat_transport.go | sed -n '540,585p'
```

## Step 2: Write the guide, add the inventory script, validate, and prepare delivery

After the evidence pass, I replaced the ticket templates with a complete index, task list, changelog, design document, and this diary. I also added a small shell script under `scripts/` so someone continuing the work can quickly re-scan the override surface before implementation.

The goal in this step was not just to summarize findings, but to leave behind a guide that an intern could use as a practical execution manual. That required turning the evidence into a phased file-by-file plan, diagrams, pseudocode, risks, and concrete examples rather than stopping at architectural opinion.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the ticket deliverables end-to-end, validate them, and upload the result to reMarkable.

**Inferred user intent:** Produce a durable, handoff-quality design artifact rather than a short chat answer.

**Commit (code):** N/A — documentation-only ticket work

### What I did

- Replaced ticket template files with detailed content using `apply_patch`.
- Added `scripts/01-override-surface-inventory.sh`.
- Planned the required `docmgr doc relate`, `docmgr doctor`, and `remarquee` steps.

### Why

- The ticket needed to become self-contained and continuation-friendly.
- The inventory script reduces the cost of future review after code changes start.

### What worked

- The guide now explains:
  - the current architecture,
  - the actual downstream usage pattern,
  - the rationale for removal,
  - the proposed API shape,
  - the implementation phases,
  - the test plan,
  - the risks and open questions.

### What didn't work

- N/A at this stage beyond the earlier discovery-tool hiccups.

### What I learned

- The strongest intern-facing explanation is to emphasize that `step_settings_patch` and `request_overrides` are different layers. The ticket removes the second one, not the first.

### What was tricky to build

The main challenge was calibration of detail. The guide needed to be detailed enough for an intern, but still focused on this exact removal rather than becoming a general profile-registry book. The solution was to keep the guide highly structured and use file-oriented explanations instead of broad abstractions.

### What warrants a second pair of eyes

- The wording around external compatibility and deprecation.
- Whether the script inventory should later become a shared repo tool rather than staying ticket-local.

### What should be done in the future

- Once implementation starts, append new diary steps with exact diffs, test failures, and final verification commands.

### Code review instructions

- Review the ticket documents in this order:
  1. `index.md`
  2. `design-doc/01-remove-geppetto-request-level-profile-overrides-design-and-implementation-guide.md`
  3. `reference/01-manuel-investigation-diary.md`
  4. `scripts/01-override-surface-inventory.sh`

### Technical details

- Files authored in this step:
  - `index.md`
  - `tasks.md`
  - `changelog.md`
  - `design-doc/01-remove-geppetto-request-level-profile-overrides-design-and-implementation-guide.md`
  - `reference/01-manuel-investigation-diary.md`

## Step 3: Add a runtime glossary so GP-41 readers can decode the overloaded terminology

After the original ticket docs were done, a follow-up question exposed a real onboarding issue: there are many unrelated symbols named `Runtime*` across Geppetto and Pinocchio, and the names are overloaded enough that a new engineer could easily misread the system. Before removing request-level overrides, it was worth documenting the vocabulary itself.

The main goal in this step was not to produce another architecture essay. It was to create a precise glossary that says which `Runtime*` names belong to profile resolution, which belong to Pinocchio app composition, which belong to JavaScript VMs, which belong to timeline projection, and which belong to turn metadata attribution.

### Prompt Context

**User prompt (verbatim):** "Ok, let's go to GP-41 . I see an awful lot of uses of things called Runtime* something or another. Make a glossary of all the words that have Runtime in them across pinocchio and geppetto and add a couple of example use as a detailed document in the ticket, and upload to remarkable."

**Assistant interpretation:** Add a new GP-41 document that explains the different Runtime-named symbols across Geppetto and Pinocchio, include practical examples, and publish the updated ticket bundle to reMarkable.

**Inferred user intent:** Reduce the naming confusion before implementation proceeds, especially for an intern or reviewer who has to navigate the code without already knowing the subsystem boundaries.

**Commit (code):** N/A — ticket documentation update

### What I did

- Created a new design doc:

```bash
docmgr doc add --ticket GP-41-REMOVE-PROFILE-OVERRIDES --doc-type design-doc --title "Runtime glossary across Geppetto and Pinocchio"
```

- Collected the architecture-significant Runtime-named identifiers with targeted `rg` searches across `geppetto` and `pinocchio`.
- Read and anchored the glossary against these files:
  - `geppetto/pkg/profiles/types.go`
  - `geppetto/pkg/profiles/registry.go`
  - `geppetto/pkg/profiles/service.go`
  - `geppetto/pkg/profiles/slugs.go`
  - `pinocchio/pkg/inference/runtime/composer.go`
  - `pinocchio/cmd/web-chat/runtime_composer.go`
  - `geppetto/pkg/js/runtime/runtime.go`
  - `geppetto/pkg/inference/tools/scopedjs/runtime.go`
  - `pinocchio/pkg/webchat/timeline_js_runtime.go`
  - `pinocchio/pkg/webchat/timeline_registry.go`
  - `geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go`
  - `geppetto/pkg/turns/keys_gen.go`
- Wrote a subsystem-grouped glossary with:
  - profile runtime terms,
  - app-owned conversation runtime terms,
  - JS runtime terms,
  - scoped JS runtime terms,
  - timeline runtime terms,
  - runtime attribution terms.
- Added three detailed example flows:
  - profile resolution into Pinocchio runtime composition,
  - timeline JS runtime loading,
  - runtime attribution copied into provider event metadata.
- Updated the ticket index, tasks, and changelog to reference the new glossary.

### Why

- The request override cleanup is hard to review if the reader does not know which `runtime` layer they are looking at.
- The codebase uses the same word for profile data, engine composition, JS VMs, and metadata. That is enough ambiguity to justify its own doc.
- The examples make the glossary useful during implementation rather than only during passive reading.

### What worked

- The symbols grouped naturally into four main runtime meanings, which made the glossary much clearer than a flat alphabetical list.
- `pinocchio/cmd/web-chat/runtime_composer.go` turned out to be the best bridge example because it consumes Geppetto profile runtime data and produces app-owned `ComposedRuntime`.
- `pinocchio/pkg/webchat/timeline_registry.go` made the timeline runtime semantics concrete because it documents the execution order and consume behavior directly.

### What didn't work

- A raw `rg` sweep finds many test names and local variables containing `runtime`, which would have made the glossary noisy and less useful. I intentionally curated the glossary to architecture-significant identifiers and called that out explicitly in the appendix.

### What I learned

- There are at least four different architectural meanings of `runtime` in active use:
  - profile runtime,
  - app-composed runtime,
  - JS execution runtime,
  - runtime attribution metadata.
- The main confusion point is not inside Geppetto alone. It is the boundary between Geppetto profile runtime and Pinocchio app runtime.
- The Pinocchio webchat package still carries some overlapping runtime request shapes, which is useful context for future cleanup.

### What was tricky to build

The tricky part was deciding what counts as "all the words that have Runtime in them." A literal grep dump would satisfy the letter of the request but fail the real goal. The actual goal was clarity, so I used a curated lexicon of architecture-significant symbols and documented that scope explicitly.

### What warrants a second pair of eyes

- Whether Pinocchio should eventually collapse its overlapping runtime request contracts into one public shape.
- Whether some of the runtime naming should be normalized after GP-41 lands, especially around `RuntimeKey` versus `ProfileKey`.

### What should be done in the future

- Use this glossary as a naming checkpoint while removing request overrides.
- Revisit it if GP-41 or later cleanup changes any public runtime names.
- Consider a later targeted ticket for runtime naming normalization if the duplicated vocabulary remains costly.

### Code review instructions

- Read the glossary before reviewing GP-41 implementation patches.
- Validate the example flows against:
  - `geppetto/pkg/profiles/service.go`
  - `pinocchio/cmd/web-chat/runtime_composer.go`
  - `pinocchio/pkg/webchat/timeline_registry.go`
  - `geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go`

### Technical details

- Main new document:
  - `design-doc/02-runtime-glossary-across-geppetto-and-pinocchio.md`
- Key inventory command shape:

```bash
rg -n "Runtime[A-Za-z0-9_]*|[A-Za-z0-9_]*Runtime[A-Za-z0-9_]*" geppetto pinocchio --glob '!**/ttmp/**'
```
  - `scripts/01-override-surface-inventory.sh`

## Step 4: Implement GP-41 Phase 1 in Geppetto core

After the documentation and glossary work, I started the first actual code-removal pass. The goal for Phase 1 was intentionally narrow: remove `RequestOverrides` from the Geppetto core resolution path without yet touching downstream HTTP contracts or deleting override-related policy fields. This keeps the first commit reviewable and reduces the chance of mixing architectural cleanup with app integration churn.

### Prompt Context

**User prompt (verbatim):** "alright, let's tackle GP-41, task by task, committing as we go."

**Assistant interpretation:** Start implementing GP-41 incrementally, validate each phase, and commit at clean boundaries.

**Inferred user intent:** Drive the cleanup through small, reviewable code changes instead of one large cross-repo batch.

**Commit (code):** Pending at the time of this diary entry

### What I did

- Removed `RequestOverrides` from `geppetto/pkg/profiles.ResolveInput`.
- Simplified `resolveRuntimeSpec` so it now only clones the merged runtime instead of parsing and applying request-level mutations.
- Deleted the request-override helper functions from `geppetto/pkg/profiles/service.go`:
  - `normalizeOverrideMap`
  - `enforceOverridePolicy`
  - `parseMiddlewareOverrideValue`
  - `parseToolOverrideValue`
  - `parseStepSettingsPatchOverrideValue`
- Updated Geppetto JS runtime bindings so `requestOverrides` now produces an explicit removal error instead of silently continuing to support it.
- Removed the stale `requestOverrides` properties from:
  - `pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
  - `pkg/doc/types/geppetto.d.ts`
- Reworked the profile service tests to cover profile runtime integration and stacked runtime integration without request overrides, and deleted the override-policy-specific tests.

### Why

- `RequestOverrides` is the narrowest entry point for request-level runtime mutation. Removing it from the core resolver makes the main path simpler immediately.
- The helper functions and tests around override parsing were only there because `RequestOverrides` existed. Keeping them after the field removal would leave dead complexity behind.
- The JS `.d.ts` surface had to match the runtime behavior. Leaving `requestOverrides` in the generated declarations would advertise an API that now intentionally errors.

### What worked

- The core change was localized. The main functional edits were in:
  - `geppetto/pkg/profiles/registry.go`
  - `geppetto/pkg/profiles/service.go`
  - `geppetto/pkg/profiles/service_test.go`
  - `geppetto/pkg/js/modules/geppetto/api_profiles.go`
  - `geppetto/pkg/js/modules/geppetto/api_engines.go`
- The remaining override-key normalization helpers (`canonicalOverrideKey`, `camelToSnake`) still have live use in stack merge policy normalization, so they could be left in place for a later phase without breaking anything.
- Focused validation already passed with:

```bash
go test ./pkg/profiles/... ./pkg/js/modules/geppetto/...
```

### What didn't work

- The first pass removed runtime support but left the generated TypeScript declarations still advertising `requestOverrides`. I caught that by searching the JS and doc surfaces again before committing.

### What I learned

- The request override feature was not just one resolver field. It had at least three layers that all needed to line up:
  - Go core API shape
  - runtime implementation and tests
  - JS binding and generated TypeScript declaration surface

### What was tricky to build

The tricky part was separating this Phase 1 work from the later policy cleanup. `PolicySpec` still mentions override-related fields, and stack merge still normalizes override keys for policy merging. That looks stale now, but it belongs to the next phase. Deleting all of it in one patch would make the review much noisier and would blur the distinction between "remove request-time mutation" and "remove the last pieces of override-related schema."

### What warrants a second pair of eyes

- Whether the JS bindings should reject removed `requestOverrides` with a hard error, as I implemented, or whether they should silently ignore the field for one short compatibility window.
- Whether the generated `.d.ts` file should be regenerated by an existing project script in a later cleanup, rather than manually patched as part of this first pass.

### Technical details

- Focused search used to catch stale JS surface references:

```bash
rg -n "requestOverrides|request_overrides|RequestOverrides" geppetto/pkg/js/modules/geppetto geppetto/pkg/doc/types geppetto/examples/js --glob '!**/ttmp/**'
```

- Focused validation command:

```bash
go test ./pkg/profiles/... ./pkg/js/modules/geppetto/...
```
