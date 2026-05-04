---
title: "Thinking Content Accumulation Bug: Full Analysis and Implementation Guide"
ticket: GP-62
doc_type: design
status: active
topics:
  - openai
  - thinking
  - streaming
  - bug
  - sessionstream
owners:
  - manuel
related_files:
  - path: pkg/steps/ai/openai/engine_openai.go
    note: OpenAI completion engine that emits thinking events during streaming
  - path: pkg/steps/ai/openai/chat_stream.go
    note: SSE stream decoder that extracts reasoning deltas from provider SSE frames
  - path: pkg/events/chat-events.go
    note: Event type definitions for reasoning and thinking events
  - path: 2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go
    note: CoinVault debug feature — must switch from EventReasoningTextDelta to EventThinkingPartial
  - path: pinocchio/cmd/web-chat/reasoning_chat_feature.go
    note: Pinocchio reasoning plugin (correct accumulation pattern using EventThinkingPartial)
  - path: pkg/steps/ai/openai_responses/engine.go
    note: OpenAI Responses engine — must stop emitting EventReasoningTextDelta
  - path: pkg/steps/ai/streamhelpers/reasoning_markdown.go
    note: Reasoning delta normalization for markdown boundaries
---

# GP-62: Thinking Content Not Accumulated in OpenAI Completion API Stream Events

## 1. Executive Summary

When using the OpenAI **chat completions** API (not the Responses API) with a
thinking-capable model (e.g., `wafer-qwen3.5-397b`), the thinking/reasoning
message content displayed in the UI always shows only the **latest delta chunk**
instead of the **accumulated full text**. For example, after streaming hundreds
of tokens, the thinking message entity shows `content: ". I'll"` — just the
most recent delta — rather than the full reasoning text generated so far.

**Root cause:** The system has two overlapping event types for thinking text.
`EventReasoningTextDelta` (delta-only, no accumulation) is a dead-end relic
consumed by exactly one handler, which naively copies the delta as the content.
The correct event — `EventThinkingPartial` (delta + accumulated completion) —
is consumed by everyone else.

**Fix:** Delete `EventReasoningTextDelta`. Switch its single consumer to
`EventThinkingPartial`. Remove all emission sites. This eliminates the bug
and removes a source of ongoing confusion.

This document explains every layer of the system, why the two types existed,
and provides a step-by-step implementation guide for the deletion.

---

## 2. System Architecture Overview

The thinking stream flows through **seven layers** from the LLM provider to
the browser UI. Understanding each layer is essential to diagnosing where the
accumulation breaks down.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        LAYER 1: Provider SSE Stream                      │
│  (wafer.ai / OpenAI-compatible / Together / etc.)                       │
│  SSE frames with delta.reasoning or delta.reasoning_content fields      │
└────────────────────────────┬────────────────────────────────────────────┘
                             │ HTTP SSE (text/event-stream)
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                  LAYER 2: Geppetto OpenAI Engine                         │
│  pkg/steps/ai/openai/engine_openai.go                                   │
│  - Reads SSE frames via chat_stream.go                                  │
│  - Accumulates thinking text in thinkingBuf (strings.Builder)           │
│  - Currently emits TWO event types per reasoning chunk (BUG):           │
│    1. EventReasoningTextDelta ← TO BE DELETED                           │
│    2. EventThinkingPartial    ← KEEP (has delta + completion)           │
│  - Emits InfoEvents: "thinking-started", "thinking-ended"               │
└────────────────────────────┬────────────────────────────────────────────┘
                             │ Go events.Event interface
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│            LAYER 3: Chat Plugins / Event Handlers                       │
│  Two independent consumers:                                             │
│                                                                          │
│  A) CoinVault RuntimeDebugFeature (2026-03-16--gec-rag/...)             │
│     - Handles EventReasoningTextDelta ← MUST SWITCH TO ThinkingPartial │
│     - Publishes to sessionstream with content = ev.Delta ← BUG         │
│                                                                          │
│  B) Pinocchio ReasoningPlugin (pinocchio/cmd/web-chat/...)              │
│     - Handles EventThinkingPartial                                      │
│     - Publishes to sessionstream with content = ev.Completion ✓         │
└────────────────────────────┬────────────────────────────────────────────┘
                             │ sessionstream.Publish()
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LAYER 4: SessionStream (Event Store + Projectors)           │
│  - Stores events in ordered log                                         │
│  - Runs ProjectTimeline() for each event                                │
│  - Produces TimelineEntity snapshots (the "view")                       │
│  - Runs ProjectUI() to produce UIEvents                                 │
└────────────────────────────┬────────────────────────────────────────────┘
                             │ WebSocket / protobuf
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LAYER 5: Browser WebSocket Client                           │
│  - Receives UIEvents and TimelineEntity updates                         │
│  - Updates the timeline registry (registry.ts: timeline.upsert)         │
│  - React components re-render on entity changes                         │
└────────────────────────────┬────────────────────────────────────────────┘
                             │ React state
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LAYER 6: React Chat UI Components                           │
│  - Renders thinking message entity with entity.content as text          │
│  - Shows streaming indicator when entity.streaming === true             │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Detailed Layer-by-Layer Analysis

