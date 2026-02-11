---
Title: Protobuf event generation in moments
Ticket: MISC-015-PROTOBUF-SKILL
Status: active
Topics:
    - events
    - go
    - serde
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-mento/Makefile
      Note: make protobuf target
    - Path: go-go-mento/buf.gen.yaml
      Note: Buf generation config for Go + TS
    - Path: go-go-mento/proto/sem/middleware/thinking_mode.proto
      Note: Upstream protobuf schema source
    - Path: moments/backend/pkg/sem/handlers/inference_handlers.go
      Note: Backend SEM mapping for inference events
    - Path: moments/backend/pkg/sem/registry/registry.go
      Note: SEM handler registry (backend)
    - Path: moments/web/src/platform/sem/handlers/thinkingMode.ts
      Note: Frontend SEM handler that decodes protobuf JSON
    - Path: moments/web/src/platform/sem/pb/proto/sem/middleware/thinking_mode_pb.ts
      Note: Generated TS protobuf artifact consumed by Moments
ExternalSources: []
Summary: 'Textbook-style guide to the SEM protobuf pipeline for moments: schema sources, Go/TS generation, event mapping, and frontend decoding.'
LastUpdated: 2026-02-01T17:15:00-05:00
WhatFor: Explain how protobuf schemas drive typed SEM event payloads in moments and how to generate/consume Go + TypeScript artifacts.
WhenToUse: Use when adding or updating SEM event payloads, regenerating protobuf outputs, or onboarding to the typed event pipeline.
---


# Protobuf event generation in moments

## Goal

Provide a precise, end-to-end explanation of how protobuf schemas define SEM event payloads and how those schemas turn into Go and TypeScript types that drive the Moments backend and UI.

## Context

Moments uses a typed-event architecture that is intentionally "JSON on the wire." The backend emits SEM frames as JSON over WebSocket for debuggability, while protobuf schemas supply the authoritative shape of the `event.data` payload. This gives us a fast path for inspection (plain JSON) and a strong path for correctness (shared schema across Go and TS).

Important nuance: within this workspace, Moments does **not** contain the proto source tree or Buf configs. The protobuf source-of-truth lives in sibling repos (notably `go-go-mento/proto/sem/**`, also `pinocchio/proto/sem/**`). The TypeScript artifacts committed under `moments/web/src/platform/sem/pb/proto/sem/**` are generated elsewhere and copied in. The guide below therefore describes both the upstream generation pipeline and how Moments consumes the outputs.

## Quick Reference

- **Proto source-of-truth (upstream):** `go-go-mento/proto/sem/{base,domain,middleware,team,timeline}`
- **Buf config (upstream):** `go-go-mento/buf.yaml`, `go-go-mento/buf.gen.yaml`
- **Codegen command (upstream):** `cd go-go-mento && make protobuf`
- **Go output (upstream):** `go-go-mento/go/pkg/sem/pb/proto/sem/...`
- **TS output (upstream):** `go-go-mento/web/src/sem/pb/proto/sem/...`
- **TS output (Moments):** `moments/web/src/platform/sem/pb/proto/sem/...`
- **Backend SEM mapping (Moments):** `moments/backend/pkg/sem/handlers/*.go`
- **Backend registry (Moments):** `moments/backend/pkg/sem/registry/registry.go`
- **Frontend decoding (Moments):** `moments/web/src/platform/sem/handlers/*.ts`
- **SEM frame envelope:**

```json
{ "sem": true, "event": { "type": "thinking.mode.started", "id": "...", "data": { "itemId": "..." } } }
```

## Conceptual Model: Typed schema, JSON wire

A good mental model is to treat protobuf as the **schema compiler** and SEM as the **distribution protocol**. The compiler guarantees that Go and TypeScript speak the same data language; the protocol keeps the wire easy to inspect and debug.

### Dataflow diagram

```
  proto/sem/*.proto (upstream)
            |
            | buf generate
            v
  Go protobuf types      TS protobuf types
  (go-go-mento/go/...)   (go-go-mento/web/...)
            |                    |
            | (Moments uses)     | (Copied into Moments)
            v                    v
  Go event -> SEM handler -> JSON SEM frame -> WebSocket -> TS handler -> Timeline entity
```

The strict invariant is: **`event.data` must match the proto JSON mapping for the chosen message.** Everything else is plumbing.

## Layer 1: Protobuf schemas (source-of-truth)

### Directory layout

Upstream schemas are organized by semantic family:

- `proto/sem/base`       : base LLM/tool/log shapes
- `proto/sem/middleware` : middleware event payloads
- `proto/sem/domain`     : domain-specific events (e.g., team analysis)
- `proto/sem/team`       : team selection payloads
- `proto/sem/timeline`   : timeline entity snapshots (not currently used in Moments)

The package and file names are intentionally stable; the JSON mapping (lowerCamel) is what the UI expects.

### JSON mapping (proto3)

Protobuf fields are defined in `snake_case` but serialized to `lowerCamel` in JSON by default. For example:

```
message ThinkingModePayload {
  string mode = 1;
  string phase = 2;
  string reasoning = 3;
  google.protobuf.Struct extra_data = 10;
}
```

Serializes to JSON like:

```
{ "mode": "...", "phase": "...", "reasoning": "...", "extraData": { ... } }
```

Moments backend handlers must therefore produce JSON keys that match the proto JSON mapping, not the Go struct tags used in event payloads.

## Layer 2: Code generation (Buf)

### Buf configuration (upstream)

`go-go-mento/buf.gen.yaml` defines two generators:

```yaml
plugins:
  - plugin: buf.build/bufbuild/es
    out: web/src/sem/pb
    opt:
      - target=ts
      - import_extension=none

  - plugin: buf.build/protocolbuffers/go
    out: go/pkg/sem/pb
    opt:
      - paths=source_relative
```

The `buf` CLI is invoked from the Makefile:

```bash
cd go-go-mento
make protobuf
```

That command runs `buf generate` via `go run` (pinning a Buf version), then regenerates the TypeScript artifacts and formats/lints the web output.

### What Moments actually consumes

- **TypeScript:** Moments commits the generated TS files under `moments/web/src/platform/sem/pb/proto/sem/**`. Headers in these files show `protoc-gen-es` as the generator and include a `go_package` path that points at go-go-mento, which strongly suggests these were generated upstream and copied in.
- **Go:** Moments currently does not store generated Go protobuf types. Backend SEM handlers therefore build JSON maps manually, aiming to match the proto JSON mapping.

## Layer 3: Backend event mapping (Moments)

### The SEM registry

The backend uses a registry keyed by concrete Go event types:

- Register: `semregistry.RegisterByType[*EventX](handler)`
- Dispatch: `semregistry.Handle(e)`

Handlers produce one or more SEM frames and wrap them in `{ sem: true, event: ... }` via `WrapSem`.

### Two valid mapping styles

Moments currently uses **manual map construction** (no Go protobuf types), but the intended style (seen upstream) is **proto -> JSON**.

#### Manual mapping (current Moments pattern)

```go
// Pseudocode of the current approach
payload := map[string]any{
  "itemId": ev.ItemID,
  "data": map[string]any{
    "mode": ev.Data.Mode,
    "phase": ev.Data.Phase,
    "reasoning": ev.Data.Reasoning,
    "extraData": ev.Data.ExtraData,
  },
}
frame := map[string]any{"type": "thinking.mode.started", "id": ev.ItemID, "data": payload}
return [][]byte{WrapSem(frame)}, nil
```

#### Protobuf mapping (upstream pattern)

```go
// Pseudocode of the upstream approach
msg := &semMw.ThinkingModeStarted{ItemId: ev.ItemID, Data: payload}
obj, err := pbToMap(msg) // protojson.Marshal -> map
frame := buildSemFrame("thinking.mode.started", ev.ItemID, obj, ev.Metadata())
return [][]byte{wrapSem(frame)}, nil
```

Both approaches must emit **JSON that matches the protobuf JSON mapping** expected by `fromJson(...)` on the frontend.

### Key helpers and invariants

- `WrapSem` wraps `{ sem: true, event: ... }`.
- `semEventID` and `semItemID` provide stable identifiers.
- **Invariant:** `event.type` and `event.data` must agree on schema; the UI does not do schema repair.

## Layer 4: Frontend decoding (Moments)

The frontend uses `@bufbuild/protobuf` and the generated `*Schema` descriptors to decode JSON maps into typed messages.

Example from a thinking mode handler:

```ts
const pb = fromJson(ThinkingModeStartedSchema, (ev.data || {}) as any);
const entityId = pb.itemId;
```

For `int64` fields, Buf returns `bigint` (or sometimes `string`), so handlers frequently coerce to number with helpers like:

```ts
function toNumber(value: unknown): number | undefined {
  if (typeof value === 'bigint') return Number(value);
  if (typeof value === 'string') return Number(value);
  if (typeof value === 'number') return value;
  return undefined;
}
```

### Not all handlers are typed (yet)

Some SEM handlers still parse plain JSON because no protobuf schema exists (for example, `team.member.removed`, question widgets, or debate widgets). These are explicitly marked in code with TODOs. The typed pipeline is therefore **partially adopted** in Moments.

