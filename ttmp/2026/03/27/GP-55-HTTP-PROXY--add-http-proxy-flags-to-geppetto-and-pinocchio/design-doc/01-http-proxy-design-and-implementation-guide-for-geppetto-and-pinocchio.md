---
Title: HTTP proxy design and implementation guide for Geppetto and Pinocchio
Ticket: GP-55-HTTP-PROXY
Status: active
Topics:
    - geppetto
    - pinocchio
    - glazed
    - config
    - inference
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/cli/bootstrap/engine_settings.go
      Note: |-
        Builds hidden base inference settings and final merged settings used by Pinocchio runtime paths.
        Builds the hidden base inference settings that Pinocchio later merges with profiles
    - Path: geppetto/pkg/inference/engine/factory/factory.go
      Note: |-
        Central engine-construction seam where provider-specific clients are selected.
        Central engine construction seam where provider-specific clients are selected
    - Path: geppetto/pkg/sections/sections.go
      Note: |-
        Registers ai-client in CreateGeppettoSections and in the legacy environment/config middleware whitelist.
        Registers ai-client in CreateGeppettoSections and in the legacy config and environment whitelist
    - Path: geppetto/pkg/steps/ai/gemini/engine_gemini.go
      Note: Gemini-specific transport/auth integration required to use a custom HTTP client safely.
    - Path: geppetto/pkg/steps/ai/settings/flags/client.yaml
      Note: |-
        Declares the Glazed ai-client CLI/config surface that Pinocchio already consumes.
        Declares the Glazed ai-client CLI and config surface that Pinocchio already consumes
    - Path: geppetto/pkg/steps/ai/settings/http_client.go
      Note: Implemented shared proxy-aware HTTP client helper recommended by the design guide.
    - Path: geppetto/pkg/steps/ai/settings/http_client_test.go
      Note: Focused tests for explicit proxy URLs
    - Path: geppetto/pkg/steps/ai/settings/settings-client.go
      Note: |-
        Defines the shared ai-client transport settings struct and is the natural home for proxy fields.
        Defines the shared ai-client transport settings struct and is the natural home for proxy fields
    - Path: pinocchio/cmd/pinocchio/main.go
      Note: |-
        Root Pinocchio CLI wiring for repository-loaded commands and shared Geppetto middlewares.
        Root Pinocchio CLI wiring for loaded commands and shared Geppetto middlewares
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: |-
        Main loaded-command runtime path that resolves final inference settings before starting inference.
        Main Pinocchio runtime path that resolves final inference settings before starting inference
ExternalSources: []
Summary: Detailed intern-facing analysis showing that proxy support belongs in Geppetto's ai-client section, explaining how Pinocchio already inherits that section, and laying out a concrete implementation plan to build proxy-aware HTTP clients and wire them into all provider engines used by Pinocchio.
LastUpdated: 2026-03-27T09:09:15.784134085-04:00
WhatFor: Help an unfamiliar engineer add first-class HTTP proxy flags to Geppetto and Pinocchio without guessing where configuration ownership, profile resolution, and provider HTTP transport wiring currently live.
WhenToUse: Use when adding explicit proxy configuration, transport-level client settings, or other cross-provider HTTP behavior to Geppetto-backed Pinocchio commands.
---



# HTTP proxy design and implementation guide for Geppetto and Pinocchio

## Executive Summary

This ticket answers three concrete questions.

First, the proxy settings should be added to the shared Glazed `ai-client` section, not to `ai-chat`, not to provider-specific sections, and not to profile settings. `ai-client` already owns transport-facing fields such as timeout, organization, and user-agent in [geppetto/pkg/steps/ai/settings/settings-client.go](../../../../../../pkg/steps/ai/settings/settings-client.go) and [geppetto/pkg/steps/ai/settings/flags/client.yaml](../../../../../../pkg/steps/ai/settings/flags/client.yaml). It is already registered in `CreateGeppettoSections()` at [geppetto/pkg/sections/sections.go:63-69](../../../../../../pkg/sections/sections.go) and included in the legacy middleware whitelist at [geppetto/pkg/sections/sections.go:257-281](../../../../../../pkg/sections/sections.go).

Second, Pinocchio already has the configuration plumbing needed to carry these fields from flags and config into runtime settings. The root `pinocchio` command and loaded repository commands use Geppetto's shared sections and middleware at [pinocchio/cmd/pinocchio/main.go:245-277](../../../../../../../pinocchio/cmd/pinocchio/main.go) and [pinocchio/pkg/cmds/cobra.go:12-25](../../../../../../../pinocchio/pkg/cmds/cobra.go). Runtime resolution then produces final merged `InferenceSettings` in [pinocchio/pkg/cmds/cmd.go:225-265](../../../../../../../pinocchio/pkg/cmds/cmd.go) via the Geppetto bootstrap helpers in [geppetto/pkg/cli/bootstrap/engine_settings.go:61-150](../../../../../../pkg/cli/bootstrap/engine_settings.go).

