---
Title: 'Moments backend: rename RunID to SessionID'
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
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T15:50:41.875609067-05:00
WhatFor: ""
WhenToUse: ""
---

# Moments backend: rename RunID to SessionID

## Goal

Remove the legacy **RunID** naming from `moments/backend` and replace it with the canonical **SessionID** naming used by Geppetto’s session API and `events.EventMetadata` (`SessionID`, `InferenceID`, `TurnID`).

This is a *naming + plumbing* refactor: it should not change behavior, only eliminate ambiguity.

## Background (current truth)

- In Geppetto, **SessionID** is the long-lived multi-turn identifier (stored on `Turn.Metadata` via `turns.KeyTurnMetaSessionID`).
- **InferenceID** is unique per single inference execution (per `RunInference` call) (stored via `turns.KeyTurnMetaInferenceID`).
- Many parts of Moments still use:
  - `Conversation.RunID` to mean *session id*,
  - `FindConversationByRunID` to mean *lookup by session id*,
  - and `events.EventMetadata.RunID` (which no longer exists in Geppetto).

## Inventory: where `RunID` exists in `moments/backend`

### 1) Webchat core

- `backend/pkg/webchat/conversation.go`
  - `Conversation.RunID` field (actually session id)
  - `ConvManager.FindConversationByRunID`
  - various seed/metadata wiring via `turns.KeyTurnMetaSessionID.Set(&conv.Turn.Metadata, conv.RunID)`
- `backend/pkg/webchat/router.go`
  - log fields and API responses referencing `conv.RunID`
  - session-id seeding that still uses `conv.RunID`
- `backend/pkg/webchat/loops.go`
  - manually constructs `geppetto/pkg/events.EventMetadata{RunID: ...}` (needs `SessionID`)
  - also uses `conv.RunID` as fallback when turn metadata is missing
- `backend/pkg/webchat/forwarder.go`
  - logs `md.RunID` (needs `md.SessionID`)
- `backend/pkg/webchat/log_blocks_middleware.go`, `backend/pkg/webchat/langfuse_middleware.go`, `backend/pkg/webchat/moments_global_prompt_middleware.go`
  - build event metadata or log with `runID` variables; many of these also instantiate `events.EventMetadata{RunID: ...}`.

### 2) Conversation lookup adapters (API surface)

These interfaces and adapters hard-code RunID naming even though it’s “session id”:

- `backend/pkg/app/sink_registry.go`
  - `convManagerAdapter.FindConversationByRunID`
  - `memoryConvManagerAdapter.FindConversationByRunID`
- `backend/pkg/doclens/doc_suggestions_sink.go`
  - interface `FindConversationByRunID`
- `backend/pkg/teamchat/team_suggestions_sink.go`
  - interface `FindConversationByRunID`
- `backend/pkg/memory/extractor.go`
  - interface `FindConversationByRunID`
  - fallback path: “derive convID via RunID” (should become via SessionID)
- `backend/pkg/inference/middleware/teamselection/*`
  - interface/comments/logic around `FindConversationByRunID`

### 3) SEM handlers & tests

- `backend/pkg/sem/handlers/tool_handlers.go`
  - uses `md.RunID` to maintain `runToTurn` cache (should become `SessionID` and `sessionToTurn`)
- `backend/pkg/sem/handlers/tool_cache_test.go`
  - uses `events.EventMetadata{RunID: ...}` (should become `SessionID`)

### 4) Artifact & autosummary packages

- `backend/pkg/artifact/middleware.go`
  - constructs structures containing `RunID` fields (needs rename)
- `backend/pkg/artifact/buffer.go`
  - uses `"run_id"` log keys; variable naming likely `runID`
- `backend/pkg/autosummary/summary_client.go`
  - response struct has `RunID string \`json:"run_id"\``
  - this is *potentially* an external contract; we can rename the Go field to `SessionID` while keeping the JSON tag `run_id` if needed.

## Design decisions for the refactor

### Naming policy

- Replace Go identifiers:
  - `RunID` → `SessionID` (fields, locals, function names, interface methods)
  - `FindConversationByRunID` → `FindConversationBySessionID`
- For protocol/log compatibility:
  - Prefer **adding** `session_id` outputs/fields and keeping `run_id` only where required by an external consumer.
  - If a consumer is unknown, default to returning both (`run_id` legacy alias + `session_id` canonical) and remove the alias later as a follow-up.

### InferenceID propagation (Moments-specific)

Moments has custom loops (e.g. `backend/pkg/webchat/loops.go`) that call `eng.RunInference` directly and emit tool events manually. Unlike Geppetto’s `session.ToolLoopEngineBuilder`, these paths currently do **not** set `turns.KeyTurnMetaInferenceID`.

As part of “new naming”, we should also:
- Read `InferenceID` from `Turn.Metadata` when constructing `events.EventMetadata`.
- Ensure `InferenceID` is set when missing in these custom loops (best-effort, no behavior change beyond adding correlation).

## Proposed implementation order (minimizes churn)

1. Rename `Conversation.RunID` → `Conversation.SessionID` and update all references in `backend/pkg/webchat/*`.
2. Rename `ConvManager.FindConversationByRunID` → `FindConversationBySessionID` and update all interfaces/adapters/sinks.
3. Replace all `events.EventMetadata{RunID: ...}` and `md.RunID` usage with `SessionID` (and also wire `InferenceID` from turn metadata where possible).
4. Update SEM handler caches (`runToTurn` → `sessionToTurn`) and fix tests.
5. Update `autosummary/summary_client.go` field naming (Go field name `SessionID`, JSON tag left as `run_id` if needed).
6. Run `go test ./...` and `make lint` in `moments/backend`.

## Notes / risks

- This is a wide mechanical refactor; the main risk is missing one `RunID` reference across interface boundaries.
- If `moments/web` or external clients depend on `run_id` response fields, switching to `session_id` only would be a breaking change; prefer dual-field responses until consumers are updated.
