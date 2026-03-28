---
Title: Postmortem and intern guide to the Together Qwen thinking-stream bug
Ticket: GP-57-TOGETHER-THINKING
Status: active
Topics:
    - geppetto
    - together
    - reasoning
    - streaming
    - openai-compatibility
    - bug
    - pinocchio
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/webchat/sem_translator.go
      Note: Pinocchio semantic translation layer for reasoning events.
    - Path: pkg/inference/engine/factory/factory.go
      Note: Engine selection between chat-completions and Responses-style paths.
    - Path: pkg/steps/ai/openai/chat_stream.go
      Note: Provider-aware SSE decoder that normalizes reasoning deltas.
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: Runtime chat-stream entrypoint and the stream=true fix.
    - Path: pkg/steps/ai/openai/helpers.go
      Note: Request builder that shows why runtime stream forcing must happen later.
    - Path: ttmp/2026/03/27/GP-57-TOGETHER-THINKING--investigate-missing-together-qwen-thinking-stream-in-openai-compatible-chat-completions/scripts/together_qwen_probe.go
      Note: Primary experiment harness used throughout the postmortem.
ExternalSources:
    - https://docs.together.ai/docs/openai-api-compatibility
    - https://docs.together.ai/docs/chat-overview
    - https://huggingface.co/Qwen/Qwen3.5-9B/blob/main/README.md
    - https://huggingface.co/blog/open-responses
    - https://github.com/sashabaranov/go-openai/blob/master/chat_stream.go
Summary: Detailed analysis, design notes, and postmortem for the Together Qwen reasoning-stream bug, including the architectural context an intern needs to understand Geppetto, Pinocchio, the experiment matrix, the root causes, the implemented fix, and the remaining SDK-specific follow-up.
LastUpdated: 2026-03-28T16:20:00-04:00
WhatFor: ""
WhenToUse: ""
---


# Postmortem and intern guide to the Together Qwen thinking-stream bug

## How to read this document

This report is written for a new intern who has not worked in Geppetto or Pinocchio before. It explains the bug at three levels:

- The product level: what the user observed and why it mattered.
- The system level: which components participate in a profile-backed chat streaming run.
- The code level: which files, structs, and event shapes you need to understand to safely change this area.

If you only need the short version, read:

1. Executive summary
2. Root cause
3. Fix that landed
4. Remaining open issue

If you need to extend or rework this system, read the full sections in order.

## Executive summary

The user reported that the profile `together-qwen-3.5-9b` did not show thinking deltas. The problem looked at first like a single “Together reasoning field mismatch” issue, but the investigation showed two separate failures:

1. A Geppetto runtime bug:
   The custom chat-completions streaming path was opening an SSE reader but did not reliably force `stream=true` in the actual request body. When the profile did not explicitly request streaming, Together returned a non-streaming response and Geppetto observed `chunks_received=0`.

2. A library boundary mismatch:
   Together’s raw stream exposes reasoning as `choices[0].delta.reasoning`, while `github.com/sashabaranov/go-openai` models the undocumented reasoning field as `reasoning_content`. As a result, the SDK did not surface Together’s reasoning payload in its typed stream delta.

The implemented fix addressed the Geppetto runtime bug. Geppetto now forces `stream=true` at the request boundary used by the custom SSE path, and live runs against the real Together profile now emit `reasoning-text-delta` and `partial-thinking` events.

The remaining open question is narrower and cleaner now:

- Why does `go-openai` receive the Together stream but expose only repeated `role="assistant"` chunks and not the reasoning or text deltas we can see in raw SSE?

That remaining issue is documented, artifact-backed, and ready for follow-up work.

## What the system is

There are three layers you need to keep straight:

1. Geppetto:
   The inference/runtime library. This is where engines, streaming decode, turn building, tool-call handling, and event emission live.

2. Pinocchio:
   The application layer that uses Geppetto. It resolves profiles, launches engines, receives events, and turns those events into UI/timeline/webchat semantics.

3. Provider APIs:
   The external inference services such as OpenAI, Together, Claude, Gemini, and compatible routers. Even when providers claim “OpenAI compatibility,” they are not identical at the streaming field level.

