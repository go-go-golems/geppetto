---
Title: Progressive Structured Data Extraction
Slug: geppetto-playbook-progressive-structured-data
Short: Step-by-step guide to extract structured data from streaming LLM output using filtering sinks and custom extractors.
Topics:
- geppetto
- events
- structured-data
- playbook
- streaming
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Progressive Structured Data Extraction

This playbook walks through extracting structured data from streaming LLM output. Instead of waiting for the full response, you'll receive typed events as structured payloads grow — enabling real-time UI updates and progressive parsing.

## Use Cases

- **Citations**: Extract references as the model mentions them
- **Actions**: Capture tool-like structured commands inline
- **Metadata**: Pull out structured annotations from prose
- **Forms**: Progressively validate structured input

## Prerequisites

- A working Geppetto streaming setup (see [Events](../topics/04-events.md))
- Understanding of the filtering sink architecture

## Concept

The model outputs structured blocks inline with text:

```
Here's what I found:

<geppetto:citations:v1>
```yaml
citations:
  - title: "GPT-4 Technical Report"
    authors: ["OpenAI"]
    year: 2023
```
</geppetto:citations:v1>

The report discusses...
```

The `FilteringSink` intercepts these blocks, routes raw bytes to your extractor, and emits typed events while stripping the tags from the text stream.

## Steps

### Step 1: Define Your Payload Schema

Create Go types for your structured data:

```go
package main

// Your structured payload
type Citation struct {
    Title   string   `yaml:"title" json:"title"`
    Authors []string `yaml:"authors" json:"authors"`
    Year    int      `yaml:"year" json:"year"`
    URL     string   `yaml:"url,omitempty" json:"url,omitempty"`
}

type CitationsPayload struct {
    Citations []Citation `yaml:"citations" json:"citations"`
}
```

### Step 2: Define Custom Events

Create events for your structured data:

```go
import "github.com/go-go-golems/geppetto/pkg/events"

// Event emitted as citations are parsed progressively
type CitationPartialEvent struct {
    events.EventImpl
    ItemID   string           `json:"item_id"`
    Payload  CitationsPayload `json:"payload"`
    IsFinal  bool             `json:"is_final"`
}

// Event emitted when parsing completes
type CitationCompleteEvent struct {
    events.EventImpl
    ItemID   string           `json:"item_id"`
    Payload  CitationsPayload `json:"payload"`
    Success  bool             `json:"success"`
    Error    string           `json:"error,omitempty"`
}

// Register in init()
func init() {
    _ = events.RegisterEventFactory("citation-partial", func() events.Event {
        return &CitationPartialEvent{EventImpl: events.EventImpl{Type_: "citation-partial"}}
    })
    _ = events.RegisterEventFactory("citation-complete", func() events.Event {
        return &CitationCompleteEvent{EventImpl: events.EventImpl{Type_: "citation-complete"}}
    })
}
```

### Step 3: Implement the Extractor

An extractor defines the tag triple and creates sessions:

```go
import (
    "context"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/events/structuredsink"
)

type CitationsExtractor struct{}

// Tag triple: <geppetto:citations:v1>
func (e *CitationsExtractor) TagPackage() string { return "geppetto" }
func (e *CitationsExtractor) TagType() string    { return "citations" }
func (e *CitationsExtractor) TagVersion() string { return "v1" }

func (e *CitationsExtractor) NewSession(
    ctx context.Context, 
    meta events.EventMetadata, 
    itemID string,
) structuredsink.ExtractorSession {
    return &citationsSession{
        meta:   meta,
        itemID: itemID,
    }
}

var _ structuredsink.Extractor = (*CitationsExtractor)(nil)
```

### Step 4: Implement the Extractor Session

The session receives streaming callbacks and returns typed events:

```go
import (
    "context"
    "gopkg.in/yaml.v3"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/events/structuredsink/parsehelpers"
)

type citationsSession struct {
    meta    events.EventMetadata
    itemID  string
    parser  *parsehelpers.DebouncedYAML[CitationsPayload]
    rawBuf  []byte
}

func (s *citationsSession) OnStart(ctx context.Context) []events.Event {
    // Initialize debounced parser for progressive updates
    s.parser = parsehelpers.NewDebouncedYAML[CitationsPayload](parsehelpers.DebounceConfig{
        SnapshotEveryBytes: 256,  // Emit event every 256 bytes
        SnapshotOnNewline:  true, // Also emit on newlines
        MaxBytes:           64 << 10, // 64KB max
    })
    return nil // No start event needed
}

func (s *citationsSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
    s.rawBuf = append(s.rawBuf, chunk...)
    
    // Try to parse progressively
    result, shouldEmit := s.parser.Feed(chunk)
    if !shouldEmit {
        return nil
    }
    
    // Emit partial event with best-effort parse
    return []events.Event{
        &CitationPartialEvent{
            EventImpl: events.EventImpl{
                Type_:     "citation-partial",
                Metadata_: s.meta,
            },
            ItemID:  s.itemID,
            Payload: result,
            IsFinal: false,
        },
    }
}

func (s *citationsSession) OnCompleted(
    ctx context.Context, 
    raw []byte, 
    success bool, 
    err error,
) []events.Event {
    // Parse final payload
    var payload CitationsPayload
    var parseErr string
    
    if success {
        if e := yaml.Unmarshal(raw, &payload); e != nil {
            parseErr = e.Error()
            success = false
        }
    } else if err != nil {
        parseErr = err.Error()
    }
    
    return []events.Event{
        &CitationCompleteEvent{
            EventImpl: events.EventImpl{
                Type_:     "citation-complete",
                Metadata_: s.meta,
            },
            ItemID:  s.itemID,
            Payload: payload,
            Success: success,
            Error:   parseErr,
        },
    }
}
```

