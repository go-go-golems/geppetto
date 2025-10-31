# Code Review: FilteringSink Structured Data Extraction

**Date:** 2025-10-29  
**Reviewer:** Claude Sonnet 4.5  
**Files Reviewed:**
- `geppetto/pkg/events/structuredsink/filtering_sink.go`
- `geppetto/ttmp/2025-10-27/01-analysis-and-design-brainstorm-for-a-streaming-middleware-system-and-structured-data-extraction-streaming-design.md`

## Executive Summary

The implementation faithfully realizes the design with a character-by-character streaming parser that filters structured data blocks from LLM output while emitting typed events. The code is well-structured and handles the core requirements, but several areas merit attention: performance characteristics, edge-case robustness, context lifecycle management, and error recovery.

**Overall Assessment:** ✓ Good implementation with refinement opportunities  
**Critical Issues:** None  
**Recommendations:** 5 medium-priority improvements

---

## 1. Design Adherence

### ✓ Implemented as Designed

- **EventSink wrapper pattern:** Clean composition with `FilteringSink` implementing `events.EventSink`
- **Extractor registry:** Key-based lookup via `extractorKey(name, dtype)`
- **Per-stream state:** Keyed by `EventMetadata.ID` with proper lifecycle (create/delete)
- **Session delegation:** Strong typed events via `ExtractorSession` callbacks
- **Options knobs:** `EmitRawDeltas`, `EmitParsedSnapshots`, `OnMalformed`, etc.
- **Context support:** Base context with per-stream and per-item cancellation

### Δ Deviations from Design

1. **No `carryOver` buffer:**  
   The design proposed a small tail buffer for cross-delta pattern detection. Implementation uses incremental state buffers (`openTagBuf`, `fenceBuf`, `closeTagBuf`) instead, which is actually **more precise** but potentially more complex to reason about.

2. **Character-level scanning vs. regex-first:**  
   Design implied regex-based detection; implementation uses character-by-character state machine with regex only for validation. This is **more robust** for streaming but has different performance characteristics.

3. **Fence detection requires newline:**  
   Line 513 requires `\n` after fence header before capturing begins. Design didn't specify this; it's a **good safeguard** to avoid capturing the fence header itself.

---

## 2. Code Quality & Correctness

### Strong Points

1. **Concurrency safety:** Proper mutex around `byStreamID` map access (lines 154-163, 165-173)
2. **Context cancellation:** Per-stream and per-item contexts with cleanup (lines 159, 169-170, 322-323, 410)
3. **Metadata propagation:** Careful preservation of `ID`, `RunID`, `TurnID` in published events (lines 182-186)
4. **Clean separation:** Extractor sessions fully own event types and emission logic
5. **Defensive nil checks:** Line 177 skips nil events; line 84 guards nil context

### Issues & Risks

#### Medium Priority

**M1: Race Condition in `publishAll`**  
Lines 175-193: `publishAll` is called without holding `f.mu`, but it accesses `f.next` which could theoretically be modified during sink reconfiguration. If `FilteringSink` is immutable after construction (which appears to be the intent), this is safe. **Recommendation:** Document immutability contract or add `RLock` if reconfiguration is planned.

**M2: Unbounded Buffer Growth**  
Lines 223, 229, 258, 279: `rawSeen` and `filteredCompletion` grow without limit until stream ends. A malicious or buggy stream could cause memory exhaustion.  
**Recommendation:** Honor `MaxCaptureBytes` (currently unused in code) or add a separate `MaxStreamBytes` option and return an error event when exceeded.

**M3: State Leaks on Panic**  
If an extractor panics in `OnStart`/`OnDelta`/`OnUpdate`/`OnCompleted`, the state machine is left in an inconsistent state (captured=true but session=nil).  
**Recommendation:** Wrap session calls in `defer` + recover, emit error event, and reset state via `flushMalformed`.

**M4: Regex Compilation per Call**  
Line 485: `reOpen` is compiled as a package-level `var`, but `tryDetectFenceOpen` (lines 520-521) calls `sort.Strings` on `langs` slice **on every invocation**. If `AcceptFenceLangs` has many entries, this is wasteful.  
**Recommendation:** Pre-sort `opts.AcceptFenceLangs` in `withDefaults()` and document that callers must not mutate it.

