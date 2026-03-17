---
Title: Provider-native token counting for geppetto and pinocchio
Ticket: PI-03-TOKEN-COUNT-APIS
Status: active
Topics:
    - pinocchio
    - geppetto
    - glazed
    - profiles
    - analysis
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/metadata.go
      Note: Existing normalized usage metadata used for post-inference results only
    - Path: geppetto/pkg/inference/engine/factory/factory.go
      Note: Provider dispatch pattern to mirror for token counting
    - Path: geppetto/pkg/steps/ai/claude/api/messages.go
      Note: Current Claude request envelope showing inference-only fields to avoid reusing blindly
    - Path: geppetto/pkg/steps/ai/claude/helpers.go
      Note: Existing Turn-to-Claude request builder that should be refactored for reuse
    - Path: geppetto/pkg/steps/ai/openai_responses/helpers.go
      Note: Existing Turn-to-Responses request builder and reusable input projection
    - Path: geppetto/pkg/turns/inference_result.go
      Note: Existing durable inference result shape intentionally kept separate from preflight counts
    - Path: pinocchio/cmd/pinocchio/cmds/tokens/count.go
      Note: Current offline token count command and extension point
    - Path: pinocchio/cmd/pinocchio/cmds/tokens/helpers.go
      Note: Registration path for token subcommands and command-builder choice
    - Path: pinocchio/cmd/pinocchio/main.go
      Note: Root command wiring for tokens command group
    - Path: pinocchio/pkg/cmds/cobra.go
      Note: Geppetto-aware Cobra wrapper recommended for api-backed count mode
ExternalSources: []
Summary: Detailed design for adding OpenAI and Claude input-token counting to geppetto and exposing it via pinocchio tokens count.
LastUpdated: 2026-03-05T14:49:03.593472423-05:00
WhatFor: Guide an unfamiliar engineer through the current CLI/runtime architecture, the relevant provider APIs, and a concrete implementation plan for provider-native token counting.
WhenToUse: Use when implementing, reviewing, or onboarding someone onto the token-counting feature across geppetto and pinocchio.
---


# Provider-native token counting for geppetto and pinocchio

## Executive Summary

The requested feature is not just "count some tokens." It spans two layers with different responsibilities:

- `geppetto` is the provider/runtime layer that already knows how to turn a `turns.Turn` into provider-specific OpenAI and Claude requests.
- `pinocchio` is the CLI layer that already has a `tokens count` command, but that command is a legacy offline estimator built directly on local tokenizer libraries and is not wired into profiles, provider credentials, or provider-native request shapes.

The lowest-risk design is:

1. Add a new provider-backed token-counting facade in `geppetto`.
2. Implement OpenAI counting with the official `POST /v1/responses/input_tokens` endpoint.
3. Implement Claude counting with the official `POST /v1/messages/count_tokens` endpoint.
4. Reuse existing Turn-to-provider request projection logic where it is already factored well, and extract shared projection helpers where it is not.
5. Extend `pinocchio tokens count` with a mode flag instead of creating a separate top-level command.

The most important architectural decision is to keep counting separate from the inference engine interface. Preflight token counting is not an inference run, does not publish event streams, and should not be shoehorned into `engine.Engine`.

## Problem Statement

The current system has a mismatch between what the user asked for and what the code currently supports.

Observed current state:

- `pinocchio` registers a `tokens` command group at `pinocchio/cmd/pinocchio/main.go:258`, and the existing `count` subcommand is created in `pinocchio/cmd/pinocchio/cmds/tokens/helpers.go:26-55`.
- The current `tokens count` implementation in `pinocchio/cmd/pinocchio/cmds/tokens/count.go:18-99` only:
  - reads a string/file argument,
  - selects a local tokenizer codec,
  - counts locally encoded token IDs,
  - prints `Model`, `Codec`, and `Total tokens`.
- The model-to-codec mapping in `pinocchio/cmd/pinocchio/cmds/tokens/encode.go:97-109` is legacy OpenAI-centric. It knows a few OpenAI families and falls back to `r50k_base`. It has no Claude-aware behavior.
- `pinocchio` also has a second local-count path in `pinocchio/cmd/pinocchio/cmds/clip.go:110-131`, where stats are hard-coded to `cl100k_base`. That is another sign that token counting is currently an offline utility, not a provider-aware system surface.

