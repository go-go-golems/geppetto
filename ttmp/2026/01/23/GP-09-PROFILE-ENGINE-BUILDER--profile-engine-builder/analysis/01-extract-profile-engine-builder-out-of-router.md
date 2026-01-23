---
Title: Extract profile engine builder out of Router
Ticket: GP-09-PROFILE-ENGINE-BUILDER
Status: active
Topics:
    - architecture
    - backend
    - go
    - inference
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: Related Geppetto runner-builder abstraction (naming and layering guidance)
    - Path: go-go-mento/docs/reference/webchat/engine-builder.md
      Note: Baseline long-term doc describing current EngineBuilder contract
    - Path: go-go-mento/go/pkg/webchat/conversation_manager.go
      Note: Signature-based rebuild seam (GetOrCreate)
    - Path: go-go-mento/go/pkg/webchat/engine_builder.go
      Note: Current profile+override -> engine/sink/config builder (target for extraction)
    - Path: go-go-mento/go/pkg/webchat/engine_config.go
      Note: Current config signature semantics (StepSettings marshaled)
    - Path: go-go-mento/go/pkg/webchat/router.go
      Note: Current Router request/profile logic and remaining coupling
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: Reference interface-based EngineBuilder shape and signature-safety precedent
ExternalSources: []
Summary: Design analysis and incremental refactor plan to move webchat engine/profile policy out of Router behind a request-facing BuildEngineFromReq interface and a ProfileEngineBuilder implementation.
LastUpdated: 2026-01-23T08:50:40.2165869-05:00
WhatFor: Use as the blueprint for refactoring go-go-mento webchat so Router only orchestrates HTTP/WS and delegates profile/override resolution and engine composition to dedicated builder abstractions.
WhenToUse: Use when implementing GP-09 and when comparing go-go-mento webchat composition boundaries to Pinocchio/Moments/Geppetto patterns.
---


# Goal

Further extract `go-go-mento/go/pkg/webchat/engine_builder.go` (and the behavior described in `go-go-mento/docs/reference/webchat/engine-builder.md`) out of Router concerns so that:

1. Router becomes an orchestration layer that does HTTP/WS glue and delegates “engine policy + composition”.
2. Router depends on a **request-facing interface** (named here as `BuildEngineFromReq`) rather than directly encoding profile/override selection rules.
3. The webchat-specific engine builder is explicitly a **profile engine builder** (named here as `profileEngineBuilder`): it takes a `profileSlug` (plus overrides) and returns the engine (and related runtime wiring like sinks/config).

This doc is a design analysis only (no refactor implemented here).

# What exists today (go-go-mento webchat)

## Current call graph (simplified)

For HTTP `/chat`:

1. `Router.handleChatRequest` (`go-go-mento/go/pkg/webchat/router.go`)
   - parses JSON body, normalizes `conv_id`
   - resolves `profileSlug` using explicit arg / existing conversation / cookie / default
   - validates profile exists
   - calls `ConversationManager.GetOrCreate(convID, profileSlug, overrides, req)`
   - performs additional per-run setup (tool registry, Turn.Data, loop settings, goroutine run loop)

2. `ConversationManager.GetOrCreate` (`go-go-mento/go/pkg/webchat/conversation_manager.go`)
   - calls `EngineBuilder.BuildConfig(profileSlug, overrides)`
   - compares `EngineConfig.Signature()` with existing conversation
   - when changed, calls `EngineBuilder.BuildFromConfig(convID, config)` and rebuilds subscriber

3. `EngineBuilder` (`go-go-mento/go/pkg/webchat/engine_builder.go`)
   - `BuildConfig`: merge profile defaults + override parsing into an `EngineConfig`
   - `BuildFromConfig`: construct Watermill sink + wrap with profile-specific extractors; compose engine from StepSettings + middleware factories

## What is already “good” about the current design

- There is a clear **engine-config signature** seam (`ConversationManager.GetOrCreate`): rebuilds are deterministic and local.
- The builder already centralizes “compose engine + sink” so that handlers don’t repeat middleware composition logic.
- There is an explicit separation between:
  - `BuildConfig` (policy + signature inputs)
  - `BuildFromConfig` (allocation + wiring)

# What is still too coupled to Router (and why it matters)

Even with `EngineBuilder`, Router still owns several responsibilities that are “engine policy” rather than HTTP/WS:

## 1) Profile selection is duplicated and transport-shaped

Router chooses profile slug in multiple places:

- WS join: query param `profile` → cookie fallback → default.
- HTTP `/chat`: explicit arg → existing conversation profile → cookie fallback → default.

