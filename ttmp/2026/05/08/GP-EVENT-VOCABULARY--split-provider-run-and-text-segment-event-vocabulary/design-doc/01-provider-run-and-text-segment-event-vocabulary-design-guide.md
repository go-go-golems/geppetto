---
Title: Provider run and text segment event vocabulary design guide
Ticket: GP-EVENT-VOCABULARY
Status: active
Topics:
  - geppetto
  - pinocchio
  - streaming
  - observability
  - events
DocType: design-doc
Intent: long-term
Owners:
  - manuel
RelatedFiles:
  - Path: pkg/events/chat-events.go
    Note: Current Geppetto event type vocabulary that will be removed/replaced by the hard cutover.
  - Path: pkg/events/text_events.go
    Note: Current text event structs including EventPartialCompletion and EventFinal, both replaced by explicit segment events.
  - Path: pkg/steps/ai/claude/content-block-merger.go
    Note: Claude content block merger where provider envelope events and text/tool blocks are currently mapped.
  - Path: pkg/steps/ai/openai/observability.go
    Note: Existing normalized correlation-key construction for OpenAI-compatible Chat Completions.
  - Path: pkg/steps/ai/openai_responses/observability.go
    Note: Existing normalized correlation-key construction for OpenAI Responses.
  - Path: ../pinocchio/pkg/chatapp/runtime_sink.go
    Note: Current mapping from Geppetto EventFinal to Pinocchio ChatInferenceFinished, removed by this hard cutover.
  - Path: ../pinocchio/proto/pinocchio/chatapp/v1/chat.proto
    Note: Current typed protobuf payloads and provider correlation fields, replaced/reshaped around CorrelationInfo.
ExternalSources: []
Summary: Hard-cutover design for replacing overloaded inference/text finalization events with explicit run, provider-call, text-segment, reasoning-segment, and tool lifecycle events plus mandatory typed correlation IDs.
LastUpdated: 2026-05-08T06:25:00-04:00
WhatFor: Guide an atomic implementation that removes EventFinal/ChatInferenceFinished semantic ambiguity and replaces metadata heuristics with typed correlation IDs.
WhenToUse: Use before changing Geppetto event names, Pinocchio chatapp protobufs, runtime sinks, frontend timeline mapping, or SQLite trace exports.
---

# Provider run and text segment event vocabulary design guide

## 1. Executive summary

This ticket now assumes a **hard cutover**. We will not preserve the old event vocabulary as runtime compatibility aliases. The old vocabulary is the source of the ambiguity, so keeping it alive would keep the bug class alive. The implementation should remove the overloaded names and move every provider, runtime sink, protobuf payload, frontend parser, and SQLite export to the new canonical vocabulary in one coordinated branch set.

The system currently conflates these three different lifecycles:

```text
1. Chat run lifecycle:        the whole user-visible assistant turn.
2. Provider call lifecycle:   one call/message/response from a model provider.
3. Text segment lifecycle:    one visible assistant text block/row.
```

That conflation shows up as the names `EventFinal` in Geppetto and `ChatInferenceFinished` in Pinocchio. Those names sound like run/provider lifecycle events, but the code often uses them as text-segment finalizers. Claude made the problem visible because Anthropic separates provider envelope events from content block events. A Claude `message_stop` closes the provider message envelope; it does not close visible assistant text. Treating it as `EventFinal` caused Pinocchio to create a second text segment and render duplicate assistant prose.

The new canonical vocabulary is explicit:

```text
Chat run lifecycle:        ChatRunStarted -> ChatRunFinished / ChatRunStopped / ChatRunFailed
Provider call lifecycle:   ProviderCallStarted -> ProviderCallMetadataUpdated -> ProviderCallFinished
Text segment lifecycle:    TextSegmentStarted -> TextDelta -> TextSegmentFinished
Reasoning lifecycle:       ReasoningSegmentStarted -> ReasoningDelta -> ReasoningSegmentFinished
Tool lifecycle:            ToolCallStarted -> ToolCallArgumentsDelta -> ToolCallRequested -> ToolResultReady
```

The equally important design choice is identity. Every new event must carry a typed correlation envelope. No new runtime logic may route by spelunking through `metadata.Extra`. `metadata.Extra` may remain as debug-only provider baggage, but it is not part of the routing or joining contract.

The cutover removes the ambiguous old symbols. If code still compiles against `EventFinal` or `ChatInferenceFinished`, the migration is incomplete.

## 2. Hard cutover contract

A hard cutover means the old event vocabulary is not supported at runtime after the change lands. This is intentionally stricter than a compatibility migration.

### 2.1 Removed symbols and replacements

