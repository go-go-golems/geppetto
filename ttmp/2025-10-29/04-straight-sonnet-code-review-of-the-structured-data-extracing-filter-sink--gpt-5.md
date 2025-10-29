### Straight Sonnet Code Review — Structured Data Extracting FilteringSink (GPT-5)

This review evaluates `geppetto/pkg/events/structuredsink/filtering_sink.go` against the intent outlined in the design draft `ttmp/2025-10-27/01-analysis-and-design-brainstorm-for-a-streaming-middleware-system-and-structured-data-extraction-streaming-design.md`. It focuses on correctness, performance, API ergonomics, observability, and alignment with the design requirements.

### What this component does
- Filters inline structured YAML blocks marked with `<$name:dtype>```yaml ...```</$name:dtype>` from assistant text as it streams.
- Emits extractor-specific typed events (start, delta, update, completed) while forwarding a filtered text stream so end users never see the embedded YAML.
- Maintains per-stream state for consistency of partial `Completion` and `Final` text.

### Architecture and flow (as implemented)
- Wrapper `EventSink` that intercepts partial and final events, scans for tags and fenced YAML, then forwards filtered events and publishes typed events to the downstream sink.
- Per-stream state keyed by `EventMetadata.ID`; individual item sessions via `ExtractorSession` with `itemID = message_id:seq`.
- Filtering is performed in a single-byte streaming scanner with small internal buffers for the open tag, fence header, YAML content, and close tag.

Key structures and options:
```17:25:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
// Options control the filtering sink behavior.
type Options struct {
    EmitRawDeltas       bool
    EmitParsedSnapshots bool
    MaxCaptureBytes     int
    AcceptFenceLangs    []string
    OnMalformed         string // "ignore" | "forward-raw" | "error-events"
    Debug               bool   // if true, emit zerolog debug traces
}
```

```106:137:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
// Parser state per message stream
type streamState struct {
    id        uuid.UUID
    // contexts: stream-scoped and current item-scoped
    ctx       context.Context
    cancel    context.CancelFunc
    itemCtx   context.Context
    itemCancel context.CancelFunc
    // track full raw text seen so far (to avoid double-appending on final)
    rawSeen   strings.Builder
    // output buffer state for completion consistency
    filteredCompletion strings.Builder
    // small carry for boundary detection across partials
    carry string

    // capture state
    capturing bool
    name      string
    dtype     string
    seq       int
    session   ExtractorSession
    yamlBuf   strings.Builder

    // sub-state buffers for partial pattern matches
    openTagBuf   strings.Builder
    fenceBuf     strings.Builder
    closeTagBuf  strings.Builder
    inFence      bool
    fenceOpened  bool
    fenceLangOK  bool
    awaitingCloseTag bool
}
```

### Strengths
- Event-sink wrapper aligns with the design and keeps engines/provider code unchanged.
- Per-stream state and consistent `Completion` reconstruction are clean and correct.
- Extractor contracts (`Extractor` and `ExtractorSession`) are well-factored for typed fan-out.
- Metadata propagation for typed events is handled centrally before publishing.