**M5: Close Tag Detection is Fragile**  
Lines 525-533: `tryParseCloseTag` uses string suffix matching. If the model emits text like "That's all </$foo:bar>end", the close tag is detected mid-sentence.  
**Current Behavior:** The state machine expects the close tag immediately after the closing fence, so this is likely rare in practice.  
**Recommendation:** Add a state flag to ensure close tag is only parsed after fence close, not arbitrary text. (This may already be enforced by `awaitingCloseTag`; verify with tests.)

#### Low Priority

**L1: Strconv Reimplementation**  
Lines 466-482: Custom `itoa`/`strconv` to avoid importing `fmt`. This is clever but `strconv.Itoa` from stdlib is 2 lines and well-tested.  
**Recommendation:** Use `strconv.Itoa` unless there's a strong reason (e.g., binary size constraints in WASM builds).

**L2: Debug Logging Inside Hot Path**  
Lines 223, 256, 277: `log.Debug()` calls are gated by `f.opts.Debug`, but the string formatting (`.Str("delta", delta)`) happens unconditionally.  
**Recommendation:** Use `if f.opts.Debug { log.Debug()... }` to skip allocation entirely when disabled.

**L3: Unused `MaxCaptureBytes`**  
Line 21 defines it, but it's never checked. `yamlBuf` can grow without bound.  
**Recommendation:** In line 363 (inside fence), check `st.yamlBuf.Len() >= f.opts.MaxCaptureBytes` and call `flushMalformed` with a "too large" error.

---

## 3. Edge Cases & Robustness

### Well Handled

- ✓ **Split tags across deltas:** Incremental buffers detect patterns even when fragmented
- ✓ **Multiple blocks per stream:** `seq` counter and `itemID` generation support sequential captures
- ✓ **Malformed YAML:** `parseYAML` returns error; sessions decide how to surface it
- ✓ **Final event types:** Handles both `EventFinal` and legacy `EventImpl.ToText()` (lines 245-263, 266-285)
- ✓ **Empty YAML:** Line 538 rejects empty strings explicitly

### Gaps

**E1: Nested Tags**  
If model outputs `<$a:1> <$b:2> ... </$b:2> </$a:1>`, inner tag is treated as YAML content of outer tag. Design explicitly scoped this out ("no nesting support"), but it should be **documented** in godoc.

**E2: Unclosed Tag at Final**  
Lines 244-289: `handleFinal` processes remaining delta but doesn't explicitly check if `st.capturing == true` at the end. If a tag is left open, `filteredCompletion` may include partial tags.  
**Recommendation:** After line 259 and 280, add:
```go
if st.capturing {
    _ = f.publishAll(st.ctx, meta, flushMalformedAndReturn(f, meta, st))
}
```

**E3: Context Cancellation Mid-Stream**  
If `st.ctx` or `st.itemCtx` is cancelled while inside `scanAndFilter`, the extractor sessions may return events that are never published (line 241 could fail, but errors are only logged implicitly by the next sink).  
**Recommendation:** Check `ctx.Err()` before calling session callbacks; skip callbacks if context is done.

**E4: Fence Language Case Sensitivity**  
Line 519: `strings.ToLower(header)` normalizes language, but line 520-522 requires exact match in sorted slice. If `AcceptFenceLangs` contains `["YAML", "yml"]`, neither will match because `withDefaults()` provides `["yaml", "yml"]`.  
**Mitigation:** Line 30-31 hardcodes lowercase defaults, so this is only an issue if callers override.  
**Recommendation:** Document that `AcceptFenceLangs` must be lowercase, or normalize it in `withDefaults()`.

---

## 4. Performance Characteristics

### Complexity

- **Time:** O(n) per character in the stream, where n = total text length. Each character is processed once (line 299 loop).
- **Space:** O(m) where m = size of captured YAML + size of filtered completion. Bounded only by stream length in current implementation.
- **Locking:** Minimal; mutex only held during state lookup/creation/deletion (lines 154-163).

### Optimizations Done Right

- **Incremental parsing:** YAML parsing attempts are optional (lines 375-378, 390-392) and controlled by `EmitParsedSnapshots`
- **Lazy string building:** `strings.Builder` used throughout to avoid repeated allocations
- **Pass-through fast path:** Non-text events skip all processing (lines 143-150)

### Potential Bottlenecks

