---
Title: Intern guide to extracting chat streaming from go-openai and normalizing provider reasoning deltas
Ticket: GP-58-CHAT-STREAM-NORMALIZATION
Status: active
Topics:
    - inference
    - streaming
    - reasoning
    - geppetto
    - chat
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/chat_stream.go
      Note: Typed chat stream delta that lacks Together reasoning alias support
    - Path: pkg/embeddings/openai.go
      Note: Explicitly out-of-scope OpenAI embeddings client retained on go-openai
    - Path: pkg/events/chat-events.go
      Note: Shared reasoning and partial-thinking event types
    - Path: pkg/steps/ai/openai/chat_stream.go
      Note: New direct chat-completions transport
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: Current chat streaming loop and event publication behavior
    - Path: pkg/steps/ai/openai/engine_openai_test.go
      Note: Fixture-driven regression coverage for the new stream path
    - Path: pkg/steps/ai/openai/helpers.go
      Note: Current request builder and tool-call adjacency logic
    - Path: pkg/steps/ai/openai/testdata/chat-stream/deepseek_reasoning_content.sse
      Note: Fixture proving reasoning_content fallback support
    - Path: pkg/steps/ai/openai/testdata/chat-stream/text_only.sse
      Note: Fixture proving text-only regression behavior remains unchanged
    - Path: pkg/steps/ai/openai/testdata/chat-stream/together_reasoning.sse
      Note: Fixture proving Together-style reasoning alias support
    - Path: pkg/steps/ai/openai/testdata/chat-stream/tool_calls_fragmented.sse
      Note: Fixture proving fragmented tool call merging and usage preservation
    - Path: pkg/steps/ai/openai/transcribe.go
      Note: Explicitly out-of-scope transcription client retained on go-openai
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Reference implementation for direct SSE handling and reasoning persistence
    - Path: ttmp/2026/03/27/GP-57-TOGETHER-THINKING--investigate-missing-together-qwen-thinking-stream-in-openai-compatible-chat-completions/sources/experiments/raw-sse.txt
      Note: Captured Together raw SSE proving delta.reasoning is present on the wire
ExternalSources:
    - https://github.com/openai/openai-go
    - https://docs.together.ai/docs/openai-api-compatibility
    - https://docs.together.ai/docs/deepseek-faqs
Summary: Detailed design guide for replacing go-openai in the chat streaming path with a provider-aware SSE decoder while leaving embeddings and transcription unchanged.
LastUpdated: 2026-03-27T19:07:20.862503894-04:00
WhatFor: ""
WhenToUse: ""
---



# Intern guide to extracting chat streaming from go-openai and normalizing provider reasoning deltas

## Executive summary

This ticket is about one specific architectural change: Geppetto should stop using `github.com/sashabaranov/go-openai` as the runtime decoder for OpenAI-compatible chat streaming. The reason is not that the library is globally unusable. The reason is that Geppetto currently depends on the library's typed stream structs at exactly the point where provider differences matter most, and that typed boundary is dropping provider-specific reasoning deltas.

The concrete failure we already observed is Together's Qwen chat stream. Together emits reasoning tokens in `choices[0].delta.reasoning` over `POST /v1/chat/completions`, while the current `go-openai` stream delta type only exposes `reasoning_content`. Geppetto then makes the problem worse by reading only `choice.Delta.Content` from the typed chunk and ignoring any reasoning field entirely. The result is that the provider does send thinking tokens, but Geppetto never sees them.

The recommended design is intentionally narrow and staged:

1. Replace only the chat-streaming runtime path with a direct HTTP + SSE decoder that Geppetto controls.
2. Normalize provider delta variants at the raw JSON boundary into a small internal model.
3. Keep `go-openai` in embeddings and transcription for now.
4. Revisit a broader SDK migration later, after the streaming path is stable and well-tested.

## Problem statement and scope

### The problem in one sentence

