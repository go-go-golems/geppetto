---
Title: Wafer DeepSeek V4 thinking parameter probe redacted
Ticket: OAI-CHAT-THINKING
Status: active
Topics:
    - llm
    - openai
    - inference
DocType: source
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Redacted live Wafer probes showing DeepSeek-V4-Pro accepts OpenAI Chat Completions thinking.type and reasoning_effort fields."
LastUpdated: 2026-05-03T12:05:00-04:00
WhatFor: "Provider behavior evidence for implementing chat-completions thinking controls."
WhenToUse: "Use when validating the request JSON contract against Wafer/DeepSeek V4."
---

# Wafer DeepSeek V4 thinking parameter probe redacted

Authorization used the locally configured Wafer API key. The key is not stored here.

## Disabled thinking

Request shape:

```json
{
  "model": "DeepSeek-V4-Pro",
  "messages": [{"role": "user", "content": "Say hi"}],
  "max_tokens": 16,
  "stream": false,
  "thinking": {"type": "disabled"}
}
```

Observed result:

```text
HTTP_STATUS:200
message.content: "Hi there! How can I help you today?"
message.reasoning_content: null
```

## Enabled high effort thinking

Request shape:

```json
{
  "model": "DeepSeek-V4-Pro",
  "messages": [{"role": "user", "content": "Say hi"}],
  "max_tokens": 16,
  "stream": false,
  "thinking": {"type": "enabled"},
  "reasoning_effort": "high"
}
```

Observed result:

```text
HTTP_STATUS:200
message.content: null
message.reasoning_content: "We need to respond with ..."
finish_reason: "length"
```

The low `max_tokens` intentionally cut off the answer during reasoning, proving the request entered thinking mode.

## Enabled max effort thinking

Request shape:

```json
{
  "model": "DeepSeek-V4-Pro",
  "messages": [{"role": "user", "content": "Say hi"}],
  "max_tokens": 16,
  "stream": false,
  "thinking": {"type": "enabled"},
  "reasoning_effort": "max"
}
```

Observed result:

```text
HTTP_STATUS:200
message.content: null
message.reasoning_content: "We are asked: ..."
finish_reason: "length"
```

This confirms Wafer's OpenAI-compatible endpoint accepts both `thinking.type` and `reasoning_effort` for `DeepSeek-V4-Pro`.
