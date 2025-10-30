---
Title: Structured Data Event Sinks
Slug: structured-data-event-sinks
Short: Tag-only sink with extractor-owned parsing for structured blocks, YAML helpers, and a complete example.
Topics:
- events
- streaming
- structured-data
- sinks
- yaml
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Structured Data Event Sinks

## Overview

Structured Data Event Sinks let you extract typed, structured information from live LLM text streams without coupling parsing to the core engine. In the current v2 design, the sink only understands XML-like open/close tags and routes raw block payloads to pluggable extractors. Each extractor owns its parsing (for example YAML → a Go struct) and emits typed events in real time as the text arrives.

This document explains the architecture, APIs, and helper utilities, and shows how to build an extractor using the citations example.

## Core Concepts

- **Tag-only sink**: The sink detects blocks delimited by `<$name:dtype>` … `</$name:dtype>` and forwards the raw, unmodified payload to an extractor that registered for `(name, dtype)`.
- **Extractor-owned parsing**: Extractors decide how to parse. YAML is common, but JSON or any custom format works.
- **Streaming lifecycle**: For each block, the sink calls `OnStart`, streams chunks via `OnRaw`, and finalizes with `OnCompleted` (with the full raw payload and success status).
- **Typed events**: Extractors return domain-specific events (e.g., `citations-update`) that the sink publishes alongside filtered text.
- **Debounced parsing**: The `parsehelpers` package provides a `YAMLController` that parses progressively (on newline or every N bytes) and on finalization.
- **Malformed handling**: If a block is not properly closed at stream end, the sink flushes according to `OnMalformed` policy and calls `OnCompleted(..., success=false, err)`.

## How It Works

### Tag format

Blocks are marked in the stream using XML-like tags (the `\`` is to avoid rendering issues in nested markdown).

```text
<$citations:v1>
`\``yaml
citations:
  - title: GPT-4 Technical Report
    authors: [OpenAI]
`\``
</$citations:v1>
```

Within a stream, tags can be interleaved with normal text. The sink removes the structured blocks from the forwarded “filtered text,” and sends the raw payload inside the tags to the registered extractor session.

### Streaming flow

1. Upstream produces `EventPartialCompletion` deltas and a final `EventFinal`.
2. The sink maintains per-stream state, detects tags incrementally, and forwards chunks inside a captured block to the active extractor session.
3. For every open block:
   - `OnStart(ctx)` is called once.
   - `OnRaw(ctx, chunk)` is called for each captured chunk.
   - When the close tag arrives (or at final), `OnCompleted(ctx, fullRaw, success, err)` is called.
4. The sink publishes:
   - Filtered partials/final text (with structured blocks removed), and
   - All events returned by extractor methods.

### Malformed blocks

If the stream finishes with an unclosed block, the sink applies the configured policy:

- `error-events` (default): call `OnCompleted(..., success=false, err)` so the extractor can emit an error event.
- `forward-raw`: reinsert a best-effort raw reconstruction into the filtered text.
- `ignore`: drop captured payload.

## Public API and Types

### Sink options

```go
type Options struct {
    MaxCaptureBytes int      // (reserved for future; currently not enforced in v2)
    OnMalformed     string   // "ignore" | "forward-raw" | "error-events" (default)
    Debug           bool     // emit debug traces via zerolog
}
```

### Extractors

Extractors register for a `(name, dtype)` pair and produce per-block sessions.

```go
type Extractor interface {
    Name() string
    DataType() string
    NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession
}

type ExtractorSession interface {
    OnStart(ctx context.Context) []events.Event
    OnRaw(ctx context.Context, chunk []byte) []events.Event
    OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event
}
```

### Constructing a sink

```go
next := /* your downstream events.EventSink (collector, bus, etc.) */
sink := structuredsink.NewFilteringSink(next, structuredsink.Options{
    Debug:       false,
    OnMalformed: "error-events",
},
    &myExtractor{name: "citations", dtype: "v1"},
)
```

## YAML Helpers for Streaming Parsers

The `parsehelpers` package provides a debounced YAML parser that plays well with partial text.

```go
type DebounceConfig struct {
    SnapshotEveryBytes int
    SnapshotOnNewline  bool
    ParseTimeout       time.Duration
    MaxBytes           int
}

// NewDebouncedYAML constructs a controller that you feed with bytes as they arrive.
func NewDebouncedYAML[T any](cfg DebounceConfig) *YAMLController[T]

// FeedBytes appends chunk and (when cadence triggers) tries to parse into T.
func (c *YAMLController[T]) FeedBytes(chunk []byte) (*T, error)

// FinalBytes attempts a final parse from the provided raw bytes (if non-empty)
// or from the internal buffer. Code fences are stripped automatically.
func (c *YAMLController[T]) FinalBytes(raw []byte) (*T, error)

// StripCodeFenceBytes returns (lang, body) for ```lang\n ... \n``` blocks.
func StripCodeFenceBytes(b []byte) (string, []byte)
```

Notes:

- Fences are optional. If present, the helper strips them and parses the body. The returned `lang` is lowercased and available if you want to gate on `yaml`, `json`, etc.
- Debouncing lets you emit “best so far” snapshots as the block grows, reducing parse churn.
- Use `MaxBytes` to bound memory, and `ParseTimeout` to guard against pathological parse stalls.

## Building an Extractor (Citations Example)

Below is a trimmed version of the citations extractor that parses YAML as it streams and emits domain events.

```go
type CitationItem struct {
    Title   string
    Authors []string
}