Geppetto currently treats "OpenAI-compatible chat streaming" as if the stream schema were stable across providers, but real providers diverge at the delta-field level, especially for reasoning or thinking output.

### Observed evidence

The existing chat engine in `pkg/steps/ai/openai/engine_openai.go` opens a typed stream with:

- `client.CreateChatCompletionStream(ctx, *req)` at `pkg/steps/ai/openai/engine_openai.go:206`

Inside the streaming loop, it currently extracts only:

- `delta = choice.Delta.Content` at `pkg/steps/ai/openai/engine_openai.go:263`

That means even if the provider sent some other delta field, Geppetto would not publish it.

The `go-openai` stream delta struct is also restrictive:

- `ReasoningContent string 'json:"reasoning_content,omitempty"'` at `/home/manuel/go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/chat_stream.go:19`

There is no `reasoning` field in that struct. So if Together returns `delta.reasoning`, the JSON decoder drops it before Geppetto can inspect it.

The prior Together experiment captured the raw SSE response directly in:

- `ttmp/.../GP-57-TOGETHER-THINKING.../sources/experiments/raw-sse.txt`

That stream clearly shows:

- `delta.content` is empty
- `delta.reasoning` contains the thinking tokens

So the failure is not "Together does not send thinking." The failure is "our chat-streaming stack discards the field."

### Scope of this ticket

In scope:

- the OpenAI-compatible chat streaming runtime path in `pkg/steps/ai/openai/`
- reasoning delta normalization across provider variants
- publishing reasoning events from chat-completions streams
- persisting reasoning blocks for chat-completions providers that expose reasoning text
- tests and fixtures for the new behavior

Out of scope:

- replacing the embeddings client in `pkg/embeddings/openai.go`
- replacing the transcription client in `pkg/steps/ai/openai/transcribe.go`
- rewriting the existing Open Responses engine in `pkg/steps/ai/openai_responses/`
- doing a repo-wide migration to `openai-go/v3`

## Current-state architecture

### The high-level flow today

The current OpenAI-compatible chat path looks like this:

```text
Turn
  -> MakeCompletionRequestFromTurn(...)
  -> go-openai ChatCompletionRequest
  -> go-openai CreateChatCompletionStream(...)
  -> typed chunks from library
  -> Geppetto reads Delta.Content only
  -> partial/final text events
  -> assistant text + tool call blocks appended to Turn
```

In more detail:

1. The engine factory selects the chat engine in `pkg/inference/engine/factory/factory.go`.
2. `OpenAIEngine.RunInference(...)` in `pkg/steps/ai/openai/engine_openai.go` builds a request from turn blocks.
3. `MakeCompletionRequestFromTurn(...)` in `pkg/steps/ai/openai/helpers.go` converts Geppetto turns into a `go_openai.ChatCompletionRequest`.
4. The engine opens a `go-openai` stream and loops over typed responses.
5. The engine publishes `partial` text events and later a `final` event.
6. Tool calls are merged and appended back to the output turn.

### Why the request builder is not the main bug

The request-builder side in `pkg/steps/ai/openai/helpers.go` is large, but it is not the part currently losing Together reasoning deltas. Its main responsibilities are:

- convert turn blocks into provider messages
- preserve tool-call / tool-result adjacency
- set model-specific request fields such as `max_tokens` vs `max_completion_tokens`
- attach structured output config
- apply per-turn inference overrides

Those responsibilities are all upstream of the bug. The thinking tokens are already being produced by the provider after the request is sent. The data loss happens on the streaming response decode path.

That is why this ticket should not start by rewriting `MakeCompletionRequestFromTurn(...)`.

### Why the Open Responses engine matters as a reference implementation

The Open Responses engine in `pkg/steps/ai/openai_responses/engine.go` is already doing the type of work we need:

- direct HTTP request construction
- explicit SSE reading with `bufio.Reader`
- per-event-name normalization
- custom event publication for reasoning and text
- persistence of reasoning blocks into the output turn