At the same time, `geppetto` already has the pieces that make provider-native counting practical:

- `geppetto/pkg/inference/engine/factory/factory.go:50-93` chooses provider engines from `StepSettings`.
- `geppetto/pkg/steps/ai/openai_responses/helpers.go:112-242` already builds a Responses request from `StepSettings` plus `turns.Turn`.
- `geppetto/pkg/steps/ai/claude/helpers.go:24-257` already builds a Claude Messages request from `StepSettings` plus `turns.Turn`.
- `geppetto/pkg/events/metadata.go:3-23` and `geppetto/pkg/turns/inference_result.go:15-40` already normalize post-inference usage metadata, but they are about actual provider usage after inference, not preflight counting.

So the problem is:

- the CLI has a token-count command, but it is not provider-native;
- the runtime knows how to build provider requests, but it has no count-only facade;
- there is no clean bridge between the two.

## Scope

In scope:

- OpenAI official API input-token counting.
- Anthropic official API input-token counting.
- `pinocchio tokens count` support for choosing local estimate versus provider API count.
- Profile-aware provider/model/credential resolution for the API-backed path.
- Unit and integration-level tests around request projection and HTTP handling.

Out of scope for the first implementation:

- Gemini or other providers.
- Persisting count results as if they were inference results.
- Estimating output tokens.
- Counting "the entire runtime prompt" after arbitrary middleware mutation. The first version should count the `Turn` the command constructs plus the settings/fields the count path explicitly knows about.
- OpenAI-compatible third-party endpoints like AnyScale or Fireworks unless they independently prove compatibility with the official count endpoint shapes.

## System Orientation For A New Intern

### What geppetto is

`geppetto` is the provider/runtime layer. Its job is to:

- hold cross-provider data structures like `turns.Turn`,
- parse and merge provider settings into `StepSettings`,
- choose the correct inference engine,
- translate `Turn` blocks into provider-specific HTTP payloads,
- normalize provider outputs back into provider-agnostic metadata.

The files to read first are:

- `geppetto/pkg/steps/ai/settings/settings-step.go`
- `geppetto/pkg/steps/ai/settings/settings-chat.go`
- `geppetto/pkg/inference/engine/factory/factory.go`
- `geppetto/pkg/steps/ai/openai_responses/helpers.go`
- `geppetto/pkg/steps/ai/claude/helpers.go`

### What pinocchio is

`pinocchio` is the CLI/integration layer. Its job is to:

- register top-level Cobra commands,
- attach glazed/geppetto parsing middleware when a command needs profile/provider settings,
- decode command input and print user-facing output.

The files to read first are:

- `pinocchio/cmd/pinocchio/main.go`
- `pinocchio/pkg/cmds/cobra.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/helpers.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/count.go`

### How settings flow from CLI flags into provider logic

The important configuration path is:

```text
pinocchio Cobra command
  -> geppetto middlewares
  -> parsed glazed values
  -> settings.StepSettings
  -> geppetto provider factory / helper
  -> provider-specific HTTP request
```

Concrete evidence:

- `pinocchio/pkg/cmds/cobra.go:12-25` wraps a command with `sections.GetCobraCommandGeppettoMiddlewares`.
- `geppetto/pkg/sections/sections.go:171-340` resolves config, env, profiles, and CLI flags in the correct precedence order.
- `geppetto/pkg/steps/ai/settings/settings-step.go:271-325` decodes parsed values into `StepSettings`.

This matters because the new API-backed count path should consume the same `StepSettings` object the inference stack uses. That keeps provider selection, model selection, and credential loading consistent.

### What a Turn is

The token-count APIs should operate on a `turns.Turn`, because that is the canonical prompt/history container already used by inference engines.

For this feature, think of a `Turn` as "a list of ordered blocks," where each block can be:

- system text,
- user text,
- assistant text,
- tool call,
- tool result,
- reasoning block,
- or another provider-specific extension.

That matters because provider-native token counting should count the actual provider request shape, not just the raw user string.

## Current-State Architecture

### Current pinocchio token count path

The current `pinocchio tokens count` command is a plain local utility:

