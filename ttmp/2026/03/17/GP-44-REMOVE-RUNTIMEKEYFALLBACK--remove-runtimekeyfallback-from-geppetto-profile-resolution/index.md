---
Title: Remove RuntimeKeyFallback from geppetto profile resolution
Ticket: GP-44-REMOVE-RUNTIMEKEYFALLBACK
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - cleanup
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go
      Note: Defines ResolveInput and ResolvedProfile public API surface.
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go
      Note: Implements ResolveEffectiveProfile and contains the only RuntimeKeyFallback behavior.
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go
      Note: JS profiles.resolve surface still accepts runtimeKeyFallback and runtimeKey.
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go
      Note: JS engines.fromProfile surface still accepts runtimeKey even though engine construction does not use it.
ExternalSources: []
Summary: Plan for removing RuntimeKeyFallback from geppetto profile resolution, including API impact, implementation sequencing, tests, docs cleanup, and downstream review guidance.
LastUpdated: 2026-03-17T16:50:00-04:00
WhatFor: Use this ticket to drive the removal of RuntimeKeyFallback from the Go resolver API, JS bindings, tests, docs, and examples without changing actual runtime selection behavior.
WhenToUse: Use when implementing or reviewing RuntimeKeyFallback removal, or when onboarding a new engineer to the geppetto profile resolver.
---

# Remove RuntimeKeyFallback from geppetto profile resolution

## Overview

This ticket captures a focused cleanup in the profile-resolution subsystem: remove `RuntimeKeyFallback` from the public resolver input and remove the corresponding JS-facing `runtimeKey` / `runtimeKeyFallback` options that only affect the reflected `ResolvedProfile.RuntimeKey` output. The analysis shows that this field does not participate in registry selection, profile lookup, stack expansion, runtime merge, step-settings resolution, runtime fingerprinting, or engine instantiation. It behaves as a detached label rather than a meaningful runtime-control mechanism.

The primary design document is the implementation guide in `design-doc/01-...md`. The diary records how the evidence was gathered and how the ticket was structured for follow-on implementation.

## Key Links

- Design doc: `design-doc/01-remove-runtimekeyfallback-from-geppetto-analysis-design-and-implementation-guide.md`
- Diary: `reference/01-investigation-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Current Status

Current status: **active**

The ticket is in design-ready state. The recommended next action is implementation in a small hard-cut cleanup change that touches resolver types, JS APIs, docs, examples, and tests.

## Scope

In scope:

- Remove `ResolveInput.RuntimeKeyFallback`.
- Remove `ResolvedProfile.RuntimeKey`.
- Remove JS `profiles.resolve({ runtimeKeyFallback, runtimeKey })`.
- Remove JS `engines.fromProfile(..., { runtimeKey })`.
- Update examples, type declarations, tests, and docs.

Out of scope:

- Reworking profile stack resolution.
- Changing runtime fingerprint behavior.
- Removing registry-stack resolution.
- Removing unrelated legacy cleanup candidates tracked in GP-45.

## Tasks

See [tasks.md](./tasks.md) for the detailed implementation checklist and validation steps.

## Changelog

See [changelog.md](./changelog.md) for the investigation record and documentation milestones.

## Structure

- `design-doc/`: primary analysis, architecture explanation, phased implementation guide
- `reference/`: diary and reviewer context
- `scripts/`: reserved for any future ticket-local helper scripts
- `archive/`: reserved for superseded drafts if the design changes
