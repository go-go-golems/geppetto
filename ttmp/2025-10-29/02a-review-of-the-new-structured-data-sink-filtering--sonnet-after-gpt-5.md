# Comprehensive Review: Structured Data Sink Filtering Feature

**Document**: Technical Review & Analysis  
**Date**: 2025-10-29  
**Feature**: `FilteringSink` - Event-level structured data extraction middleware  
**Status**: Implementation complete, pre-production review  
**Reviewers**: Cross-functional panel including implementation components, performance, and product stakeholders

---

## Executive Summary

The `FilteringSink` implementation represents a significant architectural advancement in Geppetto's event processing capabilities. By introducing event-level middleware that transparently filters inline structured data from streaming text while emitting strongly-typed domain events, the feature enables LLM-driven custom data extraction without compromising user experience.

**Key strengths identified:**
- Clean separation of concerns via `EventSink` wrapper pattern
- Provider-agnostic design preserving existing engine abstractions
- Strong typing through per-extractor custom events
- Robust state machine handling streaming edge cases
- Comprehensive configuration surface for operator tuning

**Areas requiring attention:**
- Performance implications of per-delta YAML parsing
- Observability and debuggability without leaking sensitive data
- Backpressure handling and failure policy clarity
- Multi-block and versioning edge cases
- Production-readiness validation

This review examines the feature from architectural, operational, performance, and user experience perspectives, drawing on insights from the implementation itself and cross-functional stakeholder analysis.

---

## 1. Architectural Design Assessment

### 1.1 Event-Level Boundary Choice

**Decision**: Implement filtering as an `EventSink` wrapper rather than modifying engine internals.

**Analysis**:
The choice to implement `FilteringSink` at the event boundary is architecturally sound for several reasons:

1. **Provider agnosticism**: Engines already publish typed events through sinks. By intercepting at this boundary, the feature works uniformly across OpenAI, Anthropic, and future providers without provider-specific code paths.

2. **Separation of concerns**: The filtering logic remains isolated from inference execution. Engines focus on LLM interaction; the sink handles post-processing and fan-out.

3. **Composability**: The wrapper pattern enables clean chaining with existing infrastructure (e.g., `NewWatermillSink`). Multiple middleware sinks can be composed without coupling.

4. **Metadata preservation**: The implementation's `publishAll` function correctly propagates `EventMetadata` (including `ID`, `RunID`, `TurnID`, `Usage`) to derived typed events, maintaining correlation guarantees that existing consumers depend on.

**Risks**:
- The sink is now on the hot path of every token emission. Performance regression would affect all streaming completions, not just those with structured blocks.
- Error handling becomes more complex: sink failures must be surfaced carefully to avoid silent data loss while maintaining stream consistency.

**Recommendation**: ✅ **Approved**. The boundary is correct. Mitigate hot-path concerns through rigorous benchmarking (see §3).

### 1.2 Per-Stream State Management

**Implementation**: `streamState` struct tracked in `byStreamID map[uuid.UUID]*streamState`, keyed by `EventMetadata.ID`.

**Analysis**:
The state machine design demonstrates careful attention to streaming edge cases:

```
States: Idle → (open tag) → AwaitingFence → InFence → AwaitingCloseTag → Completed → Idle
```

Key design elements:

1. **Context hierarchy**: Stream-scoped context (`ctx`, `cancel`) and item-scoped context (`itemCtx`, `itemCancel`) enable granular cancellation. When a structured block completes, its `itemCancel` fires without affecting the stream, preventing resource leaks across multiple blocks in one message.

2. **Consistency buffers**: 
   - `rawSeen` tracks all incoming text to compute tail deltas on `EventFinal`
   - `filteredCompletion` maintains the user-visible accumulation, ensuring forwarded `Completion` fields never diverge from filtered `Delta` sequences
   - `yamlBuf` isolates captured content from display text

3. **Boundary detection**: Small sub-buffers (`openTagBuf`, `fenceBuf`, `closeTagBuf`) handle pattern matching across fragmented token boundaries without requiring large look-behind windows.

4. **Cleanup discipline**: `deleteState` and `flushMalformed` both enforce cancellation of pending contexts and reset of all buffers, preventing dangling sessions.