- `pinocchio/cmd/pinocchio/cmds/tokens/count.go:23-35` defines only `model` and `codec` flags.
- `pinocchio/cmd/pinocchio/cmds/tokens/count.go:66-95` chooses a codec locally and counts encoded token IDs.
- `pinocchio/cmd/pinocchio/cmds/tokens/helpers.go:27-31` builds this command with the generic `cli.BuildCobraCommand`, not the geppetto-aware wrapper.

Implications:

- no `--profile`,
- no `--profile-registries`,
- no provider credentials,
- no provider-specific request construction,
- no way to count tools, system prompts, or reasoning metadata in the provider's own wire format.

### Current geppetto provider selection path

Provider engine selection already exists and is centralized:

- `geppetto/pkg/inference/engine/factory/factory.go:55-93` dispatches on `settings.Chat.ApiType`.
- `geppetto/pkg/steps/ai/types/types.go:5-17` defines provider constants like `openai`, `openai-responses`, and `claude`.

This is useful because the new counting facade can follow the same provider dispatch idea without changing inference behavior.

### Current OpenAI request-building path

There are two OpenAI-related request builders:

- `geppetto/pkg/steps/ai/openai/helpers.go:85-220` builds Chat Completions requests.
- `geppetto/pkg/steps/ai/openai_responses/helpers.go:112-242` builds Responses requests.

The OpenAI Responses helper is particularly important because:

- it already converts a `Turn` into `input` items via `buildInputItemsFromTurn` in `geppetto/pkg/steps/ai/openai_responses/helpers.go:266-320`,
- it already threads through reasoning, structured output, tool choice, and parallel tool-call options,
- it already uses the same OpenAI key/base URL fields that the runtime uses.

Inference:

- the new OpenAI token-count implementation should be built beside the Responses helper, not inside the old Chat Completions helper.
- even when the user's provider selection is `openai`, the provider-native count path should still use the official OpenAI count endpoint, which lives under the Responses API surface.

### Current Claude request-building path

Claude already has a strong Turn-to-request conversion path, but it is less factored than the OpenAI Responses one:

- `geppetto/pkg/steps/ai/claude/helpers.go:24-257` builds a full `api.MessageRequest`,
- the same function mixes:
  - block-to-message conversion,
  - system prompt handling,
  - tool ordering,
  - sampling parameter validation,
  - and final request assembly.

Inference:

- Claude token counting should not duplicate this whole function.
- the first refactor should extract a reusable "message projection" helper that yields `system` plus `messages`, then let inference and count requests assemble their provider-specific envelopes from that shared projection.

### Existing usage metadata path

`geppetto` already has a provider-agnostic usage shape:

- `geppetto/pkg/events/metadata.go:3-23`
- `geppetto/pkg/turns/inference_result.go:15-40`

That shape is for actual inference results and includes input, output, cached, and cache-read metrics.

Important design constraint:

- do not overload these structures for preflight count-only operations.
- a count-only operation should return a dedicated count result type rather than pretending it was a completed inference.

## External API Ground Truth

Verified on 2026-03-05 using official docs.

### OpenAI

Official source:

- `https://developers.openai.com/api/reference/responses/input_tokens`

Observed from the official API reference/search result:

- OpenAI documents a dedicated endpoint for input token counts.
- The documented path is `POST /v1/responses/input_tokens`.
- The documented response shape includes an object named `response.input_tokens` and an `input_tokens` integer.
- The documented request body accepts the same major input-shaping concepts used by Responses, including conversation/input, model, reasoning, text, tool choice, tools, and truncation.

Practical consequence:

- for OpenAI, provider-native counting should be implemented against the Responses API count endpoint, not against local `tiktoken` heuristics.
- the request should be built from the same Turn-to-Responses projection used by inference, but sent through a narrower count-specific request struct containing only documented count fields.

### Anthropic / Claude

Official sources:

- `https://docs.anthropic.com/en/api/messages-count-tokens`
- `https://docs.anthropic.com/en/api/client-sdks`

Observed from the official API reference:

- Anthropic documents `POST /v1/messages/count_tokens`.
- The documentation explicitly states that the endpoint counts the number of tokens in a list of messages and also accounts for system prompts and tools.
- The documented response includes an `input_tokens` field.
- The same docs surface additional request-shaping fields such as `thinking`, `tool_choice`, and `tools`.

