---
Title: Profiles provider base refactor redacted summary
Ticket: WAFER-AI-404
Status: active
Topics:
    - llm
    - pinocchio
    - profiles
DocType: source
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/.config/pinocchio/profiles.yaml
ExternalSources: []
Summary: "Redacted summary of the local profiles.yaml refactor that moved repeated provider credentials/base URLs/API types into base profiles."
LastUpdated: 2026-05-03T12:00:00-04:00
WhatFor: "Audit trail for the local profile stack refactor without storing API keys."
WhenToUse: "Use when reviewing or reversing the provider base profile layout."
---

# Profiles provider base refactor redacted summary

Sensitive source file:

```text
/home/manuel/.config/pinocchio/profiles.yaml
```

Backup created before the refactor:

```text
/home/manuel/.config/pinocchio/profiles.yaml.bak-20260503-115557-provider-bases
```

## Base profiles created or normalized

- `wafer-base`
- `together-base`
- `cerebras-base`
- `openai-responses-base`
- `claude-base`
- `gemini-base`
- `ollama-openai-base`
- `groq-base`
- `litellm-base`
- `mistral-base`
- `anyscale-base`
- `openrouter-base`
- `z-ai-base`

Each base profile owns provider-specific settings such as:

- `chat.api_type`
- `api.api_keys.*` where needed
- `api.base_urls.*` where needed
- provider-level `client` settings where needed (`z-ai-base` owns its timeout)

## Leaf profile pattern

Leaf profiles now stack on a provider base and keep only model-specific settings.

Example shape:

```yaml
openai-responses-base:
  inference_settings:
    chat:
      api_type: openai-responses
    api:
      api_keys:
        openai-api-key: '***'

gpt-5-mini:
  stack:
    - profile_slug: openai-responses-base
  inference_settings:
    chat:
      engine: gpt-5-mini
```

Provider/model examples:

```yaml
groq-base:
  inference_settings:
    chat:
      api_type: openai
    api:
      api_keys:
        openai-api-key: '***'
      base_urls:
        openai-base-url: https://api.groq.com/openai/v1

kimi:
  stack:
    - profile_slug: groq-base
  inference_settings:
    chat:
      engine: moonshotai/kimi-k2-instruct
```

## Validation performed

All non-base stacked profiles were checked with:

```bash
PINOCCHIO_PROFILE=<profile> pinocchio code professional --print-inference-settings --non-interactive hello
```

Every checked profile resolved successfully.

A script also verified there are no leaf-profile repetitions of:

- `inference_settings.api.api_keys`
- `inference_settings.api.base_urls`
- `inference_settings.chat.api_type`
- `inference_settings.client`

Result:

```text
violations []
```
