---
Title: Profile Registry Core
Ticket: GP-21-PROFILE-REGISTRY-CORE
Status: complete
Topics:
    - architecture
    - backend
    - geppetto
    - persistence
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/types.go
      Note: Core profile and registry domain model.
    - Path: geppetto/pkg/profiles/validation.go
      Note: Registry/profile invariant validation and error semantics.
    - Path: geppetto/pkg/profiles/service.go
      Note: StoreRegistry orchestration and effective runtime resolution.
    - Path: geppetto/pkg/profiles/store.go
      Note: Profile store interfaces and save options.
    - Path: geppetto/pkg/profiles/file_store_yaml.go
      Note: YAML-backed store behavior and persistence flow.
    - Path: geppetto/pkg/profiles/sqlite_store.go
      Note: SQLite-backed durable persistence flow.
ExternalSources: []
Summary: Foundation ticket for hardening profile registry core model, invariants, and storage behavior prior to extension and middleware cutovers.
LastUpdated: 2026-02-24T13:43:05.150546902-05:00
WhatFor: Provide a stable and well-tested profile registry substrate used by extension/CRUD/middleware tickets.
WhenToUse: Use when implementing or reviewing model, validation, and persistence behavior for profile registries.
---


# Profile Registry Core

## Overview

This ticket defines the base work required to make profile registries a reliable source of truth across Geppetto, Pinocchio, and Go-Go-OS. The focus is core correctness: strong model contracts, deterministic validation behavior, persistence safety, and explicit default-selection semantics.

This ticket intentionally excludes profile-extension payloads and middleware schema/provenance machinery. Those are handled in downstream tickets.

## Key Links

- Design: [Implementation Plan - Profile Registry Core](./design-doc/01-implementation-plan-profile-registry-core.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- backend
- geppetto
- persistence
- migration

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
