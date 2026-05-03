---
Title: Live Wafer DeepSeek V4 Pro high thinking validation redacted
Ticket: OAI-CHAT-THINKING
Status: active
Topics:
    - llm
    - openai
    - inference
DocType: source
Intent: short-term
Owners: []
RelatedFiles:
    - /home/manuel/.config/pinocchio/profiles.yaml
ExternalSources: []
Summary: "Redacted validation showing wafer-deepseek-v4-pro resolves high-effort thinking settings and streams reasoning_content."
LastUpdated: 2026-05-03T12:23:00-04:00
WhatFor: "Runtime validation evidence for the default Wafer DeepSeek V4 Pro thinking profile."
WhenToUse: "Use when reviewing whether wafer-deepseek-v4-pro itself, not only fast/max variants, works after implementation."
---

# Live Wafer DeepSeek V4 Pro high thinking validation redacted

Validation used the local workspace build via:

```bash
cd pinocchio
PINOCCHIO_PROFILE=wafer-deepseek-v4-pro go run ./cmd/pinocchio code professional --print-inference-settings --non-interactive hello
PINOCCHIO_PROFILE=wafer-deepseek-v4-pro go run ./cmd/pinocchio --log-level debug --with-caller code professional --non-interactive "Say hi"
```

The Authorization header/key remained in local config and is not stored here.

## Resolved settings

Important settings excerpt:

```yaml
api:
  base_urls:
    openai-base-url: https://pass.wafer.ai/v1
chat:
  api_type: openai
  engine: DeepSeek-V4-Pro
openai:
  chat_reasoning_effort: high
  thinking_type: enabled
```

## Live run

Important log excerpt:

```text
Making request to openai from turn blocks chat_reasoning_effort=high ... model=DeepSeek-V4-Pro ... thinking_type=enabled

--- Thinking started ---
We are: "You are an experienced technology professional and technical leader in software..."
The user said: "Say hi"
I need to respond concisely: a simple "Hi." is appropriate.
Hi.
OpenAI stream completed chunks_received=37

--- Thinking ended ---
OpenAI RunInference completed (streaming)
```

Observation: `wafer-deepseek-v4-pro` itself now uses enabled/high thinking settings, Wafer streams reasoning text, and Geppetto renders thinking start/end events plus final answer text.
