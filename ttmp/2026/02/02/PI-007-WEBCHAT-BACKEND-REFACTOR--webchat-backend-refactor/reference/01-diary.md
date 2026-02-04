---
Title: Diary
Ticket: PI-007-WEBCHAT-BACKEND-REFACTOR
Status: active
Topics:
    - webchat
    - backend
    - bugfix
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/README.md
      Note: |-
        Note API/UI handlers (commit 94f8d20)
        Update conv manager + ordering notes (commit 1828999)
        Eviction docs (commit 9c8adce)
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Eviction CLI flags (commit 9c8adce)
    - Path: pinocchio/pkg/doc/topics/webchat-backend-internals.md
      Note: Update pool concurrency notes (commit 011c824)
    - Path: pinocchio/pkg/doc/topics/webchat-backend-reference.md
      Note: Document non-blocking pool (commit 011c824)
    - Path: pinocchio/pkg/doc/topics/webchat-framework-guide.md
      Note: |-
        Document handler split/mount pattern (commit 94f8d20)
        Document seq/stream_id ordering (commit 1828999)
    - Path: pinocchio/pkg/doc/topics/webchat-sem-and-ui.md
      Note: Document seq-based versioning for streaming/hydration (commit 4964d10)
    - Path: pinocchio/pkg/doc/topics/webchat-user-guide.md
      Note: Eviction doc section (commit 9c8adce)
    - Path: pinocchio/pkg/webchat/connection_pool.go
      Note: Non-blocking pool implementation (commit 011c824)
    - Path: pinocchio/pkg/webchat/connection_pool_test.go
      Note: Backpressure drop test (commit 011c824)
    - Path: pinocchio/pkg/webchat/conv_manager_eviction.go
      Note: Eviction loop implementation (commit 9c8adce)
    - Path: pinocchio/pkg/webchat/conv_manager_eviction_test.go
      Note: Eviction tests (commit 9c8adce)
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: |-
        ConvManager lifecycle extraction (commit 2a29380)
        Track lastActivity for eviction (commit 9c8adce)
    - Path: pinocchio/pkg/webchat/router.go
      Note: |-
        Use StripPrefix when mounting webchat under a subpath (commit bf2c934)
        Split UI/API handlers and fs.FS usage (commit 94f8d20)
        Router delegates conversation lifecycle to manager (commit 2a29380)
        Router delegates queue prep/drain (commit 51929ea)
        Configure eviction and update activity (commit 9c8adce)
    - Path: pinocchio/pkg/webchat/router_handlers_test.go
      Note: UI/API handler tests (commit 94f8d20)
    - Path: pinocchio/pkg/webchat/router_mount_test.go
      Note: Mount/redirect tests for subpath integration (commit bf2c934)
    - Path: pinocchio/pkg/webchat/send_queue.go
      Note: Queue/idempotency helpers (commit 51929ea)
    - Path: pinocchio/pkg/webchat/send_queue_test.go
      Note: Queue helper tests (commit 51929ea)
    - Path: pinocchio/pkg/webchat/server.go
      Note: NewServer accepts fs.FS (commit 94f8d20)
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Derive seq from Redis stream IDs (commit 1828999)
    - Path: pinocchio/pkg/webchat/stream_coordinator_test.go
      Note: Stream ordering tests (commit 1828999)
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Propagate event.seq through timeline projection (commit 4964d10)
    - Path: pinocchio/pkg/webchat/timeline_store.go
      Note: Require seq-based versions in timeline upserts (commit 4964d10)
    - Path: pinocchio/pkg/webchat/types.go
      Note: Router staticFS now fs.FS (commit 94f8d20)
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-03T19:53:36.549345638-05:00
WhatFor: ""
WhenToUse: ""
---









# Diary

## Goal

Track the webchat backend refactor work for PI-007, including decisions, changes, test runs, and follow-up risks.