Practical consequence:

- for Claude, provider-native counting should use the Messages count endpoint.
- the builder should preserve the same user/system/tool ordering rules as the inference path.
- it should not blindly reuse inference-only fields like `stream`.

## Gap Analysis

The gaps between current code and the requested outcome are straightforward:

1. There is no provider-native counting facade in `geppetto`.
2. The current pinocchio command is not wired into `StepSettings`, profiles, or provider credentials.
3. OpenAI and Claude request builders are asymmetrically factored:
   - OpenAI Responses already exposes a reusable projection seam.
   - Claude currently needs one refactor step to avoid duplication.
4. There is no normalized result type for preflight token counts.
5. There are no tests around token-count endpoints because the feature does not exist yet.

## Proposed Solution

### High-level design

Add a new token-counting subsystem in `geppetto` and call it from `pinocchio tokens count`.

Recommended package layout:

```text
geppetto/
  pkg/
    inference/
      tokencount/
        types.go
        counter.go
        factory.go
    steps/
      ai/
        openai_responses/
          token_count.go
        claude/
          token_count.go
```

The subsystem responsibilities should be:

- `pkg/inference/tokencount`
  - define a provider-agnostic interface and result type,
  - dispatch on `StepSettings`,
  - stay independent from `engine.Engine`.
- `pkg/steps/ai/openai_responses/token_count.go`
  - build and send `POST /responses/input_tokens`.
- `pkg/steps/ai/claude/token_count.go`
  - build and send `POST /messages/count_tokens`.

### Why not add this to `engine.Engine`

Do not change `engine.Engine` to add `CountTokens`.

Reasons:

- Counting is a preflight query, not an inference run.
- It does not produce streamed events.
- It does not mutate a conversation the way inference does.
- It would force unrelated engines and test fakes to implement a method that is conceptually outside their main job.

This is a classic case for a sibling subsystem, not an engine-interface expansion.

### Proposed geppetto API

Suggested types:

```go
package tokencount

type Source string

const (
    SourceProviderAPI Source = "provider_api"
    SourceEstimate    Source = "estimate"
)

type Result struct {
    Provider    string
    Model       string
    InputTokens int
    Source      Source
    Endpoint    string
    RequestKind string
}

type Counter interface {
    CountTurn(ctx context.Context, t *turns.Turn) (*Result, error)
}
```

Suggested factory:

```go
func NewFromStepSettings(ss *settings.StepSettings) (Counter, error) {
    provider := normalizedProviderFromStepSettings(ss)

    switch provider {
    case "openai", "openai-responses":
        return openai_responses.NewTokenCounter(ss), nil
    case "claude", "anthropic":
        return claude.NewTokenCounter(ss), nil
    default:
        return nil, fmt.Errorf("provider %q does not support provider token counting", provider)
    }
}
```

### OpenAI implementation details

Recommended implementation strategy:

1. Keep `buildInputItemsFromTurn` as the core projection function.
2. Introduce a new count-specific request struct, narrower than `responsesRequest`.
3. Reuse the same model, reasoning, text format, and tool-related fields when they are documented for the count endpoint.
4. Send the request to `POST {baseURL}/responses/input_tokens`.
5. Parse a small count response.

Important detail:

- do not send the full inference request struct unchanged.
- `responsesRequest` currently includes fields like `Stream`, `Include`, `Store`, and `ServiceTier` in `geppetto/pkg/steps/ai/openai_responses/helpers.go:15-31`.
- those are inference-oriented and should not be assumed valid for the count endpoint unless the official count docs explicitly say so.

Suggested request/response sketch:

```go
type inputTokensRequest struct {
    Model      string           `json:"model"`
    Input      []responsesInput `json:"input"`
    Text       *responsesText   `json:"text,omitempty"`
    Reasoning  *reasoningParam  `json:"reasoning,omitempty"`
    Tools      []any            `json:"tools,omitempty"`
    ToolChoice any              `json:"tool_choice,omitempty"`
}

type inputTokensResponse struct {
    Object      string `json:"object"`
    InputTokens int    `json:"input_tokens"`
}
```

### Claude implementation details

Recommended implementation strategy:

