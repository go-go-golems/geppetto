### Review of the Structured Data Filtering Sink (post “presidential debate” synthesis)

This report provides a sober, detailed analysis of the structured data filtering sink feature (FilteringSink), synthesizing perspectives surfaced during the mock “presidential debate” between the design document, implementation, and constituent components. It consolidates strengths, risks, and concrete recommendations grounded in actual code behavior.

### Purpose and scope

- **Purpose**: Evaluate the FilteringSink’s architecture, correctness, ergonomics, performance, and operational profile, drawing on the debate viewpoints and code readings.
- **Scope**: The `FilteringSink` implementation, its stream state machine, YAML parsing strategy, error handling, extractor interface contract, event publication semantics, context lifecycles, and example usage patterns.

### Executive summary

- **What works well**:
  - **Provider-agnostic event transformation**: The sink cleanly wraps an `EventSink`, requiring no engine/provider changes while enabling structured extraction and text filtering in one place.
  - **Typed extractor contract**: Extractors define their own typed events and mapping; the sink remains agnostic and simply publishes the extractor’s events.
  - **Filtered completion consistency**: Forwarded partials maintain their own filtered `Completion`, keeping the user-visible text consistent after filtering.
  - **Multiple blocks per stream**: Per-stream sequencing (`seq`) and per-item contexts support more than one structured block per message.

- **Primary risks and gaps**:
  - **State machine complexity**: Multiple buffers and flags raise the cognitive and correctness load; test coverage becomes critical.
  - **Aggressive YAML parsing**: Parsing on almost every character when `EmitParsedSnapshots` is enabled is CPU- and allocation-heavy.
  - **Malformed/unfinished capture finalization**: Streams that end mid-capture do not reliably emit completion/error for the extractor before state deletion.
  - **Fence language check inefficiency**: Sorting accepted languages on every detection attempt wastes cycles.
  - **Context and lifecycle semantics**: Cancellation and cleanup occur, but behavior for in-flight sessions on abrupt finalization isn’t consistently surfaced to extractors.

### Architectural overview (brief)

- The sink intercepts `EventPartialCompletion` and `EventFinal`, scanning assistant text for structured blocks marked with `<$name:dtype>`, a fenced YAML block (```yaml … ```), and a closing tag.
- During capture, the sink removes the whole block from the forwarded text stream, while delegating to an extractor session (`OnStart`, `OnDelta`, `OnUpdate`, `OnCompleted`) to emit typed events.
- Per-stream state tracks parsing/capture buffers and the filtered completion, ensuring forwarded deltas/aggregates remain consistent.

### Code-grounded observations

- **YAML snapshots parsed per character (performance risk)**

```390:392:geppetto/pkg/events/structuredsink/filtering_sink.go
                if st.session != nil && f.opts.EmitParsedSnapshots {
                    snapshot, perr := parseYAML(st.yamlBuf.String())
                    typed = append(typed, st.session.OnUpdate(st.itemCtx, snapshot, perr)...)
                }
```

  - When `EmitParsedSnapshots` is true, `parseYAML` is invoked for almost every byte appended while inside the fence. This implies frequent string allocation from the `strings.Builder` and many failing parses (until the YAML becomes valid), impacting CPU and GC.

- **Fence language check sorts on each attempt**

```520:522:geppetto/pkg/events/structuredsink/filtering_sink.go
    sort.Strings(langs)
    i := sort.SearchStrings(langs, normalized)
    return i < len(langs) && langs[i] == normalized
```

  - `langs` is sorted within the hot-path detection function. This work repeats for every candidate fence header; it should be preprocessed once (or replaced by a `map[string]struct{}` lookup).

- **Final event handling deletes state even if mid-capture**

