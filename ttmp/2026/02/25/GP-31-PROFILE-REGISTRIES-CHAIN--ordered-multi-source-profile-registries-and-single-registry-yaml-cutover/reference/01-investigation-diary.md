---
Title: Investigation diary
Ticket: GP-31-PROFILE-REGISTRIES-CHAIN
Status: active
Topics:
    - profile-registry
    - pinocchio
    - geppetto
    - migration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-31-PROFILE-REGISTRIES-CHAIN--ordered-multi-source-profile-registries-and-single-registry-yaml-cutover/design-doc/01-implementation-guide-ordered-profile-registries-chain-and-single-registry-yaml-cutover.md
      Note: Main design deliverable for GP-31
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/codec_yaml.go
      Note: Existing YAML format behavior analyzed for single-registry hard cut
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service.go
      Note: Existing resolution/fingerprint/provenance behavior analyzed for reuse
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/profile_api.go
      Note: Existing CRUD registry selection behavior analyzed for GP-31 scope
ExternalSources: []
Summary: Diary for scoping and documenting GP-31 ordered source-chain registry loading design.
LastUpdated: 2026-02-25T16:58:20-05:00
WhatFor: Capture rationale and decisions for GP-31 planning deliverable.
WhenToUse: Use when reviewing why GP-31 is structured the way it is and what tradeoffs were accepted.
---

# Investigation diary

## Goal

Track how GP-31 design scope was converted from user proposal into a concrete implementation guide and task breakdown.

## Step 1: Define GP-31 architecture and hard-cut behavior

This step translated the requested source-chain model (`--profile-registries file1,file2,file3`) into a concrete implementation guide that is compatible with the existing stack-profile resolution engine. The focus was to preserve deterministic behavior and avoid reintroducing overlay complexity.

The design intentionally keeps source chaining as a routing/ownership concern and reuses existing stack merge + provenance + fingerprint logic unchanged.

### Prompt Context

**User prompt (verbatim):** "so i would propose something like: --profile-registries file1,file2,file3 where we automatically determine if it's yaml or sqlite, and i tink the yaml would contain the slug and the db can contain multiple registries, and then you just chose profile names and the resolution is done in the order of the registries loaded. And for CRUD, allow all registries for now. Can you write up a detailed implementation guide and make a new ticket for this? This would also mean updating our yaml config format now to only contain a single registry."

**Assistant interpretation:** create a new planning ticket and produce a detailed implementation guide for ordered multi-source registry loading and single-registry YAML runtime format.

**Inferred user intent:** move from conceptual direction to executable engineering plan that can be implemented phase-by-phase without ambiguity.

### What I did

1. Created ticket `GP-31-PROFILE-REGISTRIES-CHAIN`.
2. Added a design doc with:
   - source autodetection,
   - ordered profile resolution semantics,
   - registry ownership/write routing,
   - single-registry YAML hard cut,
   - CRUD exposure scope and risks.
3. Added granular tasks by phase (settings, loader, chain router, resolution semantics, YAML cutover, tests, docs).
4. Updated ticket index and changelog for review readiness.

### Why

1. The requested behavior affects both runtime selection and storage semantics; it required a full contract definition before coding.
2. Existing code already has mature stack merge/provenance behavior; plan should reuse this rather than create parallel composition layers.

### What worked

1. Existing service abstractions already support most of the required mechanics (registry slugs, cross-registry stack refs, fingerprint/provenance metadata).
2. A router-style source chain model cleanly supports ordered resolution without overlay merge semantics.

### What didn't work

N/A

### What I learned

1. The key complexity is not stack merge itself; it is source ownership and deterministic profile lookup rules across multiple stores.
2. Exposing all registries in CRUD while using YAML for private credentials creates clear data-exposure risk and should be called out explicitly.

### What was tricky to build

The tricky part was reconciling three constraints without introducing new complexity:

1. ordered profile search across registries,
2. single source ownership for writes,
3. no overlay abstraction.

This required explicitly separating “registry chain routing” from “stack field merge semantics.”

### What warrants a second pair of eyes

1. Startup policy for duplicate registry slugs across sources (hard fail vs tolerated-first-wins).
2. Error mapping for writes against read-only (YAML) sources in CRUD handlers.
3. Whether profile trace payloads need immediate redaction if YAML registries include secrets and CRUD read exposure stays open.

### What should be done in the future

1. Implement Phase 1-8 tasks from `tasks.md`.
2. Add a follow-up ticket for registry visibility controls/redaction if private YAML registries remain exposed in CRUD.

### Code review instructions