### 3.1 Layer 1: Provider SSE Stream

**What happens:** The LLM provider (e.g., wafer.ai hosting Qwen 3.5) streams
chat completion responses as Server-Sent Events (SSE). Each SSE `data:` frame
contains a JSON object with a `choices[0].delta` object.

**Key fields in the delta object:**
- `content`: Normal text output (the "saying" text)
- `reasoning` or `reasoning_content`: The thinking/reasoning text delta

**Example SSE frame:**
```
data: {"id":"chatcmpl-abc","choices":[{"index":0,"delta":{"reasoning":". I'll"},"finish_reason":null}]}
```

**File reference:** `pkg/steps/ai/openai/chat_stream.go` — function
`normalizeChatStreamEvent()` extracts these fields:

```go
// Lines ~230-240 (paraphrased)
if s, ok := stringValue(delta["reasoning"]); ok && s != "" {
    ret.DeltaReasoning = s
} else if s, ok := stringValue(delta["reasoning_content"]); ok && s != "" {
    ret.DeltaReasoning = s
}
```

The function normalizes both `reasoning` and `reasoning_content` into a single
`DeltaReasoning` field, so the rest of the system doesn't need to care which
field the provider uses.

### 3.2 Layer 2: Geppetto OpenAI Engine

**File:** `pkg/steps/ai/openai/engine_openai.go`

**What happens:** The `OpenAIEngine.RunInference()` method reads SSE frames in
a loop. For each frame containing a reasoning delta, it:

1. Appends the delta to an internal `thinkingBuf` (a `strings.Builder`)
2. Emits an `EventReasoningTextDelta` with just the delta ← **redundant**
3. Emits an `EventThinkingPartial` with both the delta AND the accumulated completion

**Critical code (lines ~274-281):**

```go
if response.DeltaReasoning != "" {
    if !thinkingStarted {
        thinkingStarted = true
        e.publishEvent(ctx, events.NewInfoEvent(metadata, "thinking-started", nil))
    }
    thinkingBuf.WriteString(streamhelpers.NormalizeReasoningDelta(
        thinkingBuf.String(), response.DeltaReasoning))
    e.publishEvent(ctx, events.NewReasoningTextDelta(metadata, response.DeltaReasoning))
    e.publishEvent(ctx, events.NewThinkingPartialEvent(
        metadata, response.DeltaReasoning, thinkingBuf.String()))
}
```

**Accumulation is correct here.** The `thinkingBuf` grows with each chunk, and
`EventThinkingPartial.Completion` contains the full accumulated text. The
`EventReasoningTextDelta` line is pure duplication — it sends the same delta
but without any accumulated text. **This line should simply be deleted.**

### 3.3 Layer 3A: CoinVault RuntimeDebugFeature (THE BUG)

**File:** `2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go`

**What happens:** This plugin handles `EventReasoningTextDelta` and publishes
it to the sessionstream event store.

**Buggy code (lines ~67-76):**

```go
case *gepevents.EventReasoningTextDelta:
    return true, runtime.Publish(ctx, coinVaultReasoningDeltaUI, map[string]any{
        "messageId":       runtime.MessageID + "-thinking",
        "parentMessageId": runtime.MessageID,
        "role":            "thinking",
        "delta":           ev.Delta,
        "content":         ev.Delta,    // ← BUG: Only the delta!
        "text":            ev.Delta,    // ← BUG: Only the delta!
        "status":          "streaming",
        "streaming":       true,
        "eventType":       string(event.Type()),
    })
```