type citationsPayload struct {
    Citations []CitationItem `yaml:"citations"`
}

// Domain events emitted by the extractor (examples)
type EventCitationStarted struct { events.EventImpl; ItemID string }
type EventCitationDelta   struct { events.EventImpl; ItemID,  Delta string }
type EventCitationUpdate  struct { events.EventImpl; ItemID string; Entries []CitationItem; Error string }
type EventCitationCompleted struct { events.EventImpl; ItemID string; Entries []CitationItem; Success bool; Error string }

type citationsExtractor struct{ name, dtype string }
func (ce *citationsExtractor) Name() string     { return ce.name }
func (ce *citationsExtractor) DataType() string { return ce.dtype }
func (ce *citationsExtractor) NewSession(ctx context.Context, meta events.EventMetadata, itemID string) structuredsink.ExtractorSession {
    return &citationsSession{ctx: ctx, itemID: itemID}
}

type citationsSession struct {
    ctx         context.Context
    itemID      string
    lastValid   []CitationItem
    lastValidOK bool
    ctrl        *parsehelpers.YAMLController[citationsPayload]
}

func (cs *citationsSession) OnStart(ctx context.Context) []events.Event {
    cs.ctrl = nil
    return []events.Event{&EventCitationStarted{EventImpl: events.EventImpl{Type_: "citations-started"}, ItemID: cs.itemID}}
}

func (cs *citationsSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
    if cs.ctrl == nil {
        cs.ctrl = parsehelpers.NewDebouncedYAML[citationsPayload](parsehelpers.DebounceConfig{
            SnapshotEveryBytes: 512,
            SnapshotOnNewline:  true,
            MaxBytes:           64 << 10,
        })
    }
    evs := []events.Event{&EventCitationDelta{EventImpl: events.EventImpl{Type_: "citations-delta"}, ItemID: cs.itemID, Delta: string(chunk)}}
    if snap, err := cs.ctrl.FeedBytes(chunk); snap != nil || err != nil {
        var entries []CitationItem
        if err == nil && snap != nil && len(snap.Citations) > 0 {
            entries = snap.Citations
            cs.lastValid, cs.lastValidOK = entries, true
        }
        if len(entries) > 0 || cs.lastValidOK {
            if len(entries) == 0 { entries = cs.lastValid }
            evs = append(evs, &EventCitationUpdate{EventImpl: events.EventImpl{Type_: "citations-update"}, ItemID: cs.itemID, Entries: entries})
        }
    }
    return evs
}

func (cs *citationsSession) OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event {
    entries := cs.lastValid
    if err == nil && raw != nil {
        if cs.ctrl == nil { cs.ctrl = parsehelpers.NewDebouncedYAML[citationsPayload](parsehelpers.DebounceConfig{}) }
        if snap, perr := cs.ctrl.FinalBytes(raw); perr == nil && snap != nil && len(snap.Citations) > 0 {
            entries = snap.Citations
            cs.lastValid, cs.lastValidOK = entries, true
        } else if perr != nil {
            success, err = false, perr
        }
    }
    errStr := ""; if err != nil { errStr = err.Error() }
    return []events.Event{&EventCitationCompleted{EventImpl: events.EventImpl{Type_: "citations-completed"}, ItemID: cs.itemID, Entries: entries, Success: success, Error: errStr}}
}
```

### Why this design works well

- The extractor is fully in control of parsing, debounce cadence, and output model.
- The sink’s responsibilities stay minimal and robust (tag detection and routing).
- You can have multiple extractors covering different `(name, dtype)` pairs in the same stream.

## Wiring Everything Together

Here is a compact end-to-end setup illustrating the sink and the citations extractor. Replace `next` with your own `events.EventSink` implementation (collector, broker, logger, etc.).

```go
collector := &eventCollector{} // implements events.EventSink
ex := &citationsExtractor{name: "citations", dtype: "v1"}

sink := structuredsink.NewFilteringSink(collector, structuredsink.Options{
    Debug:       false,
    OnMalformed: "error-events",
}, ex)

// As text streams in, forward partials/final to the sink:
_ = sink.PublishEvent(&events.EventPartialCompletion{EventImpl: events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta}, Delta: delta})
// ... later ...
_ = sink.PublishEvent(events.NewFinalEvent(meta, finalText))
```

## Error Handling and Policies

- Prefer keeping extractors side-effect free, returning typed events rather than logging. Surface errors via events so UIs can render them.
- Choose `OnMalformed` to match your UX:
  - `error-events` makes incomplete blocks explicit to clients.
  - `forward-raw` is useful when you want to preserve the authored text at all costs.
  - `ignore` keeps the UI clean if partial blocks are expected and unimportant.

## Performance and Resource Tips

- Use `DebounceConfig` to balance responsiveness and parse cost.
- Bound memory with `MaxBytes` in the YAML controller.
- Extractors should preallocate when final size is known and keep transformations pure; log at edges only.
- Streams are cancelled via context when an item completes and when the whole stream completes.

## Testing

- Unit test extractors with representative partial sequences to verify debounced updates and the final parse.
- Include malformed-close test cases to confirm `OnMalformed` handling and error event emission.
- Use the citations demo as a reference for event flows and UI wiring.

## See Also

- Sink implementation: `geppetto/pkg/events/structuredsink/filtering_sink.go`
- YAML helpers: `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go`
- Example app: `geppetto/cmd/examples/citations-event-stream/main.go`


