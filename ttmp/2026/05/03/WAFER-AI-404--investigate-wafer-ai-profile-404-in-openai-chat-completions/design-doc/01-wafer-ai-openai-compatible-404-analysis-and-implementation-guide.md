---
Title: Wafer AI OpenAI-compatible 404 analysis and implementation guide
Ticket: WAFER-AI-404
Status: active
Topics:
    - llm
    - openai
    - pinocchio
    - wafer-ai
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../.config/pinocchio/profiles.yaml
      Note: |-
        Local Wafer profiles currently store full chat-completions endpoint in openai-base-url; contains secrets
        Refactored local Wafer profiles to stack on wafer-base with one shared credential and corrected base URL
    - Path: geppetto/pkg/cli/bootstrap/inference_debug.go
      Note: --print-inference-settings source trace proves profile value reaches final settings
    - Path: geppetto/pkg/steps/ai/openai/chat_stream.go
      Note: |-
        OpenAI chat engine constructs final /chat/completions endpoint from provider base URL
        Added HTTP 404 hint/warning when configured OpenAI-compatible base URL looks like an operation endpoint
    - Path: geppetto/pkg/steps/ai/openai/chat_stream_test.go
      Note: Regression tests for 404 hint and suspicious base URL detection
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: Runtime streaming path logs failures but not currently the computed endpoint
    - Path: geppetto/ttmp/2026/05/03/WAFER-AI-404--investigate-wafer-ai-profile-404-in-openai-chat-completions/sources/04-deepseek-thinking-mode-defuddle.md
      Note: Defuddle source for DeepSeek thinking-mode parameters
ExternalSources: []
Summary: Wafer profile 404 is caused by storing the full chat-completions endpoint in openai-base-url while Geppetto appends /chat/completions itself.
LastUpdated: 2026-05-03T11:45:00-04:00
WhatFor: Use when fixing or reviewing Wafer/OpenAI-compatible provider configuration and request diagnostics.
WhenToUse: When Pinocchio/Geppetto OpenAI-compatible profiles show 404 even though curl against the provider endpoint succeeds.
---









# Wafer AI OpenAI-compatible 404 analysis and implementation guide

## Executive summary

The immediate Wafer AI 404 is a profile configuration problem, not a failing API key and not evidence that `PINOCCHIO_PROFILE` is being ignored.

The current `wafer-deepseek-v4-pro` profile resolves `api.base_urls.openai-base-url` to:

```text
https://pass.wafer.ai/v1/chat/completions
```

Geppetto's OpenAI chat engine treats `openai-base-url` as the provider **base** URL and appends `/chat/completions` at runtime. With the current profile, the actual request target becomes:

```text
https://pass.wafer.ai/v1/chat/completions/chat/completions
```

A direct curl probe showed that Wafer accepts streaming chat completions at `/v1/chat/completions`, while the double-appended path returns HTTP 404. Therefore the minimal operator fix is to change all Wafer OpenAI-chat profiles in `~/.config/pinocchio/profiles.yaml` to use:

```yaml
base_urls:
  openai-base-url: https://pass.wafer.ai/v1
```

There are two worthwhile code/documentation follow-ups:

1. Add request-endpoint logging in the OpenAI chat streaming path, with credentials excluded.
2. Add validation or a warning when an `*-base-url` value already ends in a known operation path such as `/chat/completions`.

## Problem statement and scope

The user reported this contrast:

1. Direct curl to Wafer succeeds when POSTing to `https://pass.wafer.ai/v1/chat/completions`.
2. `PINOCCHIO_PROFILE=wafer-deepseek-v4-pro llmunix --log-level trace --with-caller hello` fails with:

```text
OpenAI streaming request failed error="chat completions error: status=404"
Error: inference failed: chat completions error: status=404
```

The requested analysis focused on whether the OpenAI API base URL or another profile setting is wrong, and on the fact that debug logging does not currently show the final HTTP URL used for the request.

This document covers:

- the profile value resolved by Pinocchio's `--print-inference-settings` path;
- the OpenAI chat engine's URL construction behavior;
- live endpoint evidence against Wafer;
- an implementation guide for the immediate config fix and the more durable diagnostics improvements.

This document does not change production code by itself. It is an analysis and implementation guide.

## Current-state analysis

### Profile resolution works and applies the Wafer profile

`pinocchio code professional --print-inference-settings` with `PINOCCHIO_PROFILE=wafer-deepseek-v4-pro` showed the selected profile reaches final inference settings. The saved evidence is:

- `sources/print_inference_settings_wafer_deepseek_redacted.log`

Key resolved settings from that log:

```yaml
settings:
  api:
    api_keys:
      openai-api-key: '***'
    base_urls:
      openai-base-url: https://pass.wafer.ai/v1/chat/completions
  chat:
    api_type: openai
    engine: DeepSeek-V4-Pro
    stream: true
```

The `sources` section also shows the value came from the selected profile:

```yaml
source: profile
profile_slug: wafer-deepseek-v4-pro
value: https://pass.wafer.ai/v1/chat/completions
```

So the problem is not that Pinocchio fails to load the profile. It loads the profile and faithfully applies the base URL stored there.

### The OpenAI chat engine appends `/chat/completions`

In `geppetto/pkg/steps/ai/openai/chat_stream.go`, `resolveChatStreamConfig` reads `openai-base-url` and constructs the final endpoint as follows:

```go
baseURL, ok := apiSettings.BaseUrls[string(apiType)+"-base-url"]
endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
```

Line-anchored evidence:

- `geppetto/pkg/steps/ai/openai/chat_stream.go:60` reads the provider-specific base URL.
- `geppetto/pkg/steps/ai/openai/chat_stream.go:64` appends `/chat/completions`.
- `geppetto/pkg/steps/ai/openai/chat_stream.go:90` creates the HTTP request from that computed endpoint.

That contract means `openai-base-url` must be the base API root, such as:

```text
https://api.openai.com/v1
https://pass.wafer.ai/v1
```

It must not be the operation endpoint:

```text
https://pass.wafer.ai/v1/chat/completions
```

### The existing runtime log does not print the final endpoint

`geppetto/pkg/steps/ai/openai/engine_openai.go` resolves the stream config, builds the request body, then calls `openChatCompletionStream`:

- `engine_openai.go:56` calls `resolveChatStreamConfig`.
- `engine_openai.go:213` logs `OpenAI using streaming mode`.
- `engine_openai.go:214` calls `openChatCompletionStream`.
- `engine_openai.go:216` logs only the error if the HTTP call fails.

The current logs show model, messages, and the generic HTTP status, but not the endpoint. That explains the user's observation that `--print-inference-settings` shows the configured base URL, while trace logging still does not reveal the final request URL.

### Live Wafer endpoint evidence

The saved probe is:

- `sources/01-curl-endpoint-probe-redacted.md`

With the locally configured Wafer API key redacted, curl showed:

| URL | Result |
| --- | --- |
| `https://pass.wafer.ai/v1/chat/completions` | HTTP 200 with SSE chat completion chunks |
| `https://pass.wafer.ai/v1/chat/completions/chat/completions` | HTTP 404 |

This exactly matches the failure mode produced by putting the full endpoint into `openai-base-url` and then letting Geppetto append `/chat/completions` again.

## Gap analysis

### Gap 1: Profile stores endpoint where Geppetto expects base URL

Current Wafer profiles in `~/.config/pinocchio/profiles.yaml` use this shape:

```yaml
profiles:
  wafer-deepseek-v4-pro:
    inference_settings:
      chat:
        api_type: openai
        engine: DeepSeek-V4-Pro
      api:
        base_urls:
          openai-base-url: https://pass.wafer.ai/v1/chat/completions
```

For Geppetto's OpenAI chat engine, this should be:

```yaml
profiles:
  wafer-deepseek-v4-pro:
    inference_settings:
      chat:
        api_type: openai
        engine: DeepSeek-V4-Pro
      api:
        base_urls:
          openai-base-url: https://pass.wafer.ai/v1
```

The same correction applies to the other Wafer OpenAI-compatible chat profiles found locally:

- `wafer-qwen3.5-397b`
- `wafer-deepseek-v4-pro`
- `wafer-glm-5.1`

### Gap 2: CLI flag overrides are not a reliable quick workaround in this profile path

A quick attempt to pass `--openai-base-url https://pass.wafer.ai/v1` did not override the profile value in the final resolved settings. The source trace showed the CLI value first, followed by the profile value, with the profile winning:

```yaml
source: cobra
value: https://pass.wafer.ai/v1
source: profile
value: https://pass.wafer.ai/v1/chat/completions
final value: https://pass.wafer.ai/v1/chat/completions
```

That means the reliable fix is to edit the profile file or adjust the profile resolution precedence intentionally in a separate ticket. For this incident, treat the profile file as the source of truth.

### Gap 3: Debugging visibility is one layer too shallow

`--print-inference-settings` answers "what base URL was configured?" It does not answer "what exact URL did the HTTP client call after engine-specific path construction?"