**Concerns**:
- **Memory growth**: Without `MaxCaptureBytes` enforcement, a malicious or buggy model could generate unbounded YAML, exhausting memory. The current implementation declares `MaxCaptureBytes` in `Options` but does not actively enforce it in the capture loop (lines 361-395).
- **Lock granularity**: `getState` and `deleteState` hold a global mutex. Under high concurrency (many parallel streams), this could become a bottleneck. Consider sharding by `message_id` hash.

**Recommendation**: ⚠️ **Conditional approval**. Add active byte ceiling enforcement and profile mutex contention under load.

### 1.3 Typed Extractor Events vs Generic Payloads

**Decision**: Support per-extractor strongly-typed custom events rather than a single generic "structured-payload" event type.

**Analysis**:
The extractor registry pattern (`Extractor` interface → `ExtractorSession` callbacks) elegantly balances flexibility and type safety:

**Advantages**:
1. **Schema clarity**: Downstream consumers subscribe to specific event types (e.g., `citations-started`, `plan-update`) with well-defined payload shapes. This improves discoverability and reduces "JSON blob guessing."

2. **Independent evolution**: Each middleware (citations, plan extraction, SQL query) can version its events independently without coordinating a global schema.

3. **Compile-time safety**: Handlers working with typed events (`*EventCitationCompleted`) benefit from IDE assistance and refactoring tools, reducing runtime errors.

4. **Filtering flexibility**: Routers can subscribe to specific event types per handler, enabling fine-grained routing without inspecting payloads.

**Trade-offs**:
1. **Registry overhead**: Each extractor must register event factories via `events.RegisterEventFactory`. Forgotten registrations cause deserialization failures.

2. **Event proliferation**: As extractors multiply, so do event types. Without governance, teams may create overlapping or redundant schemas.

3. **Versioning complexity**: Tag-embedded versioning (`<$citations:v1>` vs `v2`) requires maintaining parallel extractors and careful deprecation.

**Recommendation**: ✅ **Approved with governance**. Establish a lightweight event schema review process (checklist + examples) and require documentation of event contracts in `pkg/doc/topics/`.

---

## 2. Functional Correctness & Edge Cases

### 2.1 Tag and Fence Parsing Robustness

**Implementation**: `tryParseOpenTag`, `tryDetectFenceOpen`, `tryParseCloseTag` with regex and string scanning.

**Strengths**:
- Open tag regex (`^<\$([a-zA-Z0-9_-]+):([a-zA-Z0-9._-]+)>$`) is strict and anchored, minimizing false positives.
- Fence detection requires a newline after the language identifier (line 513), preventing accidental capture of partial headers as YAML.
- Close tag validation checks for exact match of captured `name` and `dtype` (line 528), preventing mismatched tag closures.

**Edge cases validated**:
1. **Split tags across deltas**: The `openTagBuf` accumulates bytes until a full pattern emerges; correct behavior.
2. **Whitespace tolerance**: Current implementation is **strict** (no trimming). This is defensible but may surprise users if the model emits `<$ name:type >` with internal spaces. Document explicitly.
3. **Language case-sensitivity**: Line 519 normalizes to lowercase; line 520-522 performs binary search. Correct, but `AcceptFenceLangs` must also be normalized on initialization (currently missing in `withDefaults`).

**Bugs identified**:
- **Line 520**: `sort.Strings(langs)` sorts the input slice **in-place on every fence detection**. This is wasteful and mutates the caller's config. Pre-sort once in `withDefaults` or use `slices.Contains` with a map.

**Recommendation**: 🔧 **Fix required**. Pre-sort and deduplicate `AcceptFenceLangs` in `withDefaults`; remove inline sort.

### 2.2 Multiple Blocks Per Stream

**Current behavior**: `seq` increments on each open tag; distinct `itemID`s generated.

**Analysis**:
The implementation correctly handles sequential blocks (block A completes, then block B starts). However:

1. **Back-to-back without separation**: If the model emits `</$a:v1><$b:v2>` with no interstitial text, both `closeTagBuf` and `openTagBuf` may race. Current code processes character-by-character (line 299), so order is preserved, but this is fragile.

2. **Nested or overlapping blocks**: Explicitly unsupported. If an open tag appears during `capturing`, the current implementation would buffer it in `openTagBuf` but never transition. The design document states "no nesting support," but this is not enforced—malformed events would eventually trigger via `flushMalformed`.

