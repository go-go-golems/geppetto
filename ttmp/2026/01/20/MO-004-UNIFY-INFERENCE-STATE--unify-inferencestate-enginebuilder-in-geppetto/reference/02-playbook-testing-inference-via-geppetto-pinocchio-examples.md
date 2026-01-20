---
Title: 'Playbook: Testing Inference via geppetto/pinocchio examples'
Ticket: MO-004-UNIFY-INFERENCE-STATE
Status: active
Topics:
    - inference
    - architecture
    - webchat
    - prompts
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/examples/generic-tool-calling/main.go
      Note: Tool loop smoke test
    - Path: geppetto/cmd/examples/openai-tools/main.go
      Note: Responses thinking smoke test
    - Path: pinocchio/cmd/agents/simple-chat-agent/main.go
      Note: Agent TUI smoke test
    - Path: pinocchio/cmd/examples/simple-redis-streaming-inference/main.go
      Note: Redis transport smoke test
    - Path: pinocchio/cmd/web-chat/README.md
      Note: Webchat usage notes
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-20T15:40:45.898346639-05:00
WhatFor: ""
WhenToUse: ""
---


# Playbook: Testing Inference via geppetto/pinocchio examples

## Goal

Provide a repeatable set of commands to validate:

- provider inference works (OpenAI Chat Completions and OpenAI Responses)
- streaming event routers work
- “thinking” events flow (Responses)
- tool calling loop works
- pinocchio TUI/webchat behave in real world runs (not just `go test`)

## Context

### Pre-reqs

- You have provider credentials in your environment (do **not** paste them in logs/docs):
  - `OPENAI_API_KEY` (required for OpenAI Chat Completions + OpenAI Responses)
  - optionally: `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`
- You’re in the workspace root:
  - `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api`
- For TUI tests, use `tmux` so the process is guaranteed to terminate and logs can be captured.
- For Redis streaming tests, Redis must be reachable if you enable `--redis-enabled`.

### Terminology

- **OpenAI Chat Completions**: `--ai-api-type openai`
- **OpenAI Responses**: `--ai-api-type openai-responses` (stricter validation, supports “thinking” items)
- **Tool loop**: iterative inference that detects tool calls, executes tools, appends tool results, and re-prompts (bounded by max iterations).

## Quick Reference

### A. geppetto examples (workspace module: `geppetto/`)

#### A1) Basic single-pass inference (non-streaming)

```bash
cd geppetto
go run ./cmd/examples/simple-inference --pinocchio-profile 4o-mini --prompt "Say hello."
```

What it tests:
- EngineBuilder creation path
- `InferenceState` + `core.Session` single-pass run

Expected:
- Prints a “Final Turn” with an assistant response.

#### A2) Streaming inference (Watermill event router)

```bash
cd geppetto
go run ./cmd/examples/simple-streaming-inference \
  --pinocchio-profile 4o-mini \
  --prompt "Write one sentence about penguins." \
  --output-format text \
  --verbose \
  --log-level info
```

What it tests:
- EventRouter lifecycle (publisher/subscriber)
- streaming events from provider → sink → router handlers

Expected:
- streaming output (partial/final) + final printed turn.

#### A3) OpenAI Responses “thinking” smoke test (no tools)

```bash
cd geppetto
go run ./cmd/examples/openai-tools test-openai-tools \
  --ai-api-type openai-responses \
  --ai-engine gpt-5-mini \
  --mode thinking \
  --prompt "What is 23*17 + 55?"
```

What it tests:
- Responses streaming (thinking deltas + final)
- provider validation and parameter compatibility

Expected:
- “Thinking started/ended” + final output.

#### A4) Tool calling loop (provider-agnostic tool helpers)

```bash
cd geppetto
go run ./cmd/examples/generic-tool-calling \
  --pinocchio-profile 4o-mini \
  --prompt "What's the weather in Paris and what is 2+2?" \
  --tools-enabled \
  --max-iterations 3 \
  --log-level info
```

What it tests:
- `core.Session` tool-loop path (`Registry` + `ToolConfig`)
- tool call / tool result blocks appear in the final turn

Expected:
- Final output includes Tool Call / Tool Result blocks.

#### A5) Middleware demo (non-tool middleware + optional tool config on Turn)

```bash
cd geppetto
go run ./cmd/examples/middleware-inference \
  --pinocchio-profile 4o-mini \
  --prompt "write hello in uppercase" \
  --with-uppercase \
  --with-logging
```

What it tests:
- middleware wrapping still works when inference is driven by Session.

#### A6) Claude tools (only if you have Claude credentials configured)

```bash
cd geppetto
go run ./cmd/examples/claude-tools test-claude-tools \
  --ai-api-type claude \
  --ai-engine claude-3-5-sonnet-latest
```

What it tests:
- provider swap + tool middleware wrapper path still works.

#### A7) Citations event stream UI (does not call inference)

```bash
cd geppetto
go run ./cmd/examples/citations-event-stream
```

What it tests:
- a Bubble Tea UI around streamed YAML parsing/extraction.

Note:
- This example is not an inference runner. It won’t validate provider calls.

### B. pinocchio examples (workspace module: `pinocchio/`)

