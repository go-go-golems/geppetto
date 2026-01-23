---
Title: Postmortem
Ticket: GP-09-PROFILE-ENGINE-BUILDER
Status: active
Topics:
    - architecture
    - backend
    - go
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/analysis/01-extract-profile-engine-builder-out-of-router.md
      Note: |-
        Design blueprint for the refactor (definitions, phases, and rationale)
        Design blueprint
    - Path: geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md
      Note: |-
        Step-by-step implementation diary with commands, failures, and commits
        Implementation diary
    - Path: moments/backend/pkg/webchat/engine_from_req.go
      Note: |-
        Moments request policy builder implementation
        Moments request policy builder
    - Path: moments/backend/pkg/webchat/router.go
      Note: |-
        Moments router now delegates request policy for HTTP + WS
        Moments Router delegates request policy
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: |-
        Pinocchio config builder now derives Tools and enforces AllowOverrides
        Pinocchio config now derives Tools + enforces AllowOverrides
    - Path: pinocchio/pkg/webchat/engine_config.go
      Note: |-
        Pinocchio engine config signature now includes Tools + sanitized step metadata
        Pinocchio EngineConfig signature includes Tools + sanitized metadata
    - Path: pinocchio/pkg/webchat/engine_from_req.go
      Note: |-
        Pinocchio request policy builder implementation
        Pinocchio request policy builder
    - Path: pinocchio/pkg/webchat/router.go
      Note: |-
        Pinocchio router now delegates request policy for /ws and /chat*
        Pinocchio Router delegates request policy + filters tools
ExternalSources: []
Summary: 'Engineering postmortem for GP-09: extracted request policy into a request-facing builder and applied the pattern in Pinocchio + Moments; added tests and commits; documented scope pivot away from legacy go-go-mento.'
LastUpdated: 2026-01-23T13:31:19.443199991-05:00
WhatFor: Use as the detailed engineering record of what changed (and why) for GP-09, including commits, tests, review checklist, and risks to double-check.
WhenToUse: Use when reviewing or extending the GP-09 refactor, debugging profile selection drift, or implementing similar request-policy extraction patterns elsewhere.
---


# Postmortem

## Goal

This document is the detailed engineering postmortem for ticket `GP-09-PROFILE-ENGINE-BUILDER`. It is meant to be the “reviewable narrative” version of the diary: what we changed, why, what failed along the way, what shipped (with exact commits), how to validate, and what a reviewer should scrutinize.

## Context

GP-09 started as a design analysis for extracting “engine/profile policy” out of webchat Routers behind a request-facing interface (`BuildEngineFromReq`) and a profile-based engine builder (`profileEngineBuilder`). The analysis used go-go-mento as the “historical” reference implementation, but you explicitly redirected scope to the maintained codebases: **Pinocchio** and **Moments**.

The implementation work therefore:

- Avoids changes to legacy `go-go-mento/`.
- Applies Phase 1 (request policy extraction) in **Pinocchio** and **Moments**.
- In **Pinocchio**, also implements the config-driven tools and override enforcement portions because Pinocchio already has an explicit EngineBuilder shape and an engine signature seam.

## Quick Reference

### Shipped artifacts (commits)

- Pinocchio: `3b8cae7` — `webchat: factor request policy and tool selection`
- Moments: `fe3e9dcf` — `webchat: extract request policy builder`
- Ticket docs (Geppetto): `d7fddd4` — `GP-09: record implementation diary and close tasks`
- Ticket docs (Geppetto): `4b93804` — `GP-09: add diary bookkeeping step`

### Validation commands

```bash
cd pinocchio && go test ./... -count=1
cd moments/backend && go test ./... -count=1
```

### New key abstractions

**Pinocchio**

- `EngineFromReqBuilder` (request-facing policy):
  - Input: `*http.Request`
  - Output: `EngineBuildInput{ConvID, ProfileSlug, Overrides}` + `*ChatRequestBody` (for HTTP)
  - Error: `*RequestBuildError{Status, ClientMsg, Err}` (typed for handler use)
- `EngineConfig.Tools []string` is now part of the config and included in `EngineConfig.Signature()`.

**Moments**

- `EngineFromReqBuilder` mirrors the same concept, but is adapted to Moments’ profile registry type.
- Moments keeps token-injection and `draft_bundle_id` parsing in the WS handler (policy builder remains focused on conv/profile/overrides).

### Request policy precedence rules (as implemented)

These are the precedence rules to verify in review.

**Pinocchio**

