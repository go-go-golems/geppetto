---
Title: 'Docs migration plan: MO-003..006 outcomes + go-go-mento webchat docs'
Ticket: GP-10-CLOSE-OLD-TICKETS
Status: active
Topics:
    - infrastructure
    - geppetto
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-mento/docs/reference/webchat/README.md
      Note: Primary legacy webchat doc index being triaged
    - Path: ../../../../../../../go-go-mento/docs/reference/webchat/backend-internals.md
      Note: Legacy backend internals to adapt
    - Path: ../../../../../../../go-go-mento/docs/reference/webchat/backend-reference.md
      Note: Legacy backend reference to adapt
    - Path: ../../../../../../../go-go-mento/docs/reference/webchat/debugging-and-ops.md
      Note: Legacy debugging/ops guide to adapt
    - Path: ../../../../../../../go-go-mento/docs/reference/webchat/frontend-integration.md
      Note: Legacy frontend integration to adapt
    - Path: ../../../../../../../go-go-mento/docs/reference/webchat/sem-and-widgets.md
      Note: Legacy SEM+widgets reference to prune/adapt
    - Path: ../../../../../../../pinocchio/cmd/web-chat/README.md
      Note: Existing web-chat example doc (merge target)
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Current frontend WS + hydration manager
    - Path: ../../../../../../../pinocchio/pkg/doc/topics/webchat-backend-internals.md
      Note: New doc created per migration plan
    - Path: ../../../../../../../pinocchio/pkg/doc/topics/webchat-backend-reference.md
      Note: New doc created per migration plan
    - Path: ../../../../../../../pinocchio/pkg/doc/topics/webchat-debugging-and-ops.md
      Note: New doc created per migration plan
    - Path: ../../../../../../../pinocchio/pkg/doc/topics/webchat-framework-guide.md
      Note: Existing pinocchio webchat guide (merge target)
    - Path: ../../../../../../../pinocchio/pkg/doc/topics/webchat-frontend-integration.md
      Note: New doc created per migration plan
    - Path: ../../../../../../../pinocchio/pkg/doc/topics/webchat-overview.md
      Note: New index doc created in Step 9
    - Path: ../../../../../../../pinocchio/pkg/doc/topics/webchat-sem-and-ui.md
      Note: New doc created per migration plan
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: Current pinocchio webchat backend surface
    - Path: pkg/doc/playbooks/04-migrate-to-session-api.md
      Note: Existing geppetto migration playbook (merge/link target)
ExternalSources: []
Summary: What documentation to preserve from MO-003..MO-006 work and go-go-mento webchat docs, and where to migrate it in geppetto/pinocchio for developer onboarding + productive usage.
LastUpdated: 2026-01-25T09:29:54.415039613-05:00
WhatFor: Provide a concrete plan for migrating/refactoring older webchat/inference documentation into the current geppetto/pinocchio doc structure, including what to copy, what to merge, and what to archive.
WhenToUse: When closing MO-003..MO-006 and deciding how to make the final architecture discoverable and usable for a team of developers.
---




# Docs migration plan: MO-003..006 outcomes + go-go-mento webchat docs

## Goal

Turn the MO-003..MO-006 refactor series into **team-usable documentation** that:
- explains “how it works” (architecture + invariants),
- shows “how to use it” (copy/paste recipes and tested run commands),
- makes “how to debug it” fast (common failure modes + where to look),
- lives in the **right repo** (geppetto vs pinocchio) and matches the **current** package structure.

This doc also reviews `go-go-mento/docs/reference/webchat/README.md` (and its sibling docs) and recommends what to:
- copy + adapt into pinocchio,
- merge into geppetto’s core docs,
- keep as historical reference (archive / mark as deprecated).

## Background: where docs should live now

There are three “classes” of docs in this workspace:

1) **Ticket docs (`geppetto/ttmp/...`)** — great for implementation diaries and long-form analysis during a refactor, but not where developers will look day-to-day.
2) **Library docs (`geppetto/pkg/doc/**`)** — the stable, canonical documentation for reusable primitives (events, turns, inference/session, tool loop, migration playbooks).
3) **App/framework docs (`pinocchio/pkg/doc/**` + `pinocchio/cmd/*/README.md`)** — how to build/run/debug the pinocchio applications and frameworks (TUI and webchat), including frontends under `pinocchio/cmd/web-chat/web/`.

