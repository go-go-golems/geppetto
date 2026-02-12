---
Title: Diary
Ticket: GP-009-JS-ENGINE-PROFILES
Status: active
Topics:
    - geppetto
    - goja
    - javascript
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Implementation diary for JS engine profile/config factories
LastUpdated: 2026-02-12T11:17:55.287417724-05:00
WhatFor: Track implementation and validation details for engines.fromProfile/fromConfig JS APIs.
WhenToUse: Use when reviewing profile/config engine factory behavior and precedence rules in GP-009.
---

# Diary

## Goal

Implement profile/config-driven JS engine factories:

- `engines.fromProfile(profile?, opts?)`
- `engines.fromConfig(opts)`

with deterministic precedence and direct integration into existing engine factory code.

## Step 1: Add Engine Factory APIs to JS Module

I extended exports and runtime to support profile/config constructors.

### Changes

- Updated `pkg/js/modules/geppetto/module.go`:
  - added exports:
    - `engines.fromProfile`
    - `engines.fromConfig`
- Updated `pkg/js/modules/geppetto/api.go`:
  - added profile/config settings builders
  - added provider inference + env key fallback helpers
  - integrated with `enginefactory.NewEngineFromStepSettings`

## Step 2: Precedence and Resolution Rules

Implemented precedence for `fromProfile`:

1. explicit `profile` arg
2. `opts.profile`
3. `PINOCCHIO_PROFILE`
4. default `"4o-mini"`

Other behavior:

- Provider/API type:
  - explicit `opts.apiType` or `opts.provider` if supplied
  - otherwise inferred from model/profile name
- API key resolution:
  - `opts.apiKey`
  - env fallback by provider (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`/`GOOGLE_API_KEY`)

## Step 3: Safe Config Overrides

Added optional overrides from JS options:

- `model`
- `temperature`
- `topP`
- `maxTokens`
- `timeoutSeconds` / `timeoutMs`
- `baseURL`

Also added required compat aliases:

- openai-responses key path includes `openai-api-key`

## Step 4: Tests and Smoke Script

### Unit tests added

- `pkg/js/modules/geppetto/module_test.go`:
  - `TestEnginesFromProfileAndFromConfigResolution`
  - `TestEngineFromProfileInferenceIntegration_Gemini` (live, opt-in with `GEPPETTO_LIVE_INFERENCE_TESTS=1`)

### Smoke script added

- `geppetto/ttmp/2026/02/12/GP-009-JS-ENGINE-PROFILES--js-api-engine-profile-and-config-factory/scripts/test_engine_profile_smoke.js`

### Commands run

```bash
go test ./pkg/js/modules/geppetto -run 'TestEnginesFromProfileAndFromConfigResolution|TestEngineFromProfileInferenceIntegration_Gemini' -count=1 -v
node geppetto/ttmp/2026/02/12/GP-009-JS-ENGINE-PROFILES--js-api-engine-profile-and-config-factory/scripts/test_engine_profile_smoke.js
```

### Outcomes

- Resolution test passed.
- Live integration test skipped by default (`GEPPETTO_LIVE_INFERENCE_TESTS` gate).
- Smoke script passed and executed real inference command against:
  - `PINOCCHIO_PROFILE=gemini-2.5-flash-lite`

## Bug Encountered and Fixed

Issue:

- `fromProfile(undefined, { profile: "opts-model" })` incorrectly produced a `config` engine name.

Fix:

- Added a `fromProfile` mode flag in engine creation path and corrected naming semantics:
  - profile mode -> `profile:<resolved-profile>`
  - config mode -> `config`

## Review Pointers

- `pkg/js/modules/geppetto/module.go`
- `pkg/js/modules/geppetto/api.go`
- `pkg/js/modules/geppetto/module_test.go`
- `geppetto/ttmp/2026/02/12/GP-009-JS-ENGINE-PROFILES--js-api-engine-profile-and-config-factory/scripts/test_engine_profile_smoke.js`