## Step 1: Fix Router Mount Prefix Handling

Updated the webchat Router mount behavior to correctly strip subpaths, allowing it to be embedded under prefixes like `/api/webchat`. Added focused tests for the prefix mount and base-path redirect to make the regression hard to reintroduce.

This unblocks embedding without requiring a full refactor, and gives a safe starting point for subsequent API/UI splitting.

### Prompt Context

**User prompt (verbatim):** "Work on PI-007-WEBCHAT-BACKEND-REFACTOR with docmgr.

then update PI-007-WEBCHAT-BACKEND-REFACTOR and import the file /tmp/webchat-backend-refactor.md with `docmgr import file --file ... --ticket ...` and then read it and create a detailed list of tasks for the implementation of the refactor.

Then, work on the tasks one by one, working, keeping a frequent detailed diary, checking the task off, committing to git, etc..."

**Assistant interpretation:** Import the refactor blueprint into docmgr, build a detailed task list, and begin implementing tasks one by one with commits and diary updates.

**Inferred user intent:** Establish a structured, auditable refactor plan and see incremental progress with traceable changes.

**Commit (code):** bf2c934 — "Fix webchat mount prefix handling"

### What I did
- Updated `Router.Mount` to use `http.StripPrefix`, with a base-path redirect for the non-trailing-slash mount.
- Added mount tests covering prefix strip and redirect behavior.
- Ran `go test ./pinocchio/pkg/webchat -count=1` locally; pre-commit ran `go test ./...`, `go generate ./...`, and `golangci-lint`.

### Why
- `http.ServeMux` does not strip prefixes; mounting at `/api/webchat` previously broke route matching.
- A redirect for the bare prefix avoids ambiguous 404s when users hit `/api/webchat` instead of `/api/webchat/`.

### What worked
- New tests confirm the handler is reachable at `/api/webchat/chat` and the base path redirects.
- Pre-commit checks passed after the successful commit.

### What didn't work
- Initial `git commit -m "Fix webchat mount prefix handling"` timed out after 10s while pre-commit hooks ran `go test ./...` (command timeout).

### What I learned
- The repo’s pre-commit hook runs `go test ./...` and frontend build steps, so commits need longer timeouts.

### What was tricky to build
- Ensuring `http.ServeMux` pattern matching and redirect semantics stay correct for both `"/"` and non-root prefixes without breaking the existing in-process router usage.

### What warrants a second pair of eyes
- Confirm the redirect status code (`308`) is acceptable for existing clients and does not interfere with websocket upgrade flows.

### What should be done in the future
- N/A

### Code review instructions
- Start at `pinocchio/pkg/webchat/router.go` and `pinocchio/pkg/webchat/router_mount_test.go`.
- Validate with `go test ./pinocchio/pkg/webchat -count=1` (or run the full pre-commit hooks if desired).

### Technical details
- `Router.Mount` now strips prefixes and adds a redirect for the base path to avoid routing mismatches.

## Step 2: Split UI and API Handlers

Separated UI asset serving from API/websocket endpoints by introducing `APIHandler()` and `UIHandler()` on the router, while keeping the default `Router` composition intact. This makes it possible to mount UI and API on different paths or hosts without losing existing behavior.

I also generalized the static filesystem type to `fs.FS` (instead of `embed.FS`) so tests can use `fstest.MapFS`, and updated docs to describe the new handler split and the improved mount pattern.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the next refactor task by splitting UI and API concerns and documenting the new integration patterns.

**Inferred user intent:** Make the backend easier to embed and reuse, with clear separation between UI assets and API endpoints.

**Commit (code):** 94f8d20 — "Split webchat UI and API handlers"

