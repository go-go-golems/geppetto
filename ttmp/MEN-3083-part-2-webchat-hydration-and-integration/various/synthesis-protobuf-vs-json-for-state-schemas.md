---
Title: Synthesis — Protobuf vs JSON for Widget State Schemas (Hydration Snapshots)
Ticket: MEN-3083-part-2
Status: active
Topics:
    - backend
    - frontend
    - events
    - hydration
    - snapshots
DocType: analysis
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: A practical, code-oriented synthesis of whether to define persisted widget state (Pinocchio snapshots) using Protobuf (JSON-encoded) vs current JSON structs, with concrete file references, symbols, tradeoffs, and migration guidance.
LastUpdated: 2025-11-04T13:19:03.199848664-05:00
---



# Synthesis — Protobuf vs JSON for Widget State Schemas (Hydration Snapshots)

## 1) Purpose and Scope

This document addresses a technical decision: should we use Protocol Buffers (Protobuf) to define the schema for persisted widget state snapshots in Pinocchio, or should we continue with our current approach of hand-written Go structs and JSON serialization?

### Background Context

Our team already uses Protobuf successfully in go-go-mento to define typed SEM (Semantic Event Model) event payloads. We have Protobuf schemas that generate both Go and TypeScript code, giving us type safety across the stack. However, we still serialize these events as JSON over WebSocket—we're not using binary Protobuf on the wire. This gives us the best of both worlds: compile-time type checking via generated code, but human-readable JSON for debugging.

Meanwhile, Pinocchio (our webchat server framework) persists server-side widget state as "snapshots" in SQLite. These snapshots power our hydration feature, which lets the UI instantly restore conversation state after page reloads or reconnects. Currently, these snapshots are defined as Go structs with JSON tags (like `LLMTextSnapshot`, `ToolCallSnapshot`, etc.) and stored as JSON text in the database.

### The Question

Should we extend our Protobuf usage to also define snapshot schemas? This would mean:
- Writing `.proto` files for each snapshot type (similar to what we do for SEM events)
- Generating Go and TypeScript code from those schemas
- Still storing JSON in the database (using `protojson` encoding), so we keep debuggability
- Getting stronger type safety and schema evolution guarantees

This document walks through the current implementation, presents three concrete options (continue with JSON + add validation, adopt Protobuf, or use JSON Schema), discusses tradeoffs, and provides a safe migration path if we choose Protobuf. It's written for a developer who's new to the codebase and needs to understand both what we have today and how we might evolve it.

### Target scope: go-go-mento as the implementation site (PostgreSQL)

Pinocchio’s snapshot system is a reference-quality prototype. The production implementation will live in go-go-mento:
- A dedicated snapshot subsystem inside `go-go-mento` (not reusing Pinocchio’s SQLite code)
- PostgreSQL as the durable store (JSONB columns for debuggability)
- Protobuf-defined snapshot schemas (JSON-encoded via `protojson`)
- Hydration API served by go-go-mento (consumed by the existing `hydrateTimelineThunk`)
- Support for many custom, typed events and widgets (team analysis, inner thoughts, debate, mode changes, tools, etc.)

## 2) Current Implementation — Server-Side Snapshots (Pinocchio)

To understand the decision ahead, let's walk through how snapshots work today in Pinocchio.

### What Are Snapshots?

Snapshots are server-side representations of widget state. When a backend tool runs (like the team analyzer or calculator), events are emitted that both (1) get sent to the frontend via WebSocket as SEM frames, and (2) get projected into typed snapshot records and persisted in SQLite. This way, when a user refreshes the page or reconnects, the frontend can fetch all the snapshots for a conversation and instantly reconstruct the timeline—no need to replay the entire event stream.

### Snapshot Types

We currently have six snapshot kinds, defined as Go structs in `pinocchio/pkg/snapshots/types.go`:

1. **`LLMTextSnapshot`** - Represents assistant or user text messages (role, text, streaming flag)
2. **`ToolCallSnapshot`** - Represents tool invocations (name, input, execution status)
3. **`ToolResultSnapshot`** - Represents tool outputs (result payload)
4. **`TeamAnalysisSnapshot`** - Domain-specific for team analysis features (team size, progress, summary)
5. **`AgentModeSnapshot`** - Agent mode switch announcements
6. **`LogEventSnapshot`** - Log entries promoted to timeline entities

All snapshot types share a common base structure called `SnapshotBase`, which includes fields like `ConversationID`, `EntityID`, `Kind` (discriminator), `Status`, timestamps, `Version` (for ordering and version-gating), and `Flags` (arbitrary metadata).

