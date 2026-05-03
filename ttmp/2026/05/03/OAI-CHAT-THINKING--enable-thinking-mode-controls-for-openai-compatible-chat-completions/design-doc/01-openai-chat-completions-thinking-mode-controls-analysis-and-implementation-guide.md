---
Title: OpenAI chat completions thinking mode controls analysis and implementation guide
Ticket: OAI-CHAT-THINKING
Status: active
Topics:
    - llm
    - openai
    - inference
    - streaming
    - profiles
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/engine/inference_config.go
      Note: Generic per-turn reasoning effort exists but thinking toggle does not
    - Path: geppetto/pkg/steps/ai/openai/chat_types.go
      Note: OpenAI Chat Completions request type currently lacks thinking and reasoning_effort fields
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: Streaming path already emits thinking/reasoning events from reasoning_content deltas
    - Path: geppetto/pkg/steps/ai/openai/helpers.go
      Note: Request builder should wire thinking settings into ChatCompletionRequest
    - Path: geppetto/pkg/steps/ai/settings/openai/chat.yaml
      Note: Flag schema currently limits reasoning effort to Responses-oriented low/medium/high
    - Path: geppetto/pkg/steps/ai/settings/openai/settings.go
      Note: OpenAI settings need thinking toggle and chat-compatible effort semantics
    - Path: geppetto/ttmp/2026/05/03/OAI-CHAT-THINKING--enable-thinking-mode-controls-for-openai-compatible-chat-completions/sources/01-deepseek-thinking-mode-defuddle.md
      Note: DeepSeek thinking mode API source
    - Path: geppetto/ttmp/2026/05/03/OAI-CHAT-THINKING--enable-thinking-mode-controls-for-openai-compatible-chat-completions/sources/02-wafer-deepseek-thinking-probe-redacted.md
      Note: Live Wafer request-shape evidence
ExternalSources:
    - https://api-docs.deepseek.com/guides/thinking_mode
Summary: Design and implementation guide for adding DeepSeek-style thinking controls to Geppetto's OpenAI Chat Completions path.
LastUpdated: 2026-05-03T12:05:00-04:00
WhatFor: Use when implementing provider-native thinking controls for OpenAI-compatible chat-completions models such as Wafer DeepSeek-V4-Pro.
WhenToUse: Use before modifying OpenAI chat request structs, OpenAI settings, profile schema docs, and tests.
---









# OpenAI chat completions thinking mode controls analysis and implementation guide

## Executive summary

Geppetto already parses and streams `reasoning_content` from OpenAI-compatible Chat Completions responses, so DeepSeek/Wafer thinking output can be displayed once the provider emits it. The missing piece is request-side configuration: the OpenAI Chat Completions request type currently has no `thinking` field and does not send `reasoning_effort` on the chat-completions path.

DeepSeek's current OpenAI-format thinking contract is:

```json
{
  "thinking": {"type": "enabled"},
  "reasoning_effort": "high"
}
```

or for maximum effort:

```json
{
  "thinking": {"type": "enabled"},
  "reasoning_effort": "max"
}
```

or for fast/no-thinking mode:

```json
{
  "thinking": {"type": "disabled"}
}
```

A live Wafer probe confirmed that `DeepSeek-V4-Pro` accepts these fields on `https://pass.wafer.ai/v1/chat/completions`.

The recommended implementation is to add provider-agnostic-but-OpenAI-chat-scoped settings under `inference_settings.openai`, extend `ChatCompletionRequest`, and wire both profile defaults and per-turn overrides into `MakeCompletionRequestFromTurn`.

## Problem statement and scope

Users need to control thinking mode for OpenAI-compatible Chat Completions providers, especially Wafer-hosted DeepSeek V4. Today, profiles can choose the provider and model, and the engine can stream `reasoning_content`, but there is no supported profile field or CLI flag that emits DeepSeek's Chat Completions thinking controls.

In scope:

1. Add profile/flag settings for OpenAI Chat Completions thinking toggle and effort.
2. Add JSON request fields for `thinking` and `reasoning_effort` on the OpenAI chat path.
3. Preserve existing Responses API reasoning behavior.
4. Add tests for request JSON generation and streaming reasoning output compatibility.
5. Document profile examples for Wafer/DeepSeek V4.

