# Diary

## Goal

Document the research process for analyzing hydration in pinocchio vs go-go-mento and identifying ordering issues with user messages.

## Step 1: Initialize ticket + scan codebase surface area

I created the ticket workspace and initial documents, then performed a broad code scan to locate hydration-related entry points in both repos. The goal was to map the surface area before diving into the specific flows.

This step establishes a working inventory of files and paths that define hydration behavior and likely touch message ordering.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket GP-011-CHECK-HYDRATION with docmgr.

DO a deep analysis comparing how hydration is done in pinocchio and how it is done in go-go-mento, which is where it originally comes from.

We had problems with adding user messages to the hydration, and now when we restore from the DB using hydration, the ordering of user messages is off. I know we are not necessarily using the redis transport, which might some of the reason. In any case, do a deep analysis and write up of how hydration works in both repositories (one big textbook chapter for each, in the document, who is in a textbook detailed style, as if written by peter norvig).

Then compare what is potentially missing or off in ht epinocchio version, and how it can be fixed, using go-go-mento as an inspiration.

Write a detailed well research document. Keep a frequent detailed research diary where you document the steps done in your research (not just at the end)."

**Assistant interpretation:** Create a docmgr ticket, research hydration implementations in both repos, write a deep comparative document with two long-form chapters plus a gap analysis, and keep a detailed research diary with multiple steps.

**Inferred user intent:** Identify why user message ordering is incorrect after hydration in pinocchio by grounding the analysis in the original go-go-mento design and producing a fix-oriented comparison.

**Commit (code):** N/A

### What I did
- Ran `docmgr ticket create-ticket` and added analysis + diary docs for GP-011-CHECK-HYDRATION.
- Scanned both repositories for hydration-related files using `rg`.

### Why
- Establish a shared workspace and a file inventory before detailed reading and synthesis.

### What worked
- Found hydration entry points in pinocchio (router, timeline store, frontend hydration gating) and go-go-mento (timelinehydration package, webchat router, CLI docs, frontend hook).

### What didn't work
- N/A

### What I learned
- Pinocchio exposes both legacy `/hydrate` and newer `/timeline` hydration paths; go-go-mento has a full timeline hydration subsystem and extensive docs.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- N/A

### Technical details
- Key paths discovered: `pinocchio/pkg/webchat/router.go`, `pinocchio/pkg/webchat/timeline_store.go`, `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`, `go-go-mento/go/pkg/persistence/timelinehydration/*`, `go-go-mento/go/pkg/webchat/router.go`, `go-go-mento/go/pkg/webchat/conversation.go`.

## Step 2: Read pinocchio hydration flows (server + frontend)

I inspected the pinocchio webchat router, timeline store, timeline projector, and frontend hydration logic to understand where entities are created, ordered, and rendered. The goal was to isolate where user messages enter the hydration pipeline and how ordering is derived on restore.

This step focuses on pinocchio’s dual hydration modes (/hydrate frame buffer vs /timeline durable snapshots) and how created/updated timestamps and ordering are determined.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket GP-011-CHECK-HYDRATION with docmgr.

DO a deep analysis comparing how hydration is done in pinocchio and how it is done in go-go-mento, which is where it originally comes from.

We had problems with adding user messages to the hydration, and now when we restore from the DB using hydration, the ordering of user messages is off. I know we are not necessarily using the redis transport, which might some of the reason. In any case, do a deep analysis and write up of how hydration works in both repositories (one big textbook chapter for each, in the document, who is in a textbook detailed style, as if written by peter norvig).

Then compare what is potentially missing or off in ht epinocchio version, and how it can be fixed, using go-go-mento as an inspiration.

Write a detailed well research document. Keep a frequent detailed research diary where you document the steps done in your research (not just at the end)."

**Assistant interpretation:** Dig into pinocchio’s hydration code paths to document the full data flow and ordering rules.

**Inferred user intent:** Identify concrete reasons user messages are misordered after DB hydration by analyzing pinocchio’s storage and ordering logic.

