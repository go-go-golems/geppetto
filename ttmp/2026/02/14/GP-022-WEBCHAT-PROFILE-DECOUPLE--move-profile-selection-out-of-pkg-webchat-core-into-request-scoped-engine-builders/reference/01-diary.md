---
Title: Diary
Ticket: GP-022-WEBCHAT-PROFILE-DECOUPLE
Status: active
Topics:
    - architecture
    - pinocchio
    - chat
    - migration
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/design-doc/01-profile-decoupling-analysis-and-migration-plan.md
      Note: Design source used to derive execution task phases
    - Path: geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/tasks.md
      Note: Detailed execution checklist for implementation slices
ExternalSources: []
Summary: Implementation diary for resolver-plan cutover and profile decoupling work.
LastUpdated: 2026-02-14T16:46:37.715353859-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Track implementation of GP-022 as an execution log: what changed, why it changed, test outcomes, commits, and follow-up risks.

## Step 1: Execution Setup and Task Breakdown

I started this implementation phase by turning the ticket into an executable work plan rather than a high-level design ticket. The immediate objective was to establish strict task sequencing and tracking so code migration could proceed slice-by-slice with tests and commits.

This step focused on process readiness, not code behavior changes. I created a detailed phased checklist and initialized a diary that will capture every implementation slice, including failures and validation commands.

### Prompt Context

**User prompt (verbatim):** "Ok, create a detailed list of tasks in the ticket now, and then start working on them one by one,testing as you go, committing, checking things off, and updating the diary."

**Assistant interpretation:** Create a granular implementation task list, then execute migration tasks iteratively with tests, commits, checkbox updates, and diary updates.

**Inferred user intent:** Move from planning to active delivery with disciplined execution tracking and auditable progress.

**Commit (code):** N/A (task + diary scaffolding stage)

### What I did
- Created a `reference` diary document for GP-022 via `docmgr doc add`.
- Replaced the placeholder task list with a detailed phased checklist in `tasks.md`:
  - core resolver-plan cutover
  - app migrations (`cmd/web-chat`, `web-agent-example`)
  - testing, docs, and completion loop requirements
- Updated the active implementation plan/status to start Phase 1 next.

### Why
- The migration is broad and cross-repo. A detailed ordered checklist is required to avoid hidden dependency mistakes and to keep commits scoped.
- The user explicitly requested task-by-task execution with diary updates and test evidence.

### What worked
- Ticket workspace now contains a concrete task map that can be checked off incrementally.
- Diary scaffolding is in place and ready for per-slice updates.

### What didn't work
- N/A in this step.

### What I learned
- The ticket had strong design coverage but lacked execution granularity; converting it to phased tasking materially reduced ambiguity for the implementation sequence.

### What was tricky to build
- The trickiest part in this step was choosing task granularity that is neither too broad (untrackable) nor too fragmented (administrative overhead). I resolved this by structuring tasks as phase-level deliverables plus explicit per-slice loop requirements.

### What warrants a second pair of eyes
- The phase boundaries in `tasks.md` should be reviewed once Phase 1 starts, in case some core/API tasks need to be reordered after compile/test feedback.

### What should be done in the future
- Begin Phase 1 core refactor immediately and track each code slice with tests and commit hashes in subsequent diary steps.

### Code review instructions
- Start with `tasks.md` in the GP-022 workspace and confirm phases/tasks align with the latest design decisions.
- Validate that diary process requirements are explicit before code changes start.

### Technical details
- Files updated:
  - `geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/tasks.md`
  - `geppetto/ttmp/2026/02/14/GP-022-WEBCHAT-PROFILE-DECOUPLE--move-profile-selection-out-of-pkg-webchat-core-into-request-scoped-engine-builders/reference/01-diary.md`

## Step 2: Request Resolver API Cutover (Slice 1)

This step implemented the first code slice of the migration: replacing the old `BuildEngineFromReq` contract with a new resolver-based request plan API in core webchat, then updating `web-agent-example` to use it. I intentionally preserved existing behavior while changing the contract shape so downstream slices can remove profile semantics with less churn.

The primary objective was to establish one consistent request entry point (`Resolve(req)`) for both HTTP chat and websocket attachment flows. This moved runtime request data into a single plan object and removed dependency on returning parsed body pointers through the old builder interface.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Begin implementation, execute one slice at a time, test and commit each slice, and keep ticket diary/tasks synchronized.

**Inferred user intent:** Start real migration work immediately and make progress auditable via commits, tests, and diary records.