Third, the current gap is not parsing. The current gap is transport construction. The codebase already has `ClientSettings.HTTPClient`, but provider paths mostly do not build or inject a proxy-aware client from it. The old OpenAI chat path creates a client from only API settings at [geppetto/pkg/steps/ai/openai/helpers.go:618-630](../../../../../../pkg/steps/ai/openai/helpers.go). Claude builds a new API client but does not inject `settings.Client.HTTPClient` in the main engine path at [geppetto/pkg/steps/ai/claude/engine_claude.go:72-73](../../../../../../pkg/steps/ai/claude/engine_claude.go). OpenAI Responses still calls `http.DefaultClient` directly at [geppetto/pkg/steps/ai/openai_responses/engine.go:167-183](../../../../../../pkg/steps/ai/openai_responses/engine.go). Gemini uses `genai.NewClient(...)` without `option.WithHTTPClient(...)` at [geppetto/pkg/steps/ai/gemini/engine_gemini.go:108-114](../../../../../../pkg/steps/ai/gemini/engine_gemini.go).

The recommended implementation is:

1. Add explicit proxy fields to `ClientSettings` and `client.yaml`.
2. Create one shared helper that builds a proxy-aware `*http.Client` from `ClientSettings`.
3. Wire that helper into provider engine creation.
4. Let Pinocchio inherit the new flags automatically through existing Geppetto section/bootstrap paths.
5. Keep proxy ownership in app config, env, or CLI flags, not in engine profiles.

## Problem Statement

The user goal is practical and narrow:

- pass proxy flags to `pinocchio`,
- have those flags resolve through the existing Geppetto/Glazed settings system,
- and have the resulting provider HTTP requests actually use the proxy.

Today the codebase is close, but not complete.

The code already has a shared transport-oriented settings surface:

- `ClientSettings` contains timeout, organization, user-agent, and `HTTPClient` in [geppetto/pkg/steps/ai/settings/settings-client.go:15-21](../../../../../../pkg/steps/ai/settings/settings-client.go).
- `CreateGeppettoSections()` always includes the `ai-client` section in [geppetto/pkg/sections/sections.go:116-126](../../../../../../pkg/sections/sections.go).
- `InferenceSettings.UpdateFromParsedValues(...)` always decodes the `ai-client` section into `ss.Client` in [geppetto/pkg/steps/ai/settings/settings-inference.go:290-344](../../../../../../pkg/steps/ai/settings/settings-inference.go).
- Geppetto bootstrap traces already reflect `ClientSettings` fields using reflection in [geppetto/pkg/cli/bootstrap/inference_debug.go:50-58](../../../../../../pkg/cli/bootstrap/inference_debug.go).

So the structural ownership question is mostly already answered by the codebase: cross-provider transport settings belong in `ai-client`.

The missing parts are:

1. there is no proxy field in the shared settings struct or `client.yaml`,
2. there is no shared helper that turns `ClientSettings` into a configured `*http.Client`,
3. provider clients are built inconsistently,
4. some paths already honor custom `HTTPClient`, but only partially,
5. and Pinocchio `web-chat` is intentionally not exposing the full `CreateGeppettoSections()` CLI surface, so its behavior needs to be called out explicitly.

This means a contributor can add a flag and still fail to satisfy the user story if they do not also wire the HTTP client all the way into the provider SDKs.

## Requested Outcome And Non-Goals

The requested outcome for this ticket is:

- a detailed design and implementation guide for explicit proxy support in Geppetto and Pinocchio,
- stored in a docmgr ticket,
- with a clear answer about section ownership,
- and with a file-by-file implementation plan.

The recommended implementation scope for the actual feature should be:

- `pinocchio` root and loaded commands,
- `pinocchio js`,
- provider engines used by the standard engine factory.

The recommended non-goals for the first implementation are:

- changing the profile-registry model to carry proxy config,
- redesigning all transport settings across embeddings and token counting in the same patch,
- widening `cmd/web-chat` to expose all low-level AI flags unless explicitly requested.

## Direct Answer: Which Glazed Section Should Own Proxy?

The proxy belongs in `ai-client`.

### Why `ai-client` is the correct section

`ai-client` is already the transport layer section.

