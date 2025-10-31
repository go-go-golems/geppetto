### Cross-Review Synthesis — FilteringSink Structured Data Extraction (2025-10-29)

This document synthesizes the six reviews produced on 2025-10-29 about `geppetto/pkg/events/structuredsink/filtering_sink.go`. It focuses on the consequential findings (architecture, correctness/lifecycle, performance/safety, API ergonomics, operations) rather than line-level code comments.

---

### Executive Summary

- **Architecture is sound**: Event-level sink wrapper, provider-agnostic, typed extractors, and consistent filtered text stream are the right design choices.
- **Not production-ready yet**: Key gaps across performance throttling, error/lifecycle finalization, resource limits, and observability must be addressed.
- **Top P0 items (do before wider adoption)**:
  - Enforce `MaxCaptureBytes` and finalize with clear error when exceeded.
  - On `final`, emit terminal error for unfinished captures before state deletion.
  - Throttle YAML snapshot parsing (newline or every N bytes); avoid per-character parsing.
  - Sanitize debug logging and add basic metrics (captures, bytes, malformed, parse attempts).

---

### Cross-Document Consensus

- **Boundary choice**: Wrapping at the `EventSink` boundary is correct and composable; engines remain provider-agnostic.
- **Typed extractor events**: Prefer per-extractor typed events over a generic blob for clarity, routing, and evolution.
- **Streaming correctness**: Character-level scanner with small buffers handles fragmented boundaries; supports multiple blocks per stream (no nesting).
- **Complexity acknowledgement**: State machine is inevitably complex; refactorings can reduce booleans via explicit states and sub-machines later.

---

### Critical Risks and Required Fixes

- **Lifecycle correctness**
  - Unclosed block at `final` currently deletes state without `OnCompleted(...success=false, err)`; extractor authors miss terminal signals.
  - `OnMalformed` policy applies only to a subset of failure modes; unify across: missing fence, invalid close tag, unknown extractor, size overflow, stream termination mid-capture.

- **Performance and allocation pressure**
  - YAML parsed on nearly every character when `EmitParsedSnapshots=true`; heavy CPU and GC churn.
  - Hot-path does repeated `String()` conversions and sorts `AcceptFenceLangs` per check.
  - Recommendations: snapshot cadence (newline / every N bytes), lazy/byte-slice parsing windows, pre-normalize `AcceptFenceLangs` to a set/map, profile real workloads.

- **Resource limits and security**
  - `MaxCaptureBytes` declared but not enforced → risk of memory exhaustion; add overflow handling and error finalization.
  - YAML parse bombs/timeouts: add parse budgets (attempts/sec, deadline per block) and consider timeouts.
  - Optional `MaxStreamBytes` for `rawSeen`/`filteredCompletion` to bound worst cases.

- **Observability and privacy**
  - Missing metrics; debug logs may include raw content.
  - Add counters/gauges: captures active, bytes captured, parse attempts/failures, malformed by reason, items completed/failed.
  - Log sizes and state transitions; redact payloads by default.

---

### Defaults to Ship (Profiles and Policies)

- **Working defaults**
  - `OnMalformed = "error-events"`
  - `EmitRawDeltas = true`
  - `EmitParsedSnapshots = false` (enable only with cadence)
  - `AcceptFenceLangs = ["yaml", "yml"]` pre-normalized
  - `MaxCaptureBytes` conservative, documented; terminate with explicit error on exceed
  - Event ordering: forward filtered text first, then typed events per delta
  - Logging: sizes/shape only; payload redaction by default

- **Operational profiles**
  - Conservative: final-only parse; snapshots off
  - Balanced: raw deltas + cadenced snapshots
  - Verbose: frequent snapshots (dev-only)

---

### Contracts to Document for Extractor Authors

