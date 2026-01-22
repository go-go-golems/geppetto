---
Title: 'Moments web: run_id usage audit'
Ticket: GP-02-REMOVE-TURN-RUN-ID
Status: active
Topics:
    - geppetto
    - turns
    - inference
    - refactor
    - design
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../moments/web/docs/event-driven-widgets.md
      Note: Docs describe run_id as a SEM/protobuf correlation bridge
    - Path: ../../../../../../../moments/web/src/platform/api/chatApi.ts
      Note: Chat start response still returns run_id
    - Path: ../../../../../../../moments/web/src/platform/sem/handlers/planning.ts
      Note: Planning widgets key aggregates by runId (protobuf run_id)
ExternalSources: []
Summary: Inventory and migration guidance for `run_id` / `runId` usage in Moments’ web frontend.
LastUpdated: 2026-01-22T16:55:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Moments web: `run_id` usage audit

## Goal

Document where Moments’ web frontend uses `run_id` / `runId`, what it means in each context, and what
needs to stay stable while the backend migrates to canonical `{session_id,inference_id,turn_id}`.

This is written in the context of the GP-02 refactor, where:

- **SessionID** is the long-lived multi-turn session identifier.
- **InferenceID** is unique per single `RunInference` call (one tool-loop execution).
- We keep `run_id` as a legacy serialized/log field name in some places for backwards compatibility.

## Inventory

### API responses (`run_id`)

- `moments/web/src/platform/api/chatApi.ts` declares:
  - `StartChatResponse.run_id: string`
  - `StartChatResponse.conv_id: string`

This matches the existing backend contract for `POST /rpc/v1/chat`.

### Typed SEM widgets (`runId` / protobuf `run_id`)

- `moments/web/src/platform/sem/handlers/planning.ts` aggregates and de-dupes planning/execution SEM
  frames by a `runId` string, derived from:
  - `pb.run?.runId` (protobuf JSON) or
  - `ev.id` (SEM frame id).
- `moments/web/src/platform/timeline/types.ts` uses `PlanningEntity.props.runId` and also uses
  `PlanningEntity.id` as the aggregate key (set to `runId` in the handler).
- `moments/web/src/platform/sem/pb/proto/sem/middleware/planning_pb.ts` is generated code where the
  wire field is `run_id` but the TS accessor is `runId`.

### Mocks + storybook (`run_id`)

- `moments/web/src/mocks/chatHandlers.ts` returns `run_id` in chat start mock responses.
- `moments/web/src/features/chat/ChatPage.stories.tsx` includes story responses with `run_id`.

### “Run” naming that is unrelated to correlation IDs

There are other `runId` variables in the web codebase that mean “attempt number” / “exchange run” and
are not part of the backend correlation contract (e.g. OAuth flow retry tracking). Those are out of
scope for this ticket.

## Findings (what `run_id` means today)

### 1) Chat start response: `run_id` is treated as “chat session correlation id”

The chat start API returns `{run_id, conv_id}`. In the current frontend code, `conv_id` is the
primary handle used to select/subscribe to a conversation, while `run_id` is mainly preserved as a
returned value and used in mocks/stories.

Implication: the frontend is likely resilient to adding `session_id` alongside `run_id` without
breaking UI behavior, because `run_id` does not appear to be a primary indexing key in the web app.

### 2) Planning widget: protobuf `run_id` is a de-dupe key, not necessarily a “session id”

The planning SEM handler treats `runId` as the identity of a planning/execution aggregate:

- It creates/updates a `PlanningAggregate` keyed by `runId`.
- It sets `PlanningEntity.id = runId`.
- It uses it to merge `planning.*` and `execution.*` updates into one timeline widget.

This `runId` is sourced from the planning protobuf payload (`pb.run.runId`) or the SEM event id.
Semantically, this is closer to “this one planning/execution run” than “the multi-turn chat session”.

Implication: if we move the backend to canonical IDs, the planning pipeline likely wants to map
**planning.run_id → InferenceID** (or a new, explicitly-named “analysis id”) rather than SessionID.

### 3) Protobuf contracts embed `run_id` on the wire

Because `run_id` is a protobuf field name, renaming it would be a breaking change for the
forwarder/server/client trio unless we do a staged migration (new field added first, dual-stamping,
then remove old field in a major version).

## Recommendations (migration path)

### Short-term (compatible; preferred)

1. **Keep `run_id` in the chat start response** and treat it as a legacy alias for **SessionID**.
2. Add (and document) a `session_id` field on the same response:
   - The backend can return both now; the frontend can accept both.
3. For planning SEM widgets:
   - Keep protobuf `run_id` stable for now.
   - Ensure it is consistently derived from the per-inference identifier (InferenceID), not from the
     long-lived SessionID.

### Medium-term (cleanup)

1. Introduce explicit fields in planning protobuf messages:
   - `inference_id` (or `analysis_id`) alongside `run_id`.
2. Update the forwarder + web handler to key aggregates by the new field.
3. Deprecate `run_id` and remove it in a major schema bump once all clients have migrated.

## Cross-repo linkage (where to look next)

- Moments backend chat router should keep returning `run_id` (legacy) while also returning canonical
  `session_id` and `inference_id`.
- Geppetto canonical semantics are documented in:
  - `geppetto/ttmp/2026/01/22/GP-02-REMOVE-TURN-RUN-ID--remove-turn-runid-store-sessionid-in-turn-metadata/reference/02-refactor-postmortem.md`