Out of scope:

1. Implementing the feature in this document. This is an implementation guide.
2. Changing OpenAI Responses request semantics.
3. Forcing all providers to support `thinking`; the feature should be opt-in and should omit fields when unset.

## Source evidence

### DeepSeek docs

The Defuddle-extracted source is stored at:

- `sources/01-deepseek-thinking-mode-defuddle.md`

Key facts from the source:

- OpenAI-format thinking toggle:

```json
{"thinking": {"type": "enabled/disabled"}}
```

- OpenAI-format thinking effort:

```json
{"reasoning_effort": "high/max"}
```

- Defaults/mapping:
  - thinking defaults to `enabled`;
  - regular requests default to `high`;
  - some complex agent requests may be automatically set to `max`;
  - `low` and `medium` map to `high`;
  - `xhigh` maps to `max`.

- DeepSeek says thinking mode does not support `temperature`, `top_p`, `presence_penalty`, or `frequency_penalty`. For compatibility, setting them may not error, but has no effect.

### Wafer live probe

The redacted probe source is stored at:

- `sources/02-wafer-deepseek-thinking-probe-redacted.md`

Observed behavior:

1. `thinking.type=disabled` returned HTTP 200 with normal `message.content` and `reasoning_content: null`.
2. `thinking.type=enabled`, `reasoning_effort=high` returned HTTP 200 with `reasoning_content`.
3. `thinking.type=enabled`, `reasoning_effort=max` returned HTTP 200 with `reasoning_content`.

This confirms that Wafer's OpenAI-compatible endpoint accepts the DeepSeek request contract.

## Current-state architecture

### OpenAI Chat Completions request type lacks thinking fields

`geppetto/pkg/steps/ai/openai/chat_types.go` defines `ChatCompletionRequest`.

Line evidence:

- `chat_types.go:13-32` contains model, messages, max token fields, sampling fields, stream options, tool fields, and response format.
- There is currently no `Thinking` field.
- There is currently no `ReasoningEffort` field on the chat-completions request.

Current shape excerpt:

```go
type ChatCompletionRequest struct {
    Model               string
    Messages            []ChatCompletionMessage
    MaxTokens           int
    MaxCompletionTokens int
    Temperature         float32
    TopP                float32
    // ...
    ParallelToolCalls   *bool
}
```

### OpenAI Chat Completions request builder already centralizes request fields

`geppetto/pkg/steps/ai/openai/helpers.go` builds the request in `MakeCompletionRequestFromTurn`.

Line evidence:

- `helpers.go:360-368` reads OpenAI presence/frequency penalties.
- `helpers.go:370-378` sanitizes some settings for models recognized by `isReasoningModel`.
- `helpers.go:432-446` constructs `ChatCompletionRequest`.
- `helpers.go:448-470` applies structured output.

This is the correct place to wire profile-default thinking controls into the request.

### The OpenAI chat stream decoder already handles reasoning output

`geppetto/pkg/steps/ai/openai/engine_openai.go` consumes normalized stream events.

Line evidence:

- `engine_openai.go:273-280` checks `response.DeltaReasoning` and emits reasoning/thinking events.

`geppetto/pkg/steps/ai/openai/chat_stream.go` normalizes both `reasoning` and `reasoning_content` deltas. That means request-side controls should unlock existing output handling.

### Existing OpenAI settings have Responses-oriented reasoning fields

`geppetto/pkg/steps/ai/settings/openai/settings.go` already has:

```go
ReasoningEffort *string `yaml:"reasoning_effort,omitempty" glazed:"openai-reasoning-effort"`
ReasoningSummary *string `yaml:"reasoning_summary,omitempty" glazed:"openai-reasoning-summary"`
```

Line evidence:

- `settings.go:21-30` defines reasoning-related OpenAI settings.
- `chat.yaml:32-39` exposes `openai-reasoning-effort` as a choice of `low|medium|high` and describes it as "for Responses API models".