#### B1) pinocchio TUI chat (real world multi-turn)

Use tmux, capture logs, and confirm multiple turns work.

```bash
cd pinocchio
rm -f /tmp/pinocchio-tui.log
tmux kill-session -t pinocchio-tui-smoke 2>/dev/null || true
tmux new-session -d -s pinocchio-tui-smoke \
  "go run ./cmd/pinocchio code professional 'hello' \
    --ai-api-type openai-responses \
    --ai-engine gpt-5-mini \
    --chat \
    --log-level debug \
    --log-file /tmp/pinocchio-tui.log \
    --with-caller"

# Submit + quit (in this UI, Tab submits; Alt-q quits).
sleep 2
tmux send-keys -t pinocchio-tui-smoke 'What is 2+2?' Tab
sleep 10
tmux send-keys -t pinocchio-tui-smoke 'And what is 3+3?' Tab
sleep 10
tmux send-keys -t pinocchio-tui-smoke M-q
sleep 1
tmux kill-session -t pinocchio-tui-smoke 2>/dev/null || true

tail -n 200 /tmp/pinocchio-tui.log
```

What it tests:
- multi-turn state persistence via `InferenceState.Turn`
- Responses provider “thinking” items do not break the second message

#### B2) pinocchio webchat server (browser-driven)

```bash
cd pinocchio
go run ./cmd/web-chat --log-level debug --with-caller
```

Then:
- open the webchat UI in your browser (local address printed by server)
- send 2+ prompts in the same conversation

What it tests:
- webchat conversation state stored in `conv.Inf.Turn`
- Responses “reasoning item must be followed” 400 does not reappear on turn 2

#### B3) pinocchio agent example (TUI) — tool loop + thinking events

```bash
cd pinocchio
rm -f /tmp/simple-chat-agent.log
tmux kill-session -t agent-smoke 2>/dev/null || true
tmux new-session -d -s agent-smoke \
  "go run ./cmd/agents/simple-chat-agent simple-chat-agent \
    --ai-api-type openai-responses \
    --ai-engine gpt-5-mini \
    --ai-max-response-tokens 256 \
    --openai-reasoning-summary auto \
    --log-level debug \
    --log-file /tmp/simple-chat-agent.log \
    --with-caller"

sleep 2
tmux send-keys -t agent-smoke 'hello' Tab
sleep 15
tmux send-keys -t agent-smoke M-q
sleep 1
tmux kill-session -t agent-smoke 2>/dev/null || true

tail -n 250 /tmp/simple-chat-agent.log
```

What it tests:
- tool-loop run path via `core.Session` in a Bubble Tea backend
- “thinking partial” events reaching the UI forwarder

#### B4) pinocchio example: simple chat step (YAML-based command)

```bash
cd pinocchio
go run ./cmd/examples/simple-chat chat --pinocchio-profile default
```

Notes:
- This example exercises the “PinocchioCommand / RunWithOptions” machinery more than it exercises Session directly.

#### B5) pinocchio example: Redis streaming inference

If you have Redis running:

```bash
cd pinocchio
go run ./cmd/examples/simple-redis-streaming-inference \
  --prompt "Stream this response via Redis." \
  --redis-enabled \
  --redis-addr localhost:6379 \
  --redis-group chat-ui \
  --redis-consumer ui-1 \
  --verbose \
  --log-level debug
```

What it tests:
- Watermill Redis Streams transport + router + sink wiring

Note:
- This example currently calls `eng.RunInference` directly and is single-turn; it’s mostly a transport test.

## Usage Examples

### “Fast check” after refactors (recommended)

```bash
cd geppetto
go run ./cmd/examples/openai-tools test-openai-tools --ai-api-type openai-responses --ai-engine gpt-5-mini --mode thinking --prompt "What is 2+2?"
```

```bash
cd pinocchio
tmux new-session -d -s pinocchio-tui-smoke "go run ./cmd/pinocchio code professional 'hello' --ai-api-type openai-responses --ai-engine gpt-5-mini --chat"
```

### “Full sweep” (slower)

Run:

- A1, A2, A3, A4
- B1, B2, B3

and confirm:

- multi-turn does not regress
- tool events still flow
- Responses thinking doesn’t break follow-up prompts

## Related

### Which examples still benefit from InferenceState?

Already benefit (and should keep using it):

- pinocchio TUI chat (`cmd/pinocchio … --chat`)
- pinocchio agent TUI (`cmd/agents/simple-chat-agent …`)
- pinocchio webchat (`cmd/web-chat`)
- geppetto examples that run inference via `core.Session` (most of `geppetto/cmd/examples/*` except the citations UI)

Could benefit (but not required):

- `pinocchio/cmd/examples/simple-redis-streaming-inference` (transport-focused; currently calls `eng.RunInference` directly)
- `pinocchio/cmd/examples/simple-chat` (exercises PinocchioCommand runner; it would benefit indirectly if that runner standardizes on InferenceState internally)

Not an inference runner (InferenceState doesn’t apply):

- `geppetto/cmd/examples/citations-event-stream` (UI for streamed YAML extraction; no provider inference calls)
<!-- Link to related documents or resources -->