- The struct name is `ClientSettings`, and it already owns timeout, organization, user-agent, and `HTTPClient` in [geppetto/pkg/steps/ai/settings/settings-client.go:15-21](../../../../../../pkg/steps/ai/settings/settings-client.go).
- The YAML/flag definition lives in [geppetto/pkg/steps/ai/settings/flags/client.yaml](../../../../../../pkg/steps/ai/settings/flags/client.yaml).
- The section is initialized from defaults in `CreateGeppettoSections()` at [geppetto/pkg/sections/sections.go:63-69](../../../../../../pkg/sections/sections.go).
- The section is included in hidden bootstrap parsing because `ResolveBaseInferenceSettings(...)` builds from `cfg.BuildBaseSections()` and Pinocchio's bootstrap config points that to `CreateGeppettoSections()` in [pinocchio/pkg/cmds/profilebootstrap/profile_selection.go:18-29](../../../../../../../pinocchio/pkg/cmds/profilebootstrap/profile_selection.go) and [geppetto/pkg/cli/bootstrap/engine_settings.go:31-58](../../../../../../pkg/cli/bootstrap/engine_settings.go).

That is exactly the behavior we want for proxy settings:

- shared across providers,
- available via config/env/flags,
- present in debug traces,
- and merged before engine construction.

### Why not `ai-chat`

`ai-chat` is about model-facing request parameters: engine, provider type, temperature, stop sequences, structured output, and cache settings in [geppetto/pkg/steps/ai/settings/settings-chat.go:22-45](../../../../../../pkg/steps/ai/settings/settings-chat.go). Proxy settings are not chat semantics. They are transport semantics.

### Why not provider-specific sections

Provider-specific sections such as `openai-chat`, `claude-chat`, and `gemini-chat` are for provider-native request payload parameters, not for shared TCP/HTTP transport concerns. A proxy is not OpenAI-specific or Claude-specific. Putting it into provider sections would duplicate config and make multi-provider commands harder to reason about.

### Why not `profile-settings` or engine profiles

This repo's newer documentation explicitly draws a boundary: profiles should select behavior, not infrastructure. See [geppetto/pkg/doc/tutorials/08-build-streaming-tool-loop-agent-with-glazed-flags.md:443-449](../../../../../../pkg/doc/tutorials/08-build-streaming-tool-loop-agent-with-glazed-flags.md). A proxy is infrastructure. It belongs in base app config, env, or flags, not in runtime profile metadata.

That point matters here because `MergeInferenceSettings(...)` in [geppetto/pkg/engineprofiles/inference_settings_merge.go:20-49](../../../../../../pkg/engineprofiles/inference_settings_merge.go) would technically merge any new `client.*` fields if they appear in profile inference settings. The ticket should not encourage that usage. The intended operator model should stay:

- profiles select model/runtime behavior,
- app config and CLI own transport and credentials.

## Current-State Architecture

This section maps the actual runtime path an intern needs to understand before editing code.

### High-level data flow

```text
Pinocchio Cobra command
  -> Geppetto Glazed sections mounted
  -> flags/env/config parsed into Values
  -> Geppetto bootstrap builds hidden base InferenceSettings
  -> Pinocchio resolves profile selection and merges profile overlay
  -> engine factory selects provider engine
  -> provider engine builds SDK/client
  -> SDK or raw HTTP transport performs request
```

The proxy feature only works if all layers in that chain agree about ownership.

### Layer 1: shared section registration

`CreateGeppettoSections()` is the source of truth for the reusable full-flags AI surface. It creates and returns:

- `ai-chat`,
- `ai-client`,
- provider sections,
- embeddings,
- `ai-inference`,
- `profile-settings`.

That happens in [geppetto/pkg/sections/sections.go:35-127](../../../../../../pkg/sections/sections.go).

This is why `ai-client` is the right place to add proxy flags: any Pinocchio command already reusing this helper will inherit them automatically.

### Layer 2: Pinocchio command registration

The root `pinocchio` binary uses the Geppetto middleware path when loading repository commands and when building some built-in commands:

- loaded repository commands in [pinocchio/cmd/pinocchio/main.go:245-252](../../../../../../../pinocchio/cmd/pinocchio/main.go),
- clip command in [pinocchio/cmd/pinocchio/main.go:271-277](../../../../../../../pinocchio/cmd/pinocchio/main.go),
- shared helper in [pinocchio/pkg/cmds/cobra.go:12-25](../../../../../../../pinocchio/pkg/cmds/cobra.go).

