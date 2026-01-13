---
Title: 'Design: context-carried tool registry + serializable Turn.Data'
Ticket: 001-PASS-TOOLS-THROUGH-CONTEXT
Status: active
Topics:
    - geppetto
    - turns
    - tools
    - context
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design for passing runtime tool registry via context, storing only serializable tool definitions in Turn.Data, and updating engines/middleware/executors accordingly."
LastUpdated: 2025-12-18T12:07:47.119564179-05:00
---

# Design: context-carried tool registry + serializable Turn.Data

## Executive Summary

Stop storing runtime `tools.ToolRegistry` objects inside `Turn.Data`. Instead:

- **Serialize** tool availability as `[]ToolDefinition` (or equivalent) inside `Turn.Data` for persistence + replay.
- **Carry runtime executors** (a live `ToolRegistry`) through the inference/middleware/tool-execution pipeline via `context.Context`.

This aligns with the hard constraint that Turn/Block Data+Metadata must remain serializable, while still supporting dynamic per-turn tools and middleware that can add/modify tools.

## Problem Statement

Today we attach a `tools.ToolRegistry` interface value to `Turn.Data[turns.DataKeyToolRegistry]`. This causes:

- Non-serializable values inside turn state (bad for YAML/JSON, DB persistence, replay)
- Ad-hoc type assertions at read sites
- Special-casing at persistence boundaries to avoid serializing the registry object

We need a design that:

- Keeps Turn/Block bags serializable
- Still lets engines advertise tools
- Still lets executors run tools
- Still lets middleware adjust the available tool set

## Proposed Solution

### 1) Introduce a context helper for tool registry

Add a small package (name bikeshed): `geppetto/pkg/inference/toolcontext`:

```go
// toolcontext/context.go (sketch)
package toolcontext

type ctxKey struct{}

func WithRegistry(ctx context.Context, reg tools.ToolRegistry) context.Context
func RegistryFrom(ctx context.Context) (tools.ToolRegistry, bool)
```

Rules:

- Middleware that needs a registry uses `toolcontext.RegistryFrom(ctx)`
- Entry points (routers/handlers) use `toolcontext.WithRegistry(ctx, reg)` instead of storing the registry in the Turn

### 2) Store tool definitions (serializable) on the Turn

Add a new Turn.Data key, e.g.:

- `turns.DataKeyToolDefinitions` (or similar)

Store:

- `[]tools.ToolDefinition` (or whatever canonical “definition” struct is used across providers)

Engines read tool definitions from Turn.Data to construct provider tool schema payloads.

### 3) Tool execution uses context registry (not Turn.Data)

Tool execution should receive a `context.Context` and resolve tool executors from the registry in that context.

If an executor is not found:

- return a structured tool error (and keep existing debug logging; do not silence)

## Design Decisions

### Decision: context is the correct “runtime-only” carrier

Rationale:

- per-request lifetime, easy propagation, no persistence coupling
- avoids mixing “snapshot state” (turn) with “live services” (executors)

### Decision: tool definitions are the serializable representation

Rationale:

- engines and tracing only need definitions
- definitions are stable and can be persisted/replayed

### Decision: keep Turn.Data serializable; no runtime objects

Rationale:

- simplifies persistence and makes invariants clear (no special-cases)

## Alternatives Considered

### Alternative A: keep storing ToolRegistry in Turn.Data and “just don’t persist it”

Rejected because it violates the “all bags are serializable” rule and keeps the system in an inconsistent state (runtime-only values mixed into a persisted object graph).

### Alternative B: add `Turn.Runtime` (not serialized) alongside `Turn.Data`

Viable, but more invasive: touches core `turns` types and serializers. Context-based passing is less invasive and aligns with Go’s existing request-scoped patterns.

### Alternative C: store tool registry as JSON (serialize executors)

Not realistic: executors are code; you can’t serialize them meaningfully.

## Implementation Plan

1. Introduce `toolcontext.WithRegistry/RegistryFrom`
2. Add `turns.DataKeyToolDefinitions` and wire up writers (routers/handlers)
3. Update engines to read definitions list instead of registry object
4. Update middleware/tracing to read definitions list
5. Update any tool execution path to resolve executors from `ctx` registry
6. Update `pinocchio/pkg/middlewares/sqlitetool/middleware.go` to mutate:
   - runtime registry via `ctx`, and/or
   - definitions list + rebuild registry (pick one consistent approach)
7. Remove old uses of `turns.DataKeyToolRegistry` (or reserve it only for serializable forms if desired)
8. Add tests:
   - engine request contains tools when definitions exist
   - turn YAML/JSON round-trips with tool definitions
   - tool execution fails clearly when registry missing from ctx

## Open Questions

- Should “tool definitions” be the canonical input everywhere (registry built from defs), or do we allow registry-only in ctx for some paths?
- Do we need a single canonical `ToolDefinition` type across providers (OpenAI/Claude/Gemini), or an internal neutral one that gets mapped?
- How do we want to version the Turn.Data keys (vs/slug/version) once we implement the new typed-key system?

## References

- Analysis doc in this ticket: `analysis/01-analysis-passing-tool-registry-through-context.md`

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
