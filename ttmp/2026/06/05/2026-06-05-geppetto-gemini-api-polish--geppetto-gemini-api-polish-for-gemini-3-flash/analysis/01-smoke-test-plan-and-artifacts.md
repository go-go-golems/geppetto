---
Title: Smoke Test Plan and Artifacts
Ticket: 2026-06-05-geppetto-gemini-api-polish
Status: active
Topics:
    - geppetto
    - providers
    - reasoning
    - streaming
    - tools
DocType: analysis
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: pkg/steps/ai/gemini/engine_gemini.go
      Note: |-
        Direct provider smoke target.
        Direct Geppetto smoke target
    - Path: pkg/steps/ai/gemini/stream_reducer.go
      Note: |-
        Event behavior to validate with direct smokes.
        Event behavior to validate in smoke artifacts
    - Path: ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/01-gemini-sdk-capability-probe.sh
      Note: First completed smoke-plan artifact script
    - Path: ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/artifacts/sdk-capability-probe.json
      Note: First completed smoke-plan artifact output
ExternalSources:
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/01-gemini-3-developer-guide.md
    - ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/sources/08-google-genai-go-pkg.md
Summary: Planned direct Geppetto-first smoke matrix for Gemini 3 Flash / Gemini 3 API polish work.
LastUpdated: 2026-06-05T16:35:00-04:00
WhatFor: Use to run and archive Gemini provider smoke tests before routing through llm-proxy.
WhenToUse: Read before creating smoke scripts, running live Gemini calls, or evaluating smoke artifacts.
---



# Smoke Test Plan and Artifacts

## Goal

Validate Gemini provider behavior through Geppetto itself before using `llm-proxy`. The direct Geppetto smoke path should reveal provider issues in request construction, streaming reduction, tool-call continuation, thinking configuration, thought signatures, usage, finish reasons, and `InferenceResult` metadata.

## Smoke Order

1. Fixture tests in `pkg/steps/ai/gemini`.
2. SDK capability probe script.
3. Direct Geppetto live smokes.
4. `llm-proxy` smokes only after direct Geppetto results are understood.

## Planned Scripts

| Script | Purpose | Output |
|---|---|---|
| `scripts/01-gemini-sdk-capability-probe.sh` | Compile-time check of old vs new SDK fields. | `scripts/artifacts/sdk-capability-probe.json` |
| `scripts/02-generate-gemini-smoke-profiles.py` | Generate local redacted profile templates for Gemini models. | local profile YAML and redacted artifact copy |
| `scripts/03-gemini-geppetto-smoke/main.go` | Run direct Geppetto engine smokes. | events NDJSON, turn YAML, summary JSON |
| `scripts/04-gemini-llm-proxy-smoke.py` | Run proxy smokes after direct smokes pass. | OpenAI-compatible request/response artifacts |

## Direct Geppetto Smoke Matrix

| Case | Required before implementation? | Required after implementation? | Expected artifact |
|---|---:|---:|---|
| plain text | yes | yes | assistant text block, usage, completed finish class |
| streaming text events | yes | yes | text segment started/delta/finished events |
| function call | yes | yes | tool call requested, tool block, provider ID if available |
| tool loop | yes | yes | tool result accepted and final text generated |
| visible thinking | no if old SDK cannot request it | yes | reasoning events and reasoning block |
| thinking + tool loop | no if old SDK cannot preserve signatures | yes | no 400, signature round-trip evidence |
| max token finish | yes | yes | max-token/truncated finish classification |
| malformed function call / provider error | optional | yes | structured error artifact |

## Artifact Requirements

Every smoke run must write files under `scripts/artifacts/`. Do not leave only `/tmp` evidence.

Recommended per-run files:

```text
scripts/artifacts/<case>-summary.json
scripts/artifacts/<case>-events.ndjson
scripts/artifacts/<case>-turn.json
scripts/artifacts/<case>-inference-result.json
scripts/artifacts/<case>-raw-provider-chunks.ndjson
scripts/artifacts/<case>-profile.redacted.yaml
```

Every artifact must avoid API keys and bearer tokens. Profile artifacts should preserve model name, API type, base URL host, and non-secret settings.

## Success Criteria

Direct Geppetto smoke is considered passing when:

- `RunInference` returns no error for the positive cases,
- canonical events validate against expected event types,
- final `turns.Turn` contains the expected block sequence,
- `InferenceResult` is present in turn metadata,
- usage is present when provider returns usage,
- tool-call IDs are stable through tool result replay,
- thought signatures are preserved for Gemini 3 thinking/tool loops after implementation,
- no thought text leaks into assistant-visible answer text.

## Current Artifacts

