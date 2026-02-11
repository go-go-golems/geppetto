---
Title: 'Go-go-mento: strict schema exchange between Go and TypeScript'
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
      Note: Buf codegen for Go + TS
    - Path: go-go-mento/go/cmd/mento-service/cmds/webchat/entities/schemas.go
      Note: CLI for dumping schemas
    - Path: go-go-mento/go/pkg/persistence/timelinehydration/projector.go
      Note: protojson -> map conversion
    - Path: go-go-mento/go/pkg/webchat/schemas/from_pb.go
      Note: JSON schema generation from protobuf descriptors
    - Path: go-go-mento/proto/sem/timeline/message.proto
      Note: Schema example with schema_version
    - Path: go-go-mento/proto/sem/timeline/tool.proto
      Note: Schema example with Struct fields
ExternalSources: []
Summary: Reference architecture for Go <-> TS schema exchange in go-go-mento using protobuf + protojson + Buf, with JSON schema derivation from protobuf descriptors.
LastUpdated: 2026-02-01T17:05:00-05:00
WhatFor: Explain go-go-mento's canonical workflow for sharing strict schemas between Go and TypeScript while keeping JSON as the transport format.
WhenToUse: Use when designing new cross-language payloads or validating data exchange rules between Go services and the TS frontend.
---


# Go-go-mento: strict schema exchange between Go and TypeScript

## Goal

Document the go-go-mento architecture for exchanging structured data between Go and TypeScript using a single authoritative schema, while still shipping JSON over the wire and into storage. This reference treats protobuf as the schema compiler (not necessarily the wire format) and shows how Go, TS, and JSON schemas stay aligned.

## Context

go-go-mento uses protobuf as the canonical schema language and Buf as the code generator. Rather than switching the transport to binary protobuf, it keeps JSON at the boundaries (WebSocket frames, DB JSONB payloads, REST responses) for debuggability. The invariants are enforced by:

- single source-of-truth proto definitions
- generated Go + TypeScript types
- protojson serialization rules (camelCase field names)
- JSON schema generation by reflecting over protobuf descriptors

This design enables strict schema consistency without sacrificing the practical benefits of JSON transports.

## Quick Reference

### Canonical pipeline

```
proto schema (go-go-mento/proto/...)  --buf generate-->  Go + TS types
                   |                                           |
                   |                                   (protojson)
                   +-------------- JSON schema <---------------+
                                (descriptor reflection)
```

### Key files

- Schema definitions: `go-go-mento/proto/sem/timeline/*.proto`
- Buf config: `go-go-mento/buf.yaml`, `go-go-mento/buf.gen.yaml`
- Codegen command: `go-go-mento/Makefile` target `make protobuf`
- JSON map conversion (Go): `go-go-mento/go/pkg/persistence/timelinehydration/projector.go` (`protoToMap`)
- JSON schema from protobuf descriptors (Go): `go-go-mento/go/pkg/webchat/schemas/from_pb.go`
- CLI to dump schemas (Go): `go-go-mento/go/cmd/mento-service/cmds/webchat/entities/schemas.go`
- TS decoding helpers: `go-go-mento/web/src/sem/handlers/*.ts` (uses `fromJson`)

### Essential invariants

- Protobuf is the only authoritative schema.
- JSON field names follow protojson rules (camelCase).
- Generated Go + TS artifacts are committed together.
- JSON schemas are derived from protobuf descriptors (not handwritten).

## Architecture narrative (Norvig-style)

The system begins with a single choice: we want the *precision* of a schema language and the *ergonomics* of JSON. Protobuf supplies the precision. JSON supplies the ergonomics. The rest of the architecture is the bridge between these two worlds.

A schema in `proto/` describes the payload. Buf generates code for Go and TypeScript. Go code constructs typed messages, then renders them to JSON using `protojson`. TypeScript uses `@bufbuild/protobuf` to parse the JSON back into strongly typed structures. At no point is a handwritten Go struct or TS interface allowed to define the payload shape; those are always derived from the same proto.

Finally, the system goes one step further: it introspects protobuf descriptors to emit JSON Schemas. These schemas serve as a second, machine-checkable contract that can be used by tooling, CLI output, or validation pipelines. The result is a circular contract: protobuf defines the data, codegen enforces it in Go/TS, protojson renders it to JSON, and JSON Schema exposes it for external tooling.

## Core mechanism: Protobuf as schema compiler

### 1) Schema definition

Example: a timeline message snapshot.

```proto
// go-go-mento/proto/sem/timeline/message.proto
message MessageSnapshotV1 {
  uint32 schema_version = 1; // =1
  string role = 2;           // "user" | "assistant"
  string content = 3;        // text
  bool streaming = 4;
  map<string, string> metadata = 10;
}
```