## Worked example: thinking mode lifecycle

**Schema (upstream):** `proto/sem/middleware/thinking_mode.proto`

**Backend events:** `moments/backend/pkg/inference/events/thinking_mode.go` emits `thinking.mode.{started,update,completed}`.

**Backend SEM handler:** `moments/backend/pkg/sem/handlers/inference_handlers.go` maps event payloads into JSON maps that match the proto JSON mapping.

**Frontend handler:** `moments/web/src/platform/sem/handlers/thinkingMode.ts` decodes `ev.data` using `ThinkingModeStartedSchema` / `ThinkingModeUpdateSchema` / `ThinkingModeCompletedSchema` and emits timeline entities.

The key idea is that **`event.data` mirrors the protobuf JSON mapping even though Moments currently builds it by hand.**

## Adding a new typed SEM event (usage guide)

### 1) Define the protobuf schema (upstream)

Choose the right semantic family and add a message under `go-go-mento/proto/sem/...`.

### 2) Regenerate Go + TS

```bash
cd go-go-mento
make protobuf
```

This updates:

- `go-go-mento/go/pkg/sem/pb/proto/sem/...`
- `go-go-mento/web/src/sem/pb/proto/sem/...`

### 3) Sync TypeScript artifacts into Moments

Copy the generated TS modules into:

```
moments/web/src/platform/sem/pb/proto/sem/...
```

If any import paths assume `@/sem/...`, update them to the Moments alias (`@platform/sem/...`) used by the UI.

### 4) Implement the backend SEM mapping in Moments

- Add a handler under `moments/backend/pkg/sem/handlers`.
- Emit SEM frames with `WrapSem`.
- Ensure JSON keys match the proto JSON mapping.

### 5) Implement the frontend SEM handler in Moments

- Create or update a handler in `moments/web/src/platform/sem/handlers`.
- Use `fromJson(Schema, ev.data as JsonObject)`.
- Return `add`/`upsert` commands for the timeline store.

### 6) Validate end-to-end

- Start the backend, trigger the event, and confirm the UI receives SEM frames.
- If parsing fails, inspect the raw SEM payload and compare it to the proto JSON mapping.

## Common pitfalls and invariants

- **JSON field names:** proto fields are `snake_case` but JSON is `lowerCamel`.
- **int64 handling:** `fromJson` yields `bigint` or `string` for large integer fields; convert carefully.
- **google.protobuf.Struct:** JSON becomes an open object; avoid assuming fixed shape.
- **Schema drift:** if you update `.proto` but forget to regenerate TS, handlers will silently misparse.
- **Partial adoption:** some SEM handlers still use plain JSON; do not assume everything is typed.

## Pseudocode: canonical pipeline

```
function on_event(ev):
    msg = protobuf_message_for(ev)        # or manual map that mirrors JSON mapping
    data = proto_to_json_map(msg)
    frame = { sem: true, event: { type: ev.type, id: ev.id, data: data } }
    send_over_ws(frame)

function on_sem_frame(frame):
    schema = schema_registry[frame.event.type]
    pb = fromJson(schema, frame.event.data)
    entity = to_timeline_entity(pb)
    upsert(entity)
```

## Usage Examples

### Example A: backend mapping (manual JSON)

```go
// moments/backend/pkg/sem/handlers/inference_handlers.go (pattern)
sem := map[string]any{
  "type": "thinking.mode.started",
  "id":   semEventID(md.ID, ev.ItemID),
  "data": map[string]any{
    "itemId": ev.ItemID,
    "data": map[string]any{
      "mode": ev.Data.Mode,
      "phase": ev.Data.Phase,
      "reasoning": ev.Data.Reasoning,
      "extraData": ev.Data.ExtraData,
    },
  },
}
return [][]byte{WrapSem(sem)}, nil
```

### Example B: frontend decoding

```ts
import { fromJson } from '@bufbuild/protobuf';
import { ThinkingModeStartedSchema } from '../pb/proto/sem/middleware/thinking_mode_pb';

const pb = fromJson(ThinkingModeStartedSchema, (ev.data || {}) as any);
const id = pb.itemId;
```

### Example C: regenerate protobuf outputs (upstream)

```bash
cd go-go-mento
make protobuf
```

## Related

- `moments/backend/pkg/sem/handlers/*.go` (backend SEM mapping)
- `moments/web/src/platform/sem/handlers/*.ts` (frontend SEM decoding)
- `go-go-mento/buf.gen.yaml` and `go-go-mento/proto/sem/**` (schema + codegen)
