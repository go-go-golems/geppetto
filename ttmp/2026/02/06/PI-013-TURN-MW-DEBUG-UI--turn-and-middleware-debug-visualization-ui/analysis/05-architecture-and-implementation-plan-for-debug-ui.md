---
Title: Architecture and Implementation Plan for Debug UI
Ticket: PI-013-TURN-MW-DEBUG-UI
Status: active
Topics:
    - websocket
    - middleware
    - turns
    - events
    - frontend
    - react
    - redux
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/02/06/PI-013-TURN-MW-DEBUG-UI--turn-and-middleware-debug-visualization-ui/sources/local/ui-design-turn.md
      Note: UI wireframes this plan implements
    - Path: geppetto/ttmp/2026/02/06/PI-013-TURN-MW-DEBUG-UI--turn-and-middleware-debug-visualization-ui/analysis/01-turn-and-middleware-debug-ui-requirements-and-ux-specification.md
      Note: Requirements spec this plan derives from
    - Path: pinocchio/pkg/webchat/router.go
      Note: Existing REST API endpoints to extend
    - Path: pinocchio/pkg/webchat/turn_store.go
      Note: TurnStore interface (turn snapshots)
    - Path: pinocchio/pkg/webchat/turn_store_sqlite.go
      Note: SQLite turn store implementation
    - Path: pinocchio/pkg/webchat/timeline_store.go
      Note: TimelineStore interface (projection entities)
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: SEM frame to timeline entity conversion
    - Path: pinocchio/pkg/webchat/sem_buffer.go
      Note: In-memory circular buffer of recent SEM frames
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Conversation struct with all per-conversation state
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: EngineBuilder interface and BuildConfig/BuildFromConfig
    - Path: geppetto/pkg/turns/types.go
      Note: Turn, Block, Metadata, Data types
    - Path: geppetto/pkg/turns/keys.go
      Note: Canonical metadata keys including middleware attribution
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Snapshot phase invocations (pre_inference, post_inference, post_tools)
    - Path: geppetto/pkg/inference/middleware/middleware.go
      Note: Middleware interface (HandlerFunc wrapping)
ExternalSources: []
Summary: Architecture and phased implementation plan for debug UI, updated with post-review decisions on correlation contract, endpoint migration, separate app location, diff strategy, acceptance tests, and run→session terminology migration (PI-017).
LastUpdated: 2026-02-07T09:50:00-05:00
WhatFor: Translate wireframes into an implementation-ready engineering plan with explicit migration and validation constraints.
WhenToUse: Use when implementing or reviewing debug UI backend/frontend work after the PI-013 engineering review decisions.
---

# Architecture and Implementation Plan for Debug UI

This document translates the [UI wireframes](../sources/local/ui-design-turn.md) into a concrete architecture and step-by-step implementation plan. It covers the backend REST API, React + Redux Toolkit frontend, and the phased build order.

## Decision Addendum (2026-02-07)

The following decisions are now part of this plan:

1. **Critical 1**: Event/snapshot joining must be deterministic via an explicit correlation contract, not heuristic matching.
2. **High 3**: Migrate to `/debug` route namespace with no backwards compatibility for legacy `/turns` and `/timeline` routes.
3. **High 4**: Use a fully separate debug app rooted at `web-agent-example/cmd/web-agent-debug`.
4. **Medium 1**: Snapshot diffing must be identity-aware and detect reorder operations (not index-only).
5. **Medium 4**: Add explicit acceptance tests for correlation correctness, ordering, migration, and tracing coverage.
6. **High 2**: Security hardening concerns are explicitly deprioritized for this scope.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [What Already Exists](#what-already-exists)
3. [What Needs to Be Built](#what-needs-to-be-built)
4. [Backend REST API Design](#backend-rest-api-design)
5. [Frontend Architecture](#frontend-architecture)
6. [Redux State Shape](#redux-state-shape)
7. [Screen-by-Screen Implementation](#screen-by-screen-implementation)
8. [Implementation Phases](#implementation-phases)
9. [Middleware Tracing (Backend Extension)](#middleware-tracing)
10. [Anomaly Detection](#anomaly-detection)
11. [Risk Register](#risk-register)
12. [Decision Addendum (2026-02-07)](#decision-addendum-2026-02-07)
13. [Acceptance Test Matrix](#acceptance-test-matrix)

---

## Architecture Overview

```
+---------------------------------------------------------------------------+
|  Debug UI (React + Redux Toolkit + React Router)                          |
|                                                                           |
|  +----------+  +--------------+  +--------------+  +------------------+  |
|  | Session  |  | Turn         |  | Event        |  | Middleware       |  |
|  | Overview |  | Inspector    |  | Inspector    |  | Chain View       |  |
|  | (Scr 1)  |  | (Scr 2+3)   |  | (Scr 5)     |  | (Scr 4)         |  |
|  +----+-----+  +------+-------+  +------+-------+  +------+-----------+  |
|       |               |                 |                  |              |
|       v               v                 v                  v              |
|  +--------------------------------------------------------------------+  |
|  |                    Redux Store                                      |  |
|  |  sessionsSlice . turnsSlice . eventsSlice . middlewareSlice         |  |
|  |  filtersSlice . anomaliesSlice . uiSlice                           |  |
|  +--------------------------------+-----------------------------------+  |
|                                   |                                      |
|  +--------------------------------+-----------------------------------+  |
|  |                    API Layer (RTK Query)                            |  |
|  |  useGetConversationsQuery . useGetTurnsQuery . useGetEventsQuery   |  |
|  |  useGetTimelineQuery . useGetMiddlewareTraceQuery                  |  |
|  +--------------------------------+-----------------------------------+  |
+-----------------------------------+--------------------------------------+
                                    | HTTP + WebSocket
                                    v
+---------------------------------------------------------------------------+
|  Backend (pinocchio/pkg/webchat)                                          |
|                                                                           |
|  +----------------+  +----------------+  +----------------------------+  |
|  | /debug/convs   |  | /debug/turns   |  | /debug/events             |  |
|  | /debug/conv/:id|  | /debug/timeline|  | /debug/mw-trace           |  |
|  +-------+--------+  +-------+--------+  +-------+--------------------+  |
|          |                   |                    |                       |
|          v                   v                    v                       |
|  +------------+      +------------+      +----------------+              |
|  | ConvManager |      | TurnStore  |      | semFrameBuffer |              |
|  | (in-memory) |      | (SQLite)   |      | (in-memory)    |              |
|  +------------+      +------------+      +----------------+              |
|                                                                           |
|  +------------+      +----------------+  +----------------------------+  |
|  | Session    |      | TimelineStore  |  | MiddlewareTraceStore       |  |
|  | (in-memory)|      | (SQLite)       |  | (NEW - SQLite)             |  |
|  +------------+      +----------------+  +----------------------------+  |
+---------------------------------------------------------------------------+
```

---

## What Already Exists

| Component | Location | What It Provides |
|-----------|----------|-----------------|
| **TurnStore** | `pinocchio/pkg/webchat/turn_store_sqlite.go` | Turn snapshots by conv_id/session_id/phase with YAML payloads |
| **TimelineStore** | `pinocchio/pkg/webchat/timeline_store_sqlite.go` | Timeline entities (messages, tool calls) with version-based queries |
| **SEM Frame Buffer** | `pinocchio/pkg/webchat/sem_buffer.go` | Last 1000 SEM frames per conversation (in-memory circular buffer) |
| **ConvManager** | `pinocchio/pkg/webchat/conversation.go` | All active conversations with their state |
| **`GET /turns`** | `pinocchio/pkg/webchat/router.go:670` | Query turn snapshots by conv_id, session_id, phase, since_ms |
| **`GET /timeline`** | `pinocchio/pkg/webchat/router.go:620` | Query timeline entities by conv_id, since_version |
| **`GET /ws`** | `pinocchio/pkg/webchat/router.go:500` | WebSocket for live SEM events |
| **`POST /debug/step/*`** | `pinocchio/pkg/webchat/router.go:386` | Step-mode debug controls |
| **Snapshot Hooks** | `geppetto/pkg/inference/toolloop/loop.go:126,131,160` | pre_inference, post_inference, post_tools phases |
| **Block Metadata** | `geppetto/pkg/turns/keys.go` | `geppetto.middleware@v1` key for middleware attribution |
| **Turn YAML Serde** | `geppetto/pkg/turns/serde/serde.go` | Serialize/deserialize turns as YAML |

### Existing REST API Summary

| Method | Path | Purpose | Store |
|--------|------|---------|-------|
| `GET` | `/turns` | List turn snapshots | TurnStore (SQLite) |
| `GET` | `/timeline` | List timeline entities | TimelineStore (SQLite) |
| `GET` | `/ws` | WebSocket stream | SEM buffer + live events |
| `POST` | `/chat` | Submit prompt | ConvManager |
| `GET` | `/api/chat/profiles` | List profiles | ProfileRegistry |
| `POST` | `/debug/step/enable` | Enable step mode | StepController |
| `POST` | `/debug/step/disable` | Disable step mode | StepController |
| `POST` | `/debug/continue` | Resume from pause | StepController |

---

## What Needs to Be Built

### Backend (New Endpoints)

| Endpoint | Purpose | Screen | Data Source |
|----------|---------|--------|-------------|
| `GET /debug/turns` | Canonical turn snapshot endpoint (migrated from `/turns`) | 1, 2, 3 | TurnStore |
| `GET /debug/timeline` | Canonical timeline endpoint (migrated from `/timeline`) | 1, 5 | TimelineStore |
| `GET /debug/conversations` | List all conversations with metadata | 1 | ConvManager |
| `GET /debug/conversation/:id` | Single conversation detail | 1 | ConvManager + TurnStore |
| `GET /debug/conversation/:id/sessions` | List sessions within a conversation | 1 | TurnStore (distinct session_ids) |
| `GET /debug/events/:conv_id` | List SEM frames for a conversation | 1, 5 | semFrameBuffer |
| `GET /debug/turn/:conv_id/:session_id/:turn_id` | All phases for one turn | 2 | TurnStore |
| `GET /debug/mw-trace/:conv_id/:inference_id` | Middleware chain trace | 4 | MiddlewareTraceStore (NEW) |

### Backend (New Stores)

| Store | Purpose | Schema |
|-------|---------|--------|
| **MiddlewareTraceStore** | Records before/after turn state per middleware layer | `(conv_id, inference_id, layer_index, mw_name, pre_yaml, post_yaml, duration_ms)` |

### Backend (New Infrastructure)

| Component | Purpose |
|-----------|---------|
| **Tracing Middleware Wrapper** | Wraps each middleware to capture before/after Turn snapshots |
| **Final persistence lineage field** | Keep existing persister-based `final` snapshot and mark source/provenance explicitly |

### Frontend (New Application)

The debug UI is a **fully separate SPA** (not embedded in the chat UI), located at:

- `web-agent-example/cmd/web-agent-debug`

It uses the same backend APIs and websocket event stream, but has independent build/deploy lifecycle.

---

## Backend REST API Design

All debug endpoints are gated behind `PINOCCHIO_WEBCHAT_DEBUG=1`.

Migration decision:

- Canonical endpoints become `/debug/*` only.
- Legacy `/turns` and `/timeline` routes are removed (no compatibility aliases).

### `GET /debug/conversations`

Lists all active conversations.

**Response:**
```json
{
  "conversations": [
    {
      "id": "conv_8a3f",
      "profile_slug": "general",
      "session_id": "sess_02",
      "engine_config_sig": "...",
      "is_running": false,
      "ws_connections": 2,
      "last_activity": "2026-02-06T14:32:18Z",
      "turn_count": 3,
      "has_timeline": true
    }
  ]
}
```

**Handler -> Store mapping:**
```
ConvManager.mu.Lock()
for id, conv := range cm.conns {
    // Read: conv.ID, conv.ProfileSlug, conv.SessionID, conv.EngConfigSig
    // Read: conv.pool.Count(), conv.lastActivity
    // Read: len(conv.Sess.Turns) for turn count
    // Read: conv.timelineProj != nil for has_timeline
}
```

### `GET /debug/conversation/:id`

Returns full conversation metadata including engine config and session info.

**Response:**
```json
{
  "id": "conv_8a3f",
  "profile_slug": "general",
  "session_id": "sess_abc",
  "engine_config": {
    "profile_slug": "general",
    "system_prompt": "You are...",
    "middlewares": [{"name": "logging-mw"}, {"name": "system-prompt-mw"}],
    "tools": ["get_weather_forecast"]
  },
  "turn_count": 3,
  "is_running": false,
  "ws_connections": 2
}
```

**Handler -> Store mapping:**
```
conv := cm.Get(id)
// Read: conv.Sess.SessionID for session_id
// Read: conv.EngineConfig (need to store full config, not just signature)
// Read: all fields from GET /debug/conversations plus engine config
```

**Conversation struct change:** Store the full `EngineConfig` (not just signature) on `Conversation` so the debug endpoint can return it.

### `GET /debug/conversation/:id/sessions`

Lists distinct sessions within a conversation.

**Response:**
```json
{
  "sessions": [
    {
      "session_id": "sess_01",
      "turn_count": 1,
      "first_turn_at": "2026-02-06T14:32:01Z",
      "last_turn_at": "2026-02-06T14:32:05Z"
    },
    {
      "session_id": "sess_02",
      "turn_count": 2,
      "first_turn_at": "2026-02-06T14:32:08Z",
      "last_turn_at": "2026-02-06T14:32:18Z"
    }
  ]
}
```

**Handler -> Store mapping:**
```
snapshots := turnStore.List(ctx, TurnQuery{ConvID: id})
// Group by session_id, count turns, extract min/max created_at_ms
```

**TurnStore change needed:** Add a `DistinctSessions(ctx, convID) ([]SessionSummary, error)` method, or compute from `List` results.

### `GET /debug/events/:conv_id`

Returns SEM frames from the in-memory buffer. Optionally filters by type and supports pagination.

**Query params:** `?type=llm.delta&since_seq=N&limit=100`

**Response:**
```json
{
  "events": [
    {
      "type": "llm.start",
      "id": "msg-a1b2c3d4",
      "seq": 1707053365100000000,
      "stream_id": "main",
      "data": {"role": "assistant"},
      "received_at": "2026-02-06T14:32:08.123Z"
    }
  ],
  "total": 47,
  "buffer_capacity": 1000
}
```

**Handler -> Store mapping:**
```
conv := cm.Get(convID)
frames := conv.semBuf.Snapshot()
// Parse each frame JSON, filter by type/seq, paginate
```

**semFrameBuffer change:** Store arrival timestamp alongside each frame. Add `SnapshotParsed()` method that returns structured frame data (currently stores raw `[]byte`).

### `GET /debug/turn/:conv_id/:session_id/:turn_id`

Returns all snapshot phases for a single turn, with parsed block data.

**Response:**
```json
{
  "conv_id": "conv_8a3f",
  "session_id": "sess_02",
  "turn_id": "turn_02",
  "phases": {
    "pre_inference": {
      "captured_at": "2026-02-06T14:32:08.001Z",
      "turn": {
        "id": "turn_02",
        "blocks": [
          {
            "index": 0,
            "kind": "system",
            "role": "system",
            "payload": {"text": "..."},
            "metadata": {"geppetto.middleware@v1": "system-prompt-mw"}
          },
          {
            "index": 1,
            "kind": "user",
            "role": "user",
            "payload": {"text": "..."}
          }
        ],
        "metadata": {
          "geppetto.session_id@v1": "sess_abc",
          "geppetto.inference_id@v1": "inf_b7e2"
        },
        "data": {}
      }
    },
    "post_inference": { "..." : "..." },
    "post_tools": { "..." : "..." },
    "final": { "..." : "..." }
  }
}
```

**Handler -> Store mapping:**
```
snapshots := turnStore.List(ctx, TurnQuery{
    ConvID:    convID,
    SessionID: sessionID,
})
// Filter to matching turn_id, group by phase
// Parse YAML payload into structured Turn JSON
// Convert block kinds to strings, include block indices
```

**TurnStore change:** The YAML payloads need to be parsed server-side into JSON for the frontend. Add a `TurnQuery.TurnID` filter field.

### `GET /debug/mw-trace/:conv_id/:inference_id`

Returns the middleware chain execution trace for a specific inference.

**Response:**
```json
{
  "conv_id": "conv_8a3f",
  "inference_id": "inf_b7e2",
  "chain": [
    {
      "layer": 0,
      "name": "logging-mw",
      "pre_blocks": 4,
      "post_blocks": 4,
      "blocks_added": 0,
      "blocks_removed": 0,
      "blocks_changed": 0,
      "metadata_changes": ["geppetto.usage@v1"],
      "duration_ms": 1
    },
    {
      "layer": 1,
      "name": "system-prompt-mw",
      "pre_blocks": 4,
      "post_blocks": 4,
      "blocks_added": 0,
      "blocks_removed": 0,
      "blocks_changed": 1,
      "changed_blocks": [{"index": 0, "kind": "system", "change": "content_modified"}],
      "duration_ms": 2
    }
  ],
  "engine": {
    "model": "claude-3.5-sonnet",
    "input_blocks": 4,
    "output_blocks": 6,
    "latency_ms": 2412,
    "tokens_in": 847,
    "tokens_out": 312,
    "stop_reason": "end_turn"
  }
}
```

**Handler -> Store mapping:**
```
traces := mwTraceStore.ListByInference(ctx, convID, inferenceID)
// Each trace has pre_yaml and post_yaml
// Compute block diffs: compare block lists, identify adds/removes/changes
// Compute metadata diffs
```

This endpoint requires the **MiddlewareTraceStore** (new backend store, see [Middleware Tracing](#middleware-tracing) section).

### `GET /debug/mw-trace/:conv_id/:inference_id/:layer`

Returns the full turn YAML for a specific middleware trace snapshot (for the diff view).

**Response:**
```json
{
  "layer": 1,
  "name": "system-prompt-mw",
  "pre_turn": { "id": "...", "blocks": [...], "metadata": {...} },
  "post_turn": { "id": "...", "blocks": [...], "metadata": {...} }
}
```

---

## Frontend Architecture

### Technology Stack

| Concern | Technology |
|---------|-----------|
| Framework | React 18+ |
| State management | Redux Toolkit (RTK) + RTK Query |
| Routing | React Router v6 |
| Styling | CSS Modules or Tailwind (match existing webchat convention) |
| JSON diff | `deep-diff` or custom block-level diff |
| YAML display | `react-syntax-highlighter` for raw view |
| Build | Vite (same as existing webchat) |

### Component Tree

```
<DebugApp>
  <AppShell>
    <Sidebar>
      <SessionList />           <- /debug/conversations
    </Sidebar>
    <MainContent>
      <Routes>
        <Route path="/"
               element={<SessionOverview />} />                    <- Screen 1
        <Route path="/turn/:convId/:sessionId/:turnId"
               element={<TurnInspector />} />                     <- Screen 2
        <Route path="/diff/:convId/:sessionId/:turnId"
               element={<SnapshotDiff />} />                      <- Screen 3
        <Route path="/mw/:convId/:inferenceId"
               element={<MiddlewareChain />} />                   <- Screen 4
        <Route path="/event/:convId/:seq"
               element={<EventInspector />} />                    <- Screen 5
        <Route path="/sink/:convId/:turnId"
               element={<StructuredSinkView />} />                <- Screen 6
      </Routes>
    </MainContent>
    <FilterBar />               <- Screen 7 (overlay)
    <AnomalyPanel />            <- Screen 8 (slide-out)
  </AppShell>
</DebugApp>
```

### Key Components per Screen

#### Screen 1: Session Overview
```
<SessionOverview>
  <CorrelationIdBar convId sessionId />
  <TimelineLanes>
    <StateTrackLane>              <- Turn snapshots as vertical cards
      <TurnPhaseCard turn phase />
    </StateTrackLane>
    <EventTrackLane>              <- SEM events as a vertical list
      <EventDot event />
    </EventTrackLane>
    <ProjectionLane>              <- Timeline entities
      <EntityCard entity />
    </ProjectionLane>
  </TimelineLanes>
  <NowMarker />                   <- Live streaming indicator
</SessionOverview>
```

#### Screen 2: Turn Inspector
```
<TurnInspector>
  <CorrelationIdBar convId inferenceId turnId />
  <PhaseTabBar phases activePhase onSelect />
  <CompareDropdown phaseA phaseB onDiff />
  <BlockList>
    <BlockCard index block provenance isNew />
      <RawJsonToggle payload />
    </BlockCard>
  </BlockList>
  <TurnDetailsPanel metadata data />
</TurnInspector>
```

#### Screen 3: Snapshot Diff
```
<SnapshotDiff>
  <DiffHeader phaseA phaseB />
  <SideBySideBlocks>
    <DiffBlockRow blockA blockB status />   <- status: same|added|removed|changed
  </SideBySideBlocks>
  <MetadataDiff metaA metaB />
  <DiffSummaryBar added removed changed reordered />
</SnapshotDiff>
```

#### Screen 4: Middleware Chain
```
<MiddlewareChain>
  <ChainHeader inferenceId mwCount duration />
  <OnionLayers>
    <MiddlewareLayer name layer>
      <PrePostSummary pre post />
      <InlineDiffLink />
    </MiddlewareLayer>
    <EngineCenter model latency tokens />
  </OnionLayers>
  <InvisibleChangesPanel metadata data />
</MiddlewareChain>
```

#### Screen 5: Event Inspector
```
<EventInspector>
  <CorrelationIdBar sessionId inferenceId turnId seq streamId />
  <ViewModeTabs active onSelect />     <- Semantic | SEM | Raw Wire
  <SemanticView event />               <- Human-readable event card
  <SemEnvelopeView envelope />         <- JSON viewer
  <RawWireView raw />                  <- Provider-native JSON
  <CorrelatedNodesPanel>
    <StateTrackLink block />
    <EventNeighbors prev next />
    <ProjectionLink entity />
  </CorrelatedNodesPanel>
  <TrustSignals checks />
</EventInspector>
```

#### Screen 6: Structured Sink View
```
<StructuredSinkView>
  <ThreeColumnLayout>
    <RawOutputColumn text highlights />
    <FilteredTextColumn text gaps />
    <ExtractedEventsColumn events />
  </ThreeColumnLayout>
  <ExtractionLog entries />
</StructuredSinkView>
```

---

## Redux State Shape

```typescript
interface DebugState {
  // RTK Query handles caching for API data.
  // These slices hold UI-specific and computed state.

  sessions: {
    selectedConvId: string | null;
    selectedSessionId: string | null;
  };

  turns: {
    selectedTurnId: string | null;
    selectedPhase: string;
      // 'pre_inference' | 'post_inference' | 'post_tools' | 'final'
    comparePhaseA: string | null;
    comparePhaseB: string | null;
  };

  events: {
    selectedSeq: number | null;
    viewMode: 'semantic' | 'sem' | 'raw';
    liveStreamEnabled: boolean;
  };

  filters: {
    eventTypes: string[];        // checked event types
    snapshotPhases: string[];    // checked phases
    middlewares: string[];       // checked middleware names
    blockKinds: string[];        // checked block kinds
  };

  anomalies: {
    pinned: Anomaly[];
    autoDetected: Anomaly[];
    panelOpen: boolean;
  };

  ui: {
    sidebarCollapsed: boolean;
    filterBarOpen: boolean;
    inspectorPanel: 'none' | 'event' | 'turn' | 'mw';
  };
}
```

### RTK Query API Definition

```typescript
const debugApi = createApi({
  reducerPath: 'debugApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/debug/' }),
  endpoints: (builder) => ({
    getConversations: builder.query<ConversationSummary[], void>({
      query: () => 'conversations',
    }),
    getConversation: builder.query<ConversationDetail, string>({
      query: (id) => `conversation/${id}`,
    }),
    getSessions: builder.query<SessionSummary[], string>({
      query: (convId) => `conversation/${convId}/sessions`,
    }),
    getTurns: builder.query<TurnSnapshot[], TurnQuery>({
      query: ({ convId, sessionId, phase, sinceMs, limit }) =>
        `turns?` + new URLSearchParams({
          ...(convId && { conv_id: convId }),
          ...(sessionId && { session_id: sessionId }),
          ...(phase && { phase }),
          ...(sinceMs && { since_ms: String(sinceMs) }),
          ...(limit && { limit: String(limit) }),
        }),
    }),
    getTimeline: builder.query<TimelineSnapshot, TimelineQuery>({
      query: ({ convId, sinceVersion, limit }) =>
        `timeline?` + new URLSearchParams({
          conv_id: convId,
          ...(sinceVersion && { since_version: String(sinceVersion) }),
          ...(limit && { limit: String(limit) }),
        }),
    }),
    getTurnDetail: builder.query<TurnDetail, TurnDetailQuery>({
      query: ({ convId, sessionId, turnId }) =>
        `turn/${convId}/${sessionId}/${turnId}`,
    }),
    getEvents: builder.query<EventsResponse, EventQuery>({
      query: ({ convId, type, sinceSeq, limit }) =>
        `events/${convId}?` + new URLSearchParams({
          ...(type && { type }),
          ...(sinceSeq && { since_seq: String(sinceSeq) }),
          ...(limit && { limit: String(limit) }),
        }),
    }),
    getMwTrace: builder.query<MwTrace, MwTraceQuery>({
      query: ({ convId, inferenceId }) =>
        `mw-trace/${convId}/${inferenceId}`,
    }),
    getMwTraceSnapshot: builder.query<MwTraceSnapshot, MwSnapshotQuery>({
      query: ({ convId, inferenceId, layer }) =>
        `mw-trace/${convId}/${inferenceId}/${layer}`,
    }),
  }),
});
```

### Key Type Definitions

```typescript
interface TurnSnapshot {
  conv_id: string;
  session_id: string;
  turn_id: string;
  phase: 'pre_inference' | 'post_inference' | 'post_tools' | 'final';
  created_at_ms: number;
  turn: ParsedTurn;
}

interface ParsedTurn {
  id: string;
  blocks: ParsedBlock[];
  metadata: Record<string, any>;
  data: Record<string, any>;
}

interface ParsedBlock {
  index: number;
  id?: string;
  kind: 'system' | 'user' | 'llm_text' | 'tool_call'
      | 'tool_use' | 'reasoning' | 'other';
  role?: string;
  payload: Record<string, any>;
  metadata: Record<string, any>;
}

interface SemEvent {
  type: string;
  id: string;
  seq: number;
  stream_id?: string;
  data: any;
  received_at: string;
}

interface Anomaly {
  id: string;
  severity: 'warning' | 'info' | 'error';
  title: string;
  description: string;
  location: {
    convId: string;
    turnId?: string;
    seq?: number;
    blockIndex?: number;
  };
}
```

---

## Screen-by-Screen Implementation

### Screen 1: Session Overview -- Data Flow

```
Component              API Call                    Store / Query
---------------------  --------------------------- --------------------------
SessionList            GET /debug/conversations    ConvManager.conns (iterate)
TurnList (sidebar)     GET /debug/turns?conv_id=X  TurnStore.List(convID)
StateTrackLane         GET /debug/turn/X/Y/*       TurnStore.List(convID,sessionID)
EventTrackLane         GET /debug/events/X         semFrameBuffer.Snapshot()
ProjectionLane         GET /debug/timeline?conv_id=X TimelineStore.GetSnapshot()
```

**Cross-highlighting logic (frontend):** use explicit correlation fields plus sequence hints:
1. Resolve primary join key `(conv_id, session_id, inference_id, turn_id)`.
2. Use `event.seq` only as secondary temporal ordering/context.
3. Highlight matches only when deterministic join keys align; do not fuzzy-match by index.

### Screen 2: Turn Inspector -- Data Flow

```
Component              API Call                         Store / Query
---------------------  -------------------------------- ----------------------
PhaseTabBar            GET /debug/turn/X/Y/Z            TurnStore (all phases)
BlockList              (from above, selected phase)      Parse YAML -> blocks
ProvenanceBadge        block.metadata["geppetto.middleware@v1"]
IsNewBadge             Compare block list vs prior turn  (frontend diff)
TurnDetailsPanel       turn.metadata + turn.data         (from above)
```

**"From Turn N" vs "NEW" detection:**
The frontend compares the current turn's block list against the previous turn's final snapshot (fetched separately). Blocks present in both are "from Turn N"; blocks only in the current turn are "NEW".

### Screen 3: Snapshot Diff -- Data Flow

```
Component              API Call                    Store / Query
---------------------  --------------------------- ------------------
SideBySideBlocks       GET /debug/turn/X/Y/Z       TurnStore (2 phases)
MetadataDiff           (computed from above)        Frontend diff
DiffSummaryBar         (computed from above)        Frontend diff
```

**Diff algorithm:** identity-aware block diff with reorder detection.

1. Match blocks by stable identity in priority order:
   - `block.id`
   - tool call/result IDs in payload (`id`)
   - fallback fingerprint (`kind`, `role`, normalized payload hash)
2. Classify outcomes:
   - `ADDED`, `REMOVED`, `CHANGED`, `UNCHANGED`, `MOVED`
3. Treat index-only shifts as `MOVED`, not add/remove.
4. Metadata diff remains key-based (shallow key delta plus value-change detection).

### Screen 4: Middleware Chain -- Data Flow

```
Component              API Call                              Store / Query
---------------------  ------------------------------------- ------------------
OnionLayers            GET /debug/mw-trace/X/INF_ID          MwTraceStore
EngineCenter           (from above, engine field)             Computed from trace
InlineDiffLink         GET /debug/mw-trace/X/INF_ID/LAYER    MwTraceStore
InvisibleChanges       (from above, metadata_changes)         Computed from trace
```

**Requires MiddlewareTraceStore** -- see [Middleware Tracing](#middleware-tracing) section.

### Screen 5: Event Inspector -- Data Flow

```
Component              API Call                    Store / Query
---------------------  --------------------------- ------------------
SemanticView           GET /debug/events/X?seq=N   semFrameBuffer
SemEnvelopeView        (same event, raw JSON)       semFrameBuffer
CorrelatedNodesPanel   Cross-query: /debug/turns + /debug/timeline + /debug/events
TrustSignals           Computed from event neighbors
```

**Cross-correlation resolution (frontend):**
1. Event envelope carries explicit `correlation` object (`conv_id`, `session_id`, `inference_id`, `turn_id`).
2. Query `/debug/turns?conv_id=X&session_id=Y` to find matching turn -> link to block.
3. Query `/debug/timeline?conv_id=X` to find timeline entity with matching ID.
4. Query `/debug/events/X` with neighboring seq numbers for context

### Screen 6: Structured Sink View -- Data Flow

This screen is the **most complex to instrument**. It requires:
1. Capturing raw model output before FilteringSink processing
2. Capturing filtered output after tag removal
3. Capturing extracted structured events

**Approach:** Wrap FilteringSink with a debug recording layer that captures all three streams per turn. Store in a new `SinkTraceStore` or extend TurnStore with a `sink_trace` phase.

**Deferred to Phase 5** -- can be implemented later since the core value is in Screens 1-5.

---

## Implementation Phases

### Phase 0: Correlation Contract + API Migration Prerequisites

**Goal:** make joins deterministic and lock route migration decisions before UI implementation.

**Backend tasks:**
1. Add explicit event/snapshot correlation contract fields (`conv_id`, `session_id`, `inference_id`, `turn_id`, optional `snapshot_seq`).
2. Extend snapshot storage/query model to expose stable join metadata.
3. Implement canonical `/debug/turns` and `/debug/timeline` endpoints.
4. Remove legacy `/turns` and `/timeline` routes (no compatibility aliases).
5. Add `DistinctSessions` query path instead of deriving session summaries from limited `List(...)` results.

**Frontend tasks:**
1. Update debug API client assumptions to `/debug` canonical routes only.
2. Implement deterministic join utility based on correlation fields, with `seq` as secondary ordering context.

**Deliverable:** stable join contract + migrated route surface + baseline tests.

### Phase 1: Foundation (Backend + Skeleton UI)

**Goal:** List conversations, browse turn snapshots, display blocks.

**Backend tasks:**
1. Add `EngineConfig` field to `Conversation` struct (store full config, not just signature)
2. Implement `GET /debug/conversations` handler -> iterate ConvManager
3. Implement `GET /debug/conversation/:id` handler -> return full conv metadata
4. Implement `GET /debug/conversation/:id/sessions` handler -> query dedicated session-summary path
5. Add `TurnID` filter to `TurnQuery` struct
6. Implement `GET /debug/turn/:conv_id/:session_id/:turn_id` -> return all phases with parsed JSON
7. Add YAML-to-JSON parsing utility for turn payloads (server-side, using `serde.FromYAML` + JSON marshal)

**Frontend tasks:**
1. Scaffold Vite + React + Redux Toolkit + React Router app at `web-agent-example/cmd/web-agent-debug/`
2. Configure RTK Query API layer with base endpoints
3. Implement `<SessionList>` component with conversation cards
4. Implement `<TurnList>` sidebar with turn cards (block count, phase, timestamp)
5. Implement `<TurnInspector>` with phase tabs and block list
6. Implement `<BlockCard>` with kind icons, role, payload text, raw JSON toggle
7. Implement `<CorrelationIdBar>` with copyable chips
8. Basic routing: `/debug/` -> session list, `/debug/turn/:convId/:sessionId/:turnId` -> inspector

**Deliverable:** Can browse conversations -> turns -> blocks across snapshot phases.

### Phase 2: Timeline Lanes + Events (Session Overview)

**Goal:** Three-lane synchronized timeline view with cross-highlighting.

**Backend tasks:**
1. Add `received_at` timestamp to `semFrameBuffer` entries
2. Implement `GET /debug/events/:conv_id` handler -> parse buffer, filter, paginate
3. Add `SnapshotParsed()` method to semFrameBuffer

**Frontend tasks:**
1. Implement `<SessionOverview>` three-lane layout
2. Implement `<StateTrackLane>` with turn phase cards arranged vertically
3. Implement `<EventTrackLane>` with event dots, type labels
4. Implement `<ProjectionLane>` with timeline entity cards
5. Implement cross-highlighting: click node -> highlight correlated nodes in all lanes
6. Implement `<NowMarker>` with live pulse indicator
7. Implement click-to-navigate: click turn card -> navigate to Turn Inspector
8. WebSocket integration for live event streaming into the event lane

**Deliverable:** Full Session Overview (Screen 1) with live streaming and cross-highlighting.

### Phase 3: Diff View + Event Inspector + Filters

**Goal:** Snapshot comparison, event detail, and filtering.

**Backend tasks:**
1. (None -- diff computation is frontend-only using existing `/debug/turn/` data)

**Frontend tasks:**
1. Implement `<SnapshotDiff>` side-by-side view (Screen 3)
2. Implement block-level diff algorithm (added/removed/changed/same)
3. Implement metadata diff display
4. Implement `<DiffSummaryBar>` with change counts
5. Implement `<EventInspector>` with three view modes (Screen 5)
6. Implement `<SemanticView>` -- human-readable event card with type-specific rendering
7. Implement `<SemEnvelopeView>` -- JSON syntax highlighting
8. Implement `<CorrelatedNodesPanel>` with cross-links
9. Implement `<TrustSignals>` -- sequence monotonicity, ID matching checks
10. Implement `<FilterBar>` with checkbox groups (Screen 7)
11. Implement `<AnomalyPanel>` with auto-detection and pinning (Screen 8)

**Deliverable:** Screens 3, 5, 7, 8 complete.

### Phase 4: Middleware Tracing + Chain View

**Goal:** Per-middleware before/after snapshots and onion visualization.

**Backend tasks:**
1. Implement `MiddlewareTraceStore` interface and SQLite implementation
2. Implement tracing middleware wrapper (see [Middleware Tracing](#middleware-tracing))
3. Wire tracing wrapper into `composeEngineFromSettings()` when debug mode enabled
4. Implement `GET /debug/mw-trace/:conv_id/:inference_id` handler
5. Implement `GET /debug/mw-trace/:conv_id/:inference_id/:layer` handler

**Frontend tasks:**
1. Implement `<MiddlewareChain>` onion-layer visualization (Screen 4)
2. Implement `<MiddlewareLayer>` with PRE/POST summary and inline diff link
3. Implement `<EngineCenter>` card
4. Implement `<InvisibleChangesPanel>`
5. Add middleware chain link from Turn Inspector -> Middleware Chain view

**Deliverable:** Screen 4 complete. Full pipeline visibility.

### Phase 5: Structured Sink View + Polish

**Goal:** FilteringSink debug visualization and UX polish.

**Backend tasks:**
1. Implement debug recording wrapper for FilteringSink
2. Store raw/filtered/extracted streams per turn
3. Implement `GET /debug/sink-trace/:conv_id/:turn_id` handler

**Frontend tasks:**
1. Implement `<StructuredSinkView>` three-column layout (Screen 6)
2. Implement synchronized scrolling between columns
3. Implement extraction highlight regions with connector arrows
4. Implement `<ExtractionLog>` timeline
5. Polish: keyboard shortcuts, responsive layout, dark mode toggle
6. Storybook stories for all components with synthetic data

**Deliverable:** All 8 screens complete.

---

## Middleware Tracing

### The Gap

Currently, middlewares are opaque wrappers (`func(HandlerFunc) HandlerFunc`). There is no built-in mechanism to record what each middleware did to the turn. The `geppetto.middleware@v1` block metadata key exists but is only set by middlewares that opt in.

### Solution: Tracing Wrapper

Add a `TracingMiddleware` wrapper that captures before/after Turn snapshots for each middleware in the chain:

```go
// geppetto/pkg/inference/middleware/tracing.go

type TraceEntry struct {
    Layer      int
    Name       string
    PreTurn    *turns.Turn  // Cloned before calling next
    PostTurn   *turns.Turn  // Result after calling next
    Duration   time.Duration
    Error      error
}

type TraceCollector struct {
    mu      sync.Mutex
    entries []TraceEntry
}

// WrapWithTracing wraps each middleware to capture before/after state.
func WrapWithTracing(
    collector *TraceCollector,
    middlewares []Middleware,
    names []string,
) []Middleware {
    traced := make([]Middleware, len(middlewares))
    for i, mw := range middlewares {
        layer := i
        name := names[i]
        traced[i] = func(next HandlerFunc) HandlerFunc {
            wrappedNext := mw(next)
            return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
                pre := t.Clone()
                start := time.Now()
                result, err := wrappedNext(ctx, t)
                dur := time.Since(start)

                collector.mu.Lock()
                collector.entries = append(collector.entries, TraceEntry{
                    Layer:    layer,
                    Name:     name,
                    PreTurn:  pre,
                    PostTurn: result,
                    Duration: dur,
                    Error:    err,
                })
                collector.mu.Unlock()

                return result, err
            }
        }
    }
    return traced
}
```

### Integration Point

Tracing should be integrated where the middleware list is actually materialized (including built-ins), i.e. the `composeEngineFromSettings(...)` path.

```go
func composeEngineFromSettings(...) (engine.Engine, error) {
    // build middleware slice including built-ins (tool reorder, system prompt)
    mws := materializeMiddlewares(...)

    if os.Getenv("PINOCCHIO_WEBCHAT_DEBUG") == "1" {
        collector := middleware.NewTraceCollector()
        mws = middleware.WrapWithTracing(collector, mws, layerNames)
        // collector can be attached to trace writer/store for this session
    }

    return buildEngineWithMiddlewares(base, mws)
}
```

Important: tracing is effectively one wrapper per middleware layer, not one single outer middleware.

### Storage

The `MiddlewareTraceStore` persists trace entries to SQLite:

```sql
CREATE TABLE middleware_traces (
    conv_id        TEXT NOT NULL,
    inference_id   TEXT NOT NULL,
    layer_index    INTEGER NOT NULL,
    mw_name        TEXT NOT NULL,
    pre_yaml       TEXT NOT NULL,
    post_yaml      TEXT NOT NULL,
    duration_ms    INTEGER NOT NULL,
    error_text     TEXT,
    created_at_ms  INTEGER NOT NULL,
    PRIMARY KEY (conv_id, inference_id, layer_index)
);

CREATE INDEX idx_mw_traces_conv
    ON middleware_traces(conv_id);
CREATE INDEX idx_mw_traces_inference
    ON middleware_traces(conv_id, inference_id);
```

### Performance Impact

- Turn cloning is O(n) where n = number of blocks. For typical turns (3-20 blocks), this is <1ms.
- YAML serialization for storage is ~1-5ms per snapshot.
- Only enabled when `PINOCCHIO_WEBCHAT_DEBUG=1`, so zero cost in production.

---

## Anomaly Detection

### Auto-detected Anomalies (Frontend)

| Anomaly | Detection Logic | Severity |
|---------|----------------|----------|
| Out-of-sequence event | `event[i].seq > event[i+1].seq` | warning |
| Tool ID mismatch | tool_call block with no matching tool_use by ID | warning |
| Unrecognized event type | SEM event type not in known set | info |
| Delta truncation | >20 consecutive llm.delta events | info |
| Missing final event | llm.start without corresponding llm.final | warning |
| Duplicate entity ID | Two entities with same ID but different kinds | error |
| Stale streaming | Entity with `streaming: true` older than 30s | warning |

### User-pinned Anomalies

Any node in any view can be right-clicked -> "Pin as anomaly". The pinned anomaly stores:
- Node reference (conv_id, seq or turn_id, block index)
- User-provided note (optional)
- Timestamp

### Export

The anomaly panel supports "Export JSON" which serializes all pinned + auto-detected anomalies for bug reports.

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Middleware tracing overhead** | Turn cloning adds latency per middleware | Only enable when debug flag is set; profile with realistic chains |
| **SEM buffer loss** | In-memory buffer loses events on restart | Accept for MVP; consider persisting events to SQLite in Phase 5 |
| **YAML parsing performance** | Large turns (50+ blocks) may be slow to parse on every request | Cache parsed JSON alongside YAML in TurnStore; add pagination |
| **Cross-highlighting complexity** | Correlation contract gaps can produce ambiguous joins | Enforce deterministic correlation fields and reject heuristic-only joins |
| **FilteringSink recording** | Capturing all three streams adds complexity | Defer to Phase 5; the core debug value is in Screens 1-4 |
| **Frontend bundle size** | Debug UI adds to the overall bundle | Separate Vite entry point; only loaded when navigating to /debug/ |
| **Route migration breakage** | Removing `/turns` and `/timeline` can break old clients/scripts | Coordinate one release, update all clients, and enforce with integration tests |

---

## Acceptance Test Matrix

### Correlation and joining

1. Event envelopes include required correlation fields for debug-relevant event types.
2. Turn snapshots can be joined to event stream via deterministic keys `(conv_id, session_id, inference_id, turn_id)`.
3. `seq` remains monotonic and is used as secondary temporal ordering, not primary identity.

### Route migration (no backwards compatibility)

1. `/debug/turns` and `/debug/timeline` endpoints pass integration tests.
2. Legacy `/turns` and `/timeline` are removed and return not found.
3. Debug app test suite uses only `/debug/*` endpoints.

### Middleware tracing

1. Trace records include built-in and configured middlewares in execution order.
2. Each trace row captures pre/post turn snapshots, duration, and error state.
3. Query by `(conv_id, inference_id)` returns complete chain.

### Diff correctness

1. Reordered blocks are classified as `MOVED`.
2. Stable-ID blocks match across phase snapshots even when indices change.
3. Metadata diffs and payload diffs are reported separately.

---

## File Inventory: What Gets Created

### Backend (Go)

| File | Purpose | Phase |
|------|---------|-------|
| `pinocchio/pkg/webchat/debug_handlers.go` | All `/debug/*` HTTP handlers | 1-4 |
| `pinocchio/pkg/webchat/mw_trace_store.go` | MiddlewareTraceStore interface | 4 |
| `pinocchio/pkg/webchat/mw_trace_store_sqlite.go` | SQLite implementation | 4 |
| `geppetto/pkg/inference/middleware/tracing.go` | TracingMiddleware wrapper | 4 |
| `pinocchio/pkg/webchat/turn_json.go` | YAML to JSON conversion utility | 1 |

### Frontend (TypeScript/React)

| File | Purpose | Phase |
|------|---------|-------|
| `web-agent-example/cmd/web-agent-debug/` | Debug SPA root (separate app) | 1 |
| `web-agent-example/cmd/web-agent-debug/src/api/debugApi.ts` | RTK Query API definitions | 1 |
| `web-agent-example/cmd/web-agent-debug/src/store/` | Redux slices | 1-3 |
| `web-agent-example/cmd/web-agent-debug/src/components/SessionList.tsx` | Conversation list | 1 |
| `web-agent-example/cmd/web-agent-debug/src/components/TurnList.tsx` | Turn sidebar | 1 |
| `web-agent-example/cmd/web-agent-debug/src/components/TurnInspector.tsx` | Block-level turn view | 1 |
| `web-agent-example/cmd/web-agent-debug/src/components/BlockCard.tsx` | Single block renderer | 1 |
| `web-agent-example/cmd/web-agent-debug/src/components/CorrelationIdBar.tsx` | Copyable ID chips | 1 |
| `web-agent-example/cmd/web-agent-debug/src/components/SessionOverview.tsx` | Three-lane timeline | 2 |
| `web-agent-example/cmd/web-agent-debug/src/components/lanes/` | StateTrack, EventTrack, Projection | 2 |
| `web-agent-example/cmd/web-agent-debug/src/components/SnapshotDiff.tsx` | Side-by-side diff | 3 |
| `web-agent-example/cmd/web-agent-debug/src/components/EventInspector.tsx` | Event detail views | 3 |
| `web-agent-example/cmd/web-agent-debug/src/components/FilterBar.tsx` | Checkbox filter groups | 3 |
| `web-agent-example/cmd/web-agent-debug/src/components/AnomalyPanel.tsx` | Anomaly list + pinning | 3 |
| `web-agent-example/cmd/web-agent-debug/src/components/MiddlewareChain.tsx` | Onion-layer visualization | 4 |
| `web-agent-example/cmd/web-agent-debug/src/components/StructuredSinkView.tsx` | Three-column sink debug | 5 |

---

## See Also

- [UI Wireframes](../sources/local/ui-design-turn.md) -- ASCII wireframes this plan implements
- [Requirements Spec](01-turn-and-middleware-debug-ui-requirements-and-ux-specification.md) -- Full FR/IR/NFR requirements
- [Designer Primer](02-designer-primer-turns-blocks-middlewares-and-structured-events.md) -- Conceptual grounding for turns, blocks, middlewares
- [Adding a New Event Type](../../../../pinocchio/pkg/doc/topics/webchat-adding-event-types.md) -- How the existing SEM pipeline works
- [Backend Internals](../../../../pinocchio/pkg/doc/topics/webchat-backend-internals.md) -- Timeline projector, StreamCoordinator internals