| Removed symbol | Replacement | Required downstream change |
|---|---|---|
| `EventTypeStart` / `EventPartialCompletionStart` | `EventProviderCallStarted` or `EventTextSegmentStarted` | Provider adapters must choose the actual lifecycle being started. |
| `EventTypePartialCompletion` / `EventPartialCompletion` | `EventTextDelta` | Text deltas carry typed `Correlation`, including `segment_id` and `correlation_key`. |
| `EventTypeFinal` / `EventFinal` | `EventTextSegmentFinished` **or** `EventProviderCallFinished` | Providers must stop using one event for both text and provider finalization. |
| `EventTypePartialThinking` / `EventThinkingPartial` | `EventReasoningDelta` | Reasoning streams get their own segment lifecycle and correlation. |
| `EventTypeToolCall` / `EventToolCall` | `EventToolCallStarted`, `EventToolCallArgumentsDelta`, `EventToolCallRequested` | Tool-call streaming and completed tool requests are distinct. |
| `ChatInferenceStarted` | `ChatRunStarted` | Pinocchio run lifecycle becomes explicit. |
| `ChatTokensDelta` | `ChatTextDelta` | Browser text streaming uses segment vocabulary. |
| `ChatInferenceFinished` | `ChatTextSegmentFinished` | Finished text rows no longer masquerade as whole inference completion. |
| `ChatInferenceStopped` | `ChatRunStopped` or `ChatTextSegmentStopped` | Stop semantics must say whether the run or a segment stopped. |

The hard cutover rule is simple:

```text
No new code handles legacy events.
No new protobuf carries duplicate legacy scalar identity fields outside CorrelationInfo.
No new SQLite view depends on old EventFinal/ChatInferenceFinished semantics.
```

### 2.2 Atomic cutover scope

The cutover must update these repositories together:

```text
Geppetto        event types, provider adapters, observability records, tests
Pinocchio       chatapp protobufs, runtime sink, projections, plugins, debug export
CoinVault       external protobuf mirror, websocket parser, timeline entities, tests
Trace browser   SQLite views/pages/scripts that inspect event names and correlation
Docs            provider/event semantics docs and ticket references
```

The project is already operating in a multi-repo workspace, so an atomic local cutover is feasible. Partial compatibility is more expensive than atomic breakage because every compatibility branch preserves the ambiguous model we are trying to remove.

## 3. Vocabulary foundation

A new engineer should start with the mental model, not the code. The same assistant response is represented differently at each layer.

### 3.1 Chat run

A **chat run** is the user-visible assistant turn. It begins when the user submits a prompt and ends when the assistant is finished, stopped, or failed. It can contain multiple provider calls and tool executions.

Example:

```text
User asks: "Compare low-stock and out-of-stock gold items."
Run starts.
Provider call 1 asks for sql_doc.
Tool executes.
Provider call 2 asks for sql_query.
Tool executes.
Provider call 3 writes the final answer.
Run finishes.
```

Run events should answer questions like:

- Is the assistant currently working?
- Did the user stop the run?
- Did the run finish successfully?
- Which run do all provider calls and transcript segments belong to?

Run events should not carry visible assistant text deltas.

### 3.2 Provider call

A **provider call** is one API call/message/response from a model provider. A chat run can contain several provider calls when tools are involved. Provider calls own metadata such as:

- provider and model;
- provider response/message ID;
- stop reason;
- usage;
- duration;
- finish class;
- provider call index.

Provider-call events should answer questions like:

- Which provider API call produced this stream object?
- Did the provider stop because of `tool_use`, `stop`, `end_turn`, `max_tokens`, or an error?
- What usage did the provider report for this call?

Provider-call events should not create browser text rows.

### 3.3 Text segment

A **text segment** is one visible assistant text block or row. It begins, receives deltas, and finishes. A single chat run can contain several text segments.

Text-segment events should answer questions like:

- Which visible text row should this delta append to?
- When is that row finished?
- Which provider object produced this text?

Text-segment events are the only events that create or finish assistant text rows.

### 3.4 Reasoning segment

A **reasoning segment** is a provider reasoning or summary stream. It is separate from visible assistant text. Some UIs render it, some collapse it, and some suppress it. It needs its own segment IDs because reasoning can appear concurrently with text or tool streams.

### 3.5 Tool lifecycle

A **tool call** is a model request for host-side execution. It may stream arguments before it is complete. The host then executes the tool and emits a result. Tool calls are transcript entities, but they are not text segments.

## 4. Current-state evidence

This design is based on source evidence captured in this ticket's `sources/` directory. The important observations are summarized here.

### 4.1 Geppetto has overloaded start/final names

`pkg/events/chat-events.go` defines `EventTypeStart`, `EventTypeFinal`, `EventTypePartialCompletion`, and `EventTypePartialThinking`. The source comment says start-to-final are for text completion. That is already a clue that these names are not run-level names. Evidence: `sources/geppetto-events-chat-event-types.lines.txt`.

`pkg/events/text_events.go` defines `EventFinal` as a struct with a `Text` field. It does not distinguish "text segment finished" from "provider call finished." Evidence: `sources/geppetto-events-text-events.lines.txt`.

### 4.2 Pinocchio maps EventFinal to ChatInferenceFinished

`../pinocchio/pkg/chatapp/runtime_sink.go` contains the direct transformation:

```text
Geppetto EventFinal
  -> ensureTextSegmentID()
  -> publish EventInferenceFinished
```

That is the part of the system that transformed Claude `message_stop` finalization into a browser-visible text segment. Evidence: `sources/pinocchio-runtime-sink.lines.txt`.

The same file also closes active text when a tool event arrives. That boundary fallback made sense with the old vocabulary, but the new vocabulary should prefer explicit `TextSegmentFinished` before tool events.

