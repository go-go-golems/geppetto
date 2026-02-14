---
Title: Diary
Ticket: GP-023
Status: completed
Topics:
    - webchat
    - architecture
    - pinocchio
    - refactor
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/runtime_composer.go
      Note: New core runtime composer contract
    - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/pkg/webchat/conversation.go
      Note: ConvManager refactor to composer + RuntimeFingerprint
    - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/runtime_composer.go
      Note: App-owned runtime composition policy in web-chat
    - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/cmd/web-agent-example/runtime_composer.go
      Note: App-owned runtime composition policy in web-agent-example
ExternalSources: []
Summary: Implementation diary for GP-023 runtime composer extraction and app migrations.
LastUpdated: 2026-02-14T18:10:00-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Execute GP-023 end-to-end: remove `BuildConfig`/`BuildFromConfig` from `pkg/webchat`, move runtime policy into app-owned composers in `cmd/web-chat` and `web-agent-example`, validate behavior via tests, and close the ticket.

## Step 1: Core runtime composer cutover in `pkg/webchat`

This step replaced the old core `EngineConfig`/`EngineBuilder` policy API with a new runtime composer contract and moved rebuild decisions to `RuntimeFingerprint`. The goal was to keep core focused on lifecycle/transport while pushing runtime policy out to app code.

During this step I removed `engine_builder.go` and `engine_config.go`, rewired `Router` and `ConvManager`, and updated tests to assert composer configuration requirements instead of legacy build-config behavior.

### Prompt Context

**User prompt (verbatim):** "move on to GP-023, close tickets that ar edone"

**Assistant interpretation:** Implement GP-023 now (not just plan), finish the extraction, and close completed tickets afterward.

**Inferred user intent:** Complete the architecture cutover with no compatibility layer and keep ticket bookkeeping accurate.

**Commit (code):** `04dc5e6` — "webchat: replace BuildConfig with app-owned runtime composer"

### What I did
- Added `pkg/webchat/runtime_composer.go` with:
  - `RuntimeComposeRequest`
  - `RuntimeArtifacts`
  - `RuntimeComposer` interface + function adapter
- Added `WithRuntimeComposer(...)` router option and enforced non-nil runtime composer in `NewRouter`.
- Refactored `ConvManager` in `conversation.go`:
  - removed `buildConfig/buildFromConfig` callbacks
  - switched to composer call path
  - renamed rebuild key field to `RuntimeFingerprint`
  - kept `RuntimeKey` separate
  - stored `SeedSystemPrompt` and `AllowedTools` on conversation runtime state
- Updated `router.go`:
  - added `convRuntimeComposer()` wrapper to apply default sink and optional sink wrapper
  - passed chat/ws plan overrides to `GetOrCreate`
  - removed `BuildConfig` call during inference startup
  - selected tools from conversation runtime (`AllowedTools`)
- Updated `server.go` to forward router options (`NewServer(..., opts ...RouterOption)`).
- Removed legacy files:
  - `pkg/webchat/engine_builder.go`
  - `pkg/webchat/engine_config.go`
  - `pkg/webchat/engine_config_test.go`
- Added/updated tests:
  - `pkg/webchat/router_handlers_test.go` now asserts missing composer error
  - `pkg/webchat/engine_from_req_test.go` dropped obsolete `BuildConfig` test

### Why
- The old builder API embedded app policy in core.
- Composer API keeps core transport/lifecycle-focused and makes policy explicit at app composition points.

### What worked
- Core compiles without `EngineConfig` public API.
- Conversation rebuild behavior now cleanly keys on runtime fingerprint.
- `go test ./pkg/webchat ./cmd/web-chat` and `go test ./...` in `pinocchio` both passed.

### What didn't work
- First compile pass failed with:
  - `pkg/webchat/router.go:558:21: undefined: middleware`
- Fixed by restoring the missing middleware import and rerunning tests.

### What I learned
- A small `Router`-side composer wrapper is the right place to keep default sink creation and sink wrapping while preserving app-owned policy composition.

### What was tricky to build
- The old inference path rebuilt tool allowlists via `BuildConfig` at request time. After cutover, I had to persist the effective tool allowlist on conversation runtime state and read it during inference start so tool selection remained deterministic.

