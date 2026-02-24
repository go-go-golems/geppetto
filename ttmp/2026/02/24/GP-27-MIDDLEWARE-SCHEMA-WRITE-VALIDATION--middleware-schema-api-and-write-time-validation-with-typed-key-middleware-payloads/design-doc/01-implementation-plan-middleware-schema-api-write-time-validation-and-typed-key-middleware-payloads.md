---
Title: Implementation plan middleware schema API write-time validation and typed-key middleware payloads
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-24T16:21:25.826734063-05:00
WhatFor: Specify strict profile-write middleware validation, schema-discovery APIs, and typed-key middleware payload storage with hard-error semantics.
WhenToUse: Use when implementing middleware schema contracts across profile CRUD, runtime composers, and frontend schema consumers.
---

# Implementation plan middleware schema API write-time validation and typed-key middleware payloads

## Executive Summary

Today middleware config validation is mostly deferred to runtime composition. That allows invalid middleware payloads to be persisted and fail only when a profile is used. This ticket moves middleware validation to write-time in profile CRUD, adds schema discovery APIs, and standardizes middleware config payloads under typed-key extensions to enforce namespaced/versioned ownership.

Key decisions captured from GP-20 follow-up:

- keep profile-level extensions only (no registry-level extensions),
- validate middleware references and config at profile write-time,
- keep unknown middleware behavior as hard error,
- expose middleware and extension schemas via API,
- use typed-key extension payloads for middleware config,
- keep parse/provenance traces debug-only.

## Problem Statement

Current gaps:

1. invalid middleware config can be stored and only fails later at compose-time;
2. frontend cannot discover middleware/extension schemas from backend API;
3. middleware config in `runtime.middlewares[].config` is unnamespaced and not versioned;
4. no single write-path contract enforces middleware existence + config validity before persistence.

## Proposed Solution

### 1. Write-Time Middleware Validation in Profile CRUD

During `CreateProfile` / `UpdateProfile`:

1. for each `runtime.middlewares[]` entry, resolve middleware definition by name;
2. if definition is missing, return `ErrValidation` (hard error);
3. resolve+validate config against definition JSON Schema;
4. persist canonical/validated payload only.

Validation will use existing `middlewarecfg.Resolver` with source layers:

- schema defaults,
- profile write payload.

No request-layer overrides at write-time.

### 2. Typed-Key Middleware Config Payloads

Introduce canonical typed-key extension payload for middleware config, e.g.:

- `middleware.agentmode_config@v1`
- `middleware.sqlite_config@v1`

`runtime.middlewares[]` remains the enable/order/identity list (`name`, `id`, `enabled`), while config payloads move to `profile.extensions`.

Composer resolution becomes:

1. select middleware instances from `runtime.middlewares`;
2. load config payload from typed-key extension map;
3. validate against definition schema;
4. build chain.

### 3. Schema Discovery API

Add read-only API endpoints:

- `GET /api/chat/schemas/middlewares`
- `GET /api/chat/schemas/extensions`

Each response includes key/name, version, JSON Schema, and optional UI hints.

### 4. Middleware CRUD Scope

No separate middleware CRUD endpoints are introduced. Middleware lifecycle remains profile CRUD:

- create/update profile with `runtime.middlewares` + `extensions`,
- set default profile,
- delete profile.

## Design Decisions

1. Unknown middleware names are hard validation errors.
2. Registry-level extensions remain out of scope in this ticket.
3. Schema APIs are read-only discovery surfaces, not mutation endpoints.
4. Trace/provenance output stays debug-only and not part of standard profile API contracts.
5. Middleware config under typed-key extensions is the long-term canonical model.

## Alternatives Considered

### A) Keep compose-time-only validation

Rejected because invalid profiles persist and fail late.

### B) Add dedicated middleware CRUD resources

Rejected for now because middleware config is profile-scoped by design and should stay within profile CRUD transaction boundaries.

### C) Keep `runtime.middlewares[].config` as canonical

Rejected because it lacks namespaced/versioned ownership and weakens extension-key governance.

## Implementation Plan

1. Add middleware write-validator service that depends on definition registry.
2. Integrate validator into profile create/update service flow.
3. Define typed-key convention for middleware config extension payloads.
4. Refactor composers to read middleware config from typed-key extensions.
5. Add strict failure when middleware definition is missing.
6. Add schema API surfaces for middleware and extension schemas.
7. Add API contract tests and frontend decoder tests.
8. Update docs/help pages.

## Open Questions

1. Exact schema endpoint payload shape for frontend form generation metadata (`title`, `description`, ui hints).
2. Whether middleware instance ID should be encoded into typed-key names or keyed in payload structure.

## References

- `geppetto/pkg/profiles/service.go`
- `geppetto/pkg/profiles/types.go`
- `geppetto/pkg/inference/middlewarecfg/resolver.go`
- `pinocchio/pkg/webchat/http/profile_api.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`