Rule of thumb for migration:
- If the doc teaches a *reusable primitive* (session lifecycle, tool loop, event sinks): **move/merge into `geppetto/pkg/doc`**.
- If the doc teaches *pinocchio webchat specifics* (router endpoints, Redis consumer groups, SEM translator, frontend wsManager): **move/merge into `pinocchio/pkg/doc` (or `pinocchio/cmd/web-chat/README.md`)**.
- If it’s *product-specific* (go-go-mento identity, Mento React widgets, domain entities): **do not move into geppetto/pinocchio**; keep it in go-go-mento (or move to `moments/docs` if that is the active product surface).

## What the MO-003..MO-006 series “landed” (final structure)

From the refactor sequence, the stable conceptual outcomes to document are:

### geppetto (core)

- **Events are context-driven** (attach sinks to `context.Context`; publishers call `events.PublishEventToContext`).
- **Session-style orchestration exists** (a session owns multi-turn state and exposes a “start + cancel + wait” surface).
- **Tool loop exists as a shared primitive** (tool calling loop + an enginebuilder API that composes base engine + middlewares + tool loop + event sinks).
- **Turns/blocks ordering and strict provider validation are real constraints** (OpenAI Responses API strictness was a forcing function).

Most of this is already partially documented in:
- `geppetto/pkg/doc/topics/04-events.md`
- `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md`

### pinocchio (TUI + webchat)

- **Webchat framework** exists under `pinocchio/pkg/webchat/` with:
  - `StreamCoordinator` and `ConnectionPool` (Redis→WS bridging and WS fanout),
  - semantic translation (`sem_translator.go`),
  - run queue + idempotency,
  - optional persistence/hydration hooks (`timeline_*`).
- The webchat example app lives in `pinocchio/cmd/web-chat/` (Vite frontend + wsManager + Redux-like store).

This is partially documented in:
- `pinocchio/pkg/doc/topics/webchat-framework-guide.md`
- `pinocchio/cmd/web-chat/README.md`

## What docs a team actually needs (recommended doc set)

### 1) geppetto docs (for “library users” and infra maintainers)

**Must-have**
- “Events and sinks” (already exists): explain context sinks, Watermill sinks, and anti-patterns like accidental duplicate sink attachment.
- “Session + cancellation model” (partially exists as migration playbook): explain the *current* recommended lifecycle vocabulary and how UIs/webchat use it.
- “Tool loop + enginebuilder” (currently spread across examples): a short reference doc that explains the enginebuilder options and how it maps to the tool loop.

**Nice-to-have**
- A “common failure modes” section for strict provider validation (Responses ordering constraints) and how to debug malformed turn snapshots.

### 2) pinocchio webchat backend docs (for “backend webchat devs”)

**Must-have**
- Architecture overview (beginner-friendly): router endpoints, per-conversation state, Redis topics and consumer group strategy, how streaming frames reach WS clients.
- Backend reference: API + semantics for `StreamCoordinator`, `ConnectionPool`, `Conversation`, `timeline_store/*`.
- Backend internals: goroutine lifecycle, ack semantics, ordering guarantees, how to avoid races (synchronous callbacks).
- Debugging/ops: “what logs to look for”, “common failure modes”, and sequence diagrams for connect + message flow.

### 3) pinocchio webchat frontend docs (for “frontend / fullstack devs”)

**Must-have**
- `wsManager` contract and hydration gating (what happens before and after socket open; how buffered frames are applied).
- SEM registry + handler patterns (how events map to UI entities; how to add new mappings).
- Timeline store / snapshot endpoints (`/hydrate`, `/timeline`) and how they relate to `seq` and durable versions.

## go-go-mento webchat docs: what to copy, merge, or leave

