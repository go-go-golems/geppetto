---
Title: Implementation Plan
Ticket: GP-009-JS-ENGINE-PROFILES
Status: active
Topics:
    - geppetto
    - goja
    - javascript
    - inference
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Plan for engine factory APIs based on profile/config from JS
LastUpdated: 2026-02-12T11:03:45.593143904-05:00
WhatFor: Implement JS engine factories that resolve provider/model configuration from profiles and explicit options.
WhenToUse: Use when implementing or reviewing JS-driven engine construction from profile names or provider config objects.
---

# Implementation Plan

## Goal

Allow JS callers to create inference engines without manually wiring Go engine objects by adding profile/config-based factory APIs.

## Problem Statement

Current JS API supports:

- `engines.echo`
- `engines.fromFunction`

It lacks native constructors for real provider-backed inference engines from profile/config, forcing external Go wiring.

## Target API

- `engines.fromProfile(profile, opts?)`
- `engines.fromConfig(opts)`

Where `opts` can include normalized runtime overrides such as:

- model
- timeout
- max tokens
- temperature
- provider-specific flags when safely mappable

## Resolution Rules

Proposed precedence:

1. Explicit function arg (`profile` / `opts`)
2. Explicit fields inside `opts`
3. `PINOCCHIO_PROFILE` environment variable
4. repository default profile behavior

Unknown profile/provider should return structured, actionable errors.

## Scope

In scope:

- JS API surface for profile/config engine creation.
- Mapping from JS options to existing Go engine constructors.
- Validation and error reporting.
- Tests and smoke script using `PINOCCHIO_PROFILE=gemini-2.5-flash-lite`.

Out of scope:

- Full provider settings parity for every backend in first iteration.
- Runtime hot-swapping engine internals during active inference.

## Implementation Approach

1. Identify stable Go entry points for profile resolution.
2. Add new methods to `engines` export in `pkg/js/modules/geppetto/module.go`.
3. Implement parsing/validation in `pkg/js/modules/geppetto/api.go`.
4. Reuse existing builder/session flow once engine is created.
5. Add tests for:
   - success with valid profile,
   - unknown profile failure,
   - override precedence behavior.

## Testing Plan

- Unit tests:
  - `go test ./pkg/js/modules/geppetto -count=1`
- Inference smoke script (real model path):
  - `PINOCCHIO_PROFILE=gemini-2.5-flash-lite node geppetto/ttmp/2026/02/12/GP-009-JS-ENGINE-PROFILES--js-api-engine-profile-and-config-factory/scripts/test_engine_profile_smoke.js`

## Risks and Mitigations

- Risk: option-shape drift across providers.
  - Mitigation: implement strict normalization and explicit unsupported-option errors.
- Risk: environment-dependent behavior confusion.
  - Mitigation: document precedence and expose resolved profile in debug logging.

## Exit Criteria

- `engines.fromProfile` and `engines.fromConfig` are available and tested.
- Profile resolution precedence is documented and validated by tests.
- Smoke script demonstrates real inference on `gemini-2.5-flash-lite`.
