---
Title: Structured Sinks and the FilteringSink
Slug: geppetto-structured-sinks
Short: Extracting structured data from LLM text streams using tagged blocks and the FilteringSink.
Topics:
- geppetto
- events
- structured-sinks
- architecture
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

# Structured Sinks and the FilteringSink

## The Problem

LLMs produce text. But sometimes you want the model to produce structured data — a mode switch command, a citation list, a planning decision — alongside natural language prose. How do you reliably extract that structured data from a streaming text response?

The naive approach is to parse the final text with regex after inference completes. But this doesn't work for streaming (you want structured events as they arrive), and regex is fragile against model output variations.

## The Solution: Tagged Structured Blocks

Geppetto uses a convention where structured data is embedded in the model's text output, wrapped in XML-like tags with a package/type/version identifier:

~~~text
Here is my analysis of the situation...

<myapp:ModeSwitch:v1>
```yaml
new_mode: research
reason: "Need to gather more information"
```
</myapp:ModeSwitch:v1>

Based on this, I'll switch to research mode.
~~~

The **FilteringSink** watches the streaming text, detects these tagged blocks, extracts their payload, and:

1. Forwards clean prose text downstream (tags removed).
2. Emits typed structured events for programmatic handling.

This turns a single LLM text stream into two outputs: clean user-facing text and machine-readable structured events.

## How It Works

### Architecture

```
LLM Engine
    │ (streaming text events: partial, partial, partial, final)
    ▼
FilteringSink
    ├── Watches for <pkg:type:ver> tags in streaming text
    ├── Extracts payload between open/close tags
    ├── Forwards filtered text (tags removed) to downstream sink
    └── Calls registered Extractors to produce typed events
         │
         ▼
    Downstream EventSink
         (receives both filtered text events and typed structured events)
```

### Tag Format

Tags follow a three-part identifier: `<package:type:version>`

```
Open tag:  <myapp:ModeSwitch:v1>
Close tag: </myapp:ModeSwitch:v1>
```

Each part (package, type, version) can contain alphanumeric characters, underscores, dashes, and dots. The three-part naming prevents collisions between different features.

### Stream Processing

The FilteringSink processes events as they stream, not after completion:

1. **Partial completion events** (`llm.delta`): Each text chunk is scanned for tags. Text outside tags is forwarded immediately. Text inside tags is accumulated as payload.

2. **Final events**: Any remaining text is processed. If a structured block is unclosed at final, the malformed block policy applies.

Because tags can split across streaming chunks (e.g., the open tag `<myapp:M` arrives in one chunk and `odeSwitch:v1>` in the next), the sink maintains a parser state machine per stream.

## The Extractor Interface

Each structured block type is handled by a registered **Extractor**:

```go
type Extractor interface {
    TagPackage() string
    TagType() string
    TagVersion() string
    NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession
}

type ExtractorSession interface {
    OnStart(ctx context.Context) []events.Event
    OnRaw(ctx context.Context, chunk []byte) []events.Event
    OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event
}
```

### Extractor Lifecycle

For each tagged block found in the stream:

1. **Open tag detected** → `Extractor.NewSession()` creates a session for this block instance.
2. **`OnStart(ctx)`** → Called immediately. Return events to emit (e.g., a "started" status event).
3. **`OnRaw(ctx, chunk)`** → Called for each payload chunk as it streams. Return events to emit (e.g., progressive parsing results).
4. **`OnCompleted(ctx, raw, success, err)`** → Called when the close tag is detected (`success=true`) or the block is malformed (`success=false`). Return final events.

Each session gets its own context, derived from the stream context, allowing cancellation detection.

## Setting Up a FilteringSink

```go
import "github.com/go-go-golems/geppetto/pkg/events/structuredsink"

// Create your extractor
citationsExtractor := &MyCitationsExtractor{} // implements Extractor

// Wrap your downstream sink
downstream := /* your EventSink (e.g., WatermillSink) */
sink := structuredsink.NewFilteringSink(downstream, structuredsink.Options{
    Malformed: structuredsink.MalformedErrorEvents,
}, citationsExtractor)

// Use this sink in your context
ctx = events.WithEventSinks(ctx, sink)
```

### Options

```go
type Options struct {
    MaxCaptureBytes int             // Maximum payload size (0 = unlimited)
    Malformed       MalformedPolicy // How to handle unclosed blocks
    Debug           bool            // Enable debug logging
}
```