```249:262:geppetto/pkg/events/structuredsink/filtering_sink.go
        filtered, typed := f.scanAndFilter(meta, st, delta)
        st.filteredCompletion.WriteString(filtered)
        _ = f.publishAll(st.ctx, meta, typed)
        out := events.NewFinalEvent(meta, st.filteredCompletion.String())
        f.deleteState(meta)
        return f.next.PublishEvent(out)
```

  - If a stream ends without seeing a closing fence/tag, there is no explicit “malformed-finalization” path that calls the extractor session’s `OnCompleted` with an error before `deleteState`. This risks silent loss of a terminal signal to the extractor.

- **Error policy applied only in a limited malformed case**

```432:459:geppetto/pkg/events/structuredsink/filtering_sink.go
func flushMalformed(f *FilteringSink, meta events.EventMetadata, st *streamState, out *strings.Builder, typed *[]events.Event) {
    switch f.opts.OnMalformed {
    case "ignore":
        // drop everything captured so far
    case "forward-raw":
        // write back the buffered pieces
        out.WriteString(st.openTagBuf.String())
        out.WriteString(st.fenceBuf.String())
        out.WriteString(st.yamlBuf.String())
        out.WriteString(st.closeTagBuf.String())
    case "error-events":
        if st.session != nil {
            *typed = append(*typed, st.session.OnCompleted(st.itemCtx, nil, false, errors.New("malformed structured block"))...)
        }
    }
    // reset state
    st.capturing = false
    ...
}
```

  - This policy is triggered in a particular malformed path (e.g., no realistic fence discovered), but not explicitly on “final without proper closure,” leaving a gap in how malformed terminations are surfaced to extractors.

- **Metadata propagation is careful but partial**

```179:187:geppetto/pkg/events/structuredsink/filtering_sink.go
        if ok {
            // keep Type_ and payload; only set meta if zero UUID
            if impl.Metadata_.ID == uuid.Nil {
                impl.Metadata_.ID = meta.ID
            }
            if impl.Metadata_.RunID == "" { impl.Metadata_.RunID = meta.RunID }
            if impl.Metadata_.TurnID == "" { impl.Metadata_.TurnID = meta.TurnID }
        }
```

  - The sink ensures essential IDs are present on extractor-emitted events if missing. Consider documenting which metadata fields are guaranteed to propagate (and which are not, such as usage counters or extras), to set consumer expectations.

### Strengths (expanded)

- **Separation of concerns**: The sink is generic; extractors own their typed schemas and the translation from YAML to domain events.
- **Forwarding correctness for text**: The separate `filteredCompletion` ensures the end-user never sees the structured blocks, while still receiving timely partial updates.
- **Session lifecycle per captured block**: `itemID(messageID:seq)` gives stable identity across multiple captures within the same message.
- **Compatibility with the existing router**: Works with current Watermill sink/router patterns; no engine/provider refactors.

### Weaknesses and risks (expanded)

- **Complex state machine and buffer handling**: Numerous builders and flags (open tag buffer, fence buffer, close tag buffer, YAML buffer; flags for `capturing`, `inFence`, `fenceOpened`, `fenceLangOK`, `awaitingCloseTag`) increase the chance of edge-case bugs and make the logic harder to reason about.
- **Performance pitfalls**:
  - Parsing snapshots on each character (see code reference above) leads to high CPU and memory churn.
  - Sorting accepted languages repeatedly in hot paths.
  - Rebuilding strings from builders for every parse attempt.
- **Termination semantics**:
  - On `final`, if a capture has started but not closed, the extractor may miss a terminal `OnCompleted` with an error when `deleteState` is invoked.
  - Policy-driven malformed handling is only partially applied across error modes.
- **Context semantics**:
  - Stream-level and per-item contexts are created and cancelled, but guarantees to extractor authors are not yet well specified (e.g., whether `OnCompleted` is always called before cancellation, how abrupt stream termination is signaled, etc.).
- **Observability**:
  - Debug logs exist, but metrics for parse attempts, state resets, malformed events, and capture sizes would help surface operational health.

### Ergonomics for extractor authors