This selection policy is exactly the kind of logic you want behind a single interface so it can evolve (e.g., identity-scoped defaults, org/team profiles, prompt resolver requirements) without touching handler code.

## 2) Tool registry composition is in Router, not in the builder

In `Router.handleChatRequest`, tool registry building is done inline:

- It registers all tools from `r.toolFactories`.
- It filters by `p.DefaultTools`.

Consequences:

- The builder’s `EngineConfig.Tools` and tool override parsing (`overrides["tools"]`) do not currently drive actual tool availability.
- Tool availability is not included in the engine-config signature in a way that’s enforced at runtime (tools are built separately from the config used to decide rebuild).

If the “engine build” abstraction is meant to be real, the tool registry composition needs to sit with the config/policy layer (or at least be driven by it).

## 3) Run-loop policy (max iterations, timeout, step mode) is split across layers

The `Profile` type includes loop-related config (`MaxIterations`, `TimeoutSeconds`, `LoopName`, `AllowOverrides`) but:

- `MaxIterations` and `TimeoutSeconds` are enforced in Router, not in the builder/config.
- `LoopName` and `AllowOverrides` are currently unused.

This prevents “engine configuration” from being a single debuggable source of truth for what a run will do.

## 4) Builder has a hard dependency on `ConversationManager` (circularity workaround)

`EngineBuilder.SetConversationManager` is a pragmatic fix for circular init, but it means the “engine composition” layer depends on the whole conversation manager type rather than a minimal interface.

That makes it harder to:

- test builder logic without a full manager
- reuse builder in other contexts
- keep dependencies acyclic as the system grows

# Issues found (opportunities to fix while extracting)

## A) EngineConfig signature likely embeds secrets

`go-go-mento/go/pkg/webchat/engine_config.go` uses `json.Marshal(ec)` where `ec.StepSettings` is a pointer to `settings.StepSettings`.

In other parts of this monorepo (notably Pinocchio’s PI-001 work), StepSettings are treated as potentially containing secrets (API keys), so signatures must be derived from a **sanitized metadata view** rather than the full struct.

Recommendation:
- Change the signature to exclude secrets and reduce churn noise (only include stable metadata fields that affect engine behavior).

## B) Tool overrides are currently “parsed but not applied”

`EngineBuilder.BuildConfig` merges `Profile.DefaultTools` + `overrides["tools"]` into `EngineConfig.Tools`, but `Router.handleChatRequest` builds the tool registry based on `Profile.DefaultTools` only.

Recommendation:
- Drive tool registry filtering from `EngineConfig.Tools` (or remove tool overrides entirely if they are not a supported feature).

## C) `Profile.AllowOverrides` exists but is unused

If overrides are security-relevant (e.g., allowing system prompt changes), the profile should be able to disallow them.

Recommendation:
- Either enforce `AllowOverrides` in the policy layer (builder/config) or remove the field to avoid a false sense of safety.

# Target shape (what “real EngineBuilder” should mean here)

The key split implied by your request is a *two-stage* boundary:

1. **Request → engine-build input** (`BuildEngineFromReq`)
2. **Profile engine builder** (`profileEngineBuilder`) that builds engine/sink/config from profile+overrides (and possibly additional non-HTTP dependencies)

## Stage 1: Request-facing interface (`BuildEngineFromReq`)

Router should depend on something like:

```go
type EngineFromReqBuilder interface {
    BuildEngineFromReq(req *http.Request) (EngineBuildInput, error)
}

type EngineBuildInput struct {
    ConvID      string
    ProfileSlug string
    Overrides   map[string]any
}
```

Notes:

- This interface is intentionally request-shaped. It absorbs cookie/query/path/body precedence rules and can evolve independently from Router.
- `EngineBuildInput` becomes the canonical “what profile/overrides does this request want?” structure.

Router can then do:

1) parse request (or delegate body parsing too, if desired)
2) call `BuildEngineFromReq`
3) call `ConversationManager.GetOrCreate` with the returned `EngineBuildInput`

## Stage 2: Webchat profile engine builder (`profileEngineBuilder`)

The existing go-go-mento `EngineBuilder` is already close to this concept, but should be made explicit as profile-based composition:

```go
type ProfileEngineBuilder interface {
    BuildConfig(profileSlug string, overrides map[string]any) (EngineConfig, error)
    BuildFromConfig(convID string, config EngineConfig) (engine.Engine, events.EventSink, error)
    // Optional: BuildTools(config EngineConfig) (tools.ToolRegistry, error)
}
```

