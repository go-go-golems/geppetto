---
Title: Gemini canonical event migration analysis
Ticket: GP-EVENT-VOCABULARY
Status: active
Topics:
    - geppetto
    - streaming
    - observability
    - events
    - inference
DocType: analysis
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: pkg/events/chat-events.go
      Note: Legacy event type constants can only be removed after Gemini and tool executor are migrated.
    - Path: pkg/events/text_events.go
      Note: |-
        Legacy text event structs and constructors targeted for deletion after provider cutover.
        Legacy text event structs cannot be deleted until Gemini no longer references their constructors
    - Path: pkg/events/tool_events.go
      Note: |-
        Legacy tool event structs and constructors targeted for deletion after provider/tool-executor cutover.
        Legacy tool event structs cannot be deleted until Gemini and tool executor no longer reference them
    - Path: pkg/inference/tools/base_executor.go
      Note: |-
        Host-side tool execution still emits legacy tool execution/result events and must move to canonical tool lifecycle events.
        Local tool execution still emits legacy tool execution/result events and needs canonical execution/result lifecycle mapping
    - Path: pkg/steps/ai/gemini/engine_gemini.go
      Note: |-
        Gemini provider still emits legacy start/partial/final/tool-call events and must be migrated before deleting legacy event structs.
        Gemini provider still emits legacy text/final/tool-call events and needs canonical provider-call/text/tool lifecycle mapping
ExternalSources: []
Summary: Gemini-specific analysis and checklist for removing the remaining Geppetto legacy event emissions before deleting EventFinal, EventPartialCompletion, and legacy tool event types.
LastUpdated: 2026-05-08T07:20:00-04:00
WhatFor: Use this as the focused migration plan for the Gemini provider and local tool executor before removing legacy chat event types from Geppetto.
WhenToUse: Read before touching pkg/steps/ai/gemini, pkg/inference/tools/base_executor.go, or deleting legacy EventFinal/EventPartialCompletion/EventToolCall symbols.
---


# Gemini canonical event migration analysis

## Implementation status

Phase 5B has now been implemented for the Gemini provider and the local tool executor. Validation output is saved in:

```text
various/gemini-canonical-migration-validation.log
```

The analysis below documents the reason for the work and the intended mapping. The remaining deletion work is in the broader Geppetto event package: structured sinks, printers, JS bindings, examples, tests, and docs still need to stop depending on legacy event structs before those structs/constants can be removed.

## 1. Why this document exists

During the hard-cutover pass we verified Claude, OpenAI Responses, and OpenAI-compatible Chat Completions, then started discussing deletion of the old Geppetto event types. A deletion scan showed that Gemini was still outside the provider migration set.

The important active runtime matches are:

```text
pkg/steps/ai/gemini/engine_gemini.go
  NewStartEvent
  NewToolCallEvent
  NewPartialCompletionEvent
  NewFinalEvent

pkg/inference/tools/base_executor.go
  NewToolCallExecuteEvent
  NewToolCallExecutionResultEvent
```

So `pkg/events/chat-events.go`, `pkg/events/text_events.go`, and `pkg/events/tool_events.go` cannot be cleaned out yet. The old symbols are still part of active runtime behavior, not just historical docs.

Source evidence was captured in:

```text
sources/geppetto-gemini-engine-legacy-events.lines.txt
sources/geppetto-tool-executor-legacy-events.lines.txt
various/gemini-and-tool-executor-legacy-event-inventory.txt
```

## 2. Current Gemini behavior

Gemini currently creates one `EventMetadata` value for the provider call and uses it for every emitted event. It enriches the metadata with model/settings, session/inference/turn IDs, settings metadata, and runtime attribution.

The legacy emission shape is:

```text
before stream:             EventTypeStart / EventPartialCompletionStart
text chunk:                EventTypePartialCompletion / EventPartialCompletion
function call chunk:       EventTypeToolCall / EventToolCall
stream end:                EventTypeFinal / EventFinal
stream error:              EventTypeError / EventError
```

The engine also accumulates the returned `turns.Turn` exactly as before:

```text
message text      -> turns.NewAssistantTextBlock(message)
function calls    -> turns.NewToolCallBlock(id, name, args)
usage/stop reason -> metadata and inference-result persistence
```

That return-turn behavior should stay intact. The migration only changes the event vocabulary and correlation envelope.

## 3. Target canonical mapping

### 3.1 Provider-call lifecycle

Gemini should emit a provider-call lifecycle around the `GenerateContentStream` call:

```text
before reading stream:     EventProviderCallStarted
metadata chunks:           EventProviderCallMetadataUpdated when finish reason or usage changes
stream EOF:                EventProviderCallFinished
stream error:              EventRunFailed or EventError remains for generic errors, plus no provider-call success finish
```

Recommended correlation:

```go
providerCorr := events.BuildProviderCallCorrelation(
    "gemini",
    metadata.InferenceID,
    runID,
    0,
    responseID,
)
```

Gemini may not expose a stable response ID in the current SDK stream. That is fine. `ProviderCallID` and `CorrelationKey` should still be stable because `BuildProviderCallCorrelation` falls back to run/inference scope plus provider-call index.

### 3.2 Text segment lifecycle

Gemini text deltas should be modeled as explicit text segments:

```text
first non-empty genai.Text:        EventTextSegmentStarted
for each non-empty text delta:     EventTextDelta
end of actual text segment:        EventTextSegmentFinished
```

Rules:

- Do not emit `EventTextSegmentStarted` until actual text exists.
- Do not emit `EventTextSegmentFinished` if no text segment was started.
- EOF is allowed to close an active text segment, but it must be treated as closing the active text segment, not as provider-final text synthesis.
- Use a stable segment correlation for all text events in one Gemini message stream.

Recommended shape:

```go
textCorr := events.BuildSegmentCorrelation(providerCorr, "", 0, events.SegmentTypeText)
textCorr.Model = modelName
textCorr.SessionID = metadata.SessionID
textCorr.InferenceID = metadata.InferenceID
textCorr.TurnID = metadata.TurnID
```

The exact provider object ID may remain empty for Gemini text if the SDK does not expose one.

### 3.3 Tool lifecycle

Gemini function calls arrive as complete `genai.FunctionCall` parts rather than streaming argument fragments. The canonical mapping can therefore skip `EventToolCallArgumentsDelta` unless a future SDK shape exposes partial arguments.

For each function call:

```text
function call observed:     EventToolCallStarted
same function call ready:   EventToolCallRequested
```

Rules:

- Preserve the existing generated UUID behavior when Gemini does not provide a tool-call ID.
- Put that ID in both the event payload `ToolCallID` and typed correlation `ToolCallID`.
- Use `SegmentTypeTool` / `StreamKindToolCall` for tool correlation.
- Preserve `pendingCalls` and `turns.NewToolCallBlock` behavior unchanged.

Recommended shape:

```go
toolCorr := events.BuildSegmentCorrelation(providerCorr, toolCallID, toolIndex, events.SegmentTypeTool)
toolCorr.ToolCallID = toolCallID
```

Then emit:

```text
EventToolCallStarted(toolCallID, toolName)
EventToolCallRequested(toolCallID, toolName, inputJSON)
```

### 3.4 Provider metadata and inference result

Gemini already extracts finish reason, usage, duration, and has-tool-calls. Keep using those values for `engine.BuildInferenceResultFromEventMetadata`, but also publish them in canonical provider-call events:

```text
EventProviderCallMetadataUpdated(stop_reason, usage)  // optional, when metadata is observed
EventProviderCallFinished(stop_reason, finish_class, usage, duration_ms, has_tool_calls)
```

The finish class should match the shared provider-call result semantics used by OpenAI/Claude. At minimum:

```text
has tool calls        -> tool_calls_pending
normal text finish    -> completed
safety/error stop     -> provider_stop or error-like class, depending on existing helper conventions
```

If no shared finish-class helper exists for Gemini yet, add a small provider-local mapping and keep it covered by tests.

## 4. Local tool executor follow-up

Deleting legacy tool events also requires migrating the host-side executor in `pkg/inference/tools/base_executor.go`.

Current legacy mapping:

```text
PublishStart   -> EventToolCallExecute
PublishResult  -> EventToolCallExecutionResult
```

Canonical target:

```text
PublishStart   -> EventToolExecutionStarted
PublishResult  -> EventToolResultReady, then EventToolCallFinished
```

The executor does not currently receive the provider-call correlation from the model event that requested the tool. We should not reintroduce `metadata.Extra` as a hidden join mechanism. Acceptable options are:

1. Add a small context helper that carries an `events.Correlation` for the current tool call into execution.
2. Build a minimal correlation from the tool call ID/name when no provider context exists, leaving provider-call fields empty but preserving `ToolCallID` and `CorrelationKey` where possible.

The migration should prefer option 1 if the tool loop already has access to the canonical tool-request event correlation. If not, option 2 is acceptable as an interim executor-local correlation, but it should be documented in code and tests as execution-only provenance.

## 5. Test plan

Add or update Gemini tests to cover at least:

1. **Text-only stream**
   - emits provider-call started;
   - emits text segment started/delta/finished;
   - emits provider-call finished;
   - does not emit start/partial/final legacy events.

2. **Function-call stream**
   - emits provider-call started/finished;
   - emits tool-call started/requested;
   - does not synthesize a text segment if no text exists;
   - preserves generated tool call ID in both event payload and correlation.

3. **Mixed text and function-call stream**
   - text segment is closed explicitly;
   - tool request has separate tool correlation;
   - provider-call finished indicates `has_tool_calls=true`.

4. **Safety/empty response**
   - emits provider-call finished with stop reason/usage if available;
   - emits no text segment finished unless a text segment was started.

5. **Error while streaming**
   - emits the existing error signal or canonical failed signal as chosen by the broader run lifecycle;
   - does not emit provider-call finished as if successful;
   - does not emit legacy final.

6. **Tool executor canonicalization**
   - `BaseToolExecutor.PublishStart` emits `EventToolExecutionStarted`;
   - `BaseToolExecutor.PublishResult` emits `EventToolResultReady` and `EventToolCallFinished`;
   - no `EventToolCallExecute` or `EventToolCallExecutionResult` remains in active executor code.

## 6. Deletion gate impact

The final Geppetto deletion gate must wait for Gemini and the tool executor. After this migration, the following should be true in active runtime code:

```bash
rg "New(Start|PartialCompletion|Final|ThinkingPartial|ToolCall)Event" pkg cmd --glob '!doc/**'
rg "EventType(Start|PartialCompletion|Final|PartialThinking|ToolCall)" pkg cmd --glob '!doc/**'
rg "EventToolCallExecute|EventToolCallExecutionResult|NewToolCallExecuteEvent|NewToolCallExecutionResultEvent" pkg cmd --glob '!doc/**'
```

Expected result: only historical documentation, archived ticket notes, or deliberate compatibility-removal notes remain. There should be no active provider, executor, router, printer, sink, or JavaScript binding dependency on the deleted legacy event types.
