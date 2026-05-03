---
Title: Curl endpoint probe redacted
Ticket: WAFER-AI-404
Status: active
Topics:
    - llm
    - openai
    - pinocchio
    - wafer-ai
DocType: source
Intent: short-term
Owners: []
RelatedFiles:
    - /home/manuel/.config/pinocchio/profiles.yaml
ExternalSources: []
Summary: "Redacted curl evidence showing Wafer accepts /v1/chat/completions but returns 404 for the double-appended path."
LastUpdated: 2026-05-03T11:46:00-04:00
WhatFor: "Provider endpoint evidence for WAFER-AI-404."
WhenToUse: "Use when validating the Wafer base URL versus operation endpoint distinction."
---

# Curl endpoint probe (redacted)

The Authorization header used the locally configured Wafer API key but is not stored here.

## Correct endpoint

Command shape:

```bash
curl -sS -w '\nHTTP_STATUS:%{http_code}\n' -X POST \
  https://pass.wafer.ai/v1/chat/completions \
  -H 'Authorization: Bearer ***' \
  -H 'Content-Type: application/json' \
  -d '{"model":"DeepSeek-V4-Pro","messages":[{"role":"user","content":"Hello!"}],"max_tokens":8,"stream":true}'
```

Observed result: HTTP 200 and SSE frames beginning with:

```text
data: {"id":"...","object":"chat.completion.chunk","created":1777822856,"model":"DeepSeek-V4-Pro","choices":[{"index":0,"delta":{"role":"assistant","content":"","reasoning_content":null,"tool_calls":null},"logprobs":null,"finish_reason":null,"matched_stop":null}],"usage":null}

data: {"id":"...","object":"chat.completion.chunk","created":1777822856,"model":"DeepSeek-V4-Pro","choices":[{"index":0,"delta":{"role":null,"content":"Hi","reasoning_content":null,"tool_calls":null},"logprobs":null,"finish_reason":null,"matched_stop":null}],"usage":null}
```

## Double-appended endpoint

Command shape:

```bash
curl -sS -w '\nHTTP_STATUS:%{http_code}\n' -X POST \
  https://pass.wafer.ai/v1/chat/completions/chat/completions \
  -H 'Authorization: Bearer ***' \
  -H 'Content-Type: application/json' \
  -d '{"model":"DeepSeek-V4-Pro","messages":[{"role":"user","content":"Hello!"}],"max_tokens":8,"stream":true}'
```

Observed result:

```text
HTTP_STATUS:404
```
