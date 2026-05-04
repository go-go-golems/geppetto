---
title: "Shared ChatPlugin Architecture: Analysis and Implementation Guide"
ticket: GP-63
doc_type: design
status: active
topics:
  - chatapp
  - plugins
  - refactoring
  - reuse
owners:
  - manuel
related_files:
  - path: pinocchio/pkg/chatapp/features.go
    note: ChatPlugin interface definition
  - path: pinocchio/pkg/chatapp/chat.go
    note: Engine that runs ChatPlugins — runtimeEventSink, base projectors, event names
  - path: pinocchio/cmd/web-chat/reasoning_chat_feature.go
    note: Current working reasoning plugin (to be moved to pkg)
  - path: pinocchio/cmd/web-chat/agentmode_chat_feature.go
    note: Agent mode plugin (stays in cmd for now)
  - path: 2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go
    note: CoinVault's buggy reimplementation of reasoning + tool calls (to be replaced)
  - path: pinocchio/pkg/ui/forwarders/agent/forwarder.go
    note: Old TUI forwarder — does the same thing as plugins but for bubbletea
  - path: pinocchio/pkg/chatapp/pb/proto/pinocchio/chatapp/v1/chat.pb.go
    note: Proto types: ChatMessageUpdate, ChatMessageEntity
  - path: 2026-03-16--gec-rag/internal/pb/proto/coinvault/webchat/v1/runtime_debug.pb.go
    note: CoinVault proto types: CoinVaultToolCall, CoinVaultToolResult, CoinVaultReasoningDelta
---

# GP-63: Extract Shared ReasoningPlugin and ToolCallPlugin into pinocchio/pkg/chatapp/plugins

## 1. Executive Summary

Pinocchio's `chatapp` package defines a `ChatPlugin` interface that lets
consumers handle geppetto inference events and project them into sessionstream
UI events and timeline entities. Today there are three separate implementations
of the same concepts — reasoning streams and tool calls — scattered across
`cmd/web-chat/`, `pkg/ui/forwarders/agent/`, and `2026-03-16--gec-rag/`. The
coinvault version is buggy (broken thinking content accumulation, broken tool
result display).

This ticket extracts a **ReasoningPlugin** and a **ToolCallPlugin** into
`pinocchio/pkg/chatapp/plugins/` as reusable, shared `ChatPlugin`
implementations. Both pinocchio's `cmd/web-chat` and coinvault will import and
use them, eliminating all duplicated event handling, proto definitions, and
projector logic.

---

## 2. Problem Statement

### 2.1 Three implementations, same concepts

Every chatapp consumer needs to handle:
- **Reasoning/thinking streams** — `EventThinkingPartial`, `EventInfo`
  (thinking-started/ended)
- **Tool calls** — `EventToolCall`, `EventToolCallExecute`, `EventToolResult`,
  `EventToolCallExecutionResult`

Today these are handled in three places:

| Location | Reasoning | Tool Calls | Output API |
|---|---|---|---|
| `pinocchio/cmd/web-chat/reasoning_chat_feature.go` | ✓ (correct) | ✗ | sessionstream `ChatPlugin` |
| `pinocchio/pkg/ui/forwarders/agent/forwarder.go` | ✓ (correct) | ✓ (correct) | bubbletea `timeline.UIEntity*` |
| `2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go` | ✓ (broken) | ✓ (works) | sessionstream `ChatPlugin` |

### 2.2 The coinvault bugs

The coinvault `RuntimeDebugFeature` has multiple issues:

- **Reasoning proto has no `content` field.** `CoinVaultReasoningDelta` only has
  `message_id`, `delta`, and `status`. The accumulated text from
  `EventThinkingPartial.Completion` is dropped on the floor.
- **`ProjectTimeline` uses `firstNonEmpty(delta, entity.content)` instead of
  accumulating.** It always picks the delta, overwriting any previously
  accumulated content.
- **Tool call protos are bespoke.** `CoinVaultToolCall`/`CoinVaultToolResult`
  duplicate what the base chatapp proto already supports (or will support once
  we add tool call fields).