### Gaps vs design and notable issues
- Tag robustness and whitespace: The open tag parser requires an exact match without whitespace variance, contradicting the “robust to whitespace” design intent.
```484:500:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
var (
    reOpen = regexp.MustCompile(`^<\$([a-zA-Z0-9_-]+):([a-zA-Z0-9._-]+)>$`)
)

func tryParseOpenTag(st *streamState, s string) bool {
    if !strings.HasPrefix(s, "<$") {
        return false
    }
    if strings.HasSuffix(s, ">") {
        m := reOpen.FindStringSubmatch(s)
        if len(m) == 3 {
            st.name = m[1]
            st.dtype = m[2]
            st.openTagBuf.Reset()
            return true
        }
        // invalid tag
        return false
    }
    return false
}
```
- Max capture ceiling never enforced: `Options.MaxCaptureBytes` is defined but unused. This can lead to unbounded memory growth on large payloads.
- Per-byte emission and parsing: The scanner emits `OnDelta` per character and attempts YAML parsing on each character, which is expensive and noisy.
```386:393:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
// mid-yaml: optionally emit delta/snapshot
if f.opts.EmitRawDeltas && st.session != nil {
    typed = append(typed, st.session.OnDelta(st.itemCtx, string(ch))...)
}
if st.session != nil && f.opts.EmitParsedSnapshots {
    snapshot, perr := parseYAML(st.yamlBuf.String())
    typed = append(typed, st.session.OnUpdate(st.itemCtx, snapshot, perr)...)
}
```
- Double emission of raw deltas at fence close: On closing the fence, the code re-emits the entire buffer again, effectively duplicating data already sent as per-char deltas.
```371:379:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
// emit accumulated delta without the closing fence
if f.opts.EmitRawDeltas && st.session != nil {
    typed = append(typed, st.session.OnDelta(st.itemCtx, s)...)
}
// parse snapshot on close as well
snapshot, perr := parseYAML(s)
if st.session != nil {
    if f.opts.EmitParsedSnapshots {
        typed = append(typed, st.session.OnUpdate(st.itemCtx, snapshot, perr)...)
    }
}
```
- Fence close detection over-approximates: It looks for a literal "```" suffix in the YAML buffer without ensuring it’s on its own line, which can mis-detect YAML that legitimately contains backticks.
```361:370:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
if st.inFence {
    // Collect YAML until closing fence
    st.yamlBuf.WriteByte(ch)
    if strings.HasSuffix(st.yamlBuf.String(), "```") {
        // remove the trailing fence from yamlBuf
        s := st.yamlBuf.String()
        if len(s) >= 3 { s = s[:len(s)-3] }
        st.yamlBuf.Reset()
        st.yamlBuf.WriteString(s)
        // ...
    }
}
```
- Sorting mutation and cost in fence detection: Accepted languages are sorted on every check, mutating the slice and adding per-byte cost.
```519:523:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
normalized := strings.ToLower(header)
sort.Strings(langs)
i := sort.SearchStrings(langs, normalized)
return i < len(langs) && langs[i] == normalized
```
- Malformed "forward-raw" doesn’t restore the tag: `flushMalformed` attempts to forward pieces but the open tag buffer was reset at capture start, so the original open tag is lost.
```311:318:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
st.yamlBuf.Reset()
st.fenceBuf.Reset()
st.closeTagBuf.Reset()
st.inFence = false
st.fenceOpened = false
st.fenceLangOK = false
st.awaitingCloseTag = false
```
```438:445:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
// write back the buffered pieces
out.WriteString(st.openTagBuf.String())
out.WriteString(st.fenceBuf.String())
out.WriteString(st.yamlBuf.String())
out.WriteString(st.closeTagBuf.String())
// ... error-events branch
if st.session != nil {
    *typed = append(*typed, st.session.OnCompleted(st.itemCtx, nil, false, errors.New("malformed structured block"))...)
}
```
- Unused/ineffective state:
  - `streamState.carry` is defined but unused.
  - `fenceLangOK` is set but not consulted.
```119:136:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
// small carry for boundary detection across partials
carry string
// ...
fenceLangOK  bool
```
- Logging may leak raw YAML in debug mode: partial handler logs the full incoming `delta` prior to capture, potentially including YAML content.
```222:229:/home/manuel/workspaces/2025-10-27/build-geppetto-event-structured-data-middleware/geppetto/pkg/events/structuredsink/filtering_sink.go
st := f.getState(meta)
if f.opts.Debug { log.Debug().Str("stream", meta.ID.String()).Str("event", "partial").Str("delta", delta).Msg("incoming") }
st.rawSeen.WriteString(delta)
```

### Correctness and edge cases
- Multiple structured blocks per stream: supported via `seq` and reset upon completion — good.
- Unknown extractor: gracefully falls back to forwarding the tag text — good.
- Nested tags: not supported by design; current state machine would likely mis-handle nesting — acceptable given scope.
- Final-tail handling: computes final tail by subtracting `rawSeen` prefix and rescans — aligns with requirement.
- Malformed handling: policy-driven, but tag loss in `forward-raw` breaks “forward the original raw” semantics; needs fixing.

### Performance
- Per-byte `String()` calls allocate copies of buffers on each character, causing O(n^2) behavior for larger YAML sections.
- YAML parsing on every character is overkill; move to periodic (newline or N bytes) snapshots.
- Sorting `AcceptFenceLangs` repeatedly is wasteful; pre-normalize and pre-sort or use a map/set.
- No `MaxCaptureBytes` enforcement risks memory blowups on large payloads.

### API ergonomics and options
- Options are sensible, but `MaxCaptureBytes` must be honored.
- Consider adding knobs for snapshot throttle (e.g., `SnapshotEveryBytes`, `SnapshotOnNewline`).
- Clarify `EmitRawDeltas` semantics (bytes vs lines vs chunks) and ensure no duplicate emissions at fence close.

### Observability and logging
- Keep existing debug logging but avoid including raw YAML. Either scrub deltas when capturing or log sizes only (e.g., `delta_len`).
- Typed events are published with proper metadata propagation — good.

### Concurrency and lifecycle
- Per-stream state keyed by UUID is protected during creation/deletion with a mutex; scanning mutates per-stream state without locking, which is typically fine if events for a given stream are processed sequentially. If the sink can be called concurrently for the same stream, consider synchronization or serialization guarantees at the router.
- Context lifecycles (`ctx`, `itemCtx`) are well managed and cancelled on completion — good.

### Security
- Avoid logging raw YAML (even under `Debug`) to prevent content leaks.
- Consider a configurable redaction strategy if logging structured contents.

### Test coverage suggestions
- Multiple blocks in a single stream (sequential) with proper `itemID` sequencing.
- Fragmentation across deltas: open tag split, fence header split, close fence split, close tag split.
- Wrong or unsupported fence language behavior.
- Malformed termination across all `OnMalformed` policies (including verifying `forward-raw` restores original text).
- Large payloads and `MaxCaptureBytes` behavior.
- Snapshot parsing throttle correctness (once added).
- Final-tail correctness when partials did not include all bytes (ensuring no double filtering).

### Concrete improvements (proposed)
1) Correctness and semantics
- [ ] Accept whitespace-tolerant open/close tag parsing per design (allow minimal whitespace around name/dtype).
- [ ] Ensure fence close detection requires backticks on their own line (e.g., `\n```\s*$`).
- [ ] Do not re-emit full buffer as a delta at fence close; either emit nothing or only a final small sentinel delta if desired.
- [ ] Fix `OnMalformed: forward-raw` to include the exact open tag; retain the literal open tag string before resetting buffers.
- [ ] Honor `MaxCaptureBytes`: truncate with policy (error-events vs forward-raw) once limit reached.