Loaded YAML commands also bake Geppetto sections into their command descriptions at [pinocchio/pkg/cmds/loader.go:65-85](../../../../../../../pinocchio/pkg/cmds/loader.go).

That means the standard `pinocchio` surfaces already have a strong inheritance story for new `ai-client` flags.

### Layer 3: hidden base settings and final merged settings

The Geppetto bootstrap package reconstructs hidden base settings by rebuilding the shared sections and parsing config/env/defaults:

- [geppetto/pkg/cli/bootstrap/engine_settings.go:26-58](../../../../../../pkg/cli/bootstrap/engine_settings.go)

Pinocchio then resolves final settings and runtime profile selection through:

- [pinocchio/pkg/cmds/profilebootstrap/profile_selection.go:18-29](../../../../../../../pinocchio/pkg/cmds/profilebootstrap/profile_selection.go)
- [pinocchio/pkg/cmds/profilebootstrap/engine_settings.go:19-33](../../../../../../../pinocchio/pkg/cmds/profilebootstrap/engine_settings.go)
- [pinocchio/pkg/cmds/cmd.go:235-265](../../../../../../../pinocchio/pkg/cmds/cmd.go)

There is a second important Pinocchio path for interactive profile switching. `baseSettingsFromParsedValuesWithBase(...)` strips only profile-derived parse steps, then re-decodes the remaining shared sections, including `ai-client`, into a fresh `InferenceSettings` in [pinocchio/pkg/cmds/profile_base_settings.go:21-89](../../../../../../../pinocchio/pkg/cmds/profile_base_settings.go). That means a proxy field in `ai-client` will also behave correctly for profile switching flows.

### Layer 4: engine selection

The standard factory chooses the provider in [geppetto/pkg/inference/engine/factory/factory.go:47-93](../../../../../../pkg/inference/engine/factory/factory.go). This is the narrowest common engine-construction seam shared by Pinocchio runtime flows.

If an engineer wants proxy support to work for "Pinocchio uses provider X", this factory path is the seam they must understand.

### Layer 5: provider client construction

This is the current inconsistency zone.

#### OpenAI chat completions

`OpenAIEngine.RunInference(...)` builds a client with `MakeClient(e.settings.API, ...)` at [geppetto/pkg/steps/ai/openai/engine_openai.go:55-58](../../../../../../pkg/steps/ai/openai/engine_openai.go). `MakeClient(...)` only receives `APISettings`, then constructs `go_openai.DefaultConfig(apiKey)` and `go_openai.NewClientWithConfig(config)` in [geppetto/pkg/steps/ai/openai/helpers.go:618-630](../../../../../../pkg/steps/ai/openai/helpers.go).

That signature is a design smell for this feature: it has no access to `ClientSettings`, so it cannot apply explicit proxy configuration.

#### Claude

`ClaudeEngine.RunInference(...)` creates `api.NewClient(apiKey, baseURL)` at [geppetto/pkg/steps/ai/claude/engine_claude.go:72](../../../../../../pkg/steps/ai/claude/engine_claude.go). The underlying Claude client owns its own `*http.Client` in [geppetto/pkg/steps/ai/claude/api/completion.go:51-80](../../../../../../pkg/steps/ai/claude/api/completion.go) and exposes `SetHTTPClient(...)`.

That is good news: the library already has the injection seam we need. The problem is that the main engine path currently does not use it.

#### OpenAI Responses

The Responses engine uses raw `http.NewRequestWithContext(...)` and `http.DefaultClient.Do(req)` in [geppetto/pkg/steps/ai/openai_responses/engine.go:167-183](../../../../../../pkg/steps/ai/openai_responses/engine.go) and later again in the non-stream path near [geppetto/pkg/steps/ai/openai_responses/engine.go:849](../../../../../../pkg/steps/ai/openai_responses/engine.go). That means there is no explicit runtime-controlled HTTP client at all today.

#### Gemini

Gemini uses Google's SDK:

- `genai.NewClient(ctx, option.WithAPIKey(apiKey), option.WithEndpoint(baseURL))` when base URL is set,
- otherwise `genai.NewClient(ctx, option.WithAPIKey(apiKey))`,

in [geppetto/pkg/steps/ai/gemini/engine_gemini.go:108-114](../../../../../../pkg/steps/ai/gemini/engine_gemini.go).

The local module version in this workspace supports `option.WithHTTPClient(...)`; see `go doc google.golang.org/api/option.WithHTTPClient`. So the injection seam exists, but the engine does not currently use it.

### Current implicit proxy behavior

A subtle but important observation: many current paths are already proxy-aware through Go's default transport, but only through ambient environment variables.

