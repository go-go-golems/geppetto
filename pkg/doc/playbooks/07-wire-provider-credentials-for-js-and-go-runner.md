---
Title: "Wire provider credentials for JS and go runner"
Slug: wire-provider-credentials-js-go-runner
Short: Deterministic credential/provider wiring for `gp.engines.*` and temporal-relationships go extraction runner.
Topics:
  - profiles
  - configuration
  - javascript
  - extraction
Commands:
  - geppetto
  - temporal-relationships
Flags:
  - profile
  - profile-registries
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Playbook
---

# Wire provider credentials for JS and go runner

## Goal

Configure provider credentials and provider settings explicitly through profile/runtime settings (or explicit config inputs), with no runtime environment fallback in engine-construction paths.

## Before you start

- Have a registry source (YAML/SQLite), for example `~/.config/pinocchio/profiles.yaml`.
- Ensure your profile includes keys in provider-specific sections:
  - `openai-chat.openai-api-key`
  - `claude-chat.claude-api-key`
- Know your selected profile slug (`--profile` for go runner, `fromProfile("...")` for JS).

## Step 1: Profile patch shape (required)

```yaml
runtime:
  step_settings_patch:
    ai-chat:
      ai-api-type: claude
      ai-engine: claude-haiku-4-5
    claude-chat:
      claude-api-key: <redacted>
      claude-base-url: https://api.anthropic.com
```

OpenAI variant:

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

## Step 2: JS wiring (`fromProfile` preferred)

Host module setup:

```go
reg := require.NewRegistry()
geppettomodule.Register(reg, geppettomodule.Options{
  ProfileRegistry: profileRegistry,
})
```

Script:

```javascript
const gp = require("geppetto");
const eng = gp.engines.fromProfile("mento-haiku");
const s = gp.createSession({ engine: eng });
```

If profile keys are missing, engine creation should fail explicitly.

## Step 3: JS ad-hoc (`fromConfig`)

```javascript
const eng = gp.engines.fromConfig({
  apiType: "openai",
  model: "gpt-4o-mini",
  apiKey: "<explicit-key>",
  baseURL: "https://api.openai.com/v1"
});
```

`fromConfig` expects explicit `apiKey` for provider-backed execution.

## Step 4: Go runner wiring (profile path preferred)

```bash
cd temporal-relationships

go run ./cmd/temporal-relationships \
  --profile-registries "yaml://$HOME/.config/pinocchio/profiles.yaml" \
  --db-path /tmp/temporal.db \
  go extract \
  --config ./config/structured-event-extraction.example.yaml \
  --input-file ./anonymized/a2be5ded.txt \
  --profile mento-haiku
```

## Step 5: Go runner explicit config fallback path

```yaml
engine:
  mode: config
  apiType: openai
  model: gpt-4o-mini
  apiKey: <explicit-key>
```

Use this only when profile registry is intentionally not used.

## Step 6: Validate determinism

- Run once with your current shell.
- Unset provider env vars in shell.
- Run again.
- Results should match when profile/config keys are complete.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `missing API key ...` | key not set in profile patch or explicit config | add provider key to `openai-chat`/`claude-chat` or pass explicit `apiKey` |
| `engines.fromProfile requires a configured profile registry` | host omitted `Options.ProfileRegistry` | wire registry in module options |
| works locally, fails in CI | local env vars had masked missing profile/config key | remove env dependency and complete profile/config key wiring |
| profile patch applied but key still absent | wrong section slug (legacy `api` section) | use schema-backed section slugs (`openai-chat`, `claude-chat`) |

## See Also

- [Geppetto JavaScript API Reference](../topics/13-js-api-reference.md)
- [Geppetto JavaScript API User Guide](../topics/14-js-api-user-guide.md)
- [Profile Registry in Geppetto](../topics/01-profiles.md)