### 4.3 Existing observability already knows the right identity fields

`pkg/observability/observer.go` already defines record fields for provider, model, response ID, item ID, output index, summary index, choice index, stream kind, correlation key, tool-call ID, and tool-call index. Evidence: `sources/geppetto-observability-record.lines.txt`.

This design promotes those fields from trace records and `metadata.Extra` into a typed event correlation envelope.

### 4.4 OpenAI providers already build correlation keys

`pkg/steps/ai/openai/observability.go` constructs normalized keys for OpenAI-compatible Chat Completions. Evidence: `sources/geppetto-openai-chat-observability.lines.txt`.

`pkg/steps/ai/openai_responses/observability.go` extracts Responses API item IDs and builds item/output/summary correlation. Evidence: `sources/geppetto-openai-responses-observability.lines.txt`.

The problem is not lack of correlation thinking. The problem is lack of a shared typed contract.

### 4.5 Pinocchio protobufs carry provider fields but not a canonical correlation envelope

`../pinocchio/proto/pinocchio/chatapp/v1/chat.proto` currently repeats provider fields on `ChatMessageUpdate`, `ReasoningUpdate`, `ToolCallUpdate`, and `ToolResultUpdate`. Evidence: `sources/pinocchio-chat-proto-correlation-fields.lines.txt`.

The hard cutover replaces that repetition with one `CorrelationInfo` message embedded in every canonical payload.

## 5. Canonical event vocabulary

### 5.1 Geppetto event names

| Family | New event | Meaning | Transcript event? |
|---|---|---|---|
| Run | `EventRunStarted` | The user-visible assistant run began. | No |
| Run | `EventRunFinished` | The user-visible assistant run finished successfully. | No |
| Run | `EventRunStopped` | The run was stopped/cancelled. | No |
| Run | `EventRunFailed` | The run failed. | No |
| Provider call | `EventProviderCallStarted` | One provider API call/message/response began. | No |
| Provider call | `EventProviderCallMetadataUpdated` | Provider stop reason, usage, or IDs changed. | No |
| Provider call | `EventProviderCallFinished` | One provider call envelope ended. | No |
| Text | `EventTextSegmentStarted` | One assistant text segment began. | Yes |
| Text | `EventTextDelta` | Visible assistant text delta arrived. | Yes |
| Text | `EventTextSegmentFinished` | One assistant text segment finished. | Yes |
| Reasoning | `EventReasoningSegmentStarted` | One reasoning segment began. | Optional UI |
| Reasoning | `EventReasoningDelta` | Reasoning delta arrived. | Optional UI |
| Reasoning | `EventReasoningSegmentFinished` | One reasoning segment finished. | Optional UI |
| Tool | `EventToolCallStarted` | Provider started/identified a tool call. | Yes |
| Tool | `EventToolCallArgumentsDelta` | Tool arguments streamed. | Optional UI/debug |
| Tool | `EventToolCallRequested` | Tool call is complete and ready for execution. | Yes |
| Tool | `EventToolExecutionStarted` | Host started executing the tool. | Yes |
| Tool | `EventToolResultReady` | Host produced the tool result. | Yes |
| Error | `EventError` | Provider/runtime error. | Yes/debug |

### 5.2 Pinocchio backend event names

| Old event removed | New backend event |
|---|---|
| `ChatInferenceStarted` | `ChatRunStarted` |
| `ChatTokensDelta` | `ChatTextDelta` |
| `ChatInferenceFinished` | `ChatTextSegmentFinished` |
| `ChatInferenceStopped` | `ChatRunStopped` / `ChatRunFailed` |
| implicit provider metadata | `ChatProviderCallStarted`, `ChatProviderCallMetadataUpdated`, `ChatProviderCallFinished` |
| existing reasoning plugin events | `ChatReasoningSegmentStarted`, `ChatReasoningDelta`, `ChatReasoningSegmentFinished` |
| existing tool plugin events | `ChatToolCallStarted`, `ChatToolCallArgumentsDelta`, `ChatToolCallRequested`, `ChatToolResultReady`, `ChatToolCallFinished` |

There should be no runtime compatibility aliases. Frontend code must update to the new event names.

## 6. Mandatory correlation design

The correlation design is not optional. Every canonical event carries `Correlation`. Every Pinocchio protobuf payload carries `CorrelationInfo`. The fields let SQLite, browser tests, and humans join the chain without heuristics.

### 6.1 Geppetto Correlation struct