`go doc net/http.DefaultTransport` shows that the default transport uses `ProxyFromEnvironment`, and `go doc net/http.ProxyFromEnvironment` confirms that it reads `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY`.

That means:

- bare `&http.Client{}` instances usually inherit environment-based proxy behavior,
- `http.DefaultClient` also inherits environment-based proxy behavior,
- but there is still no first-class Pinocchio/Geppetto CLI/config surface for explicit proxy configuration,
- and there is no provenance/debug visibility for ambient process-wide proxy env.

This ticket should preserve current env behavior by default, while adding explicit CLI/config ownership.

### Existing evidence that custom `HTTPClient` is already the intended abstraction

Two token-count code paths already use `settings.Client.HTTPClient` if present:

- Claude token counting in [geppetto/pkg/steps/ai/claude/token_count.go:51-55](../../../../../../pkg/steps/ai/claude/token_count.go)
- OpenAI Responses token counting in [geppetto/pkg/steps/ai/openai_responses/token_count.go:96-101](../../../../../../pkg/steps/ai/openai_responses/token_count.go)

This is strong evidence that the intended abstraction already exists. The main inference engines simply have not been normalized onto it yet.

## Special Case: `cmd/web-chat`

`cmd/web-chat` is intentionally not a full-flags Geppetto command.

Its command description mounts only:

- `profile-settings`,
- `redis` settings,

at [pinocchio/cmd/web-chat/main.go:75-103](../../../../../../../pinocchio/cmd/web-chat/main.go).

It still resolves hidden base inference settings through `profilebootstrap.ResolveBaseInferenceSettings(parsed)` at [pinocchio/cmd/web-chat/main.go:173-180](../../../../../../../pinocchio/cmd/web-chat/main.go), so config/env-based `ai-client` fields would still matter behind the scenes. But explicit CLI proxy flags would not automatically appear on `web-chat` because it does not mount `CreateGeppettoSections()`.

This is not a bug. It is a product decision.

Recommendation for this ticket:

- do not widen `web-chat` to expose low-level `ai-client` flags unless the user explicitly asks for that command too,
- document this exception clearly,
- keep the first implementation focused on the standard `pinocchio` CLI runtime path.

## Proposed Solution

The implementation should add first-class proxy configuration to `ClientSettings`, then standardize provider client creation around a shared proxy-aware `*http.Client`.

### Recommended user-facing settings

I recommend adding exactly these new fields first:

```go
type ClientSettings struct {
    Timeout              *time.Duration `yaml:"timeout,omitempty"`
    TimeoutSeconds       *int           `yaml:"timeout_second,omitempty" glazed:"timeout"`
    Organization         *string        `yaml:"organization,omitempty" glazed:"organization"`
    UserAgent            *string        `yaml:"user_agent,omitempty" glazed:"user-agent"`
    ProxyURL             *string        `yaml:"proxy_url,omitempty" glazed:"proxy-url"`
    ProxyFromEnvironment *bool          `yaml:"proxy_from_environment,omitempty" glazed:"proxy-from-environment"`
    HTTPClient           *http.Client   `yaml:"-" json:"-"`
}
```

Why this shape:

- `proxy-url` gives the user a direct explicit proxy flag.
- `proxy-from-environment` preserves current env-driven behavior by default, but gives a way to force direct connections when needed.
- this keeps the first implementation small and understandable.

I do not recommend adding `no-proxy` in the first cut. Go already has mature `NO_PROXY` semantics in `ProxyFromEnvironment`. Reimplementing that matching logic for explicit flags makes the first patch larger and riskier. If operator demand appears later, it can be a follow-up ticket.

### Recommended default behavior

The safest behavior is:

1. if `ProxyURL` is set, use it,
2. else if `ProxyFromEnvironment` is unset or true, use environment proxy behavior,
3. else make direct connections.

That preserves today's ambient behavior while enabling explicit Pinocchio-owned configuration.

### Recommended helper boundary

Add one shared helper in Geppetto, close to `ClientSettings`, for example:

- `geppetto/pkg/steps/ai/settings/http_client.go`

The helper should:

1. clone the default transport,
2. preserve the standard library defaults, including HTTP/2 and idle connection behavior,
3. set `Transport.Proxy` according to `ClientSettings`,
4. apply timeout from `ClientSettings`,
5. return a ready `*http.Client`,
6. cache the client on `ClientSettings.HTTPClient` when appropriate.

Pseudocode:

```go
func EnsureHTTPClient(cs *ClientSettings) (*http.Client, error) {
    if cs == nil {
        return http.DefaultClient, nil
    }
    if cs.HTTPClient != nil {
        return cs.HTTPClient, nil
    }

    baseTransport, ok := http.DefaultTransport.(*http.Transport)
    if !ok {
        return nil, fmt.Errorf("default transport is %T, expected *http.Transport", http.DefaultTransport)
    }
    tr := baseTransport.Clone()

    useEnv := true
    if cs.ProxyFromEnvironment != nil {
        useEnv = *cs.ProxyFromEnvironment
    }

    switch {
    case cs.ProxyURL != nil && strings.TrimSpace(*cs.ProxyURL) != "":
        u, err := url.Parse(strings.TrimSpace(*cs.ProxyURL))
        if err != nil {
            return nil, err
        }
        tr.Proxy = http.ProxyURL(u)
    case useEnv:
        tr.Proxy = http.ProxyFromEnvironment
    default:
        tr.Proxy = nil
    }

    client := &http.Client{Transport: tr}
    if cs.Timeout != nil {
        client.Timeout = *cs.Timeout
    } else if cs.TimeoutSeconds != nil {
        client.Timeout = time.Duration(*cs.TimeoutSeconds) * time.Second
    }

    cs.HTTPClient = client
    return client, nil
}
```

This helper should validate and normalize the proxy URL once instead of letting each provider do it differently.

### Recommended provider wiring matrix

| Provider path | Current state | Required change |
|---|---|---|
| OpenAI chat completions | `MakeClient(...)` only sees `APISettings` | change the signature to accept `ClientSettings` or `InferenceSettings`, then set `config.HTTPClient` before `NewClientWithConfig(...)` |
| OpenAI Responses | raw `http.DefaultClient` | replace with shared ensured client |
| Claude | client already exposes `SetHTTPClient(...)` | call `SetHTTPClient(ensuredClient)` in main engine path |
| Gemini | SDK supports `option.WithHTTPClient(...)` | pass `option.WithHTTPClient(ensuredClient)` into `genai.NewClient(...)` |

### Recommended Pinocchio behavior after implementation

After the feature is implemented, the happy path should look like this:

```text
pinocchio some-command --proxy-url http://proxy.internal:8080
  -> proxy-url parsed into ai-client section
  -> ai-client decoded into ClientSettings
  -> final InferenceSettings built by Pinocchio bootstrap
  -> engine factory constructs provider engine
  -> provider engine asks settings helper for effective HTTP client
  -> outgoing requests use proxy.internal:8080
```

The same should also work from config:

```yaml
ai-client:
  timeout: 120
  proxy_url: http://proxy.internal:8080
  proxy_from_environment: false
```

And, by naming convention already used elsewhere in Pinocchio, the expected environment variable would be `PINOCCHIO_PROXY_URL`.

## Design Decisions

### Decision 1: put proxy in `ai-client`

This is the core decision. It matches current code ownership and avoids duplication.

### Decision 2: do not put proxy in profiles

This respects the current architecture direction that profiles choose runtime behavior, not transport infrastructure.

### Decision 3: preserve environment proxy behavior by default

This avoids breaking existing users who already depend on `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`.

### Decision 4: add one shared HTTP client helper instead of ad hoc provider patches

Without a shared helper, each provider will re-implement:

- timeout behavior,
- proxy precedence,
- URL parsing,
- future TLS or transport tuning.

That would create exactly the kind of drift that made this ticket necessary.

### Decision 5: keep `cmd/web-chat` out of first implementation scope

`web-chat` intentionally exposes a smaller command surface. Expanding it should be a conscious product/API decision, not collateral damage from a Pinocchio CLI transport ticket.

## Implementation Plan

This section is written as a practical checklist for a new intern.

### Phase 1: extend shared settings and Glazed surface

Files:

- `geppetto/pkg/steps/ai/settings/settings-client.go`
- `geppetto/pkg/steps/ai/settings/flags/client.yaml`
- `geppetto/pkg/steps/ai/settings/settings-inference.go`

Tasks:

1. Add `ProxyURL` and `ProxyFromEnvironment` to `ClientSettings`.
2. Extend `NewClientSettings()` defaults so existing behavior remains unchanged.
   - Recommendation: leave `ProxyURL` nil.
   - Recommendation: default `ProxyFromEnvironment` to `true`.
3. Add `proxy-url` and `proxy-from-environment` to `client.yaml`.
4. Update `GetMetadata()` and `GetSummary(verbose)` so proxy settings appear in debugging output.
   - Avoid printing secrets if credentials are ever embedded in proxy URLs.
   - If proxy URLs may contain credentials, add a redaction helper before printing them.