### What I did
- Split `registerHTTPHandlers` into `registerUIHandlers` and `registerAPIHandlers`, with `APIHandler()` / `UIHandler()` accessors.
- Switched the static FS type to `fs.FS` and used `fs.ReadFile` to allow non-embed test FS.
- Added tests for UI index serving and API handler isolation.
- Updated webchat docs to show the new mount pattern and handler split usage.
- Ran `go test ./pinocchio/pkg/webchat -count=1`; pre-commit ran repo-wide tests, codegen, and lint.

### Why
- Making UI serving optional improves composability and lets hosts serve UI separately from API/WS endpoints.
- Using `fs.FS` enables lightweight tests without forcing embed-only FS types.

### What worked
- Tests validate UI handler index serving and API handler non-responsiveness for `/`.
- Pre-commit checks passed after the commit.

### What didn't work
- N/A

### What I learned
- Switching to `fs.FS` is low-impact and makes handlers more testable without changing call sites.

### What was tricky to build
- Ensuring the new handlers preserve the existing default behavior while allowing API/UI separation without duplicating logic.

### What warrants a second pair of eyes
- Confirm that the `fs.FS` change doesn’t break any downstream callers relying on `embed.FS` specifics.

### What should be done in the future
- N/A

### Code review instructions
- Start at `pinocchio/pkg/webchat/router.go`, `pinocchio/pkg/webchat/types.go`, and `pinocchio/pkg/webchat/router_handlers_test.go`.
- Validate with `go test ./pinocchio/pkg/webchat -count=1`.

### Technical details
- `APIHandler()` and `UIHandler()` wrap dedicated muxes, while `registerHTTPHandlers()` still composes both for default usage.

## Step 3: Extract Conversation Manager Lifecycle

Moved conversation lifecycle ownership into `ConvManager` by adding config hooks and new `GetOrCreate` / `AddConn` / `RemoveConn` methods. The router now delegates to the manager rather than embedding lifecycle logic itself.

This centralizes state and keeps router handlers focused on HTTP/WS concerns, preparing the ground for later refactor steps like queue extraction and eviction.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the next refactor task by moving conversation lifecycle control into the manager.

**Inferred user intent:** Keep responsibilities cleanly separated and reduce router coupling to conversation internals.

**Commit (code):** 2a29380 — "Refactor conversation manager lifecycle"

### What I did
- Added `ConvManagerOptions`, lifecycle hooks, and `GetOrCreate`/`AddConn`/`RemoveConn` methods in `conversation.go`.
- Updated `NewRouter` to construct the manager with injected build/timeline hooks.
- Routed WS/chat handlers and debug endpoints through manager methods.
- Ran `go test ./pinocchio/pkg/webchat -count=1`; pre-commit ran repo-wide tests, codegen, and lint.

### Why
- Moving lifecycle logic into a dedicated manager reduces router responsibilities and makes future refactors (eviction, queue extraction) cleaner.

### What worked
- The new manager retains existing behavior while keeping router handlers thin.
- Tests and pre-commit checks passed after the change.

### What didn't work
- N/A

### What I learned
- The existing build hooks (`BuildConfig`, `BuildFromConfig`, subscriber creation) are easy to inject into a manager without changing call sites.

### What was tricky to build
- Preserving timeline projector wiring and stream restart logic while moving code out of the router.

### What warrants a second pair of eyes
- Validate that all manager dependency hooks are set before `GetOrCreate` is ever called, especially in custom Router constructions.

### What should be done in the future
- N/A

### Code review instructions
- Start at `pinocchio/pkg/webchat/conversation.go` and `pinocchio/pkg/webchat/router.go`.
- Validate with `go test ./pinocchio/pkg/webchat -count=1`.

### Technical details
- `ConvManager` now owns conversation creation and stream wiring via injected hooks and uses `SetIdleTimeoutSeconds` / `SetTimelineStore` for configuration updates.

## Step 4: Move Queue/Idempotency Logic into Conversation Helpers

Shifted the chat queue and idempotency logic out of `router.go` into conversation-level helpers (`PrepareRun`, `ClaimNextQueued`), and added unit tests for idempotency replay, queueing, and drain behavior. The router now just orchestrates between preparation and `startRunForPrompt`.

