Title: Structured metadata in LLM events and how to pass it to the Bobatea timeline UI
Short: Avoid brittle map parsing in `LLMTextModel` by passing typed metadata from `events.EventMetadata`/`conversation.LLMMessageMetadata` through timeline props
Topics:
- geppetto
- bobatea
- timeline
- metadata
- bubbletea
IsTemplate: false
IsTopLevel: false
ShowPerDefault: false
SectionType: Report

## Purpose

`LLMTextModel` currently extracts engine/model/temperature/usage from a loose `map[string]any` under `props["metadata"]`, requiring multiple fallback paths. This report analyzes how to robustify metadata by leveraging the typed structures already present in Geppetto (`events.EventMetadata`, `conversation.LLMMessageMetadata`) and how to pass them cleanly to the Bobatea timeline models.

## What exists today

- Typed metadata in Geppetto:
  - `conversation.LLMMessageMetadata` with fields like `Engine`, `Temperature`, `Usage{InputTokens, OutputTokens}`.
  - `events.EventMetadata` embeds `LLMMessageMetadata` and adds correlation IDs and `Extra`.
    ```go
    // 295+:geppetto/pkg/events/chat-events.go
    type EventMetadata struct {
        conversation.LLMMessageMetadata
        ID conversation.NodeID
        RunID, TurnID string
        Extra map[string]interface{}
    }
    ```

- Data flow for LLM text in the chat app:
  - The backend (fake or real) sends stream messages (`StreamStartMsg`, `StreamCompletionMsg`, `StreamDoneMsg`) that may include `EventMetadata`.
  - The chat model translates these messages into timeline entity lifecycle events for kind `llm_text`:
    ```go
    // 419-446:bobatea/pkg/chat/model.go
    m.timelineCtrl.OnCreated(timeline.UIEntityCreated{ ID: {..., Kind: "llm_text"}, Renderer: {Kind: "llm_text"}, Props: {"role":"assistant","text":""}, StartedAt: time.Now() })
    m.timelineCtrl.OnUpdated(timeline.UIEntityUpdated{ ID: {...}, Patch: {"text": v.Completion}, ... })
    m.timelineCtrl.OnCompleted(timeline.UIEntityCompleted{ ID: {...}, Result: {"text": v.Completion} })
    ```
  - Today, timeline patches do not carry typed metadata; `LLMTextModel` attempts to reconstruct metadata from maps.

- Fake backend:
  - Emits tool-call timeline entities directly.
  - Emits stream messages (for LLM text) but does not populate `EventMetadata` (no engine/usage).

## Problem

- UI model must parse various ad-hoc shapes in `props["metadata"]` (maps, nested maps), which is brittle and costly.

## Recommendation: Pass typed metadata through timeline props

1) Attach typed metadata to timeline `llm_text` entities as-is:
   - When translating stream messages in the chat model, include an additional `metadata` field in `Props`/`Patch`/`Result`:
   ```go
   // Pseudocode in bobatea/pkg/chat/model.go
   func toUIMetadata(ev *events.EventMetadata) any { return ev } // pass typed when available

   case conversationui.StreamStartMsg:
       m.timelineCtrl.OnCreated(timeline.UIEntityCreated{ 
           ID: id(llm_text), Renderer: {Kind:"llm_text"}, 
           Props: map[string]any{"role":"assistant","text":"", "metadata": toUIMetadata(v.EventMetadata)},
           StartedAt: time.Now(),
       })
   case conversationui.StreamCompletionMsg:
       m.timelineCtrl.OnUpdated(timeline.UIEntityUpdated{
           ID: id(llm_text), Patch: map[string]any{"text": v.Completion, "metadata": toUIMetadata(v.EventMetadata)}, ...,
       })
   case conversationui.StreamDoneMsg:
       m.timelineCtrl.OnCompleted(timeline.UIEntityCompleted{
           ID: id(llm_text), Result: map[string]any{"text": v.Completion, "metadata": toUIMetadata(v.EventMetadata)},
       })
   ```

2) In `LLMTextModel`, prefer typed assertions before any map inspection:
   ```go
   // Pseudocode inside model
   if mdRaw, ok := patch["metadata"]; ok {
       switch md := mdRaw.(type) {
       case *events.EventMetadata:
           // Use md.LLMMessageMetadata.Engine, md.LLMMessageMetadata.Usage, etc.
       case events.EventMetadata:
           // same
       case *conversation.LLMMessageMetadata:
           // use directly
       case conversation.LLMMessageMetadata:
           // use directly
       case map[string]any:
           // fallback legacy parsing
       }
   }
   ```

This avoids any reliance on map keys in the standard case while maintaining backward compatibility.

## Alternative: Introduce a Bobatea-native metadata struct

To decouple the UI layer from Geppetto types, define a tiny `timeline.LLMMeta` struct in Bobatea:
```go
// bobatea/pkg/timeline/types.go (example)
type LLMMeta struct {
    Engine string
    Model  string
    Temperature *float64
    Usage struct{ InputTokens, OutputTokens int }
}
```
Provide adapter helpers:
```go
func FromEventMetadata(em *events.EventMetadata) *timeline.LLMMeta { ... }
func FromLLMMessageMetadata(mm *conversation.LLMMessageMetadata) *timeline.LLMMeta { ... }
```
Then always pass a `*timeline.LLMMeta` instance in `props["metadata"]`. The model does a single type assertion (`*timeline.LLMMeta`) and renders it directly. This reduces coupling in Bobatea while still avoiding map parsing.

Trade-offs:
- Direct use of Geppetto types: simplest in a monorepo, zero-copy. Tighter coupling.
- Bobatea-native struct: cleaner UI boundary, small copy/conversion cost.

## Fake backend metadata

For demos, populate basic metadata to exercise the UI rendering:
```go
md := events.EventMetadata{
    LLMMessageMetadata: conversation.LLMMessageMetadata{
        Engine: "fake-engine", Temperature: ptr(0.2),
        Usage: &conversation.Usage{InputTokens: 12, OutputTokens: 34},
    },
    // IDs optional
}
metadata := conversationui.StreamMetadata{ ID: conversation2.NewNodeID(), EventMetadata: &md }
```

## Backward compatibility

- Keep the current map-based fallback for third-party injectors that pass plain maps.
- Prefer typed values whenever available; they bypass the fallback.

## Suggested implementation steps

1) Chat model: include `metadata` in timeline patches for `llm_text` from `Stream*` messages.
2) LLMTextModel: add type assertions for `events.EventMetadata` and `conversation.LLMMessageMetadata` before any map parsing.
3) Optional: define `timeline.LLMMeta` and adapter functions; update chat model to pass that instead.
4) Fake backend: set `EventMetadata` in `StreamMetadata` for demo coverage.

## Expected impact

- Eliminates brittle key-walking in `LLMTextModel` for standard flows.
- Improves performance and readability of the UI model.
- Provides a clear, typed boundary between Geppetto events and UI rendering.