**Commit (code):** `7993131` — "webchat: replace BuildEngineFromReq with request resolver plan API"  
**Commit (code):** `0c998fb` — "web-agent-example: switch to ConversationRequestResolver API"

### What I did
- In `pinocchio/pkg/webchat/engine_from_req.go`:
  - replaced `EngineFromReqBuilder` with `ConversationRequestResolver`.
  - introduced `ConversationRequestPlan` with `ConvID`, `RuntimeKey`, `Overrides`, `Prompt`, `IdempotencyKey`.
  - replaced `RequestBuildError` with `RequestResolutionError`.
  - migrated default implementation from `BuildEngineFromReq` to `Resolve`.
- In `pinocchio/pkg/webchat/types.go`:
  - replaced router field `engineFromReqBuilder` with `requestResolver`.
- In `pinocchio/pkg/webchat/router_options.go`:
  - replaced `WithEngineFromReqBuilder(...)` with `WithConversationRequestResolver(...)`.
- In `pinocchio/pkg/webchat/router.go`:
  - switched WS/chat handlers to call `resolver.Resolve(req)`.
  - removed dependency on resolver returning parsed request body.
  - adapted idempotency/prompt handling to read from request plan.
- In `pinocchio/pkg/webchat/engine_from_req_test.go`:
  - updated tests to validate resolver-plan outputs and new error type.
- In `web-agent-example/cmd/web-agent-example/engine_from_req.go`:
  - migrated custom builder to implement `Resolve(req)` and return `ConversationRequestPlan`.
- In `web-agent-example/cmd/web-agent-example/main.go`:
  - updated router option call to `WithConversationRequestResolver(...)`.
- Ran `gofmt` across changed files.

### Why
- This is the first mandatory cutover step toward the new architecture and clean retirement of `BuildEngineFromReq`.
- It enables subsequent profile-removal work to happen in a single resolver surface instead of dual contracts.

### What worked
- `pinocchio/pkg/webchat` test suite passed after migration.
- `pinocchio/cmd/web-chat/...` package checks passed.
- Both codebases committed cleanly with scoped messages.
- `pinocchio` pre-commit hook completed successfully (`go test ./...`, lint/generate/vet sequence).

### What didn't work
- Running `go test ./...` in `web-agent-example` failed due baseline module dependency issues unrelated to this slice:
  - `no required module provides package github.com/go-go-golems/geppetto/pkg/layers`
  - `no required module provides package github.com/go-go-golems/glazed/pkg/cmds/layers`
  - `no required module provides package github.com/go-go-golems/glazed/pkg/cmds/parameters`
- Command used:
  - `go test ./...` (in `web-agent-example/`)

### What I learned
- The API contract swap can be landed without destabilizing core behavior when treated as a transport/policy adapter layer first.
- The web-agent-example module has existing dependency resolution drift that limits end-to-end package testing in this workspace state.

### What was tricky to build
- The main tricky part was keeping router behavior stable while removing parsed-body return semantics from the old builder API.  
  Symptoms: chat handler previously depended on `body` for prompt/idempotency, and WS/chat handlers used old typed error names.  
  Resolution: moved prompt/idempotency into `ConversationRequestPlan`, updated resolver errors to `RequestResolutionError`, and adjusted handlers to use plan fields directly.

### What warrants a second pair of eyes
- The default resolver currently still uses profile semantics internally (intentionally for this first slice). Reviewers should verify this transitional state is acceptable until the next slice removes profile concepts from core.
- The naming `RuntimeKey` currently maps to profile slug in default resolver; this is transitional and should be audited as profile removal proceeds.

### What should be done in the future
- Next slice: remove profile concepts from core (`Profile`, `ProfileRegistry`, core profile endpoints) and migrate core defaults to profile-agnostic behavior.

### Code review instructions
- Start with:
  - `pinocchio/pkg/webchat/engine_from_req.go`
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/router_options.go`
  - `web-agent-example/cmd/web-agent-example/engine_from_req.go`
- Validate with:
  - `go test ./pkg/webchat/...` in `pinocchio/`
  - `go test ./cmd/web-chat/...` in `pinocchio/`
- Note expected baseline failure in `web-agent-example` full test command due unresolved module deps in this workspace.

### Technical details
- Commands run:
  - `go test ./pkg/webchat/...` (`pinocchio`) -> pass
  - `go test ./cmd/web-chat/...` (`pinocchio`) -> pass
  - `go test ./...` (`web-agent-example`) -> baseline dependency failure (see above)
- Files changed in this slice:
  - `pinocchio/pkg/webchat/engine_from_req.go`
  - `pinocchio/pkg/webchat/engine_from_req_test.go`
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/router_options.go`
  - `pinocchio/pkg/webchat/types.go`
  - `web-agent-example/cmd/web-agent-example/engine_from_req.go`
  - `web-agent-example/cmd/web-agent-example/main.go`