### How Snapshots Are Created and Stored

The flow works like this:

1. **Event Projection**: When a backend event arrives (like `EventToolCall` or `EventTeamAnalysisResult`), the `TimelineProjector` (specifically `BasicProjector.Apply`) transforms it into one or more typed snapshots. This happens in `pinocchio/pkg/snapshots/projector.go`.

2. **Context-Aware Persistence**: The projector initially keys snapshots by `RunID` (from event metadata), but we need them keyed by `conv_id` for hydration. So before persisting, `ProjectAndPersist` checks if a conversation ID is attached to the context (via `WithConversationID`) and overrides the base's `ConversationID` field.

3. **Side-Channel Storage**: In `pinocchio/pkg/webchat/conversation.go`, the `convertAndBroadcast` function attaches the `conv_id` to the context and calls `SemanticEventsFromEventWithProjection`, which both emits SEM frames to WebSocket clients AND persists snapshots via `ProjectAndPersist`. This persistence is non-blocking and errors are logged but don't break streaming.

4. **SQLite Storage**: The `SQLiteSnapshotStore` (in `pinocchio/pkg/snapshots/sqlite_store.go`) stores snapshots in a table with schema:
   ```sql
   CREATE TABLE snapshots (
       conversation_id TEXT NOT NULL,
       entity_id       TEXT NOT NULL,
       kind            TEXT NOT NULL,
       version         INTEGER NOT NULL,
       status          TEXT,
       started_at      INTEGER,
       updated_at      INTEGER,
       snapshot_json   TEXT NOT NULL,
       flags_json      TEXT,
       PRIMARY KEY (conversation_id, entity_id)
   )
   ```
   The `snapshot_json` column holds the full JSON-serialized snapshot. There's also an index on `(conversation_id, updated_at)` for efficient hydration queries.

### JSON Marshaling and Discriminator Pattern

Snapshots are marshaled to JSON using `MarshalSnapshot` (just `json.Marshal` today) and unmarshaled using a discriminator pattern in `UnmarshalSnapshot` (in `pinocchio/pkg/snapshots/codec.go`):

```go
func UnmarshalSnapshot(b []byte) (Snapshot, error) {
    var probe struct { Kind SnapshotKind `json:"kind"` }
    json.Unmarshal(b, &probe)
    switch probe.Kind {
        case KindLLMText: /* unmarshal into LLMTextSnapshot */
        case KindToolCall: /* unmarshal into ToolCallSnapshot */
        // ... etc
    }
}
```

This lets us store polymorphic snapshots in a single table column while maintaining type safety when reading them back out.

### Hydration API

When the frontend needs to restore state, it calls `GET /api/conversations/{convId}/timeline?sinceVersion=...` (handled in `pinocchio/pkg/webchat/router.go`). The router queries `GetByConversation`, unmarshals the snapshots, and returns them as JSON. The optional `sinceVersion` parameter supports incremental hydration (only fetch snapshots newer than what the client already has).

The router also persists user input snapshots when `/chat` is called, so hydration includes both the user's prompt and the assistant's responses.

### Frontend Consumption