This isolates queue behavior into testable helpers and reduces handler complexity.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the next refactor task by pulling queue/idempotency logic into conversation helpers and covering it with tests.

**Inferred user intent:** Make queueing behavior reliable and easier to reason about while shrinking router responsibilities.

**Commit (code):** 51929ea — "Move chat queue logic into conversation helpers"

### What I did
- Added `PrepareRun` and `ClaimNextQueued` helpers to `send_queue.go`.
- Updated `/chat` handling and queue draining to call those helpers.
- Added unit tests covering idempotent replay, queueing when busy, immediate start, and queue drain.
- Ran `go test ./pinocchio/pkg/webchat -count=1`; pre-commit ran repo-wide tests, codegen, and lint.

### Why
- Encapsulating queue/idempotency logic reduces router surface area and makes queue behavior directly testable.

### What worked
- Tests verify queue behavior without requiring full HTTP/WS integration.
- Pre-commit checks passed after the change.

### What didn't work
- N/A

### What I learned
- Minimal helper APIs (`PrepareRun`, `ClaimNextQueued`) are sufficient to cover queueing and idempotency flows.

### What was tricky to build
- Preserving response semantics (queued vs running) while keeping router responses intact.

### What warrants a second pair of eyes
- Confirm `PrepareRun` preserves all previously returned fields for queued/running responses.

### What should be done in the future
- N/A

### Code review instructions
- Start at `pinocchio/pkg/webchat/send_queue.go`, `pinocchio/pkg/webchat/router.go`, and `pinocchio/pkg/webchat/send_queue_test.go`.
- Validate with `go test ./pinocchio/pkg/webchat -count=1`.

### Technical details
- `PrepareRun` handles idempotency replay, queue enqueue, and running slot claims in one place.

## Step 5: Derive Stream Sequence from Redis IDs

Updated the stream coordinator to derive the `seq` cursor from Redis stream IDs when available, and added unit tests to verify both derived and fallback sequencing. Documentation was updated to describe the new ordering guarantees and the `stream_id`/`seq` fields.

This improves ordering stability across restarts and aligns the backend with the refactor plan.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the stream ordering task and update docs/tests accordingly.

**Inferred user intent:** Ensure streaming order is stable and well-documented, without carrying legacy behavior.

**Commit (code):** 1828999 — "Derive stream seq from Redis IDs"

### What I did
- Derived `seq` from Redis stream IDs (`xid`/`redis_xid`) when present, with fallback to the local counter.
- Removed legacy metadata key extraction (`stream_id`/`redis_stream_id`) to avoid backward-compat shims.
- Added tests for `deriveSeqFromStreamID` and stream coordinator sequencing.
- Updated webchat docs to describe `seq` and `stream_id` behavior.
- Ran `go test ./pinocchio/pkg/webchat -count=1`; pre-commit ran repo-wide tests, codegen, and lint.

### Why
- Redis stream IDs provide a monotonic order source that remains stable across restarts.
- Removing legacy metadata keys keeps the implementation focused and predictable.

### What worked
- Tests confirm derived sequencing and fallback behavior.
- Documentation now reflects the stable ordering contract.

### What didn't work
- N/A

### What I learned
- Normalizing Redis stream IDs into a numeric sequence keeps ordering monotonic without changing the SEM envelope shape.

### What was tricky to build
- Ensuring the global sequence counter advances when a derived sequence is higher, without reordering or race risks.

### What warrants a second pair of eyes
- Validate the derived sequence formula (`ms*1_000_000 + seq`) is acceptable for all Redis stream ID sizes in use.

### What should be done in the future
- N/A

### Code review instructions
- Start at `pinocchio/pkg/webchat/stream_coordinator.go` and `pinocchio/pkg/webchat/stream_coordinator_test.go`.
- Validate with `go test ./pinocchio/pkg/webchat -count=1`.