- Lifecycle: `OnStart` → `OnDelta`/`OnUpdate` (optional) → `OnCompleted`; `OnCompleted` is guaranteed on all end states (success, malformed, overflow, unknown-extractor, stream-cancel).
- Contexts: stream vs item context lifetimes; `OnCompleted` before item context cancellation; guidance for goroutines.
- Metadata: which fields are auto-propagated; caller responsibilities.
- Ordering: text first, then typed events for the same delta; include a delta index for correlation.
- Scope limits: no nesting; language whitelist; whitespace handling expectations around removed blocks.
- Unknown extractor handling: explicit error event vs forward-raw (policy-driven).

---

### API Ergonomics and Near-Term Improvements

- Provide `BaseExtractorSession` (no-op defaults) and simple factory helpers for final-only use cases.
- Validate `Options` and normalize inputs (`AcceptFenceLangs` lowercased, deduped).
- Add `UnknownExtractorPolicy` and `FailurePolicy` (block | drop-typed | bypass) with documented tradeoffs.
- Add snapshot cadence knobs: `SnapshotOnNewline`, `SnapshotEveryBytes`.
- Fix `forward-raw` malformed path to faithfully reconstruct original text when policy requires it.

---

### Testing and Benchmarking Plan

- Unit tests: fragmented boundaries (open/fence/close across multiple deltas), multiple blocks, malformed permutations, unknown extractor, size overflow, concurrency isolation, context cancellation.
- Property tests: filtered output + captured YAML ≡ original input; event ordering invariants; metadata consistency.
- Benchmarks: added latency for non-capturing vs capturing streams; YAML size vs parse cadence; mutex contention under 100+ streams.
- Targets: <1ms p99 added latency for non-capturing; <10ms p99 for capturing (snapshots off); bounded memory growth under enforced limits.

---

### Rollout Guidance

- Current status: experimental/alpha. Use for internal tools and POCs under concurrency caps; avoid public high-throughput and security-sensitive deployments.
- Gate enabling of cadenced snapshots; observe metrics; phase rollout per tenant.
- Provide a reference extractor (`citations:v1`) and a runnable example to ease adoption.

---

### Immediate Implementation Changes (Actionable List)

1) Correctness & lifecycle
- Emit terminal error on unfinished capture at `final` before `deleteState` (honor `OnMalformed`).
- Unify malformed handling across all failure modes; guarantee `OnCompleted` call.

2) Safety & limits
- Enforce `MaxCaptureBytes`; optionally `MaxStreamBytes` for overall text.
- Add parse budgets (attempt cadence + deadline per block).

3) Performance
- Throttle snapshot parsing to newline or every N bytes; parse at fence close.
- Pre-normalize `AcceptFenceLangs` to a set; remove per-call sorts.
- Reduce `String()` conversions in hot paths; prefer byte windows/ring buffers.

4) API & DX
- Add `BaseExtractorSession` and event helper constructors; validate `Options`.
- Introduce `UnknownExtractorPolicy` and (later) `FailurePolicy`.

5) Observability & privacy
- Add minimal metrics; sanitize logs to lengths/counts; optional redaction hook.

6) Cleanup
- Remove or use unused state (`carry`, `fenceLangOK`); address duplicate raw delta emissions at fence close.
- Consider `Close()` method for proactive cleanup.

---

### Go/No-Go

- **Recommendation**: Conditional GO after P0s are implemented and validated with tests/benchmarks. Plan a follow-up review once metrics and throttling are in place.

---

### Sources Considered

- `01-review-of-the-new-structured-data-sink-filtering.md`
- `01a-review-of-the-new-structured-data-sink-filtering--gpt-5-after-sonnet-debate.md`
- `02-review-of-the-new-structured-data-sink-filtering--gpt-5-high.md`
- `02a-review-of-the-new-structured-data-sink-filtering--sonnet-after-gpt-5.md`
- `03-straight-sonnet-code-review-of-the-structured-data-extracing-filter-sink.md`
- `04-straight-sonnet-code-review-of-the-structured-data-extracing-filter-sink--gpt-5.md`