The `content` field is set to `ev.Delta` (just the latest chunk like `". I'll"`),
not to any accumulated value — because `EventReasoningTextDelta` has no
accumulated value to offer. The event struct only has a `Delta` field.

**This is the only consumer of `EventReasoningTextDelta` in the entire
codebase.** Once we switch it to `EventThinkingPartial` (which has
`ev.Completion`), the bug is fixed and `EventReasoningTextDelta` has zero
remaining consumers.

### 3.4 Layer 3B: Pinocchio ReasoningPlugin (CORRECT REFERENCE)

**File:** `pinocchio/cmd/web-chat/reasoning_chat_feature.go`

This plugin handles `EventThinkingPartial` and does it correctly:

```go
case *gepevents.EventThinkingPartial:
    return true, runtime.Publish(ctx, reasoningDeltaEventName, map[string]any{
        "content":   ev.Completion,  // ✓ Accumulated full text
        "text":      ev.Completion,  // ✓ Accumulated full text
        "chunk":     ev.Delta,       // Just the delta
        // ...
    })
```

This is the correct pattern: `ev.Completion` contains the full accumulated
thinking text. **This is what RuntimeDebugFeature should also use.**

### 3.5 Layer 4: SessionStream Projectors

The sessionstream library runs two projection passes:

1. **ProjectTimeline**: Produces `TimelineEntity` snapshots that accumulate
   state across events. The view parameter gives access to the current entity
   state, so `content += delta` works for accumulation.

2. **ProjectUI**: Produces `UIEvent` payloads that are sent directly to the
   browser. These are NOT accumulated — they are point-in-time snapshots.

**Implication:** If the UIEvent payload has `content = delta`, the browser
receives only the delta. The browser could theoretically accumulate itself,
but currently it does not — it trusts the `content` field from the event.

### 3.6 Layer 5: Browser WebSocket Client

The browser receives `UIEvent` and `TimelineEntity` updates via WebSocket. The
`registry.ts` in the frontend calls `timeline.upsert()` for each entity update.
The console log the user provided shows:

```
registry.ts:131 [sem] timeline.upsert chat-msg-2-thinking
{entity: data: content: ". I'll" delta: ". I'll" eventType: "reasoning-text-delta"}
```

This confirms the entity's `content` field is being set to just the delta on
every update. The `[sem]` prefix indicates this is from the sessionstream
client library.

### 3.7 Layer 6: JS Event Encoder

**File:** `pkg/js/modules/geppetto/api_events.go`

The `encodeEventPayload` function already handles `EventThinkingPartial`
correctly:

```go
case *events.EventThinkingPartial:
    payload["delta"] = e.Delta
    payload["completion"] = e.Completion
```

It does NOT handle `EventReasoningTextDelta` — which is fine because we're
about to delete that type entirely.

---

## 4. Why Two Event Types Existed (And Why We Don't Need Both)

### How we got here

`EventThinkingPartial` (type `"partial-thinking"`) is the **original** event
type, added when thinking support was first built. It follows the same pattern
as `EventPartialCompletion`: a `Delta` for the latest chunk and a `Completion`
for the accumulated text so far.

`EventReasoningTextDelta` (type `"reasoning-text-delta"`) was added later when
the OpenAI **Responses API** was integrated. The Responses API has a specific
server-sent event type called `response.reasoning_text.delta`, and this event
type was created as a 1:1 mapping of that wire protocol event — a raw,
delta-only carrier with no accumulation.

The problem: both OpenAI engines (completions and responses) then started
emitting **both** event types for every reasoning chunk. This created a
redundant dual emission where the "raw" delta event had no accumulated text
and the "cooked" thinking event did.

### Consumer audit