On the frontend (example: go-go-mento's `web/src/store/timeline/timelineSlice.ts`), a thunk called `hydrateTimelineThunk(convId)` fetches the snapshots, then `mapSnapshotToEntity` converts each snapshot into a local `TimelineEntity`. These entities are upserted into the Redux store with version-gating logic: `upsertEntity` only accepts updates if `incoming.version > existing.version`, preventing stale data from overwriting newer streaming updates.

## 3) Current Implementation — Typed SEM Events with Protobuf (go-go-mento)

To see why Protobuf might be a good fit for snapshots, let's look at how we're already using it successfully for SEM events in go-go-mento.

### The Protobuf Pattern We Already Use

In go-go-mento, we define SEM event payloads using Protobuf schemas. For example, `go-go-mento/proto/sem/domain/team_analysis.proto` defines messages like `TeamAnalysisStart`, `TeamAnalysisProgress`, and `TeamAnalysisResult`. These include fields like `analysis_id`, `team_size`, `progress`, and `visualization_data`.

When we run `make protobuf`, the Buf toolchain generates Go code (in `go-go-mento/go/pkg/sem/pb/proto/sem/.../*.pb.go`) and TypeScript code (in `go-go-mento/web/src/sem/pb/proto/sem/.../*_pb.ts`). Both languages get strongly-typed structs/interfaces that match the proto schema exactly.

### JSON Over the Wire, Protobuf for Types

Here's the clever part: we don't send binary Protobuf over WebSocket. Instead, we use `protojson.Marshal` to convert proto messages to JSON. This happens in helpers like `pbToMap` (in `go-go-mento/go/pkg/webchat/handlers/helpers.go`), which takes a `proto.Message` and returns a `map[string]any` suitable for embedding in a SEM frame's `data` field.

So a typical flow looks like:
1. Backend creates a typed proto message: `&TeamAnalysisResult{AnalysisId: "abc", NetworkScore: 0.85, ...}`
2. `pbToMap` serializes it to JSON: `{"analysisId": "abc", "networkScore": 0.85, ...}`
3. That JSON goes into the SEM frame's `data` field and gets sent over WebSocket
4. Frontend receives the JSON and can either parse it loosely OR use `fromJson(TeamAnalysisResultSchema, data)` to get a typed object

This gives us debugging wins (we can inspect JSON in browser dev tools or with `jq`) AND type safety (the compiler checks that we're accessing the right fields).

### Why This Works Well

The forwarder in `go-go-mento/go/pkg/webchat/forwarder.go` uses a type-based registry to map backend events to SEM frames. For example, when it sees an `EventTeamAnalysisResult`, it calls a typed handler that constructs the proto message, converts it to JSON via `pbToMap`, and wraps it in a SEM frame. The frontend's `useChatStream.ts` receives these frames and dispatches them to Redux, where `mapSnapshotToEntity` (if hydrating) or direct entity updates (if streaming) keep the UI in sync.

The key insight: Protobuf is our **schema definition language**, not our wire format. We get schema evolution (optional fields, deprecation tags), cross-language consistency, and validation, all while keeping JSON's readability.

## 4) Problem Statement: Why We're Considering a Change

The current JSON-based snapshot system works fine for our MVP, but as we scale up widget types and frontend consumers, we're starting to hit limitations:

### Schema Evolution Challenges

Imagine we need to add a `requiredPermissions` field to `ToolCallSnapshot`. With our current approach:
- We add the field to the Go struct with an `omitempty` JSON tag
- We update the TypeScript interface (hopefully remembering to do it!)
- Old snapshots in the database don't have this field
- When we unmarshal them, Go gives us a zero value (`nil` for the slice)
- The frontend might crash if it doesn't defensively check for absence

There's no explicit contract about what "optional" means semantically. Is `requiredPermissions == nil` the same as "no permissions required"? Or does it mean "permissions not yet determined"? The code doesn't say—you have to read comments or tribal knowledge.

Protobuf, by contrast, has explicit optional/required semantics and field presence detection. You can distinguish between "field not set" and "field set to empty value."

### Type Drift Risk

Right now, our Go structs in `pinocchio/pkg/snapshots/types.go` and our TypeScript interfaces in `go-go-mento/web/src/pages/Chat/timeline/types.ts` are manually kept in sync. If a backend engineer adds a field and forgets to update the frontend types, we won't find out until runtime—likely in production when a user hits a code path that accesses the new field.

With Protobuf codegen, both Go and TypeScript are generated from the same `.proto` source. If you change the schema, both sides update atomically when you run `make protobuf`. The compiler enforces consistency.

### Validation Gap

Currently, we have zero validation at write-time. If a bug causes us to persist a `ToolCallSnapshot` with an empty `Name` field, we'll only discover it later when the frontend tries to render it. With Protobuf, we can add field-level validators (required fields, enum constraints) and get compile-time or at least early runtime checking.

### Why JSON Storage Still Matters

Despite these issues, we absolutely want to keep JSON in the database. Debugging production issues often involves running SQL queries like:

```bash
sqlite3 snapshots.db "SELECT snapshot_json FROM snapshots WHERE entity_id='tool-abc123'"
```

Being able to read the raw JSON immediately is invaluable. Binary Protobuf would make this much harder. Fortunately, Protobuf supports JSON encoding via `protojson`, so we can have our cake and eat it too.

## 5) Options: Three Approaches We Could Take

We've identified three viable paths forward, each with different tradeoffs. Let's walk through them in detail.

### Option A: Stay with JSON, Add Validation and Process

This option keeps our current JSON-based approach but addresses the validation and evolution gaps through discipline and tooling.

**What we'd do:**

1. **Add Validation Methods**: Extend the `Snapshot` interface to include a `Validate() error` method. Each snapshot type (`LLMTextSnapshot`, `ToolCallSnapshot`, etc.) would implement basic checks like "Name field must not be empty" or "Status must be one of pending/running/completed/error".

2. **Enforce Validation at Write Time**: Modify `SQLiteSnapshotStore.Upsert` to call `snap.Validate()` before marshaling and inserting into the database. If validation fails, we return an error and don't persist bad data.

3. **Document Evolution Rules**: Create a clear policy for how we handle schema changes. For example: "All new fields must use `omitempty` and be treated as optional. The frontend must defensively check for absence. When removing a field, deprecate it for at least 2 releases before deleting."

4. **Add Cross-Language Tests**: Write integration tests that generate Go snapshots, marshal them to JSON, and check that TypeScript can parse them correctly. This catches drift early.

**Pros:**
- Fastest to implement—no new build dependencies, no codegen
- Keeps our simple, readable storage format
- Team is already familiar with this pattern

**Cons:**
- No automatic sync between Go and TypeScript types—we still have to remember to update both sides
- Validation logic is hand-written and could get complex
- No compiler help for evolution (we rely on code review and testing)

**Files we'd touch:**
- `pinocchio/pkg/snapshots/types.go` - add `Validate()` implementations
- `pinocchio/pkg/snapshots/sqlite_store.go` - call validation before write
- Frontend test suites - add schema compliance tests

### Option B: Protobuf Schemas with JSON Encoding (Recommended for Growth)

This option adopts Protobuf as our schema definition language while keeping JSON as the storage format. It mirrors our successful SEM event pattern.

**What we'd do:**

1. **Define Proto Schemas**: Create `.proto` files for each snapshot type. For example, `pinocchio/proto/snapshots/llm_text.proto` might look like:
   ```proto
   message LLMTextSnapshot {
     SnapshotBase base = 1;
     string role = 2;
     string text = 3;
     bool streaming = 4;
   }
   ```
   We'd define `SnapshotBase` once and embed it in each concrete type.

2. **Wire Up Codegen**: Integrate Buf into Pinocchio's build (similar to go-go-mento). Running `make protobuf` would generate Go code in `pinocchio/pkg/snapshots/pb/` and optionally TypeScript code if needed by the Pinocchio webchat UI.

3. **Dual-Marshal Pattern**: Update `MarshalSnapshot` to use `protojson.Marshal` instead of `json.Marshal`. The output is still JSON text—just generated via Protobuf instead of Go's reflection-based marshaler. Update `UnmarshalSnapshot` to detect whether the JSON came from a proto-generated snapshot (perhaps via a `Format` field) and unmarshal accordingly.

4. **Optional Format Versioning**: Add a `Format` field to `SnapshotBase` (e.g., `Format int` where `1 = legacy JSON`, `2 = protobuf JSON`) and a `SchemaVersion` to track evolution. New snapshots would set `Format = 2`; old ones remain `Format = 1` (or unset, defaults to 1).

5. **Validation Integration**: Protobuf's generated code includes field presence checks. We can add a wrapper `Validate()` method that checks business-level constraints on top of the proto-level guarantees.

**Pros:**
- **Single Source of Truth**: The `.proto` file is the canonical schema. Go and TypeScript are generated from it, so they can't drift.
- **Schema Evolution Built-In**: Protobuf has well-defined rules for forward/backward compatibility. Field numbers are stable, optional fields have clear semantics, and you can mark fields as deprecated.
- **Keeps JSON Debuggability**: We still store readable JSON in SQLite. You can still `SELECT snapshot_json` and pipe it to `jq`.
- **Leverage Existing Tooling**: The team already knows Buf and Protobuf from go-go-mento. No learning curve.

**Cons:**
- **Build Dependency**: We introduce Buf/protoc into Pinocchio's build process. CI needs to run codegen.
- **Migration Complexity**: We need to support dual-read (old JSON-only snapshots + new protobuf-JSON snapshots) during rollout. This adds code paths.
- **Generated Code Overhead**: Proto-generated structs can be verbose. We may need wrapper types for convenience.

**Files we'd touch or create:**
- **New**: `pinocchio/proto/snapshots/*.proto` - schema definitions
- **New**: Codegen outputs in `pinocchio/pkg/snapshots/pb/` (Go) and possibly frontend (TS)
- **Modified**: `pinocchio/pkg/snapshots/codec.go` - switch between JSON and protojson based on `Format`
- **Modified**: `pinocchio/pkg/snapshots/types.go` - possibly wrap generated types or migrate to them entirely
- **No change**: `pinocchio/pkg/snapshots/sqlite_store.go` - still stores TEXT, so no schema migration needed

### Option C: JSON Schema as the Canonical Source

This middle-ground option uses JSON Schema (a widely-adopted standard for describing JSON structure) as the schema definition language, with codegen for Go and TypeScript.

**What we'd do:**

1. **Author JSON Schemas**: Write JSON Schema documents for each snapshot type. JSON Schema uses JSON itself to describe structure, constraints, and documentation. For example:
   ```json
   {
     "$schema": "http://json-schema.org/draft-07/schema#",
     "type": "object",
     "properties": {
       "role": {"type": "string", "enum": ["user", "assistant"]},
       "text": {"type": "string"},
       "streaming": {"type": "boolean"}
     },
     "required": ["role", "text"]
   }
   ```

2. **Generate Code**: Use tools like `go-jsonschema` (for Go) and `json-schema-to-typescript` (for TypeScript) to generate structs and interfaces from these schemas.

3. **Runtime Validation**: Integrate JSON Schema validators (like `ajv` for TypeScript and Go's `jsonschema` package) to check snapshots at write-time (backend) and potentially read-time (frontend).

4. **Store as Before**: Continue storing JSON text in SQLite. The schema just provides compile-time types and runtime validation.

**Pros:**
- **Stays in JSON Ecosystem**: No need to learn Protobuf. JSON Schema is more familiar to web developers.
- **Widely Supported**: Lots of tooling, editors understand it (autocomplete, validation), and it's a standard.
- **Validation Built-In**: JSON Schema's whole purpose is validation. You get constraint checking for free.

**Cons:**
- **Go Codegen Quality**: Tools like `go-jsonschema` exist but aren't as polished as Protobuf's Go generator. Generated code can be clunky.
- **Evolution Semantics**: JSON Schema doesn't have as opinionated a stance on schema evolution as Protobuf. You have to define your own policies.
- **Different Toolchain**: Introduces yet another build dependency (not Buf/protoc, but JSON Schema tooling). Since we already use Protobuf elsewhere, this adds complexity rather than reducing it.

**Files we'd touch:**
- **New**: JSON Schema files (perhaps `pinocchio/schemas/snapshots/*.json`)
- **New**: Codegen configuration and Makefile targets
- **Modified**: Similar files to Option A, but with generated types instead of hand-written ones

## 6) Migration Strategy (if we choose Option B: Protobuf)

If we decide to go with Protobuf, we need a safe, incremental migration plan that doesn't break production or require risky database backfills.

### Core Principles

1. **No Database Schema Changes**: We keep storing TEXT in the `snapshot_json` column. Protobuf JSON looks just like regular JSON from SQLite's perspective.

2. **Additive Rollout**: We add new code paths without removing old ones. New snapshots use Protobuf; old snapshots continue to work.

3. **No Backfill**: We don't try to migrate existing snapshots. They age out naturally via TTL policies (conversations older than N days are cleaned up).

### Step-by-Step Plan

**Phase 1: Dual-Read Support**

First, we make the system able to read both old-style JSON and new Protobuf JSON. We add a `Format` field to `SnapshotBase`:

```go
type SnapshotBase struct {
    // ... existing fields ...
    Format int `json:"format,omitempty"` // 0 or missing = legacy, 1 = protobuf-json
}
```

Then we update `UnmarshalSnapshot` in `codec.go` to check this field and route to the appropriate unmarshaler:

```go
func UnmarshalSnapshot(b []byte) (Snapshot, error) {
    var probe struct {
        Kind   SnapshotKind `json:"kind"`
        Format int          `json:"format"`
    }
    json.Unmarshal(b, &probe)
    
    if probe.Format == 1 {
        // Use protojson.Unmarshal with generated types
        return unmarshalProtoSnapshot(probe.Kind, b)
    } else {
        // Legacy path (current discriminator logic)
        return unmarshalLegacySnapshot(probe.Kind, b)
    }
}
```

Deploy this. Now the system can read both formats.

**Phase 2: Switch Writers**

Next, we update `MarshalSnapshot` to use `protojson.Marshal` and set `Format = 1`:

```go
func MarshalSnapshot(s Snapshot) ([]byte, error) {
    s.Base().Format = 1
    return protojson.Marshal(s) // Returns JSON text
}
```

Deploy this. New snapshots are Protobuf-generated, but they're still readable as JSON in the database.

**Phase 3: Add Validation**

Now that we're using proto-generated types, add a `Validate()` wrapper that checks business constraints. Call it in `SQLiteSnapshotStore.Upsert` before writing.

**Phase 4: Monitor and Clean Up (Optional)**

Over time, old snapshots age out. Once you're confident, you could optionally remove the legacy unmarshal path—but there's no urgency since it doesn't hurt to keep it.

### What Files Change

- `pinocchio/pkg/snapshots/types.go` - add `Format` field to `SnapshotBase`
- `pinocchio/pkg/snapshots/codec.go` - implement dual-read logic
- `pinocchio/pkg/snapshots/sqlite_store.go` - add validation calls
- New proto files and generated code (see Option B description)

### go-go-mento adaptation (greenfield)

For go-go-mento, snapshots are new. We can start at Phase 2 (emit Protobuf JSON from day one) and skip dual-read entirely. No `Format` flag is needed unless we later import legacy rows.

## 7) Decision Guidance: How to Choose

Here's a decision matrix to help guide the choice based on your current situation and priorities.

### Choose Option A (JSON + Validation) if:

- **You're early-stage**: You have ≤ 10 snapshot kinds and they're not changing frequently
- **Speed matters**: You want the absolute fastest path to getting basic validation in place
- **Team size is small**: A small, tightly-coordinated team can keep Go and TS types in sync manually without much risk
- **You want minimum dependencies**: No build tooling changes, no codegen to debug

**When this makes sense**: During initial prototyping or for simple, stable features. If snapshots are more of an internal implementation detail than a public API, manual JSON might be fine.

### Choose Option B (Protobuf + JSON Encoding) if:

- **You expect growth**: You're planning to add many new snapshot types (new widget kinds, richer state) and fields will evolve frequently
- **Cross-team or cross-repo usage**: Multiple frontend applications (go-go-mento, Pinocchio UI, external integrations) consume snapshots, so drift is risky
- **You value compile-time safety**: You want the compiler to catch mismatches between Go and TypeScript immediately
- **Consistency with existing stack**: The team already uses Protobuf for SEM events. Extending the pattern keeps the architecture consistent.

**When this makes sense**: For production systems where snapshot schemas are a contract between backend and frontend (or multiple frontends). If you're building a platform where snapshots might eventually be exposed via API, Protobuf's evolution guarantees are valuable.

### Choose Option C (JSON Schema) if:

- **Your team is web-focused**: Frontend engineers are more comfortable with JSON Schema than Protobuf
- **You want validation without Protobuf**: You want codegen and validation but don't want to commit to the Protobuf ecosystem
- **You're OK with manual evolution policies**: You're willing to define and enforce your own schema evolution rules

**When this makes sense**: If you're heavily invested in JSON tooling elsewhere in the stack or if your team finds Protobuf's syntax and semantics too opaque. However, note that introducing a third schema system (after Go structs and Protobuf for SEM) adds cognitive load.

### Recommendation

Given that:
1. The team is already using Protobuf successfully for SEM events in go-go-mento
2. Snapshots are a cross-language contract (Go backend, TypeScript frontend)
3. The feature is still evolving (new snapshot kinds are likely)

**Option B (Protobuf + JSON encoding)** is the strongest choice for long-term maintainability. The migration is incremental and low-risk, and it aligns with existing team skills and tooling.

For simpler, more stable use cases, **Option A (JSON + validation)** may be sufficient and faster to implement.

Whatever you choose, the important thing is to make schema evolution an explicit, first-class concern rather than an afterthought.

---

## Appendix — Key Code Snippets

### Pinocchio JSON discriminator (current implementation)

```go
// pinocchio/pkg/snapshots/codec.go
func UnmarshalSnapshot(b []byte) (Snapshot, error) {
    var probe struct { Kind SnapshotKind `json:"kind"` }
    if err := json.Unmarshal(b, &probe); err != nil { return nil, err }
    switch probe.Kind {
        case KindLLMText:
            var s LLMTextSnapshot
            err := json.Unmarshal(b, &s)
            return &s, err
        // ... other cases
    }
}
```

This pattern works well for a small number of types but requires manual maintenance for each new snapshot kind.

### go-go-mento Protobuf JSON mapping (pattern we'd reuse for snapshots)

```go
// go-go-mento/go/pkg/webchat/handlers/helpers.go
func pbToMap(msg proto.Message) (map[string]any, error) {
    b, err := protojson.Marshal(msg)
    if err != nil {
        return nil, err
    }
    var m map[string]any
    _ = json.Unmarshal(b, &m)
    return m, err
}
```

This helper shows how we convert a typed proto message to JSON for embedding in SEM frames. The same pattern would work for snapshots: proto messages in Go code, JSON text in the database and over HTTP.

## 8) Cross-Repo File Index (quick reference)

Pinocchio (backend snapshots):
- `pinocchio/pkg/snapshots/types.go` — Snapshot kinds and structs
- `pinocchio/pkg/snapshots/codec.go` — JSON marshal/unmarshal
- `pinocchio/pkg/snapshots/sqlite_store.go` — SQLite schema + CRUD
- `pinocchio/pkg/snapshots/projector.go` — event→snapshot projection; `ProjectAndPersist`
- `pinocchio/pkg/webchat/conversation.go` — `WithConversationID`; side-channel persistence
- `pinocchio/pkg/webchat/router.go` — Hydration API, user snapshot on `/chat`

go-go-mento (typed SEM events + hydration consumer today; planned snapshot persistence):
- Existing:
  - `go-go-mento/proto/sem/domain/team_analysis.proto` — example typed schema
  - `go-go-mento/go/pkg/webchat/handlers/helpers.go` — `pbToMap` (proto→JSON map)
  - `go-go-mento/go/pkg/webchat/forwarder.go` — registry + SEM mapping
  - `go-go-mento/web/src/hooks/useChatStream.ts` — WS stream; hydrates on open
  - `go-go-mento/web/src/store/timeline/timelineSlice.ts` — hydration mapping + version-gated upserts
- Planned (new):
  - `go-go-mento/proto/snapshots/*.proto` — snapshot schemas (LLM, tool call/result, analysis, status, etc.)
  - `go-go-mento/go/pkg/snapshots/*` — types, codec, projector, context helpers
  - `go-go-mento/go/pkg/snapshots/postgres_store.go` — `SnapshotStore` for PostgreSQL (pgx)
  - `go-go-mento/go/pkg/webchat/router.go` — Hydration handler: `GET /api/conversations/{convId}/timeline`
  - `go-go-mento/go/pkg/webchat/conversation.go` — call `ProjectAndPersist` during streaming

## 10) go-go-mento Snapshot System (PostgreSQL): Design & Plan

### Goals and requirements

- Persist server-side state for many custom widgets and event families (e.g., `team.analysis.*`, `inner.thoughts.*`, `mode.evaluation.*`, `next.thinking.mode.*`, `debate.*`, `llm.*`, `tool.*`).
- Hydrate timelines on reload reliably (initial burst via HTTP fetch) and merge with live streams using version gating.
- Use Protobuf as the canonical schema language; keep JSON (protojson) as the storage and API format for debuggability.
- Store snapshots in PostgreSQL for production-readiness, indexing, and observability.
- Keep the SEM stream unchanged for real-time rendering; snapshots are the durable, typed mirror.

### Backend architecture (new in go-go-mento)

1) Protobuf snapshot schemas (JSON-encoded)
- Define proto messages for each durable snapshot kind under `go-go-mento/proto/snapshots/*`.
- Include a shared `SnapshotBase` (conv_id, entity_id, kind, status, version, timestamps, flags) and embed it in each message.
- Generate Go types (and TS types if needed by web) via Buf.

2) PostgreSQL schema (JSONB + version gating)
- Table (proposed):
  - `snapshots(conv_id text, entity_id text, kind text, version bigint, status text, started_at timestamptz, updated_at timestamptz default now(), snapshot_json jsonb not null, flags_json jsonb, primary key (conv_id, entity_id))`
- Indexes:
  - `create index idx_snapshots_conv_updated on snapshots (conv_id, updated_at desc);`
  - `create index idx_snapshots_conv_version on snapshots (conv_id, version desc);`
- Upsert with version gating at the database layer (simplified):
  - `on conflict (conv_id, entity_id) do update ... where excluded.version > snapshots.version;`

3) Go package structure (new)
- `go-go-mento/go/pkg/snapshots/types.go` — `Snapshot` interface + `SnapshotBase` (wrappers around generated types where needed)
- `go-go-mento/go/pkg/snapshots/codec.go` — marshal/unmarshal via `protojson`; include `kind` in JSON for frontend mapping
- `go-go-mento/go/pkg/snapshots/store.go` — `SnapshotStore` interface: `Upsert(ctx, s)`, `GetByConversation(ctx, convID, sinceVersion)`, `GetByEntity(ctx, convID, entityID)`
- `go-go-mento/go/pkg/snapshots/postgres_store.go` — `SnapshotStore` implementation using `pgxpool`
- `go-go-mento/go/pkg/snapshots/projector.go` — `TimelineProjector` and `ProjectAndPersist(ctx, store, event)` to map events → typed snapshots
- `go-go-mento/go/pkg/snapshots/context.go` — `WithConversationID/ConversationIDFromContext` helpers (conv_id override)

