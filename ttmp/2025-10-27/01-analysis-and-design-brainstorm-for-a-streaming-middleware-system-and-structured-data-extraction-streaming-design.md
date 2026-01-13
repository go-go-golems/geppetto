### Streaming middleware for event-level structured data extraction (design draft)

This document analyzes the current events/middleware model in Geppetto and proposes an event-level middleware that filters inline structured payloads out of streaming text while emitting dedicated structured events in real time. The goal is to enable LLM-driven custom data extraction without showing the embedded YAML to the end-user.

### Context recap

- Engines publish typed streaming events (start, partial, final, tool-call, tool-result, error, interrupt, etc.), each carrying `EventMetadata` with stable `message_id`, `run_id`, `turn_id` and evolving usage.
- Sinks deliver those events (e.g., Watermill via `NewWatermillSink`) and can also be attached to context via `events.WithEventSinks(ctx, ...)` so helpers/tools publish without plumbing.
- Middlewares (today) wrap `engine.Engine` around `RunInference(ctx, *turns.Turn)`; they are provider-agnostic and operate on the turn/block level. We want a complementary mechanism acting on the event stream.

Key references:
- Events overview and Watermill routing: see `geppetto/pkg/doc/topics/04-events.md`.
- Event types and metadata: see `geppetto/pkg/events/chat-events.go`.
- Turn-based middlewares: see `geppetto/pkg/doc/topics/09-middlewares.md`.

### Requirement

- Recognize streaming markers inside assistant text:

  <$middlewareName:$dataType>
  ```yaml
  ... YAML, possibly long ...
  ```
  </$middlewareName:$dataType>

- While streaming partial completions:
  - Filter the entire tagged block (open tag, fenced YAML, close tag) out of the text stream.
  - Update downstream text events so the user never sees the extracted data.
  - Incrementally publish dedicated structured events representing the captured YAML (deltas, parsed snapshots, and final completion), keyed by the same `EventMetadata` correlation.
- Apply to both partial and final text events (and do nothing for non-text events like tool calls).
- Support multiple structured blocks in a single stream; no support for nesting (first-cut).
- Robust to marker splits across token boundaries and incomplete/broken YAML.

### High-level design

We introduce an event-level middleware in the form of a wrapping `EventSink` that can transform and fan-out events before forwarding them to the next sink(s).

- New package: `geppetto/pkg/events/middleware/structuredsink` (name indicative).
- Type: `FilteringSink` implementing `events.EventSink`.
- Composition: `FilteringSink` wraps one or more downstream sinks. Event → FilteringSink → downstream sinks.
- Responsibility:
  - Maintain per-stream state keyed by `EventMetadata.ID` (the stable `message_id`).
  - Scan `EventPartialCompletion` deltas for the marker, handle fragmented tokens, capture YAML inside fenced block, and filter that text out from the forwarded text stream.
  - Emit dedicated structured-data events as the YAML content grows; attempt best-effort incremental parsing to produce structured snapshots.
  - On `EventFinal`, finish any in-flight capture, emit completion/failure, forward a filtered `final` event (text without the captured blocks).

Why a Sink wrapper? Engines, helpers and existing middlewares already publish through sinks (engine-configured or context-carried). A wrapper sink lets us process the same unified stream without modifying engines or routers, and preserves provider-agnostic behavior.

### Per-middleware custom event types (strong typing)

Each structured-data middleware defines and emits its own strongly typed custom events. The filtering sink is agnostic of those types; it only invokes the middleware session to produce events which already carry the correct `Type_` and payload. This preserves strong typing end-to-end and avoids a catch‑all generic schema.

Example: a “citations:v1” extractor could define and register its own events:

```go
// Package citations
type EventCitationStarted struct {
    events.EventImpl
    ItemID string `json:"item_id"`
}

type EventCitationDelta struct {
    events.EventImpl
    ItemID string `json:"item_id"`
    Delta  string `json:"delta"`
}

type EventCitationUpdate struct {
    events.EventImpl
    ItemID  string         `json:"item_id"`
    Entries []CitationItem `json:"entries,omitempty"`
    Error   string         `json:"error,omitempty"`
}

type EventCitationCompleted struct {
    events.EventImpl
    ItemID  string         `json:"item_id"`
    Entries []CitationItem `json:"entries,omitempty"`
    Success bool           `json:"success"`
    Error   string         `json:"error,omitempty"`
}

func init() {
    _ = events.RegisterEventFactory("citations-started", func() events.Event {
        return &EventCitationStarted{EventImpl: events.EventImpl{Type_: "citations-started"}}
    })
    _ = events.RegisterEventFactory("citations-delta", func() events.Event {
        return &EventCitationDelta{EventImpl: events.EventImpl{Type_: "citations-delta"}}
    })
    _ = events.RegisterEventFactory("citations-update", func() events.Event {
        return &EventCitationUpdate{EventImpl: events.EventImpl{Type_: "citations-update"}}
    })
    _ = events.RegisterEventFactory("citations-completed", func() events.Event {
        return &EventCitationCompleted{EventImpl: events.EventImpl{Type_: "citations-completed"}}
    })
}
```

