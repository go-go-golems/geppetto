---
Title: Structured Data Extraction from Streaming Output
Slug: geppetto-tutorial-structured-data-extraction
Short: Build an application that extracts structured data from streaming LLM output in real-time.
Topics:
- geppetto
- tutorial
- structured-data
- streaming
- events
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Structured Data Extraction from Streaming Output

This tutorial teaches you how to extract structured data (like citations, actions, or metadata) from streaming LLM output in real-time. Instead of waiting for the complete response and parsing afterward, you'll receive typed events as structured payloads grow.

## What You'll Build

An application that:
- Streams assistant responses while extracting inline citations
- Emits typed events as citations are parsed progressively
- Strips structured blocks from the visible text stream
- Handles malformed blocks gracefully

## Prerequisites

- Understanding of [Events](../topics/04-events.md)
- Basic familiarity with YAML parsing
- Working streaming setup (see [Streaming Tutorial](01-streaming-inference-with-tools.md))

## Learning Objectives

- Understand the FilteringSink architecture
- Define structured payload schemas
- Implement custom extractors and sessions
- Handle progressive parsing with debouncing
- Wire filtering sinks into your application

## The Problem

LLMs can output structured data inline with prose:

```
Here's what I found in the research:

<geppetto:citations:v1>
```yaml
citations:
  - title: "Attention Is All You Need"
    authors: ["Vaswani et al."]
    year: 2017
```
</geppetto:citations:v1>

The transformer architecture revolutionized NLP...
```

You want to:
1. Show the user "Here's what I found..." and "The transformer architecture..." (without the tags)
2. Extract the citations as structured data
3. Do this in real-time as tokens stream in

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           Engine                                 │
│                              │                                   │
│                              ▼                                   │
│                    ┌─────────────────┐                          │
│                    │ FilteringSink   │                          │
│                    │  ├─ Parser      │                          │
│                    │  └─ Extractors  │                          │
│                    └────────┬────────┘                          │
│                             │                                    │
│           ┌─────────────────┼─────────────────┐                 │
│           ▼                 ▼                 ▼                 │
│    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│    │ Filtered    │  │ Citation    │  │ Citation    │           │
│    │ Text Events │  │ Partial     │  │ Complete    │           │
│    │ (no tags)   │  │ Events      │  │ Events      │           │
│    └─────────────┘  └─────────────┘  └─────────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

## Step 1: Define Your Payload Schema

Create Go types matching your structured data:

```go
package main

// The structured data you're extracting
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

## Step 2: Define Custom Events

Create events for your structured data lifecycle:

```go
import "github.com/go-go-golems/geppetto/pkg/events"

// Emitted progressively as data is parsed
type CitationPartialEvent struct {
    events.EventImpl
    ItemID  string           `json:"item_id"`
    Payload CitationsPayload `json:"payload"`
}

// Emitted when parsing completes
type CitationCompleteEvent struct {
    events.EventImpl
    ItemID  string           `json:"item_id"`
    Payload CitationsPayload `json:"payload"`
    Success bool             `json:"success"`
    Error   string           `json:"error,omitempty"`
}

// Register in init() so they can be deserialized
func init() {
    _ = events.RegisterEventFactory("citation-partial", func() events.Event {
        return &CitationPartialEvent{
            EventImpl: events.EventImpl{Type_: "citation-partial"},
        }
    })
    _ = events.RegisterEventFactory("citation-complete", func() events.Event {
        return &CitationCompleteEvent{
            EventImpl: events.EventImpl{Type_: "citation-complete"},
        }
    })
}
```

## Step 3: Implement the Extractor

The extractor defines which tags to handle and creates sessions:

```go
import (
    "context"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/events/structuredsink"
)

type CitationsExtractor struct{}

// These define the tag: <geppetto:citations:v1>
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

// Verify interface compliance
var _ structuredsink.Extractor = (*CitationsExtractor)(nil)
```

## Step 4: Implement the Extractor Session

The session receives streaming callbacks and returns typed events:

```go
import "gopkg.in/yaml.v3"

type citationsSession struct {
    meta    events.EventMetadata
    itemID  string
    rawBuf  []byte
    lastLen int  // Track when to emit partial events
}

// Called when opening tag is detected
func (s *citationsSession) OnStart(ctx context.Context) []events.Event {
    // Optional: emit a "started" event
    return nil
}

// Called for each chunk of bytes inside the block
func (s *citationsSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
    s.rawBuf = append(s.rawBuf, chunk...)

    // Emit partial event every 256 bytes
    if len(s.rawBuf)-s.lastLen < 256 {
        return nil
    }
    s.lastLen = len(s.rawBuf)

    // Try to parse what we have so far
    var payload CitationsPayload
    _ = yaml.Unmarshal(s.rawBuf, &payload) // Ignore errors for partial data

    return []events.Event{
        &CitationPartialEvent{
            EventImpl: events.EventImpl{
                Type_:     "citation-partial",
                Metadata_: s.meta,
            },
            ItemID:  s.itemID,
            Payload: payload,
        },
    }
}

// Called when closing tag is detected (or on error/malformed)
func (s *citationsSession) OnCompleted(
    ctx context.Context,
    raw []byte,
    success bool,
    err error,
) []events.Event {
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

## Step 5: Create the Filtering Sink Chain

Wire the filtering sink between your engine and downstream handlers:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/events/structuredsink"
    "github.com/go-go-golems/geppetto/pkg/inference/middleware"
)