```go
package events

type Correlation struct {
    // Application/runtime scope.
    SessionID   string `json:"session_id,omitempty"`
    RunID       string `json:"run_id,omitempty"`
    InferenceID string `json:"inference_id,omitempty"`
    TurnID      string `json:"turn_id,omitempty"`

    // Provider-call scope.
    ProviderCallID    string `json:"provider_call_id,omitempty"`
    ProviderCallIndex int32  `json:"provider_call_index,omitempty"`
    Provider          string `json:"provider,omitempty"`
    Model             string `json:"model,omitempty"`
    ResponseID        string `json:"response_id,omitempty"`

    // Provider item/block scope.
    ItemID            string `json:"item_id,omitempty"`
    OutputIndex       *int32 `json:"output_index,omitempty"`
    SummaryIndex      *int32 `json:"summary_index,omitempty"`
    ChoiceIndex       *int32 `json:"choice_index,omitempty"`
    ContentBlockIndex *int32 `json:"content_block_index,omitempty"`

    // Transcript segment scope.
    SegmentID    string `json:"segment_id,omitempty"`
    SegmentIndex int32  `json:"segment_index,omitempty"`
    SegmentType  string `json:"segment_type,omitempty"` // text, reasoning, tool_call, tool_result
    StreamKind   string `json:"stream_kind,omitempty"`  // provider-normalized stream kind

    // Tool scope.
    ToolCallID    string `json:"tool_call_id,omitempty"`
    ToolCallIndex *int32 `json:"tool_call_index,omitempty"`

    // Normalized join keys.
    CorrelationKey       string `json:"correlation_key,omitempty"`
    ParentCorrelationKey string `json:"parent_correlation_key,omitempty"`
}
```

### 6.2 Correlation requirements by event family

| Event family | Required IDs |
|---|---|
| Run events | `session_id`, `run_id`, `inference_id`, `turn_id`, `correlation_key` |
| Provider-call events | run IDs plus `provider_call_id`, `provider_call_index`, `provider`, `model`, `correlation_key` |
| Text segment events | provider-call IDs plus `segment_id`, `segment_index`, `segment_type=text`, `stream_kind=text`, `correlation_key`, `parent_correlation_key` |
| Reasoning events | provider-call IDs plus `segment_id`, `segment_type=reasoning`, `stream_kind=reasoning` or `reasoning_summary`, `correlation_key` |
| Tool events | provider-call IDs plus `tool_call_id`, `tool_call_index` when available, `stream_kind=tool_call` or `tool_args`, `correlation_key` |
| Tool result events | run IDs plus `tool_call_id`, `parent_correlation_key` pointing to tool call, result `correlation_key` |

### 6.3 Correlation key rules

A normalized key is a stable join key, not a display label. It should be deterministic for all fragments of the same logical stream object.

Rules:

1. Prefix with provider family.
2. Include provider-call identity.
3. Include provider-native item/block/choice/tool identity when available.
4. Include stream kind.
5. Never depend on rendered message IDs such as `chat-msg-1:text:2`.

## 7. Provider-specific correlation plans

### 7.1 Claude

Claude exposes message IDs and content block indexes. Tool-use blocks also expose tool IDs. Later events may not repeat the message ID, so the merger owns the active provider-call correlation.

Recommended keys:

```text
claude:<provider_call_id>:provider-call
claude:<provider_call_id>:block:<content_block_index>:text
claude:<provider_call_id>:block:<content_block_index>:tool:<tool_call_id>
claude:<provider_call_id>:block:<content_block_index>:tool-index:<tool_call_index>
```

Claude mapping table:

| Anthropic event | New Geppetto event | Required correlation |
|---|---|---|
| `message_start` | `EventProviderCallStarted` | `provider_call_id`, `provider_call_index`, `provider=claude`, `model`, `response_id=message.id`, provider-call `correlation_key` |
| `content_block_start type=text` | `EventTextSegmentStarted` | provider-call IDs, `content_block_index`, `segment_id`, `segment_type=text`, text `correlation_key` |
| `content_block_delta text_delta` | `EventTextDelta` | same `segment_id` and `correlation_key`, sequence |
| `content_block_stop type=text` | `EventTextSegmentFinished` | same `segment_id` and `correlation_key` |
| `content_block_start type=tool_use` | `EventToolCallStarted` | provider-call IDs, `content_block_index`, `tool_call_id`, tool `correlation_key` |
| `content_block_delta input_json_delta` | `EventToolCallArgumentsDelta` | same tool `correlation_key`, cumulative args, sequence |
| `content_block_stop type=tool_use` | `EventToolCallRequested` | same tool `correlation_key`, complete input |
| `message_delta` | `EventProviderCallMetadataUpdated` | provider-call IDs, stop reason, usage |
| `message_stop` | `EventProviderCallFinished` | provider-call IDs, stop reason, usage, duration, finish class |

Claude `message_delta` and `message_stop` never emit text events.

### 7.2 OpenAI Responses

OpenAI Responses exposes typed output items. Prefer provider-native `item_id` when available.

Recommended keys:

```text
openai-responses:<response_id>:provider-call
openai-responses:<response_id>:item:<item_id>
openai-responses:<response_id>:output:<output_index>:text
openai-responses:<response_id>:item:<item_id>:summary:<summary_index>
openai-responses:<response_id>:item:<item_id>:tool
```

Mapping table:

| Responses event | New Geppetto event | Required correlation |
|---|---|---|
| first response event / `response.created` | `EventProviderCallStarted` | `provider_call_id`, `response_id`, provider-call key |
| `response.output_item.added type=message` | `EventTextSegmentStarted` | `item_id`, `output_index`, `segment_id`, item key |
| `response.output_text.delta` | `EventTextDelta` | same text segment key |
| `response.output_item.done type=message` | `EventTextSegmentFinished` | same text segment key |
| `response.reasoning_text.delta` | `EventReasoningDelta` | reasoning segment key with `item_id`/`summary_index` |
| `response.output_item.done type=function_call` | `EventToolCallRequested` | tool `item_id`, tool call ID/name/input |
| `response.completed` | `EventProviderCallFinished` | stop reason, usage, duration, no text finalization |