### Step 5: Wire Up the Filtering Sink

Create the filtering sink and chain it with your downstream sink:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/events/structuredsink"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
)

// Create downstream sink (connects to router)
downstreamSink := middleware.NewWatermillSink(router.Publisher, "chat")

// Wrap with filtering sink
filteringSink := structuredsink.NewFilteringSink(
    downstreamSink,
    structuredsink.Options{
        Malformed: structuredsink.MalformedErrorEvents, // Emit error events on parse failure
        Debug:     false,
    },
    &CitationsExtractor{}, // Register your extractor
)
```

### Step 6: Use the Filtering Sink with Engine

Create the engine normally and attach the filtering sink to context at runtime:

```go
eng, err := factory.NewEngineFromParsedLayers(parsedLayers)
if err != nil {
    return err
}
```

### Step 7: Add System Instructions

Tell the model how to emit structured blocks:

```go
const systemPrompt = `You are a research assistant. When citing sources, use this exact format:

<geppetto:citations:v1>
` + "```yaml" + `
citations:
  - title: "Paper Title"
    authors: ["Author Name"]
    year: 2023
    url: "https://..."
` + "```" + `
</geppetto:citations:v1>

Always use this format for citations.`

turn := &turns.Turn{}
turns.AppendBlock(turn, turns.NewSystemTextBlock(systemPrompt))
turns.AppendBlock(turn, turns.NewUserTextBlock("What are the key papers on transformer architecture?"))
```

### Step 8: Handle the Events

Add handlers for your custom events:

```go
router.AddHandler("citations", "chat", func(msg *message.Message) error {
    defer msg.Ack()
    ev, _ := events.NewEventFromJson(msg.Payload)
    
    switch e := ev.(type) {
    case *CitationPartialEvent:
        // Progressive update - refresh UI
        fmt.Printf("Found %d citations so far...\n", len(e.Payload.Citations))
        
    case *CitationCompleteEvent:
        if e.Success {
            fmt.Printf("Extracted %d citations:\n", len(e.Payload.Citations))
            for _, c := range e.Payload.Citations {
                fmt.Printf("  - %s (%d)\n", c.Title, c.Year)
            }
        } else {
            fmt.Printf("Citation parsing failed: %s\n", e.Error)
        }
    }
    return nil
})
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/events/structuredsink"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/turns"
    "golang.org/x/sync/errgroup"
    "gopkg.in/yaml.v3"
)

// ... (Citation types and events from above)

func main() {
    ctx := context.Background()
    
    // 1. Create router
    router, _ := events.NewEventRouter()
    defer router.Close()
    
    // 2. Add text printer
    router.AddHandler("printer", "chat", events.StepPrinterFunc("", os.Stdout))
    
    // 3. Add citation handler
    router.AddHandler("citations", "chat", func(msg *message.Message) error {
        defer msg.Ack()
        ev, _ := events.NewEventFromJson(msg.Payload)
        if complete, ok := ev.(*CitationCompleteEvent); ok && complete.Success {
            fmt.Printf("\n--- Extracted %d citations ---\n", len(complete.Payload.Citations))
        }
        return nil
    })
    
    // 4. Create filtering sink chain
    downstreamSink := middleware.NewWatermillSink(router.Publisher, "chat")
    filteringSink := structuredsink.NewFilteringSink(
        downstreamSink,
        structuredsink.Options{Malformed: structuredsink.MalformedErrorEvents},
        &CitationsExtractor{},
    )
    
    // 5. Create engine (no engine options/sinks at construction time)
    eng, _ := factory.NewEngineFromParsedLayers(parsedLayers)
    
    // 6. Build Turn with instructions
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewSystemTextBlock(systemPrompt))
    turns.AppendBlock(turn, turns.NewUserTextBlock("Summarize key NLP papers from 2023"))
    
    // 7. Run
    eg, groupCtx := errgroup.WithContext(ctx)
    eg.Go(func() error { return router.Run(groupCtx) })
    eg.Go(func() error {
        <-router.Running()
        _, err := eng.RunInference(events.WithEventSinks(groupCtx, filteringSink), turn)
        return err
    })
    _ = eg.Wait()
}
```

## Malformed Block Policies

Control what happens when a structured block is malformed:

| Policy | Behavior |
|--------|----------|
| `MalformedErrorEvents` | Emit error event, don't include in text |
| `MalformedReconstructText` | Insert raw block back into text stream |
| `MalformedIgnore` | Silently drop the block |

```go
structuredsink.Options{
    Malformed: structuredsink.MalformedReconstructText, // Show broken blocks as text
}
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| No structured events | Tag mismatch | Check `TagPackage/Type/Version` match prompt format |
| Events not reaching handler | Sink order wrong | Filtering sink must wrap downstream sink |
| Partial events missing | Debounce too high | Lower `SnapshotEveryBytes` in config |
| Parse errors | YAML formatting | Ensure model uses proper YAML indentation |
| Tags appear in output | No extractor registered | Register extractor for that tag triple |

## See Also

- [Events](../topics/04-events.md) — Event system reference
- Example: `geppetto/cmd/examples/citations-event-stream/main.go`
- Tests: `geppetto/pkg/events/structuredsink/filtering_sink_test.go`