1. Extract a projection helper from `MakeMessageRequestFromTurn`.
2. The projection helper should return the data common to both inference and counting:
   - `system` prompt,
   - ordered `messages`,
   - optionally tool declarations and count-safe tool-choice/thinking fields when supported.
3. Build a separate count request struct for the count endpoint.
4. Send the request to `POST {baseURL}/v1/messages/count_tokens`.
5. Parse the returned `input_tokens`.

Suggested refactor shape:

```go
type MessageProjection struct {
    System   string
    Messages []api.Message
}

func (e *ClaudeEngine) buildMessageProjectionFromTurn(t *turns.Turn) (*MessageProjection, error) {
    // extracted from MakeMessageRequestFromTurn
}
```

Then:

```go
type countTokensRequest struct {
    Model      string        `json:"model"`
    System     string        `json:"system,omitempty"`
    Messages   []api.Message `json:"messages"`
    Tools      []api.Tool    `json:"tools,omitempty"`
    ToolChoice any           `json:"tool_choice,omitempty"`
    Thinking   any           `json:"thinking,omitempty"`
}
```

Important detail:

- do not reuse `api.MessageRequest` unchanged because it includes inference-oriented fields such as `MaxTokens` and `Stream` at `geppetto/pkg/steps/ai/claude/api/messages.go:20-34`.
- the count request should include only fields documented for the count endpoint.

### Pinocchio CLI design

Recommendation: extend `pinocchio tokens count` rather than introducing a new top-level verb.

Why:

- the command already exists and already has the right user mental model,
- the user explicitly said "or flags on token count, to be precise,"
- it preserves backward compatibility for operators who just want a fast local estimate.

Recommended new flags on `tokens count`:

- `--count-mode=estimate|api|auto`
- keep existing `--model` and `--codec`
- inherit geppetto-aware flags by building this command with `BuildCobraCommandWithGeppettoMiddlewares`

Behavior:

- `estimate`
  - current behavior,
  - fully local,
  - works without credentials,
  - respects `--model` or `--codec`.
- `api`
  - requires provider/model selection through profile or explicit flags,
  - requires valid credentials,
  - returns provider-native input-token count.
- `auto`
  - try provider-native count for supported providers,
  - fall back to local estimate only when provider API counting is unavailable or not configured.

The `count` command should be the only token subcommand upgraded to geppetto-aware middleware in the first pass. `encode`, `decode`, `list-models`, and `list-codecs` can remain plain utility commands.

### CLI control flow

Recommended control flow in `pinocchio/cmd/pinocchio/cmds/tokens/count.go`:

```go
func (cc *CountCommand) RunIntoWriter(ctx context.Context, parsed *values.Values, w io.Writer) error {
    s := decodeCountSettings(parsed)

    if s.CountMode == "estimate" {
        return runLocalEstimate(s, w)
    }

    ss, err := settings.NewStepSettingsFromParsedValues(parsed)
    if err != nil {
        return err
    }

    turn := &turns.Turn{
        Blocks: []turns.Block{
            turns.NewUserTextBlock(s.Input),
        },
    }

    counter, err := tokencount.NewFromStepSettings(ss)
    if err != nil {
        if s.CountMode == "auto" {
            return runLocalEstimate(s, w)
        }
        return err
    }

    res, err := counter.CountTurn(ctx, turn)
    if err != nil {
        if s.CountMode == "auto" {
            return runLocalEstimate(s, w)
        }
        return err
    }

    return printResult(w, res)
}
```

### Proposed architecture diagram

```text
Current
-------
pinocchio tokens count
  -> local codec selection
  -> local tokenizer library
  -> printed token count

Proposed
--------
pinocchio tokens count --count-mode api
  -> geppetto command middlewares
  -> parsed values + profiles + env + flags
  -> settings.StepSettings
  -> tokencount factory
  -> provider-specific count client
  -> official provider count endpoint
  -> normalized result
  -> printed token count
```

### File-by-file implementation plan

Phase 1: geppetto token-count facade

- Add `geppetto/pkg/inference/tokencount/types.go`
- Add `geppetto/pkg/inference/tokencount/factory.go`
- Add `geppetto/pkg/inference/tokencount/counter.go`

Phase 2: OpenAI