### What warrants a second pair of eyes
- Runtime metadata scope in `RuntimeArtifacts` (`SeedSystemPrompt`, `AllowedTools`) should be reviewed for long-term API stability.

### What should be done in the future
- If runtime metadata grows, split optional metadata from hard contract fields to avoid overfitting core.

### Code review instructions
- Start here:
  - `pinocchio/pkg/webchat/runtime_composer.go`
  - `pinocchio/pkg/webchat/conversation.go`
  - `pinocchio/pkg/webchat/router.go`
- Validate:
  - `cd pinocchio && go test ./pkg/webchat ./cmd/web-chat`
  - `cd pinocchio && go test ./...`

### Technical details
- Commands run:
  - `gofmt -w ...` across touched go files
  - `go test ./pkg/webchat ./cmd/web-chat`
  - `go test ./...` (pinocchio)
- Pre-commit hook also ran full `go test ./...`, `go generate ./...`, lint/vet in `pinocchio` before commit.

## Step 2: App composer migrations and docs cleanup

After core cutover, I migrated both app entrypoints to explicit composer wiring and updated stale docs that still referenced removed APIs. This completed the ticket’s app migration and docs phases.

The web-agent example now composes runtime in app code and uses the updated sink wrapper signature based on compose request overrides.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish remaining GP-023 phases and close done tickets.

**Inferred user intent:** Ensure migration is complete across both applications and documentation, then mark ticket done.

**Commit (code):** `8221473` — "web-agent-example: adopt webchat runtime composer contract"

### What I did
- Implemented app composer for `cmd/web-chat`:
  - `pinocchio/cmd/web-chat/runtime_composer.go`
  - runtime fingerprint generation with API-key-safe step metadata
  - override parsing/validation moved from core to app
- Wired `cmd/web-chat/main.go` to pass `WithRuntimeComposer(...)`.
- Added `cmd/web-chat/runtime_composer_test.go` for:
  - API key redaction in fingerprint
  - override type validation
- Updated `cmd/web-chat/profile_policy_test.go` to provide a test composer when constructing router.
- Implemented app composer for `web-agent-example`:
  - `web-agent-example/cmd/web-agent-example/runtime_composer.go`
  - updated `main.go` to pass `WithRuntimeComposer(...)`
  - updated `sink_wrapper.go` signature to consume `RuntimeComposeRequest`
- Updated docs removing stale builder/config references:
  - `pinocchio/pkg/doc/topics/webchat-backend-reference.md`
  - `pinocchio/pkg/doc/tutorials/03-thirdparty-webchat-playbook.md`

### Why
- GP-023 requires app-side runtime composition in both apps and removal of stale docs referencing deleted core APIs.

### What worked
- `go test ./cmd/web-agent-example` and `go test ./...` in `web-agent-example` passed.
- `go test ./...` in `pinocchio` still passed after docs/code updates.

### What didn't work
- N/A in this step.

### What I learned
- Both apps can share the same composer pattern while keeping runtime policy fully local to app command code.

### What was tricky to build
- Keeping the disco sink wrapping behavior while removing `EngineConfig` required changing the wrapper contract to inspect `RuntimeComposeRequest.Overrides` instead of parsed core config structs.

### What warrants a second pair of eyes
- Confirm the updated tutorial/backend docs match the intended external-facing API naming and examples.

### What should be done in the future
- Add a small shared helper package for override parsing/fingerprint construction if more apps adopt the composer pattern.

### Code review instructions
- Review app migrations:
  - `pinocchio/cmd/web-chat/main.go`
  - `pinocchio/cmd/web-chat/runtime_composer.go`
  - `web-agent-example/cmd/web-agent-example/main.go`
  - `web-agent-example/cmd/web-agent-example/runtime_composer.go`
  - `web-agent-example/cmd/web-agent-example/sink_wrapper.go`
- Review docs:
  - `pinocchio/pkg/doc/topics/webchat-backend-reference.md`
  - `pinocchio/pkg/doc/tutorials/03-thirdparty-webchat-playbook.md`
- Validate:
  - `cd pinocchio && go test ./...`
  - `cd web-agent-example && go test ./...`

### Technical details
- Commands run:
  - `gofmt -w ...` for app changes
  - `go test ./cmd/web-agent-example`
  - `go test ./...` (web-agent-example)