## Step 3: Core Request Policy Becomes Runtime-Key Generic

This slice removed additional profile-specific behavior from core request handling. Core resolver no longer depends on `chat_profile` cookies or core-managed profile API endpoints, and debug API payloads now expose a generic runtime identity key.

The result is a clearer separation: core handles request resolution into a runtime key and execution inputs, while app-level code becomes the proper place for profile UX endpoints/policy.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue implementation slices with testing, commits, task checks, and diary updates.

**Inferred user intent:** Progressively eliminate core profile coupling while keeping builds/tests stable and traceable.

**Commit (code):** `292abe4` — "webchat: drop core profile endpoints and cookie-based request policy"  
**Commit (code):** `1892538` — "web-agent-example: simplify resolver to runtime-key-only policy"

### What I did
- In `pinocchio/pkg/webchat/engine_from_req.go`:
  - removed profile registry dependency from default resolver.
  - removed cookie-based fallback (`chat_profile`) from request resolution.
  - switched WS/query key from profile-centric to runtime-centric (`runtime`).
  - removed profile existence validation in core resolver.
- In `pinocchio/pkg/webchat/router.go`:
  - removed core `/api/chat/profiles` and `/api/chat/profile` handlers.
  - updated WS request logging to `runtime_query`.
  - kept resolver integration in place with new generic behavior.
- In `pinocchio/pkg/webchat/router_debug_routes.go` and tests:
  - changed debug response field from `profile` to `runtime_key`.
- In `web-agent-example/cmd/web-agent-example/engine_from_req.go`:
  - removed stale profile parsing no-ops and kept strict default runtime-key policy.

### Why
- This slice directly addresses the boundary issue where core owned profile endpoints and cookie semantics.
- It prepares the codebase for full removal of profile types/registry from core by reducing active dependencies first.

### What worked
- `go test ./pkg/webchat/...` passed after changes.
- `go test ./cmd/web-chat/...` passed.
- Core pre-commit hook passed (`go test ./...`, generate/build/lint/vet in `pinocchio`).

### What didn't work
- `go test ./cmd/web-agent-example` remains blocked by pre-existing module dependency setup issues:
  - missing `github.com/go-go-golems/geppetto/pkg/layers`
  - missing `github.com/go-go-golems/glazed/pkg/cmds/layers`
  - missing `github.com/go-go-golems/glazed/pkg/cmds/parameters`

### What I learned
- Removing profile HTTP endpoints and cookie fallback from core is low-risk when done after the resolver contract cutover.
- Debug payload field rename can be done backend-first while frontend mapping is updated in later app slices.

### What was tricky to build
- The tricky part was keeping behavior coherent while changing both request policy and debug schema in one slice.  
  Symptoms: tests and handler assumptions still referred to profile naming and cookie behavior.  
  Resolution: updated resolver tests to runtime precedence rules, removed endpoint handlers in one patch, and renamed debug response keys with matching test updates.

### What warrants a second pair of eyes
- Whether default resolver should keep accepting legacy `profile` query as temporary alias (currently no; only `runtime`).
- Whether WS hello payload `profile` field should be renamed in the immediate next slice to avoid mixed terminology.

### What should be done in the future
- Next slice should remove core `Profile`/`ProfileRegistry`/`AddProfile` surfaces and migrate `cmd/web-chat` to app-owned profile registry and handlers.

### Code review instructions
- Review files in this order:
  - `pinocchio/pkg/webchat/engine_from_req.go`
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/router_debug_routes.go`
  - `pinocchio/pkg/webchat/router_debug_api_test.go`
- Validate with:
  - `go test ./pkg/webchat/...` in `pinocchio/`
  - `go test ./cmd/web-chat/...` in `pinocchio/`

### Technical details
- Commands run:
  - `go test ./pkg/webchat/...` (`pinocchio`) -> pass
  - `go test ./cmd/web-chat/...` (`pinocchio`) -> pass
  - `go test ./cmd/web-agent-example` (`web-agent-example`) -> baseline setup failure
- Files changed in this slice:
  - `pinocchio/pkg/webchat/engine_from_req.go`
  - `pinocchio/pkg/webchat/engine_from_req_test.go`
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/router_debug_routes.go`
  - `pinocchio/pkg/webchat/router_debug_api_test.go`
  - `web-agent-example/cmd/web-agent-example/engine_from_req.go`
