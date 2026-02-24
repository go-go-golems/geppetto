---
Title: Profile Registry Conversation Reference and FAQ
Ticket: GP-20-PROFILE-REGISTRY-EXTENSIONS
Status: active
Topics:
    - architecture
    - geppetto
    - pinocchio
    - chat
    - frontend
    - persistence
    - migration
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/registry.go
      Note: Registry interfaces discussed in the migration from app flags to profile-driven behavior.
    - Path: geppetto/pkg/profiles/service.go
      Note: Runtime resolution behavior and where profile selection semantics are enforced.
    - Path: pinocchio/pkg/webchat
      Note: Webchat surface where CRUD and profile selection behavior are consumed.
    - Path: go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go
      Note: Concrete seeding and app-level profile setup discussed during testing.
    - Path: geppetto/pkg/profiles/slugs.go
      Note: Typed slug types raised in discussion.
    - Path: geppetto/pkg/profiles/types.go
      Note: Core model limitations that motivated typed-key extension pattern.
ExternalSources: []
Summary: Consolidated write-up of the profile-registry migration discussion, covering key questions, decisions, tradeoffs, and implementation direction across geppetto, pinocchio, and go-go-os.
LastUpdated: 2026-02-24T23:40:00-05:00
WhatFor: Preserve decision context and practical answers to repeated questions about profile registries, CRUD routes, profile switching, migration, and extensibility.
WhenToUse: Use when onboarding contributors, reviewing migration rationale, or checking prior decisions before adding new profile-registry behavior.
---

# Profile Registry Conversation Reference and FAQ

## Goal

Capture the working conversation around profile registries so implementation decisions remain reproducible and discoverable.

This document is intentionally practical:

- what was asked,
- what was clarified,
- what was implemented,
- what design direction emerged for extensibility.

## Context

The project goal shifted from app-local profile config and binary flags toward registry-driven profile management:

- profile registries should be loadable from disk and database-backed;
- pinocchio web-chat and go-go-os should list/select/create profiles via reusable CRUD routes;
- middleware/AI-engine behavior should be profile-driven instead of env-flag-driven;
- compatibility shims should be reduced over time and replaced with explicit migration playbooks.

During implementation, several supporting threads emerged:

- slug custom types,
- symbol/API renames and temporary alias policy,
- removal of env-gated middleware toggle (`PINOCCHIO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE`),
- CRUD route reuse across applications,
- UI behavior bugs around profile selection,
- profile semantics (conversation-bound or switchable mid-conversation),
- long-term extensibility (starter suggestions and app-specific payloads).

## Quick Reference

### Core Decisions Snapshot

1. Profile registry CRUD should be reusable, not app-specific.
2. Runtime behavior should come from selected profile resolution, not environment toggles.
3. Default profile should be explicit and present in registries to avoid UI ambiguity.
4. Slug custom types are desirable and already align with current `RegistrySlug/ProfileSlug/RuntimeKey`.
5. Extension payloads should use typed keys (pattern 2) rather than repeatedly changing core structs.
6. Backward-compat aliases should eventually be removed; migrations should be documented.

### Key FAQ

**Q: Are CRUD routes app-specific or registry-specific?**  
A: They are registry-specific by design and can be mounted in different apps as shared handlers.

**Q: Can go-go-os reuse the same CRUD routes if moved/shared under webchat package boundaries?**  
A: Yes. That is the direction and was explicitly validated as part of planning.

**Q: Does conversation persist selected profile, or can profile change mid-conversation?**  
A: Current behavior allows selecting profile context before/while chatting depending on integration state; robust semantics should persist profile reference per conversation message stream to avoid silent runtime drift.

**Q: Why did UI show inventory and not switch to default properly?**  
A: The selector/value mapping and available profile defaults needed fixes. This was corrected by explicit slug handling and default labeling behavior.

**Q: Can we have a real "default" profile with no middlewares?**  
A: Yes. Seeding now supports explicit `default` profile separate from inventory-focused profiles.

**Q: Can we add starter suggestions per profile?**  
A: Yes, but cleanly this should be extension-backed (pattern 2), not hard-coded in core.

**Q: Is profile registry internal to go-go-os?**  
A: No. It is a Geppetto-level concept intended for reuse by pinocchio, go-go-os, and third parties.

### Operational API Snippets

List profiles in registry:

```http
GET /api/chat/profiles?registry=default
```

Set default profile:

```http
PUT /api/chat/profiles/default?registry=default
Content-Type: application/json

{"profile_slug":"inventory"}
```