- Add `geppetto/pkg/steps/ai/openai_responses/token_count.go`
- Reuse `buildInputItemsFromTurn` from `geppetto/pkg/steps/ai/openai_responses/helpers.go`
- Keep request struct narrow and count-specific

Phase 3: Claude

- Refactor `geppetto/pkg/steps/ai/claude/helpers.go`
- Extract a shared projection helper
- Add `geppetto/pkg/steps/ai/claude/token_count.go`
- Add or reuse a small HTTP helper in `geppetto/pkg/steps/ai/claude/api`

Phase 4: pinocchio CLI

- Update `pinocchio/cmd/pinocchio/cmds/tokens/count.go`
- Update `pinocchio/cmd/pinocchio/cmds/tokens/helpers.go`
- Build the `count` subcommand with `pinocchio/pkg/cmds.BuildCobraCommandWithGeppettoMiddlewares`
- Keep the rest of the token utility commands unchanged

Phase 5: tests and docs

- Add `httptest` coverage for both provider clients
- Add CLI tests for `estimate`, `api`, and `auto`
- Update user-facing help for `tokens count`

## Design Decisions

### Decision 1: Extend `tokens count`, do not add a new top-level CLI command

Reasoning:

- It matches the current command layout.
- It preserves backward compatibility.
- It matches the user's explicit preference.

### Decision 2: Create a sibling counting subsystem, do not widen `engine.Engine`

Reasoning:

- It keeps preflight analysis separate from inference execution.
- It avoids touching every engine implementation and test double.

### Decision 3: Keep offline and provider-native counting both available

Reasoning:

- The existing local count path is useful and fast.
- Provider-native counts are more correct for provider-specific wire shapes.
- Users need both.

### Decision 4: Reuse request projection logic, but do not blindly reuse inference request structs

Reasoning:

- Shared projection reduces drift.
- Count endpoints have different documented bodies than inference endpoints.
- Narrow request structs make accidental unsupported-field leakage much less likely.

### Decision 5: Do not persist count results as inference metadata

Reasoning:

- A count-only request is analysis, not generation.
- Persisting it into `InferenceResult` or event streams would create misleading semantics.

## Alternatives Considered

### Alternative A: Keep everything local with tokenizer libraries

Rejected because:

- it does not satisfy the requirement to count through provider APIs,
- it is not Claude-aware in the current command,
- it drifts from the true provider wire payload when tools, reasoning, or structured-output fields matter.

### Alternative B: Add `CountTokens` to every inference engine

Rejected because:

- it is the wrong abstraction boundary,
- it increases interface churn,
- it couples preflight analysis to event-stream-producing engines.

### Alternative C: Add a new `pinocchio token-count-api` command

Rejected because:

- it duplicates the existing `tokens count` surface,
- it makes the CLI harder to learn,
- it is unnecessary given the user's "flags on token count" guidance.

### Alternative D: Reuse Chat Completions request builders for OpenAI counting

Rejected because:

- the official count endpoint being targeted is on the Responses API surface,
- a chat-completions-only path would either stay approximate or require a separate undocumented assumption.

## Testing Strategy

### Unit tests

OpenAI:

- verify Turn-to-count-request conversion for:
  - plain user/system text,
  - reasoning-enabled requests,
  - structured output,
  - tool declarations,
  - tool call history.

Claude:

- verify extracted projection preserves:
  - system prompt,
  - user/assistant ordering,
  - tool-use adjacency,
  - omitted inference-only fields on count requests.

### HTTP tests

Use `httptest.Server` for both providers:

- success response parsing,
- malformed JSON,
- non-2xx responses,
- missing credentials,
- unsupported provider/base URL cases.

### CLI tests

Add tests for:

- `tokens count --count-mode estimate`
- `tokens count --count-mode api --ai-api-type openai-responses`
- `tokens count --count-mode api --ai-api-type claude`
- `tokens count --count-mode auto` with provider failure falling back to estimate

### Regression tests

Protect existing behavior:

- existing local count output still works without provider credentials,
- `encode`/`decode`/`list-*` token commands remain unaffected,
- count command does not require profiles in `estimate` mode.

## Risks And Open Questions

### Risk: OpenAI-compatible providers may not support the official count endpoint

