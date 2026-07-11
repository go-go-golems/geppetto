---
Title: Provider-aware inference-profile default normalization guide
Ticket: GEP-PROFILE-DEFAULTS
Status: active
Topics:
    - javascript
    - profiles
    - inference
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/inference/engine/factory/factory.go
      Note: Provider support and provider-specific validation requirements
    - Path: repo://pkg/js/modules/geppetto/api_agent.go
      Note: Agent builder clone-and-normalize call site
    - Path: repo://pkg/js/modules/geppetto/api_agent_profile_test.go
      Note: Existing sparse-profile JavaScript test to extend into request construction
    - Path: repo://pkg/js/modules/geppetto/api_engine_builder.go
      Note: Engine builder clone-and-normalize call site
    - Path: repo://pkg/js/modules/geppetto/api_engines.go
      Note: Existing JavaScript runtime normalization hook to make provider-aware
    - Path: repo://pkg/steps/ai/settings/settings-inference.go
      Note: Constructors for API, client, and provider settings defaults
ExternalSources: []
Summary: Intern-ready analysis and implementation guide for making omitted provider setting blocks behave as initialized defaults in Geppetto JavaScript inference profiles.
LastUpdated: 2026-07-10T19:04:30.489569035-04:00
WhatFor: Define the failure, runtime invariant, provider matrix, code changes, tests, and review procedure for inference-profile default normalization.
WhenToUse: Read before changing profile resolution, JavaScript agent construction, engine settings validation, or provider-specific settings initialization.
---


# Provider-aware inference-profile default normalization guide

## Executive Summary

Geppetto profile YAML permits omitted sections. This is desirable: a profile should declare its intended model, provider, credentials, and overrides without repeating empty provider objects. The JavaScript API, however, resolves those sparse profiles into `InferenceSettings` values whose omitted pointer fields remain nil. The current agent builder only creates missing `API` settings, while provider execution requires additional objects such as `Client`, `OpenAI`, `Claude`, or `Gemini`.

The result is a delayed runtime failure. A script can resolve a profile and build an agent successfully, then fail only when `session.next().run()` attempts provider request construction. The design changes the existing JavaScript boundary function `ensureInferenceSettingsProviderDefaults` into a provider-aware normalizer that returns errors, materializes only defaultable settings, and leaves all user intent explicit. The change is contained in Geppetto; callers such as the transcript-RAG playground should not need to add `openai: {}` or `client: {}` boilerplate.

## System model

```mermaid
flowchart LR
  A[Profile YAML] --> B[Engine-profile registry resolver]
  B --> C[InferenceSettings wrapper]
  C --> D[JavaScript agent().inference(settings)]
  D --> E[Clone settings]
  E --> F[Provider-aware normalization]
  F --> G[Engine factory]
  G --> H[Agent session]
  H --> I[Request construction]
```

The normalizer belongs at F. At that point Geppetto has an immutable registry-resolved profile wrapper, knows the selected `chat.api_type`, and can safely clone before augmentation. Earlier YAML parsing cannot know which provider-specific defaults matter after profile stacking. Later request construction is too late because error messages lose the configuration boundary and every provider would need duplicated defensive initialization.

## Problem statement

The observed JavaScript path was:

```javascript
const settings = gp.inferenceProfiles.load("profiles.yaml").resolve("assistant");
const agent = gp.agent().inference(settings).build();
const session = agent.session().id("summary").build();
session.next().user("Summarize this text.").run();
```

With a sparse OpenAI-compatible profile, the following failures appeared during execution:

```text
GoError: missing client settings
GoError: no openai settings
```

The immediate cause is visible in provider code:

- classic OpenAI request construction requires non-nil `settings.Client` and `settings.OpenAI`;
- Claude requires non-nil `settings.Client` and `settings.Claude`;
- Gemini's factory requires `settings.Gemini` and runtime construction uses `settings.Client`;
- the JavaScript normalizer currently creates only `settings.API` and a Claude base URL.

The existing JavaScript regression test proves agent/session creation from sparse YAML but does not enter request construction. It therefore misses the failure.

## Required invariant

After the JavaScript agent builder clones registry-resolved settings and before it invokes `enginefactory.NewEngineFromSettings`, the following must hold:

```text
if Chat and Chat.ApiType are present:
    API is non-nil, with non-nil APIKeys, BaseUrls, AllowHTTP, AllowLocalNetworks
    Client is non-nil

    if api type is classic OpenAI-compatible:
        OpenAI is non-nil
    if api type is Claude or anthropic:
        Claude is non-nil
        default claude base URL is supplied only when absent
    if api type is Gemini:
        Gemini is non-nil
```

The normalizer must not create `Chat`, set `Chat.ApiType`, select an engine/model, manufacture API keys, override base URLs, or enable HTTP/local-network access. Those values are profile intent and security policy, not defaults.

## Provider matrix

| API type | Runtime-owned defaults | Explicit profile requirements | Notes |
|---|---|---|---|
| `openai`, `anyscale`, `fireworks` | API, Client, OpenAI | API type, engine, credentials; non-OpenAI compatible endpoints require their base URL | Classic OpenAI request construction directly dereferences Client and OpenAI. |
| `openai-responses`, `open-responses` | API, Client | API type, engine, credentials | No separate `OpenAI` settings pointer is currently required by Responses. |
| `claude`, `anthropic` | API, Client, Claude | API type, engine, credentials | Supply `https://api.anthropic.com` only if `claude-base-url` is absent. |
| `gemini` | API, Client, Gemini | API type, engine, credentials | Factory validates Gemini; runtime uses Client for HTTP transport. |
| `ollama` | none in this ticket | N/A | Standard engine factory does not support this chat provider. Do not mask that error. |