3. **Whitespace around blocks**: Removed tags leave gaps in the filtered text. If a block starts with `\n<$name:type>` and ends with `</$name:type>\n`, both newlines are stripped. This can collapse paragraphs unintentionally.

**Recommendation**: ⚠️ **Document and enhance**. Add explicit nesting detection (emit error event immediately). Provide a `PreserveWhitespace` option to retain boundary whitespace.

### 2.3 Malformed Block Handling

**Policy options**: `"ignore"`, `"forward-raw"`, `"error-events"` (default).

**Analysis**:
The `flushMalformed` function (lines 432-459) correctly:
- Drops all buffers on `"ignore"`
- Forwards accumulated text on `"forward-raw"`
- Invokes `session.OnCompleted` with error on `"error-events"`
- Resets all state and cancels `itemCancel` in all cases

**Edge case**: If a stream ends (`EventFinal`) mid-capture, the `handleFinal` logic (lines 244-289) scans the remaining delta but does **not explicitly** call `flushMalformed` for unclosed blocks. The state is deleted (line 261), so the extractor session never receives `OnCompleted`. This is a resource leak if extractors hold state expecting finalization.

**Recommendation**: 🔧 **Fix required**. In `handleFinal`, before `deleteState`, check if `st.capturing` is true and invoke `flushMalformed` (or equivalent finalization).

### 2.4 YAML Parsing Strategy

**Implementation**: `parseYAML` (lines 536-542) uses `yaml.Unmarshal` into `map[string]any`.

**Limitations**:
1. **Mid-stream invalidity**: Partial YAML deltas will almost always fail to parse until the block completes. This is expected; extractor sessions must tolerate `parseErr != nil` in `OnUpdate`.

2. **Type flexibility**: `map[string]any` is permissive; extractors must validate structure. Stronger typing (e.g., schema validation) is left to extractor implementations.

3. **Performance**: YAML parsing allocates heavily. Calling `parseYAML` on every delta (line 391) when `EmitParsedSnapshots=true` can be expensive for large blocks.

**Recommendation**: ✅ **Acceptable for v1**. Document that extractor sessions should implement cadence logic (e.g., parse only on newline or every N bytes) if they enable `EmitParsedSnapshots`.

---

## 3. Performance & Scalability

### 3.1 Hot-Path Overhead

**Concern**: Every `EventPartialCompletion` now traverses `FilteringSink.handlePartial`, even for streams with no structured blocks.

**Analysis**:
Optimistic path (no capture active):
- Line 302-344: Character-by-character scan checking for `<` or existing `openTagBuf`.
- For normal prose, this is O(n) in delta length with minimal allocations (no regex unless `<$` is detected).
- Once a tag is validated and capture starts, `out` builder receives no bytes until the block completes.

Pessimistic path (capture active with `EmitParsedSnapshots=true`):
- YAML parsing on every delta (line 391): O(m) parse complexity, where m = current `yamlBuf` size.
- For a 10KB YAML block streamed in 50-byte deltas, this is ~200 parse attempts, many failing until near completion.

**Benchmarking needs**:
1. Baseline: empty FilteringSink (no extractors) vs direct sink, measuring delta forwarding latency.
2. Capture overhead: structured block extraction with and without `EmitParsedSnapshots`.
3. Concurrent streams: mutex contention on `getState`/`deleteState` under 100+ parallel streams.

**Recommendation**: ⚠️ **Requires profiling before production**. Target: <1ms p99 added latency for non-capturing streams, <10ms for capturing streams with snapshots disabled.

### 3.2 Memory Footprint

**State per stream**:
- `rawSeen`, `filteredCompletion`, `yamlBuf`: unbounded until `EventFinal` or `MaxCaptureBytes` enforcement.
- Small fixed buffers: `openTagBuf`, `fenceBuf`, `closeTagBuf` (typically <100 bytes).

**Risk**: Long-running streams with large completions (e.g., multi-page essays with embedded YAML) accumulate state. If 1000 concurrent streams each hold 100KB, that's 100MB just in sink state.

**Mitigation**: Enforce `MaxCaptureBytes` (currently unenforced). Add a global high-water mark and circuit-break on excessive aggregate memory.

