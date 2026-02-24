---
Title: Implementation Plan - Profile Extensions and CRUD
Ticket: GP-22-PROFILE-EXTENSIONS-CRUD
Status: active
Topics:
    - architecture
    - backend
    - geppetto
    - pinocchio
    - chat
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/types.go
      Note: Profile model extension fields and slug/type boundaries.
    - Path: geppetto/pkg/profiles/validation.go
      Note: Validation boundaries and field-path error reporting.
    - Path: geppetto/pkg/profiles/service.go
      Note: CRUD orchestration, write options, and conflict handling.
    - Path: geppetto/pkg/profiles/codec_yaml.go
      Note: Canonical YAML shape and backward parse support.
    - Path: geppetto/pkg/profiles/sqlite_store.go
      Note: Durable persistence and read-modify-write behavior.
    - Path: pinocchio/pkg/webchat/http/profile_api.go
      Note: Shared REST handlers for list/get/create/update/delete/default/select.
    - Path: pinocchio/pkg/webchat/router.go
      Note: Route mount integration into webchat server.
    - Path: go-go-os/packages/engine/src/chat/runtime/profileApi.ts
      Note: Runtime client for CRUD calls and response decoding.
ExternalSources: []
Summary: End-to-end implementation plan for typed extension payload support and reusable profile registry CRUD endpoints across Pinocchio and Go-Go-OS.
LastUpdated: 2026-02-24T13:12:02-05:00
WhatFor: Define the technical contract and phased rollout for extension metadata plus API-level profile lifecycle management.
WhenToUse: Use when implementing profile extension keys/codecs, REST contracts, and frontend profile-management integration.
---

# Implementation Plan - Profile Extensions and CRUD

## Executive Summary

This ticket delivers the user-facing profile platform:

1. extension payloads that allow app-specific profile metadata without modifying geppetto core structs repeatedly,
2. reusable CRUD endpoints exposed from shared webchat package code,
3. a stable API contract used by both Pinocchio and Go-Go-OS.

This work depends on GP-21 for core model correctness and storage guarantees. It intentionally avoids middleware-schema unification internals (GP-23) and runtime cutover wiring (GP-24), except where API design must anticipate those tickets.

## Problem Statement

Current profile data model covers runtime defaults and policy but lacks a standardized way to attach app-specific typed metadata. As a result, every app feature risks one of two failures:

- hardcoding app-specific fields directly in shared profile structs, or
- shipping opaque `map[string]any` blobs with no stable contract and no typed decode path.

At the same time, profile CRUD behavior exists but must be treated as a shared contract rather than app-private handler logic. Frontends need predictable response shape, clear defaults, and stable error semantics so profile selection/management UI works consistently.

Without this ticket:

- starter suggestions and other profile-scoped UX metadata remain ad-hoc,
- third-party clients cannot rely on one API shape,
- Go-Go-OS and Pinocchio risk diverging profile behavior over time.

## Goals and Non-Goals

Goals:

- add extension payload storage in profile model with type-safe access pattern,
- preserve unknown extension payloads end-to-end for forward compatibility,
- provide reusable CRUD routes under shared `pkg/webchat/http`,
- guarantee list/get/create/update/delete/set-default/select semantics and status codes,
- keep contract stable for TypeScript clients.

Non-goals:

- middleware parameter schema/provenance engine (GP-23),
- full runtime composer cutover in every app entry point (GP-24),
- release/migration process docs (GP-25).

## Proposed Solution

### 1. Extension Data Model

Add extension maps to `Profile` (and optionally `ProfileRegistry` if needed by consumers):

```go
type ExtensionMap map[string]any

type Profile struct {
    Slug        ProfileSlug      `json:"slug" yaml:"slug"`
    DisplayName string           `json:"display_name,omitempty" yaml:"display_name,omitempty"`
    Description string           `json:"description,omitempty" yaml:"description,omitempty"`
    Runtime     RuntimeSpec      `json:"runtime,omitempty" yaml:"runtime,omitempty"`
    Policy      PolicySpec       `json:"policy,omitempty" yaml:"policy,omitempty"`
    Metadata    ProfileMetadata  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
    Extensions  ExtensionMap     `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}
```

Introduce typed key helpers so extension payload users do not parse raw maps everywhere:

```go
type ExtensionKey[T any] struct {
    Key string // e.g. "webchat.starter_suggestions@v1"
}