**Commit (code):** N/A

### What I did
- Read `pinocchio/pkg/webchat/router.go` for `/hydrate`, `/timeline`, and user message persistence.
- Read `pinocchio/pkg/webchat/timeline_store_sqlite.go` for schema and ordering queries.
- Read `pinocchio/pkg/webchat/timeline_projector.go` for SEM→timeline projection.
- Read `pinocchio/pkg/webchat/conversation.go` and `sem_buffer.go` for stream hookup and frame buffering.
- Read `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`, `timelineSlice.ts`, and `chat/ChatWidget.tsx` for hydration usage and ordering on the client.

### Why
- These files define how user messages are inserted, how snapshots are ordered, and how UI insertion order is preserved after hydration.

### What worked
- Identified two hydration paths (SEM frame buffer vs durable timeline).
- Confirmed that durable snapshots are ordered by `created_at_ms` with a fallback to `entity_id` in SQLite.
- Located the explicit user message `Upsert` in `/chat` handling (separate from SEM projection).

### What didn't work
- N/A

### What I learned
- Timeline ordering in pinocchio is driven by `created_at_ms`, which defaults to the first upsert time, not a stream-derived version.
- Frontend preserves snapshot order directly; it does not sort by timestamps after hydration.

### What was tricky to build
- Distinguishing optimistic user message echoes from durable hydration entities, since the client clears the timeline on hydrate.

### What warrants a second pair of eyes
- The implicit ordering tie-breaker (`entity_id`) when `created_at_ms` collides could be a subtle source of reordering.

### What should be done in the future
- N/A

### Code review instructions
- N/A

### Technical details
- Ordering for full snapshot: SQL in `pinocchio/pkg/webchat/timeline_store_sqlite.go` uses `ORDER BY created_at_ms ASC, entity_id ASC`.
- User message persistence: `pinocchio/pkg/webchat/router.go` uses `TimelineStore.Upsert` with `Id: "user-"+turnID` and no explicit `CreatedAtMs`.
- Frontend hydration: `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts` loads `/timeline` and inserts entities in received order.

## Step 3: Read go-go-mento hydration flows (timelinehydration + webchat)

I inspected the go-go-mento timeline hydration subsystem, its webchat integration, and the frontend hydration hook to build a complete model of the original design. The goal was to capture how ordering and versioning are intended to work, especially with Redis stream IDs.

This step establishes the baseline design that pinocchio should align with.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket GP-011-CHECK-HYDRATION with docmgr.

DO a deep analysis comparing how hydration is done in pinocchio and how it is done in go-go-mento, which is where it originally comes from.

We had problems with adding user messages to the hydration, and now when we restore from the DB using hydration, the ordering of user messages is off. I know we are not necessarily using the redis transport, which might some of the reason. In any case, do a deep analysis and write up of how hydration works in both repositories (one big textbook chapter for each, in the document, who is in a textbook detailed style, as if written by peter norvig).

Then compare what is potentially missing or off in ht epinocchio version, and how it can be fixed, using go-go-mento as an inspiration.

Write a detailed well research document. Keep a frequent detailed research diary where you document the steps done in your research (not just at the end)."

**Assistant interpretation:** Study go-go-mento’s hydration architecture and ordering mechanics to use as a reference for the pinocchio gap analysis.

**Inferred user intent:** Use the original implementation to explain what pinocchio should preserve (ordering, versioning, user message persistence).

**Commit (code):** N/A

### What I did
- Read `go-go-mento/go/pkg/persistence/timelinehydration/{aggregator,projector,service,repo}.go`.
- Read `go-go-mento/go/pkg/webchat/{conversation,router,stream_coordinator,event_version_cache}.go`.
- Read `go-go-mento/web/src/hooks/useTimelineHydration.ts` and `web/src/store/timeline/timelineSlice.ts`.
- Skimmed `go-go-mento/docs/reference/persistence/timeline-hydration.md` and the `webchat hydrate` CLI entrypoint for offline replay context.