### 7.3 OpenAI-compatible Chat Completions

Chat Completions lacks item IDs for content/reasoning streams. Use provider response ID plus choice index and stream kind. If response ID is missing on early chunks, use generated `provider_call_id` and fill `response_id` when known.

Recommended keys:

```text
<provider>-chat:<response_id-or-provider_call_id>:provider-call
<provider>-chat:<response_id-or-provider_call_id>:choice:<choice_index>:content
<provider>-chat:<response_id-or-provider_call_id>:choice:<choice_index>:reasoning
<provider>-chat:<response_id-or-provider_call_id>:choice:<choice_index>:tool:<tool_call_id>
<provider>-chat:<response_id-or-provider_call_id>:choice:<choice_index>:tool-index:<tool_call_index>
```

Mapping table:

| Chat Completions stream condition | New Geppetto event | Required correlation |
|---|---|---|
| first chunk | `EventProviderCallStarted` | generated provider call ID, provider/model, response ID if available |
| first non-empty `delta.content` for a choice | `EventTextSegmentStarted` | choice index, content key, segment ID |
| `delta.content` | `EventTextDelta` | content key, segment ID |
| first non-empty reasoning delta | `EventReasoningSegmentStarted` | reasoning key, segment ID |
| reasoning delta | `EventReasoningDelta` | reasoning key |
| `delta.tool_calls[index]` first seen | `EventToolCallStarted` | tool key by ID or index |
| subsequent tool argument deltas | `EventToolCallArgumentsDelta` | same tool key |
| `finish_reason=tool_calls` | `EventTextSegmentFinished` for active content only, `EventToolCallRequested`, `EventProviderCallFinished` | no extra text finalization |
| `finish_reason=stop` | `EventTextSegmentFinished` for active content, `EventProviderCallFinished` | no provider EOF text event |

## 8. Geppetto event API sketches

### 8.1 Base event shape

Every new event embeds or exposes `Correlation` directly. It should not be necessary to inspect `EventMetadata.Extra` to route it.

```go
type CorrelatedEvent interface {
    events.Event
    Correlation() events.Correlation
}
```

### 8.2 Text segment events

```go
type EventTextSegmentStarted struct {
    EventImpl
    Corr Correlation `json:"correlation"`
    Role string `json:"role,omitempty"`
}

type EventTextDelta struct {
    EventImpl
    Corr     Correlation `json:"correlation"`
    Delta    string      `json:"delta"`
    Text     string      `json:"text"`
    Sequence int64       `json:"sequence,omitempty"`
}

type EventTextSegmentFinished struct {
    EventImpl
    Corr         Correlation `json:"correlation"`
    Text         string      `json:"text"`
    FinishReason string      `json:"finish_reason,omitempty"`
}
```

### 8.3 Provider-call events

```go
type EventProviderCallStarted struct {
    EventImpl
    Corr Correlation `json:"correlation"`
}

type EventProviderCallMetadataUpdated struct {
    EventImpl
    Corr         Correlation `json:"correlation"`
    StopReason   string      `json:"stop_reason,omitempty"`
    StopSequence string      `json:"stop_sequence,omitempty"`
    Usage        *Usage      `json:"usage,omitempty"`
}

type EventProviderCallFinished struct {
    EventImpl
    Corr         Correlation `json:"correlation"`
    StopReason   string      `json:"stop_reason,omitempty"`
    FinishClass  string      `json:"finish_class,omitempty"`
    Usage        *Usage      `json:"usage,omitempty"`
    DurationMs   *int64      `json:"duration_ms,omitempty"`
    HasToolCalls bool        `json:"has_tool_calls,omitempty"`
}
```

### 8.4 Tool events

```go
type EventToolCallStarted struct {
    EventImpl
    Corr       Correlation `json:"correlation"`
    ToolCallID string      `json:"tool_call_id"`
    ToolName   string      `json:"tool_name,omitempty"`
}

type EventToolCallArgumentsDelta struct {
    EventImpl
    Corr       Correlation `json:"correlation"`
    ToolCallID string      `json:"tool_call_id"`
    Delta      string      `json:"delta"`
    Arguments  string      `json:"arguments"`
    Sequence   int64       `json:"sequence,omitempty"`
}

type EventToolCallRequested struct {
    EventImpl
    Corr       Correlation `json:"correlation"`
    ToolCallID string      `json:"tool_call_id"`
    ToolName   string      `json:"tool_name"`
    Input      string      `json:"input"`
}
```

## 9. Pinocchio protobuf design

### 9.1 Canonical CorrelationInfo

Hard cutover means no duplicated top-level provider fields. Identity lives in `CorrelationInfo`.

