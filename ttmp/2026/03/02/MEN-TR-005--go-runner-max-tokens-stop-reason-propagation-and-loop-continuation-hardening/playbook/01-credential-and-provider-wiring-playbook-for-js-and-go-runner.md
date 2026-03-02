---
Title: Credential and provider wiring playbook for JS and Go runner
Ticket: MEN-TR-005
Status: active
Topics:
    - temporal-relationships
    - geppetto
    - stop-policy
    - claude
    - extraction
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/extractor/gorunner/run.go
      Note: Go runner engine resolution and StepSettings wiring path
    - Path: scripts/01-repro-max-tokens-stop-reason.sh
      Note: End-to-end reproducible extraction command sequence
ExternalSources: []
Summary: Operational runbook for deterministic credential and provider wiring across Geppetto JS bindings and the temporal-relationships Go extraction runner.
LastUpdated: 2026-03-02T15:40:00-05:00
WhatFor: Provide one canonical process for loading provider settings and credentials without environment fallback.
WhenToUse: Use when onboarding engineers, wiring new profiles/providers, or debugging extraction startup failures.
---

# Credential and provider wiring playbook for JS and Go runner

## Purpose

Establish one deterministic, repeatable credential wiring model for:

- Geppetto JavaScript engine construction (`fromProfile` and `fromConfig`), and
- Temporal Relationships Go runner extraction (`go extract`).

This playbook explicitly avoids runtime `os.Getenv(...)` key fallback in engine construction paths.

## Environment Assumptions

1. You are on the workspace root that contains both repos:
   - `geppetto/`
   - `temporal-relationships/`
2. You have a profile registry source available (YAML or SQLite), for example:
   - `~/.config/pinocchio/profiles.yaml`
3. Your selected profile includes provider keys in schema-backed step settings sections, for example:

```yaml
runtime:
  step_settings_patch:
    ai-chat:
      ai-api-type: claude
      ai-engine: claude-haiku-4-5
    claude-chat:
      claude-api-key: <redacted>
```

OpenAI example:

```yaml
runtime:
  step_settings_patch:
    ai-chat:
      ai-api-type: openai
      ai-engine: gpt-4o-mini
    openai-chat:
      openai-api-key: <redacted>
      openai-base-url: https://api.openai.com/v1
```

## Credential Contract (Canonical)

### Rule 1

Provider credentials are runtime inputs, not ambient process state.

### Rule 2

Engine construction should succeed only when keys are present in explicit inputs:

- profile-derived `EffectiveStepSettings`, or
- explicit API options (`fromConfig({apiKey: ...})`, go config `engine.apiKey`).

### Rule 3

Provider defaults (base URLs) may be applied implicitly; provider keys may not.

## Wiring Path A: JS + `fromProfile` (Preferred)

### Host wiring (Go side)

```go
reg := require.NewRegistry()
mod.Register(reg, geppettomodule.Options{
    ProfileRegistry: profileRegistry, // required for fromProfile
})
```

### JS usage

```javascript
const gp = require("geppetto");

const eng = gp.engines.fromProfile("mento-haiku", {
  runtimeKey: "default"
});

const s = gp.createSession({ engine: eng });
```

### Expected behavior

- profile resolves through registry stack,
- key comes from profile runtime patch,
- missing key fails explicitly with provider-key validation error.

## Wiring Path B: JS + `fromConfig` (Explicit Ad-hoc)

Use this when profile registry is unavailable or for intentionally local/manual scripts.

```javascript
const gp = require("geppetto");

const eng = gp.engines.fromConfig({
  apiType: "openai",
  model: "gpt-4o-mini",
  apiKey: "<explicit-key>",
  baseURL: "https://api.openai.com/v1",
  maxTokens: 512
});
```

Notes:

- `apiKey` is required for provider-backed execution.
- Do not rely on `OPENAI_API_KEY`/`ANTHROPIC_API_KEY` being read by runtime engine helpers.

## Wiring Path C: Go Runner + Profile Registries (Preferred)

### CLI command pattern

```bash
cd temporal-relationships

go run ./cmd/temporal-relationships \
  --profile-registries "yaml://$HOME/.config/pinocchio/profiles.yaml" \
  --db-path /tmp/temporal.db \
  go extract \
  --config ./config/structured-event-extraction.example.yaml \
  --input-file ./anonymized/a2be5ded.txt \
  --profile mento-haiku \
  --timeline-printer=false \
  --print-result=true \
  --result-format=json
```

### Config expectations

`engine.mode` must be non-`echo` (for provider execution). Profile resolution path is selected by supplying registry sources.

```yaml
engine:
  mode: profile

profiles:
  registrySources: []
```

(The CLI `--profile-registries` value is merged with config sources.)

## Wiring Path D: Go Runner + Explicit Config (Fallback Mode)

If no profile registries are provided, configure provider explicitly:

```yaml
engine:
  mode: config
  apiType: claude
  model: claude-haiku-4-5
  apiKey: <explicit-key>
  baseURL: https://api.anthropic.com
```

Run:

```bash
go run ./cmd/temporal-relationships \
  --db-path /tmp/temporal.db \
  go extract \
  --config /tmp/manual-config.yaml \
  --input-file ./anonymized/a2be5ded.txt \
  --timeline-printer=false
```

## Minimal Validation Procedure

### 1. Verify profile source selected

- Confirm `--profile-registries` points at expected file/DSN.
- Confirm requested `--profile` exists in registry.

### 2. Run extraction

Use the reproducible ticket script:

```bash
bash ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/scripts/01-repro-max-tokens-stop-reason.sh
```

### 3. Confirm engine actually ran

Check script output includes:

- `run_inference_starts > 0`
- non-empty final result object

### 4. Confirm no hidden env dependency

Temporarily clear provider env var in shell and re-run. Behavior should remain identical when profile/config keys are correctly wired.

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---|---|---|
| `missing API key <provider-api-key>` | key missing in profile patch or explicit config | add key to `openai-chat`/`claude-chat` (or explicit `apiKey`) |
| profile resolves but provider call fails auth | wrong key value or wrong profile selected | verify profile slug and key source |
| JS `fromProfile` throws registry error | host did not pass `Options.ProfileRegistry` | wire profile registry in module options |
| config appears valid but still missing key | using legacy patch section (`api`) instead of schema section | migrate to `openai-chat` or `claude-chat` sections |
| works on one machine only | machine had env vars previously masking missing profile keys | remove env assumptions and fix profile/config wiring |

## Security Handling

1. Never commit literal provider keys into repo files.
2. Keep keys in user-local secure registry/config paths.
3. In debug output, print key source and key-name fields, never key values.
4. Treat profile registry backups as sensitive artifacts.

## Exit Criteria

You are done when all criteria hold:

1. JS `fromProfile` and/or `fromConfig` path starts without env fallback assumptions.
2. Go runner extraction starts successfully using profile/config-provided keys only.
3. Repro script runs end-to-end on longest anonymized transcript.
4. Documentation and ticket diary/changelog capture the final contract.

## Notes

- This playbook intentionally separates **credential loading** from **provider defaults**.
- `baseURL` defaults are acceptable convenience defaults; credentials are not.
- If you need env-based bootstrapping for local ergonomics, do it as an explicit pre-processing step that writes into profile/config inputs before runtime resolution.