Other middlewares (e.g., “plan:v2”, “sql-query:v1”) define their own event types similarly.

### Extractor registry and configuration

Multiple extractors are registered, keyed by (`name`, `dataType`) from the tag. Each extractor owns its event schema and streaming behavior via a session interface. The sink delegates streaming callbacks; the extractor returns already-typed events to publish.

API sketch:

```go
type Extractor interface {
    Name() string        // from <$name:$dataType>
    DataType() string
    NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession
}

type ExtractorSession interface {
    // Called when the capture starts
    OnStart(ctx context.Context) []events.Event
    // Raw YAML deltas as they arrive (might be empty if only snapshots are desired)
    OnDelta(ctx context.Context, raw string) []events.Event
    // A best-effort parsed snapshot; parseErr is non-nil if YAML not valid yet
    OnUpdate(ctx context.Context, snapshot map[string]any, parseErr error) []events.Event
    // Called when the fenced block closes or the stream ends
    OnCompleted(ctx context.Context, final map[string]any, success bool, err error) []events.Event
}

type Options struct {
    EmitRawDeltas        bool
    EmitParsedSnapshots  bool
    MaxCaptureBytes      int
    AcceptFenceLangs     []string // default: ["yaml","yml"]
    OnMalformed          string   // "ignore" | "forward-raw" | "error-events"
}

func NewFilteringSink(next events.EventSink, opts Options, extractors ...Extractor) *FilteringSink
```

Behavior knobs let operators control verbosity and failure semantics; extractors remain fully in control of which typed events to emit.

### Stream state machine

Per `message_id`, the sink maintains a small state machine and buffers:

- States: `Idle → TagOpenFound → AwaitingFenceOpen → InFence → AwaitingFenceClose → Completed → Idle`
- Buffers:
  - `carryOver`: a small tail (e.g., last 64 bytes) for cross-delta pattern detection.
  - `filteredCompletion`: the text shown to the user (original completion minus removed tagged blocks).
  - `yamlBuf`: captured YAML inside the fence.
  - `name`, `dataType`, `seq`: identifiers for the current item. `itemID = message_id + ":" + seq` to support multiple captures per stream.

Tag/fence detection:
- Open tag regex (robust to whitespace): `(?s)<\$(?P<name>[a-zA-Z0-9_-]+):(?P<dtype>[a-zA-Z0-9._-]+)>`
- Code fence open: start of a triple backtick with accepted language (yaml|yml) optionally with trailing spaces and newline.
- Code fence close: matching triple backticks on its own line.
- Close tag: `</\$(?P=name):(?P=dtype)>`

Processing algorithm (simplified):

```go
on PartialCompletion(delta):
  st := getState(meta.ID)
  text := st.carryOver + delta

  // 1) If not capturing, search for open tag → transition and emit Started
  // 2) While InFence, split incoming text into {yamlPart, outsidePart}
  //    - Append yamlPart to st.yamlBuf
  //    - If EmitRawDeltas: events.Publish(session.OnDelta(ctx, yamlPart) ...)
  //    - Try parse st.yamlBuf as YAML → map[string]any
  //      - Publish session.OnUpdate(ctx, snapshot, parseErr) ... (extractor emits typed events)
  // 3) Detect fence/close tag boundaries even when split across deltas
  //    - On close: publish session.OnCompleted(ctx, final, success, err) and reset to Idle
  // 4) For outsidePart (the user-visible text), compute filteredDelta by removing any tag/fence fragments consumed in steps above
  //    - Append filteredDelta to st.filteredCompletion
  //    - Forward a new PartialCompletion event with Delta=filteredDelta, Completion=st.filteredCompletion
  st.carryOver = tail(text)

on Final(text):
  st := getState(meta.ID)
  // Run the same filter over the remaining segment to finish open captures
  // If capture left open → depending on OnMalformed, either finalize with error or forward raw
  filteredText := filter(st, text)
  forward Final with Text=filteredText
  cleanup st
```