### Technical details
- `deriveSeqFromStreamID` parses `<ms>-<seq>` and `StreamCoordinator` now sets `seq` from Redis when possible.

## Step 6: Add Idle Conversation Eviction

Introduced an eviction loop in the conversation manager to remove idle conversations with no connections or queued/running work. Added configuration flags for eviction idle/interval, updated docs, and wrote unit tests to verify eviction and skip behavior.

This prevents unbounded growth when clients generate random conversation IDs.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the eviction loop and document the new configuration flags.

**Inferred user intent:** Avoid memory growth and make idle cleanup predictable and configurable.

**Commit (code):** 9c8adce — "Add idle conversation eviction"

### What I did
- Added eviction configuration and loop helpers to `ConvManager`, with idle/interval settings.
- Tracked `lastActivity` on conversations and updated it on connections, queue operations, and run completion.
- Added CLI flags `--evict-idle-seconds` and `--evict-interval-seconds` with defaults.
- Updated docs to describe the eviction tuning flags.
- Added unit tests for eviction and “busy” conversations.
- Ran `go test ./pinocchio/pkg/webchat -count=1`; pre-commit ran repo-wide tests, codegen, and lint.

### Why
- Eviction is required to safely embed the backend in long-running processes without memory growth.

### What worked
- Tests confirm idle conversations are evicted and busy ones are retained.
- Eviction starts automatically when the event router runs.

### What didn't work
- N/A

### What I learned
- Centralizing eviction in `ConvManager` keeps router/server responsibilities simple.

### What was tricky to build
- Ensuring `lastActivity` updates cover both HTTP and WS interactions without over-complicating state management.

### What warrants a second pair of eyes
- Validate that eviction doesn’t race with active WebSocket connections in high-concurrency workloads.

### What should be done in the future
- N/A

### Code review instructions
- Start at `pinocchio/pkg/webchat/conv_manager_eviction.go`, `pinocchio/pkg/webchat/conversation.go`, and `pinocchio/pkg/webchat/conv_manager_eviction_test.go`.
- Validate with `go test ./pinocchio/pkg/webchat -count=1`.

### Technical details
- `ConvManager.StartEvictionLoop` runs periodic sweeps and removes conversations that are idle past `evict-idle-seconds`.

## Step 7: Make ConnectionPool Non-Blocking

Reworked `ConnectionPool` to use per-connection writer goroutines and buffered send channels, enforcing backpressure by dropping connections when buffers fill. Added tests with a stub connection to verify drop behavior and updated backend docs to reflect the new concurrency model.

This removes head-of-line blocking from slow clients and keeps broadcasts fast.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement non-blocking websocket broadcasts with backpressure and document the new behavior.

**Inferred user intent:** Ensure slow clients do not stall the entire conversation stream and keep the backend reusable.

**Commit (code):** 011c824 — "Make ConnectionPool non-blocking"

### What I did
- Replaced direct `WriteMessage` under a mutex with per-connection writer goroutines and buffered channels.
- Added drop-on-full backpressure handling and write deadlines.
- Wrote a unit test using a stub connection to validate dropping when buffers fill.
- Updated backend reference and internals docs to describe the new behavior.
- Ran `go test ./pinocchio/pkg/webchat -count=1`; pre-commit ran repo-wide tests, codegen, and lint.

### Why
- Slow WebSocket clients should not block other clients or stall broadcasts.

### What worked
- The connection drop test confirms backpressure behavior without real network sockets.
- Broadcasts are now non-blocking and safe under concurrency.

### What didn't work
- N/A

### What I learned
- A small buffered channel per client plus a drop policy is enough to keep broadcasts fast and predictable.

### What was tricky to build
- Ensuring connection cleanup is safe when both the writer goroutine and the broadcaster may drop a client.