| Consumer | Event Used | Accumulated? |
|---|---|---|
| Pinocchio `ReasoningPlugin` | `EventThinkingPartial` | ✓ Yes (`ev.Completion`) |
| Pinocchio `forwarder.go` | `EventThinkingPartial` | ✓ Yes |
| Pinocchio `timeline_persist.go` | `EventThinkingPartial` | ✓ Yes |
| Pinocchio `backend.go` | `EventThinkingPartial` | ✓ Yes |
| Geppetto JS `api_events.go` | `EventThinkingPartial` | ✓ Yes |
| Geppetto `step-printer-func.go` | `EventThinkingPartial` | ✓ Yes |
| Geppetto `openai-tools` example | `EventThinkingPartial` | ✓ Yes |
| **CoinVault `runtime_debug_feature.go`** | `EventReasoningTextDelta` | ✗ **No** (`ev.Delta`) |

**7 consumers use `EventThinkingPartial` (correct). 1 consumer uses
`EventReasoningTextDelta` (buggy). Zero consumers need the delta-only event.**

The conclusion is clear: `EventReasoningTextDelta` is dead weight. It was a
protocol-mirroring artifact that should never have been emitted alongside
`EventThinkingPartial`.

---

## 5. Root Cause Summary

The bug has a single root cause with a single fix:

**`EventReasoningTextDelta` should not exist.** It was created as a raw
protocol mirror for the Responses API wire format, but it serves no purpose
that `EventThinkingPartial` doesn't already cover — and it lacks the
accumulated `Completion` field that every consumer needs.

The single consumer that uses it (`runtime_debug_feature.go`) does so
incorrectly because the event has no accumulated content to offer. Switching
it to `EventThinkingPartial` fixes the bug immediately.

---

## 6. Implementation Guide: Delete EventReasoningTextDelta

### Step 1: Switch RuntimeDebugFeature to EventThinkingPartial

**File:** `2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go`

Replace the `EventReasoningTextDelta` handler with an `EventThinkingPartial`
handler:

```go
// BEFORE (buggy):
case *gepevents.EventReasoningTextDelta:
    return true, runtime.Publish(ctx, coinVaultReasoningDeltaUI, map[string]any{
        "delta":   ev.Delta,
        "content": ev.Delta,    // ← only the latest chunk
        "text":    ev.Delta,    // ← only the latest chunk
        // ...
    })

// AFTER (fixed):
case *gepevents.EventThinkingPartial:
    return true, runtime.Publish(ctx, coinVaultReasoningDeltaUI, map[string]any{
        "messageId":       runtime.MessageID + "-thinking",
        "parentMessageId": runtime.MessageID,
        "role":            "thinking",
        "delta":           ev.Delta,
        "content":         ev.Completion,  // ✓ accumulated full text
        "text":            ev.Completion,  // ✓ accumulated full text
        "status":          "streaming",
        "streaming":       true,
        "eventType":       "partial-thinking",
    })
```

Note the `eventType` value changes from `"reasoning-text-delta"` to
`"partial-thinking"`. If any downstream consumer keys on this string,
it will need updating. Check the `ProjectTimeline` method in the same file
— it currently matches on `coinVaultReasoningDeltaUI` (an event name constant),
not on `eventType`, so it continues to work.

### Step 2: Remove EventReasoningTextDelta emission from OpenAI engine

**File:** `pkg/steps/ai/openai/engine_openai.go`

Delete one line:

```go
// DELETE THIS LINE:
e.publishEvent(ctx, events.NewReasoningTextDelta(metadata, response.DeltaReasoning))
```

The `EventThinkingPartial` emission on the next line already carries both the
delta and the accumulated text. No information is lost.

### Step 3: Remove EventReasoningTextDelta emission from OpenAI Responses engine

**File:** `pkg/steps/ai/openai_responses/engine.go`

Find all calls to `NewReasoningTextDelta` and delete them. There are two (lines
~485 and ~491):

```go
// DELETE THESE TWO LINES:
e.publishEvent(ctx, events.NewReasoningTextDelta(metadata, d))
// ...
e.publishEvent(ctx, events.NewReasoningTextDelta(metadata, s))
```

Each is already followed by a `NewThinkingPartialEvent` call that covers the
same data point but with the accumulated completion text.

### Step 4: Delete the EventReasoningTextDelta type definition

**File:** `pkg/events/chat-events.go`

Remove:
- The `EventTypeReasoningTextDelta` constant (line ~67)
- The `EventReasoningTextDelta` struct and constructor (lines ~1027-1033)
- The `case EventTypeReasoningTextDelta:` in `NewEventFromJson` (lines ~633-636)