4) Integration points (minimal, non-breaking)
- `convertAndBroadcast` (in `go/pkg/webchat/conversation.go`): attach conv_id to context and call `ProjectAndPersist` side-channel before/after emitting frames. Alternatively, introduce `SemanticEventsFromEventWithProjection` mirroring the Pinocchio helper.
- HTTP Hydration API (new in `router.go`): `GET /api/conversations/{convId}/timeline?sinceVersion=` → fetch from `SnapshotStore` and return JSON array of typed snapshots (keys that `mapSnapshotToEntity` already expects: `kind`, `entity_id`, `version`, `updated_at`, ...).

### Frontend integration (web)
- `useChatStream.ts` already dispatches `hydrateTimelineThunk(convId)` on WS open. Keep as-is; endpoint now served by go-go-mento.
- `timelineSlice.ts` already has `mapSnapshotToEntity` and version-gated `upsertEntity`. Ensure hydration API returns fields aligned with this mapping.
- For new custom snapshots, extend `mapSnapshotToEntity` (or split into a registry) to map those kinds into UI entities.

### Custom event families → snapshot kinds (examples)
- `team.analysis.*` → `TeamAnalysisSnapshot` (captures progress/result, network score, visualization)
- `llm.*` → `LLMTextSnapshot` (assistant/user messages, final text)
- `tool.*` → `ToolCallSnapshot` + `ToolResultSnapshot`
- `agent.mode` → `AgentModeSnapshot`
- Families like `inner.thoughts.*`, `mode.evaluation.*`, `next.thinking.mode.*`, `debate.*` → define durable snapshots where rendering requires persistence across reloads (not every SEM event must be persisted).