### What warrants a second pair of eyes
- Review the drop policy and write timeout to ensure they fit expected production workloads.

### What should be done in the future
- N/A

### Code review instructions
- Start at `pinocchio/pkg/webchat/connection_pool.go` and `pinocchio/pkg/webchat/connection_pool_test.go`.
- Review doc updates in `pinocchio/pkg/doc/topics/webchat-backend-reference.md` and `pinocchio/pkg/doc/topics/webchat-backend-internals.md`.
- Validate with `go test ./pinocchio/pkg/webchat -count=1`.

### Technical details
- `ConnectionPool` now enqueues frames to per-client channels and drops on full buffers to prevent stalls.

## Step 8: Require Stream Sequence Versions for Timeline Storage

Replaced timeline store auto-incrementing with explicit sequence numbers derived from SEM stream frames and removed any compatibility shims. Updated the projector to read `event.seq`, pushed the version into every upsert, and aligned docs to reflect seq-based hydration/merge behavior.

### Prompt Context

**User prompt (verbatim):** "don't use backwards compatibility at all or v2 interfaces, just rip things out."

**Assistant interpretation:** Remove compatibility layers and change the timeline storage interface directly to require explicit versions based on stream sequencing.

**Inferred user intent:** Keep the refactor clean and decisive by dropping transitional APIs and relying on the new versioning semantics everywhere.

**Commit (code):** 4964d10 — "Use stream seq versions for timeline store"

### What I did
- Updated `TimelineStore` to require an explicit version for `Upsert` and adjusted the in-memory/SQLite implementations accordingly.
- Parsed `event.seq` in `TimelineProjector` and propagated it through all upserts, including planning aggregation.
- Updated the router’s manual user-message upsert to stamp a seq-derived version.
- Adjusted timeline store tests to assert explicit versioning and reject `version=0`.
- Updated SEM + frontend integration docs to describe seq-based streaming/hydration merges and refreshed the StreamCoordinator reference.
- Ran `go test ./pkg/webchat -count=1`.
- Pre-commit ran `go test ./...`, `go generate ./...` (including Vite build), `go build ./...`, `golangci-lint run -v --max-same-issues=100`, and `go vet -vettool=/tmp/geppetto-lint ./...`.

### Why
- Timeline versions need to align with stream ordering, and the code should enforce that without any transitional V2 interface.

### What worked
- All updated tests passed and linters reported zero issues.
- Docs now describe the new seq-based version semantics clearly.

### What didn't work
- `git commit -m "Use stream seq versions for timeline store"` timed out twice while pre-commit was running (`10s` and `20s` timeouts). Re-ran with a longer timeout to complete the commit.

### What I learned
- The stream cursor already provides enough ordering metadata; the timeline store can rely on it directly without extra interfaces.

### What was tricky to build
- Ensuring every timeline upsert path had access to a sequence (including planning aggregates and manual user-message inserts) while removing all auto-increment behavior.

### What warrants a second pair of eyes
- Confirm the chosen seq for manual user prompt upserts (time-based) is acceptable and won’t confuse ordering relative to stream-sourced events.

### What should be done in the future
- N/A

### Code review instructions
- Start at `pinocchio/pkg/webchat/timeline_store.go`, `pinocchio/pkg/webchat/timeline_store_sqlite.go`, and `pinocchio/pkg/webchat/timeline_projector.go`.
- Review docs in `pinocchio/pkg/doc/topics/webchat-sem-and-ui.md`, `pinocchio/pkg/doc/topics/webchat-frontend-integration.md`, and `pinocchio/pkg/doc/topics/webchat-backend-reference.md`.
- Validate with `go test ./pinocchio/pkg/webchat -count=1`.

### Technical details
- Timeline upserts now require an explicit version and use `event.seq` from SEM frames; `version=0` is rejected.
- Manual user-message upserts stamp a seq based on the current time to preserve ordering without a compatibility layer.
