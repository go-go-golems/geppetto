---
Title: GP-40 Implementation Tasks
Ticket: GP-40-OPINIONATED-GO-APIS
Status: active
Topics:
    - geppetto
    - go-api
    - architecture
    - go
DocType: planning
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/session/session.go
      Note: Core session type that the new runner will wrap
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: Existing builder that the new runner will assemble internally
    - Path: geppetto/pkg/inference/middlewarecfg/resolver.go
      Note: Middleware-use resolution path to reuse
ExternalSources: []
Summary: Slice-by-slice implementation task board for building the new Geppetto opinionated runner package and migrating first-party examples.
LastUpdated: 2026-03-18T03:29:00-04:00
WhatFor: Use as the live execution board while implementing GP-40 in reviewable commits.
WhenToUse: Use when tracking or reviewing runner implementation progress.
---

# Tasks

## Completed Discovery And Design Work

- [x] Create a Manuel-specific GP-40 workspace and diary without modifying the colleague's parallel GP-40 workspace
- [x] Analyze Geppetto core runner, session, tool loop, tools, middleware, middlewarecfg, and profile surfaces
- [x] Analyze Pinocchio and downstream usage in CozoDB Editor, CoinVault, and Temporal Relationships
- [x] Write a detailed architecture and rationale document for the opinionated runner direction
- [x] Update the GP-40 design after GP-41, GP-42, GP-43, and GP-45 moved resolution and policy to the app side
- [x] Add a concrete implementation plan document for `pkg/inference/runner`
- [x] Add practical event-driven examples showing streaming and channel-backed usage

## Implementation Workboard

### Slice 1: Package Skeleton And Public Boundary

- [x] Create `geppetto/pkg/inference/runner/`
- [x] Add `types.go` with the public `Runtime`, `StartRequest`, `PreparedRun`, `ToolRegistrar`, and result types
- [x] Add `options.go` with `Runner`, `Option`, default loop/tool config handling, and constructor helpers
- [x] Add `errors.go` with package-scoped validation errors
- [x] Add initial package docs so the public boundary is obvious in `go doc`
- [ ] Commit the boundary freeze as the first implementation commit

### Slice 2: Tool Registration Helpers

- [x] Add `tools.go`
- [x] Implement `FuncTool(...)` and `MustFuncTool(...)`
- [x] Implement registry construction from tool registrars
- [x] Implement registry filtering from `Runtime.ToolNames`
- [x] Add tests covering nil registrars, duplicate tools, and name filtering
- [ ] Commit the tool-registration slice

### Slice 3: Middleware Resolution And Engine Assembly

- [ ] Add `middleware.go`
- [ ] Resolve direct `Runtime.Middlewares` first
- [ ] Resolve `Runtime.MiddlewareUses` through `middlewarecfg` when direct middlewares are absent
- [ ] Inject system-prompt middleware in one consistent place
- [ ] Build the base engine from final `StepSettings`
- [ ] Wrap the engine with the resolved middleware chain
- [ ] Add tests covering direct middleware, middleware-use resolution, and prompt injection
- [ ] Commit the middleware and engine-assembly slice

### Slice 4: Prepare

- [ ] Add `prepare.go`
- [ ] Validate request shape and final runtime input
- [ ] Create or attach a session
- [ ] Append the seed prompt or provided seed turn
- [ ] Build the `enginebuilder.Builder`
- [ ] Build and attach the registry, event sinks, snapshot hook, persister, and step controller
- [ ] Return a `PreparedRun` with the assembled session, engine, registry, and initial turn
- [ ] Add tests covering prompt-only, seed-turn, and invalid-input paths
- [ ] Commit the `Prepare(...)` slice

### Slice 5: Start And Run

- [ ] Add `run.go`
- [ ] Implement `Start(...)` on top of `Prepare(...)`
- [ ] Implement `Run(...)` as sync prepare-start-wait flow
- [ ] Return structured results instead of forcing callers to inspect the raw session state
- [ ] Add tests for sync and async execution
- [ ] Add at least one event-sink test proving the streaming path still works
- [ ] Commit the `Start(...)` and `Run(...)` slice

### Slice 6: First-Party Examples And Package Documentation

- [ ] Add or migrate one minimal CLI example to the new runner
- [ ] Add or migrate one tools example to the new runner
- [ ] Add or migrate one event-driven example to the new runner
- [ ] Update Geppetto docs so `pkg/inference/runner` is the recommended entry point for new apps
- [ ] Commit the examples and docs slice

### Slice 7: Validation And Ticket Close-Out

- [ ] Run focused tests for `pkg/inference/runner`
- [ ] Run full Geppetto lint and repo tests
- [ ] Update GP-40 changelog with the implementation sequence and commit ids
- [ ] Update the GP-40 diary with exact commands, failures, and review guidance
- [ ] Re-upload the refreshed GP-40 bundle to reMarkable and verify the remote listing
- [ ] Mark the ticket complete once code and docs are aligned