### Phase 2: add shared HTTP client construction helper

Files:

- new file in `geppetto/pkg/steps/ai/settings/`, for example `http_client.go`
- possible tests in the same package

Tasks:

1. Implement `EnsureHTTPClient(...)`.
2. Clone `http.DefaultTransport` instead of creating a zero-value transport.
3. Parse and validate `ProxyURL`.
4. Preserve timeout semantics.
5. Add unit tests for:
   - explicit proxy URL,
   - env-proxy fallback,
   - env-proxy disabled,
   - invalid proxy URL.

### Phase 3: wire provider engines

Files:

- `geppetto/pkg/steps/ai/openai/helpers.go`
- `geppetto/pkg/steps/ai/openai/engine_openai.go`
- `geppetto/pkg/steps/ai/claude/engine_claude.go`
- `geppetto/pkg/steps/ai/openai_responses/engine.go`
- `geppetto/pkg/steps/ai/gemini/engine_gemini.go`

Tasks:

1. OpenAI chat completions:
   - change `MakeClient(...)` so it can see `ClientSettings`,
   - call the shared helper,
   - assign the resulting client to `go_openai.ClientConfig.HTTPClient`.

2. Claude:
   - call shared helper,
   - inject via `client.SetHTTPClient(...)`.

3. OpenAI Responses:
   - call shared helper once per engine request path,
   - replace direct `http.DefaultClient.Do(req)` calls with the ensured client.

4. Gemini:
   - call shared helper,
   - pass `option.WithHTTPClient(client)` into `genai.NewClient(...)`.

### Phase 4: verify Pinocchio inheritance paths

Files:

- `pinocchio/pkg/cmds/cobra.go`
- `pinocchio/pkg/cmds/cmd.go`
- `pinocchio/pkg/cmds/loader.go`
- `pinocchio/cmd/pinocchio/main.go`
- `pinocchio/cmd/pinocchio/cmds/js.go`

Tasks:

1. Do not add custom Pinocchio-only proxy parsing unless testing reveals a genuine gap.
2. Verify that the existing inheritance path already exposes the new fields on:
   - loaded YAML commands,
   - repository commands,
   - `pinocchio js`.
3. Add a small regression test or smoke-style test that proves `PINOCCHIO_PROXY_URL` or `--proxy-url` reaches final `InferenceSettings.Client`.

The likely result is that no Pinocchio runtime code changes are needed here beyond tests and possibly documentation.

### Phase 5: explicitly document `web-chat` behavior

Files:

- `pinocchio/cmd/web-chat/README.md` if the feature is extended there,
- otherwise only this ticket doc.

Tasks:

1. Decide whether `web-chat` should gain explicit proxy flags now.
2. If no:
   - document that `web-chat` can still consume proxy config through hidden base config/env resolution,
   - but does not expose `--proxy-url` on its CLI because it does not mount `ai-client`.
3. If yes:
   - mount `ai-client` explicitly or reuse a small shared transport section,
   - accept that this widens the public `web-chat` CLI.

Recommendation: no for this ticket.

### Phase 6: optional consistency cleanup

This is not required for the user story, but it will make the system cleaner.

Candidate files:

- `geppetto/pkg/steps/ai/claude/token_count.go`
- `geppetto/pkg/steps/ai/openai_responses/token_count.go`
- `geppetto/pkg/embeddings/openai.go`
- `geppetto/pkg/embeddings/ollama.go`
- `geppetto/pkg/embeddings/settings_factory.go`
- `geppetto/pkg/js/modules/geppetto/api_engines.go`

Reason:

- token count paths already partially use `HTTPClient`,
- embeddings and JS engine creation should ideally converge on the same helper over time,
- otherwise users will get "proxy works for inference but not for embeddings" surprises.

## Testing Strategy

### Unit tests

Add focused tests around the new helper.

Recommended cases:

1. `ProxyURL` set, env disabled:
   - helper returns client with proxy function pointing at explicit URL.
2. `ProxyURL` unset, env enabled:
   - helper leaves proxy behavior equivalent to `http.ProxyFromEnvironment`.
3. `ProxyURL` unset, env disabled:
   - helper returns direct transport with no proxy function.
4. invalid proxy URL:
   - helper returns a clear error.

### Provider-level tests

Add or extend tests that capture actual outbound host/transport behavior.

Recommended approach:

1. Create a test HTTP client with a custom `RoundTripper`.
2. Feed it through the new helper or directly via `ClientSettings.HTTPClient`.
3. Assert the engine path uses that client rather than `http.DefaultClient`.

