---
Title: Implementation Plan - Runtime Cutover in Pinocchio and Go-Go-OS
Ticket: GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS
Status: active
Topics:
    - architecture
    - backend
    - pinocchio
    - chat
    - migration
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Pinocchio runtime bootstrap and profile service initialization.
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Pinocchio profile runtime composition path.
    - Path: pinocchio/pkg/webchat/server.go
      Note: Shared server composition and route registration lifecycle.
    - Path: pinocchio/pkg/webchat/http/profile_api.go
      Note: Shared CRUD/current-profile handler registration.
    - Path: go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go
      Note: Inventory app mount points and server integration.
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go
      Note: Inventory profile selection resolution path.
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go
      Note: Inventory runtime composer and middleware chain building.
    - Path: go-go-os/packages/engine/src/chat/state/profileSlice.ts
      Note: Frontend profile state transitions and selected profile behavior.
ExternalSources: []
Summary: Detailed cutover plan for mounting reusable profile CRUD routes and aligning runtime composition/profile selection semantics across Pinocchio and Go-Go-OS.
LastUpdated: 2026-02-24T13:12:02-05:00
WhatFor: Provide a concrete rollout and validation sequence to move both apps onto shared profile-registry behavior.
WhenToUse: Use when implementing and validating app-level runtime integration after core profile/middleware tickets are in place.
---

# Implementation Plan - Runtime Cutover in Pinocchio and Go-Go-OS

## Executive Summary

This ticket is the production cutover layer. It ensures both runtime hosts consume profile-registry APIs and runtime composition consistently.

Primary deliverables:

1. shared CRUD route registration in both server binaries,
2. runtime composer behavior parity for profile/middleware/tool application,
3. removal of backwards compatibility toggles and environment fallbacks,
4. end-to-end frontend verification that profile switching is effective.

## Problem Statement

Infrastructure work can still fail users if app entrypoints diverge:

- one app mounts full CRUD and another only list/select,
- one runtime composer respects profile middleware config while another hardcodes middleware chain,
- frontend state updates selected profile but server conversation runtime does not switch accordingly.

These gaps already appeared in practice (selection changes in UI with no runtime effect). This ticket eliminates those inconsistencies by making app-level integration explicit, testable, and shared where possible.

## Scope and Cutover Policy

Scope:

- Pinocchio `cmd/web-chat` and Go-Go-OS inventory server integration,
- reusable CRUD route mounting from shared package,
- runtime composition parity and profile-selection flow.

Cutover policy:

- hard cutover only,
- no compatibility env vars,
- no dual middleware composer branches once new path is active.

## Proposed Solution

### 1. Reusable CRUD Routes as Shared Primitive

Treat `pinocchio/pkg/webchat/http/profile_api.go` as canonical route module. Both apps call the same registration helper:

```go
webchathttp.RegisterProfileAPIHandlers(mux, profileRegistry, opts)
```

Only app-specific options differ (default registry slug, cookie name/base path if required).

### 2. Runtime Composer Alignment

Both apps must compose engines from resolved profile runtime data:

- requested/selected profile slug resolution,
- profile runtime resolution including middleware/tool defaults,
- deterministic chain build + engine rebuild when profile version changes.

Pseudo:

```pseudo
request -> resolve registry/profile
       -> fetch effective profile runtime
       -> compose runtime engine from resolved middleware+tools
       -> attach engine to conversation context
```

### 3. Profile Selection Semantics

Selection lifecycle:

1. user sets current profile via `POST /api/chat/profile`,
2. server stores selected profile cookie/state,
3. next conversation creation or explicit switch resolves selected profile,
4. conversation runtime carries selected profile + profile version.

Decisions:

- profile can be changed for new conversations immediately,
- existing conversation behavior must be explicit: either lock on creation or allow switch with rebuild trigger; document and test selected policy.

### 4. Route Surface Parity

Both apps expose the same route set and status behaviors:

- list/get/create/update/delete/default/current-profile set/get.

Any app-specific route prefixing should be mechanically applied, not endpoint-by-endpoint rewrites.

### 5. Removal of Compatibility Surfaces

Remove or ignore legacy compatibility branches:

- no env toggle for old middleware integration,
- no legacy fallback route variants,
- no app-local alternative profile DTOs.

## Design Decisions

1. Shared CRUD route code stays in package layer, not copied into each app.
2. Runtime composer is request-scoped and profile-driven in both apps.
3. Hard cutover removes compatibility codepaths once migrated.
4. Profile selection behavior is explicitly documented and test-covered.
5. Frontend APIs target shared route contract only.
6. In-flight conversation policy: when selected profile changes and the same `conv_id` is reused, runtime is rebuilt if runtime fingerprint changes; this enables explicit mid-conversation profile switching while keeping behavior deterministic.

## Alternatives Considered

### A. Keep separate app-local CRUD handlers

Rejected: duplicated logic and contract drift.

### B. Keep old and new runtime composition paths behind env flags

Rejected: prolongs ambiguity and doubles test surface.

### C. Defer Go-Go-OS integration until later

Rejected: creates immediate divergence and breaks confidence in shared platform claims.

## Implementation Plan

### Phase A - Shared Route Adoption

1. Verify shared profile API handler module has all CRUD/current-profile endpoints.
2. Mount those handlers in Pinocchio server startup path.
3. Mount same handlers in Go-Go-OS inventory server startup path.
4. Add route smoke tests in both apps.

### Phase B - Runtime Composer Parity

1. Refactor Pinocchio runtime composer to consume resolved profile runtime only.
2. Refactor Go-Go-OS runtime composer to consume same resolved profile runtime semantics.
3. Remove hardcoded middleware/tool defaults in app layer where profile runtime should decide.
4. Add parity tests with identical profile inputs across both apps.

### Phase C - Profile Selection Behavior

1. Validate `POST /api/chat/profile` updates selected profile state.
2. Ensure selected profile is applied when creating or rebuilding conversation runtime.
3. Add tests for changing selected profile from A to B and verifying effective runtime shift.
4. Decide and document behavior for existing in-progress conversations.

### Phase D - Frontend Wiring

1. Ensure frontend uses shared endpoint shapes and registry query semantics.
2. Fix selection UX so control reflects server-confirmed selected profile.
3. Add integration tests for profile selection + first message runtime response.

### Phase E - Hard Cutover Cleanup

1. Remove obsolete compatibility env vars and toggles.
2. Remove dead app-local adapters replaced by shared handlers.
3. Update startup logs/help text to reflect final configuration surfaces.

## Open Questions

1. Should active conversations lock profile at creation or support mid-conversation profile switch with explicit endpoint?
2. Do we need profile selection persistence beyond cookie scope (e.g., user/session DB)?
3. Should Go-Go-OS expose all CRUD verbs in production UI initially or gate create/update/delete behind feature flags?

## References

- `pinocchio/pkg/webchat/http/profile_api.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`
- `go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go`
- `go-go-os/packages/engine/src/chat/runtime/profileApi.ts`