- HTTP `/chat*`:
  1. explicit profile via path `/chat/{profile}`
  2. existing conversation profile (if same `conv_id`)
  3. cookie `chat_profile`
  4. default `"default"`
- WS `/ws`:
  1. query `profile=...`
  2. cookie `chat_profile`
  3. existing conversation profile
  4. default `"default"`

**Moments**

- HTTP `/chat*` (builder):
  1. explicit profile via `profileSlugFromRequest(req)` (query first, then path)
  2. cookie `chat_profile`
  3. existing conversation profile (if same `conv_id`)
  4. default `"default"`
- WS `/ws` (builder):
  1. query `profile=...`
  2. cookie `chat_profile`
  3. existing conversation profile
  4. default `"default"`

Note: These rules are the most likely source of subtle behavior differences; reviewers should verify the intended ordering for your UI/client behavior.

## Detailed Engineering Narrative

### What we were trying to fix (root problem)

Across these codebases, webchat Routers were mixing multiple responsibilities:

1. Request policy: “How do we derive `conv_id`, `profileSlug`, and `overrides` from this HTTP/WS request?”
2. Engine/profile config policy: “How do profile defaults + overrides become a deterministic engine config?”
3. Runtime wiring: “How do we build a registry of tools/middlewares and start the run loop?”

This coupling leads to:

- Duplicated precedence logic between HTTP and WS (or between `/chat` and `/chat/{profile}` paths).
- Drift between config (what the signature says) and runtime behavior (what tools are actually available).
- Harder-to-test behavior: request policy was embedded in handlers rather than in a unit-testable component.

### Scope pivot (go-go-mento rollback)

Implementation started in go-go-mento because GP-09’s analysis references it, but you explicitly redirected focus to Pinocchio and Moments. The go-go-mento edits were rolled back completely and were not committed.

This pivot is recorded in the diary and in Step 5 notes, including:

- The initial `go test` workspace/module mismatch encountered in go-go-mento.
- A policy rejection preventing a single `git restore && rm` rollback command.
- Full rollback performed via patches instead.

### Pinocchio implementation details (commit 3b8cae7)

#### Phase 1: Extract request policy (BuildEngineFromReq-style)

Changes:

- Added `pkg/webchat/engine_from_req.go` with:
  - `EngineFromReqBuilder` interface
  - `DefaultEngineFromReqBuilder` implementation
  - `RequestBuildError` typed error
  - `EngineBuildInput` + `ChatRequestBody`
- Added a `ConversationLookup` interface and implemented `ConvManager.GetConversation` so request policy can consult the existing conversation profile without pulling in Router internals.
- Router now:
  - Uses the builder for `/ws`
  - Uses a unified handler path for both `/chat` and `/chat/{profile}` (handled as `/chat` and `/chat/` patterns)

Rationale:

- Centralizes and tests precedence rules.
- Ensures request body parsing happens once and remains consistent.

#### Phase 2: Tools are driven by config

Changes:

- Added `Tools []string` to `EngineConfig` and `engineConfigSignature` in `pkg/webchat/engine_config.go`.
- Included `Tools` in `EngineConfig.Signature()` so rebuild decisions and runtime tool registry can’t drift.
- Updated `BuildConfig` to derive `Tools` from:
  - `Profile.DefaultTools`
  - optional `overrides["tools"]` (parsed/validated)
- Updated `/chat` handler to build the tool registry filtered by `EngineConfig.Tools`.

Rationale:

- Previously, tools were effectively “side config” and could diverge from the signature seam.
- By including tools in config/signature and filtering the registry by config, you can reason about tool availability as part of the engine configuration.

#### Phase 2: Enforce `Profile.AllowOverrides`

Changes:

- Added `validateOverrides(p, overrides)` in `pkg/webchat/engine_builder.go`:
  - Rejects `system_prompt`, `middlewares`, or `tools` overrides when `AllowOverrides=false`.
  - Validates override types.

Note:

- `step_mode` is not treated as an engine-shaping override and is intentionally handled elsewhere (the run loop) so it does not get blocked by `AllowOverrides`.

#### Tests added/updated

- `pkg/webchat/engine_from_req_test.go`:
  - Profile precedence tests (HTTP + WS)
  - conv_id generation test (UUID)
- `pkg/webchat/engine_config_test.go`:
  - Signature changes when `Tools` changes (in addition to prior determinism checks)

#### Notable operational detail

Pinocchio’s pre-commit hook runs frontend build + npm audit output. The commit output reported npm vulnerabilities; this postmortem does not address those because they are unrelated to the webchat refactor, but they are worth tracking separately.

