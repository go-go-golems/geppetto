---
Title: 'InferenceResult implementation plan: metadata contract, RunInferenceWithResult wrapper, provider parity, and legacy key migration'
Ticket: MEN-TR-005
Status: active
Topics:
    - temporal-relationships
    - geppetto
    - stop-policy
    - claude
    - extraction
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/metadata.go
      Note: LLMInferenceData struct used for normalized inference metadata
    - Path: geppetto/pkg/inference/engine/engine.go
      Note: |-
        Existing Engine contract and RunInference signature
        Current engine interface baseline
    - Path: geppetto/pkg/inference/engine/run_with_result.go
      Note: Implemented wrapper API from this plan
    - Path: geppetto/pkg/inference/middleware/middleware.go
      Note: Middleware handler signature coupling
    - Path: geppetto/pkg/inference/session/builder.go
      Note: InferenceRunner interface uses RunInference today
    - Path: geppetto/pkg/inference/toolloop/enginebuilder/builder.go
      Note: Middleware wiring and runner call path
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Main loop orchestration call to RunInference
    - Path: geppetto/pkg/js/modules/geppetto/api_types.go
      Note: JS binding engine adapter implementing RunInference
    - Path: geppetto/pkg/steps/ai/claude/engine_claude.go
      Note: |-
        Provider implementation to populate result metadata
        Provider parity implementation
    - Path: geppetto/pkg/steps/ai/gemini/engine_gemini.go
      Note: Provider implementation already setting some turn metadata
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: Provider implementation to populate result metadata
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: |-
        Provider implementation to populate result metadata
        Provider parity target
    - Path: geppetto/pkg/turns/inference_result.go
      Note: Implemented canonical metadata schema from this plan
    - Path: geppetto/pkg/turns/keys_gen.go
      Note: |-
        Canonical turn metadata keys to extend and migrate
        Turn metadata key migration target
    - Path: temporal-relationships/internal/extractor/gorunner/run.go
      Note: |-
        Downstream stop-policy consumer to switch to canonical InferenceResult metadata
        Canonical-first stop-policy consumer migration
        Consumer migration implementation
    - Path: temporal-relationships/internal/extractor/httpapi/server.go
      Note: REST/SSE output compatibility impacts
ExternalSources: []
Summary: Detailed implementation plan to introduce a canonical InferenceResult metadata section on Turn, add a RunInferenceWithResult wrapper, make providers populate it consistently, and migrate consumers away from legacy scalar keys.
LastUpdated: 2026-03-02T17:05:00-05:00
WhatFor: Convert inference-outcome signaling into a deterministic cross-provider contract without breaking current RunInference call chains.
WhenToUse: Use when implementing InferenceResult, changing engine/session contracts, or migrating stop-policy and extraction consumers to canonical result metadata.
---



# InferenceResult implementation plan: metadata contract, RunInferenceWithResult wrapper, provider parity, and legacy key migration

## Executive Summary

This plan adopts your requested direction with minimal interface churn:

1. Add a canonical structured inference outcome to `Turn.Metadata` (`inference_result`).
2. Add `RunInferenceWithResult(ctx, turn) -> (turn, result, error)` as a compatibility wrapper around existing `RunInference`.
3. Make all providers populate canonical inference result data consistently.
4. Migrate consumers (especially loop stop policy) to read canonical result first.
5. Deprecate and eventually remove direct legacy scalar-key usage (`stop_reason`, `usage`, `model`) after migration gates.

This avoids a hard break to `engine.Engine` while achieving deterministic semantics for stop reason, usage, truncation, and completion class.

## Problem Being Solved

Today inference outcome data is fragmented:

1. `RunInference` returns only `*turns.Turn`.
2. Event metadata often contains richer stop/usage details than turn metadata.
3. Provider engines are inconsistent in what they persist onto the turn.
4. Loop policy in `temporal-relationships` reads turn scalar keys only.

This creates runtime ambiguity (`max_tokens` seen in stream, absent in turn metadata), making stop behavior provider-dependent.