1. **YAML parsing on every character (when enabled):**  
   Lines 390-392: If `EmitParsedSnapshots` is true, YAML is parsed **on every character** inside the fence. For a 10KB YAML block, this is 10K parse attempts.  
   **Recommendation:** Parse only on newline boundaries (`ch == '\n'`) or every N bytes (e.g., 1024).

2. **Regex in `tryParseOpenTag`:**  
   Line 493: `reOpen.FindStringSubmatch` is called for every `>` character when `openTagBuf` has content.  
   **Mitigation:** Line 492 short-circuits if string doesn't end with `>`.  
   **Recommendation:** Add early exit if `st.openTagBuf.Len() < 7` (minimum valid tag is `<$a:b>`).

3. **Event allocation per delta:**  
   Lines 232-236: A new `EventPartialCompletion` is allocated for every incoming partial, even if `filteredDelta` is empty.  
   **Recommendation:** Skip publishing if `filteredDelta == "" && st.filteredCompletion.Len() == 0` on the first partial (to avoid empty events).

---

## 5. API Design & Usability

### Strengths

- **Clear constructor:** `NewFilteringSink` and `NewFilteringSinkWithContext` with explicit extractor list
- **Options struct:** Forward-compatible; easy to add fields without breaking callers
- **Session interface:** Flexible enough for diverse use cases (deltas, snapshots, completion)
- **Context-aware:** Per-item contexts enable cancellation of long-running extractor logic

### Weaknesses

**A1: No Public `Close()` Method**  
If the sink is used in a long-lived process with ephemeral streams, there's no way to force cleanup of `byStreamID`. While `deleteState` is called on final, a buggy engine that never emits final would leak memory.  
**Recommendation:** Add `Close()` method that cancels all stream contexts and clears the map.

**A2: Extractor Errors are Swallowed**  
Lines 175-193: If an extractor returns events that fail to publish, the error is returned from `PublishEvent`, but the state machine continues. The stream may be left in `capturing=true` forever.  
**Recommendation:** On publish error, call `flushMalformed` for the current item and log the error.

**A3: No Metrics/Observability**  
There's no way to monitor how many blocks are captured, how many parse failures occur, or stream lifecycle without adding logging.  
**Recommendation:** Add optional `Metrics` interface in `Options` with callbacks like `OnCaptureStart`, `OnCaptureComplete`, `OnParseError`.

**A4: `OnMalformed` is Stringly Typed**  
Line 23: `string` type with magic values ("ignore", "forward-raw", "error-events").  
**Recommendation:** Define `type MalformedPolicy int` with constants `MalformedIgnore`, `MalformedForwardRaw`, `MalformedErrorEvents`.

---

## 6. Security & Safety

### Threats Mitigated

- ✓ **Injection via tag names:** Regex limits chars to `[a-zA-Z0-9_-]` (line 485)
- ✓ **Fence language spoofing:** Validated against whitelist (lines 506-523)
- ✓ **Context leaks:** Cancelled on stream end (lines 169-171)

### Threats Not Addressed

**S1: Denial of Service via Large Captures**  
`MaxCaptureBytes` is defined but not enforced. A model outputting 1GB of YAML will consume 1GB of memory.  
**Severity:** Medium (requires malicious or broken model)  
**Recommendation:** Enforce limit in line 363; emit error event and reset state.

**S2: ReDoS via Open Tag Regex**  
Line 485: `reOpen` uses `^<\$([a-zA-Z0-9_-]+):([a-zA-Z0-9._-]+)>$`. The `+` quantifiers are safe because character classes are deterministic, but if regex is changed to support more complex patterns, ReDoS becomes a risk.  
**Recommendation:** Document that regex must avoid backtracking; consider using `regexp2` or a hand-written parser for tags.

**S3: Logging Delta Content**  
Line 223: `Str("delta", delta)` logs user content, which may contain PII.  
**Recommendation:** Truncate delta to first 50 chars or hash it: `.Str("delta", truncate(delta, 50))`.

---

## 7. Testing Gaps

The implementation would benefit from tests covering:

1. **Fragmented patterns:** Tag/fence/close split across 3+ deltas with 1 char each
2. **Interleaved captures:** Multiple `<$a:1>...</$a:1> <$b:2>...</$b:2>` in one stream
3. **Malformed close tag:** `</$wrong:type>` when expecting `</$foo:bar>`
4. **Context cancellation:** Cancel `baseCtx` mid-stream; verify sessions are stopped
5. **Extractor panic:** Session panics in `OnDelta`; verify state is cleaned up
6. **MaxCaptureBytes enforcement:** Once implemented, test truncation behavior
7. **Large deltas:** Single partial with 1MB of text containing multiple blocks
8. **Empty/whitespace-only fence content:** `<$a:1>\n```yaml\n   \n```\n</$a:1>`
9. **Fence without newline:** `<$a:1>```yamlkey:value` (should not capture until `\n` per line 513)
10. **Concurrent streams:** Publish partials for 10 different `message_id`s in parallel goroutines

---

## 8. Recommendations Summary

### Critical (Do Before Production)
None identified.

### High Priority
1. **Enforce `MaxCaptureBytes`** to prevent memory exhaustion (S1, L3)
2. **Throttle YAML parsing** to newline/chunk boundaries when `EmitParsedSnapshots=true` (performance bottleneck #1)
3. **Handle unclosed tags at final** explicitly via `flushMalformed` (E2)

### Medium Priority
4. **Add `Close()` method** for sink lifecycle management (A1)
5. **Pre-sort `AcceptFenceLangs`** in `withDefaults()` (M4)
6. **Wrap extractor calls in panic recovery** (M3)
7. **Add `MaxStreamBytes` option** and enforce on `rawSeen`/`filteredCompletion` (M2)

### Low Priority
8. Use `strconv.Itoa` instead of custom implementation (L1)
9. Gate debug logging to skip formatting (L2)
10. Define `MalformedPolicy` enum (A4)
11. Add metrics/observability hooks (A3)
12. Truncate logged deltas (S3)

---

## 9. Architectural Notes

### What This Design Does Well

1. **Separation of concerns:** Filtering logic is independent of extractor domain logic
2. **Type safety:** Strong typed events end-to-end (no `map[string]any` in the sink API)
3. **Composability:** Sink wrapping enables middleware chains
4. **Provider agnostic:** Works with any engine emitting standard events

### Potential Future Extensions

1. **Middleware chain:** `type EventMiddleware func(EventHandlerFunc) EventHandlerFunc` for composable transforms
2. **Streaming validators:** Validate YAML against JSON Schema as it's captured
3. **Compression:** Store `yamlBuf` compressed if `MaxCaptureBytes` is large
4. **Checkpoint/resume:** Serialize `streamState` to survive restarts (for long-running conversations)
5. **Multiple downstream sinks:** Broadcast events to multiple sinks (currently only one `next`)

---

## 10. Conclusion

The `FilteringSink` implementation is a solid realization of the design with good attention to streaming correctness and extractor flexibility. The character-level state machine is more complex than initially envisioned but handles fragmentation robustly. Main areas for improvement are performance (YAML parsing frequency, buffer growth limits) and error recovery (panic handling, unclosed tags).

**Recommendation:** Address high-priority items (especially `MaxCaptureBytes` enforcement and parsing throttle) before using with untrusted models or large payloads. Add comprehensive tests for edge cases. Consider metrics hooks for production observability.

**Code Quality Score:** 7.5/10  
**Production Readiness:** 6/10 (after high-priority fixes: 8/10)

---

## Appendix: Suggested Immediate Fixes

### Fix 1: Enforce MaxCaptureBytes

```go
// Line 363, inside fence capture:
if st.inFence {
    // Check limit before appending
    if f.opts.MaxCaptureBytes > 0 && st.yamlBuf.Len() >= f.opts.MaxCaptureBytes {
        flushMalformed(f, meta, st, &out, &typed)
        continue
    }
    st.yamlBuf.WriteByte(ch)
    // ... rest of logic
}
```

### Fix 2: Throttle YAML Parsing

```go
// Lines 390-392, replace with:
if st.session != nil && f.opts.EmitParsedSnapshots && ch == '\n' {
    snapshot, perr := parseYAML(st.yamlBuf.String())
    typed = append(typed, st.session.OnUpdate(st.itemCtx, snapshot, perr)...)
}
```

### Fix 3: Handle Unclosed Tags at Final

```go
// Line 259, after filtered/typed processing:
if st.capturing {
    if st.session != nil {
        typed = append(typed, st.session.OnCompleted(st.itemCtx, nil, false, errors.New("unclosed structured block at final"))...)
    }
    st.capturing = false
}
```

---

**End of Review**