### 2) Code generation (Buf)

`buf.gen.yaml` generates both Go and TypeScript outputs:

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

Command:

```bash
cd go-go-mento
make protobuf
```

### 3) Go -> JSON (protojson)

Go produces a protobuf message and converts it to a JSON map with protojson.

```go
payload := &tlpb.MessageSnapshotV1{
  SchemaVersion: 1,
  Role:          "assistant",
  Content:       ev.Text,
  Streaming:     false,
}

m, err := protoToMap(payload) // protojson.Marshal -> map[string]any
```

### 4) TS <- JSON (fromJson)

TypeScript parses the JSON map back into a typed protobuf message.

```ts
const pb = fromJson(MessageSnapshotV1Schema, ev.data as any);
const content = pb.content;
```

### 5) JSON Schema (descriptor reflection)

Protobuf descriptors are reflected into JSON Schema for tool-facing contract dumps.

```go
schemas := webs.PBEntitySchemas()
// kind -> schema (JSON Schema map)
```

## JSON schemas: a second contract layer

The JSON schema generator is deliberately mechanical. It walks the protobuf descriptors, converts scalar types, expands repeated fields to arrays, and treats `google.protobuf.Struct` as an open object. It also preserves protojson naming (camelCase) to match actual payloads.

Key rules from `from_pb.go`:

- `string` -> JSON string
- `int32/int64/...` -> JSON integer
- `float/double` -> JSON number
- `enum` -> JSON string with `enum` values
- `repeated` -> JSON array
- `google.protobuf.Struct` -> open object (`additionalProperties: true`)

This gives a direct mechanical proof that Go and TS are interpreting the same payload shape that the JSON schema describes.

## Example: tool snapshots (schema + conversion)

### Protobuf definition

```proto
// go-go-mento/proto/sem/timeline/tool.proto
message ToolCallSnapshotV1 {
  uint32 schema_version = 1;
  string name = 2;
  google.protobuf.Struct input = 3;
  string status = 4;   // queued|running|completed|error
  double progress = 5; // 0..1
}
```

### Go conversion

```go
payload := &tlpb.ToolCallSnapshotV1{
  SchemaVersion: 1,
  Name:          ev.ToolCall.Name,
  Input:         toStructFromJSONString(ev.ToolCall.Input),
  Status:        "running",
  Progress:      1.0,
}

m, err := protoToMap(payload)
```

### TS decoding

```ts
const pb = fromJson(ToolCallSnapshotV1Schema, ev.data as any);
const status = pb.status;
```

## Design principles (portable beyond SEM events)

1) **Schema-first:** protobuf is the single source-of-truth; everything else derives.
2) **JSON transport:** keep JSON at the edges for debugging and tool compatibility.
3) **Generated types only:** do not handwrite parallel Go or TS payload structs.
4) **Descriptor-derived JSON schemas:** use reflection to prevent schema drift.
5) **Schema versioning inside payloads:** include `schema_version` fields to allow safe evolution.

## Pseudocode: full exchange loop

```
# Author a schema
proto = define_message("SnapshotV1", fields...)

# Generate code
buf_generate(proto) -> go_types, ts_types

# Produce in Go
msg = SnapshotV1(...)
json_payload = protojson(msg)
store_or_send(json_payload)

# Consume in TS
msg = fromJson(SnapshotV1Schema, json_payload)
render(msg)

# Verify schema
schemas = descriptors_to_json_schema(proto)
```

## Usage Examples

### Example A: add a new cross-language payload

1. Add a new message under `proto/sem/timeline/*.proto`.
2. Run `make protobuf` to regenerate Go + TS types.
3. Update Go code to create the typed message and call `protoToMap`.
4. Update TS code to parse with `fromJson` using the generated `*Schema`.
5. (Optional) Dump JSON schemas via the `webchat entities schemas` command to verify the payload contract.

### Example B: dump JSON schemas (CLI)

```bash
# go-go-mento CLI: dump entity payload schemas derived from protobuf
mento-service webchat entities schemas
```

## Related

- `go-go-mento/buf.gen.yaml` (codegen configuration)
- `go-go-mento/proto/sem/timeline/*.proto` (authoritative schemas)
- `go-go-mento/go/pkg/persistence/timelinehydration/projector.go` (protojson -> map)
- `go-go-mento/go/pkg/webchat/schemas/from_pb.go` (descriptor reflection to JSON schema)
- `go-go-mento/go/cmd/mento-service/cmds/webchat/entities/schemas.go` (CLI schema dump)