### 2.3 The duplication cost

Every new chatapp consumer (another CLI, another web app, another agent) must
reimplement reasoning and tool call handling from scratch. The forwarder in
`pkg/ui/forwarders/agent/` does the same work as the `ChatPlugin` in
`cmd/web-chat/`, just targeting a different output API. There is no shared core.

---

## 3. Architecture: The ChatPlugin System

### 3.1 The ChatPlugin interface

Defined in `pinocchio/pkg/chatapp/features.go`:

```go
type ChatPlugin interface {
    RegisterSchemas(reg *sessionstream.SchemaRegistry) error
    HandleRuntimeEvent(ctx context.Context, runtime RuntimeEventContext, event gepevents.Event) (handled bool, err error)
    ProjectUI(ctx context.Context, ev sessionstream.Event, session *sessionstream.Session, view sessionstream.TimelineView) ([]sessionstream.UIEvent, bool, error)
    ProjectTimeline(ctx context.Context, ev sessionstream.Event, session *sessionstream.Session, view sessionstream.TimelineView) ([]sessionstream.TimelineEntity, bool, error)
}
```

A `ChatPlugin` has four responsibilities:

1. **Register schemas** — tell sessionstream what event names, UI event names,
   and timeline entity kinds it uses, along with the protobuf message types for
   each.
2. **Handle runtime events** — receive geppetto inference events
   (`EventThinkingPartial`, `EventToolCall`, etc.) and publish sessionstream
   backend events (typed protobuf payloads).
3. **Project timeline** — translate backend events into timeline entity
   snapshots. This is where state accumulation happens (e.g., appending deltas
   to content).
4. **Project UI** — translate backend events into UI events sent to the
   browser. These are point-in-time snapshots.

### 3.2 The event flow

```
┌──────────────────────────────────────────────────────────────┐
│  Geppetto Engine                                             │
│  Emits: EventThinkingPartial, EventToolCall, EventFinal, etc │
└──────────────────────┬───────────────────────────────────────┘
                       │ Go interface: events.EventSink
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  chatapp.Engine.runtimeEventSink.PublishEvent()              │
│  - Handles EventPartialCompletion, EventFinal, EventError,   │
│    EventInterrupt directly (base chat flow)                   │
│  - Falls through to handleFeatureRuntimeEvent() for all else │
└──────────────────────┬───────────────────────────────────────┘
                       │ chatapp.ChatPlugin.HandleRuntimeEvent()
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  ChatPlugin.HandleRuntimeEvent()                             │
│  - Receives typed geppetto event                            │
│  - Publishes sessionstream.Event via runtime.Publish()       │
│  - Payload is a protobuf message (ChatMessageUpdate, etc.)   │
└──────────────────────┬───────────────────────────────────────┘
                       │ sessionstream event store
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  SessionStream Hub                                           │
│  - Stores event in ordered log per session                   │
│  - Runs ProjectTimeline() → TimelineEntity snapshots         │
│  - Runs ProjectUI() → UIEvent payloads                       │
│  - Fans out UIEvents via UIFanout (WebSocket, etc.)          │
└──────────────────────┬───────────────────────────────────────┘
                       │ WebSocket / protobuf
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  Browser / Consumer                                          │
│  - Receives UIEvents and TimelineEntity updates              │
│  - React components re-render                                │
└──────────────────────────────────────────────────────────────┘
```

### 3.3 The RuntimeEventContext

```go
type RuntimeEventContext struct {
    SessionID sessionstream.SessionId
    MessageID string
    Publish   func(ctx context.Context, eventName string, payload proto.Message) error
}
```

The `Publish` function sends a sessionstream event. The plugin constructs a
protobuf message and calls `runtime.Publish(ctx, eventName, payload)`. The
`eventName` must have been registered via `RegisterSchemas`.

### 3.4 The base chatapp events

The `chatapp` package already defines these event names and proto types:

| Event Name | Proto Type | Purpose |
|---|---|---|
| `ChatUserMessageAccepted` | `ChatMessageUpdate` | User message persisted |
| `ChatInferenceStarted` | `ChatMessageUpdate` | Assistant inference begins |
| `ChatTokensDelta` | `ChatMessageUpdate` | Streaming text/thinking chunk |
| `ChatInferenceFinished` | `ChatMessageUpdate` | Inference complete |
| `ChatInferenceStopped` | `ChatMessageUpdate` | Inference interrupted/errored |

And these UI events and timeline entity:

| UI Event Name | Timeline Entity Kind |
|---|---|
| `ChatMessageAccepted` | `ChatMessage` |
| `ChatMessageStarted` | `ChatMessage` |
| `ChatMessageAppended` | `ChatMessage` |
| `ChatMessageFinished` | `ChatMessage` |
| `ChatMessageStopped` | `ChatMessage` |

The base `chatapp.Engine` handles `EventPartialCompletion` by publishing
`ChatTokensDelta` with a `ChatMessageUpdate` proto. The base
`baseTimelineProjection` handles accumulation correctly — it sets
`entity.Content = payload.GetContent()` on every `ChatTokensDelta` event.

**Key insight:** If a plugin publishes events using the base event names
(`ChatTokensDelta`) and base proto type (`ChatMessageUpdate`), the base
timeline and UI projectors handle everything automatically. No custom
`ProjectTimeline` or `ProjectUI` needed.

---

## 4. Existing Implementations: Detailed Comparison

### 4.1 pinocchio `reasoningPlugin` (the correct reference)

**File:** `pinocchio/cmd/web-chat/reasoning_chat_feature.go` (246 lines)

This plugin handles `EventThinkingPartial` and `EventInfo` (thinking-started,
thinking-ended, reasoning-summary). It publishes events using **the base
chatapp event names and proto types**:

```go
// HandleRuntimeEvent
switch ev := event.(type) {
case *gepevents.EventThinkingPartial:
    runtime.Publish(ctx, "ChatTokensDelta", &chatappv1.ChatMessageUpdate{
        MessageId: reasoningMessageID,   // "chat-msg-N:thinking"
        Role:      "thinking",
        Chunk:     ev.Delta,
        Text:      ev.Completion,        // accumulated text
        Content:   ev.Completion,        // accumulated text
        Status:    "streaming",
        Streaming: true,
    })
case *gepevents.EventInfo:
    switch ev.Message {
    case "thinking-started":
        runtime.Publish(ctx, "ChatInferenceStarted", ...)
    case "thinking-ended":
        runtime.Publish(ctx, "ChatInferenceFinished", ...)
    }
}
```

Because it uses `ChatTokensDelta` and `ChatMessageUpdate`, the base
`baseTimelineProjection` and `baseUIProjection` handle everything. The plugin's
custom `ProjectTimeline`/`ProjectUI` methods are present but essentially
pass-through logic that could be eliminated.

**This is the pattern we want to replicate for tool calls.**

### 4.2 coinvault `RuntimeDebugFeature` (the buggy one)

**File:** `2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go`

This plugin handles thinking and tool calls using **bespoke protobuf types**:

```go
// Thinking — uses CoinVaultReasoningDelta (has no content field!)
case *gepevents.EventThinkingPartial:
    runtime.Publish(ctx, "CoinVaultReasoningDelta", &CoinVaultReasoningDelta{
        MessageId: runtime.MessageID + "-thinking",
        Delta:     ev.Delta,      // only delta, Completion is dropped
        Status:    "streaming",
    })

// Tool calls — uses CoinVaultToolCall
case *gepevents.EventToolCall:
    runtime.Publish(ctx, "CoinVaultToolCall", &CoinVaultToolCall{
        Id:       ev.ToolCall.ID,
        ToolName: ev.ToolCall.Name,
        Input:    structFromAny(parseMaybeJSON(ev.ToolCall.Input)),
    })
```