### Authorization and safety
- Ensure `conv_id` authorization in hydration handler (the public server should attach identity; router enforces that the caller can access the conversation).
- Keep server-side validation (`Validate()`) on write to prevent corrupt data.

### Operations and scale
- Retention policy: periodic cleanup job by `updated_at`/`version` window (e.g., keep last N days or last M versions per entity).
- Observability: add logs/metrics around upsert conflicts, hydration size, and version skew.

### go-go-mento migration note
- This is greenfield for snapshots in go-go-mento. We can start directly with Protobuf JSON (no dual-read/Format flag needed). The dual-read section applies if we ever ingest legacy rows.

### If You Choose Option B (Protobuf + JSON Encoding)

This is a more substantial investment—plan for about 3-5 days initially, with ongoing work as you add new snapshot types:

1. **Define Proto Schemas** (1 day): Create `pinocchio/proto/snapshots/base.proto` for `SnapshotBase`, then one file per snapshot kind (e.g., `llm_text.proto`, `tool_call.proto`). Ensure field numbers are stable and use `optional` appropriately.

2. **Set Up Buf Build** (half day): Add `buf.yaml` and `buf.gen.yaml` to Pinocchio. Configure it to generate Go code in `pkg/snapshots/pb/` and optionally TypeScript if Pinocchio's frontend needs it. Add a `make protobuf` target. Test that codegen works.