Important reference points:

- it publishes `partial` output chunks at `pkg/steps/ai/openai_responses/engine.go:298`
- it publishes `reasoning-text-delta` and mirrored `partial-thinking` at `pkg/steps/ai/openai_responses/engine.go:486-497`
- it persists a reasoning block at `pkg/steps/ai/openai_responses/engine.go:518-547`
- it normalizes `response.reasoning.delta` to `response.reasoning_text.delta` at `pkg/steps/ai/openai_responses/engine.go:1104-1112`

This is important because the new chat-streaming code should not invent a second independent reasoning model. It should align with the event semantics and turn persistence patterns that the Open Responses engine already established.

### Events already available for reasoning

Geppetto already has event types for reasoning output in `pkg/events/chat-events.go`:

- `EventTypePartialThinking` at `pkg/events/chat-events.go:19`
- `EventTypeReasoningTextDelta` at `pkg/events/chat-events.go:67`
- `EventTypeReasoningTextDone` at `pkg/events/chat-events.go:68`

Constructors already exist:

- `NewThinkingPartialEvent(...)` at `pkg/events/chat-events.go:356`
- `NewReasoningTextDelta(...)` at `pkg/events/chat-events.go:1028`
- `NewReasoningTextDone(...)` at `pkg/events/chat-events.go:1037`

This means the ticket is not blocked on event-system work. The missing piece is that the chat-completions engine never publishes those events.

### Turn model and reasoning blocks

The output turn is composed of `turns.Block` values in `pkg/turns/types.go`.

The OpenAI chat engine already appends:

- assistant text blocks
- tool call blocks

The Open Responses engine additionally appends reasoning blocks. The new chat-streaming path should follow that same model when the provider exposes reasoning text.

That will keep reasoning visible in a normalized Geppetto representation instead of leaving it buried in provider-specific stream chunks.

## Root cause analysis

### Root cause 1: typed stream schema mismatch

The current stream decoder is `go-openai`, which expects a specific JSON shape. The library includes `reasoning_content`, but the Together stream we observed exposes `reasoning`.

Observed mismatch:

```text
Together raw SSE:
choices[0].delta.reasoning

go-openai typed delta:
choices[0].delta.reasoning_content
```

Because the field names differ, the typed decoder silently discards the provider field.

### Root cause 2: Geppetto only consumes text deltas

Even if the library exposed a reasoning field, Geppetto would still lose the data today because the chat engine only reads `choice.Delta.Content`.

Current behavior:

```go
choice := response.Choices[0]
delta = choice.Delta.Content
if delta != "" {
    message += delta
    publish partial text event
}
```

There is no parallel path for:

- reasoning text
- provider-native reasoning aliases
- mirrored `partial-thinking`
- final reasoning block persistence

### Root cause 3: "OpenAI-compatible" is not enough of a contract

Providers often match the request envelope and endpoint names but diverge in:

- streaming delta fields
- reasoning field names
- usage shape
- tool-call chunk fragmentation
- finish reasons
- extra request knobs like `chat_template_kwargs`

The current stack assumes that if the endpoint is `/chat/completions`, the stream chunk type is effectively standardized. That assumption is the design bug.

## Design goals

The new design should satisfy all of the following:

1. Preserve current text-streaming behavior for plain text providers.
2. Capture reasoning deltas from providers that emit `reasoning`.
3. Capture reasoning deltas from providers that emit `reasoning_content`.
4. Reuse Geppetto's existing reasoning event model.
5. Reuse the Open Responses engine's reasoning persistence pattern where practical.
6. Avoid destabilizing embeddings and transcription.
7. Avoid forcing a broad SDK migration before the bug is fixed.

## Proposed architecture

### Summary

Introduce a small internal streaming layer under `pkg/steps/ai/openai/` that Geppetto owns. That layer will:

1. send the chat request over raw HTTP
2. read SSE frames
3. decode each frame into a small internal raw map or struct
4. normalize the provider-specific delta fields into one internal representation
5. feed normalized chunks into the existing engine logic for text, reasoning, tool calls, usage, and final turn persistence

### Recommended package shape

The exact filenames can vary, but the logical split should be:

```text
pkg/steps/ai/openai/
  engine_openai.go                  # orchestration, event publishing, turn mutation
  helpers.go                        # request-building, existing turn->message conversion
  chat_stream_client.go             # HTTP request + SSE transport
  chat_stream_sse.go                # SSE frame reader
  chat_stream_normalize.go          # provider delta normalization
  chat_stream_types.go              # internal structs for normalized chunks
  chat_stream_test.go               # parser/normalizer tests
  engine_openai_stream_test.go      # end-to-end engine tests
```

This split matters because there are two different concerns:

- transport and decode
- Geppetto-side semantics

If they are mixed together in one giant function, the code will become hard to test and hard to extend for the next provider quirk.

### New internal data model

Do not stream provider chunks directly into event publication. Normalize them first.

Suggested internal types:

```go
type ChatStreamEvent struct {
    DeltaText      string
    DeltaReasoning string
    ToolCalls      []NormalizedToolCallDelta
    Usage          *NormalizedUsage
    FinishReason   *string
    RawPayload     map[string]any
}

type NormalizedToolCallDelta struct {
    Index     int
    ID        string
    NameDelta string
    ArgsDelta string
}

type NormalizedUsage struct {
    PromptTokens     int
    CompletionTokens int
    CachedTokens     int
    ReasoningTokens  int
}
```

This does two things:

- it makes the engine independent from a third-party chat stream struct
- it gives us one stable place to normalize provider variants

### New runtime flow

```text
Turn
  -> MakeCompletionRequestFromTurn(...)
  -> serialize request body
  -> custom HTTP POST /chat/completions
  -> custom SSE reader
  -> normalize delta fields
  -> publish text/reasoning/tool events
  -> append assistant text / reasoning / tool blocks to Turn
```

### Sequence diagram

```text
OpenAIEngine.RunInference
  |
  | build request from Turn
  v
ChatStreamClient.Open(request)
  |
  | POST /v1/chat/completions
  v
SSE reader
  |
  | read "data: {...}"
  v
Normalizer
  |
  | delta.content           -> DeltaText
  | delta.reasoning         -> DeltaReasoning
  | delta.reasoning_content -> DeltaReasoning
  | delta.tool_calls        -> ToolCalls
  v
OpenAIEngine event loop
  |
  | publish partial / partial-thinking / reasoning-text-delta / tool-call
  | accumulate final assistant text
  | accumulate final reasoning text
  v
Turn mutation + final event
```

## Detailed design decisions

### Decision 1: remove `go-openai` from the streaming boundary only

Rationale:

- this fixes the observed bug directly
- it is a smaller refactor than a full SDK migration
- embeddings and transcription are not implicated in the Together failure

Evidence:

- chat streaming is in `pkg/steps/ai/openai/engine_openai.go`
- embeddings are isolated in `pkg/embeddings/openai.go`
- transcription is isolated in `pkg/steps/ai/openai/transcribe.go`

Consequence:

- `go-openai` remains in the module for now
- chat streaming becomes the first path Geppetto owns end-to-end

### Decision 2: keep request construction stable initially

Phase 1 should keep the existing `MakeCompletionRequestFromTurn(...)` logic as intact as possible.

There are two acceptable sub-options:

1. Minimal-change option:
   - still build a `go_openai.ChatCompletionRequest`
   - marshal it to JSON ourselves
   - stop using `CreateChatCompletionStream`

2. Cleaner option:
   - introduce an internal request struct for chat completions
   - migrate `helpers.go` away from `go_openai.ChatCompletionMessage` and friends

Recommended choice for this ticket:

- start with the minimal-change option

Why:

- it limits the blast radius
- it targets the actual bug
- it preserves the complex turn-to-request logic already encoded in `helpers.go`

### Decision 3: normalize provider variants explicitly

The normalizer should intentionally recognize multiple delta aliases:

- `delta.content`
- `delta.reasoning`
- `delta.reasoning_content`
- `delta.tool_calls`

If both `reasoning` and `reasoning_content` are somehow present, prefer one deterministic rule and document it. The simplest rule is:

1. if `reasoning` is non-empty, use it
2. else if `reasoning_content` is non-empty, use it

This should be explicit code, not implicit struct tags.

### Decision 4: align chat-completions reasoning events with Open Responses semantics

The chat-completions path should emit:

- `reasoning-text-delta`
- `partial-thinking`
- `reasoning-text-done`

This matters because UI and sinks already understand those event types from the Open Responses path. Reusing them avoids a parallel event ecosystem for the same user-visible concept.

### Decision 5: persist a reasoning block when the provider exposes raw reasoning text

If the chat provider emits live reasoning text, persist it to the output turn as `turns.BlockKindReasoning`, mirroring the existing pattern in the Open Responses engine.

That makes the reasoning visible for:

- debugging
- history replay
- downstream tooling
- future migrations to other providers

If the provider does not expose reasoning separately, do not synthesize a reasoning block from assistant text.

## Proposed algorithms

### SSE read loop

Pseudocode:

```go
func ReadChatStream(respBody io.Reader, onEvent func(eventName string, payload []byte) error) error {
    r := bufio.NewReader(respBody)
    var eventName string
    var dataBuf strings.Builder

    flushPending := func() error {
        if dataBuf.Len() == 0 {
            return nil
        }
        err := onEvent(eventName, []byte(dataBuf.String()))
        eventName = ""
        dataBuf.Reset()
        return err
    }

    for {
        line, err := r.ReadString('\n')
        if err == io.EOF {
            return flushPending()
        }
        if err != nil {
            return err
        }

        line = strings.TrimRight(line, "\r\n")
        switch {
        case strings.HasPrefix(line, "event:"):
            eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
        case strings.HasPrefix(line, "data:"):
            data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
            if data == "[DONE]" {
                return flushPending()
            }
            dataBuf.WriteString(data)
        case line == "":
            if err := flushPending(); err != nil {
                return err
            }
        }
    }
}
```

### Delta normalization

Pseudocode:

```go
func NormalizeChatChunk(raw map[string]any) ChatStreamEvent {
    ev := ChatStreamEvent{}

    choice := firstChoice(raw)
    delta := asMap(choice["delta"])

    ev.DeltaText = asString(delta["content"])

    if s := asString(delta["reasoning"]); s != "" {
        ev.DeltaReasoning = s
    } else if s := asString(delta["reasoning_content"]); s != "" {
        ev.DeltaReasoning = s
    }

    ev.ToolCalls = normalizeToolCalls(delta["tool_calls"])
    ev.Usage = normalizeUsage(raw["usage"])
    ev.FinishReason = maybeString(choice["finish_reason"])
    ev.RawPayload = raw
    return ev
}
```

### Engine consumption logic

Pseudocode:

```go
var textBuf strings.Builder
var reasoningBuf strings.Builder
toolCallMerger := NewToolCallMerger()

for ev := range normalizedStream {
    if ev.DeltaReasoning != "" {
        reasoningBuf.WriteString(ev.DeltaReasoning)
        publish(NewReasoningTextDelta(meta, ev.DeltaReasoning))
        publish(NewThinkingPartialEvent(meta, ev.DeltaReasoning, reasoningBuf.String()))
    }

    if ev.DeltaText != "" {
        textBuf.WriteString(ev.DeltaText)
        publish(NewPartialCompletionEvent(meta, ev.DeltaText, textBuf.String()))
    }

    if len(ev.ToolCalls) > 0 {
        toolCallMerger.AddNormalized(ev.ToolCalls)
    }

    if ev.Usage != nil {
        updateMetadataUsage(meta, ev.Usage)
    }

    if ev.FinishReason != nil {
        stopReason = ev.FinishReason
    }
}

if reasoningBuf.Len() > 0 {
    publish(NewReasoningTextDone(meta, reasoningBuf.String()))
    append reasoning block to Turn
}

append assistant text block
append tool call blocks
publish final event
```