- The `ExtractorSession` contract is powerful but places a cognitive burden on authors who only need final results. Many use cases do not need `OnDelta` or `OnUpdate`, yet must implement them or route around them.
- Metadata propagation is mostly handled by the sink, but authors still need to construct correctly typed events and manage error reporting semantics.
- Example code should model context usage and error finalization explicitly to guide authors.

### Recommendations (prioritized)

- **P0 – Correctness and lifecycle**
  - **Emit terminal error on unfinished capture at final**: In `handleFinal`, detect `st.capturing || st.inFence || st.awaitingCloseTag` after the last `scanAndFilter` pass. If still true, finalize via policy with `OnCompleted(..., success=false, err=...)` before `deleteState`.
  - **Unify malformed/error policy**: Apply `OnMalformed` consistently for all early/late termination modes (no fence, no close fence, invalid close tag, tag mismatch, oversized capture).

- **P1 – Performance**
  - **Throttle snapshot parsing**: Parse only on newline boundaries or every N bytes; also parse on fence close. Make N configurable (e.g., `SnapshotParseEveryBytes` or `SnapshotParseEveryNewlines`).
  - **Preprocess accepted fence languages**: Convert to a normalized `map[string]struct{}` on sink construction. Avoid per-call sorting and binary search.
  - **Cap capture size**: Enforce `MaxCaptureBytes` during accumulation; on exceed, apply policy and finalize with error to bound memory and CPU.
  - **Minimize allocations**: Avoid converting the whole builder to a string for each parse attempt; consider incremental decode or diff-based parsing windows.

- **P2 – API and ergonomics**
  - **Extractor adapter**: Provide a helper adapter (e.g., `FinalOnlyExtractor`) that surfaces only a `OnCompleted`-style callback, internally handling deltas/snapshots as no-ops.
  - **Event helpers**: Provide constructors for common typed event patterns that auto-fill metadata, enforcing consistent naming and keys.
  - **Context contract**: Document when item contexts are created/cancelled, and whether `OnCompleted` is guaranteed on all paths.

- **P2 – Observability and testing**
  - **Metrics**: Counters for parse attempts, successful snapshots, malformed terminations by reason, capture sizes, state resets, extractor event counts.
  - **Targeted test suite**: Fragmented tokens (open/close tag, fence), multiple blocks, unknown extractor, wrong language, unfinished blocks at final, max size exceeded, and concurrency scenarios.

### Additional considerations

- **Ordering and correlation**: The sink emits filtered partials first, then extractor events for the same delta. Consumers should not assume strict relative ordering across sinks/queues unless documented.
- **UTF-8 boundaries**: Byte-wise scanning is fine for ASCII markers ("<$", "```", "</$"). The implementation does not interpret non-ASCII content, so multi-byte Unicode in YAML is safe, but tests should still assert correctness around boundary splits.
- **Security**: YAML parsing is user-controlled input from model output. Size caps and parse throttling are essential. Consider parser timeouts or guardrails if feasible.

### Concrete next steps

- **Implement**: Terminal error emission on unfinished captures during final handling, before state deletion, honoring `OnMalformed`.
- **Refactor**: Fence language check to precomputed set; optional light refactor to reduce state flags (e.g., encode phases in a single enum for readability).
- **Optimize**: Snapshot parse throttling; enforce `MaxCaptureBytes`.
- **Document**: Context lifecycle guarantees, event metadata propagation rules, and ordering guarantees. Provide an extractor authoring guide with a minimal “final-only” example.
- **Test**: Add table-driven tests that cover the above edge cases and performance characteristics.

### Conclusion

The FilteringSink delivers meaningful capability: extracting structured payloads in real time from LLM text streams without exposing them to end-users, while preserving typed event workflows. Its current implementation is functionally rich but exhibits avoidable performance costs and lifecycle edge cases in malformed/unfinished scenarios. With targeted fixes—particularly around finalization correctness, snapshot parse throttling, and fence-language preprocessing—the feature can mature into a robust, maintainable middleware component suitable for production workloads. Clearer contracts and examples for extractor authors will further reduce friction and enable consistent downstream adoption.