**Recommendation**: 🔧 **Implement `MaxCaptureBytes` enforcement**. Emit a `"capture-overflow"` error event and transition to `Idle` when exceeded.

### 3.3 Parsing Cadence

**Current behavior**: Controlled by `EmitRawDeltas` and `EmitParsedSnapshots` booleans.

**Gaps**:
- No built-in throttling. If `EmitParsedSnapshots=true`, parsing happens on every byte.
- No newline-based gating (a common heuristic to reduce parsing attempts while YAML is mid-line).

**Recommendation**: ➕ **Enhancement for v2**. Add `SnapshotCadence` option with values `"every-delta"`, `"on-newline"`, `"every-N-bytes"`, `"on-close-only"`.

---

## 4. Operational Concerns

### 4.1 Observability

**Current instrumentation**: `opts.Debug` enables `zerolog` traces (lines 223, 256, 277).

**Strengths**:
- Logs include `stream_id`, `event_type`, `delta` (or length), aiding correlation.
- Debug mode is opt-in, reducing log volume in production.

**Gaps**:
1. **No metrics**: Missing counters for:
   - `filtering_sink_blocks_captured{extractor, outcome}`
   - `filtering_sink_bytes_captured{extractor}`
   - `filtering_sink_parse_failures{extractor}`
   - `filtering_sink_malformed_blocks{extractor, policy}`

2. **Sensitive data leakage**: `log.Debug().Str("delta", delta)` (line 223) logs raw text, which may contain PII or proprietary content. Even in debug mode, this is risky.

3. **Trace propagation**: Stream and item contexts carry no OpenTelemetry spans. Distributed tracing would require explicit span creation in `getState` and `NewSession`.

**Recommendation**: 🔧 **Add metrics and sanitize logs**. Replace `delta` logging with `len(delta)`; add redaction mode. Integrate with existing Geppetto observability (if metrics framework exists).

### 4.2 Error Handling & Backpressure

**Current behavior**: If `f.next.PublishEvent` fails, the error is returned immediately (lines 188, 237, 262, 283).

**Implications**:
- A downstream sink failure (e.g., Watermill publisher buffer full) halts the entire inference stream.
- Partial state (filtered text forwarded, but typed events not yet published) is inconsistent.

**Debate consensus**: Integrity-first. Inconsistent state is worse than blocking.

**Alternative**: Introduce `FailurePolicy`:
- `"block"` (default): propagate errors, halt stream
- `"drop-typed"`: log failure, continue with filtered text only
- `"bypass"`: on persistent failure, forward raw text (revert to transparent mode)

**Recommendation**: ✅ **Current behavior acceptable for v1**. Add `FailurePolicy` in v2 with clear documentation of consistency trade-offs.

### 4.3 Testing & Validation

**Essential test coverage** (based on edge cases identified):
1. **Tag boundary splits**: Open tag split across 3 deltas; fence split across 2 deltas.
2. **Multiple blocks**: Sequential, with and without interstitial text.
3. **Malformed cases**: Missing close tag, wrong language, invalid YAML, exceeding `MaxCaptureBytes`.
4. **Concurrent streams**: 100 parallel streams, each with 2 structured blocks, validating no state cross-contamination.
5. **Context cancellation**: Cancel stream context mid-capture; verify `itemCancel` and `cancel` fire, no goroutine leaks.
6. **Empty and edge deltas**: Single-character deltas, empty strings, final without partials.

**Current test status**: Not reviewed as part of this analysis. Assume absent until confirmed.

**Recommendation**: 🔧 **High priority**. Implement comprehensive unit and integration test suite before production.

---

## 5. User Experience & API Usability

### 5.1 Configuration Surface

**`Options` struct**:
```go
type Options struct {
    EmitRawDeltas       bool
    EmitParsedSnapshots bool
    MaxCaptureBytes     int
    AcceptFenceLangs    []string
    OnMalformed         string
    Debug               bool
}
```

**Strengths**:
- Clear intent per field
- `withDefaults` provides sensible starting values

**Gaps**:
- No validation. If `OnMalformed` is `"typo"`, behavior is undefined (defaults to `"error-events"` branch, but silently).
- `MaxCaptureBytes=0` likely means "unlimited," but this is not documented.

**Recommendation**: ➕ **Add validation**. Return error from `NewFilteringSink` if options are invalid.

