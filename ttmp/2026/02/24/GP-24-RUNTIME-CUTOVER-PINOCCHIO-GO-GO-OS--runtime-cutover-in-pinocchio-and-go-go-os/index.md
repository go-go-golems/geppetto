---
Title: Runtime Cutover in Pinocchio and Go-Go-OS
Ticket: GP-24-RUNTIME-CUTOVER-PINOCCHIO-GO-GO-OS
Status: active
Topics:
    - architecture
    - backend
    - pinocchio
    - chat
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Primary Pinocchio web-chat runtime bootstrap and profile registry wiring.
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Request-scoped runtime composition from resolved profile runtime.
    - Path: pinocchio/pkg/webchat/router.go
      Note: Shared webchat server route/middleware/tools registration and composition hooks.
    - Path: pinocchio/pkg/webchat/http/profile_api.go
      Note: Reusable CRUD and current-profile endpoints to mount in both apps.
    - Path: go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go
      Note: Inventory server bootstrap and route mounting.
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go
      Note: Inventory runtime composition path consuming profile runtime.
    - Path: go-go-os/packages/engine/src/chat/runtime/useSetProfile.ts
      Note: Frontend profile selection behavior tied to current-profile endpoints.
ExternalSources: []
Summary: Application cutover plan for adopting shared profile registry CRUD routes and runtime composition semantics in Pinocchio and Go-Go-OS.
LastUpdated: 2026-02-24T13:12:02-05:00
WhatFor: Coordinate the hard cutover from app-specific profile behavior to shared registry-driven runtime behavior.
WhenToUse: Use when wiring profile APIs/runtime composers in application entry points and validating end-to-end behavior.
---

# Runtime Cutover in Pinocchio and Go-Go-OS

## Overview

This ticket applies the platform changes to running applications. Core infrastructure and schemas are not enough until both binaries consume them the same way.

Focus areas:

- mount shared CRUD routes in both servers,
- remove app-local fallback/compat logic,
- ensure runtime composers apply resolved profile runtime deterministically,
- guarantee frontend profile selection actually affects conversation runtime.

## Key Links

- Design: [Implementation Plan - Runtime Cutover in Pinocchio and Go-Go-OS](./design-doc/01-implementation-plan-runtime-cutover-in-pinocchio-and-go-go-os.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- backend
- pinocchio
- chat
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