1. Review implementation contract in:
   - `design-doc/01-implementation-guide-ordered-profile-registries-chain-and-single-registry-yaml-cutover.md`.
2. Validate scope completeness in:
   - `tasks.md`.
3. Cross-check assumptions with current code in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service.go`,
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/codec_yaml.go`,
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/profile_api.go`.

### Technical details

1. Ticket created via `docmgr ticket create-ticket --ticket GP-31-PROFILE-REGISTRIES-CHAIN ...`.
2. Documents created via `docmgr doc add` for `design-doc` and `reference`.
3. No code changes in this step; deliverable is planning and implementation guidance.

## Step 2: Scope correction to strict stack-only runtime model

This step applied a design correction to match the requested hard-cut model more tightly. The prior draft still assumed optional registry switching and YAML `default_profile_slug`; both were removed from runtime scope.

The updated model treats loaded registry sources as a pure precedence stack and resolves profiles from stack top to bottom without runtime registry switching inputs.

### Prompt Context

**User prompt (verbatim):** "we don't need a default_profile_slug in yaml files either, that's weird. we just have the stack of profile registries and the one on top is the one used. I don't think we need to switch registries, it's just a stack."

**Assistant interpretation:** remove default profile slug semantics from runtime YAML and remove runtime registry switching from the GP-31 design.

**Inferred user intent:** simplify GP-31 to a strict hard-cut stack model and avoid extra decision surface.

### What I did

1. Updated design doc semantics to:
   - stack-top-first profile precedence,
   - no runtime registry selector in this flow,
   - runtime YAML single-registry format with `default_profile_slug` rejected.
2. Updated tasks to reflect:
   - no runtime registry switching implementation,
   - explicit validation for rejecting `default_profile_slug`,
   - tests for stack-top-first behavior.
3. Added changelog entry documenting the scope correction.

### Why

1. The user requested a stricter model with fewer knobs.
2. Removing runtime registry switching keeps behavior deterministic and simpler to reason about.
3. Removing `default_profile_slug` from runtime YAML avoids carrying an unnecessary selector mechanism in this stack model.

### What worked

1. Scope correction is now consistent across design and tasks.
2. The revised contract remains compatible with existing stack merge/provenance internals.

### What didn't work

N/A

### What I learned

1. `registry` as a runtime selector and `default_profile_slug` as a runtime default are separate complexity multipliers; dropping both simplifies the contract significantly.

### What was tricky to build

The tricky part was ensuring all planning artifacts remained internally consistent after the scope correction:

1. design semantics,
2. task phases,
3. change history/diary rationale.

### What warrants a second pair of eyes

1. Whether any existing runtime API surfaces should still carry `registry_slug` for non-GP-31 flows, or be removed/ignored consistently.
2. Whether stack-top-first should be defined as “last entry wins” vs “first entry wins” in CLI docs/examples; current design uses last entry as top.

### What should be done in the future

1. Implement GP-31 with explicit tests proving:
   - no runtime registry switching,
   - `default_profile_slug` rejection for runtime YAML sources,
   - stack-top-first resolution determinism.

## Step 3: Final consistency pass for stack-only wording

This step tightened remaining wording so GP-31 does not imply any registry switching path and consistently frames the model as a single ordered registry stack.

### Prompt Context

**User prompt (verbatim):** "we don't need a default_profile_slug in yaml files either, that's weird. we just have the stack of profile registries and the one on top is the one used. I don't think we need to switch registries, it's just a stack."

**Assistant interpretation:** keep the existing hard-cut decisions and remove any ambiguous wording that might suggest optional registry selection.

**Inferred user intent:** finalize the contract language before implementation so there is no confusion during GP-31 coding.

### What I did

1. Re-checked all GP-31 ticket docs for mentions of runtime registry switching and `default_profile_slug`.
2. Updated `index.md` summary and overview text to explicitly state stack-top-first lookup and avoid selector-style phrasing.
3. Verified that remaining `--registry` mention is only in rejected alternatives, not in proposed behavior.

### Why

1. Small wording ambiguity at planning stage can become accidental behavior during implementation.
2. This clarification aligns all ticket artifacts with the intended hard cut contract.

### What worked

1. Design doc, tasks, changelog, and index are now aligned on stack-only semantics.

### What didn't work

N/A

### What I learned

1. Eliminating selector vocabulary from user-facing docs is as important as removing selector code paths.

### What should be done in the future

1. Keep implementation PR review checklist explicit: no runtime registry selector inputs, no runtime `default_profile_slug`, stack-top-first profile lookup only.

## Related

- `GP-28-STACK-PROFILES`
- `GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER`
