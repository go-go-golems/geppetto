---
title: "Multiple Thinking Blocks Across Tool Loops: Analysis and Implementation Guide"
ticket: GP-64
doc_type: design
status: active
topics:
  - chatapp
  - plugins
  - reasoning
  - tool-calls
  - sessionstream
  - hydration
owners:
  - manuel
related_files:
  - path: pinocchio/pkg/chatapp/plugins/reasoning.go
    note: Shared ReasoningPlugin; currently derives one thinking entity ID per assistant message
  - path: pinocchio/pkg/chatapp/plugins/toolcall.go
    note: Shared ToolCallPlugin; emits stable tool call/result timeline entities
  - path: pinocchio/pkg/chatapp/chat.go
    note: chatapp.Engine publishes runtime events and owns the assistant message ID
  - path: 2026-03-16--gec-rag/web/src/ws/parsing.ts
    note: CoinVault frontend parses UI events and hydration snapshots into Redux timeline entities
  - path: 2026-03-16--gec-rag/web/src/store/timelineSlice.ts
    note: Redux timeline store upserts by entity ID, so repeated IDs overwrite previous entries
  - path: 2026-03-16--gec-rag/web/src/components/timeline/TimelineEntityRow.tsx
    note: Renders role=thinking ChatMessage entities as collapsible Thoughts blocks
---

# GP-64: Preserve Multiple Thinking Blocks Across Tool Loops

## 1. Executive Summary

CoinVault now uses the shared `pinocchio/pkg/chatapp/plugins.ReasoningPlugin`
and `ToolCallPlugin`. This fixed the earlier single-stream thinking
accumulation bug, but it exposed a second, distinct bug for **multi-step tool
loops**.

When a model alternates between reasoning and tool calls, the UI should show a
chronological transcript like this:

```text
User prompt
Thinking block #1
Tool call #1
Tool result #1
Thinking block #2
Tool call #2
Tool result #2
Thinking block #3
Assistant final answer
```

Instead, observed behavior is:

- during live streaming, the first thinking block moves downward as later tool
  events arrive;
- its full content briefly appears in the new position;
- then it gets cleared and reused for the next thinking stream;
- on hydration/reload, only the **last** thinking block is shown;
- the surviving thinking block can appear before the tool calls rather than in
  its original chronological position.

The most likely root cause is that `ReasoningPlugin` uses exactly one thinking
entity ID per assistant message:

```go
reasoningEntityID(runtime.MessageID) == runtime.MessageID + ":thinking"
```

All reasoning phases inside one assistant message therefore project into the
same sessionstream timeline entity. Since both sessionstream and the frontend
store timeline state by `(kind, id)`, later thinking phases overwrite earlier
thinking phases. The frontend then sorts one surviving entity by its original
created ordinal and updated ordinal, which explains why hydration can show only
one thinking block and why its ordering can look wrong relative to tool calls.

The fix is to make reasoning block identity **segment-aware**: one unique
thinking entity per contiguous reasoning phase, not one per assistant answer.

---

## 2. User-Observed Bug

### 2.1 Reproduction context

The bug was observed in CoinVault at:

```text
http://localhost:5173/?conv_id=f3ac411b-1b17-455d-9b16-d70e1e831d0b
```

The exact conversation is expected to include a repeated pattern:

```text
thinking -> tool call -> tool result -> thinking -> tool call -> tool result -> ...
```

### 2.2 Live UI symptoms

During a live run:

1. The first `Thoughts` block appears.
2. A tool call appears.
3. A tool result appears.
4. A later reasoning phase begins.
5. The first `Thoughts` block appears to move downward, now after the tool
   result.
6. The content in that moved block briefly looks like the full previous
   thinking content.
7. The block is cleared/replaced and starts filling with the new thinking
   content.

This is the classic visual signature of an entity ID being reused: React/Redux
is not rendering a new row; it is updating the existing row and its sort keys.

### 2.3 Hydration symptoms

After reload/hydration:

- only the final thinking block remains;
- earlier thinking blocks are gone;
- the single surviving thinking block may appear before the tool calls, because
  the original entity was created early and then overwritten by later events.

Hydration makes the issue more obvious because a snapshot contains only final
projected entity states, not the UI event stream history.

---

## 3. Current Architecture

### 3.1 Runtime event stream

The backend receives geppetto runtime events for one assistant turn. A
multi-tool reasoning model can emit a sequence like:

```text
EventInfo("thinking-started")
EventThinkingPartial(delta="...", completion="...")
EventInfo("thinking-ended")
EventToolCall(id="call-A")
EventToolCallExecute(id="call-A")
EventToolCallExecutionResult(id="call-A")
EventInfo("thinking-started")
EventThinkingPartial(delta="...", completion="...")
EventInfo("thinking-ended")
EventToolCall(id="call-B")
EventToolCallExecute(id="call-B")
EventToolCallExecutionResult(id="call-B")
EventPartialCompletion(...)
EventFinal(...)
```

All of these events occur inside one `chatapp.Engine` runtime run. The engine
allocates one parent assistant message ID, for example:

```text
chat-msg-7
```

Plugins receive that parent ID through `chatapp.RuntimeEventContext.MessageID`.

### 3.2 Current ReasoningPlugin identity strategy

`ReasoningPlugin` derives a thinking entity ID from the parent message ID:

```go
func reasoningEntityID(messageID string) string {
    messageID = strings.TrimSpace(messageID)
    if messageID == "" {
        return ""
    }
    return messageID + ":thinking"
}
```

So every thinking phase inside `chat-msg-7` uses exactly this timeline entity:

```text
Kind: ChatMessage
Id:   chat-msg-7:thinking
Role: thinking
```

### 3.3 Current ReasoningPlugin projection

For every reasoning event, `ProjectTimeline` does:

```go
entity, hadEntity := currentReasoningEntity(view, messageID)
content := payload["content"]
if content == "" {
    content = entity.Content // carry previous content forward
}
entity.MessageId = messageID
entity.Role = "thinking"
entity.Content = content
entity.Text = content
return TimelineEntity{Kind: ChatMessage, Id: messageID, Payload: entity}
```

This is correct for **one reasoning phase**, but wrong for multiple phases with
the same `messageID`.

### 3.4 ToolCallPlugin identity strategy

By contrast, `ToolCallPlugin` uses tool-call-specific IDs:

```text
Kind: ChatToolCall
Id:   payload.ToolCallId

Kind: ChatToolResult
Id:   payload.ToolCallId + ":result"
```

This preserves multiple tool calls because each tool call has a distinct ID.

### 3.5 Frontend store behavior

CoinVault parses backend UI events and snapshots into Redux `TimelineEntity`
objects. The timeline store keeps entities by ID and upserts on each event.

The important invariant is:

```text
same frontend entity id => update existing row
new frontend entity id  => render a new row
```

Since all thinking phases use `chat-msg-7:thinking`, the frontend does exactly
what it was asked to do: it updates the same row.

---

## 4. Why the Bug Happens

### 4.1 One entity ID collapses many logical blocks

The model produces multiple logical thinking blocks:

```text
Thinking block A
Thinking block B
Thinking block C
```

But the plugin projects them all to the same identity:

```text
chat-msg-7:thinking
chat-msg-7:thinking
chat-msg-7:thinking
```

Sessionstream timeline projection is entity-snapshot based. It is not an event
log renderer. The latest projection for `(ChatMessage, chat-msg-7:thinking)`
replaces the previous projection for that same key.

### 4.2 Completion semantics reset per provider/model phase

`EventThinkingPartial` carries:

- `Delta`: the new token chunk;
- `Completion`: the accumulated content for the current reasoning stream.

For one reasoning stream, `Completion` grows monotonically:

```text
"I need"
"I need to call"
"I need to call inventory lookup"
```

Across a tool boundary, a provider or stream normalizer may start a new
reasoning stream whose completion begins again from empty/current-phase text:

```text
first phase Completion:  "I should call lookup A..."
second phase Completion: "Now that I have result A..."
```

When the same entity ID is reused, the second phase's early deltas overwrite the
first phase's final content. This exactly matches the observed "full contents ->
cleared -> filled again" behavior.

### 4.3 Hydration only sees final entity state

During live streaming, UI events may temporarily show state transitions. During
hydration, however, the browser receives a snapshot of final timeline entities.
If three reasoning phases all used one entity ID, the snapshot can only contain
one `ChatMessage` entity for that ID.

That explains why hydration shows only the last thinking block.

### 4.4 Ordering is contaminated by entity reuse

Timeline display order uses created/updated ordinals. For an entity that is
updated many times, there are two competing intuitions:

- created ordinal says "this entity started near the beginning";
- updated ordinal says "this entity was last touched near the end".

Depending on sorting and upsert details, the reused thinking entity can appear
too early or move downward during live updates. Both are artifacts of using one
entity to represent multiple transcript rows.

---

## 5. Correct Behavior and Invariants

The transcript should satisfy these invariants:

1. **One logical reasoning phase becomes one timeline entity.**
2. **A reasoning entity ID is stable only within that phase.**
3. **Tool calls/results keep their existing stable IDs.**
4. **Hydration reconstructs the same transcript order as live streaming.**
5. **A later reasoning phase must never overwrite an earlier reasoning phase.**
6. **Reasoning content accumulation remains per-phase, not global across the
   whole assistant message.**

The desired final entity sequence for one assistant turn should be:

```text
ChatMessage       chat-msg-7-user
ChatMessage       chat-msg-7:thinking:1
ChatToolCall      call-A
ChatToolResult    call-A:result
ChatMessage       chat-msg-7:thinking:2
ChatToolCall      call-B
ChatToolResult    call-B:result
ChatMessage       chat-msg-7:thinking:3
ChatMessage       chat-msg-7
```

The suffix format is illustrative. The important point is that thinking segment
IDs must be unique and deterministic within the event stream.

---

## 6. Implementation Strategy

### 6.1 Recommended fix: segment-aware ReasoningPlugin

Add per-run segment tracking to `ReasoningPlugin.HandleRuntimeEvent` so each
contiguous reasoning phase gets its own ID.

The plugin needs to know, for each parent assistant message ID:

- the current reasoning segment number;
- whether a reasoning segment is currently open.

A minimal data model:

```go
type ReasoningPlugin struct {
    mu       sync.Mutex
    segments map[string]reasoningSegmentState // key: parent message ID
}

type reasoningSegmentState struct {
    Current int
    Active  bool
}
```

Segment ID derivation:

```go
func reasoningSegmentEntityID(parentID string, segment int) string {
    return fmt.Sprintf("%s:thinking:%d", parentID, segment)
}
```

### 6.2 Segment lifecycle rules

Handle events as follows:

```text
thinking-started:
  if no active segment:
      increment segment counter
      mark active=true
  publish ChatReasoningStarted for current segment ID

EventThinkingPartial:
  if no active segment:
      increment segment counter
      mark active=true
  publish ChatReasoningDelta for current segment ID

thinking-ended:
  if active segment:
      publish ChatReasoningFinished for current segment ID
      mark active=false
  else:
      ignore or publish finish for latest segment only if needed

reasoning-summary:
  publish to current active segment if one exists;
  otherwise start a new segment and immediately finish it
```

This handles providers that emit explicit `thinking-started`/`thinking-ended`
and providers that only emit `EventThinkingPartial`.

### 6.3 Pseudocode

