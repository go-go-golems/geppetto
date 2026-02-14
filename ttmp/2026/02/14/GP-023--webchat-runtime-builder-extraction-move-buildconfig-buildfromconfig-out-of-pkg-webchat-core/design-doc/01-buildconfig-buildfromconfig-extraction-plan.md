---
Title: BuildConfig and BuildFromConfig Extraction Plan
Ticket: GP-023
Status: active
Topics:
  - webchat
  - architecture
  - pinocchio
  - refactor
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
  - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/engine_builder.go
    Note: Current core ownership of BuildConfig and BuildFromConfig
  - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/conversation.go
    Note: ConvManager wiring to buildConfig/buildFromConfig callbacks
  - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/router.go
    Note: startInferenceForPrompt composes runtime using core builder methods
  - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/profile_policy.go
    Note: App-owned request policy; natural home for runtime composition policy
ExternalSources: []
Summary: Plan to remove BuildConfig/BuildFromConfig from pkg/webchat and make runtime composition fully app-owned.
---

# Executive Summary

`pkg/webchat` still owns `BuildConfig` and `BuildFromConfig`, which encode application policy (defaults, override semantics, settings decoding). This violates the architectural direction established in GP-022 (core lifecycle + transport in package, policy in app).

This ticket proposes a clean extraction: app code provides a runtime composer callback (or interface) that returns `(engine, sink, fingerprint, runtime metadata)` and core only manages conversation lifecycle, queueing, persistence hooks, websocket transport, and rebuild orchestration.

# Problem Statement

Even after resolver cutover, core webchat still contains policy logic:

- system prompt defaulting (`"You are an assistant"`)
- middleware/tool override validation and parsing
- step-settings decoding from parsed command values
- composition coupling between runtime key and core-side engine config model

Consequences:

- third-party apps are forced through core assumptions they should own
- runtime behavior changes require touching `pkg/webchat`
- core abstractions leak app semantics and reduce composability

# Proposed Solution

## 1) Introduce app-owned runtime composer contract

Add a new contract in `pkg/webchat` for runtime materialization that is policy-free:

```go
type RuntimeArtifacts struct {
    Engine      engine.Engine
    Sink        events.EventSink
    Fingerprint string
    RuntimeKey  string
}

type RuntimeComposer interface {
    Compose(ctx context.Context, req RuntimeComposeRequest) (RuntimeArtifacts, error)
}

type RuntimeComposeRequest struct {
    ConvID     string
    RuntimeKey string
    Overrides  map[string]any
}
```

Router option:

```go
func WithRuntimeComposer(c RuntimeComposer) RouterOption
```

`pkg/webchat` should fail fast on startup if no composer is configured (no hidden default policy).

## 2) Remove core engine-builder policy types

Delete or internalize these from `pkg/webchat` public API:

- `EngineConfig`
- `EngineBuilder` interface
- `Router.BuildConfig`
- `Router.BuildFromConfig`
- parser helpers for override policy (`validateOverrides`, etc.)

These become app-private implementation details in `cmd/web-chat` and any third-party app.

## 3) ConvManager rebuild semantics on fingerprint only

`ConvManager.GetOrCreate(...)` already moved to signature-based rebuild decisions. Replace `EngineConfigSignature` naming with generic `RuntimeFingerprint` once composer lands to avoid stale naming.

Behavior:

- `GetOrCreate(convID, runtimeKey, overrides)` calls composer
- compares returned `Fingerprint` to conversation `RuntimeFingerprint`
- rebuilds engine/sink/subscriber only when fingerprint differs

## 4) Move default runtime policy to app layer

In `cmd/web-chat`:

- keep request policy in `profile_policy.go`
- add app runtime composer implementation (profile defaults + overrides + step settings)
- wrap sink with app extraction behavior if needed

In `web-agent-example`:

- runtime composer defines default runtime (`default`) + middleware behavior
- no dependency on core policy helpers

# Design Decisions

1. **No fallback builder in core**
- Rationale: force explicit ownership and prevent regression into implicit defaults.

2. **Fingerprint supplied by composer**
- Rationale: app decides what runtime dimensions should trigger rebuilds.

3. **RuntimeKey remains in core conversation metadata**
- Rationale: observability/debug and WS hello context still need runtime identity.

4. **No compatibility shim**
- Rationale: requested clean cutover; no legacy adapter layer.

# Alternatives Considered

## A) Keep `BuildConfig` in core, move only `BuildFromConfig`
Rejected: still centralizes policy parsing/defaulting in core.

## B) Keep `EngineConfig` as generic core DTO
Rejected: tends to re-accumulate app policy and schema drift over time.

## C) Make resolver return engine directly
Rejected: resolver purpose is request policy. Runtime composition has distinct lifecycle and should remain separately injectable.

# Implementation Plan

## Phase 1: Core API reshape

1. Add `RuntimeComposer` + `RuntimeArtifacts` + `RuntimeComposeRequest`.
2. Add `WithRuntimeComposer` router option.
3. Wire `ConvManager` to use composer output and fingerprint.
4. Rename conversation field from `EngineConfigSignature` to `RuntimeFingerprint`.

## Phase 2: Remove old builder API

1. Remove `EngineConfig` and related core builder methods.
2. Remove override parser helpers from core.
3. Update tests in `pkg/webchat` to use test composers.

## Phase 3: App migrations

1. Implement composer in `cmd/web-chat`.
2. Implement composer in `web-agent-example`.
3. Ensure existing behavior parity via targeted tests.

## Phase 4: Docs and cleanup

1. Update webchat architecture docs with resolver + composer separation.
2. Update tutorial samples.
3. Remove stale references from comments and debug payload docs.

# Risks and Mitigations

- **Risk:** behavior drift in runtime defaults after move.
  - Mitigation: snapshot tests in app layers for default + override combinations.

- **Risk:** missing composer setup causes runtime failures.
  - Mitigation: explicit startup error when composer is nil.

- **Risk:** fingerprint mismatch logic could trigger unnecessary rebuilds.
  - Mitigation: app-level deterministic fingerprint tests with stable ordering.

# Open Questions

1. Should composer receive parsed command values directly, or app capture them in closure at router creation time?
2. Should sink wrapping be part of composer return, or remain a separate `WithEventSinkWrapper` mechanism?
3. Do we want a standard helper in app code for deterministic fingerprint hashing/JSON normalization?

# Acceptance Criteria

- `pkg/webchat` has no public `BuildConfig`/`BuildFromConfig` API.
- Runtime composition policy lives entirely in app layer.
- Core rebuild decisions are based on app-provided fingerprint.
- `cmd/web-chat` and `web-agent-example` compile and run with explicit composers.
- Docs reflect resolver + composer split with no legacy API references.
