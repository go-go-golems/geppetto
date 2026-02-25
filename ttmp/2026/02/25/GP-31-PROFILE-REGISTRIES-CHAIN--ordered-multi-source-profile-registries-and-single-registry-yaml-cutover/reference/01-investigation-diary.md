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
LastUpdated: 2026-02-25T16:24:30-05:00
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

## Related

- `GP-28-STACK-PROFILES`
- `GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER`