```go
func (p *ReasoningPlugin) HandleRuntimeEvent(ctx context.Context, runtime chatapp.RuntimeEventContext, event gepevents.Event) (bool, error) {
    parentID := strings.TrimSpace(runtime.MessageID)
    if parentID == "" {
        return false, nil
    }

    switch ev := event.(type) {
    case *gepevents.EventInfo:
        switch ev.Message {
        case "thinking-started":
            segmentID := p.startReasoningSegment(parentID)
            return true, publishStarted(segmentID, parentID)

        case "thinking-ended":
            segmentID, ok := p.currentReasoningSegment(parentID)
            if !ok {
                return false, nil
            }
            p.finishReasoningSegment(parentID)
            return true, publishFinished(segmentID, parentID)

        case "reasoning-summary":
            segmentID := p.ensureReasoningSegment(parentID)
            p.finishReasoningSegment(parentID)
            return true, publishFinishedWithContent(segmentID, parentID, infoText(ev.Data))
        }

    case *gepevents.EventThinkingPartial:
        segmentID := p.ensureReasoningSegment(parentID)
        return true, publishDelta(segmentID, parentID, ev.Delta, ev.Completion)
    }

    return false, nil
}
```

Helper semantics:

```go
func (p *ReasoningPlugin) startReasoningSegment(parentID string) string {
    p.mu.Lock()
    defer p.mu.Unlock()
    state := p.segments[parentID]
    if !state.Active {
        state.Current++
        state.Active = true
    }
    p.segments[parentID] = state
    return reasoningSegmentEntityID(parentID, state.Current)
}

func (p *ReasoningPlugin) ensureReasoningSegment(parentID string) string {
    p.mu.Lock()
    defer p.mu.Unlock()
    state := p.segments[parentID]
    if !state.Active {
        state.Current++
        state.Active = true
    }
    p.segments[parentID] = state
    return reasoningSegmentEntityID(parentID, state.Current)
}

func (p *ReasoningPlugin) finishReasoningSegment(parentID string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    state := p.segments[parentID]
    state.Active = false
    p.segments[parentID] = state
}
```

### 6.4 Fresh-cutover helper API

GP-64 is a fresh cutover. The shared plugin API should expose segment-aware IDs
as the canonical identity form:

```go
func ReasoningSegmentEntityID(messageID string, segment int) string
```

Any remaining single-block helper should be treated only as an internal
first-segment convenience, not as a compatibility promise. Downstream callers
must move to segment-aware IDs rather than supporting both old and new schemas.

---

## 7. Projection Changes

### 7.1 Backend timeline projection

`ProjectTimeline` can mostly stay as-is if incoming payloads already carry a
unique segment-aware `messageId`.

The key change is upstream in `HandleRuntimeEvent`: payloads must contain:

```text
messageId:       chat-msg-7:thinking:2
parentMessageId: chat-msg-7
segment:         2
```

Adding `segment` is optional for rendering but useful for debugging and tests.

### 7.2 UI projection

`ProjectUI` also mostly stays as-is. Since it forwards the payload's
`messageId`, the frontend will see distinct IDs and create distinct Redux rows.

### 7.3 Frontend live event parsing

`web/src/ws/parsing.ts` currently does:

```ts
const id = stringValue(payload.messageId) || `thinking-${eventOrdinal}`;
return { id, kind: "message", data: { role: "thinking", ... } };
```

If the backend sends `chat-msg-7:thinking:1`, `chat-msg-7:thinking:2`, etc., no
frontend identity change is required. The parser will naturally create separate
entities.

### 7.4 Frontend hydration parsing

Hydration uses `parseSnapshotEntity()` for `ChatMessage` timeline entities. It
sets:

```ts
id: payload.messageId || entity.id
```

Again, if the backend emits unique timeline entity IDs, hydration naturally
works.

---

## 8. Tests to Add

### 8.1 ReasoningPlugin unit test: multiple segments

Add a test in `pinocchio/pkg/chatapp/plugins/reasoning_test.go`.

Test sequence:

```go
plugin := plugins.NewReasoningPlugin()
runtime := RuntimeEventContext{MessageID: "chat-msg-1", Publish: capture}

send thinking-started
send ThinkingPartial("a", "aaa")
send thinking-ended
send ToolCall // ignored by reasoning plugin
send thinking-started
send ThinkingPartial("b", "bbb")
send thinking-ended
```

Expected published reasoning payload IDs:

```text
chat-msg-1:thinking:1
chat-msg-1:thinking:1
chat-msg-1:thinking:1
chat-msg-1:thinking:2
chat-msg-1:thinking:2
chat-msg-1:thinking:2
```

Important assertions:

- first and second reasoning blocks have different IDs;
- deltas within one block keep the same ID;
- `thinking-ended` closes the current segment;
- a subsequent `thinking-started` opens a new segment.

### 8.2 ReasoningPlugin projection test: separate timeline entities

Project two segment payloads into a fake timeline view and assert:

```go
entities[0].Id == "chat-msg-1:thinking:1"
entities[1].Id == "chat-msg-1:thinking:2"
```

Also assert content is preserved independently.

### 8.3 chatapp integration test: hydration snapshot order

Add a fake runtime engine that emits:

```text
thinking #1
ToolCall call-A
ToolResult call-A
thinking #2
ToolCall call-B
ToolResult call-B
final answer
```

Run through a real `sessionstream.Hub`, then snapshot. Assert final snapshot
contains:

```text
ChatMessage / chat-msg-1-user
ChatMessage / chat-msg-1:thinking:1
ChatToolCall / call-A
ChatToolResult / call-A:result
ChatMessage / chat-msg-1:thinking:2
ChatToolCall / call-B
ChatToolResult / call-B:result
ChatMessage / chat-msg-1
```

The exact array order may depend on snapshot ordering rules. If the store sorts
by created ordinal, assert created ordinals are monotonic in the expected
logical order.

### 8.4 CoinVault frontend parser test

Add a test in `web/src/ws/parsing.test.ts` that constructs two
`ChatReasoningAppended` UI frames with different `messageId`s:

```text
chat-msg-1:thinking:1
chat-msg-1:thinking:2
```

Assert parsed frontend entities have different IDs and do not overwrite each
other when reduced through the timeline slice.

---

## 9. Fresh-Cutover Considerations

### 9.1 Existing persisted conversations

GP-64 does not attempt to migrate old persisted conversations. Conversations
created with folded IDs such as `chat-msg-7:thinking` may remain folded because
the earlier segment identity was never present in the snapshot store.

The cutover target is simple: new runs emit only segment-aware IDs such as
`chat-msg-7:thinking:1`, `chat-msg-7:thinking:2`, `chat-msg-7:text:1`, and
`chat-msg-7:text:2`.

### 9.2 No dual frontend protocol

The frontend should parse the shared chatapp protocol only:

- `ChatMessage*` with `ChatMessageUpdate` / `ChatMessageEntity`;
- `ChatReasoning*` with shared reasoning payloads;
- `ChatToolCall*` with `ToolCallUpdate` / `ToolCallEntity`;
- `ChatToolResultReady` with `ToolResultUpdate` / `ToolResultEntity`.

Do not keep GP-64-era compatibility branches for legacy CoinVault runtime-debug
messages such as `CoinVaultReasoningDelta`, `CoinVaultReasoningDone`,
`CoinVaultToolCall`, or `CoinVaultToolResult`.

### 9.3 Interaction with summaries

Some providers emit `reasoning-summary` after or instead of the full reasoning
stream. The implementation must decide whether summaries:

1. replace the current segment content;
2. become a separate summary segment;
3. are ignored if full thinking already exists.

Current behavior publishes `reasoning-summary` as a finished reasoning payload
for the same thinking entity. To minimize behavior change, keep that semantics
within the **current segment**:

```text
thinking-started -> deltas -> reasoning-summary -> thinking-ended
```

The summary should update/finish the same segment, not create a new one, when a
segment is active.

If a `reasoning-summary` arrives with no active segment, create and finish a new
summary segment.

---

## 10. Alternative Designs Considered

### 10.1 Use event ordinal in the reasoning entity ID

Example:

```text
chat-msg-7:thinking:ord-42
```

Pros:
- trivially unique;
- no plugin state needed.