func (k ExtensionKey[T]) Get(p *Profile) (T, bool, error)
func (k ExtensionKey[T]) Set(p *Profile, value T) error
```

### 2. Extension Codec Registry

Define codec interface for validating/normalizing payloads by extension key:

```go
type ExtensionCodec interface {
    Key() string
    Decode(raw any) (any, error)
    Encode(v any) (any, error)
}
```

Behavior:

- known key with invalid payload -> validation error,
- known key with valid payload -> normalized payload stored,
- unknown key -> preserved without loss (default permissive mode).

### 3. CRUD API Contract (Reusable)

Shared handlers remain in `pinocchio/pkg/webchat/http/profile_api.go` and are mounted by both apps.

Required endpoints:

- `GET /api/chat/profiles?registry=<slug>` list profiles,
- `POST /api/chat/profiles` create profile,
- `GET /api/chat/profiles/{slug}` fetch profile,
- `PATCH /api/chat/profiles/{slug}` partial update,
- `DELETE /api/chat/profiles/{slug}` delete profile,
- `POST /api/chat/profiles/default` set default profile,
- `GET /api/chat/profile` get currently selected profile,
- `POST /api/chat/profile` set currently selected profile.

Contract requirements:

- list response is an array, never map-index object,
- extension payload is present on get/list/create/update responses,
- clear 400/404/409/422 mapping for slug/validation/conflict/policy cases.

### 4. Store and Round-Trip Guarantees

Both YAML and SQLite stores must round-trip extension payloads losslessly, including unknown keys.

The persistence invariant:

```text
read -> mutate unrelated field -> write -> read
```

must never delete unknown extension entries.

### 5. Frontend Consumer Expectations

TypeScript client and state slices should use explicit DTO types:

```ts
type ProfileDocument = {
  registry: string;
  slug: string;
  display_name?: string;
  description?: string;
  runtime?: RuntimeSpec;
  policy?: PolicySpec;
  metadata?: ProfileMetadata;
  extensions?: Record<string, unknown>;
  is_default?: boolean;
};
```

This enables UI features like starter suggestions without app-side forked endpoints.

## Design Decisions

1. Extension keys are versioned strings (`namespace.feature@vN`) to support schema evolution.
2. Unknown extensions are preserved by default to avoid data loss across mixed-version deployments.
3. Validation/normalization lives at service boundary, not only in API handlers.
4. CRUD handlers stay in shared package and are mounted by applications as-is.
5. Response shape is normalized around arrays and explicit objects (no map-index JSON artifacts).
6. Registry-level `extensions` is deferred in GP-22; this ticket standardizes only profile-level `extensions` until a concrete registry-scope use case is implemented.

## Alternatives Considered

### A. Add dedicated first-class fields for every new app feature

Rejected: quickly bloats shared structs and forces geppetto release for every app-level metadata change.

### B. Store extensions only as opaque blobs with no typed helpers

Rejected: pushes repeated decode/validation logic into every consumer and increases runtime type errors.

### C. Keep CRUD handlers app-private per binary

Rejected: creates drift between Pinocchio and Go-Go-OS, breaks third-party expectations.

## Implementation Plan

### Phase A - Model and Validation

1. Add extension map fields and deep-copy behavior to clone methods.
2. Add extension key parser/constructor with slug-like validation rules.
3. Add validation for extension-key syntax and payload serializability.

### Phase B - Codec Registry

1. Add extension codec registry interfaces and in-memory implementation.
2. Wire codec normalization into create/update service paths.
3. Add permissive unknown-key pass-through and strict mode for tests.

### Phase C - CRUD Contract Hardening

1. Audit all profile API responses for consistent DTO shapes.
2. Ensure list endpoint emits arrays deterministically sorted by slug.
3. Add extension fields to request/response DTOs for create/update/get/list.
4. Harden status-code mapping by error category.

### Phase D - Reuse Across Apps

1. Ensure Pinocchio web-chat mounts shared handlers only.
2. Ensure Go-Go-OS mounts the same shared handlers with app-specific options only.
3. Keep profile-selection cookie and current-profile behavior consistent.

### Phase E - Test Matrix

1. Unit tests for key parsing, codec decode, clone isolation.
2. Store tests for YAML and SQLite extension round-trip.
3. API contract tests for CRUD shape/status semantics.
4. TS client tests for list/get/update response decoding.

### Phase F - Documentation Hooks

1. Add API examples with extensions in docs.
2. Add extension key naming conventions and versioning rules.
3. Add starter-suggestions extension example as reference payload.

## Open Questions

1. Do we require extension codec registration at startup (strict) or allow lazy registration?
2. Should strict unknown-key rejection be configurable per API request for admin tooling?

## References

- `geppetto/pkg/profiles/types.go`
- `geppetto/pkg/profiles/validation.go`
- `geppetto/pkg/profiles/service.go`
- `pinocchio/pkg/webchat/http/profile_api.go`
- `go-go-os/packages/engine/src/chat/runtime/profileApi.ts`