## Design Goals

1. Define a single canonical durable inference outcome contract.
2. Keep existing `RunInference` call graph operational during migration.
3. Enforce provider parity through tests and helper utilities.
4. Support progressive migration for JS bindings, session/toolloop, and app consumers.
5. Reduce long-term dependency on scattered scalar keys.

## Non-Goals

1. Immediate removal of all legacy keys in one PR.
2. Replacing event streaming metadata model.
3. Reworking toolloop orchestration interface in this step.

## Proposed Contract

### 1) New canonical `InferenceResult` payload in `Turn.Metadata`

Add a typed metadata key backed by a versioned struct.

```go
// package engine (or package turns/inferencekeys if preferred)
type FinishClass string

const (
    FinishCompleted        FinishClass = "completed"
    FinishMaxTokens        FinishClass = "max_tokens"
    FinishToolCallsPending FinishClass = "tool_calls_pending"
    FinishInterrupted      FinishClass = "interrupted"
    FinishError            FinishClass = "error"
    FinishUnknown          FinishClass = "unknown"
)

type InferenceResult struct {
    Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
    Model    string `json:"model,omitempty" yaml:"model,omitempty"`

    StopReason  *string      `json:"stop_reason,omitempty" yaml:"stop_reason,omitempty"`
    FinishClass FinishClass  `json:"finish_class,omitempty" yaml:"finish_class,omitempty"`
    Truncated   bool         `json:"truncated,omitempty" yaml:"truncated,omitempty"`
    Usage       *events.Usage `json:"usage,omitempty" yaml:"usage,omitempty"`
    MaxTokens   *int         `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
    DurationMs  *int64       `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty"`

    RequestID   string       `json:"request_id,omitempty" yaml:"request_id,omitempty"`
    ResponseID  string       `json:"response_id,omitempty" yaml:"response_id,omitempty"`
    Extra       map[string]any `json:"extra,omitempty" yaml:"extra,omitempty"`
}
```

Codegen key addition in turns spec:

```yaml
# geppetto/pkg/spec/geppetto_codegen.yaml
turn_metadata:
  - name: InferenceResult
    type_expr: engine.InferenceResult
    value_key: inference_result
```

Resulting typed key target:

```go
turns.KeyTurnMetaInferenceResult
```

### 2) New wrapper API

Do not break `engine.Engine` now. Introduce a wrapper helper + optional interface.

```go
// optional provider-native contract
// implemented gradually by engines that can return richer result directly
type EngineWithResult interface {
    RunInferenceWithResult(ctx context.Context, t *turns.Turn) (*turns.Turn, *InferenceResult, error)
}

// compatibility helper; default path wraps old RunInference + extraction from turn metadata
func RunInferenceWithResult(ctx context.Context, eng Engine, t *turns.Turn) (*turns.Turn, *InferenceResult, error)
```

Wrapper behavior:

1. If `eng` implements `EngineWithResult`, call directly.
2. Else call `eng.RunInference`.
3. Normalize result by reading `KeyTurnMetaInferenceResult`; if absent, synthesize from legacy keys and safe defaults.
4. Backfill canonical `KeyTurnMetaInferenceResult` if synthesis happened.

### 3) Legacy key policy

Short term:

1. Keep writing legacy scalar keys for compatibility.
2. Canonical source of truth becomes `inference_result`.

Mid term:

1. Consumers read canonical first, legacy fallback second.
2. Add deprecation warnings for direct legacy reads in key call sites.

Long term:

1. Stop direct writes of scalar keys (optional mirror kept behind flag if needed).
2. Remove fallback code once all known consumers are migrated.

## Architecture Diagram

```mermaid
flowchart LR
    A[Provider Engine\nClaude/OpenAI/Responses/Gemini] --> B[RunInference]
    B --> C[Turn Metadata\nKeyTurnMetaInferenceResult]
    C --> D[RunInferenceWithResult wrapper\n(normalize + backfill)]
    D --> E[Session/Toolloop/Gorunner]
    C --> F[Persistence]
    G[Event Metadata] --> D
    G --> H[SSE/UI telemetry]
