---
Title: Open Responses Example Config and Validation
Ticket: GP-56-OPEN-RESPONSES
Status: active
Topics:
    - geppetto
    - open-responses
    - reasoning
    - streaming
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/steps/ai/settings/flags/chat.yaml
      Note: CLI flag surface for selecting the open-responses provider
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Runtime that now normalizes reasoning aliases and persists richer reasoning blocks
    - Path: geppetto/pkg/steps/ai/openai_responses/helpers.go
      Note: Request builder that now replays reasoning summary payloads
ExternalSources: []
Summary: Example profile/config snippets and validation commands for the implemented open-responses runtime path.
LastUpdated: 2026-03-27T23:35:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Open Responses Example Config and Validation

Use this playbook when wiring a profile or a direct CLI invocation onto Geppetto's implemented Open Responses path.

## Example Profile

```yaml
slug: open-responses-demo
profiles:
  default:
    slug: default
    inference_settings:
      chat:
        api_type: open-responses
        engine: gpt-5-mini
        stream: true
      api:
        api_keys:
          open-responses-api-key: ${OPENAI_API_KEY}
        base_urls:
          open-responses-base-url: https://api.openai.com/v1
```

Notes:

- `api_type: open-responses` is now the preferred canonical provider name.
- `openai-responses` still works as a compatibility alias.
- The runtime also accepts `openai-api-key` / `openai-base-url` fallback aliases.

## Example CLI

```bash
go run ./cmd/examples/streaming-inference/main.go \
  --ai-api-type=open-responses \
  --ai-engine=gpt-5-mini \
  --ai-stream=true
```

## What To Verify

- Reasoning-capable models infer or accept the `open-responses` provider path.
- Streaming reasoning deltas appear even when upstream events use `response.reasoning.delta`.
- Persisted reasoning blocks contain:
  - `text`
  - `summary`
  - `encrypted_content`
- Follow-up requests replay reasoning summary payloads rather than discarding them.

## Focused Validation Commands

```bash
go test ./pkg/inference/engine/factory ./pkg/inference/tokencount/factory ./pkg/steps/ai/openai_responses ./pkg/js/modules/geppetto -count=1
go test ./pkg/steps/ai/openai_responses ./pkg/turns -count=1
docmgr doctor --ticket GP-56-OPEN-RESPONSES --stale-after 30
```

## Current Remaining Gap

The remaining implementation item is fixture and example coverage for a real non-OpenAI Open Responses provider trace. The runtime now handles provider naming, reasoning payload persistence, and reasoning delta alias normalization, but the ticket still needs a provider-specific trace bundle beyond synthetic tests.