| Artifact | Status | Notes |
|---|---|---|
| `scripts/01-gemini-sdk-capability-probe.sh` | complete | Runs in an isolated temporary Go module and does not edit Geppetto's module files. |
| `scripts/artifacts/sdk-capability-probe.json` | complete | Shows legacy SDK supports baseline calls but lacks Gemini 3 fields; modern SDK supports them. |
| `scripts/03-gemini-geppetto-smoke/main.go` | created | Direct Geppetto smoke runner for `plain-text`, `tool-call`, and `tool-loop`; resolves credentials and model settings from Geppetto/Pinocchio profile registries. |
| `scripts/artifacts/plain-text-gemini-2.5-flash-summary.json` | passed | Direct profile-backed Geppetto smoke produced text, usage, provider-call events, and completed finish class. |
| `scripts/artifacts/tool-call-gemini-2.5-flash-summary.json` | passed | Direct profile-backed Geppetto smoke produced a tool-call block and `tool_calls_pending` finish class. |
| `scripts/artifacts/tool-loop-gemini-2.5-flash-summary.json` | passed | Direct profile-backed Geppetto smoke accepted a client-side tool result and produced final assistant text. |
| `scripts/artifacts/plain-text-gemini-3-pro-summary.json` | provider/model access failure | Legacy SDK/API path returned 404: `models/gemini-3-pro is not found for API version v1beta, or is not supported for generateContent`. |
| `scripts/artifacts/tool-call-gemini-3-pro-summary.json` | provider/model access failure | Same Gemini 3 profile/API-version failure as plain text. |
| `scripts/artifacts/tool-loop-gemini-3-pro-summary.json` | provider/model access failure | Same Gemini 3 profile/API-version failure as plain text. |
| `pkg/steps/ai/gemini/modern_adapter_test.go` | complete | Fixture tests cover thought parts/signatures, provider function-call IDs, thoughts token usage metadata, and tool-call/tool-result replay shape with `google.golang.org/genai`. |
| `pkg/steps/ai/gemini/modern_engine.go` | complete | Live Gemini engine path now uses `google.golang.org/genai`, profile-backed API key settings, modern response IDs, thoughts usage metadata, and provider-native tool-call IDs. |
| `scripts/artifacts/plain-text-gemini-2.5-flash-summary.json` | passed after modern migration | Direct profile-backed smoke passes through the modern SDK path; response ID and thoughts token metadata are present. |
| `scripts/artifacts/tool-call-gemini-2.5-flash-summary.json` | passed after modern migration | Tool call passes through modern SDK path. |
| `scripts/artifacts/tool-loop-gemini-2.5-flash-summary.json` | passed after modern migration | Client-driven tool result replay passes through modern SDK path. |
| `scripts/artifacts/plain-text-gemini-2.5-flash-gemini-3-flash-preview-summary.json` | passed | `gemini-3-flash-preview` model override produced direct text through the modern SDK path. |
| `scripts/artifacts/tool-call-gemini-2.5-flash-gemini-3-flash-preview-summary.json` | passed | `gemini-3-flash-preview` emitted a provider-native short function-call ID and thoughts token usage. |
| `scripts/artifacts/tool-loop-gemini-2.5-flash-gemini-3-flash-preview-summary.json` | passed | `gemini-3-flash-preview` accepted tool-result replay and completed. |
| `scripts/artifacts/visible-thinking-gemini-2.5-flash-gemini-3-flash-preview-summary.json` | passed/no thought parts | Thinking was requested and usage included totals, but the provider did not emit visible thought parts in this smoke. |
| `scripts/04-gemini-llm-proxy-smoke.py` | complete | Starts `llm-proxy` on an ephemeral local port with the local profile registry and runs Gemini-backed OpenAI-compatible smoke cases. |
| `scripts/artifacts/llm-proxy-gemini-smoke-summary.json` | passed | `/v1/models`, `/v1/completions`, completions SSE, `/v1/chat/completions`, chat SSE, tool-call, and client-driven tool-loop all passed with profile `gemini-3-flash-preview`. |

SDK probe summary:

```json
{
  "legacy": "github.com/google/generative-ai-go v0.20.1",
  "modern": "google.golang.org/genai v1.58.0",
  "old_baseline": true,
  "old_modern_fields": false,
  "new_modern_fields": true
}
```

## Running the Direct Geppetto Smoke Runner

The direct runner intentionally uses Geppetto's engine factory instead of `llm-proxy`, because provider behavior must be isolated first. It resolves the selected profile from the local profile registry and does not read provider keys from raw environment variables.

```bash
cd /home/manuel/workspaces/2026-06-04/llm-proxy/geppetto
go run ./ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/03-gemini-geppetto-smoke --case plain-text --profile gemini-2.5-flash
go run ./ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/03-gemini-geppetto-smoke --case tool-call --profile gemini-2.5-flash
go run ./ttmp/2026/06/05/2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash/scripts/03-gemini-geppetto-smoke --case tool-loop --profile gemini-2.5-flash
```

By default, the runner uses the first existing local profile registry from `~/.config/pinocchio/profiles.yaml` or `~/.pinocchio/config/profiles.yaml`. Override with `--profile-registries /path/to/profiles.yaml` when needed.

## Known Initial Constraints

- Current Geppetto uses `github.com/google/generative-ai-go/genai` v0.20.1.
- Current implementation likely cannot pass Gemini 3 thought-signature smokes because the old SDK does not expose the required public fields.
- Live model access may depend on account availability and exact model names. Artifact summaries must distinguish `model_not_found` / access errors from Geppetto mapping failures.