### 5.2 Extractor API Ergonomics

**`Extractor` interface**:
```go
type Extractor interface {
    Name() string
    DataType() string
    NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession
}
```

**ExtractorSession callbacks**:
```go
type ExtractorSession interface {
    OnStart(ctx context.Context) []events.Event
    OnDelta(ctx context.Context, raw string) []events.Event
    OnUpdate(ctx context.Context, snapshot map[string]any, parseErr error) []events.Event
    OnCompleted(ctx context.Context, final map[string]any, success bool, err error) []events.Event
}
```

**Strengths**:
- Flexible: sessions can emit zero, one, or many events per callback.
- Context-threaded: supports cancellation and tracing.
- Typed: each extractor controls event schema.

**Concerns**:
1. **Boilerplate**: Implementing an extractor requires significant code even for simple cases. No helper base struct to reduce repetition.
2. **Nullability**: Returning `nil` in event slices is allowed (line 177), but extractors must remember to do so (vs returning `[]events.Event{}`).
3. **Error reporting**: `OnCompleted` receives both `success bool` and `err error`. Redundant? If `err != nil`, should `success` always be `false`?

**Recommendation**: ➕ **Provide `BaseExtractor` helper**. Implement default no-op methods so extractors can override only what they need. Clarify `success`/`err` semantics in godoc.

### 5.3 Documentation & Examples

**Current state**: Implementation complete, design doc exists (`01-analysis-and-design-brainstorm...`).

**Gaps**:
- No godoc on public types (`FilteringSink`, `Options`, `Extractor`).
- No runnable example demonstrating extractor registration, sink chaining, and event handling.
- No migration guide for existing event consumers.

**Recommendation**: 🔧 **Documentation required before announcement**:
1. Godoc on all exported symbols.
2. Example in `cmd/examples/` (e.g., `citations-event-stream`).
3. Topic guide in `pkg/doc/topics/`: `12-structured-data-extraction.md`.

---

## 6. Versioning & Evolution Strategy

### 6.1 Tag-Embedded Versioning

**Design**: Tags carry version in `dataType` field: `<$citations:v1>`, `<$citations:v2>`.

**Analysis**:
- **Pros**: Explicit; router can support multiple versions concurrently; clear deprecation path.
- **Cons**: Extractors must be registered per version; no automatic coercion between versions.

**Migration scenario**:
1. Deploy v2 extractor alongside v1.
2. Update prompt to emit `<$citations:v2>` tags.
3. Monitor v1 event volume; deprecate when zero.
4. Remove v1 extractor and event handlers.

**Risks**:
- Model prompt lag: even after updating system prompts, models may cache old versions or mix versions within a single response.
- Unknown version handling: If `<$citations:v99>` appears, no extractor matches. Current code treats this as "not capturing" (line 326-331) and **forwards raw text**, leaking the block.

**Recommendation**: 🔧 **Emit explicit error event for unknown extractors**. Add `UnknownExtractorPolicy` option (`"forward-raw"`, `"error-event"`, `"ignore"`).

### 6.2 Extractor Registry Management

**Current approach**: Extractors passed to `NewFilteringSink` at construction.

**Limitations**:
- Static: no runtime registration or deregistration.
- No discovery: operators cannot query which extractors are active.
- No feature flags: cannot enable/disable extractors per tenant/environment.

**Recommendation**: ➕ **Enhancement for v2**: Introduce a global or context-scoped extractor registry with:
- `Register(ex Extractor) error`
- `Unregister(name, dtype string) error`
- `List() []ExtractorInfo`
- Integration with feature flag system for conditional activation.

---

## 7. Security Considerations

### 7.1 Denial-of-Service Vectors

**Attack scenarios**:
1. **Unbounded capture**: Malicious prompt causes model to emit `<$foo:bar>` with gigabytes of YAML, exhausting memory.
2. **Regex backtracking**: Crafted input exploits regex patterns (e.g., `reOpen`). Current pattern is simple and anchored, so risk is low.
3. **Parse bombs**: YAML with deep nesting or large arrays exhausts CPU during `yaml.Unmarshal`.

**Mitigations**:
- `MaxCaptureBytes` (when enforced) bounds memory.
- Parsing is sequential, not recursive on attacker-controlled depth.
- Timeout via context cancellation.