The problem is not that no reasoning setting exists. The problem is that the existing setting is Responses-oriented, does not allow `max`, and is only wired to `openai_responses` request construction.

### OpenAI Responses already maps reasoning effort

`geppetto/pkg/steps/ai/openai_responses/helpers.go` applies `s.OpenAI.ReasoningEffort` and `InferenceConfig.ReasoningEffort` to `req.Reasoning.Effort`.

Line evidence:

- `openai_responses/helpers.go:141-146` maps `s.OpenAI.ReasoningEffort`.
- `openai_responses/helpers.go:186-190` maps per-turn `InferenceConfig.ReasoningEffort`.

This path should continue to work and should not be broken by Chat Completions support.

### InferenceConfig has generic reasoning effort but no thinking toggle

`geppetto/pkg/inference/engine/inference_config.go` contains `ReasoningEffort`, but no thinking toggle.

Line evidence:

- `inference_config.go:19-21` defines `ReasoningEffort` as `low`, `medium`, `high` for OpenAI Responses.
- There is no `ThinkingType`, `ThinkingEnabled`, or equivalent field.

## Gap analysis

### Gap 1: no request JSON fields

The chat-completions path cannot emit this JSON today:

```json
"thinking": {"type": "enabled"},
"reasoning_effort": "max"
```

Even if a profile contains fields, the request struct cannot marshal them.

### Gap 2: no profile setting for thinking toggle

There is no `openai-thinking-type` or YAML field such as:

```yaml
openai:
  thinking_type: enabled
```

### Gap 3: existing reasoning effort choices omit DeepSeek `max`

DeepSeek supports `high|max`. Geppetto's `openai-reasoning-effort` and `inference-reasoning-effort` choices currently focus on `low|medium|high`.

For compatibility, DeepSeek maps `low|medium` to `high` and `xhigh` to `max`, but a first-class implementation should allow `max`.

### Gap 4: thinking-mode sampling sanitization is not tied to provider setting

DeepSeek docs say thinking mode ignores `temperature`, `top_p`, `presence_penalty`, and `frequency_penalty`. Current `helpers.go` only zeroes these for `isReasoningModel(engine)`. DeepSeek V4 model names may or may not be included in that heuristic. The implementation should explicitly sanitize when `thinking.type=enabled` for known DeepSeek/Wafer-style chat thinking requests, or at least document that providers may ignore those fields.

### Gap 5: multi-turn tool-call reasoning-content replay needs attention

DeepSeek docs state that if a thinking-mode assistant turn performed a tool call, `reasoning_content` must be passed back in subsequent requests. The current turn/message model stores reasoning text as blocks, and tool calls are stored separately. The implementation should verify whether existing message reconstruction can preserve `reasoning_content` in assistant messages when tool calls are involved.

This can be a phase-2 task if basic single-turn and non-tool multi-turn support is needed first.

## Proposed settings contract

### Add OpenAI chat thinking settings

Extend `geppetto/pkg/steps/ai/settings/openai/settings.go`:

```go
type Settings struct {
    // existing fields...

    // ThinkingType controls provider-native OpenAI Chat Completions thinking mode.
    // Known values: "", "enabled", "disabled".
    ThinkingType *string `yaml:"thinking_type,omitempty" glazed:"openai-thinking-type"`

    // ReasoningEffort applies to providers that accept Chat Completions
    // reasoning_effort. OpenAI Responses also uses this setting.
    // Known values should include low, medium, high, max, xhigh.
    ReasoningEffort *string `yaml:"reasoning_effort,omitempty" glazed:"openai-reasoning-effort"`
}
```

Add to `chat.yaml`:

```yaml
- name: openai-thinking-type
  type: choice
  choices:
    - ""
    - enabled
    - disabled
  help: Provider-native OpenAI Chat Completions thinking toggle; omit for providers that do not support it.
  default: ""

- name: openai-reasoning-effort
  type: choice
  choices:
    - ""
    - low
    - medium
    - high
    - max
    - xhigh
  help: Reasoning effort for providers that support it. For DeepSeek Chat Completions, high/max are native; low/medium map to high and xhigh maps to max.
  default: ""
```