### Moments implementation details (commit fe3e9dcf)

#### Phase 1: Extract request policy (BuildEngineFromReq-style)

Changes:

- Added `backend/pkg/webchat/engine_from_req.go` with:
  - `EngineFromReqBuilder` interface
  - `DefaultEngineFromReqBuilder` implementation
  - `RequestBuildError` typed error
  - `EngineBuildInput`
  - `ConversationLookup` interface for consulting existing conversations
- Added `engineFromReqBuilder` field to Router (`backend/pkg/webchat/types.go`) and initialized it in `NewRouterFromRegistries`.
- Updated `backend/pkg/webchat/router.go`:
  - WS join now uses `engineFromReqBuilder.BuildEngineFromReq` to resolve `conv_id` and `profileSlug`.
    - Token injection and `draft_bundle_id` remain in WS handler (not moved into request builder).
  - `/chat` and `/chat/{profile}` handlers now call `handleChatRequest(w, req)`, and `handleChatRequest` delegates request policy to the builder.

#### Tests added

- `backend/pkg/webchat/engine_from_req_test.go`:
  - Profile precedence tests (HTTP + WS)
  - conv_id generation test

#### Intentional non-changes (scope)

Unlike Pinocchio, Moments does not currently have:

- a profile-level `AllowOverrides` flag in the profile registry descriptor
- a first-class `EngineConfig` type/signature seam that includes tools/settings metadata

So this step stops at Phase 1 (request policy extraction + tests) and does not introduce “implicit support” for engine-shaping override enforcement or tool overrides.

## What went wrong / what was tricky

1. **Scope change mid-flight**: starting from go-go-mento (legacy) and pivoting to maintained repos required a careful rollback to avoid leaving accidental changes behind.
2. **Rollback tooling restriction**: one rollback command was blocked by policy; rollback proceeded via patches instead.
3. **Pinocchio handler duplication**: `/chat` and `/chat/{profile}` logic was duplicated; unifying it required careful `req.Body` usage and ensuring profile parsing remained consistent.
4. **Signature drift risk**: ensuring the engine config signature matched runtime tool availability required adding `Tools` into the signature and filtering the registry accordingly.

## What to review closely (high-signal)

1. **Precedence rules**:
   - Pinocchio: `pinocchio/pkg/webchat/engine_from_req.go`
   - Moments: `moments/backend/pkg/webchat/engine_from_req.go`
   Verify the explicit/cookie/existing/default order matches intended UI behavior for both HTTP and WS.

2. **Override enforcement boundary (Pinocchio only)**:
   - `pinocchio/pkg/webchat/engine_builder.go`
   Confirm that only “engine-shaping” overrides are blocked when `AllowOverrides=false` and that this doesn’t block legitimate runtime-only flags.

3. **Tools vs signature linkage (Pinocchio only)**:
   - `pinocchio/pkg/webchat/engine_config.go`
   - `pinocchio/pkg/webchat/router.go`
   Confirm the `Tools` list is fully represented in the signature and the tool registry filtering uses the same list.

4. **WS behavior in Moments**:
   - `moments/backend/pkg/webchat/router.go`
   Confirm `draft_bundle_id` and bearer token injection behavior is unchanged by the refactor (only profile/conv selection moved).

5. **Tests reflect intended policy**:
   - `pinocchio/pkg/webchat/engine_from_req_test.go`
   - `moments/backend/pkg/webchat/engine_from_req_test.go`
   Sanity-check that tests encode the precedence you want (it’s easy for “expected” behavior to drift into being “tested” behavior).

## Usage Examples

### How to validate locally

```bash
cd pinocchio
go test ./... -count=1

cd ../moments/backend
go test ./... -count=1
```

### How to spot-check behavior manually

**Pinocchio**

1. Start the webchat server and hit `/chat` without specifying a profile; confirm cookie/existing/default behavior.
2. Hit `/chat/{profile}` and confirm it overrides cookie/existing.
3. WS connect with `?profile=...` and confirm it overrides cookie/existing.
4. Provide `overrides.tools` and confirm the tool registry is filtered accordingly (and triggers rebuild due to signature change).

**Moments**

1. WS connect with and without `?profile=...` and confirm profile selection matches previous behavior.
2. Confirm bearer token injection via query param still works the same (frontend limitation).

## Related

- GP-09 analysis: `geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/analysis/01-extract-profile-engine-builder-out-of-router.md`
- GP-09 diary: `geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md`