### Why
- These files encode the original ordering guarantees (Redis stream version overrides, version tracker, SQL ordering) and the user message hydration flow.

### What worked
- Confirmed Redis stream IDs are converted into monotonic versions to preserve event order.
- Found explicit user message persistence + broadcast (`user.message`) with a shared version value.
- Identified the repository’s ordering (`ORDER BY updated_at, version`) for hydration queries.

### What didn't work
- N/A

### What I learned
- go-go-mento treats versions as the primary ordering mechanism and uses a version tracker to avoid collisions.
- Hydration is a full subsystem: projector/service/repo with explicit separation of concerns.

### What was tricky to build
- Mapping where version overrides are injected (stream coordinator → event_version_cache → streamOnEvent).

### What warrants a second pair of eyes
- Whether `updated_at`/`version` ordering in Postgres maps cleanly to pinocchio’s `created_at_ms` semantics.

### What should be done in the future
- N/A

### Code review instructions
- N/A

### Technical details
- Redis XID override: `go-go-mento/go/pkg/webchat/stream_coordinator.go` converts stream IDs to versions and caches them for `streamOnEvent`.
- User message persistence: `go-go-mento/go/pkg/webchat/router.go` uses `timelinehydration.NewUserMessageProjectedEntity` and emits `user.message` with `version`.
- Hydration query ordering: `go-go-mento/go/pkg/persistence/timelinehydration/repo.go` uses `ORDER BY updated_at NULLS LAST, version`.

## Step 4: Synthesize findings into the comparative analysis document

I wrote the long-form analysis document with two textbook-style chapters (pinocchio and go-go-mento), followed by a comparative diagnosis and a concrete fix blueprint. The focus was on ordering semantics, user message insertion paths, and the role of Redis stream versioning.

This step turns the code reading into a single, structured narrative with actionable conclusions.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket GP-011-CHECK-HYDRATION with docmgr.

DO a deep analysis comparing how hydration is done in pinocchio and how it is done in go-go-mento, which is where it originally comes from.

We had problems with adding user messages to the hydration, and now when we restore from the DB using hydration, the ordering of user messages is off. I know we are not necessarily using the redis transport, which might some of the reason. In any case, do a deep analysis and write up of how hydration works in both repositories (one big textbook chapter for each, in the document, who is in a textbook detailed style, as if written by peter norvig).

Then compare what is potentially missing or off in ht epinocchio version, and how it can be fixed, using go-go-mento as an inspiration.

Write a detailed well research document. Keep a frequent detailed research diary where you document the steps done in your research (not just at the end)."

**Assistant interpretation:** Produce the final research write-up, include the chapter-style deep dives and the gap analysis with fixes.

**Inferred user intent:** A reusable internal reference for debugging hydration ordering and guiding repairs.

**Commit (code):** N/A

### What I did
- Authored the analysis doc at `geppetto/ttmp/2026/01/25/GP-011-CHECK-HYDRATION--hydration-parity-analysis-pinocchio-vs-go-go-mento/analysis/01-hydration-deep-dive-pinocchio-vs-go-go-mento.md`.
- Structured it into two main chapters plus a comparative diagnosis and fix blueprint.

### Why
- The user requested a detailed research document in a textbook-like style, plus explicit gap analysis and fixes.

### What worked
- The document captures ordering semantics, user message insertion paths, and Redis/versioning differences with concrete file references.

### What didn't work
- N/A

### What I learned
- The ordering guarantee in go-go-mento is a deliberate versioning contract; pinocchio’s created_at-based ordering lacks that contract.

### What was tricky to build
- Keeping the narrative detailed while staying grounded in specific file paths and design invariants.

### What warrants a second pair of eyes
- The proposed shift in pinocchio ordering (created_at vs version) should be reviewed for compatibility with existing UI expectations.

### What should be done in the future
- N/A

### Code review instructions
- Start with the analysis doc and verify the file references map to the described behaviors.

### Technical details
- See the “Appendix: Key Files Consulted” section in the analysis doc for the full reference list.