3. **Implement Dual-Read** (1 day): Add the `Format` field to `SnapshotBase`. Update `codec.go` to detect whether a snapshot is legacy JSON or Protobuf JSON and route to the correct unmarshaler. Write unit tests for both paths.

4. **Switch Writers** (half day): Update `MarshalSnapshot` to use `protojson.Marshal`. Set `Format = 1` for all new snapshots. Deploy and monitor.

5. **Add Validation** (half day): Write a `Validate()` wrapper for proto-generated types. Call it in `Upsert`. Ensure it checks business-level constraints that proto can't express (e.g., "tool name must be in the registry").

6. **Rollout and Monitor** (ongoing): Deploy the dual-read version first, monitor for errors, then deploy the writer change. Old snapshots continue working; new ones are Protobuf-backed.

#### go-go-mento specific path (greenfield, PostgreSQL)

1. **Create snapshot protos** (1 day): `go-go-mento/proto/snapshots/*` including `SnapshotBase` and concrete kinds (LLM, tool call/result, team analysis, status, plus future widget families).
2. **Generate Go (and TS if needed)** (half day): Add Buf config to go-go-mento; generate into `go/pkg/snapshots/pb/`.
3. **PostgreSQL store** (1 day): Implement `go/pkg/snapshots/postgres_store.go` using `pgxpool` with version-gated upserts; add schema migration SQL.
4. **Projector and context helpers** (half day): `ProjectAndPersist`, `TimelineProjector`, `WithConversationID`.
5. **Integrate with streaming** (half day): In `convertAndBroadcast` (or via a helper), call projector/store as side-channel while continuing to emit SEM frames.
6. **Hydration API** (half day): Add `GET /api/conversations/{convId}/timeline?sinceVersion=` handler in `router.go`.
7. **Frontend validation** (half day): Ensure `mapSnapshotToEntity` covers new kinds; verify version gating behavior.
8. **Observability and retention** (half day): Add metrics/logs around upserts and hydration; schedule cleanup by `updated_at`.