The engine should log both values at debug level:

- `base_url`: the redacted/non-secret provider root from settings;
- `endpoint`: the computed request URL, with query strings and credentials omitted if ever present;
- `api_type`: `openai`;
- `stream`: `true`.

## Proposed solution

### Immediate operator fix

Edit `~/.config/pinocchio/profiles.yaml` and change each Wafer profile's `openai-base-url` from the operation endpoint to the API root:

```diff
- openai-base-url: https://pass.wafer.ai/v1/chat/completions
+ openai-base-url: https://pass.wafer.ai/v1
```

Recommended safe workflow:

```bash
cp ~/.config/pinocchio/profiles.yaml \
  ~/.config/pinocchio/profiles.yaml.bak-$(date +%Y%m%d-%H%M%S)

python3 - <<'PY'
from pathlib import Path
p = Path.home() / '.config' / 'pinocchio' / 'profiles.yaml'
text = p.read_text()
text = text.replace(
    'https://pass.wafer.ai/v1/chat/completions',
    'https://pass.wafer.ai/v1',
)
p.write_text(text)
PY
```

Then validate without making a model call:

```bash
PINOCCHIO_PROFILE=wafer-deepseek-v4-pro \
  pinocchio code professional --print-inference-settings --non-interactive hello
```

Expected output excerpt:

```yaml
settings:
  api:
    base_urls:
      openai-base-url: https://pass.wafer.ai/v1
```

Then validate with a real call:

```bash
PINOCCHIO_PROFILE=wafer-deepseek-v4-pro \
  pinocchio --log-level debug --with-caller code professional --non-interactive hello
```

Expected behavior: no 404; streamed response chunks are received.

### Code improvement 1: log the computed endpoint

Add debug logging after `resolveChatStreamConfig` and before `openChatCompletionStream`.

Suggested patch shape in `geppetto/pkg/steps/ai/openai/engine_openai.go`:

```go
streamCfg, err := resolveChatStreamConfig(e.settings.API, e.settings.Client, *e.settings.Chat.ApiType)
if err != nil {
    return nil, err
}
log.Debug().
    Str("api_type", string(*e.settings.Chat.ApiType)).
    Str("endpoint", streamCfg.endpoint).
    Msg("OpenAI chat completion endpoint resolved")
```

The endpoint contains no credential in the current implementation. Still, do not log headers or request body at this point.

Alternatively, log in `openChatCompletionStream` immediately before `cfg.httpClient.Do(req)`:

```go
log.Debug().
    Str("method", http.MethodPost).
    Str("url", req.URL.Redacted()).
    Msg("OpenAI chat completion HTTP request")
```

Using `req.URL.Redacted()` protects against accidental future userinfo in URLs.

### Code improvement 2: warn on likely endpoint/base-URL confusion

Add a small helper in `chat_stream.go`:

```go
func warnIfChatCompletionEndpointConfigured(apiType ai_types.ApiType, baseURL string) {
    normalized := strings.TrimRight(strings.ToLower(strings.TrimSpace(baseURL)), "/")
    if strings.HasSuffix(normalized, "/chat/completions") {
        log.Warn().
            Str("api_type", string(apiType)).
            Str("base_url", baseURL).
            Str("expected_shape", "provider API root, e.g. https://host/v1").
            Str("computed_suffix", "/chat/completions will be appended by Geppetto").
            Msg("OpenAI base URL appears to include the chat completions operation path")
    }
}
```

Call it before constructing `endpoint`:

```go
warnIfChatCompletionEndpointConfigured(apiType, baseURL)
endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
```

This preserves compatibility: existing configs still behave as before, but users get a direct warning explaining the double-append hazard.

### Code improvement 3: add unit coverage for URL construction

Extend `geppetto/pkg/steps/ai/openai/engine_openai_test.go` or add a focused `chat_stream_test.go`:

```go
func TestResolveChatStreamConfigAppendsChatCompletionsToBaseURL(t *testing.T) {
    ss := settings.NewStepSettings()
    ss.API.APIKeys["openai-api-key"] = "test-key"
    ss.API.BaseUrls["openai-base-url"] = "https://pass.wafer.ai/v1"

    cfg, err := resolveChatStreamConfig(ss.API, ss.Client, types.ApiTypeOpenAI)
    require.NoError(t, err)
    require.Equal(t, "https://pass.wafer.ai/v1/chat/completions", cfg.endpoint)
}
```

Optional second test:

```go
func TestResolveChatStreamConfigDoubleAppendDocumentsMisconfiguredEndpoint(t *testing.T) {
    ss := settings.NewStepSettings()
    ss.API.APIKeys["openai-api-key"] = "test-key"
    ss.API.BaseUrls["openai-base-url"] = "https://pass.wafer.ai/v1/chat/completions"

    cfg, err := resolveChatStreamConfig(ss.API, ss.Client, types.ApiTypeOpenAI)
    require.NoError(t, err)
    require.Equal(t, "https://pass.wafer.ai/v1/chat/completions/chat/completions", cfg.endpoint)
}
```

If the warning helper is implemented, capture zerolog output or keep the warning covered through a pure predicate helper:

```go
func looksLikeChatCompletionsEndpoint(baseURL string) bool
```

## Implementation phases

### Phase 1: Fix local Wafer profiles

1. Back up `~/.config/pinocchio/profiles.yaml`.
2. Replace Wafer `openai-base-url` values with `https://pass.wafer.ai/v1`.
3. Run `--print-inference-settings` for all Wafer profiles:
   - `wafer-qwen3.5-397b`
   - `wafer-deepseek-v4-pro`
   - `wafer-glm-5.1`
4. Confirm each profile reports the corrected base URL.
5. Run one real streaming smoke test.

### Phase 2: Add endpoint logging

1. Add debug log in `OpenAIEngine.RunInference` after `resolveChatStreamConfig`.
2. Ensure only the URL is logged; never log `Authorization`.
3. Run focused OpenAI tests:

```bash
cd geppetto
go test ./pkg/steps/ai/openai -count=1
```

4. Run a safe `--print-inference-settings` smoke path to confirm no debug output changes affect the no-network path.

### Phase 3: Add misconfiguration warning

1. Add `looksLikeChatCompletionsEndpoint(baseURL string) bool`.
2. Warn when true.
3. Add unit tests for:
   - `https://pass.wafer.ai/v1` → false;
   - `https://pass.wafer.ai/v1/` → false;
   - `https://pass.wafer.ai/v1/chat/completions` → true;
   - `https://pass.wafer.ai/v1/chat/completions/` → true.
4. Run focused package tests.

### Phase 4: Document the base URL contract

Update the OpenAI-compatible provider docs to explicitly distinguish base URL from operation endpoint.

Candidate docs:

- `geppetto/pkg/doc/topics/06-inference-engines.md`
- `pinocchio/pkg/doc/tutorials/08-migrating-legacy-pinocchio-config-to-unified-profile-documents.md`

Suggested wording:

> For `ai-api-type: openai`, configure `api.base_urls.openai-base-url` as the provider API root. The engine appends `/chat/completions` internally. For an OpenAI-compatible service whose chat endpoint is `https://example.com/v1/chat/completions`, set `openai-base-url: https://example.com/v1`.

## Testing and validation strategy

### Config-only validation

Run:

```bash
PINOCCHIO_PROFILE=wafer-deepseek-v4-pro \
  pinocchio code professional --print-inference-settings --non-interactive hello
```

Check:

```yaml
settings.api.base_urls.openai-base-url: https://pass.wafer.ai/v1
settings.chat.api_type: openai
settings.chat.engine: DeepSeek-V4-Pro
```

### Direct provider validation

Run a redacted curl probe:

```bash
curl -sS -w '\nHTTP_STATUS:%{http_code}\n' -X POST \
  https://pass.wafer.ai/v1/chat/completions \
  -H 'Authorization: Bearer ***' \
  -H 'Content-Type: application/json' \
  -d '{"model":"DeepSeek-V4-Pro","messages":[{"role":"user","content":"Hello!"}],"max_tokens":8,"stream":true}'
```

Expected: HTTP 200 with SSE frames.

### Negative validation

Run the same probe against the double-appended endpoint:

```bash
https://pass.wafer.ai/v1/chat/completions/chat/completions
```

Expected: HTTP 404. This verifies that the original symptom is explained by URL construction.

### Code validation after logging/warning changes

```bash
cd geppetto
go test ./pkg/steps/ai/openai -count=1
go test ./pkg/cli/bootstrap -count=1
```

If docs are changed:

```bash
cd pinocchio
go test ./pkg/cmds ./cmd/pinocchio/cmds -count=1
```

## Risks, alternatives, and open questions

### Risk: changing profile values affects any tool that expects full endpoint values

The local `profiles.yaml` is consumed by Pinocchio/Geppetto profile resolution. If another unrelated tool reads the same file and expects full operation endpoints, changing the value could affect that tool. The current evidence and field name (`openai-base-url`) indicate the Geppetto contract is base URL, not full endpoint.

