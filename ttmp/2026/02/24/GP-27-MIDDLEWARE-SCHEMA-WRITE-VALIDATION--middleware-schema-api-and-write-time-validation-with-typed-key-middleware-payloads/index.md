---
Title: Middleware schema API and write-time validation with typed-key middleware payloads
Ticket: GP-27-MIDDLEWARE-SCHEMA-WRITE-VALIDATION
Status: active
Topics:
    - architecture
    - backend
    - middleware
    - geppetto
    - pinocchio
    - go-go-os
    - glazed
    - migration
    - chat
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-24T23:38:00Z
WhatFor: Deliver strict write-time middleware validation and schema-discovery APIs, and move middleware config payloads to typed-key extensions for namespaced governance.
WhenToUse: Use when implementing or reviewing profile registry middleware parameter validation, schema APIs for frontend forms, and typed-key middleware payload migration.
---

# Middleware schema API and write-time validation with typed-key middleware payloads

## Overview

This ticket resolves GP-20 follow-up decisions:

- no registry-level extensions in this phase,
- middleware definitions/config must validate at profile write-time,
- unknown middleware names remain hard errors,
- hard cutover only: no migration command and no transitional fallback path,
- expose middleware and extension schemas through API for frontend tooling,
- move middleware config payloads into typed-key extensions (namespaced/versioned),
- keep provenance trace debug-only.

Current middleware CRUD model remains profile-scoped (via profile CRUD `runtime.middlewares`), not a separate middleware CRUD resource.

## Key Links

- Design: [Implementation plan middleware schema API write-time validation and typed-key middleware payloads](./design-doc/01-implementation-plan-middleware-schema-api-write-time-validation-and-typed-key-middleware-payloads.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)
- Diary: [reference/01-diary.md](./reference/01-diary.md)

## Status

Current status: **active**

## Topics

- architecture
- backend
- middleware
- geppetto
- pinocchio
- go-go-os
- glazed
- migration
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