### Step 5: Delete the EventReasoningTextDone type too

While we're cleaning up, `EventReasoningTextDone` (type `"reasoning-text-done"`)
is the companion "done" event for `EventReasoningTextDelta`. Check if it has
any consumers:

```
$ grep -rn "EventReasoningTextDone" --include="*.go" | grep -v chat-events.go
```

If it has no consumers outside `chat-events.go` and the engines that emit it,
delete it along with its constant, struct, constructor, and `NewEventFromJson`
case. The `"thinking-ended"` info event already serves the same lifecycle
purpose.

### Step 6: Verify compilation and run tests

```bash
cd geppetto && go build ./...
cd pinocchio && go build ./...
cd 2026-03-16--gec-rag && go build ./...
```

Run existing tests:

```bash
cd geppetto && go test ./pkg/events/... ./pkg/steps/ai/openai/... ./pkg/steps/ai/openai_responses/...
cd pinocchio && go test ./cmd/web-chat/... ./pkg/ui/...
```

### Step 7: Manual smoke test

1. Run pinocchio with the `wafer-qwen3.5-397b` profile
2. Send a message that triggers extended thinking
3. Open browser dev tools → WebSocket tab
4. Verify that reasoning delta UI events now have `content` equal to
   the accumulated text, not just the delta
5. Verify the thinking message in the UI shows progressively growing text

---

## 7. Testing Plan

### 7.1 Unit Test: Verify EventThinkingPartial carries accumulated text

This test already works — `EventThinkingPartial` has always had `Completion`.
No new test needed for the event itself.

### 7.2 Integration Test: Engine emits only EventThinkingPartial

```go
func TestOpenAIEngineEmitsOnlyThinkingPartial(t *testing.T) {
    // Setup mock SSE server that returns reasoning deltas
    // "Hello", " world", " from", " Qwen"
    // Collect all events from the engine

    // Assert NO EventReasoningTextDelta events received
    // Assert EventThinkingPartial events received with:
    //   Event 1: Delta="Hello", Completion="Hello"
    //   Event 2: Delta=" world", Completion="Hello world"
    //   Event 3: Delta=" from", Completion="Hello world from"
    //   Event 4: Delta=" Qwen", Completion="Hello world from Qwen"
}
```

### 7.3 Integration Test: RuntimeDebugFeature uses Completion

```go
func TestRuntimeDebugFeaturePublishesAccumulatedContent(t *testing.T) {
    feature := NewRuntimeDebugFeature()
    // Simulate three EventThinkingPartial events
    // Verify each published sessionstream event has:
    //   content = ev.Completion (accumulated), NOT ev.Delta (latest chunk)
}
```

### 7.4 Manual Smoke Test

1. Run pinocchio with the `wafer-qwen3.5-397b` profile
2. Send a message that triggers extended thinking
3. Open browser dev tools → WebSocket tab
4. Verify that each `CoinVaultReasoningDelta` UI event has `content` equal to
   the accumulated text, not just the delta
5. Verify the thinking message in the UI shows progressively growing text

---

## 8. File Reference Map

| File | Role | What to change |
|---|---|---|
| `pkg/steps/ai/openai/engine_openai.go` | OpenAI completion engine | Delete `NewReasoningTextDelta` line (~279) |
| `pkg/steps/ai/openai/chat_stream.go` | SSE stream decoder | No changes (extraction is correct) |
| `pkg/events/chat-events.go` | Event type definitions | Delete `EventReasoningTextDelta`, `EventReasoningTextDone`, their constants, constructors, and `NewEventFromJson` cases |
| `2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go` | CoinVault debug feature | Switch from `EventReasoningTextDelta` to `EventThinkingPartial`, use `ev.Completion` |
| `pinocchio/cmd/web-chat/reasoning_chat_feature.go` | Pinocchio reasoning plugin | No changes (already correct) |
| `pinocchio/pkg/ui/forwarders/agent/forwarder.go` | Agent UI forwarder | No changes (already uses `EventThinkingPartial`) |
| `pkg/js/modules/geppetto/api_events.go` | JS event encoder | No changes (already handles `EventThinkingPartial`) |
| `pkg/steps/ai/streamhelpers/reasoning_markdown.go` | Reasoning normalization | No changes |
| `pkg/steps/ai/openai_responses/engine.go` | OpenAI Responses engine | Delete two `NewReasoningTextDelta` calls (~485, ~491) |
| `pkg/events/step-printer-func.go` | Step printer | No changes (already uses `EventThinkingPartial`) |

