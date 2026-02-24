---
Title: Middleware JSON Schema and ParseStep Resolver
Ticket: GP-23-MIDDLEWARE-JSONSCHEMA-PARSESTEP
Status: complete
Topics:
    - architecture
    - backend
    - geppetto
    - pinocchio
    - middleware
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/middleware/middleware.go
      Note: Runtime middleware interface that remains unchanged.
    - Path: geppetto/pkg/profiles/runtime_settings_patch_resolver.go
      Note: Runtime step settings patch merge/apply resolver after symbol rename
    - Path: geppetto/pkg/profiles/types.go
      Note: '`MiddlewareUse` model where instance id/enabled/config are stored.'
    - Path: geppetto/pkg/sections/sections.go
      Note: Current layered source semantics used as precedence baseline.
    - Path: go-go-os/go-inventory-chat/internal/pinoweb/runtime_composer.go
      Note: App runtime composition path to migrate onto shared resolver.
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Current ad-hoc middleware override parsing path to remove.
ExternalSources: []
Summary: JSON-schema-first middleware configuration plan with ParseStep-style provenance tracking and hard cutover from ad-hoc config parsing.
LastUpdated: 2026-02-24T15:05:20.383372857-05:00
WhatFor: Define canonical middleware parameter contracts and source-layer provenance for runtime composition.
WhenToUse: Use when implementing middleware config parsing, validation, introspection, and runtime build wiring.
---



# Middleware JSON Schema and ParseStep Resolver

## Overview

This ticket establishes the canonical middleware configuration engine:

- JSON Schema defines middleware parameter contracts.
- A resolver applies layered sources with deterministic precedence.
- Every resolved value keeps provenance steps equivalent to Glazed parse logs.

This is the contract that allows profile-scoped middleware defaults, request overrides, CLI/config inputs, and frontend forms to converge on one model.

## Key Links

- Design: [Implementation Plan - Middleware JSON Schema and ParseStep Resolver](./design-doc/01-implementation-plan-middleware-json-schema-and-parsestep-resolver.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- backend
- geppetto
- pinocchio
- middleware

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