## Step 5: Analyze hydration path unification (buffer vs timeline)

I wrote a separate design note exploring whether pinocchio should unify the in-memory SEM buffer hydration and the durable timeline snapshot into a single representation. The analysis compares canonical event-log vs canonical projection approaches and proposes a pragmatic hybrid with a size-limited in-memory projection cache.

This step responds to the request to evaluate unification and suggest solutions.

### Prompt Context

**User prompt (verbatim):** "can you analyze if we can unify both hydration paths as well? why do a in memory buffer and then use a different repr? potentially we can do some kind of size limited cache of persisted in memory projections, and then use that as a store? Analyze in detail, similarly, suggest good solutions"

**Assistant interpretation:** Evaluate the architectural tradeoffs of unifying /hydrate and /timeline, and propose concrete solutions including in-memory projection caching.

**Inferred user intent:** Reduce complexity and ordering drift by converging on a single hydration representation.

**Commit (code):** N/A

### What I did
- Authored `design/02-unifying-hydration-paths-buffer-vs-timeline.md` with options, tradeoffs, and recommendations.
- Related the design doc to core buffer, projector, store, and frontend hydration files.

### Why
- The user asked for an in-depth analysis and solution proposals around unifying hydration paths.

### What worked
- The design doc articulates a clear recommended path: canonical projection + WS deltas + in-memory cache.

### What didn't work
- N/A

### What I learned
- A unified path is viable only if ordering/version contracts are enforced across both snapshot and streaming updates.

### What was tricky to build
- Balancing the benefits of SEM replay with the need for a single canonical representation.

### What warrants a second pair of eyes
- The recommended deprecation of `/hydrate` should be validated against current debugging workflows.

### What should be done in the future
- N/A

### Code review instructions
- Review the design doc and verify references align with the current pinocchio code paths.

### Technical details
- See `design/02-unifying-hydration-paths-buffer-vs-timeline.md` for the options matrix and recommended architecture.

## Step 6: Add clarification on keeping SEM frames with canonical timeline projections

I added a clarification section to the unification design doc to reflect the decision to keep SEM frames as the live stream while treating timeline projections as the canonical representation. The update explains the dual-stream contract and how versioned merging keeps hydration unified.

### Prompt Context

**User prompt (verbatim):** "yes. i want:
- keep SEM frames
- timeine projections are the canonical representation"

**Assistant interpretation:** Update the unification design to explicitly keep SEM frames while designating the timeline projection as the source of truth.

**Inferred user intent:** Preserve streaming UX and debugging while avoiding dual authoritative state representations.

**Commit (code):** N/A

### What I did
- Added Section 10 to `design/02-unifying-hydration-paths-buffer-vs-timeline.md` clarifying the dual-stream contract.

### Why
- The user requested an explicit design stance on keeping SEM frames alongside canonical timeline projections.

### What worked
- The doc now states how SEM frames map to timeline entity updates and how clients should merge by version.

### What didn't work
- N/A

### What I learned
- The “unified hydration” goal is compatible with a dual-stream delivery model if the timeline version contract is enforced.

### What was tricky to build
- Avoiding language that implies SEM is a second source of truth.

### What warrants a second pair of eyes
- Confirm that the proposed dual-stream behavior matches current frontend expectations.

### What should be done in the future
- N/A

### Code review instructions
- Review the new Section 10 in the unification design doc.

### Technical details
- Updated file: `design/02-unifying-hydration-paths-buffer-vs-timeline.md`.

## Step 7: Update unification design to Approach B with in-memory store fallback

I rewrote the unification design doc to explicitly adopt Approach B: timeline projections are canonical, SEM frames remain, and /timeline is the single hydration path. I also added the requirement that when no DB is configured, an in-memory TimelineStore is used so the same path works everywhere.

### Prompt Context

**User prompt (verbatim):** "no no, I want the approach B you described, and when no DB is used, just use an in memory DB. That way we have one single path for everything."