`geppetto` supports providers like AnyScale and Fireworks in the inference factory. Those should not be treated as automatically compatible with the OpenAI count endpoint.

Recommendation:

- first implementation supports only official OpenAI and Anthropic.
- return a clear error for unsupported providers in `api` mode.

### Risk: "prompt text" and "provider request payload" are not identical

If the CLI constructs only a single user text block, the result is accurate for that constructed `Turn`, not for every possible middleware-expanded runtime request.

Recommendation:

- document this clearly,
- keep the first version narrow,
- add richer transcript/system/tool input options later only if users actually need them.

### Open question: Should OpenAI `openai` and `openai-responses` both route to the same count client?

Recommended answer:

- yes for official OpenAI credentials/base URLs,
- no for third-party OpenAI-compatible providers unless explicitly verified.

### Open question: Should Claude count requests include structured-output fields?

Current evidence from the Anthropic count docs clearly mentions messages, system, tools, and related input-shaping fields, but not enough in this investigation to confidently claim parity for every inference-only field.

Recommendation:

- keep the first Claude count request to documented count-safe fields,
- add more fields only when verified against official docs and tests.

## Implementation Plan

### Phase 1: Add geppetto facade

1. Create a dedicated token-count package in `geppetto/pkg/inference/tokencount`.
2. Define result types and provider-dispatch logic.
3. Keep the public surface minimal and synchronous.

### Phase 2: Implement OpenAI count client

1. Build a narrow request struct for `/responses/input_tokens`.
2. Reuse `buildInputItemsFromTurn`.
3. Parse the documented count response.
4. Add `httptest` coverage.

### Phase 3: Refactor Claude projection and implement count client

1. Extract common Turn-to-message projection from `MakeMessageRequestFromTurn`.
2. Build a narrow count request struct.
3. Parse the documented `input_tokens` response.
4. Add `httptest` coverage.

### Phase 4: Wire pinocchio CLI

1. Add `count-mode` flag to `tokens count`.
2. Keep current estimate path intact.
3. Build the count subcommand with geppetto middlewares.
4. Convert the raw input string into a synthetic `turns.Turn`.
5. Print normalized results.

### Phase 5: Polish

1. Add CLI tests.
2. Update help text.
3. Add docs/examples after code lands.

## References

### Key repository files

- `pinocchio/cmd/pinocchio/main.go:256-259` - root command registration for `openai` and `tokens`
- `pinocchio/cmd/pinocchio/cmds/tokens/helpers.go:26-64` - token command registration and current command builder choices
- `pinocchio/cmd/pinocchio/cmds/tokens/count.go:18-99` - current offline count command
- `pinocchio/cmd/pinocchio/cmds/tokens/encode.go:97-109` - legacy model-to-codec mapping
- `pinocchio/cmd/pinocchio/cmds/clip.go:110-131` - second local count path using a hard-coded encoding
- `pinocchio/pkg/cmds/cobra.go:12-25` - geppetto-aware Cobra builder
- `geppetto/pkg/sections/sections.go:171-340` - profile/config/env/flag middleware chain
- `geppetto/pkg/steps/ai/settings/settings-step.go:271-325` - parsed-values to `StepSettings`
- `geppetto/pkg/inference/engine/factory/factory.go:50-93` - provider dispatch
- `geppetto/pkg/steps/ai/openai_responses/helpers.go:112-242` - Responses request assembly
- `geppetto/pkg/steps/ai/openai_responses/helpers.go:266-320` - Turn-to-input-items conversion
- `geppetto/pkg/steps/ai/claude/helpers.go:24-257` - Claude request assembly
- `geppetto/pkg/steps/ai/claude/api/messages.go:19-34` - current Claude inference request type
- `geppetto/pkg/events/metadata.go:3-23` - normalized post-inference usage metadata
- `geppetto/pkg/turns/inference_result.go:15-40` - durable inference result shape

### External API references

- OpenAI Responses input token counts: `https://developers.openai.com/api/reference/responses/input_tokens`
- Anthropic Messages count tokens: `https://docs.anthropic.com/en/api/messages-count-tokens`
- Anthropic client SDKs: `https://docs.anthropic.com/en/api/client-sdks`

## Proposed Solution

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