In this ticket, the failing path was:

```text
Pinocchio profile
  -> resolved InferenceSettings
  -> Geppetto StandardEngineFactory
  -> Geppetto OpenAI chat-completions engine
  -> custom HTTP + SSE stream reader
  -> Geppetto events
  -> Pinocchio semantic translation / UI
```

The profile used for the investigation was in:

```text
~/.config/pinocchio/profiles.yaml
```

and the specific profile slug was:

```text
together-qwen-3.5-9b
```

## Why this bug was confusing

This bug was confusing because several different layers could plausibly have been responsible:

- The model might not expose thinking at all.
- Together might expose thinking only in raw text, not a separate field.
- Geppetto might not normalize the provider field name.
- Pinocchio might drop the event after Geppetto emits it.
- The SDK might silently erase the provider-specific field.
- The request body might not actually be streaming.

All of those were reasonable hypotheses at the start.

The investigation worked because it deliberately compared three paths instead of guessing:

- Raw HTTP/SSE control
- `go-openai` streaming
- Geppetto runtime streaming

That experiment matrix is saved under:

```text
/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/ttmp/2026/03/27/GP-57-TOGETHER-THINKING--investigate-missing-together-qwen-thinking-stream-in-openai-compatible-chat-completions/sources/experiments/
```

## The relevant subsystems

### 1. Profile resolution

The first concept to understand is that Pinocchio often does not hand-author `InferenceSettings` in code. It resolves them from profile registries.

Key code paths:

- `pinocchio/cmd/pinocchio/cmds/js.go`
- `pinocchio/pkg/cmds/profilebootstrap/profile_selection.go`
- `geppetto/pkg/js/modules/geppetto/api_profiles.go`

At a high level:

- A profile registry stack is loaded.
- A profile slug is selected.
- The registry resolves and merges settings into a concrete `InferenceSettings` object.
- That object is then used to create an engine.

Pseudocode:

```go
profile := registry.ResolveEngineProfile(profileSlug)
settings := profile.InferenceSettings
engine := factory.CreateEngine(settings)
```

This matters because a profile may omit or default fields such as `chat.stream`. Bugs sometimes appear only when profile defaults differ from assumptions inside engine code.

### 2. Engine selection

Engine selection happens in:

```text
/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/inference/engine/factory/factory.go
```

This file decides which runtime engine is instantiated based on `settings.Chat.ApiType`.

Important distinction:

- `openai`
  routes to the legacy OpenAI-compatible chat-completions engine.
- `open-responses` / `openai-responses`
  routes to the Responses-style engine.

This ticket is specifically about the OpenAI-compatible chat-completions path, not the Responses path.