Cons:
- every delta would get a new ID unless carefully anchored to start ordinal;
- the plugin still needs state to know the start ordinal for the current
  segment;
- IDs are less readable.

Verdict: acceptable but not better than a segment counter.

### 10.2 Create one thinking entity per delta

Pros:
- no accumulation bug.

Cons:
- terrible UI/UX;
- hydration would show many tiny thoughts rows;
- not what users expect.

Verdict: reject.

### 10.3 Append all thinking phases into one giant block with separators

Example:

```text
[Thinking 1]
...
[Thinking 2]
...
```

Pros:
- simple entity identity;
- no disappearing data.

Cons:
- still cannot interleave thoughts with tool calls/results;
- hydration order remains wrong because there is only one entity;
- live transcript cannot show `thinking -> tool -> result -> thinking`.

Verdict: reject. The user's bug is specifically about preserving interleaving.

### 10.4 Move segmentation to the frontend

The frontend could detect repeated `ChatReasoningStarted` events and generate
local IDs.

Pros:
- no backend change.

Cons:
- hydration snapshots still contain only one folded entity;
- server-side persisted state remains wrong;
- multiple clients could disagree about IDs.

Verdict: reject. Identity must be fixed at projection/source level.

---

## 11. Implementation Checklist

1. Modify `ReasoningPlugin`:
   - add mutex and per-parent segment state;
   - initialize state in `NewReasoningPlugin()`;
   - add `ReasoningSegmentEntityID(parentID, segment)`;
   - publish segment-aware IDs.

2. Preserve current API:
   - keep `ReasoningEntityID(parentID)` for old callers/tests;
   - update tests to prefer segment IDs.

3. Add tests:
   - runtime event sequence produces multiple IDs;
   - projection creates separate timeline entities;
   - full chatapp/sessionstream integration with tool interleaving;
   - CoinVault parser preserves separate IDs.

4. Validate:
   - `go test ./pkg/chatapp/... ./cmd/web-chat/...` in pinocchio;
   - `go build ./internal/...` in coinvault;
   - `npx tsc --noEmit` and parser tests in coinvault web;
   - browser smoke test with a conversation that forces at least two tool calls.

5. Manual browser acceptance:
   - run CoinVault with a profile that emits thinking and tool calls;
   - ask a prompt that triggers multiple tool calls;
   - observe multiple `Thoughts` blocks interleaved with tool rows;
   - reload the page;
   - verify hydration shows the same sequence.

---

## 12. Acceptance Criteria

A fix is accepted when:

- A conversation with repeated `thinking -> tool call -> tool result` cycles
  renders multiple `Thoughts` blocks during live streaming.
- Earlier thoughts do not move downward and get reused.
- Earlier thoughts do not disappear when later thoughts arrive.
- Reloading the page shows all thinking blocks, not just the last one.
- Hydrated order matches live order well enough for the transcript:

```text
thinking #1 before tool call #1
result #1 before thinking #2
thinking #2 before tool call #2
```

- Existing single-thinking conversations still render as before.
- The GP-63 `wafer-qwen3.5-397b` smoke path still passes.

---

## 13. Short Diagnosis for Reviewers

The bug is not primarily in CoinVault's renderer. CoinVault is showing the state
it receives. The root issue is shared `ReasoningPlugin` identity: every
reasoning phase for one assistant response uses `chat-msg-N:thinking`.
Sessionstream and Redux correctly treat that as one entity and overwrite it.

Fix identity first. Give each reasoning phase a stable segment-specific ID. Once
IDs are unique, both live UI events and hydration snapshots should naturally
preserve multiple thinking blocks.

---

## 14. Addendum: Assistant Text Segments Have the Same Identity Problem

After the initial GP-64 analysis, we identified that the same class of bug can
also apply to normal assistant text messages, not only `role="thinking"` blocks.

The broader invariant is:

> Any visible transcript content that can occur multiple times inside one
> assistant turn must have segment-level identity, not just parent
> assistant-message identity.

That includes:

- thinking blocks;
- assistant text blocks;
- tool calls;
- tool results;
- warning/error/status messages.

