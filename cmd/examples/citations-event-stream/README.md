# Citations Event Stream Demo

Interactive bubbletea demo showcasing the `FilteringSink` structured data extraction middleware.

## What it demonstrates

**Two-row layout with three columns each:**

**Top Row - Content Streams:**
- **Left (Raw Stream)**: Complete raw text including YAML fragments and tags
- **Middle (Filtered Text)**: User-visible text with structured blocks removed
- **Right (Citations)**: Extracted citations with incremental updates

**Bottom Row - Event Streams:**
- **Left (Input Events)**: Raw partial/final events sent to the sink
- **Middle (Filtered Events)**: Filtered partial/final events emitted by the sink
- **Right (Citation Events)**: Typed citation-delta and citation-update events

This layout shows the complete event flow:
1. Input events arrive with raw text
2. FilteringSink processes and filters structured blocks
3. Filtered text events are forwarded to UI
4. Typed citation events are emitted in parallel

- **Incremental parsing**: Citations update as YAML grows character-by-character
- **Event-level filtering**: The `<$citations:v1>```yaml...```</$citations:v1>` block is filtered from the user-visible text stream while emitting typed citation events

## Running

```bash
cd geppetto/cmd/examples/citations-event-stream
go run main.go
```

## Controls

- **Space**: Toggle auto-play mode
- **n** or **→**: Step forward one chunk
- **+/-**: Adjust auto-play speed
- **r**: Reset to beginning
- **q**: Quit

## How it works

1. A simulated stream contains text with an embedded structured block
2. The `FilteringSink` wraps a recording sink and registers a `citationsExtractor`
3. As partials arrive, the sink:
   - Filters the structured block from the visible text
   - Emits `citations-delta` and `citations-update` events with parsed entries
4. The UI displays both streams in real-time

This pattern enables LLM-driven custom data extraction without showing the embedded YAML to end-users.