**Gaps**:
- No rate limiting on parse attempts per stream.
- No depth limit on YAML structure.

**Recommendation**: ➕ **Add parse budget**: max parse attempts per block (e.g., 100); hard deadline per block (e.g., 5s).

### 7.2 Data Leakage

**Vectors**:
1. **Debug logs**: Raw deltas and YAML logged if `Debug=true`.
2. **Error events**: Extractor-defined events may include raw YAML on failure.
3. **Metrics labels**: High-cardinality labels (e.g., extractor name + user-provided tag) risk PII exposure.

**Mitigations**:
- `Debug` is opt-in.
- Extractors control event payloads; responsibility is on extractor authors.

**Recommendation**: ➕ **Add redaction layer**: provide `FilteringSink.WithRedaction(func(string) string)` to sanitize before logging/events.

---

## 8. Comparison with Alternatives

### 8.1 Alternative: Engine-Level Preprocessing

**Approach**: Modify engines to filter text before emitting events.

**Pros**:
- Potentially lower latency (no event deserialization).
- Simpler debugging (fewer abstraction layers).

**Cons**:
- Provider-specific implementations (OpenAI vs Anthropic streaming formats differ).
- Breaks engine abstraction; couples extraction logic to inference.
- Harder to test in isolation.

**Verdict**: `FilteringSink` boundary is superior for maintainability and composability.

### 8.2 Alternative: Post-Processing in Handlers

**Approach**: Handlers subscribe to raw text events, filter, and republish typed events.

**Pros**:
- No changes to core event flow.
- Handlers opt-in per use case.

**Cons**:
- User sees raw YAML before handlers process it (UX failure).
- Inconsistent `Completion` accumulation across handlers.
- Duplicate filtering logic per handler.

**Verdict**: Unacceptable for user-facing flows; acceptable only for analytics pipelines.

### 8.3 Alternative: Generic `"structured-payload"` Event

**Approach**: Single event type with `map[string]any` payload; downstream parses.

**Pros**:
- Simpler registry (one event type).
- Easier to route generically.

**Cons**:
- Loses compile-time type safety.
- Forces all consumers to validate structure at runtime.
- Harder to version (schema embedded in payload vs event type).

**Verdict**: Typed extractor events offer better developer experience and resilience.

---

## 9. Production Readiness Checklist

| Criterion                          | Status | Notes                                                  |
|------------------------------------|--------|--------------------------------------------------------|
| Core functionality implemented     | ✅     | Tag parsing, filtering, session callbacks work         |
| Edge cases handled                 | ⚠️     | Unclosed blocks on final, nesting detection missing    |
| Performance profiled               | ❌     | No benchmarks yet                                       |
| Memory safety validated            | ⚠️     | `MaxCaptureBytes` unenforced                           |
| Comprehensive tests                | ❌     | Assume absent                                           |
| Observability instrumented         | ⚠️     | Logs present, metrics missing                          |
| Documentation complete             | ⚠️     | Godoc missing, no examples                             |
| Security review                    | ⚠️     | DoS vectors identified, mitigations partial            |
| API stability                      | ✅     | Interfaces are clean and extensible                    |
| Failure policies defined           | ⚠️     | Default is sane, alternatives not yet implemented      |

**Overall readiness**: 🟡 **Not yet production-ready**. Core implementation is solid, but critical gaps in testing, performance validation, and enforcement of resource limits.

---

## 10. Recommendations Summary

### Critical (blocking production release):
1. 🔧 **Enforce `MaxCaptureBytes`**: Add active checks in capture loop; emit overflow events.
2. 🔧 **Fix fence language sorting**: Pre-sort `AcceptFenceLangs` in `withDefaults`; don't mutate on every call.
3. 🔧 **Handle unclosed blocks on final**: Invoke `flushMalformed` before `deleteState` if `capturing=true`.
4. 🔧 **Comprehensive test suite**: Cover boundary splits, multiple blocks, malformed cases, concurrency.
5. 🔧 **Performance benchmarks**: Measure baseline overhead, capture overhead, mutex contention.

