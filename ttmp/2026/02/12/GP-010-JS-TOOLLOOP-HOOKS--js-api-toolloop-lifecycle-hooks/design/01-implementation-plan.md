---
Title: Implementation Plan
Ticket: GP-010-JS-TOOLLOOP-HOOKS
Status: active
Topics:
    - geppetto
    - goja
    - javascript
    - tools
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Plan for toolloop lifecycle hook APIs from JS
LastUpdated: 2026-02-12T11:03:45.592772733-05:00
WhatFor: Implement JS lifecycle hooks around toolloop execution for observability and policy control.
WhenToUse: Use when implementing or reviewing tool-call lifecycle customization from JS across Go and JS tool origins.
---

# Implementation Plan

## Goal

Expose toolloop lifecycle hooks in JS so callers can observe and influence tool execution behavior consistently.

## Problem Statement

Current JS support includes tool registration and toolloop configuration, but lacks first-class lifecycle hooks for:

- pre-call inspection/mutation,
- post-call auditing,
- error handling and retry policy.

This limits governance, tracing, and custom orchestration logic in JS-hosted workflows.

## Target API

Hook points (initial proposal):

- `beforeToolCall(ctx)` -> optional arg mutation, skip, or abort.
- `afterToolCall(ctx)` -> observe/transform result.
- `onToolError(ctx)` -> retry, transform error, or abort.

Hook context includes at minimum:

- session/turn identifiers
- tool name and call id
- args/result/error payload
- attempt number and timestamps

## Behavior Contract

- Hooks execute synchronously on JS runtime thread to avoid race conditions.
- Hook failures follow explicit policy:
  - default fail-closed for safety-sensitive flows, or
  - configurable fail-open for resilience-oriented flows.
- Retry decisions must respect global max attempts and timeout bounds.

## Scope

In scope:

- JS hook registration surface.
- Wiring hooks for both Go-origin and JS-origin tools.
- Retry/abort protocol with validation.
- Tests and one smoke script showing hook effects.

Out of scope:

- Distributed tracing/export integrations.
- Provider-specific server-side tool lifecycle hooks.

## Implementation Approach

1. Define hook config shape in builder/tools API.
2. Introduce internal adapter around tool execution path to invoke hooks.
3. Validate hook return payloads and enforce bounds.
4. Integrate retry and abort semantics with existing toolloop iteration limits.
5. Add tests for:
   - no hooks baseline behavior,
   - arg rewrite in `beforeToolCall`,
   - retry from `onToolError`,
   - deterministic abort path.

## Testing Plan

- Unit tests:
  - `go test ./pkg/js/modules/geppetto -count=1`
- Ticket smoke script:
  - `node geppetto/ttmp/2026/02/12/GP-010-JS-TOOLLOOP-HOOKS--js-api-toolloop-lifecycle-hooks/scripts/test_toolloop_hooks_smoke.js`

## Risks and Mitigations

- Risk: hook re-entrancy/callback deadlocks.
  - Mitigation: preserve single-thread runtime callback discipline and avoid nested lock coupling.
- Risk: unbounded retries.
  - Mitigation: strict cap by per-call and global limits.

## Exit Criteria

- Hook APIs are exposed and documented.
- Tool execution path uses hooks consistently for JS and Go tools.
- Tests verify retry/abort/error handling semantics.
