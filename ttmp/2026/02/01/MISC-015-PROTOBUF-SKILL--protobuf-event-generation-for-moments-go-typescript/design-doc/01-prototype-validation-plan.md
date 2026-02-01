---
Title: Prototype validation plan
Ticket: MISC-015-PROTOBUF-SKILL
Status: active
Topics:
    - events
    - go
    - serde
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Plan for a minimal proto -> Go/TS -> JSON -> TS decode prototype that validates strict schema exchange."
LastUpdated: 2026-02-01T22:35:00-05:00
WhatFor: "Define a minimal end-to-end protobuf -> JSON -> TS decode prototype to validate strict schema exchange between Go and TypeScript."
WhenToUse: "Use before implementing the test repo prototype or when validating updates to the protobuf event generation and strict schema exchange references."
---

# Prototype validation plan

## Executive Summary

Build a minimal end-to-end prototype in `/home/manuel/workspaces/2026-02-01--test-protobuf-ts-go-skill` that mirrors the go-go-mento strict schema exchange: proto schema -> buf generation (Go + TS) -> Go emits JSON (protojson) -> TS decodes with `fromJson`. The prototype validates schema-first invariants, JSON mapping rules (camelCase), `int64` handling, `google.protobuf.Struct` behavior, and the SEM-style JSON envelope without needing the full Moments stack.

## Problem Statement

The reference documents now describe a strict schema exchange model (proto -> Go/TS -> protojson -> TS `fromJson`) and emphasize that protobuf is the only authoritative schema. We need a concrete, minimal example to prove that the instructions are correct (camelCase naming, `int64` behavior, `google.protobuf.Struct` handling, schema_version usage, and schema alignment) before applying the approach in larger repos.

## Proposed Solution

Create a tiny proto schema with fields that exercise the strict exchange rules (schema_version, snake_case -> lowerCamel, nested message, `int64`, `google.protobuf.Struct`, map/repeated). Use Buf to generate Go + TS artifacts. Implement:

- A Go CLI that builds a SEM frame (JSON) using `protojson` (canonical) and writes it to disk for inspection.
- A TypeScript script that uses `fromJson` with the generated `*Schema` to parse `event.data` and prints/validates key fields.

The prototype keeps everything local (no WebSocket or UI). It outputs a JSON blob from Go, then a TS script consumes it to validate the decoding behavior.

## Design Decisions

- **Use a single, minimal proto schema** to keep the signal focused on strict schema exchange behavior.
- **Include `schema_version`, `int64`, and `google.protobuf.Struct`** to validate the tricky cases called out in the reference doc.
- **Use protojson in Go** to match the canonical go-go-mento approach (camelCase field names).
- **Stay JSON-on-the-wire** to mirror the SEM envelope and avoid coupling to protobuf binary encoding.
- **No full product dependencies** to reduce setup overhead; the goal is to validate the pipeline, not to run the product stack.

## Alternatives Considered

- **Using the real go-go-mento/Moments repos**: higher fidelity but too heavy for a quick correctness probe.
- **Skipping Go and only testing TS**: would miss the critical server-side JSON mapping concerns.
- **Binary protobuf over the wire**: contradicts the SEM "JSON on the wire" invariant that the reference document emphasizes.

## Implementation Plan

1. Inspect repo layout and confirm where proto, Go, and TS should live (including generated outputs).
2. Add a proto file with fields that exercise strict schema exchange rules (schema_version, int64, Struct, map/repeated).
3. Add Buf config + generation script to produce Go + TS outputs.
4. Implement Go emitter to output a SEM frame JSON payload via `protojson`.
5. Implement TS consumer that decodes via `fromJson` and validates values, including `int64` handling.
6. (Optional) Add a minimal JSON schema dump based on protobuf descriptors to mirror go-go-mento tooling.
7. Run the end-to-end flow and record results in the validation diary.

## Open Questions

- Should the Go emitter also include a manual map path to compare against protojson output?
- Where should generated TS files live in the test repo to mimic the canonical import paths?
- Do we need a JSON schema dump in the prototype, or is Go <-> TS validation sufficient?

## References

- `geppetto/ttmp/2026/02/01/MISC-015-PROTOBUF-SKILL--protobuf-event-generation-for-moments-go-typescript/reference/01-protobuf-event-generation-in-moments.md`
- `geppetto/ttmp/2026/02/01/MISC-015-PROTOBUF-SKILL--protobuf-event-generation-for-moments-go-typescript/reference/04-go-go-mento-strict-schema-exchange-between-go-and-typescript.md`