That distinction matters because Open Responses has a different reasoning model and event model. The Hugging Face Open Responses article explains that Responses-style systems can expose raw reasoning content, summaries, and encrypted traces in a more formalized way than legacy completions-style APIs. Source: [Hugging Face Open Responses blog](https://huggingface.co/blog/open-responses).

### 3. Request construction

Before the network call, Geppetto builds an OpenAI-style `ChatCompletionRequest` in:

```text
/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai/helpers.go
```

This file is responsible for:

- Converting turn blocks into provider messages
- Preserving tool-call adjacency rules
- Applying structured output settings
- Applying some provider-agnostic generation parameters
- Setting `Stream` based on chat settings

This is where the first important subtlety appears:

- `MakeCompletionRequestFromTurn(...)` is a request builder.
- `RunInference(...)` is the runtime behavior.

The engine runtime always uses the streaming code path, even if the profile-level `chat.stream` flag is false or omitted. That means the runtime must not blindly trust the request builder’s `Stream` field.

### 4. Custom chat streaming

The Geppetto-owned stream decoder is in:

```text
/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai/chat_stream.go
```

This file is central to the fix and to the long-term design.

Its job is:

- Marshal the outgoing chat-completions request
- Open an HTTP request with `Accept: text/event-stream`
- Read SSE frames
- Parse each `data:` payload as JSON
- Normalize provider-specific deltas into a single internal shape

The key normalization logic is:

```go
if s, ok := stringValue(delta["reasoning"]); ok && s != "" {
    ret.DeltaReasoning = s
} else if s, ok := stringValue(delta["reasoning_content"]); ok && s != "" {
    ret.DeltaReasoning = s
}
```

That means Geppetto already knows how to accept either:

- Together-style `delta.reasoning`
- DeepSeek-style `delta.reasoning_content`

This is why the Geppetto custom stream can succeed where a rigid typed SDK fails.

### 5. Event publishing

Once Geppetto decodes a normalized stream event, it publishes inference events in:

```text
/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai/engine_openai.go
```

Important event types:

- `start`
- `partial`
- `final`
- `reasoning-text-delta`
- `partial-thinking`
- `info` with `thinking-started`
- `info` with `thinking-ended`

The reason there are both `reasoning-text-delta` and `partial-thinking` events is compatibility and layering:

- `reasoning-text-delta` is the specific reasoning-stream event.
- `partial-thinking` mirrors it for existing renderers and consumers that already understand “thinking stream” semantics.

### 6. Pinocchio semantic translation

Pinocchio’s webchat/timeline layer translates Geppetto events into semantic events in:

```text
/home/manuel/workspaces/2026-03-27/use-open-responses/pinocchio/pkg/webchat/sem_translator.go
```

Relevant mappings include:

- `thinking-started` -> `llm.thinking.start`
- `partial-thinking` -> `llm.thinking.delta`
- `thinking-ended` -> `llm.thinking.final`

This means that once Geppetto emits the right thinking events, Pinocchio already knows how to project them into webchat semantics. In other words, Pinocchio was not the primary bug in this incident.

## Architecture diagram

### Logical data flow

```text
┌──────────────────────────────┐
│ ~/.config/pinocchio/profiles │
└──────────────┬───────────────┘
               │ resolve profile
               v
┌──────────────────────────────┐
│ InferenceSettings            │
│ api_type=openai             │
│ engine=Qwen/Qwen3.5-9B      │
│ base_url=https://.../v1     │
└──────────────┬───────────────┘
               │
               v
┌──────────────────────────────┐
│ StandardEngineFactory        │
│ factory.go                   │
└──────────────┬───────────────┘
               │ selects OpenAI engine
               v
┌──────────────────────────────┐
│ OpenAIEngine                 │
│ engine_openai.go             │
└──────────────┬───────────────┘
               │ build request
               v
┌──────────────────────────────┐
│ helpers.go                   │
│ ChatCompletionRequest        │
└──────────────┬───────────────┘
               │ stream HTTP request
               v
┌──────────────────────────────┐
│ chat_stream.go               │
│ HTTP + SSE + normalization   │
└──────────────┬───────────────┘
               │
               v
┌──────────────────────────────┐
│ Geppetto events              │
│ reasoning-text-delta         │
│ partial-thinking             │
│ final                        │
└──────────────┬───────────────┘
               │
               v
┌──────────────────────────────┐
│ Pinocchio sem_translator.go  │
│ llm.thinking.delta           │
│ llm.final                    │
└──────────────────────────────┘
```

### Experiment matrix

```text
                    Same real profile
                           │
      ┌────────────────────┼────────────────────┐
      │                    │                    │
      v                    v                    v
  Raw HTTP/SSE         go-openai          Geppetto custom stream
      │                    │                    │
      │                    │                    │
  reasoning visible?   reasoning visible?   reasoning visible?
      │                    │                    │
     yes                  no                  yes
```

This comparison is what turned a vague bug report into a clear root-cause analysis.

## The experiments

The experiment scripts are:

- `scripts/together_qwen_probe.go`
- `scripts/run_together_qwen_experiments.sh`
- `scripts/pinocchio_together_stream_probe.js`

The JS probe is preserved for completeness, but the Go probe is the authoritative debugger for this issue because the current `pinocchio js` command does not keep the async runtime alive long enough to act as a reliable streaming inspection harness.

### Experiment 1: Raw Together SSE control

Goal:

- Verify whether Together and the model actually emit reasoning deltas.

Result:

- Yes.
- The raw stream showed `choices[0].delta.reasoning`.

Representative output shape:

```text
data: {"choices":[{"delta":{"role":"assistant","content":"","reasoning":"Thinking"}}],...}
data: {"choices":[{"delta":{"role":"assistant","content":"","reasoning":" Process"}}],...}
```

Interpretation:

- The provider and model are capable of producing distinct reasoning deltas.
- The user’s complaint was valid.

### Experiment 2: `go-openai`

Goal:

- See what the typed SDK surfaces for the same profile.

Result:

- The stream advanced, but the captured output consisted of repeated `role="assistant"` chunks and no useful `reasoning_content` or `content` values in the bounded run.

Captured request body:

```json
{
  "model": "Qwen/Qwen3.5-9B",
  "messages": [
    {
      "role": "user",
      "content": "What is 17 * 23? Think step by step, then give a short final answer."
    }
  ],
  "max_tokens": 64,
  "temperature": 1,
  "top_p": 0.95,
  "stream": true,
  "stream_options": {
    "include_usage": true
  }
}
```

Interpretation:

- The transport is not entirely dead.
- The problem appears to be at the typed decode boundary or how the SDK reads the provider stream.

Important library fact:

- `go-openai` models the reasoning field as `reasoning_content` in its stream delta type. Source: [go-openai chat_stream.go](https://github.com/sashabaranov/go-openai/blob/master/chat_stream.go)

That is compatible with some providers, but not Together’s `delta.reasoning`.

### Experiment 3: Geppetto custom stream

Goal:

- Determine whether Geppetto’s own stream parser can observe the provider field and publish the right events.

Initial result before fix:

- No chunks observed.
- Geppetto logged `chunks_received=0`.

Post-fix result:

- Geppetto observed and published reasoning chunks immediately.

Representative logs after the fix:

```text
chunk=2 reasoning_delta="Thinking"
chunk=3 reasoning_delta=" Process"
```

Interpretation:

- Geppetto’s normalization layer is correct enough to understand Together’s `delta.reasoning`.
- The initial “no chunks” result was caused by the request-shape bug, not by the normalization logic itself.

## Root cause analysis

## Root cause 1: request-shape mismatch

### What happened

`RunInference(...)` always used the streaming execution path, but the request object it received from `MakeCompletionRequestFromTurn(...)` could still have `Stream == false` when the profile did not explicitly set streaming.

That meant the runtime did something like:

```go
req := MakeCompletionRequestFromTurn(turn) // may contain stream=false
stream := openChatCompletionStream(ctx, cfg, req)
for {
    event := stream.Recv() // expecting SSE
}
```

If Together received `stream=false`, it returned a non-streaming JSON response body. The custom SSE parser looked for `data:` frames, found none, and ended with zero chunks.

### Why this is a bug

The runtime behavior and request shape were inconsistent.

- Runtime assumption: “I am streaming.”
- Request reality: “The body may not ask the provider to stream.”

That mismatch should not exist.

### Why the bug was easy to miss

The logs from request construction still showed the pre-forced request shape from the helper stage. That made it easy to think “the request is non-streaming, so maybe the whole engine still uses the old path,” when in fact the runtime had already switched to the new custom SSE path. The problem was the final outgoing body at the runtime boundary.

## Root cause 2: provider field mismatch at the SDK boundary

### What happened

Together emits reasoning deltas as:

```json
{
  "choices": [
    {
      "delta": {
        "reasoning": "..."
      }
    }
  ]
}
```

But `go-openai` expects:

```json
{
  "choices": [
    {
      "delta": {
        "reasoning_content": "..."
      }
    }
  ]
}
```

As documented in the library source, the SDK’s stream delta struct includes:

```go
ReasoningContent string `json:"reasoning_content,omitempty"`
```

and does not model Together’s `reasoning` alias in the typed field set. Source: [go-openai chat_stream.go](https://github.com/sashabaranov/go-openai/blob/master/chat_stream.go)

### Why this matters

This is exactly the kind of boundary where “OpenAI-compatible” stops being enough. The request format may be close enough for the provider to accept it, while the response schema still differs at the delta-field level.

### Why Geppetto’s custom parser succeeds

Geppetto’s normalization logic explicitly checks both:

- `delta.reasoning`
- `delta.reasoning_content`

That is the correct abstraction boundary for a router/client that wants to support multiple OpenAI-style providers without relying on a single undocumented field spelling.

## The fix that landed

The implemented product fix was in:

```text
/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai/engine_openai.go
```

Specifically:

```go
req.Stream = true
if req.StreamOptions == nil && !strings.Contains(strings.ToLower(req.Model), "mistral") {
    req.StreamOptions = &go_openai.StreamOptions{IncludeUsage: true}
}
```

Design rationale:

- `RunInference(...)` is the streaming runtime.
- Therefore `RunInference(...)` must ensure the outgoing body is streaming.
- The fix belongs at the runtime boundary, not in profile defaults.

Regression coverage:

- A new test asserts that the outgoing body contains `stream=true` even when `ChatSettings.Stream` is false.

File:

```text
/home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/pkg/steps/ai/openai/engine_openai_test.go
```

## Why this design is the right near-term design

The current design for chat streaming should be:

- Geppetto owns the chat-completions streaming decode boundary.
- Geppetto normalizes provider-specific delta field names into internal events.
- Geppetto continues to use the typed SDK only where it is still buying us value and not erasing important fields.

That leads to a practical split:

- Chat-completions streaming:
  Geppetto-owned HTTP + SSE + normalization
- Embeddings:
  `go-openai` still acceptable
- Transcription:
  `go-openai` still acceptable

This matches the current codebase state.

## Why we did not switch the whole repo to another library

A new intern might ask:

“Why not just delete `go-openai` everywhere or migrate everything to another SDK immediately?”

The answer is scope control.

This incident required a targeted fix to restore product behavior. The smallest high-confidence repair was:

- keep the Geppetto-owned streaming layer
- fix the `stream=true` request bug
- preserve the normalized reasoning aliases

A full SDK migration would be a larger refactor with different risks and should be justified separately.

## Remaining open issue

The main unresolved item after the landed fix is:

Why does `go-openai` surface only repeated role chunks for Together Qwen in our bounded experiments, despite receiving a valid stream?

This remains open because:

- The raw control proves the provider emits reasoning.
- The Geppetto custom stream proves the payload is parseable and useful.
- The `go-openai` artifact still does not surface that data usefully.

That remaining issue does not block the current product fix, but it does matter if anyone wants to rely on the SDK stream directly in the future.

## Recommended next implementation steps

If you are continuing this ticket, do the work in this order.

### Phase 1: document and stabilize

- Update the original GP-57 design guide so it reflects the landed request-boundary fix.
- Keep the experiment matrix current when you change anything.
- Do not delete the raw control path; it is your ground truth.

### Phase 2: analyze `go-openai`

- Capture raw frames and decoded frames side by side.
- Confirm whether the SDK drops `delta.reasoning` silently or whether some other decode quirk is happening.
- Decide whether we need a local fork/patch, or whether the library should simply remain outside the chat-stream runtime boundary.

### Phase 3: provider-native extras

- Compare the raw control request body with the Geppetto request body.
- Decide whether Together-specific extras should be modeled in provider settings.

Examples worth evaluating:

- `chat_template_kwargs.enable_thinking`
- `top_k`
- other Qwen-oriented sampling defaults

This is not necessary for the core fix, but it may improve determinism and parity with model guidance from the Qwen model card.

## Pseudocode for the desired long-term chat streaming contract

```go
func RunInference(ctx context.Context, turn *Turn) (*Turn, error) {
    req := MakeCompletionRequestFromTurn(turn)

    // Runtime contract: this path is always streaming.
    req.Stream = true
    ensureStreamOptions(&req)

    stream := openChatCompletionStream(ctx, cfg, req)
    defer stream.Close()

    var assistantText string
    var reasoningText string

    for {
        ev, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            publishError(err)
            return nil, err
        }

        if ev.DeltaReasoning != "" {
            if reasoningText == "" {
                publishInfo("thinking-started")
            }
            reasoningText += ev.DeltaReasoning
            publishReasoningDelta(ev.DeltaReasoning)
            publishThinkingPartial(ev.DeltaReasoning, reasoningText)
        }

        if ev.DeltaText != "" {
            assistantText += ev.DeltaText
            publishPartial(ev.DeltaText, assistantText)
        }

        mergeToolCalls(ev.ToolCalls)
        updateUsage(ev.Usage)
        updateFinishReason(ev.FinishReason)
    }

    if reasoningText != "" {
        persistReasoningBlock(reasoningText)
        publishReasoningDone(reasoningText)
        publishInfo("thinking-ended")
    }

    persistAssistantText(assistantText)
    publishFinal(assistantText)
    return updatedTurn, nil
}
```

## API reference notes

### Together

Together documents OpenAI compatibility around `chat.completions`, base URL replacement, and streaming support. The relevant docs emphasize chat-completions style usage, not the OpenAI Responses API. Sources:

- [Together OpenAI Compatibility](https://docs.together.ai/docs/openai-api-compatibility)
- [Together Chat Overview](https://docs.together.ai/docs/chat-overview)

### Qwen 3.5 9B

The Qwen model card says Qwen 3.5 models operate in thinking mode by default and recommends thinking-oriented sampling settings such as `temperature=1.0`, `top_p=0.95`, and `top_k=20`. It also shows OpenAI-compatible deployments passing thinking configuration through provider-specific extra-body fields. Source:

- [Qwen/Qwen3.5-9B README](https://huggingface.co/Qwen/Qwen3.5-9B/blob/main/README.md)

### Open Responses

Open Responses is relevant here as architectural context, not because it was the failing runtime in this ticket. It formalizes richer reasoning items such as raw `content`, `encrypted_content`, and `summary`. This is useful context for understanding why legacy chat-completions compatibility often needs provider-specific normalization. Source:

- [Hugging Face Open Responses blog](https://huggingface.co/blog/open-responses)

## File-by-file reading guide for a new intern

Start in this order.

1. `geppetto/pkg/inference/engine/factory/factory.go`
   Learn how provider selection happens.

2. `geppetto/pkg/steps/ai/openai/helpers.go`
   Learn how turns become OpenAI-style request messages.

3. `geppetto/pkg/steps/ai/openai/chat_stream.go`
   Learn how the raw provider stream becomes normalized internal events.

4. `geppetto/pkg/steps/ai/openai/engine_openai.go`
   Learn where the runtime fix landed and how events are published.

5. `geppetto/pkg/steps/ai/openai/engine_openai_test.go`
   Learn what the intended contract is after the fix.

6. `pinocchio/pkg/webchat/sem_translator.go`
   Learn how thinking events become semantic/UI events.

7. `ttmp/.../scripts/together_qwen_probe.go`
   Learn how the experiment matrix was constructed.

8. `ttmp/.../reference/01-investigation-diary.md`
   Learn the actual debugging journey and sharp edges.

## Reviewer checklist

- Does `RunInference(...)` always send a streaming request body on the custom SSE path?
- Does `chat_stream.go` still normalize both `reasoning` and `reasoning_content`?
- Do the real Together artifacts still show raw reasoning in `raw-sse.txt` and Geppetto reasoning events in `geppetto.txt`?
- Does `go-openai.txt` still reproduce the remaining mismatch?
- Does `docmgr doctor --ticket GP-57-TOGETHER-THINKING --stale-after 30` pass?

## Final conclusions

This incident is a good example of why router/client code needs to own its streaming normalization boundary for “OpenAI-compatible” providers.

The main lessons are:

- Compatibility is not identity.
- Always compare raw provider output against typed SDK output before blaming the UI.
- The runtime boundary must enforce the transport contract it assumes.
- A three-path experiment matrix is often faster and more reliable than extended speculation.

The landed fix restored Geppetto’s ability to surface Together Qwen reasoning deltas. The remaining work is now small, well-scoped, and documented with reproducible artifacts.