## Implementation plan

### Phase 1: extract the transport and normalize reasoning

Files to touch:

- `pkg/steps/ai/openai/engine_openai.go`
- `pkg/steps/ai/openai/helpers.go`
- new `pkg/steps/ai/openai/chat_stream_*.go` files

Steps:

1. Introduce an internal streaming client that accepts:
   - base URL
   - API key
   - HTTP client
   - marshaled request body
2. Move raw HTTP and SSE reading into that client.
3. Add a normalizer that emits `ChatStreamEvent`.
4. Refactor the engine loop to consume `ChatStreamEvent` instead of `go-openai` typed chunks.
5. Add support for reasoning accumulation and event publication.

Success criteria:

- text-only streams still behave exactly as before
- Together `delta.reasoning` appears as live `partial-thinking` and `reasoning-text-delta`

### Phase 2: persist reasoning blocks from chat-completions streams

Files to touch:

- `pkg/steps/ai/openai/engine_openai.go`
- tests under `pkg/steps/ai/openai/`

Steps:

1. Accumulate final reasoning text separately from assistant text.
2. On stream completion, emit `reasoning-text-done`.
3. Append a `turns.BlockKindReasoning` block before or alongside final assistant/tool blocks using the same high-level semantics as the Open Responses engine.

Success criteria:

- output turn includes a reasoning block when reasoning was streamed
- no reasoning block is created for ordinary text-only streams

### Phase 3: harden tool-call and usage normalization

Files to touch:

- new normalizer files
- `pkg/steps/ai/openai/helpers.go`
- tests

Steps:

1. Preserve tool-call chunk merging semantics already implemented by `ToolCallMerger`.
2. Normalize usage if a provider sends it in the final chunk.
3. Ensure finish reason handling still works even with provider variance.

Success criteria:

- tool-calling behavior does not regress
- token usage still appears in `EventMetadata`

### Phase 4: decide whether to remove `go-openai` request types from chat helpers

This is optional follow-up work, not required to fix the bug.

Files potentially affected:

- `pkg/steps/ai/openai/helpers.go`
- `pkg/steps/ai/openai/engine_openai.go`
- `pkg/inference/tools/adapters.go`

Goal:

- remove residual `go-openai` request structs from the chat package if the team wants a cleaner separation

This phase should happen only after the stream extraction is already tested and stable.

## Testing and validation strategy

### What to test first

The most important tests are fixture-driven stream tests. They should not hit the real network.

Minimum fixture set:

1. OpenAI-style text-only stream
2. Together-style stream with `delta.reasoning`
3. DeepSeek-style stream with `delta.reasoning_content`
4. stream with fragmented `tool_calls`
5. stream with final usage chunk

### Why fixtures matter

This code is fundamentally about decoding provider wire formats. Fixture tests are the clearest way to lock down behavior and prevent regressions.

Recommended fixture location:

```text
pkg/steps/ai/openai/testdata/chat-stream/
  together_reasoning.sse
  deepseek_reasoning_content.sse
  text_only.sse
  tool_calls_fragmented.sse
```

### Engine-level assertions

For the Together reasoning fixture, assert all of the following:

1. at least one `reasoning-text-delta` event is published
2. at least one `partial-thinking` event is published
3. assistant text still streams normally if present
4. final output turn contains a reasoning block
5. final output turn still contains assistant text and tool calls in the expected order

### Non-goals for tests

Do not make the first test wave depend on:

- live Together credentials
- timing-sensitive network behavior
- full end-to-end CLI harnesses