This aligns with the already-adopted Pinocchio `EngineBuilder` interface shape, while letting go-go-mento keep its richer sink-wrapping pipeline.

# Concrete extraction plan (incremental, low-risk)

## Phase 1: Introduce `BuildEngineFromReq` without changing engine composition

Goal: remove profile/override resolution policy from handlers.

- Create a small component (in `go-go-mento/go/pkg/webchat/` or a subpackage) that:
  - normalizes `conv_id`
  - selects profile slug with the current precedence rules
  - extracts overrides from the already-parsed `ChatRequestBody` (recommended) or directly from `*http.Request` (less ideal)
  - validates `ProfileRegistry` contains the selected profile
- Update Router handlers to call it and pass the result to `ConversationManager.GetOrCreate`.

Outcome:
- Router becomes thinner immediately, without changing EngineBuilder internals.

## Phase 2: Make the builder truly “profile-driven” (tools + allow-overrides)

Goal: ensure EngineConfig actually drives runtime behavior.

- Enforce `Profile.AllowOverrides` inside `BuildConfig`:
  - either ignore overrides when disallowed, or reject the request with a clear error (prefer explicit errors).
- Move tool registry composition behind builder/policy:
  - implement `BuildTools(config EngineConfig)` that uses the builder’s `toolFactories`
  - drive allowed tools from `config.Tools` (which already includes defaults + override merge)
- Ensure `ConversationManager` (or Router) attaches the built registry to the Turn (or wherever toolloop expects it).

Outcome:
- Overrides and profile defaults become consistent across config/signature/runtime.

## Phase 3: Fix config signatures (no secrets) and expand signature scope

Goal: make rebuild semantics correct and safe.

- Replace `EngineConfig.Signature()` to avoid embedding raw `StepSettings`.
  - Use a sanitized metadata representation (similar to Pinocchio’s approach described in PI-001’s status update).
- Consider including run-loop policy (max iterations, timeouts) in the config/signature if changing these should trigger rebuild.

Outcome:
- Deterministic rebuilds without secret leakage in signatures/logs.

## Phase 4: Break `ConversationManager` dependency cycle cleanly

Goal: avoid builder requiring `*ConversationManager`.

- Replace `SetConversationManager(*ConversationManager)` with a minimal interface (example):

```go
type ConversationLookup interface {
    Get(convID string) (*Conversation, bool)
    FindByRunID(runID string) (*Conversation, string, bool)
}
```

- Update sink wrappers to accept `ConversationLookup` (or a narrower interface matching the extractors’ needs).

Outcome:
- Easier testing and cleaner dependency graph.

# How this relates to the other “EngineBuilder” abstractions in the monorepo

There are multiple “builder” concepts:

- `go-go-mento/go/pkg/webchat/EngineBuilder`: builds `engine.Engine` + `events.EventSink` from profile/overrides.
- `geppetto/pkg/inference/session.EngineBuilder`: builds an `InferenceRunner` for a session.
- `geppetto/pkg/inference/toolloop/enginebuilder.Builder`: builds a runner that executes single-pass or toolloop inference, attaching sinks and hooks.

Recommendations for clarity:

- Use explicit names at the webchat layer:
  - `ProfileEngineBuilder` (profile → config/engine/sink)
  - `EngineFromReqBuilder` (request → `EngineBuildInput`)
- Keep `session.EngineBuilder` reserved for “build an inference runner”, not “build an engine”.

# Open questions (should be answered before implementing)

1. Should overrides be *rejected* when disallowed, or *silently ignored*?
2. Should tool overrides be supported as a first-class feature, or should tools be profile-fixed?
3. Should engine rebuilds be triggered when loop settings change, or is rebuild strictly about engine/sink wiring?
4. Should `BuildEngineFromReq` own request body parsing, or should Router decode JSON and pass a typed body struct into it?

# Appendix: High-signal reference files

- go-go-mento (current target):
  - `go-go-mento/go/pkg/webchat/engine_builder.go`
  - `go-go-mento/go/pkg/webchat/engine_config.go`
  - `go-go-mento/go/pkg/webchat/conversation_manager.go`
  - `go-go-mento/go/pkg/webchat/router.go`
  - `go-go-mento/docs/reference/webchat/engine-builder.md`
- pinocchio/geppetto (reference patterns):
  - `pinocchio/pkg/webchat/engine_builder.go`
  - `geppetto/pkg/inference/session/builder.go`
  - `geppetto/pkg/inference/toolloop/enginebuilder/builder.go`