It also has custom `ProjectTimeline` logic that tries to accumulate thinking
content but does it wrong (`firstNonEmpty(delta, entity.Content)` always
picks delta). And it uses custom timeline entity kinds (`"message"`,
`"tool_call"`, `"tool_result"`) instead of the base `"ChatMessage"`.

### 4.3 pinocchio `forwarders/agent/forwarder.go` (the TUI path)

**File:** `pinocchio/pkg/ui/forwarders/agent/forwarder.go` (183 lines)

This is a watermill handler, not a `ChatPlugin`. It targets the old bubbletea
`timeline.UIEntity*` API instead of sessionstream. It handles the same events
but pushes them as `UIEntityCreated/Updated/Completed` messages to a TUI.

**Not in scope for this ticket** — the TUI path doesn't use sessionstream and
shouldn't be migrated. But the event handling logic should eventually be shared.

---

## 5. Design: New Proto Messages

### 5.1 New proto messages in `pinocchio/pkg/chatapp/pb/`

The base `ChatMessageUpdate` proto has fields for text streaming
(`message_id`, `role`, `chunk`, `text`, `content`, `status`, `streaming`,
`error`). It does NOT have fields for tool calls.

We need to add new proto messages for tool calls:

```protobuf
// pinocchio/pkg/chatapp/pb/proto/pinocchio/chatapp/v1/chat.proto

message ToolCallUpdate {
  string message_id = 1;
  string tool_call_id = 2;
  string tool_name = 3;
  string input = 4;          // JSON string of arguments
  bool   executing = 5;       // true when we start executing locally
  string status = 6;          // "pending", "executing", "completed"
}

message ToolResultUpdate {
  string message_id = 1;
  string tool_call_id = 2;
  string tool_name = 3;
  string result = 4;          // raw result string
  string status = 5;          // "success", "error"
}

message ToolCallEntity {
  string message_id = 1;
  string tool_call_id = 2;
  string tool_name = 3;
  string input = 4;
  bool   executing = 5;
  string status = 6;
}

message ToolResultEntity {
  string message_id = 1;
  string tool_call_id = 2;
  string tool_name = 3;
  string result = 4;
  string status = 5;
}
```

### 5.2 What about the coinvault protos?

The coinvault protos (`CoinVaultToolCall`, `CoinVaultToolResult`,
`CoinVaultReasoningDelta`, `CoinVaultReasoningDone`) in
`2026-03-16--gec-rag/internal/pb/` will be **deleted**. The shared plugins
use the new chatapp protos instead.