Create profile:

```http
POST /api/chat/profiles?registry=default
Content-Type: application/json

{
  "slug":"analyst",
  "display_name":"Analyst",
  "runtime":{"system_prompt":"..."},
  "policy":{"allow_overrides":true}
}
```

Update profile:

```http
PATCH /api/chat/profiles/analyst?registry=default
Content-Type: application/json

{"description":"Analytical workflow profile"}
```

Delete profile:

```http
DELETE /api/chat/profiles/analyst?registry=default
```

## Conversation Timeline (Condensed)

### 1) Foundational Direction

- Requested creation of profile-registry architecture ticket and broad analysis across geppetto/pinocchio/go-go-os.
- Emphasis: remove config flags in binaries and use registries as reusable objects.

### 2) Granularization and Naming

- Requested granular task breakdown.
- Asked about custom slug types similar to Glazed patterns.
- Requested cleanup of confusing symbols and naming proposals.

### 3) Compatibility and Migration Pressure

- Requested removal of env var gating for middleware integration.
- Asked for explicit explanation of compatibility behaviors.
- Asked where aliases should remain temporarily and where to remove them.

### 4) Documentation and Playbooks

- Requested migration playbooks in package docs (Glazed help page style).
- Requested legacy `profiles.yaml` migration support and CLI affordances.

### 5) CRUD and Reuse

- Confirmed CRUD endpoints should be reusable and non app-specific.
- Requested integration path to reuse routes across pinocchio and go-go-os.

### 6) Runtime and UX Semantics

- Asked if user can test profile selection now.
- Reported selector behavior issue (state updated, UI remained on inventory).
- Requested real default profile and extra seeded profiles.

### 7) Extension and Long-Term Architecture

- Asked about profile-specific starter suggestions.
- Asked if that implies modifying geppetto.
- Asked for a model where users can extend profile data without editing core package.
- This yielded pattern 2: typed-key extensible profile payloads.

## Fundamentals and Foundations

> Foundation: Profiles are runtime contracts, not just labels.

A profile should be treated as a durable runtime contract with:

- identity (`slug`),
- runtime behavior (`runtime`),
- mutability policy (`policy`),
- provenance/version (`metadata`),
- extension payloads (pattern 2 target).

> Foundation: Registry is the source of truth.

The application binary should not need to encode profile-specific feature matrices as flags. It should load registry data from stores (YAML/SQLite/API) and execute behavior from resolved profile state.

> Foundation: Compatibility is data-centric, not flag-centric.

Instead of keeping long-lived env toggles, preserve compatibility through:

- schema-compatible storage,
- explicit migration docs/tools,
- controlled deprecation windows.

## Implementation Notes from Discussion

- Typed slug approach is good and already part of current model for registry/profile/runtime identifiers.
- Selector UX should never depend on implicit blank default behavior; explicit default profile and visible labeling are required.
- For third-party-facing packages, temporary aliases can help transition, but remove once migration playbook is published and downstreams have upgrade path.
- CRUD route packages should be organized by reuse boundaries (shared package first, app mount second).
- TS/react baseline errors in monorepo can mask profile work; fix baseline quickly to keep momentum.

## Usage Examples

### Example: Extension-Driven Starter Suggestions (Target Pattern)

```yaml
slug: default
profiles:
  analyst:
    slug: analyst
    display_name: Analyst
    runtime:
      system_prompt: "You are an analytical assistant."
    extensions:
      webchat.starter_suggestions@v1:
        suggestions:
          - "Summarize last week performance"
          - "Highlight anomalies by category"
```

Consumer behavior:

1. Webchat fetches profile list/details.
2. Webchat reads `webchat.starter_suggestions@v1` when present.
3. UI renders suggestion chips for new conversation start.

### Example: Conversation Profile Binding Policy

Recommended policy pseudocode:

```pseudo
on conversation_start(profile_slug):
  conversation.profile_slug = profile_slug

on message_send(conversation_id):
  profile = registry.get(conversation.profile_slug)
  runtime = resolve(profile)
  execute(runtime)

on profile_switch_requested(conversation_id, new_slug):
  if policy.allow_midstream_switch:
    conversation.profile_slug = new_slug
    append_event("profile_switched", old, new)
  else:
    reject_with_guidance("Start a new conversation")
```

## Related

- Design doc: `design-doc/01-extensible-profile-metadata-via-typed-keys-architecture-and-implementation-plan.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->
