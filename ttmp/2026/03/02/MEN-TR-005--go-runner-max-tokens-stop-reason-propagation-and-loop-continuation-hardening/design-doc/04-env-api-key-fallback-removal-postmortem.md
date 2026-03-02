---
Title: Env API-key fallback removal postmortem
Ticket: MEN-TR-005
Status: active
Topics:
    - temporal-relationships
    - geppetto
    - stop-policy
    - claude
    - extraction
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/extractor/gorunner/run.go
      Note: Go runner credential/provider resolution path that previously used env fallback
    - Path: scripts/01-repro-max-tokens-stop-reason.sh
      Note: Repro script changed to avoid env-key preflight dependency
ExternalSources: []
Summary: Postmortem of removing runtime environment API-key fallback, including historical reasons it was added, what failed in practice, and guardrails for deterministic profile/StepSettings-based credential wiring.
LastUpdated: 2026-03-02T15:40:00-05:00
WhatFor: Explain the full cause chain for env fallback introduction and removal, then establish prevention guidance.
WhenToUse: Use when auditing credential precedence behavior or reviewing any future change touching provider key resolution.
---

# Env API-key fallback removal postmortem

## Executive Summary

On **March 2, 2026**, we removed implicit provider API-key fallback from runtime code paths in both the Geppetto JS bindings and the Temporal Relationships go runner. Before this change, missing provider keys in resolved `StepSettings` could be silently backfilled from process environment variables (for example `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`).

That fallback was likely integrated as a compatibility and ergonomics bridge during the migration from model/env-first flows to profile-registry-first flows. It reduced initial setup friction, but it also introduced hidden precedence, non-deterministic behavior across environments, and test blind spots. After removal, credential resolution is now explicit and auditable: keys come from profile registry runtime patches or explicit config fields, not ambient environment state.

## Problem Statement

The platform now aims for deterministic runtime composition through Glazed fields, profile registries, and `StepSettings`. Environment fallback violates that principle because the runtime can produce materially different engine behavior depending on shell state, even when profile/config inputs are identical.

### Observable failure pattern

- A profile or config missing a provider key could still "work" on one machine due to `os.Getenv(...)` fallback.
- The same inputs could fail on CI or another machine where env vars were absent.
- Engineers could mistakenly believe profile wiring was correct while runtime was actually reading process env.

### Why this was high impact

Credential resolution affects every provider-backed inference path. Hidden fallback at this level causes confusing behavior in CLI, web server, JS bindings, and extraction pipelines.

## Scope and Timeline

### Impacted runtime surfaces

- Geppetto JS bindings:
  - `gp.engines.fromProfile(...)`
  - `gp.engines.fromConfig(...)`
- Temporal Relationships go runner:
  - profile-resolved engine path
  - config-resolved engine path

### Concrete timeline

1. **February 2026 to early March 2026**: profile-registry hard-cutover and go runner extraction flows landed.
2. **March 2, 2026 (earlier)**: follow-on extraction hardening work identified inconsistency risk in key sourcing.
3. **March 2, 2026 (this change)**: env fallback removed and tests/docs updated.
4. **March 2, 2026 (validation)**: longest anonymized transcript repro executed successfully with profile-backed credentials only.

## What Changed

### Removed behavior

Previous (simplified):

```go
// old behavior (removed)
if stepSettingsMissingProviderKey() {
    key := os.Getenv(providerSpecificEnvName)
    if key != "" {
        injectKeyIntoStepSettings(key)
    }
}
```

This existed in:

- `geppetto/pkg/js/modules/geppetto/api_engines.go`
- `temporal-relationships/internal/extractor/gorunner/run.go`

### New behavior

Current (simplified):

```go
// new behavior
ss := resolveStepSettingsFromProfileOrConfig(...)
applyProviderDefaults(ss) // base URLs etc.
// no env lookup here
engine := NewEngineFromStepSettings(ss)
```

Keys now must be present via one of these explicit inputs:

- profile registry runtime patch (for example `openai-chat.openai-api-key`, `claude-chat.claude-api-key`), or
- explicit config/API field (for example JS `fromConfig({ apiKey: ... })`, go runner `engine.apiKey` when using config-mode)

## Why Env Fallback Was Probably Added

This section is inferential but evidence-backed by commit chronology and migration context.

### Likely motivation 1: preserve old "just set env vars" workflow

Historically, many CLI and script flows used provider env vars directly. During migration to profile-registry resolution, fallback likely reduced migration pain by keeping those flows operational.

### Likely motivation 2: bridge partial profile adoption

During hard-cutover work on JS `fromProfile`, some profiles/configs were still incomplete. Env fallback made incomplete runtime patches appear functional until full profile data was in place.

