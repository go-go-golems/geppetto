---
Title: 'Postmortem: protobuf strict schema exchange prototype'
Ticket: MISC-015-PROTOBUF-SKILL
Status: active
Topics:
    - events
    - go
    - serde
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../2026-02-01--test-protobuf-ts-go-skill/buf.gen.yaml
      Note: |-
        Buf v2 generation config; remote plugin syntax.
        Buf v2 generation config (remote plugin syntax)
    - Path: ../../../../../../../../../2026-02-01--test-protobuf-ts-go-skill/go/cmd/emitter/main.go
      Note: |-
        Go protojson emitter and SEM frame wrapper.
        Go protojson SEM emitter
    - Path: ../../../../../../../../../2026-02-01--test-protobuf-ts-go-skill/out/consume-report.json
      Note: |-
        Parsed TS report (BigInt, Struct, map, repeated).
        TS decode report
    - Path: ../../../../../../../../../2026-02-01--test-protobuf-ts-go-skill/out/sem-frame.json
      Note: |-
        JSON SEM frame output from Go emitter.
        Go output SEM frame
    - Path: ../../../../../../../../../2026-02-01--test-protobuf-ts-go-skill/proto/sem/example/v1/example_event.proto
      Note: |-
        Prototype schema exercising schema_version, int64, Struct, map, repeated.
        Prototype schema with schema_version
    - Path: ../../../../../../../../../2026-02-01--test-protobuf-ts-go-skill/web/src/consume.ts
      Note: |-
        TS fromJson consumer and validation report.
        TS fromJson consumer
ExternalSources: []
Summary: Postmortem of the strict schema exchange prototype with concrete corrections and documentation improvements.
LastUpdated: 2026-02-01T22:55:00-05:00
WhatFor: Capture what was validated, what broke, and which reference details need clarification.
WhenToUse: Use after running the prototype or when revising the strict schema exchange reference.
---


# Postmortem: Protobuf strict schema exchange prototype

## Executive summary

The prototype successfully validated the strict schema exchange pipeline: proto schema -> Buf generation -> Go protojson emission -> TS fromJson decoding. The most important findings are about tooling versions (Buf v2 plugin syntax and @bufbuild/protobuf v2 runtime), protojson int64 string encoding, and the actual TS shape of google.protobuf.Struct (JsonObject, not a Struct message). These should be made explicit in the reference doc to prevent mis-implementation.

## What we built

- Minimal proto schema with schema_version, int64, Struct, map, and repeated fields.
- Buf v2 generation config for Go + TypeScript (bufbuild/es + protocolbuffers/go).
- Go emitter that builds a message and emits a SEM-style JSON frame via protojson.
- TypeScript consumer that decodes JSON with fromJson and reports types/values.

## Validation results (key observations)

- **protojson int64 encoding:** JSON output uses strings for int64 fields (sequence_id, started_at_ms). TS fromJson parses them to **bigint**.
- **BigInt safety:** JSON.stringify fails on BigInt unless a replacer converts to string; converting to number loses precision for values above 2^53.
- **google.protobuf.Struct in TS:** Generated field type is JsonObject (plain JS object), not Struct with fields/kind. This affects how you inspect values in TS.
- **Buf v2 plugin config:** buf.gen.yaml uses `remote:` (not `plugin:`). Using the v1-style key fails with buf v2.
- **Generation path:** With `paths=source_relative`, generated Go/TS output keeps the `proto/` prefix. go_package should match the actual output path to avoid import confusion.

## Corrections needed in the original reference

1) **Buf v2 config syntax**
   - Clarify that buf v2 uses `remote:` for plugins in `buf.gen.yaml`. Provide a working snippet.

2) **Runtime/library version alignment**
   - The TypeScript runtime must match the code generator output. `protoc-gen-es v2.x` requires `@bufbuild/protobuf v2.x`.

3) **int64 JSON semantics**
   - protojson serializes int64 as strings in JSON. TS fromJson returns bigint. Emphasize BigInt handling (stringify, numeric conversion pitfalls).

4) **Struct handling in TS**
   - `google.protobuf.Struct` maps to JsonObject in generated TS (not Struct message with `fields`). Show how to read nested values directly.

5) **Output path conventions**
   - When using `paths=source_relative`, generated outputs include the `proto/` path segment. Mention how to set go_package accordingly, or switch path options if a flatter output is desired.

6) **Schema_version usage**
   - Add an explicit example of `schema_version` in a proto and show it round-tripping through Go/TS.

## Recommended guide and reference structure

A good reference should separate conceptual rules from concrete setup and troubleshooting:

1) **Purpose and invariants**
   - Schema-first, JSON transport, generated types only.
   - Required: schema_version, protojson camelCase, bigint handling.

2) **Prerequisites and versions**
   - buf version (v2), plugin versions, @bufbuild/protobuf runtime.
   - Known compatible version matrix.

3) **Repository layout**
   - proto/ tree, output paths for Go + TS, where generated code lives.

4) **Schema authoring**
   - Naming conventions, required fields, Struct/map/repeated examples.

5) **Code generation**
   - buf.yaml + buf.gen.yaml snippets (v2 syntax).
   - Makefile target or script to run buf generate.

6) **Go emission**
   - protojson usage (options), JSON output for int64, SEM envelope example.

7) **TypeScript decoding**
   - fromJson usage, bigint handling, Struct as JsonObject, sample validation checks.

8) **Validation checklist**
   - Round-trip steps and expected output signatures.

9) **Troubleshooting**
   - Common errors (buf plugin syntax, @bufbuild/protobuf version mismatch, BigInt JSON.stringify).

## What we would clarify further

- Provide a single, runnable "golden" example (proto + Go + TS) in the reference to remove ambiguity.
- Add a "gotchas" callout for int64 -> string -> bigint and Struct -> JsonObject.
- Explicitly show the SEM frame envelope example produced by Go and the expected TS output report.

