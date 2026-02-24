---
Title: Profile Extensions and CRUD
Ticket: GP-22-PROFILE-EXTENSIONS-CRUD
Status: complete
Topics:
    - architecture
    - backend
    - geppetto
    - pinocchio
    - chat
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/types.go
      Note: Profile and registry model fields where extension payload support lives.
    - Path: geppetto/pkg/profiles/validation.go
      Note: Validation boundary for extension key/value shape and API error mapping.
    - Path: geppetto/pkg/profiles/service.go
      Note: Registry service orchestration for CRUD and effective runtime resolution.
    - Path: geppetto/pkg/profiles/codec_yaml.go
      Note: YAML round-trip behavior for unknown/known extension payloads.
    - Path: geppetto/pkg/profiles/sqlite_store.go
      Note: SQLite payload persistence for extension-enabled registries.
    - Path: pinocchio/pkg/webchat/http/profile_api.go
      Note: Shared profile CRUD handlers and DTO contract.
    - Path: go-go-os/packages/engine/src/chat/runtime/profileApi.ts
      Note: TypeScript client expected to consume stable CRUD response shapes.
ExternalSources: []
Summary: Detailed plan for adding typed profile extension payloads and hardening reusable profile CRUD APIs consumed by Pinocchio and Go-Go-OS.
LastUpdated: 2026-02-24T14:22:02.990352999-05:00
WhatFor: Define how extension metadata and CRUD APIs become stable shared contracts across backend and frontend clients.
WhenToUse: Use when implementing profile-extension model changes, profile CRUD handlers, and cross-app API reuse.
---


# Profile Extensions and CRUD

## Overview

This ticket implements the profile-registry product surface that users and applications interact with directly:

- typed, versioned extension payloads on profiles (and optionally registries), and
- reusable CRUD endpoints that expose those profiles consistently to both Pinocchio web-chat and Go-Go-OS.

The goal is to make profile data extensible without adding app-specific fields into geppetto core structs every time a new UI behavior is introduced (for example starter suggestions).

## Key Links

- Design: [Implementation Plan - Profile Extensions and CRUD](./design-doc/01-implementation-plan-profile-extensions-and-crud.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- backend
- geppetto
- pinocchio
- chat

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