### Risk: some OpenAI-compatible providers use nonstandard paths

If a provider does not use `/chat/completions`, the current OpenAI chat engine cannot be configured to use a fully custom operation path; it always appends `/chat/completions`. That is not the Wafer case, because Wafer's successful endpoint is exactly `/v1/chat/completions`.

### Alternative: make `openai-base-url` accept either root or full endpoint

The engine could detect a base URL ending in `/chat/completions` and avoid appending. This is user-friendly but changes a long-standing contract and can hide configuration mistakes. A warning is safer initially.

If implemented, use a separate setting such as `openai-chat-completions-url` rather than overloading `openai-base-url`.

### Alternative: use `ai-api-type: openai-responses`

Not appropriate for this incident. The working Wafer endpoint is OpenAI Chat Completions-compatible. The Responses engine has a different path and event contract.

### Open question: should profile values override CLI flags?

In the observed `--openai-base-url` test, the profile value won over the CLI flag in the final settings. That may be intended profile-first behavior, but it makes one-off command-line overrides less useful for debugging provider endpoints. Consider a separate ticket if operator overrides are expected to win.

## Implementation update: warning and profile stack applied

This follow-up has now been implemented in the working tree and local operator config.

### Runtime 404 hint

`geppetto/pkg/steps/ai/openai/chat_stream.go` now preserves the configured `baseURL` in `chatStreamConfig`. When a chat-completions HTTP request returns 404, it checks whether the configured base URL looks suspicious:

- contains `/chat/completion`, which usually means the operation endpoint was configured as the base URL; or
- contains extra path components after `/v1/` or `/v2/`, which often means the provider root was not used.

When the heuristic matches, the returned error includes a hint like:

```text
possible OpenAI-compatible base URL misconfiguration: configured base URL path "/v1/chat/completions" already looks like a chat completions endpoint; Geppetto appends /chat/completions internally
```

The same condition also emits a warning log with `status`, `base_url`, and computed `endpoint`, but no headers or API keys.

Focused validation passed:

```bash
cd geppetto && go test ./pkg/steps/ai/openai -count=1
```

### Local Wafer profile stack

`/home/manuel/.config/pinocchio/profiles.yaml` was backed up to:

```text
/home/manuel/.config/pinocchio/profiles.yaml.bak-20260503-114952-wafer-stack
```

The Wafer profiles were rewritten to use one shared base profile:

```yaml
wafer-base:
  inference_settings:
    chat:
      api_type: openai
    api:
      api_keys:
        openai-api-key: '***'
      base_urls:
        openai-base-url: https://pass.wafer.ai/v1

wafer-deepseek-v4-pro:
  stack:
    - profile_slug: wafer-base
  inference_settings:
    chat:
      engine: DeepSeek-V4-Pro
```

Validation showed `PINOCCHIO_PROFILE=wafer-deepseek-v4-pro` now resolves:

```yaml
api.base_urls.openai-base-url: https://pass.wafer.ai/v1
chat.api_type: openai
chat.engine: DeepSeek-V4-Pro
```

The live Wafer smoke test also succeeded and streamed:

```text
Hello. How can I help you today?
OpenAI stream completed chunks_received=9
OpenAI RunInference completed (streaming)
```

Evidence files:

- `sources/02-print-inference-settings-wafer-stacked-redacted.log`
- `sources/03-live-wafer-deepseek-after-profile-stack.log`

## References

- `geppetto/pkg/steps/ai/openai/chat_stream.go:60-64` — reads base URL and appends `/chat/completions`.
- `geppetto/pkg/steps/ai/openai/chat_stream.go:90` — creates the HTTP request with the computed endpoint.
- `geppetto/pkg/steps/ai/openai/engine_openai.go:56` — resolves streaming config.
- `geppetto/pkg/steps/ai/openai/engine_openai.go:213-216` — logs streaming mode and generic request failure.
- `geppetto/pkg/cli/bootstrap/inference_debug.go:145-165` — builds the settings/source trace used by `--print-inference-settings`.
- `~/.config/pinocchio/profiles.yaml` — local Wafer profile source; API keys are sensitive and must not be copied into reports.
- `sources/print_inference_settings_wafer_deepseek_redacted.log` — final resolved settings and source trace.
- `sources/live_wafer_deepseek_404.log` — live 404 reproduction with current profile.
- `sources/01-curl-endpoint-probe-redacted.md` — direct positive/negative Wafer endpoint probe.
