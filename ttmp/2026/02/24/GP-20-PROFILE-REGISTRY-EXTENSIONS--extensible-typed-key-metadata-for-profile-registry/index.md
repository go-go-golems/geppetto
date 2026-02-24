---
Title: Extensible Typed-Key Metadata for Profile Registry
Ticket: GP-20-PROFILE-REGISTRY-EXTENSIONS
Status: complete
Topics:
    - architecture
    - geppetto
    - pinocchio
    - chat
    - frontend
    - persistence
    - migration
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/types.go
      Note: Core profile and registry domain model where extension payload support is added.
    - Path: geppetto/pkg/profiles/validation.go
      Note: Validation invariants that must include typed-key extension payload checks.
    - Path: geppetto/pkg/profiles/service.go
      Note: Runtime resolution path that can consume typed extensions without app-specific flags.
    - Path: geppetto/pkg/turns/key_families.go
      Note: Existing typed-key pattern reference used to design profile extension key APIs.
    - Path: geppetto/pkg/turns/types.go
      Note: Canonical key encoding pattern (namespace.value@vN) reused for profile extension keys.
    - Path: pinocchio/pkg/webchat
      Note: Reusable webchat integration surface that should consume profile extensions.
    - Path: go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go
      Note: Concrete app integration point where registry-backed behavior is configured.
ExternalSources:
    - local:middleware-config-proposals.md
Summary: Design and planning ticket for pattern 2, introducing typed-key extension payloads on profile registries so apps can add profile-specific behavior without forking core Geppetto structs or adding new binary flags.
LastUpdated: 2026-02-24T13:20:41.219931475-05:00
WhatFor: Capture the architecture, migration strategy, and concrete implementation tasks for extensible profile metadata powered by typed keys and reusable registry-driven APIs.
WhenToUse: Use when implementing or reviewing extensible profile capabilities across geppetto, pinocchio, and go-go-os, especially when app-specific data should be registry-backed rather than hard-coded flags.
---



# Extensible Typed-Key Metadata for Profile Registry

## Overview

This ticket defines and decomposes implementation of "pattern 2": typed-key extension payloads on top of the existing `profiles.Profile` and `profiles.ProfileRegistry` model.  
The goal is to let applications add profile-specific data and behavior declaratively through registries (YAML/SQLite/API), while keeping Geppetto core stable and avoiding app-specific feature flags in binaries.

Current artifacts in this ticket include:

- a deep design document that proposes the extension architecture, APIs, migration rules, and rollout plan;
- a conversation reference document that records the decisions and Q&A made while introducing profile registries and CRUD across geppetto/pinocchio/go-go-os;
- a granular task list suitable for iterative implementation.

## Key Links

- Design: [Extensible Profile Metadata via Typed Keys: Architecture and Implementation Plan](./design-doc/01-extensible-profile-metadata-via-typed-keys-architecture-and-implementation-plan.md)
- Design: [Middleware Configuration Registry Unification with Profile-Scoped Defaults](./design-doc/02-middleware-configuration-registry-unification-with-profile-scoped-defaults.md)
- Design: [JSON Schema First with Glazed-Style Value Provenance](./design-doc/03-json-schema-first-with-glazed-style-value-provenance.md)
- Reference: [Profile Registry Conversation Reference and FAQ](./reference/01-profile-registry-conversation-reference-and-faq.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **complete**

## Follow-Up Resolution

Implementation work split from this umbrella is now covered by GP-21 through GP-27.
GP-27 finalizes middleware schema API and write-time validation decisions tracked here.

## Topics

- architecture
- geppetto
- pinocchio
- chat
- frontend
- persistence
- migration
- backend

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