```proto
message CorrelationInfo {
  string session_id = 1;
  string run_id = 2;
  string inference_id = 3;
  string turn_id = 4;

  string provider_call_id = 5;
  int32 provider_call_index = 6;
  string provider = 7;
  string model = 8;
  string response_id = 9;

  string item_id = 10;
  optional int32 output_index = 11;
  optional int32 summary_index = 12;
  optional int32 choice_index = 13;
  optional int32 content_block_index = 14;

  string segment_id = 15;
  int32 segment_index = 16;
  string segment_type = 17;
  string stream_kind = 18;

  string tool_call_id = 19;
  optional int32 tool_call_index = 20;

  string correlation_key = 21;
  string parent_correlation_key = 22;
}
```

### 9.2 Canonical payloads

```proto
message ChatRunStarted {
  CorrelationInfo correlation = 1;
  string prompt = 2;
}

message ChatRunFinished {
  CorrelationInfo correlation = 1;
  string status = 2;
}

message ChatProviderCallFinished {
  CorrelationInfo correlation = 1;
  string stop_reason = 2;
  string finish_class = 3;
  Usage usage = 4;
  optional int64 duration_ms = 5;
  bool has_tool_calls = 6;
}

message ChatTextSegmentStarted {
  CorrelationInfo correlation = 1;
  string message_id = 2;
  string parent_message_id = 3;
  string role = 4;
}

message ChatTextDelta {
  CorrelationInfo correlation = 1;
  string message_id = 2;
  string parent_message_id = 3;
  string chunk = 4;
  string text = 5;
  int64 sequence = 6;
}

message ChatTextSegmentFinished {
  CorrelationInfo correlation = 1;
  string message_id = 2;
  string parent_message_id = 3;
  string text = 4;
  string finish_reason = 5;
}

message ChatToolCallRequested {
  CorrelationInfo correlation = 1;
  string message_id = 2;
  string tool_call_id = 3;
  string tool_name = 4;
  string input = 5;
}
```

The concrete `.proto` can split these by package/file if useful, but the architectural rule stays: every payload has one `CorrelationInfo` and no parallel identity fields.

## 10. Runtime behavior after cutover

### 10.1 Geppetto provider adapter pseudocode

Claude is the cleanest example:

```go
case MessageStartType:
    call := corr.NewProviderCall(provider="claude", responseID=event.Message.ID)
    emit ProviderCallStarted(call)

case ContentBlockStartType when block.Type == text:
    segment := call.NewTextSegment(contentBlockIndex=event.Index)
    emit TextSegmentStarted(segment)

case ContentBlockDeltaType when delta.Type == text_delta:
    segment.Append(delta.Text)
    emit TextDelta(segment, delta.Text, segment.Text)

case ContentBlockStopType when block.Type == text:
    emit TextSegmentFinished(segment, segment.Text)

case ContentBlockStartType when block.Type == tool_use:
    tool := call.NewToolCall(block.ID, block.Name, event.Index)
    emit ToolCallStarted(tool)

case ContentBlockDeltaType when delta.Type == input_json_delta:
    tool.AppendArgs(delta.PartialJSON)
    emit ToolCallArgumentsDelta(tool, delta.PartialJSON, tool.Args)

case ContentBlockStopType when block.Type == tool_use:
    emit ToolCallRequested(tool, tool.Args)

case MessageDeltaType:
    call.UpdateMetadata(stopReason, usage)
    emit ProviderCallMetadataUpdated(call)

case MessageStopType:
    call.DurationMs = elapsed
    emit ProviderCallFinished(call)
```

### 10.2 Pinocchio runtime sink pseudocode

```go
func (s *runtimeEventSink) PublishEvent(event gepevents.Event) error {
    switch ev := event.(type) {
    case *events.EventRunStarted:
        return publish(ChatRunStarted, runPayload(ev))

    case *events.EventProviderCallStarted:
        return publish(ChatProviderCallStarted, providerCallStartedPayload(ev))

    case *events.EventProviderCallMetadataUpdated:
        return publish(ChatProviderCallMetadataUpdated, providerCallMetadataPayload(ev))

    case *events.EventProviderCallFinished:
        return publish(ChatProviderCallFinished, providerCallFinishedPayload(ev))

    case *events.EventTextSegmentStarted:
        return publish(ChatTextSegmentStarted, textStartedPayload(ev))

    case *events.EventTextDelta:
        return publish(ChatTextDelta, textDeltaPayload(ev))

    case *events.EventTextSegmentFinished:
        return publish(ChatTextSegmentFinished, textFinishedPayload(ev))

    case *events.EventToolCallRequested:
        return publish(ChatToolCallRequested, toolRequestedPayload(ev))

    default:
        return featurePlugins.HandleRuntimeEvent(ev)
    }
}
```

There is no `case *events.EventFinal`. There is no `ensureTextSegmentID()` path for provider-call events.

## 11. SQLite and trace export design

### 11.1 Required tables

Add provider-call results:

```sql
create table geppetto_inference_results (
  id integer primary key,
  session_id text,
  run_id text,
  inference_id text,
  turn_id text,
  provider_call_id text,
  provider_call_index integer,
  provider text,
  model text,
  response_id text,
  stop_reason text,
  finish_class text,
  truncated integer,
  has_tool_calls integer,
  input_tokens integer,
  output_tokens integer,
  cached_tokens integer,
  cache_creation_input_tokens integer,
  cache_read_input_tokens integer,
  duration_ms integer,
  correlation_key text,
  metadata_json text
);
```