2) Performance
- [ ] Avoid per-character `String()` conversions; track the last 3–4 bytes for fence detection using a tiny ring buffer.
- [ ] Throttle YAML parsing to newline boundaries or every N bytes (configurable), with a small timer-based back-off option.
- [ ] Pre-normalize and pre-sort `AcceptFenceLangs` once in constructor; or use a `map[string]struct{}` for O(1) membership without mutation.

3) API and ergonomics
- [ ] Add `SnapshotEveryBytes int` and `SnapshotOnNewline bool` options.
- [ ] Clarify `EmitRawDeltas` contract to be line- or chunk-based; document no duplication at close.

4) Observability
- [ ] Keep `Debug` logs, but replace `.Str("delta", delta)` with `.Int("delta_len", len(delta))` once a capture starts; optionally log state transitions.
- [ ] Consider a logger adapter that can be provided by callers to standardize redaction.

5) Cleanup
- [ ] Remove or implement `streamState.carry` and `fenceLangOK`.
- [ ] Add unit tests per the above coverage suggestions.

### Small code references index
- Options and defaults: see definition of `Options` and `withDefaults`.
- Open tag parsing (strict): see `reOpen` and `tryParseOpenTag`.
- Per-byte delta and per-byte parsing: see `scanAndFilter` mid-YAML branch.
- Duplicate emission on fence close: see `scanAndFilter` fence close branch.
- Fence language sorting per check: see `tryDetectFenceOpen`.
- Malformed policy raw-forward losing tag: see capture start reset and `flushMalformed`.
- Debug logging exposing raw deltas: see `handlePartial` logging.

This implementation is close to the intended architecture and behavior, but it needs targeted fixes for correctness (tag robustness, fence-close semantics, malformed forward), substantial performance improvements (avoid per-char allocations and parsing), and honoring safety limits (`MaxCaptureBytes`). With those addressed and a focused test suite, it will be ready for broader use.