For reasoning, the plugin reuses the base `ChatMessageUpdate` proto (same as
pinocchio's `reasoningPlugin` already does). No new reasoning proto needed.

---

## 6. Design: ReasoningPlugin

### 6.1 Package location

`pinocchio/pkg/chatapp/plugins/reasoning.go`

### 6.2 Event names

The plugin reuses the **base chatapp event names**. No new event names needed:

| Geppetto Event | Backend Event | UI Event |
|---|---|---|
| `EventInfo` (thinking-started) | `ChatInferenceStarted` | `ChatMessageStarted` |
| `EventThinkingPartial` | `ChatTokensDelta` | `ChatMessageAppended` |
| `EventInfo` (thinking-ended) | `ChatInferenceFinished` | `ChatMessageFinished` |

### 6.3 Timeline entity

Uses the base `ChatMessage` timeline entity kind with a `":thinking"`
suffix on the message ID (e.g., `chat-msg-5:thinking`). The base
`baseTimelineProjection` handles accumulation correctly when the payload
has `content` set.

### 6.4 Pseudocode

```go
package plugins

type ReasoningPlugin struct{}

func NewReasoningPlugin() chatapp.ChatPlugin {
    return &ReasoningPlugin{}
}

func (p *ReasoningPlugin) RegisterSchemas(reg *sessionstream.SchemaRegistry) error {
    // No new schemas needed — reuses base ChatMessageUpdate and ChatMessageEntity
    return nil
}

func (p *ReasoningPlugin) HandleRuntimeEvent(
    ctx context.Context,
    runtime chatapp.RuntimeEventContext,
    event gepevents.Event,
) (bool, error) {
    thinkingID := runtime.MessageID + ":thinking"

    switch ev := event.(type) {
    case *gepevents.EventInfo:
        switch ev.Message {
        case "thinking-started":
            return true, runtime.Publish(ctx, chatapp.EventInferenceStarted,
                newChatMessageUpdate(thinkingID, "thinking", "", "", "",
                    "streaming", true, ""))
        case "thinking-ended":
            return true, runtime.Publish(ctx, chatapp.EventInferenceFinished,
                newChatMessageUpdate(thinkingID, "thinking", "", "", "",
                    "finished", false, ""))
        default:
            return false, nil
        }
    case *gepevents.EventThinkingPartial:
        return true, runtime.Publish(ctx, chatapp.EventTokensDelta,
            &chatappv1.ChatMessageUpdate{
                MessageId: thinkingID,
                Role:      "thinking",
                Chunk:     ev.Delta,
                Text:      ev.Completion,    // accumulated text
                Content:   ev.Completion,    // accumulated text
                Status:    "streaming",
                Streaming: true,
            })
    default:
        return false, nil
    }
}

// ProjectUI/ProjectTimeline: return nil, false, nil
// Base chatapp projectors handle everything via the base event names.
```

---

## 7. Design: ToolCallPlugin

### 7.1 Package location

`pinocchio/pkg/chatapp/plugins/toolcall.go`

### 7.2 New event names

```go
const (
    EventToolCallStarted  = "ChatToolCallStarted"
    EventToolCallUpdated  = "ChatToolCallUpdated"
    EventToolCallFinished = "ChatToolCallFinished"
    EventToolResultReady  = "ChatToolResultReady"

    UIToolCallStarted  = "ChatToolCallStarted"
    UIToolCallUpdated  = "ChatToolCallUpdated"
    UIToolCallFinished = "ChatToolCallFinished"
    UIToolResultReady  = "ChatToolResultReady"

    TimelineEntityToolCall   = "ChatToolCall"
    TimelineEntityToolResult = "ChatToolResult"
)
```

### 7.3 Schema registration

```go
func (p *ToolCallPlugin) RegisterSchemas(reg *sessionstream.SchemaRegistry) error {
    for _, err := range []error{
        reg.RegisterEvent(EventToolCallStarted, &chatappv1.ToolCallUpdate{}),
        reg.RegisterEvent(EventToolCallUpdated, &chatappv1.ToolCallUpdate{}),
        reg.RegisterEvent(EventToolCallFinished, &chatappv1.ToolCallUpdate{}),
        reg.RegisterEvent(EventToolResultReady, &chatappv1.ToolResultUpdate{}),
        reg.RegisterUIEvent(UIToolCallStarted, &chatappv1.ToolCallUpdate{}),
        reg.RegisterUIEvent(UIToolCallUpdated, &chatappv1.ToolCallUpdate{}),
        reg.RegisterUIEvent(UIToolCallFinished, &chatappv1.ToolCallUpdate{}),
        reg.RegisterUIEvent(UIToolResultReady, &chatappv1.ToolResultUpdate{}),
        reg.RegisterTimelineEntity(TimelineEntityToolCall, &chatappv1.ToolCallEntity{}),
        reg.RegisterTimelineEntity(TimelineEntityToolResult, &chatappv1.ToolResultEntity{}),
    } {
        if err != nil {
            return err
        }
    }
    return nil
}
```

### 7.4 HandleRuntimeEvent pseudocode

```go
func (p *ToolCallPlugin) HandleRuntimeEvent(
    ctx context.Context,
    runtime chatapp.RuntimeEventContext,
    event gepevents.Event,
) (bool, error) {
    switch ev := event.(type) {
    case *gepevents.EventToolCall:
        // Model requested a tool call — publish as started
        return true, runtime.Publish(ctx, EventToolCallStarted,
            &chatappv1.ToolCallUpdate{
                MessageId:  runtime.MessageID,
                ToolCallId: ev.ToolCall.ID,
                ToolName:   ev.ToolCall.Name,
                Input:      ev.ToolCall.Input,
                Status:     "pending",
            })

    case *gepevents.EventToolCallExecute:
        // We are now executing the tool locally
        return true, runtime.Publish(ctx, EventToolCallUpdated,
            &chatappv1.ToolCallUpdate{
                MessageId:  runtime.MessageID,
                ToolCallId: ev.ToolCall.ID,
                ToolName:   ev.ToolCall.Name,
                Input:      ev.ToolCall.Input,
                Executing:  true,
                Status:     "executing",
            })

    case *gepevents.EventToolResult:
        // Tool finished — publish result + mark tool call as completed
        _ = runtime.Publish(ctx, EventToolResultReady,
            &chatappv1.ToolResultUpdate{
                MessageId:  runtime.MessageID,
                ToolCallId: ev.ToolResult.ID,
                ToolName:   "",
                Result:     ev.ToolResult.Result,
                Status:     "success",
            })
        return true, runtime.Publish(ctx, EventToolCallFinished,
            &chatappv1.ToolCallUpdate{
                MessageId:  runtime.MessageID,
                ToolCallId: ev.ToolResult.ID,
                Status:     "completed",
            })

    case *gepevents.EventToolCallExecutionResult:
        // Same as EventToolResult but from local execution
        _ = runtime.Publish(ctx, EventToolResultReady,
            &chatappv1.ToolResultUpdate{
                MessageId:  runtime.MessageID,
                ToolCallId: ev.ToolResult.ID,
                ToolName:   "",
                Result:     ev.ToolResult.Result,
                Status:     "success",
            })
        return true, runtime.Publish(ctx, EventToolCallFinished,
            &chatappv1.ToolCallUpdate{
                MessageId:  runtime.MessageID,
                ToolCallId: ev.ToolResult.ID,
                Status:     "completed",
            })

    default:
        return false, nil
    }
}
```

### 7.5 ProjectTimeline pseudocode

```go
func (p *ToolCallPlugin) ProjectTimeline(
    _ context.Context,
    ev sessionstream.Event,
    _ *sessionstream.Session,
    view sessionstream.TimelineView,
) ([]sessionstream.TimelineEntity, bool, error) {
    switch ev.Name {
    case EventToolCallStarted:
        payload := ev.Payload.(*chatappv1.ToolCallUpdate)
        entity := &chatappv1.ToolCallEntity{
            MessageId:  payload.GetMessageId(),
            ToolCallId: payload.GetToolCallId(),
            ToolName:   payload.GetToolName(),
            Input:      payload.GetInput(),
            Status:     "pending",
        }
        return []sessionstream.TimelineEntity{{
            Kind: TimelineEntityToolCall,
            Id:   payload.GetToolCallId(),
            Payload: entity,
        }}, true, nil

    case EventToolCallUpdated:
        payload := ev.Payload.(*chatappv1.ToolCallUpdate)
        // Fetch existing entity from view and update
        existing := currentToolCallEntity(view, payload.GetToolCallId())
        existing.Executing = payload.GetExecuting()
        existing.Status = "executing"
        return []sessionstream.TimelineEntity{{
            Kind: TimelineEntityToolCall,
            Id:   payload.GetToolCallId(),
            Payload: existing,
        }}, true, nil

    case EventToolCallFinished:
        payload := ev.Payload.(*chatappv1.ToolCallUpdate)
        existing := currentToolCallEntity(view, payload.GetToolCallId())
        existing.Status = "completed"
        return []sessionstream.TimelineEntity{{
            Kind: TimelineEntityToolCall,
            Id:   payload.GetToolCallId(),
            Payload: existing,
        }}, true, nil

    case EventToolResultReady:
        payload := ev.Payload.(*chatappv1.ToolResultUpdate)
        entity := &chatappv1.ToolResultEntity{
            MessageId:  payload.GetMessageId(),
            ToolCallId: payload.GetToolCallId(),
            ToolName:   payload.GetToolName(),
            Result:     payload.GetResult(),
            Status:     payload.GetStatus(),
        }
        return []sessionstream.TimelineEntity{{
            Kind: TimelineEntityToolResult,
            Id:   payload.GetToolCallId() + ":result",
            Payload: entity,
        }}, true, nil

    default:
        return nil, false, nil
    }
}
```

### 7.6 ProjectUI pseudocode

Simple pass-through — just clone the payload:

```go
func (p *ToolCallPlugin) ProjectUI(
    _ context.Context,
    ev sessionstream.Event,
    _ *sessionstream.Session,
    _ sessionstream.TimelineView,
) ([]sessionstream.UIEvent, bool, error) {
    switch ev.Name {
    case EventToolCallStarted, EventToolCallUpdated,
         EventToolCallFinished, EventToolResultReady:
        return []sessionstream.UIEvent{{
            Name:    ev.Name,
            Payload: proto.Clone(ev.Payload),
        }}, true, nil
    default:
        return nil, false, nil
    }
}
```

---

## 8. Migration Plan: Step by Step

### Step 1: Add proto messages

- Add `ToolCallUpdate`, `ToolResultUpdate`, `ToolCallEntity`, `ToolResultEntity`
  to `pinocchio/pkg/chatapp/pb/proto/pinocchio/chatapp/v1/chat.proto`
- Run protobuf code generation
- Verify: `go build ./pkg/chatapp/...`

### Step 2: Create `pkg/chatapp/plugins/reasoning.go`

- Extract the `HandleRuntimeEvent` logic from
  `pinocchio/cmd/web-chat/reasoning_chat_feature.go`
- Simplify: remove the custom `ProjectUI`/`ProjectTimeline` (not needed when
  using base event names)
- Keep the `RegisterSchemas` as a no-op (reuses base schemas)
- Verify: `go build ./pkg/chatapp/plugins/...`

### Step 3: Create `pkg/chatapp/plugins/toolcall.go`

- Implement the `ToolCallPlugin` as described in Section 7
- Register new schemas (event names + proto types + timeline entity kinds)
- Verify: `go build ./pkg/chatapp/plugins/...`

### Step 4: Wire into pinocchio `cmd/web-chat`

- Replace `newReasoningPlugin()` with `plugins.NewReasoningPlugin()`
- Add `plugins.NewToolCallPlugin()` to the plugin list
- Delete `reasoning_chat_feature.go` from `cmd/web-chat/`
- Verify: `go build ./cmd/web-chat/...` + run tests

### Step 5: Wire into coinvault

- Replace `NewRuntimeDebugFeature()` with `plugins.NewReasoningPlugin()` +
  `plugins.NewToolCallPlugin()` in the Features list
- Delete `runtime_debug_feature.go`
- Delete the coinvault proto types (`CoinVaultToolCall`, `CoinVaultToolResult`,
  `CoinVaultReasoningDelta`, `CoinVaultReasoningDone`) and their `.proto` source
- Update the browser-side JS consumer to handle the new event names
  (`ChatToolCallStarted` etc. instead of `CoinVaultToolCall`)
- Verify: `go build ./internal/...` + run tests

### Step 6: Update frontend JS

The browser JS currently listens for:
- `CoinVaultReasoningDelta` → must change to `ChatMessageAppended` (via the
  base UI projection) with entity role `"thinking"`
- `CoinVaultToolCall` → must change to `ChatToolCallStarted` etc.
- `CoinVaultToolResult` → must change to `ChatToolResultReady`

The frontend will need updating to match the new event names and entity
structure. This is a browser JS change, not a Go change.

### Step 7: Cleanup

- Delete `2026-03-16--gec-rag/internal/pb/proto/coinvault/webchat/v1/runtime_debug.pb.go`
  and its `.proto` source
- Remove any dead schema registrations in coinvault
- Verify all builds and tests pass

---

## 9. File Reference Map

### New files to create

| File | Purpose |
|---|---|
| `pinocchio/pkg/chatapp/plugins/reasoning.go` | Shared ReasoningPlugin |
| `pinocchio/pkg/chatapp/plugins/toolcall.go` | Shared ToolCallPlugin |

### Existing files to modify

| File | Change |
|---|---|
| `pinocchio/pkg/chatapp/pb/proto/pinocchio/chatapp/v1/chat.proto` | Add tool call protos |
| `pinocchio/cmd/web-chat/main.go` | Use shared plugins |
| `2026-03-16--gec-rag/internal/webchat/server.go` | Use shared plugins |

### Files to delete

| File | Reason |
|---|---|
| `pinocchio/cmd/web-chat/reasoning_chat_feature.go` | Replaced by shared ReasoningPlugin |
| `2026-03-16--gec-rag/internal/webchat/runtime_debug_feature.go` | Replaced by shared plugins |
| `2026-03-16--gec-rag/internal/pb/proto/coinvault/webchat/v1/runtime_debug.proto` | Bespoke protos no longer needed |

### Files that stay unchanged

| File | Why |
|---|---|
| `pinocchio/pkg/chatapp/features.go` | ChatPlugin interface — no changes |
| `pinocchio/pkg/chatapp/chat.go` | Engine + base projectors — no changes |
| `pinocchio/pkg/ui/forwarders/agent/forwarder.go` | TUI path — out of scope |
| `pinocchio/cmd/web-chat/agentmode_chat_feature.go` | Stays in cmd for now |

---

## 10. Consumer Comparison: Before and After

### Before

```
pinocchio cmd/web-chat:
  - reasoningPlugin (package main, not importable)
  - agentModePlugin (package main, not importable)
  - NO tool call plugin

coinvault:
  - RuntimeDebugFeature (buggy reasoning + tool calls)
  - CoinVaultProjectionFeature (widget building)
  - Custom protos: CoinVaultToolCall, CoinVaultToolResult,
    CoinVaultReasoningDelta, CoinVaultReasoningDone
```

### After

```
pinocchio/pkg/chatapp/plugins/:
  + ReasoningPlugin  (shared, importable)
  + ToolCallPlugin   (shared, importable)

pinocchio cmd/web-chat:
  - uses plugins.NewReasoningPlugin()
  - uses plugins.NewToolCallPlugin()
  - agentModePlugin (stays local for now)

coinvault:
  - uses plugins.NewReasoningPlugin()
  - uses plugins.NewToolCallPlugin()
  - CoinVaultProjectionFeature (unchanged)
  - NO RuntimeDebugFeature (deleted)
  - NO custom protos (deleted)
```

---

## 11. Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Frontend JS breaks from new event names | Update JS consumer in same PR; test with live wafer profile |
| Proto generation issues | Regenerate before any Go changes; verify with `go build` |
| Timeline entity kind change breaks existing persisted sessions | Old sessions used `"message"`/`"tool_call"` kinds; new ones use `"ChatMessage"`/`"ChatToolCall"`. Migration path: accept both during hydration or clear old sessions |
| ReasoningPlugin no-ops ProjectUI/ProjectTimeline confusion | Add clear godoc comments explaining why custom projection is unnecessary |

---

## 12. Glossary

- **ChatPlugin**: A `pinocchio/pkg/chatapp` interface with four methods:
  `RegisterSchemas`, `HandleRuntimeEvent`, `ProjectUI`, `ProjectTimeline`.
- **sessionstream**: The event store + projector framework. Events are stored
  per session, projectors produce timeline entities and UI events.
- **TimelineEntity**: A snapshot of an entity (chat message, tool call, etc.)
  produced by `ProjectTimeline`. Accumulates state across events.
- **UIEvent**: A point-in-time event sent to the browser via `ProjectUI`.
- **baseTimelineProjection**: The built-in projector in `chatapp` that handles
  the base event names (`ChatTokensDelta`, etc.) and accumulates content
  correctly for `ChatMessage` entities.
- **RuntimeEventContext**: The context object passed to `HandleRuntimeEvent`,
  containing `SessionID`, `MessageID`, and a `Publish` function.