Tool calls and tool results are already mostly safe because `ToolCallPlugin`
uses per-tool-call IDs:

```text
ChatToolCall   call-A
ChatToolResult call-A:result
ChatToolCall   call-B
ChatToolResult call-B:result
```

Thinking blocks are currently unsafe because all reasoning phases for one
assistant turn use:

```text
chat-msg-N:thinking
```

Assistant text is potentially unsafe for the same reason: base `chatapp.Engine`
streams all assistant text for one assistant turn into:

```text
chat-msg-N
```

In a simple flow, assistant text appears only once at the end, so this is fine.
But richer tool-loop models can interleave assistant text with tool calls:

```text
assistant text #1
Tool call #1
Tool result #1
assistant text #2
Tool call #2
Tool result #2
assistant final text
```

If all assistant text phases use `chat-msg-N`, later text phases overwrite or
move earlier text in the same way later thinking phases overwrite earlier
thinking.

### 14.1 Generalized target transcript identity

The desired model is not only segment-aware thinking. It is segment-aware
assistant transcript rows:

```text
ChatMessage     chat-msg-7-user
ChatMessage     chat-msg-7:thinking:1
ChatMessage     chat-msg-7:text:1
ChatToolCall    call-A
ChatToolResult  call-A:result
ChatMessage     chat-msg-7:thinking:2
ChatMessage     chat-msg-7:text:2
ChatToolCall    call-B
ChatToolResult  call-B:result
ChatMessage     chat-msg-7:thinking:3
ChatMessage     chat-msg-7:text:3
```

The parent `chat-msg-7` should be treated as the logical assistant turn/run ID.
Visible rows inside that turn should use segment IDs when multiple rows are
possible.

### 14.2 Base chatapp changes to consider

The first GP-64 implementation can focus on reasoning because that is the
observed bug. However, a complete fix should also audit and likely update
`chatapp.runtimeEventSink` in `pinocchio/pkg/chatapp/chat.go`.

Today it publishes base assistant text events like:

```go
EventTokensDelta       -> messageID == runtime.MessageID
EventInferenceFinished -> messageID == runtime.MessageID
EventInferenceStopped  -> messageID == runtime.MessageID
```

A segment-aware design would allocate text segment IDs:

```text
chat-msg-7:text:1
chat-msg-7:text:2
chat-msg-7:text:3
```

Boundary rules could be similar to reasoning:

- first `EventPartialCompletion` opens text segment #1;
- continuous text deltas update the current text segment;
- tool-call/tool-result boundaries close the current text segment;
- later text deltas after a tool result open the next text segment;
- final answer finishes the current text segment rather than overwriting an
  earlier text segment.

### 14.3 Coordination between text and plugin segmentation

Because tool events are currently handled by plugins while assistant text is
handled by `runtimeEventSink`, the segmentation design needs an explicit shared
state or boundary signal. Options:

1. Keep separate segment counters but let both text and reasoning react to tool
   boundaries independently.
2. Introduce a small shared transcript segment allocator owned by
   `chatapp.Engine` and used by base text handling plus plugins.
3. Emit additional internal boundary events when tool calls/results occur.

Option 2 is cleanest long-term: a transcript segment allocator can provide IDs
for `thinking`, `text`, `warning`, and future row types while keeping the parent
assistant turn ID stable.

### 14.4 Updated scope recommendation

Rename or broaden the practical implementation scope from:

```text
Preserve multiple thinking blocks across tool loops
```

to:

```text
Preserve multiple assistant transcript segments across tool loops
```

The first implementation slice can still be:

1. fix thinking segmentation;
2. add tests proving the existing assistant text path is either safe for current
   event streams or explicitly marked as pending;
3. follow with base text segmentation if an interleaved assistant-text repro is
   available.

The acceptance criteria should include both:

- multiple thinking blocks survive live streaming and hydration;
- multiple assistant text blocks survive live streaming and hydration when a
  provider emits text before/after tool calls in the same assistant turn.