Complexity: O(n) over text length per stream; minimal look-behind (`carryOver`) covers split tokens.

### Forwarding and consistency of completion

`EventPartialCompletion` has both `Delta` and `Completion`. Since we filter out content, we cannot forward the original `Completion` (it would diverge). The sink maintains `filteredCompletion`, ensuring forwarded `Completion` is consistent with the filtered `Delta` sequence. `EventStart`, tool events, errors, interrupts, etc., are forwarded unchanged.

### Error handling

- YAML parses may fail mid-stream. We still emit deltas; extractor sessions decide how to reflect parse errors in their typed events.
- On closing tag, one last parse attempt determines success and final payload; sessions emit their own “completed” or “failed” event(s).
- If the stream ends without a closing tag, behavior is controlled by `OnMalformed` and surfaced via extractor-defined failure events.

### UI/consumer patterns

- Router handlers subscribe to the extractor-defined event types (e.g., `citations-*`) and process strongly typed payloads.
- Pretty-print YAML/JSON as needed; raw deltas and typed updates enable live UIs.
- Correlation via `message_id`/`run_id`/`turn_id` remains unchanged.

### API usage example

```go
router, _ := events.NewEventRouter()
sink := middleware.NewWatermillSink(router.Publisher, "chat")

// Configure the structured-data filtering sink with a typed extractor
opts := structuredsink.Options{EmitRawDeltas: true, EmitParsedSnapshots: true}
citations := citations.NewExtractor("citations", "v1") // defines its own typed events
mwSink := structuredsink.NewFilteringSink(sink, opts, citations)

// Engine emits to the middleware sink
eng, _ := factory.NewEngineFromParsedLayers(parsed, engine.WithSink(mwSink))

eg, groupCtx := errgroup.WithContext(ctx)
eg.Go(func() error { return router.Run(groupCtx) })
eg.Go(func() error {
    <-router.Running()
    runCtx := events.WithEventSinks(groupCtx, mwSink) // tools/helpers also publish through the same sink
    _, err := eng.RunInference(runCtx, messages)
    return err
})
_ = eg.Wait()
```

### Extensibility and composition

- The `FilteringSink` can be part of a chain if we also define a minimal event-middleware abstraction:

```go
type HandlerFunc func(ctx context.Context, ev events.Event) error
type Middleware func(HandlerFunc) HandlerFunc

// FilteringSink can internally compose a chain of Middlewares before forwarding to the downstream sink.
```

This mirrors the request-time middleware design (`NewEngineWithMiddleware`) but on events. First phase only ships the `FilteringSink` with built-in YAML extraction; the middleware chain can be added later.

### Edge cases to consider

- Multiple tagged sections per stream (supported via `seq`).
- No fencing or wrong language → controlled by `OnMalformed`.
- Whitespaces/newlines around tags/fences should be fully filtered with the block; configurable if callers prefer to keep surrounding whitespace.
- Model outputs additional comments before/after the fenced YAML inside the tags: those remain part of `yamlBuf` and are attempted to parse; they will surface in deltas and error snapshots if not valid YAML.

### Minimal implementation plan

- [ ] Implement `FilteringSink` with per-message state, scanning, filtering, and delegation to extractor sessions.
- [ ] Define a reference extractor (e.g., `citations`) which declares and registers its own typed events.
- [ ] Add unit tests for fragmented tokens (tag, fence, close), multiple blocks, malformed termination, and large payload ceilings.
- [ ] Provide an example main demonstrating router + filtering sink + typed extractor + handlers.
- [ ] Optional: add a small helper to pretty-print JSON payloads in handlers.

### Security and performance notes

- Parsing YAML on every delta can be CPU-heavy for large payloads. Throttle snapshot parsing (e.g., only on newline boundaries or every N bytes). Keep `MaxCaptureBytes` to avoid memory blowups.
- Never log raw YAML by default; prefer size/shape summaries. Use `EventMetadata.Extra` carefully to avoid leaking sensitive content.

### Summary

The proposed `FilteringSink` enables event-level structured data extraction by transparently filtering inline data blocks from the user-visible text while emitting dedicated structured events in real time. It leverages Geppetto’s event model, context-carried sinks, and custom event registry without touching engines or providers, and composes cleanly with existing Watermill-based routing.