## Malformed Block Handling

If a structured block is opened but never closed (the model didn't produce the close tag), the sink applies one of three policies:

| Policy | Behavior | When to use |
|--------|----------|-------------|
| `MalformedErrorEvents` | Calls `OnCompleted(false)`, does not reinsert text | Default. Model output errors should be visible. |
| `MalformedReconstructText` | Reinserts the captured text back into the output stream, calls `OnCompleted(false)` | When partial model output should still be visible to the user. |
| `MalformedIgnore` | Drops the captured payload silently, calls `OnCompleted(false)` | When partial blocks should be invisible. |

## Parsing Helpers

The `parsehelpers` package provides utilities for implementing extractors:

### Stripping Code Fences

Models often wrap structured output in YAML code fences. Use `StripCodeFenceBytes` to extract the inner content:

```go
import "github.com/go-go-golems/geppetto/pkg/events/structuredsink/parsehelpers"

lang, body := parsehelpers.StripCodeFenceBytes(rawPayload)
// lang = "yaml", body = the YAML content without fence markers
```

### Debounced YAML Parsing

For progressive parsing during streaming (emit "best-so-far" results before the block is complete):

```go
ctrl := parsehelpers.NewDebouncedYAML[MyPayload](parsehelpers.DebounceConfig{
    SnapshotEveryBytes: 512,   // Parse every 512 bytes
    SnapshotOnNewline:  true,  // Also parse on newlines
    MaxBytes:           64<<10, // 64KB maximum
})

// In OnRaw:
result, ok := ctrl.FeedBytes(chunk)
if ok {
    // Emit a progressive update event with result
}

// In OnCompleted:
result, err := ctrl.FinalBytes(raw)
```

## Integration with Middleware

The structured sink pattern works together with middleware to create composable prompting techniques:

1. **Middleware injects instructions** asking the model to emit structured blocks (e.g., "When you want to switch modes, emit a `<myapp:ModeSwitch:v1>` block with YAML content").

2. **Model produces text** with embedded structured blocks.

3. **FilteringSink extracts** the structured blocks and emits typed events.

4. **Middleware post-processes** by reading the typed events (e.g., updates `Turn.Data` with the new mode).

This separation keeps concerns clean: the middleware handles prompting and action, the sink handles parsing and routing.

## Example: Complete Extractor

```go
type modeSwitchExtractor struct{}

func (e *modeSwitchExtractor) TagPackage() string { return "myapp" }
func (e *modeSwitchExtractor) TagType() string    { return "ModeSwitch" }
func (e *modeSwitchExtractor) TagVersion() string { return "v1" }

func (e *modeSwitchExtractor) NewSession(
    ctx context.Context, meta events.EventMetadata, itemID string,
) ExtractorSession {
    return &modeSwitchSession{meta: meta, itemID: itemID}
}

type modeSwitchSession struct {
    meta   events.EventMetadata
    itemID string
}

func (s *modeSwitchSession) OnStart(ctx context.Context) []events.Event {
    return nil // No start event needed
}

func (s *modeSwitchSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
    return nil // Wait for complete payload
}

func (s *modeSwitchSession) OnCompleted(
    ctx context.Context, raw []byte, success bool, err error,
) []events.Event {
    if !success {
        return nil // Malformed block, skip
    }
    _, body := parsehelpers.StripCodeFenceBytes(raw)

    var payload struct {
        NewMode string `yaml:"new_mode"`
        Reason  string `yaml:"reason"`
    }
    if yamlErr := yaml.Unmarshal(body, &payload); yamlErr != nil {
        return nil
    }

    // Emit a typed event
    return []events.Event{
        events.NewInfoEvent(s.meta, fmt.Sprintf("Mode switch: %s (%s)", payload.NewMode, payload.Reason)),
    }
}
```

## See Also

- [Events](04-events.md) — Event system overview and the `EventSink` interface
- [Middlewares](09-middlewares.md) — How middleware composes with structured sinks
- [Turns and Blocks](08-turns.md) — The Turn data model that middleware operates on
- Implementation: `geppetto/pkg/events/structuredsink/filtering_sink.go`
- Parse helpers: `geppetto/pkg/events/structuredsink/parsehelpers/`