---

## 9. Pseudocode: The Correct Data Flow (After Fix)

After the fix, the flow is clean and single-path:

```
Provider SSE: data: {"delta": {"reasoning": " I'll"}}

→ chat_stream.go: normalizeChatStreamEvent()
    → chatStreamEvent{DeltaReasoning: " I'll"}

→ engine_openai.go: RunInference()
    thinkingBuf.WriteString(" I'll")  // thinkingBuf now = "...previous text... I'll"

    // Only ONE event emitted:
    publishEvent(EventThinkingPartial{
        Delta:      " I'll",
        Completion: "...previous text... I'll",
    })

→ runtime_debug_feature.go: HandleRuntimeEvent()
    // Now handles EventThinkingPartial (like everyone else)
    publish("CoinVaultReasoningDelta", {
        delta:   " I'll",
        content: "...previous text... I'll",    // ✓ accumulated
        text:    "...previous text... I'll",    // ✓ accumulated
    })

→ sessionstream: ProjectTimeline()
    entity.content = payload.content  // = "...previous text... I'll" ✓

→ WebSocket → Browser → UI
    entity.content shows accumulated text ✓
```

---

## 10. Diagram: Event Flow Before and After Fix

### Before Fix (BROKEN — dual emission)

```
Provider ──SSE──► engine_openai ──► EventReasoningTextDelta {Delta: " I'll"}
                   " I'll"          (no Completion!)
                         │
                         ├──────────► EventThinkingPartial {Delta: " I'll", Completion: "... I'll"}
                         │                   │
                         │                   └──► pinocchio (7 consumers) ✓
                         │
                         └──► runtime_debug_feature.go (1 consumer)
                              publish({content: " I'll"})  ← BUG: just delta
                                       │
                                       ▼
                              sessionstream ──WS──► Browser
                              {content: " I'll"}  ← shows only delta
```

### After Fix (CLEAN — single emission)

```
Provider ──SSE──► engine_openai ──► EventThinkingPartial {Delta: " I'll", Completion: "... I'll"}
                   " I'll"                    │
                                              ├──► pinocchio (7 consumers) ✓
                                              │
                                              └──► runtime_debug_feature.go
                                                   publish({content: "... I'll"})  ✓
                                                            │
                                                            ▼
                                                   sessionstream ──WS──► Browser
                                                   {content: "... I'll"}  ✓ accumulated
```

---

## 11. API Reference: The Single Event Type

### EventThinkingPartial (the one we keep)

```go
type EventThinkingPartial struct {
    EventImpl                    // Type, Metadata, Error, payload
    Delta      string `json:"delta"`       // Latest thinking text chunk
    Completion string `json:"completion"`  // Full accumulated thinking text
}

func NewThinkingPartialEvent(metadata EventMetadata, delta string, completion string) *EventThinkingPartial {
    return &EventThinkingPartial{
        EventImpl:  EventImpl{Type_: EventTypePartialThinking, Metadata_: metadata},
        Delta:      delta,
        Completion: completion,
    }
}
```

**Event type constant:** `EventTypePartialThinking = "partial-thinking"`

### EventImpl (embedded in all events)

```go
type EventImpl struct {
    Type_     EventType
    Error_    error
    Metadata_ EventMetadata
    payload   []byte
}
```

### EventMetadata

```go
type EventMetadata struct {
    LLMInferenceData
    ID          uuid.UUID
    SessionID   string
    InferenceID string
    TurnID      string
    Extra       map[string]interface{}
}
```

### What gets deleted

```go
// DELETE: constant
EventTypeReasoningTextDelta EventType = "reasoning-text-delta"

// DELETE: struct
type EventReasoningTextDelta struct {
    EventImpl
    Delta string `json:"delta"`
}

// DELETE: constructor
func NewReasoningTextDelta(metadata EventMetadata, delta string) *EventReasoningTextDelta { ... }

// DELETE: companion done event (if no consumers)
EventTypeReasoningTextDone  EventType = "reasoning-text-done"
type EventReasoningTextDone struct { ... }
func NewReasoningTextDone(...) { ... }
```