The directory `go-go-mento/docs/reference/webchat/` is a strong starting point, but it must be adapted:
- paths are wrong for pinocchio (`go/pkg/webchat/...` → `pinocchio/pkg/webchat/...`),
- some docs are product-specific (React/Redux paths and Mento-specific entity catalogs),
- some concepts are now “owned by geppetto docs” (InferenceState lifecycle and sink semantics are no longer app-local concerns).

### Recommended migration map

| go-go-mento source doc | Recommended destination | Action | Notes |
|---|---|---|---|
| `go-go-mento/docs/architecture/webchat/README.md` | `pinocchio/pkg/doc/topics/webchat-architecture.md` (new) | Copy + adapt | Keep it beginner-friendly; update diagram and links to pinocchio code. |
| `go-go-mento/docs/reference/webchat/backend-reference.md` | `pinocchio/pkg/doc/topics/webchat-backend-reference.md` (new) | Copy + adapt | Update API sections to match `pinocchio/pkg/webchat/stream_coordinator.go` and `connection_pool.go`. |
| `go-go-mento/docs/reference/webchat/backend-internals.md` | `pinocchio/pkg/doc/topics/webchat-backend-internals.md` (new) | Copy + adapt | Update Watermill/Redis snippets to match `pinocchio/pkg/redisstream` helpers. |
| `go-go-mento/docs/reference/webchat/debugging-and-ops.md` | `pinocchio/pkg/doc/topics/webchat-debugging-and-ops.md` (new) | Copy + adapt | Replace React StrictMode notes with pinocchio’s actual `wsManager` behavior and logs. |
| `go-go-mento/docs/reference/webchat/frontend-integration.md` | `pinocchio/pkg/doc/topics/webchat-frontend-integration.md` (new) OR extend `pinocchio/cmd/web-chat/README.md` | Copy + adapt | Update file pointers to `pinocchio/cmd/web-chat/web/src/**`. Focus on: ws lifecycle, hydration, SEM routing, store update patterns. |
| `go-go-mento/docs/reference/webchat/sem-and-widgets.md` | `pinocchio/pkg/doc/topics/webchat-sem-and-ui.md` (new) | Copy + heavily prune | Pinocchio uses protobuf-backed SEM payloads; keep handler/registry concepts, but replace the entity catalog with the pinocchio timeline kinds actually emitted. |
| `go-go-mento/docs/reference/webchat/engine-builder.md` | Merge into `pinocchio/pkg/doc/topics/webchat-framework-guide.md` (existing) | Merge | Pinocchio has `pinocchio/pkg/webchat/engine_builder.go`; keep webchat-specific “profile/overrides” docs here, but link to geppetto enginebuilder/session docs for core behavior. |
| `go-go-mento/docs/reference/webchat/inference-state.md` | `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md` (existing) + a short pinocchio note | Do not copy verbatim; replace with links | The “inference lifecycle” is now a geppetto-owned concept; pinocchio docs should explain how webchat *uses* it, not redefine it. |
| `go-go-mento/docs/reference/webchat/identity-context.md` | Keep in go-go-mento (or move to `moments/docs` if still relevant) | Do not migrate to pinocchio/geppetto | Pinocchio webchat has no identity layer; this doc is product-specific. |

### What to do inside go-go-mento after migration

Because `go-go-mento` is described as deprecated in this workspace, the clean end-state is:
- Leave `go-go-mento/docs/reference/webchat/*` in place as historical reference **but** add a top banner in its `README.md` pointing to the new canonical pinocchio docs.
- Optionally move the directory to an “archive” subfolder in go-go-mento to reduce accidental onboarding to the wrong system.

## How to turn ticket docs into stable docs (MO-003..MO-006)

This is the key “merge” work:

### Ticket-doc migration map (source → target)