## Proposed implementation

### API change

Change the private helper from:

```go
func ensureInferenceSettingsProviderDefaults(ss *settings.InferenceSettings)
```

to:

```go
func ensureInferenceSettingsProviderDefaults(ss *settings.InferenceSettings) error
```

Provider constructor helpers return errors because they initialize defaults from Glazed schemas. The normalizer must return these errors instead of silently producing partial settings.

### Pseudocode

```text
normalize(settings):
    if settings is nil or chat API type is absent:
        return nil

    initialize API defaults if missing
    initialize Client defaults if missing

    switch chat API type:
      case openai, anyscale, fireworks:
        initialize OpenAI settings if missing
      case claude, anthropic:
        initialize Claude settings if missing
        set default Claude base URL only if absent
      case gemini:
        initialize Gemini settings if missing
      case openai-responses, open-responses:
        no provider settings object required
      default:
        do not claim support; let engine factory reject it

    return nil
```

### Call sites

Both JavaScript builder paths clone a settings wrapper and call the helper:

- `agent().inference(settings).build()` in `api_agent.go`;
- `engine().inference(settings).build()` in `api_engine_builder.go`.

Both must propagate the returned error as their normal Go/JavaScript error. No public JavaScript signature changes.

## Data ownership and mutation rules

`InferenceSettings` wrappers are Go-owned snapshots. The normalizer must operate on a clone, never mutate the profile registry or the wrapper object retained by JavaScript. Each builder already clones settings before creating an engine; retain that property. Explicit profile fields survive unchanged, including supplied `client`, `openai`, `claude`, `gemini`, maps, base URLs, and outbound URL policy maps.

## Tests

### Direct unit tests

Exercise the private normalizer with sparse `InferenceSettings` for each supported type. Assert:

- missing API initializes all API maps;
- missing Client initializes the normal timeout/proxy defaults;
- OpenAI-compatible classic types initialize OpenAI settings but not unrelated provider objects;
- Claude initializes Claude settings and preserves an explicit Claude base URL;
- Gemini initializes Gemini settings;
- Responses does not synthesize a classic OpenAI settings object;
- unknown/Ollama types do not gain false provider support.

### JavaScript regression

Use a temporary profile file containing only API credentials and `chat.api_type: openai`. Resolve it through `gp.inferenceProfiles`, build an agent/session, and reach the deterministic request-construction method without making an HTTP request. The test must prove the generated request has the profile's selected model. It must fail before this change with `missing client settings` or `no openai settings`.

For network isolation, construct the engine/request directly in Go after exercising the JS profile-resolution path, or install a local fake `engine.Engine` factory seam. Do not contact OpenAI, Ollama, or any external service.

## Validation sequence

```text
gofmt -w pkg/js/modules/geppetto/api_engines.go pkg/js/modules/geppetto/api_agent.go pkg/js/modules/geppetto/api_engine_builder.go pkg/js/modules/geppetto/api_agent_profile_test.go
GOWORK=off go test ./pkg/js/modules/geppetto -count=1
GOWORK=off go test ./pkg/inference/engine/factory ./pkg/steps/ai/openai ./pkg/steps/ai/claude ./pkg/steps/ai/gemini -count=1
GOWORK=off go test ./... -count=1
```

Run lint and any repository-specific pre-commit checks after focused tests. Record exact commands and failures in the diary.

## Alternatives rejected

### Require `openai: {}` and `client: {}` in every profile

Rejected. It leaks runtime implementation detail into declarative profile files and contradicts the existing sparse profile examples. It also fails to solve equivalent Claude and Gemini omissions.

### Initialize every provider block unconditionally

Rejected. It obscures provider ownership, may create irrelevant configuration in snapshots, and makes unsupported providers look more complete than they are.

### Normalize during YAML unmarshalling

Rejected. YAML profiles can be stacked and partially overlaid. The builder boundary has the final effective API type and already owns a clone appropriate for mutable engine construction.

### Add fallbacks at each provider request method

Rejected. It duplicates logic across providers, delays errors to execution, and permits divergent defaults between the JavaScript agent and engine builders.

## Review checklist

- The normalizer returns errors and both builder call sites propagate them.
- No resolved profile wrapper or registry object is mutated.
- Explicit configuration wins over defaults.
- Security policy maps remain false/empty unless profile YAML explicitly opts in.
- Claude's default base URL remains conditional.
- OpenAI Responses and unsupported Ollama behavior do not regress.
- Tests reach request construction without network traffic.

## References

- `pkg/js/modules/geppetto/api_engines.go` — existing normalization boundary.
- `pkg/js/modules/geppetto/api_agent.go` and `api_engine_builder.go` — clone-and-build call sites.
- `pkg/steps/ai/settings/settings-inference.go` — constructors for runtime settings objects.
- `pkg/inference/engine/factory/factory.go` — provider support and validation matrix.
- `pkg/steps/ai/openai/helpers.go`, `pkg/steps/ai/claude/helpers.go`, and `pkg/steps/ai/gemini/modern_engine.go` — actual dereferences.

## Problem Statement

<!-- Describe the problem this design addresses -->

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