---

## 12. SessionStream Projector Patterns

For the intern: understanding how sessionstream projectors work is essential
for verifying the fix doesn't break the timeline.

### ProjectTimeline Pattern

```go
func (f *Feature) ProjectTimeline(
    ctx context.Context,
    ev sessionstream.Event,       // The raw event from the store
    session *sessionstream.Session, // Current session state
    view sessionstream.TimelineView, // Current entity view (for accumulation)
) ([]sessionstream.TimelineEntity, bool, error) {
    // 1. Get the current entity from the view (if it exists)
    entity := currentTimelineEntityPayload(view, "message", id)

    // 2. For EventThinkingPartial, the payload already has accumulated content
    //    so we can use it directly instead of doing our own accumulation:
    content := projectionStringValue(payload["content"])

    // 3. Write to entity
    entity["content"] = content

    // 4. Return as TimelineEntity
    return []sessionstream.TimelineEntity{{
        Kind:    "message",
        Id:      id,
        Payload: pb,
    }}, true, nil
}
```

**Key insight:** Because `EventThinkingPartial` carries the accumulated
`Completion`, the projector doesn't need to do its own `content += delta`
accumulation. It can trust the payload's `content` field directly.

### ProjectUI Pattern

```go
func (f *Feature) ProjectUI(
    ctx context.Context,
    ev sessionstream.Event,
    session *sessionstream.Session,
    view sessionstream.TimelineView,
) ([]sessionstream.UIEvent, bool, error) {
    // UIEvents are sent directly to the browser.
    // They do NOT go through accumulation on the server side.
    // Whatever is in the payload is what the browser sees.
    return []sessionstream.UIEvent{{
        Name:    "CoinVaultReasoningDelta",
        Payload: pb,
    }}, true, nil
}
```

**Key insight:** UIEvents are point-in-time snapshots. The browser receives
them as-is. This is why having `content = ev.Completion` (accumulated) rather
than `content = ev.Delta` (latest chunk) is critical.

---

## 13. Checklist for the Fix

- [ ] Switch `runtime_debug_feature.go` from `EventReasoningTextDelta` to `EventThinkingPartial`, use `ev.Completion` for content
- [ ] Delete `NewReasoningTextDelta` call in `engine_openai.go` (line ~279)
- [ ] Delete both `NewReasoningTextDelta` calls in `openai_responses/engine.go` (lines ~485, ~491)
- [ ] Delete `EventTypeReasoningTextDelta` constant from `chat-events.go`
- [ ] Delete `EventReasoningTextDelta` struct + constructor from `chat-events.go`
- [ ] Delete `case EventTypeReasoningTextDelta:` from `NewEventFromJson` in `chat-events.go`
- [ ] Audit and delete `EventReasoningTextDone` if no consumers remain
- [ ] Verify compilation across geppetto, pinocchio, coinvault
- [ ] Run existing test suites
- [ ] Manual smoke test with `wafer-qwen3.5-397b` profile
- [ ] Update GP-62 changelog

---

## 14. Glossary

- **SSE (Server-Sent Events):** HTTP streaming protocol where the server sends
  `data:` lines. Used by all OpenAI-compatible APIs for streaming responses.
- **Thinking/Reasoning:** Extended chain-of-thought text that the model
  generates before producing its final answer. Some providers call it
  "reasoning", others "thinking".
- **Delta:** An incremental text chunk in a streaming response. The provider
  sends one delta per SSE frame.
- **Completion:** The full accumulated text so far (all deltas concatenated).
- **SessionStream:** The session event store and projector framework used by
  pinocchio. Events are stored in order, and projectors produce timeline
  entities and UI events from the event log.
- **TimelineEntity:** A snapshot of an entity in the session timeline (e.g., a
  chat message, a thinking block, a tool call). Updated by `ProjectTimeline`.
- **UIEvent:** A point-in-time event sent to the browser. Produced by
  `ProjectUI`.
- **ChatPlugin:** A pinocchio extension that can handle geppetto events and
  publish sessionstream events. Examples: `ReasoningPlugin`, `RuntimeDebugFeature`.