Priority provider tests:

- OpenAI chat completions
- Claude main engine path
- OpenAI Responses streaming path
- Gemini client creation path

### Pinocchio integration tests

Recommended tests:

1. Parse a `pinocchio` command with `--proxy-url`.
2. Resolve final engine settings through `profilebootstrap.ResolveCLIEngineSettings(...)`.
3. Assert `resolved.FinalInferenceSettings.Client.ProxyURL` is set.

If feasible, add one higher-level test that proves an engine built from those settings uses the injected client.

## API References

These are the external API seams that matter for implementation.

### Go standard library

- `net/http.DefaultTransport`
  - local `go doc` confirms it uses `ProxyFromEnvironment`.
- `net/http.ProxyFromEnvironment`
  - local `go doc` confirms support for `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY`.
- `(*http.Transport).Clone()`
  - use this to preserve standard defaults.

### OpenAI Go client

Local module version: `github.com/sashabaranov/go-openai v1.41.1`

Relevant seam:

```go
type ClientConfig struct {
    BaseURL    string
    OrgID      string
    HTTPClient HTTPDoer
}
```

This appears in the local module cache file `github.com/sashabaranov/go-openai@v1.41.1/config.go`.

### Google API option package / Gemini

Local module versions:

- `github.com/google/generative-ai-go v0.20.1`
- `google.golang.org/api v0.272.0`

Relevant seam:

```go
option.WithHTTPClient(client *http.Client)
```

Local `go doc` confirms this is the supported way to inject a custom HTTP client.

## Alternatives Considered

### Alternative A: add proxy flags to provider sections

Rejected.

This duplicates cross-provider transport config and invites drift between OpenAI, Claude, and Gemini flags.

### Alternative B: rely only on `HTTP_PROXY` / `HTTPS_PROXY`

Rejected for the main user story.

This already works in many cases because Go's default transport honors environment proxies, but it does not satisfy the user requirement for explicit Pinocchio flags and it provides poor provenance/debug visibility.

### Alternative C: put proxy config in engine profiles

Rejected.

That crosses the architecture boundary described in current docs: profiles are for behavior/runtime metadata, not operator infrastructure.

### Alternative D: patch only one provider

Rejected.

The user asked for proxy flags to Pinocchio. Pinocchio is multi-provider. A one-provider patch would create misleading behavior and support debt.

## Risks And Sharp Edges

### Risk 1: proxy URLs may contain credentials

If proxy URLs include `user:pass@host`, then metadata and debug summaries must redact them before printing.

### Risk 2: `web-chat` expectations

Some readers may assume every Pinocchio command will gain the new flag automatically. That is not true for `cmd/web-chat` because it intentionally does not mount `ai-client`.

### Risk 3: direct engine construction paths outside the factory

Some examples or helper code may instantiate engines directly instead of using the standard factory. If those paths matter to users, they may need follow-up normalization.

### Risk 4: embeddings inconsistency

If this ticket only fixes inference engines, embeddings may still bypass explicit proxy settings. That should be documented as out of scope or included as a phase-2 cleanup.

## Open Questions

1. Should the first implementation expose only `--proxy-url`, or also `--proxy-from-environment`?
   - Recommendation: expose both.
2. Should `cmd/web-chat` also expose explicit proxy CLI flags?
   - Recommendation: not in the first patch.
3. Should profiles be validated to reject future `client.proxy_url` entries?
   - Recommendation: not required for the first patch, but worth considering later if operator misuse appears.

## References

Key repository references:

- `geppetto/pkg/steps/ai/settings/settings-client.go`
- `geppetto/pkg/steps/ai/settings/flags/client.yaml`
- `geppetto/pkg/sections/sections.go`
- `geppetto/pkg/cli/bootstrap/engine_settings.go`
- `geppetto/pkg/inference/engine/factory/factory.go`
- `geppetto/pkg/steps/ai/openai/helpers.go`
- `geppetto/pkg/steps/ai/openai/engine_openai.go`
- `geppetto/pkg/steps/ai/claude/engine_claude.go`
- `geppetto/pkg/steps/ai/openai_responses/engine.go`
- `geppetto/pkg/steps/ai/gemini/engine_gemini.go`
- `pinocchio/pkg/cmds/cmd.go`
- `pinocchio/pkg/cmds/loader.go`
- `pinocchio/pkg/cmds/profile_base_settings.go`
- `pinocchio/pkg/cmds/profilebootstrap/profile_selection.go`
- `pinocchio/cmd/pinocchio/main.go`
- `pinocchio/cmd/web-chat/main.go`