**Assistant interpretation:** Update the design doc to make Approach B the chosen architecture and require an in-memory timeline store fallback.

**Inferred user intent:** Eliminate dual hydration logic by mandating a single canonical projection path in all environments.

**Commit (code):** N/A

### What I did
- Rewrote `design/02-unifying-hydration-paths-buffer-vs-timeline.md` to reflect Approach B + in-memory store fallback.
- Clarified that /timeline is the sole hydration path and SEM frames remain as live stream.

### Why
- The user explicitly chose Approach B and wants a single unified path without optional /hydrate behavior.

### What worked
- The doc now describes a consistent architecture across dev/prod with identical ordering semantics.

### What didn't work
- N/A

### What I learned
- Explicitly requiring the in-memory store avoids ambiguity about dev behavior and makes testing more reliable.

### What was tricky to build
- Ensuring the doc stays consistent with prior sections while shifting the recommended architecture.

### What warrants a second pair of eyes
- Verify that the in-memory store semantics exactly match the durable store’s ordering and version rules.

### What should be done in the future
- N/A

### Code review instructions
- Review the rewritten design doc for consistency and correctness.

### Technical details
- Updated file: `design/02-unifying-hydration-paths-buffer-vs-timeline.md`.

## Step 8: Implement backend single-path timeline hydration (store + WS deltas)

I implemented the backend half of the unified hydration path: a mandatory timeline store (SQLite or in-memory), removal of /hydrate, and timeline upsert emissions alongside SEM frames. This aligns the runtime behavior with the chosen Approach B design.

The focus here was on making the projection store always available and emitting versioned timeline deltas for the frontend to consume.

### Prompt Context

**User prompt (verbatim):** "now implement them, check them off as you go, commit to git.

Keep a detailed and frequent implementation diary."

**Assistant interpretation:** Start implementing the planned changes, update tasks as I go, and commit the result with a detailed diary.

**Inferred user intent:** Deliver working code changes that realize the chosen hydration architecture, while documenting each step clearly.

**Commit (code):** N/A

### What I did
- Added an in-memory `TimelineStore` implementation (`timeline_store_memory.go`).
- Updated `RouterSettings` and `NewRouter` to always create a timeline store (SQLite if configured, in-memory otherwise).
- Removed `/hydrate` endpoint from `router.go` and updated README API docs.
- Added timeline upsert emission (`timeline_upsert.go`) and wired `TimelineProjector` to emit upserts after each store write.
- Updated user message persistence to emit a timeline upsert after `Upsert`.

### Why
- The chosen architecture requires a single canonical path (`/timeline`), with timeline projections always available regardless of DB configuration.

### What worked
- Timeline upsert events are now emitted for both projected SEM events and user messages.
- `/timeline` can be served from in-memory storage when no DB is configured.

### What didn't work
- N/A

### What I learned
- The existing proto already defines `TimelineUpsertV1`, which is a good fit for versioned WS deltas.

### What was tricky to build
- Ensuring the in-memory store mirrors SQLite ordering semantics so hydration is consistent across environments.

### What warrants a second pair of eyes
- Verify the timeline upsert events are emitted for all relevant entity types, especially throttled `llm.delta` projections.

### What should be done in the future
- N/A

### Code review instructions
- Start with `pinocchio/pkg/webchat/timeline_store_memory.go` and `pinocchio/pkg/webchat/timeline_upsert.go`.
- Validate router wiring in `pinocchio/pkg/webchat/router.go` and projector hook usage in `pinocchio/pkg/webchat/conversation.go`.

### Technical details
- New file: `pinocchio/pkg/webchat/timeline_store_memory.go` implements size-limited in-memory projections.
- Removed `/hydrate` handler in `pinocchio/pkg/webchat/router.go`.
- Added `pinocchio/pkg/webchat/timeline_upsert.go` to emit `timeline.upsert` frames.

## Step 9: Implement frontend versioned merge + timeline upsert handling