### Likely motivation 3: reduce demo/test friction

For local examples and rapid prototyping, requiring full profile patches can feel heavier than setting one env var. Fallback gave quick path ergonomics, especially in ad-hoc JS or extraction experiments.

## Root Cause Analysis

The root issue was not a single bug but a contract mismatch:

- Architecture direction: explicit profile/StepSettings contracts.
- Runtime behavior: implicit ambient fallback.

### Root cause

Credential resolution was split between two sources of truth:

1. explicit runtime config (profiles + StepSettings), and
2. implicit process env fallback.

When two sources exist and one is invisible in persisted config, determinism is lost.

### Contributing factors

- Migration period where compatibility shortcuts were favored.
- Tests that passed because env fallback masked malformed profile patch sections.
- Existing documentation still mentioning env precedence in broader profile selection context.
- Multi-surface implementation (JS bindings + go runner) increased drift risk.

### Trigger that exposed issue

After fallback removal, JS test fixtures failed because they used legacy `step_settings_patch.api.*` sections instead of schema-backed provider sections (`openai-chat`, `claude-chat`). This confirmed fallback had been masking incorrect fixture shape.

## Incident Diagram

```text
Before (problematic)

profile/config -> StepSettings (missing key)
                           |
                           v
                    env fallback injects key
                           |
                           v
                       engine runs

Result: same config can pass/fail depending on shell env


After (deterministic)

profile/config -> StepSettings (must include key)
                           |
                           v
                    no implicit env mutation
                           |
                           v
             engine creation succeeds or fails explicitly

Result: behavior matches declared runtime inputs
```

## Design Decisions and Rationale

### Decision 1: remove runtime `os.Getenv` key inference in core paths

Rationale: runtime composition should be fully derived from declared settings objects.

### Decision 2: keep provider defaults (base URLs) but not provider credentials

Rationale: base URL defaults are safe operational defaults; credentials are sensitive identity inputs and must be explicit.

### Decision 3: update docs to state no-env-fallback contract in JS engine helpers

Rationale: avoid accidental reintroduction and reduce onboarding confusion.

### Decision 4: fix tests to use schema-backed provider sections

Rationale: validate real supported contract, not legacy patch shapes.

## Alternatives Considered

### Alternative A: keep fallback but gate with flag

- Pros: easier incremental migration.
- Cons: still creates hidden behavior branches and policy drift.
- Decision: rejected.

### Alternative B: fallback only in dev mode

- Pros: dev ergonomics.
- Cons: different semantics between dev/prod; still brittle.
- Decision: rejected.

### Alternative C: fallback only when explicit toggle present in config

- Pros: explicit opt-in.
- Cons: extends credential surface and complexity; weakens profile-first direction.
- Decision: rejected.

## Verification Evidence

### Tests

- Geppetto targeted tests passed after fixture migration.
- Go runner tests passed with fallback removed.

### End-to-end run

- Longest anonymized transcript run executed with profile registry credentials.
- Result included low-token continuation evidence (`iterations=2`, repeated inference starts), confirming runtime viability without env-key fallback.

## Playbook-Grade Guardrails (Short Form)

1. Never read provider API keys from `os.Getenv(...)` inside runtime engine-construction paths.
2. If a key is missing, fail with explicit provider-key error.
3. Keep key material in profile runtime patch/provider sections or explicit API options.
4. Ensure tests do not set provider env vars unless specifically testing CLI bootstrap behavior.
5. Keep docs explicit about where keys must be supplied.

## Implementation Plan (Prevent Regression)

1. Add static grep check in CI for new `inferAPIKeyFromEnv`-like helpers in runtime paths.
2. Add contract tests that assert missing key errors when profile patches omit keys.
3. Add fixture lint check for runtime patch section slugs (`openai-chat`, `claude-chat`, etc.).
4. Keep migration notes in profile docs clear about key placement.

## Open Questions

1. Should cross-application bootstrap layers still support env-to-profile materialization as an explicit pre-runtime import step (outside engine creation), or should env usage be fully deprecated end-to-end?
2. Should we introduce a formal "credential source report" in debug output to show key provenance without leaking key values?

## References

- `geppetto/pkg/js/modules/geppetto/api_engines.go`
- `geppetto/pkg/js/modules/geppetto/module_test.go`
- `geppetto/pkg/doc/topics/13-js-api-reference.md`
- `geppetto/pkg/doc/topics/14-js-api-user-guide.md`
- `temporal-relationships/internal/extractor/gorunner/run.go`
- `temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/scripts/01-repro-max-tokens-stop-reason.sh`
