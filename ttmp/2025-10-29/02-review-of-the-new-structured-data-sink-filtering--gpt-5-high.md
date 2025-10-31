### Review of the New Structured-Data Filtering Sink

This report evaluates the event-level structured-data filtering capability introduced via the `FilteringSink` and its per-stream `streamState` parser. It synthesizes perspectives from the internal “debate” participants: implementation primitives (`FilteringSink`, `streamState`, `tryParseOpenTag`, `parseYAML`) and stakeholder viewpoints (Performance Engineer, Product/UX Lead). The goal is to converge on working defaults, clarify trade-offs, and outline safe roll-out steps.

### Purpose and scope

- **Purpose**: Filter model-emitted inline structured blocks from user-visible text while emitting strongly typed extractor events in real time.
- **Scope**: Event-level streaming only (partial and final text events); non-text events are pass-through. One level of tags (no nesting). Multiple blocks per stream are supported.

### Quick recap of the feature

- Tag format inside assistant text denotes extractable content: `<$name:dtype> \n ```yaml|yml ... ``` \n </$name:dtype>`.
- The `FilteringSink` wraps a downstream `events.EventSink`, intercepts partial/final text, filters tagged regions out of the forwarded text, and calls a registered `Extractor`’s `ExtractorSession` to emit typed events (`OnStart`, `OnDelta`, `OnUpdate`, `OnCompleted`).
- The sink maintains a consistent `filteredCompletion` so UI-visible text remains the original text minus structured blocks.
- Options surface behavior controls: `EmitRawDeltas`, `EmitParsedSnapshots`, `MaxCaptureBytes`, `AcceptFenceLangs`, `OnMalformed` (ignore | forward-raw | error-events), `Debug`.

### Consolidated analysis by topic

#### 1) Event-level boundary vs modifying engines

- **Consensus**: Event-level wrapper is the right boundary. It keeps provider engines agnostic and leverages existing sink routing. It also preserves correlation via `EventMetadata` without retooling every engine/provider.
- **Implementation**: `FilteringSink` implements `events.EventSink`, keeping forwarding logic centralized; `publishAll` ensures metadata is set when missing.
- **Risk**: Hot-path work in the sink can affect end-to-end latency. Addressed via performance profiles (see below).

#### 2) Emission strategy: raw deltas, parsed snapshots, final

- **Positions**: Product favors live confidence via snapshots; Performance prefers minimal parsing. Implementation supports both.
- **Recommendation**: Default to emitting raw deltas and final parse; gate parsed snapshots behind cadence to control CPU.
- **Cadence**: Parse on newline boundaries or every N bytes; provide knobs to tune per deployment.

#### 3) Malformed handling: defaults and safety

- **Mechanics**: `OnMalformed` governs outcomes when fences/tags are broken or streams end mid-capture.
- **Recommendation**: Default = `error-events` so consumers get explicit failure signals. Allow opt-in `forward-raw` for debugging or perf-critical paths that prefer pass-through.
- **Safety**: `streamState` must cancel per-item contexts and reset buffers on failure to avoid leaks.

#### 4) Performance and memory profile

- **Concerns**: Parsing YAML on every keystroke is O(n) with allocations; repeated attempts can hurt p99.
- **Mitigations**:
  - Turn snapshots off by default; enable cadence-based snapshots where needed.
  - Enforce `MaxCaptureBytes` hard ceiling per item; reject or finalize with explicit error when exceeded.
  - Keep detection fast: strict tag parsing (`tryParseOpenTag`) and cheap fence detection prior to parsing.
  - Provide documented “profiles”: conservative (final-only parse), balanced (deltas + cadenced snapshots), verbose (deltas + frequent snapshots) with suggested SLO budgets.

#### 5) Typed extractor events vs generic blob

- **Decision**: Prefer strongly typed per-extractor events. The sink remains agnostic and simply publishes whatever the extractor returns.
- **Rationale**: Better discoverability, routing, and UI contracts (`citations-*`, `plan-*`). Generic payloads defer schema and slow downstream ergonomics.

#### 6) Multiple blocks per stream (no nesting)

- **Design**: `seq` increments per block; `itemID = message_id:seq` maintains correlation for parallel blocks in one completion.
- **Behavior**: New open tag during active capture is treated as malformed. Close resets all item-scoped state.
- **UX**: Offer optional whitespace-collapsing around removed blocks to avoid visible gaps.

#### 7) Language scope for fenced content