| Ticket source doc | Recommended destination | Action | Notes |
|---|---|---|---|
| `MO-003-UNIFY-INFERENCE/design-doc/01-unified-conversation-handling-across-repos.md` | Keep in `ttmp` (ticket archive) | Archive / link | Useful as historical architecture exploration; too broad for “current state” docs. Link from newer docs if needed. |
| `MO-003-UNIFY-INFERENCE/design-doc/02-turn-centric-conversation-state-and-runner-api*.md` | Keep in `ttmp` (ticket archive) | Archive / link | Multiple versions exist; pick one canonical and mark others archived if you want to reduce noise. |
| `MO-004-UNIFY-INFERENCE-STATE/analysis/02-postmortem-...md` | `geppetto/pkg/doc/topics/<new>-inference-session-architecture.md` | Extract + condense | Postmortem is valuable but long; extract “what changed + how to use it + why” into a stable topic doc, keep postmortem as deep reference. |
| `MO-004-UNIFY-INFERENCE-STATE/reference/02-playbook-testing-...md` | `geppetto/pkg/doc/playbooks/<new>-testing-inference.md` OR keep linked | Copy/adapt or link | This is already in a “playbook” shape; promoting it reduces reliance on ttmp. |
| `MO-005-CLEANUP-SINKS/analysis/01-sink-cleanup-...md` | `geppetto/pkg/doc/topics/04-events.md` + `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md` | Merge | Use as source material to tighten sink guidance and remove any stale API references (e.g., engine-config sinks). |
| `MO-006-CLEANUP-CANCELLATION-LIFECYCLE/analysis/01-run-vs-conversation-vs-inference-...md` | `geppetto/pkg/doc/topics/<new>-lifecycle-and-cancellation.md` | Extract + update names | This is the core conceptual doc to make the team avoid “stuck generating” and bad ownership models. Update terminology to match current `session.Session` naming. |
| `MO-006-CLEANUP-CANCELLATION-LIFECYCLE/analysis/02-compendium-...md` | Keep in `ttmp` (deep reference) | Keep + link | Keep it as the “Norvig-style” appendix; link it from the condensed stable topic doc. |
| `MO-005-OAK-GIT-HISTORY/*` | `oak-git-db/docs/*` | Already done | The ticket already points to the standalone repo; don’t duplicate into geppetto/pinocchio docs unless you want a short “tool index” entry. |

### Move/merge into geppetto/pkg/doc

- From MO-004/MO-006: extract a short, stable “session lifecycle + cancellation” topic doc (keep the long compendium as ttmp reference).
- From MO-005: ensure `geppetto/pkg/doc/topics/04-events.md` and `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md` do not reference removed APIs (e.g., `engine.WithSink`) and clearly warn about duplicate sink attachment.
- From MO-004 playbook: keep the “testing inference via examples” section as a stable playbook (either keep as-is in ttmp and link, or copy into `geppetto/pkg/doc/playbooks/`).

### Move/merge into pinocchio/pkg/doc

- Ensure the webchat framework guide links to:
  - geppetto session migration playbook,
  - events/sinks docs,
  - the pinocchio webchat “backend reference” and “debugging” docs (once migrated).

## Recommended execution plan (docs work)

1) Create new pinocchio docs (stubs) for: architecture, backend reference, backend internals, debugging/ops, frontend integration, SEM+UI.
2) For each doc, copy the go-go-mento version and do a “mechanical pass”:
   - update file paths,
   - remove identity/product-specific sections,
   - replace “InferenceState” terminology with “Session / ExecutionHandle / session_id/inference_id”.
3) Add links from `pinocchio/pkg/doc/topics/webchat-framework-guide.md` and `pinocchio/cmd/web-chat/README.md` to the new docs.
4) Update `go-go-mento/docs/reference/webchat/README.md` with a “Moved to pinocchio” notice + links.
5) Optional: extract condensed “core lifecycle” and “sink wiring” docs from MO-006/MO-005 into `geppetto/pkg/doc/topics/`.

## Appendix: concrete evidence pointers (current code)

- Pinocchio backend components:
  - `pinocchio/pkg/webchat/stream_coordinator.go`
  - `pinocchio/pkg/webchat/connection_pool.go`
  - `pinocchio/pkg/webchat/router.go`
  - `pinocchio/pkg/webchat/sem_translator.go`
- Pinocchio frontend integration surface:
  - `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
  - `pinocchio/cmd/web-chat/web/src/sem/registry.ts`
  - `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`
- Geppetto stable docs roots:
  - `geppetto/pkg/doc/topics/00-docs-index.md`
  - `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md`
