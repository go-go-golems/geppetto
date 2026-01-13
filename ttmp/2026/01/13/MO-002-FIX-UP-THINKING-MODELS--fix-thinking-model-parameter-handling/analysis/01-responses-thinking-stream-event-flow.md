---
Title: Responses Thinking Stream Event Flow
Ticket: MO-002-FIX-UP-THINKING-MODELS
Status: active
Topics:
    - bug
    - geppetto
    - go
    - inference
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/chat-events.go
      Note: Defines EventThinkingPartial and related metadata.
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Emits reasoning and partial-thinking events from Responses SSE.
    - Path: pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go
      Note: Example handler that renders thinking events.
    - Path: pinocchio/pkg/ui/backend.go
      Note: Default chat UI handler ignores thinking/info events.
    - Path: pinocchio/pkg/ui/runtime/builder.go
      Note: Default handler selection for chat UI.
    - Path: pinocchio/pkg/webchat/forwarder.go
      Note: Web chat event mapping drops thinking events.
ExternalSources: []
Summary: Map the Responses API event flow and why pinocchio chat UI drops thinking streams.
LastUpdated: 2026-01-13T00:00:00Z
WhatFor: Explain why Responses reasoning events confuse pinocchio chat and where the gaps are.
WhenToUse: Use when debugging GPT-5/o-series reasoning summary streams in pinocchio.
---


# Responses Thinking Stream Event Flow

## Context and repro

Command used (tmux):

```
go run ./cmd/pinocchio code professional "hello" \
  --ai-engine gpt-5-mini \
  --chat \
  --ai-api-type openai-responses \
  --log-level DEBUG \
  --log-file /tmp/pinocchio-gpt5-debug.log \
  --with-caller
```

Observed behavior:
- UI responded successfully with a GPT-5 reply.
- Debug log recorded a dense burst of reasoning summary deltas and info events, but the UI rendered only the assistant text stream.
- No errors were written to the log file; the earlier fast-scrolling error in the terminal is likely a UI or stderr message outside the file or a prior request error.

## High-level event path

OpenAI Responses SSE -> geppetto Responses engine -> geppetto events -> Watermill router -> pinocchio UI handler -> Bubble Tea timeline

Key code locations:
- Responses engine event emission: `geppetto/pkg/steps/ai/openai_responses/engine.go`
- Events definitions: `geppetto/pkg/events/chat-events.go`
- pinocchio chat UI forwarder: `pinocchio/pkg/ui/backend.go`
- pinocchio chat session builder: `pinocchio/pkg/ui/runtime/builder.go`
- agent tool loop forwarder (handles thinking events): `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go`
- web chat forwarder (drops thinking events): `pinocchio/pkg/webchat/forwarder.go`

## Detailed flow (Responses -> UI)

1) Responses SSE streaming (engine)
- The Responses engine parses server-sent events from OpenAI.
- It emits a mix of event types:
  - `EventInfo` with messages like `thinking-started`, `reasoning-summary-started`, `reasoning-summary-ended`, `output-started`, `output-ended`.
  - `EventThinkingPartial` for reasoning summary deltas (event type `partial-thinking`).
  - `EventPartialCompletionStart`, `EventPartialCompletion`, `EventFinal` for assistant output text.

2) Router and UI handler (pinocchio chat)
- `runChat` wires the router and attaches a UI sink. The engine publishes to topic `ui`.
- `runtime.NewChatBuilder().BuildProgram()` defaults to `ui.StepChatForwardFunc` as the handler.
- `StepChatForwardFunc` processes:
  - `EventPartialCompletionStart`, `EventPartialCompletion`, `EventFinal`, `EventInterrupt`, `EventError`.
  - It ignores `EventInfo` and `EventThinkingPartial` entirely.

3) Resulting UX impact
- The UI only renders the assistant message stream and ignores the thinking/reasoning summary stream.
- The engine emits a high volume of `EventThinkingPartial` deltas before the assistant output starts; these are logged but not rendered.
- If terminal UI prints a transient error, it is not coming from `StepChatForwardFunc` (no error is returned for unknown events). It is more likely an unrelated stderr message or earlier request error.

## Log evidence (debug file)

Extracted from `/tmp/pinocchio-gpt5-debug.log`:
- Multiple `EventInfo` and `EventThinkingPartial` entries are received by the UI handler.
- Example sequence (trimmed):
  - `EventInfo` message `thinking-started`
  - `EventInfo` message `reasoning-summary-started`
  - many `EventThinkingPartial` events (reasoning summary deltas)
  - later `EventPartialCompletionStart` and `EventPartialCompletion` for assistant output

The log shows the handler receiving the thinking events, but there are no UI entity updates for them because the forwarder ignores them.

## Why the Responses API confuses the pinocchio chat frontend

Primary causes:
- The Responses engine emits reasoning summary events that the pinocchio chat UI does not render.
- The default chat UI event forwarder only knows about partial completion events and treats everything else as a no-op.
- The reasoning summary stream can be large and early in the timeline, so the UI appears idle while the log scrolls rapidly.

Contrast: agent tool loop UI
- `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go` explicitly renders `EventInfo` (thinking-started/ended) and `EventThinkingPartial` into a separate timeline entity with role `thinking`.
- This is the UX behavior the core pinocchio chat path lacks.

Web chat path
- `pinocchio/pkg/webchat/forwarder.go` maps only LLM output and tool calls to SEM events and drops `EventInfo`/`EventThinkingPartial` entirely.

## Hypotheses about the fast-scrolling error

No error lines appear in the debug log, so the likely explanations are:
- The terminal UI (stderr) is printing a brief error unrelated to the `--log-file` output.
- A previous run hit the known request errors (max tokens / temperature), and the user is still seeing that transient output.
- The huge volume of `partial-thinking` log lines (debug level) makes it appear as an error scroll, even though it is not.

## What to change (if we choose to fix)

Potential improvements (not implemented yet):
- Extend `pinocchio/pkg/ui/backend.go` to handle `EventInfo` and `EventThinkingPartial` similarly to the agent tool loop backend.
- Decide on a UX: render a "thinking" timeline entity or surface summary text inline with assistant output.
- For web chat, add semantic mappings for `partial-thinking` and `info` events (even if they are hidden by default).

## Open questions

- Should reasoning summary be shown in the UI by default, or only when a verbose flag is enabled?
- Is the user-visible error actually coming from a different code path (stderr from the engine) that is not captured by `--log-file`?
- Do we want to suppress logging of every thinking delta at DEBUG to reduce noise?