Add canonical segments:

```sql
create table geppetto_segments (
  id integer primary key,
  session_id text,
  run_id text,
  inference_id text,
  turn_id text,
  provider_call_id text,
  provider_call_index integer,
  segment_id text,
  segment_index integer,
  segment_type text,
  stream_kind text,
  provider text,
  model text,
  response_id text,
  item_id text,
  output_index integer,
  summary_index integer,
  choice_index integer,
  content_block_index integer,
  tool_call_id text,
  tool_call_index integer,
  correlation_key text,
  parent_correlation_key text,
  status text,
  text_len integer,
  started_record_id integer,
  finished_record_id integer
);
```

### 11.2 Verification queries

Claude tool-use metadata should be directly verifiable:

```sql
select
  provider_call_index,
  provider,
  model,
  response_id,
  stop_reason,
  finish_class,
  input_tokens,
  output_tokens,
  cache_creation_input_tokens,
  cache_read_input_tokens,
  duration_ms
from geppetto_inference_results
where provider = 'claude'
order by provider_call_index;
```

Expected:

```text
provider_call_index  provider  stop_reason  finish_class          input_tokens  output_tokens
1                    claude    tool_use     tool_calls_pending    ...           ...
2                    claude    end_turn     completed             ...           ...
```

Text duplication should be directly verifiable:

```sql
select
  segment_id,
  provider_call_id,
  segment_index,
  stream_kind,
  correlation_key,
  status,
  text_len
from geppetto_segments
where segment_type = 'text'
order by started_record_id;
```

There should be no row whose only cause is a provider envelope stop.

## 12. Implementation phases

### Phase 1: Remove and replace Geppetto event vocabulary

Files:

```text
pkg/events/chat-events.go
pkg/events/text_events.go
pkg/events/correlation.go
pkg/events/provider_call_events.go
pkg/events/segment_events.go
pkg/events/tool_events.go
```

Tasks:

1. Add `Correlation`.
2. Add run/provider/text/reasoning/tool event structs.
3. Remove or stop exporting old overloaded event constructors.
4. Update event JSON decoding to understand only canonical names.
5. Update tests to use canonical names.

### Phase 2: Migrate Claude

Files:

```text
pkg/steps/ai/claude/content-block-merger.go
pkg/steps/ai/claude/engine_claude.go
pkg/steps/ai/claude/content-block-merger_test.go
pkg/steps/ai/claude/observability_test.go
```

Tasks:

1. Generate one provider-call correlation per Claude API call.
2. Emit provider-call events from `message_start`, `message_delta`, and `message_stop`.
3. Emit text segment events from text content blocks.
4. Emit tool-call events from tool-use content blocks.
5. Assert `message_stop` emits no text event.
6. Assert `ProviderCallFinished` carries `stop_reason=tool_use` and usage for tool-use turns.

### Phase 3: Migrate OpenAI Responses

Files:

```text
pkg/steps/ai/openai_responses/streaming.go
pkg/steps/ai/openai_responses/observability.go
```

Tasks:

1. Use response IDs and item IDs as provider-native identity.
2. Emit text/reasoning/tool segment events from output items.
3. Emit provider-call finished from `response.completed`.
4. Remove any provider-completed-to-text-final behavior.

### Phase 4: Migrate OpenAI-compatible Chat Completions

Files:

```text
pkg/steps/ai/openai/engine_openai.go
pkg/steps/ai/openai/observability.go
```

Tasks:

1. Generate provider-call IDs before response IDs are known.
2. Emit text segment events from content deltas.
3. Emit reasoning segment events from reasoning deltas.
4. Emit tool-call argument events from streamed tool deltas.
5. Emit provider-call finished from finish reasons/EOF.
6. Ensure `finish_reason=tool_calls` does not emit duplicate text finished events.

### Phase 5: Replace Pinocchio protobufs and runtime sink

Files:

```text
../pinocchio/proto/pinocchio/chatapp/v1/chat.proto
../pinocchio/pkg/chatapp/runtime_sink.go
../pinocchio/pkg/chatapp/runtime_inference.go
../pinocchio/pkg/chatapp/projections.go
../pinocchio/pkg/chatapp/plugins/reasoning.go
../pinocchio/pkg/chatapp/plugins/toolcall.go
```

Tasks:

1. Add `CorrelationInfo` and canonical payloads.
2. Remove legacy event names from runtime paths.
3. Regenerate Go protobufs.
4. Update projections and plugins.
5. Update tests.

### Phase 6: Update CoinVault and SQLite tooling

Files:

```text
../2026-03-16--gec-rag/web/src/ws/*
../2026-03-16--gec-rag/web/src/pb/external/pinocchio/chat_pb.ts
../pinocchio/cmd/web-chat/app/debug_reconcile_schema.go
../pinocchio/cmd/web-chat/app/debug_reconcile_views.go
```

Tasks:

1. Mirror regenerated TypeScript protobufs.
2. Update websocket parsing for new event names and `CorrelationInfo`.
3. Add `geppetto_inference_results` and `geppetto_segments` tables.
4. Update trace-browser pages to inspect provider calls and segments.
5. Add browser SQLite verification tests.