```

## Run Path Pseudocode

```go
func RunInferenceWithResult(ctx context.Context, eng engine.Engine, t *turns.Turn) (*turns.Turn, *engine.InferenceResult, error) {
    if er, ok := eng.(engine.EngineWithResult); ok {
        out, res, err := er.RunInferenceWithResult(ctx, t)
        if err != nil {
            return out, res, err
        }
        if out != nil && res != nil {
            _ = turns.KeyTurnMetaInferenceResult.Set(&out.Metadata, *res)
            mirrorLegacyKeys(out, res)
        }
        return out, res, nil
    }

    out, err := eng.RunInference(ctx, t)
    if err != nil {
        return out, nil, err
    }
    if out == nil {
        out = t
    }

    res := extractOrSynthesizeInferenceResult(out)
    _ = turns.KeyTurnMetaInferenceResult.Set(&out.Metadata, res)
    mirrorLegacyKeys(out, &res)

    return out, &res, nil
}
```

## Provider Fill Requirements

All providers must set canonical result fields before returning.

### Required fields (all engines)

1. `Provider`
2. `Model`
3. `StopReason` when known
4. `FinishClass` (derived)
5. `Truncated` (true for `max_tokens` and equivalent provider semantics)
6. `Usage` when known
7. `DurationMs`

### Optional fields

1. `RequestID`
2. `ResponseID`
3. `MaxTokens`
4. `Extra` map for provider-specific payload values

### Provider-specific notes

1. Claude:
   - ensure final stop reason captured from streaming delta path is projected into canonical result.
2. OpenAI chat:
   - map finish reason and usage from stream chunks to canonical fields.
3. OpenAI responses:
   - map `response.completed` envelope stop reason + usage to canonical fields.
4. Gemini:
   - migrate current scalar-key writes to canonical first; keep scalar projection during compatibility window.

## Consumer Migration Plan

### Priority consumer: `temporal-relationships` gorunner

Current stop read path:

1. `stopReasonOfTurn` reads legacy scalar key only.

Target behavior:

1. Read canonical `KeyTurnMetaInferenceResult` first.
2. Fallback to legacy scalar keys.
3. Normalize empty reason to `unknown` class semantics.

Pseudocode:

```go
func stopReasonOfTurn(t *turns.Turn) string {
    if t == nil {
        return ""
    }
    if ir, ok, err := turns.KeyTurnMetaInferenceResult.Get(t.Metadata); err == nil && ok {
        if ir.StopReason != nil {
            return strings.TrimSpace(*ir.StopReason)
        }
    }
    if reason, ok, err := turns.KeyTurnMetaStopReason.Get(t.Metadata); err == nil && ok {
        return strings.TrimSpace(reason)
    }
    return ""
}
```

## Detailed File-Level Task Breakdown

### Phase 0: Contract + keys + helper

1. `geppetto/pkg/inference/engine/results.go`
   - add `InferenceResult`, `FinishClass`, normalization helpers.
2. `geppetto/pkg/spec/geppetto_codegen.yaml`
   - add `inference_result` turn metadata key.
3. regenerate keys (`go generate` path used by turns codegen).
4. `geppetto/pkg/inference/engine/run_with_result.go`
   - add wrapper + optional interface + legacy synthesis helpers.

### Phase 1: Provider implementation parity

1. `geppetto/pkg/steps/ai/claude/engine_claude.go`
2. `geppetto/pkg/steps/ai/openai/engine_openai.go`
3. `geppetto/pkg/steps/ai/openai_responses/engine.go`
4. `geppetto/pkg/steps/ai/gemini/engine_gemini.go`

Implement canonical result write and legacy mirror write.

### Phase 2: Orchestrator adoption

1. `geppetto/pkg/inference/toolloop/loop.go`
2. `geppetto/pkg/inference/toolloop/enginebuilder/builder.go`
3. `geppetto/pkg/inference/session/session.go`

Use wrapper where practical in runner pathways while preserving existing interfaces.

### Phase 3: App consumer adoption

1. `temporal-relationships/internal/extractor/gorunner/run.go`
2. `temporal-relationships/internal/extractor/httpapi/server.go`

Switch stop/result reads to canonical-first strategy.

### Phase 4: Deprecation + cleanup

1. Replace direct scalar key reads/writes where redundant.
2. Add temporary metrics/log counters for legacy fallback hits.
3. remove fallback only after no hits across representative test runs.

## Testing Strategy

### Unit tests

1. Wrapper behavior
   - engine supports `EngineWithResult`
   - engine only supports legacy `RunInference`
   - canonical metadata exists vs missing
   - fallback synthesis behavior
2. Provider tests
   - each provider writes canonical result key
   - stop reason and usage parity between event metadata and canonical result
3. Consumer tests
   - gorunner `shouldStop` path using canonical result
   - fallback behavior when canonical missing

### Contract tests (critical)

Create one shared provider compliance suite asserting:

1. after successful inference, `KeyTurnMetaInferenceResult` exists.
2. if final event had stop reason, canonical stop reason exists.
3. if usage available in event metadata, canonical usage exists.

### Regression tests

1. max tokens continuation behavior still works with canonical reads.
2. no regressions for toolloop and JS adapter paths.

## Compatibility and Rollout Policy

### Rollout gates

1. Gate A: canonical key exists and is populated by all major providers.
2. Gate B: all known consumers canonical-first.
3. Gate C: legacy fallback hit rate near zero in integration tests.

### Fallback behavior

If canonical result missing, wrapper synthesizes from:

1. legacy scalar keys on turn metadata,
2. optionally event-derived hints passed via context (if available),
3. safe defaults (`FinishUnknown`, `Truncated=false`).

## Risks and Mitigations

1. Risk: drift between provider event metadata and canonical result.
   - Mitigation: shared helper to build both from one source object.
2. Risk: partial migration creates mixed semantics.
   - Mitigation: wrapper backfill + canonical-first read policy.
3. Risk: interface confusion (`RunInference` vs `RunInferenceWithResult`).
   - Mitigation: document strict rule that new consumers use wrapper unless hard requirement for raw call.
4. Risk: premature removal of legacy keys breaks external clients.
   - Mitigation: staged deprecation with usage telemetry and release notes.

## What Else Is Needed (Beyond Requested Bullets)

1. Documentation updates:
   - turns metadata docs (`pkg/doc/topics/08-turns.md`)
   - events docs (`pkg/doc/topics/04-events.md`) to define telemetry vs canonical durability.
2. Developer guidance:
   - short migration note for engine implementers and app consumers.
3. Observability:
   - temporary logs/metrics for canonical-missing fallback usage.
4. Serialization checks:
   - ensure YAML/JSON round-trip stability for `InferenceResult` in turn serde tests.
5. Tooling:
   - inventory script update to verify canonical key presence across engines.

## Suggested Delivery Sequence (small safe PRs)

1. PR-1: InferenceResult type + key + wrapper + tests.
2. PR-2: Provider parity updates + provider tests.
3. PR-3: Gorunner canonical-first consumption + stop-policy tests.
4. PR-4: Deprecation warnings + docs + optional legacy removal gates.

## Appendix: Minimal API sketch

```go
// geppetto/pkg/inference/engine/run_with_result.go
package engine

func RunInferenceWithResult(ctx context.Context, eng Engine, t *turns.Turn) (*turns.Turn, *InferenceResult, error)

// geppetto/pkg/inference/engine/results.go
func ExtractInferenceResult(t *turns.Turn) (InferenceResult, bool, error)
func SynthesizeInferenceResult(t *turns.Turn) InferenceResult
func MirrorLegacyInferenceKeys(t *turns.Turn, r InferenceResult) error
func InferFinishClass(stopReason string, hasToolCalls bool, err error) FinishClass
```

This gives a concrete implementation path aligned with your requested architecture while keeping existing runtime contracts stable during migration.