- **Default**: `AcceptFenceLangs = ["yaml", "yml"]`.
- **Extensibility**: Permit opt-in expansion (e.g., `json`, `toml`) on a per-extractor basis. Each added language must have a corresponding parser and error model; do not coerce JSON into YAML.
- **Operational**: Track per-language metrics before widening defaults.

#### 8) Event ordering guarantees

- **Contract**: For each delta, publish the filtered text event first, then derived typed events. This preserves UI snappiness and avoids typed updates racing ahead of visible text.
- **Observability**: Include a source delta index or monotonically increasing counter in typed events for cross-checking ordering in logs/metrics.

#### 9) Observability and privacy

- **Logging**: `opts.Debug` should log shapes (byte counts, state transitions), not payloads. Avoid high-cardinality labels.
- **Metrics**: Per extractor: `bytes_captured`, `snapshots_emitted`, `parse_failures`, `malformed_blocks`, `items_completed`, `items_failed`.
- **Privacy**: Never log raw YAML by default; sampling-based debug only in controlled environments.

#### 10) Backpressure and downstream failure policy

- **Policies**: Block (strict), drop-typed (best-effort), bypass (pass-through raw).
- **Recommendation**: Default to strict blocking for integrity. Allow per-extractor overrides: non-critical extractors can be best-effort under pressure.
- **Safeguards**: Enforce `MaxCaptureBytes` and surface a health/degradation event when a policy is activated.

#### 11) Versioning and migration

- **Tag format**: Versions live in the `dtype` (e.g., `<$citations:v1>`). Registry maps to the correct extractor implementation.
- **Unknown versions**: Emit explicit error events; do not silently pass through.
- **Migration**: Support dual-publish windows and deprecation notices; downstream UIs subscribe to both until migration completes.

### Working defaults (proposed)

- `OnMalformed = "error-events"`
- `EmitRawDeltas = true`
- `EmitParsedSnapshots = false` (enable with cadence when needed)
- `AcceptFenceLangs = ["yaml", "yml"]`
- `MaxCaptureBytes` = conservative default (documented; size TBD by SLOs)
- Event ordering: text first, typed second, per delta
- Observability: metrics on by default; debug logs payload-free

### Risks and mitigations

- **CPU/alloc pressure from frequent YAML parses**: Default snapshot parsing off; cadence controls; profile with representative payloads.
- **Memory growth with large or unclosed blocks**: Enforce `MaxCaptureBytes`; finalize with explicit error on exceed or stream-end without close.
- **Ordering heisenbugs**: Document and test the text-first rule; add delta index for verification.
- **Schema sprawl with typed events**: Versioned extractors; registry-based discovery; provide docs per extractor.
- **Leakage of sensitive content in logs**: Payload-free logs by default; redaction guarantees; sampling only in safe environments.

### Open questions

- What are the baseline and target p95/p99 thresholds for snapshot cadence across supported runtimes?
- Do we need a cross-extractor debounce strategy (e.g., burst control) when multiple blocks appear rapidly in a single stream?
- Should whitespace-collapsing around removed blocks be global or per-extractor?
- What is the deprecation timeline for extractor versions, and how are consumers notified?
- Do we need a “strict mode” that forbids unknown languages entirely versus soft-fail with error events?

### Validation plan and acceptance criteria

- **Functional**:
  - Filtered text never exposes fenced content; `Completion` remains internally consistent across partials and final.
  - Multiple blocks per stream produce distinct `itemID`s and correct typed event sequences.
  - Malformed scenarios generate the expected outcomes for each `OnMalformed` mode.
- **Performance**:
  - With default profile (deltas on, snapshots off), added overhead remains within agreed p95/p99 budgets.
  - Cadenced snapshots mode demonstrates controllable CPU growth with linear relation to cadence.
- **Reliability**:
  - Backpressure policies are honored; degradation surfaces health metrics/events; no buffer leaks.
  - Versioning: unknown versions yield explicit error events; dual-publish migration validated end-to-end.

### Recommended next steps

- Ship with conservative defaults and a documented performance profile matrix.
- Add throttle knobs for snapshot cadence: bytes and/or newline-based, per extractor.
- Expand metrics and tracing: include delta index for ordering validation, per-language parser timings, and malformed taxonomy.
- Provide a reference extractor (e.g., `citations:v1`) with sample typed events and a demo app wiring.
- Author consumer-facing docs that enumerate event types per extractor, versioning policy, and example subscriptions.

### References

- Design draft: `geppetto/ttmp/2025-10-27/01-analysis-and-design-brainstorm-for-a-streaming-middleware-system-and-structured-data-extraction-streaming-design.md`
- Implementation entry point: `geppetto/pkg/events/structuredsink/filtering_sink.go`