Important design choice: change the default from `medium` to empty string if possible. Defaults should not emit provider-specific fields to every OpenAI-compatible provider.

If changing the default is too risky for OpenAI Responses behavior, introduce a separate chat-specific setting:

```go
ChatReasoningEffort *string `yaml:"chat_reasoning_effort,omitempty" glazed:"openai-chat-reasoning-effort"`
```

and keep existing `ReasoningEffort` for Responses. The tradeoff is more settings but fewer compatibility surprises.

### Optional generic InferenceConfig extension

If per-turn control is required, extend `InferenceConfig`:

```go
type InferenceConfig struct {
    // existing fields...
    ThinkingType *string `json:"thinking_type,omitempty" yaml:"thinking_type,omitempty" glazed:"inference-thinking-type"`
}
```

Then resolution order becomes:

1. turn `InferenceConfig.ThinkingType`,
2. `InferenceSettings.Inference.ThinkingType`,
3. `InferenceSettings.OpenAI.ThinkingType`,
4. unset/omit.

This is useful for UI/runtime toggles, but profile-only support can be implemented without it.

## Proposed request contract

Extend `geppetto/pkg/steps/ai/openai/chat_types.go`:

```go
type ChatCompletionRequest struct {
    // existing fields...
    ReasoningEffort string               `json:"reasoning_effort,omitempty"`
    Thinking        *ChatThinkingControl `json:"thinking,omitempty"`
}

type ChatThinkingControl struct {
    Type string `json:"type"`
}
```

Use `string` for `ReasoningEffort` to omit cleanly when empty.

## Request-building pseudocode

In `MakeCompletionRequestFromTurn`, after base parameter extraction and before constructing `ChatCompletionRequest`:

```go
thinkingType := ""
reasoningEffort := ""

if settings.OpenAI != nil {
    if settings.OpenAI.ThinkingType != nil {
        thinkingType = strings.TrimSpace(*settings.OpenAI.ThinkingType)
    }
    if settings.OpenAI.ReasoningEffort != nil {
        reasoningEffort = normalizeChatReasoningEffort(*settings.OpenAI.ReasoningEffort)
    }
}

if infCfg := engine.ResolveInferenceConfig(t, settings.Inference); infCfg != nil {
    if infCfg.ReasoningEffort != nil {
        reasoningEffort = normalizeChatReasoningEffort(*infCfg.ReasoningEffort)
    }
    if infCfg.ThinkingType != nil {
        thinkingType = strings.TrimSpace(*infCfg.ThinkingType)
    }
}

var thinking *ChatThinkingControl
switch thinkingType {
case "", "auto":
    // omit; provider default applies
case "enabled", "disabled":
    thinking = &ChatThinkingControl{Type: thinkingType}
default:
    return nil, fmt.Errorf("unsupported openai thinking_type %q", thinkingType)
}

if thinking != nil && thinking.Type == "enabled" {
    // Optional: sanitize sampling fields, or only warn.
    if isDeepSeekV4Like(engine) {
        temperature = 0
        topP = 0
        presencePenalty = 0
        frequencyPenalty = 0
    }
}
```

Then add fields to the request:

```go
req := ChatCompletionRequest{
    // existing fields...
    Thinking: thinking,
    ReasoningEffort: reasoningEffort,
}
```

Suggested normalization for DeepSeek compatibility:

```go
func normalizeChatReasoningEffort(v string) string {
    switch strings.ToLower(strings.TrimSpace(v)) {
    case "", "auto":
        return ""
    case "low", "medium":
        return "high"
    case "xhigh":
        return "max"
    case "high", "max":
        return strings.ToLower(strings.TrimSpace(v))
    default:
        return strings.ToLower(strings.TrimSpace(v)) // or reject strictly
    }
}
```

Strict validation is safer for profiles; permissive pass-through is safer for future providers. The recommended first implementation is strict for `openai-thinking-type` and permissive-but-documented for `reasoning_effort`.

## Profile examples

### Wafer DeepSeek V4 base plus variants

With the provider-base profile layout already used locally:

```yaml
wafer-base:
  slug: wafer-base
  inference_settings:
    chat:
      api_type: openai
    api:
      api_keys:
        openai-api-key: ${WAFER_API_KEY}
      base_urls:
        openai-base-url: https://pass.wafer.ai/v1

wafer-deepseek-v4-pro:
  slug: wafer-deepseek-v4-pro
  stack:
    - profile_slug: wafer-base
  inference_settings:
    chat:
      engine: DeepSeek-V4-Pro
    openai:
      thinking_type: enabled
      reasoning_effort: high

wafer-deepseek-v4-pro-max:
  slug: wafer-deepseek-v4-pro-max
  stack:
    - profile_slug: wafer-deepseek-v4-pro
  inference_settings:
    openai:
      thinking_type: enabled
      reasoning_effort: max

wafer-deepseek-v4-pro-fast:
  slug: wafer-deepseek-v4-pro-fast
  stack:
    - profile_slug: wafer-deepseek-v4-pro
  inference_settings:
    openai:
      thinking_type: disabled
```

### CLI examples

After adding flags:

```bash
PINOCCHIO_PROFILE=wafer-deepseek-v4-pro \
  pinocchio code professional --openai-thinking-type enabled --openai-reasoning-effort max hello
```

Fast/no-thinking:

```bash
PINOCCHIO_PROFILE=wafer-deepseek-v4-pro \
  pinocchio code professional --openai-thinking-type disabled hello
```

## Implementation phases

### Phase 1: Add settings and request fields

1. Add `ThinkingType` to `openai.Settings`.
2. Update `openai/chat.yaml` with `openai-thinking-type`.
3. Decide whether to expand existing `openai-reasoning-effort` choices or add `openai-chat-reasoning-effort`.
4. Add `ChatThinkingControl` and request fields to `ChatCompletionRequest`.
5. Run focused settings tests if present:

```bash
go test ./pkg/steps/ai/settings/openai ./pkg/steps/ai/settings -count=1
```

### Phase 2: Wire request construction

1. Add helper functions:
   - `normalizeChatThinkingType(string) (string, error)`
   - `normalizeChatReasoningEffort(string) string`
   - optionally `isDeepSeekV4Like(model string) bool`
2. In `MakeCompletionRequestFromTurn`, resolve settings and per-turn overrides.
3. Set `req.Thinking` only when configured.
4. Set `req.ReasoningEffort` only when configured.
5. Ensure default OpenAI-compatible requests do not gain new provider-specific fields.

### Phase 3: Add tests

Add request JSON tests in `geppetto/pkg/steps/ai/openai`.

Test cases:

1. No thinking settings → request JSON omits `thinking` and `reasoning_effort`.
2. `thinking_type: disabled` → request JSON includes:

```json
"thinking": {"type":"disabled"}
```

3. `thinking_type: enabled`, `reasoning_effort: high` → request JSON includes both fields.
4. `reasoning_effort: max` is accepted.
5. `reasoning_effort: low|medium|xhigh` normalization behavior is covered if implemented.
6. Invalid thinking type returns a clear error.

Example test skeleton:

```go
func TestMakeCompletionRequestFromTurnAddsDeepSeekThinkingControls(t *testing.T) {
    settings := testOpenAISettings("DeepSeek-V4-Pro")
    enabled := "enabled"
    max := "max"
    settings.OpenAI.ThinkingType = &enabled
    settings.OpenAI.ReasoningEffort = &max

    engine, err := NewOpenAIEngine(settings)
    require.NoError(t, err)
    req, err := engine.MakeCompletionRequestFromTurn(simpleUserTurn("hello"))
    require.NoError(t, err)

    require.Equal(t, &ChatThinkingControl{Type: "enabled"}, req.Thinking)
    require.Equal(t, "max", req.ReasoningEffort)
}
```

Add marshal test:

```go
raw, err := json.Marshal(req)
require.NoError(t, err)
require.JSONEq(t, `{"thinking":{"type":"enabled"},"reasoning_effort":"max"}`, extractRelevant(raw))
```

### Phase 4: Validate against Wafer

After implementation, run safe settings validation:

```bash
PINOCCHIO_PROFILE=wafer-deepseek-v4-pro-max \
  pinocchio code professional --print-inference-settings --non-interactive hello
```

Then live validation:

```bash
PINOCCHIO_PROFILE=wafer-deepseek-v4-pro-fast \
  pinocchio --log-level debug --with-caller code professional --non-interactive "Say hi"

PINOCCHIO_PROFILE=wafer-deepseek-v4-pro-max \
  pinocchio --log-level debug --with-caller code professional --non-interactive "Solve a small logic puzzle"
```

Expected observations:

- fast profile: mostly `content`, no/low `reasoning_content`;
- high/max profile: stream includes `reasoning_content` deltas and Geppetto emits thinking events.

### Phase 5: Documentation

Update:

- `geppetto/pkg/doc/topics/06-inference-engines.md`
- `geppetto/pkg/doc/topics/01-profiles.md` if profile examples are appropriate
- Pinocchio profile migration docs if user-facing profile examples are maintained there

Include:

- provider compatibility caveat;
- DeepSeek/Wafer example;
- warning that thinking mode may ignore sampling parameters;
- tool-call caveat about passing back `reasoning_content`.

## Risk analysis

### Risk: provider-specific fields sent to providers that reject them

Mitigation: omit `thinking` and `reasoning_effort` by default. Only send when explicitly configured.

### Risk: changing default `openai-reasoning-effort` breaks Responses behavior

Mitigation options:

1. keep existing `openai-reasoning-effort` default for Responses and add separate chat-specific field; or
2. change default to empty and update Responses code to apply its own default if needed.

The safer first implementation is a new chat-specific field if backward compatibility is a priority.

### Risk: ambiguous semantics of `reasoning_effort` across providers

DeepSeek accepts `high|max`; OpenAI Responses accepts `low|medium|high`; other OpenAI-compatible providers may accept different values or reject the field.

Mitigation: document that this is opt-in provider-native behavior. Do not auto-send for every model.

### Risk: tool-call replay with reasoning content

DeepSeek requires `reasoning_content` replay after tool calls. If Geppetto drops it when reconstructing assistant tool-call messages, multi-turn tool workflows may fail.

Mitigation: add a dedicated tool-call regression after the basic request controls land.

## Acceptance criteria

1. A profile can configure:

```yaml
openai:
  thinking_type: enabled
  reasoning_effort: max
```

2. The OpenAI Chat Completions request JSON includes:

```json
"thinking": {"type":"enabled"},
"reasoning_effort":"max"
```

3. When fields are unset, existing OpenAI-compatible requests are unchanged.
4. `thinking_type: disabled` emits `"thinking":{"type":"disabled"}`.
5. Streaming `reasoning_content` continues to emit Geppetto thinking events.
6. Focused OpenAI tests pass:

```bash
go test ./pkg/steps/ai/openai -count=1
```

7. Settings tests pass:

```bash
go test ./pkg/steps/ai/settings ./pkg/steps/ai/settings/openai -count=1
```

8. A live Wafer smoke test confirms DeepSeek V4 accepts the generated fields.

## References

- `sources/01-deepseek-thinking-mode-defuddle.md` — DeepSeek thinking-mode source documentation.
- `sources/02-wafer-deepseek-thinking-probe-redacted.md` — live Wafer acceptance probe.
- `geppetto/pkg/steps/ai/openai/chat_types.go:13-32` — current Chat Completions request shape.
- `geppetto/pkg/steps/ai/openai/helpers.go:432-446` — request construction site.
- `geppetto/pkg/steps/ai/openai/engine_openai.go:273-280` — existing reasoning delta event publication.
- `geppetto/pkg/steps/ai/settings/openai/settings.go:21-30` — existing OpenAI reasoning settings.
- `geppetto/pkg/steps/ai/settings/openai/chat.yaml:32-39` — existing `openai-reasoning-effort` choices.
- `geppetto/pkg/steps/ai/openai_responses/helpers.go:141-146` — Responses reasoning effort mapping.
- `geppetto/pkg/inference/engine/inference_config.go:19-21` — generic reasoning effort override.