I updated the frontend to consume timeline upsert events as versioned deltas and to treat timeline projections as canonical. This includes a shared mapper for timeline entities, version-aware merging in the timeline slice, and removal of the /hydrate fallback in WS hydration.

This step ensures the UI merges snapshot + deltas deterministically while keeping SEM frames for streaming.

### Prompt Context

**User prompt (verbatim):** "now implement them, check them off as you go, commit to git.

Keep a detailed and frequent implementation diary."

**Assistant interpretation:** Implement the planned frontend changes to match the new hydration contract.

**Inferred user intent:** The UI should remain responsive and consistent with the canonical timeline projections.

**Commit (code):** N/A

### What I did
- Added `sem/timelineMapper.ts` to map timeline proto entities into Redux state.
- Updated `wsManager.ts` to hydrate via `/timeline` only and to apply snapshot entities with a version.
- Added a `timeline.upsert` handler in `sem/registry.ts` to apply versioned deltas.
- Updated `timelineSlice.ts` to track `version` and merge versioned updates safely.

### Why
- The frontend needs a deterministic merge strategy when receiving timeline deltas alongside SEM frames.

### What worked
- Timeline entities now carry versions and ignore stale updates.
- `/hydrate` fallback removed from the client path.

### What didn't work
- N/A

### What I learned
- The existing SEM registry structure can be extended cleanly to handle timeline upserts.

### What was tricky to build
- Preserving existing optimistic/streaming behavior while introducing versioned canonical updates.

### What warrants a second pair of eyes
- Confirm that SEM-based streaming still feels smooth now that versioned deltas are authoritative.

### What should be done in the future
- N/A

### Code review instructions
- Review `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts` for version merge logic.
- Review `pinocchio/cmd/web-chat/web/src/sem/registry.ts` and `pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts` for delta handling.

### Technical details
- New file: `pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts`.
- `timeline.upsert` event handling is in `pinocchio/cmd/web-chat/web/src/sem/registry.ts`.

## Step 10: Tests, fixes, and task completion

I ran Go tests for the webchat package, fixed compilation errors introduced by the projector refactor, and added tests for the in-memory store and ordering semantics. I also checked off the implementation tasks and updated the docmgr changelog.

### Prompt Context

**User prompt (verbatim):** "now implement them, check them off as you go, commit to git.

Keep a detailed and frequent implementation diary."

**Assistant interpretation:** Validate the implementation, fix any breakages, and keep bookkeeping up to date.

**Inferred user intent:** High confidence in correctness plus traceable progress.

**Commit (code):** N/A

### What I did
- Ran `go test ./pinocchio/pkg/webchat -count=1` and fixed build errors in `timeline_projector.go`.
- Added `timeline_store_memory_test.go` and updated `timeline_store_sqlite_test.go` for version ordering.
- Checked off all tasks in docmgr and updated the ticket changelog.

### Why
- Ensuring basic test coverage reduces risk of regressions in the hydration pipeline.

### What worked
- Tests pass after fixing the projector’s upsert signature and adding in-memory store tests.

### What didn't work
- Initial test run failed:
  - `assignment mismatch: 2 variables but p.upsert returns 1 value`
  - Fixed by updating `timeline_projector.go` to use single-value `err := p.upsert(...)` calls.

### What I learned
- Refactoring helper signatures in hot paths requires careful pass through all call sites.

### What was tricky to build
- Keeping tests meaningful while reordering full snapshots by version instead of created_at.

### What warrants a second pair of eyes
- Review the ordering change in SQLite `GetSnapshot` to ensure it aligns with desired UI semantics.

### What should be done in the future
- N/A

### Code review instructions
- Start with `pinocchio/pkg/webchat/timeline_store_memory_test.go` and `pinocchio/pkg/webchat/timeline_store_sqlite_test.go`.
- Verify `timeline_projector.go` changes and ensure no recursion or signature mismatch.

### Technical details
- Test commands:
  - `go test ./pinocchio/pkg/webchat -count=1`
- Errors observed:
  - `assignment mismatch: 2 variables but p.upsert returns 1 value`
