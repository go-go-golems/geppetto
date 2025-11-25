---
Title: go-go-mento Webchat/Web hydration — implementation reference
Ticket: MEN-3083-part-2
Status: active
Topics:
  - frontend
  - conversation
  - events
DocType: reference
Intent: long-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Server emits SEM frames; frontend hydrates from Pinocchio snapshots via HTTP and gates updates by version. This doc lists key files and symbols.
LastUpdated: 2025-11-04
---

# go-go-mento Webchat/Web hydration — files and symbols

## 1) Purpose

Document how go-go-mento’s server and web client integrate with persistent snapshots from Pinocchio, and list the concrete files and symbols to reference when evolving hydration and streaming behavior.

## 2) Server (go-go-mento/go/pkg/webchat)

- `forwarder.go` — SEM mapping
  - `SemanticEventsFromEvent(e events.Event) [][]byte`
    - Emits typed SEM frames; ensures stable entity IDs for domain features
    - Team Analysis: uses `analysis_id` to build stable ids and maps:
      - `team.analysis.start`, `.progress`, `.relationship`, `.result`
    - LLM events: `llm.start`, `llm.delta` (noise-stripped), `llm.final`
    - Tool events: `tool.start`, `tool.delta`, `tool.result`, `tool.done`
    - Agent mode: `agent.mode`

- `router.go` — routes and conversation lifecycle
  - `WebsocketHandler()` serves `GET /rpc/v1/chat/ws?conv_id=...&profile=...`
  - `ChatHandler()` serves `POST /chat` (and `/chat/{profile}` via `ChatProfileHandler()`)
  - Profile resolution via cookie `chat_profile` or query path
  - Engine composition from parsed layers and profile defaults
  - No projector/snapshot persistence here (hydration endpoint not provided)

- `conversation.go` — reader and broadcast
  - `startReader(...)` subscribes to topic `chat:<conv_id>`
  - `convertAndBroadcast(...)` wraps `SemanticEventsFromEvent(...)` and broadcasts frames to sockets
  - Conversation state fields: `ID`, `RunID`, `Turn`, `Eng`, `Sink`, socket set, etc.

Notes:
- Server emits SEM frames only; persistent snapshots + hydration come from Pinocchio (`/api/conversations/{convId}/timeline`).

## 3) Frontend (go-go-mento/web/src)

- `hooks/useChatStream.ts`
  - Connects to `ws://<host>/rpc/v1/chat/ws?conv_id=...` and dispatches streaming updates
  - On `ws.onopen`, dispatches `hydrateTimelineThunk(convId)`
  - Maps SEM types to Redux actions:
    - LLM: `addEntity` → streaming, `appendMessageText`, `finalizeMessage`
    - Tool: `addEntity`, `updateToolProgress`, `tool.result` → `addEntities` + complete
    - Team Analysis (typed): maps start/progress/result using `analysis_id` as stable `entityId`
    - Logs and agent mode → `status` entities

- `store/timeline/timelineSlice.ts`
  - Version-gated upsert in `upsertEntity`:
    - Accept only when `incoming.version > existing.version`
    - If both versions absent, shallow prop merge (streaming path)
  - Hydration thunk `hydrateTimelineThunk(convId)`:
    - `GET /api/conversations/{convId}/timeline`
    - `mapSnapshotToEntity(snap)` converts typed snapshots (server schema) → local `TimelineEntity`
    - Dispatches `upsertEntity` per snapshot
  - Entity helpers: `addEntity`, `addEntities`, `appendMessageText`, `finalizeMessage`, `updateToolProgress`

- `pages/Chat/timeline/types.ts`
  - Defines `BaseTimelineEntity` (includes optional `version`) and domain entity variants (`MessageEntity`, `ToolCallEntity`, `ToolResultEntity`, etc.)

## 4) Cross-system contract (Pinocchio → go-go-mento)

- Hydration API: `GET /api/conversations/{convId}/timeline?sinceVersion=...`
- Snapshot JSON (typed): includes `entity_id`, `kind`, `version`, plus kind-specific fields
- Client rules:
  - Upsert by `entity_id`
  - Accept newer `version` only; ignore older/equal versions
  - Treat hydrated `llm_text` as finalized (non-streaming)

## 5) Key symbols (quick reference)

- Server
  - `SemanticEventsFromEvent` (webchat/forwarder.go)
  - `WebsocketHandler`, `ChatHandler`, `ChatProfileHandler` (webchat/router.go)
  - `startReader`, `convertAndBroadcast` (webchat/conversation.go)

- Frontend
  - `useChatStream` (hooks/useChatStream.ts)
  - `hydrateTimelineThunk`, `upsertEntity`, `mapSnapshotToEntity` (store/timeline/timelineSlice.ts)
  - `BaseTimelineEntity`, `MessageEntity`, `ToolCallEntity` (pages/Chat/timeline/types.ts)

## 6) Observations & gaps

- go-go-mento server does not implement hydration endpoints; relies on Pinocchio (or a proxy) for `/api/conversations/...`
- Team Analysis uses typed SEM events with stable correlation `analysis_id`, aligning with snapshot entity IDs
- Ensure deployments route `/api/conversations/*` to Pinocchio’s hydration service