func createSinkChain(router *events.EventRouter) events.EventSink {
    // Downstream sink connects to router
    downstream := middleware.NewWatermillSink(router.Publisher, "chat")

    // Filtering sink wraps downstream
    filtering := structuredsink.NewFilteringSink(
        downstream,
        structuredsink.Options{
            Malformed: structuredsink.MalformedErrorEvents,
            Debug:     false,
        },
        &CitationsExtractor{}, // Register your extractor
    )

    return filtering
}
```

**Malformed block policies:**

| Policy | Behavior |
|--------|----------|
| `MalformedErrorEvents` | Emit error event, drop block from text |
| `MalformedReconstructText` | Insert raw block back into text stream |
| `MalformedIgnore` | Silently drop the block |

## Step 6: Add the System Prompt

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

Always use this format for citations. The user will see your prose without these blocks.`
```

## Step 7: Handle the Events

Add handlers for your custom events:

```go
import "github.com/ThreeDotsLabs/watermill/message"

func setupHandlers(router *events.EventRouter) {
    // Text streaming (filtered - no tags visible)
    router.AddHandler("printer", "chat", events.StepPrinterFunc("", os.Stdout))

    // Citation event handler
    router.AddHandler("citations", "chat", func(msg *message.Message) error {
        defer msg.Ack()

        ev, err := events.NewEventFromJson(msg.Payload)
        if err != nil {
            return nil
        }

        switch e := ev.(type) {
        case *CitationPartialEvent:
            // Progressive update - could refresh a UI
            fmt.Printf("\r[Found %d citations...]", len(e.Payload.Citations))

        case *CitationCompleteEvent:
            if e.Success {
                fmt.Printf("\n\n=== Extracted Citations ===\n")
                for i, c := range e.Payload.Citations {
                    fmt.Printf("%d. %s (%d)\n", i+1, c.Title, c.Year)
                    if len(c.Authors) > 0 {
                        fmt.Printf("   Authors: %v\n", c.Authors)
                    }
                }
            } else {
                fmt.Printf("\nCitation parsing failed: %s\n", e.Error)
            }
        }

        return nil
    })
}
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

// ... (Citation types, events, extractor, session from above)

func main() {
    ctx := context.Background()

    // 1. Create router
    router, _ := events.NewEventRouter()
    defer router.Close()

    // 2. Add handlers
    router.AddHandler("printer", "chat", events.StepPrinterFunc("", os.Stdout))
    router.AddHandler("citations", "chat", func(msg *message.Message) error {
        defer msg.Ack()
        ev, _ := events.NewEventFromJson(msg.Payload)
        if complete, ok := ev.(*CitationCompleteEvent); ok && complete.Success {
            fmt.Printf("\n\n--- Found %d citations ---\n", len(complete.Payload.Citations))
            for _, c := range complete.Payload.Citations {
                fmt.Printf("• %s (%d)\n", c.Title, c.Year)
            }
        }
        return nil
    })

    // 3. Create filtering sink chain
    downstream := middleware.NewWatermillSink(router.Publisher, "chat")
    filteringSink := structuredsink.NewFilteringSink(
        downstream,
        structuredsink.Options{Malformed: structuredsink.MalformedErrorEvents},
        &CitationsExtractor{},
    )

    // 4. Create engine with filtering sink
    eng, _ := factory.NewEngineFromParsedLayers(parsedLayers, engine.WithSink(filteringSink))

    // 5. Build Turn
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewSystemTextBlock(systemPrompt))
    turns.AppendBlock(turn, turns.NewUserTextBlock(
        "What are the foundational papers on transformer architecture? Include citations.",
    ))

    // 6. Run
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

## Sample Output

**Console (tags stripped):**
```
Here's what I found in the research:

The transformer architecture revolutionized NLP by introducing self-attention
mechanisms that process sequences in parallel rather than sequentially.

--- Found 2 citations ---
• Attention Is All You Need (2017)
• BERT: Pre-training of Deep Bidirectional Transformers (2018)
```

The user sees clean prose while your application receives structured citation data.

## Advanced: Debounced YAML Parsing

For smoother progressive updates, use the parsehelpers package:

```go
import "github.com/go-go-golems/geppetto/pkg/events/structuredsink/parsehelpers"

type citationsSession struct {
    meta   events.EventMetadata
    itemID string
    parser *parsehelpers.DebouncedYAML[CitationsPayload]
}

func (s *citationsSession) OnStart(ctx context.Context) []events.Event {
    s.parser = parsehelpers.NewDebouncedYAML[CitationsPayload](parsehelpers.DebounceConfig{
        SnapshotEveryBytes: 256,   // Emit every 256 bytes
        SnapshotOnNewline:  true,  // Also emit on newlines
        MaxBytes:           64<<10, // 64KB max
    })
    return nil
}

func (s *citationsSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
    result, shouldEmit := s.parser.Feed(chunk)
    if !shouldEmit {
        return nil
    }
    return []events.Event{&CitationPartialEvent{...}}
}
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| Tags visible in output | Extractor not registered | Add extractor to `NewFilteringSink` |
| No extraction events | Tag mismatch | Check `TagPackage/Type/Version` match prompt |
| Partial events not firing | Debounce too high | Lower `SnapshotEveryBytes` |
| Parse errors on complete | Bad YAML format | Check model's output format |
| Wrong sink order | Filtering sink not wrapping downstream | `NewFilteringSink(downstream, ...)` |

## See Also

- [Events](../topics/04-events.md) — Event system reference
- [Progressive Structured Data Playbook](../playbooks/03-progressive-structured-data.md) — Step-by-step guide
- Example: `geppetto/cmd/examples/citations-event-stream/main.go`
- Tests: `geppetto/pkg/events/structuredsink/filtering_sink_test.go`