## 13. Testing strategy

### 13.1 Compile-time tests

The hard cutover should deliberately break old references. Use `rg` as a validation gate:

```bash
rg "EventFinal|EventPartialCompletion|ChatInferenceFinished|ChatTokensDelta|ChatInferenceStarted" geppetto pinocchio 2026-03-16--gec-rag
```

After implementation, remaining matches should be only in migration notes, archived docs, or intentionally named historical test fixtures.

### 13.2 Provider tests

Claude:

- `message_delta` emits `ProviderCallMetadataUpdated` only.
- `message_stop` emits `ProviderCallFinished` only.
- `content_block_stop text` emits `TextSegmentFinished`.
- `content_block_stop tool_use` emits `ToolCallRequested`.
- Every event has `provider_call_id` and `correlation_key`.

OpenAI Responses:

- `response.completed` emits `ProviderCallFinished` only.
- message output item done emits `TextSegmentFinished`.
- function call item done emits `ToolCallRequested`.

Chat Completions:

- `finish_reason=tool_calls` emits provider-call finished and tool requested events, not an extra text segment.
- streamed argument-only deltas preserve `tool_call_id` through stateful correlation.

### 13.3 Pinocchio tests

- `EventTextDelta` becomes `ChatTextDelta` with `CorrelationInfo` preserved.
- `EventTextSegmentFinished` becomes `ChatTextSegmentFinished` with the same segment ID and correlation key.
- `EventProviderCallFinished` becomes `ChatProviderCallFinished` and does not mutate text segment state.
- Runtime sink has no `EventFinal` branch.

### 13.4 SQLite/browser tests

For Haiku:

```text
provider call 1: stop_reason=tool_use, finish_class=tool_calls_pending
provider call 2: stop_reason=end_turn, finish_class=completed
```

Text segments:

```text
one row per actual Claude text block
no text segment created by message_stop
```

Browser:

```text
no duplicate assistant intro after tool rows
all frontend entities retain CorrelationInfo
```

## 14. Deletion checklist

The hard cutover is complete only when this checklist is satisfied.

- [ ] `EventFinal` constructor and type are removed or inaccessible to provider adapters.
- [ ] `EventPartialCompletion` is replaced by `EventTextDelta`.
- [ ] Pinocchio has no runtime branch mapping `EventFinal` to text output.
- [ ] Pinocchio protobufs no longer define `ChatInferenceStarted`, `ChatTokensDelta`, or `ChatInferenceFinished` as active events.
- [ ] CoinVault websocket parser no longer handles legacy event names.
- [ ] SQLite reconcile views use canonical event names and `CorrelationInfo` fields.
- [ ] Provider-call finished events are visible in SQLite as `geppetto_inference_results`.
- [ ] Text segments are visible in SQLite as `geppetto_segments`.
- [ ] Tests prove Claude `message_stop` never creates text.
- [ ] Tests prove `ProviderCallFinished` carries stop reason and usage.

## 15. Intern implementation notes

If you are implementing this ticket, do not start by renaming strings. Start by drawing the lifecycle you are touching.

For any provider event, ask:

1. Is this about the whole chat run?
2. Is this about one provider call?
3. Is this about a visible text segment?
4. Is this about reasoning?
5. Is this about a tool call or result?

Only after answering that should you choose an event type.

Then assign correlation:

1. What is the provider call ID?
2. What is the provider-native response/message ID?
3. What item/block/choice/tool index does this event belong to?
4. What segment ID should the browser use?
5. What normalized correlation key should SQLite join on?
6. What is the parent correlation key?

If any of those answers requires parsing a rendered message ID or looking inside `metadata.Extra`, the design is incomplete.

## 16. References

### Geppetto

- `pkg/events/chat-events.go`
- `pkg/events/text_events.go`
- `pkg/events/log_info_events.go`
- `pkg/observability/observer.go`
- `pkg/steps/ai/claude/content-block-merger.go`
- `pkg/steps/ai/claude/engine_claude.go`
- `pkg/steps/ai/openai/engine_openai.go`
- `pkg/steps/ai/openai/observability.go`
- `pkg/steps/ai/openai_responses/streaming.go`
- `pkg/steps/ai/openai_responses/observability.go`

### Pinocchio

- `../pinocchio/proto/pinocchio/chatapp/v1/chat.proto`
- `../pinocchio/pkg/chatapp/runtime_sink.go`
- `../pinocchio/pkg/chatapp/runtime_inference.go`
- `../pinocchio/pkg/chatapp/projections.go`
- `../pinocchio/pkg/chatapp/plugins/reasoning.go`
- `../pinocchio/pkg/chatapp/plugins/toolcall.go`
- `../pinocchio/cmd/web-chat/app/debug_reconcile_schema.go`
- `../pinocchio/cmd/web-chat/app/debug_reconcile_views.go`

### Captured source evidence

Line-numbered source excerpts are stored under:

```text
ttmp/2026/05/08/GP-EVENT-VOCABULARY--split-provider-run-and-text-segment-event-vocabulary/sources/
```