Those can be useful later, but the core regression needs deterministic local tests first.

## Alternatives considered

### Alternative A: just upgrade `go-openai`

Rejected for this ticket.

Why:

- the current latest version we checked still exposes `reasoning_content`, not Together's `reasoning`
- upgrading would not fix the field mismatch

### Alternative B: fork `go-openai`

Possible, but not recommended as the primary plan.

Why:

- it makes Geppetto maintain a fork for provider quirks
- the next provider mismatch would put us back in the same position
- it still leaves our engine overly coupled to third-party stream structs

### Alternative C: migrate directly to `openai-go/v3`

Reasonable long-term direction, but not the best first fix for this ticket.

Pros:

- official SDK
- better escape hatches for undocumented fields and raw JSON
- better fit for OpenAI-native Responses work

Cons for this ticket:

- it does not eliminate the need for provider-aware normalization
- it is a larger migration than necessary to fix Together reasoning
- it risks mixing two separate projects: "fix Together streaming" and "modernize SDK choice"

### Alternative D: full custom chat client immediately

Partially accepted.

This ticket recommends a full custom streaming transport, but not an immediate full replacement of every chat request type and helper struct.

That split keeps the initial refactor focused.

## Risks and sharp edges

### Risk 1: accidentally regress tool calling

The current chat engine has nontrivial tool-call handling:

- tool fragments are merged by index in `ToolCallMerger`
- tool results must remain adjacent in request construction
- final tool blocks are appended back to the turn in order

If the new stream normalizer mishandles fragmented `tool_calls`, tool execution will regress even if reasoning works.

### Risk 2: duplicated reasoning publication

If a provider sends both incremental reasoning deltas and a final full reasoning field, naive code can duplicate tokens in the accumulated reasoning buffer.

Use the same style of suffix or backfill guards already present in the Open Responses engine when a provider can repeat content on done events.

### Risk 3: mixing text and reasoning buffers

Do not append reasoning deltas to the assistant text buffer. Keep:

- `message` or `textBuf` for assistant output
- `reasoningBuf` for thinking output

This separation is necessary for correct UI behavior and correct turn persistence.

### Risk 4: over-scoping the refactor

If this ticket expands into "replace every OpenAI SDK usage," it will slow down and get riskier. The goal is narrower:

- own the chat stream boundary
- leave embeddings and transcription alone

## Suggested code review checklist

Reviewers should verify:

1. no runtime dependency on `CreateChatCompletionStream` remains in the chat engine
2. Together-style `delta.reasoning` is normalized into Geppetto reasoning events
3. DeepSeek-style `delta.reasoning_content` also works
4. text-only streams still emit the same `partial` and `final` events
5. tool-call merges and final turn block ordering are unchanged
6. embeddings and transcription still use `go-openai` and are not behaviorally changed

## References

### Primary Geppetto files

- `pkg/steps/ai/openai/engine_openai.go`
- `pkg/steps/ai/openai/helpers.go`
- `pkg/steps/ai/openai_responses/engine.go`
- `pkg/events/chat-events.go`
- `pkg/inference/engine/factory/factory.go`
- `pkg/embeddings/openai.go`
- `pkg/steps/ai/openai/transcribe.go`
- `pkg/turns/types.go`

### External library and evidence files

- `github.com/sashabaranov/go-openai` stream delta type at `/home/manuel/go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/chat_stream.go`
- Together raw SSE capture at `ttmp/2026/03/27/GP-57-TOGETHER-THINKING--investigate-missing-together-qwen-thinking-stream-in-openai-compatible-chat-completions/sources/experiments/raw-sse.txt`

### External API references

- OpenAI official Go SDK: `https://github.com/openai/openai-go`
- Together OpenAI compatibility docs: `https://docs.together.ai/docs/openai-api-compatibility`
- Together reasoning examples and FAQ: `https://docs.together.ai/docs/deepseek-faqs`