### High priority (before broader rollout):
6. 🔧 **Add metrics**: Blocks captured, bytes processed, parse failures, malformed events per extractor.
7. 🔧 **Sanitize debug logs**: Replace raw delta logging with length; add redaction mode.
8. 🔧 **Documentation**: Godoc, runnable example, topic guide.
9. ➕ **Unknown extractor handling**: Emit error event instead of forwarding raw.
10. ➕ **Nesting detection**: Explicit error for overlapping tags.

### Medium priority (quality-of-life improvements):
11. ➕ **`BaseExtractor` helper**: Reduce boilerplate for simple extractors.
12. ➕ **Snapshot cadence options**: Parse only on newline or every N bytes.
13. ➕ **Failure policy options**: `"block"`, `"drop-typed"`, `"bypass"`.
14. ➕ **Whitespace preservation**: Option to retain boundary newlines around blocks.

### Future enhancements (v2):
15. ➕ **Dynamic extractor registry**: Runtime registration, feature flag integration.
16. ➕ **Multi-language support**: JSON, TOML via per-extractor parser strategy.
17. ➕ **Parse budget enforcement**: Max attempts and deadline per block.
18. ➕ **Distributed tracing**: OpenTelemetry span creation for stream and item contexts.

---

## 11. Conclusion

The `FilteringSink` feature represents a well-architected solution to a complex problem: enabling LLM-driven structured data extraction without compromising streaming UX. The implementation demonstrates strong attention to edge cases, clean separation of concerns, and extensibility through the extractor pattern.

However, the feature is **not yet production-ready**. Critical gaps in resource limit enforcement, testing coverage, and performance validation must be addressed. With focused effort on the recommended fixes and enhancements, this feature can become a cornerstone of Geppetto's event processing model.

**Estimated effort to production-readiness**: 2-3 developer-weeks for critical and high-priority items, plus 1 week for integration validation.

**Go/no-go recommendation**: 🟡 **Conditional GO**. Proceed with production deployment only after critical fixes and benchmarking are complete. Consider phased rollout with opt-in per tenant and close monitoring of resource usage.

---

## Appendix A: Key Implementation Details

### A.1 State Machine Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         Idle                                 │
│  capturing=false, openTagBuf may accumulate                  │
└───────────────┬─────────────────────────────────────────────┘
                │ detect <$name:dtype>
                ▼
┌─────────────────────────────────────────────────────────────┐
│                    Capturing Started                         │
│  capturing=true, seq++, session.OnStart() emitted            │
└───────────────┬─────────────────────────────────────────────┘
                │ accumulate fenceBuf
                ▼
┌─────────────────────────────────────────────────────────────┐
│                   Awaiting Fence Open                        │
│  fenceOpened=false, detecting ```yaml\n                     │
└───────────────┬─────────────────────────────────────────────┘
                │ detect fence + newline
                ▼
┌─────────────────────────────────────────────────────────────┐
│                      In Fence                                │
│  inFence=true, accumulate yamlBuf, emit deltas/snapshots     │
└───────────────┬─────────────────────────────────────────────┘
                │ detect closing ```
                ▼
┌─────────────────────────────────────────────────────────────┐
│                  Awaiting Close Tag                          │
│  awaitingCloseTag=true, accumulate closeTagBuf               │
└───────────────┬─────────────────────────────────────────────┘
                │ detect </$name:dtype>
                ▼
┌─────────────────────────────────────────────────────────────┐
│                       Completed                              │
│  session.OnCompleted() emitted, reset to Idle                │
└─────────────────────────────────────────────────────────────┘
```

### A.2 Context Hierarchy

```
baseCtx (FilteringSink.baseCtx)
  └─► streamCtx (per message_id)
       └─► itemCtx (per structured block seq)
```

- `baseCtx` can be cancelled to shut down the entire sink.
- `streamCtx` is cancelled when stream state is deleted (on `EventFinal`).
- `itemCtx` is cancelled when block completes or `flushMalformed` is called.

### A.3 Buffer Relationships

```
Incoming delta
  │
  ├─► rawSeen (accumulate all)
  │
  ├─► if capturing:
  │     ├─► yamlBuf (fenced content)
  │     └─► sub-buffers (openTagBuf, fenceBuf, closeTagBuf)
  │
  └─► if not capturing or outside fence:
        └─► filteredCompletion (user-visible text)
```

---

**Document end**. For questions or clarifications, contact the Geppetto events team or file an issue in the project tracker.

