---
Title: Host-owned renewable bearer source injection for JavaScript engines
Ticket: GEP-JS-RENEWABLE-BEARER-INJECTION
Status: active
Topics:
    - javascript
    - oauth
    - inference
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/doc/playbooks/08-use-renewable-bearer-credentials.md
      Note: Public host-registration guidance
    - Path: repo://pkg/inference/engine/factory/factory.go
      Note: Existing bearer source factory option and provider propagation
    - Path: repo://pkg/inference/engine/factory/helpers.go
      Note: No-options helper causing the integration gap
    - Path: repo://pkg/js/modules/geppetto/api_engine_builder.go
      Note: JavaScript engine builder calls the no-options helper
    - Path: repo://pkg/js/modules/geppetto/api_engine_builder_test.go
      Note: Behavioral regression tests for source injection
    - Path: repo://pkg/js/modules/geppetto/module.go
      Note: Native module registration options and runtime dependency injection
    - Path: repo://pkg/steps/ai/credentials/bearer.go
      Note: Renewable bearer source contract
ExternalSources: []
Summary: Design and implementation guide for attaching a Go-owned renewable bearer source to engines constructed by the Geppetto JavaScript module.
LastUpdated: 2026-07-13T20:21:51.782709576-04:00
WhatFor: Implement and review host-only bearer source injection for JavaScript-created engines.
WhenToUse: When a Go host embeds the geppetto JavaScript module and needs OAuth-renewed OpenAI-compatible credentials.
---



# Host-owned renewable bearer source injection for JavaScript engines

## Executive summary

Geppetto already supports renewable bearer credentials in Go. A host constructs a `credentials.BearerTokenSource`, passes it through `factory.WithBearerTokenSource`, and the OpenAI and Responses engines resolve the bearer immediately before requests. The source can refresh an expired OAuth access credential, persist host-owned state, and safely retry one rejected request.

The JavaScript module exposes a fluent builder, `require("geppetto").engine().inference(settings).build()`, but its `build()` method currently calls `factory.NewEngineFromSettings(settings)` without factory options. Therefore an engine built by JavaScript cannot use the host's bearer source. The safe fix is deliberately small: add a Go-only `BearerTokenSource` field to the native-module `Options`, copy it to `moduleRuntime`, and construct the engine with `factory.NewStandardEngineFactory(factory.WithBearerTokenSource(source))` when configured. No JavaScript-visible property, function, token string, or refresh callback is added.

This document is written for an intern. It explains the boundary, the data flow, the exact files to change, tests that prove the boundary, and how to validate and review the change.

## Problem statement and scope

A host may embed Geppetto's Goja module to allow scripts to resolve profiles, create an engine, build an agent, and run inference. That host may also own an OAuth login flow and secure credential persistence. Pinocchio is one such host: it stores refreshable OAuth material in a protected, directly writable YAML registry and creates a Go `BearerTokenSource` from that state.

At present the two capabilities cannot meet on the JavaScript engine-building path. The engine builder receives an `InferenceSettings` wrapper, clones and normalizes it, then calls the no-options factory helper. The helper creates a factory with no bearer source, so the OpenAI-compatible validation path requires a static `APIKeys` entry. Adding the OAuth access or refresh material to `InferenceSettings.API.APIKeys` would make the script-visible settings object carry a secret and violates the credential ownership model.

### In scope

- A host supplies one Go `credentials.BearerTokenSource` when registering the Geppetto native module.
- JavaScript-built OpenAI-compatible and Responses engines receive that source.
- A configured source remains authoritative over static API keys, matching the existing factory contract.
- Tests prove that source-enabled JavaScript construction succeeds without a static API key and that JavaScript cannot obtain the source or credential values.
- Public Go documentation describes registration and the security boundary.

### Out of scope

- OAuth login, token refresh protocol, persistence, browser callback handling, or provider-specific policy. Those belong to the host.
- A JavaScript API for bearer strings, refresh callbacks, source names, source selection, or credential persistence.
- Altering static-key behavior when no source is configured.
- Adding bearer-source support for providers that do not consume the OpenAI-compatible source path.

## System orientation

### Terms

- **Host:** the Go application that creates a Goja runtime and registers this native module.
- **JavaScript module:** `require("geppetto")`, implemented by `pkg/js/modules/geppetto`.
- **Inference settings:** provider and model configuration. It is intentionally not the owner of renewable secrets.
- **Bearer token source:** `credentials.BearerTokenSource`, a Go interface that returns a request-time bearer and can invalidate a bearer after a `401` response.
- **Renewable credential:** access state that may be refreshed by a host-provided source. The factory treats it as authoritative over static API-key settings.

### Current build flow (observed)

`pkg/js/modules/geppetto/api_engine_builder.go:26-56` defines the fluent JavaScript builder. `inference()` only accepts a registry-resolved wrapper and clones it. `build()` clones the settings, fills provider defaults, then calls `enginefactory.NewEngineFromSettings(settings)`.

`pkg/inference/engine/factory/helpers.go:9-14` shows that this convenience helper creates `NewStandardEngineFactory()` with no options. In contrast, `pkg/inference/engine/factory/factory.go:82-88` exposes `WithBearerTokenSource`, and `CreateEngine` forwards it to OpenAI Chat at lines 136-141 and OpenAI Responses at lines 143-148. `validateOpenAISettings` deliberately bypasses static-key validation when that source is non-nil (lines 221 onward).

```text
CURRENT: no host-source path

 JavaScript                         native module                    factory
 ──────────                         ─────────────                    ───────
 engine().inference(settings)
          .build() ──────────> clone + normalize ──> NewEngineFromSettings()
                                                              │
                                                              ▼
                                               NewStandardEngineFactory()
                                               bearer source = nil
                                                              │
                                                              ▼
                                                require static API key

The only workaround is an API key in settings, which is not valid for
host-owned renewable OAuth material.
```

### Desired build flow

The host controls module options before JavaScript starts. The source is copied into Go runtime state but is never exported through `installExports`, never attached to a JavaScript object, and never serialized into engine metadata.

```text
DESIRED: Go-owned source crosses only Go calls

 Go host
 ┌─────────────────────────────────────────────────────────────────────┐
 │ source := NewRenewableBearerTokenSource(...)                         │
 │ geppetto.Register(registry, geppetto.Options{                        │
 │     BearerTokenSource: source,                                        │
 │ })                                                                    │
 └───────────────────────────────┬─────────────────────────────────────┘
                                 │ Go interface reference only
                                 ▼
 native module Options ──> moduleRuntime ──> StandardEngineFactory option
                                                    │
 JavaScript                                         ▼
 ┌───────────────────────────────┐       OpenAI-compatible engine
 │ engine().inference(s).build() │       resolves bearer per request
 │ sees only engine wrapper      │       refreshes/invalidate in Go
 └───────────────────────────────┘

No arrow returns a credential, source, or callback to JavaScript.
```

## Security model and invariants

The injection point is a capability boundary, not a configuration convenience. A bearer source can obtain a credential at request time and potentially refresh it. That capability must stay in Go, where the host controls persistence, locks, cancellation, logging, and provider policy.

The implementation must preserve these invariants:

1. **No credentials in JavaScript.** JavaScript values, exports, metadata, errors, and profile data must not contain access credentials, refresh credentials, authorization codes, or client secrets.
2. **No credentials in static settings.** `InferenceSettings.API.APIKeys` remains for static keys only. A source-enabled engine is valid without a static API key.
3. **No change without opt-in.** A nil `Options.BearerTokenSource` follows the current `NewEngineFromSettings` behavior exactly.
4. **One source per native-module registration/runtime.** The initial API is host-wide, which avoids script-controlled source selection and aligns with the existing registration-level configuration pattern.
5. **Factory remains provider-aware.** The JS module does not decide which providers support bearer sources; it delegates to `StandardEngineFactory`, which already attaches sources only to supported OpenAI-compatible engines.
6. **No new JavaScript surface.** `Object.keys(require("geppetto"))` and the TypeScript declaration surface should not gain secret-related methods.

## Proposed architecture and API

### Public Go API

Add one field to `pkg/js/modules/geppetto.Options`:

```go
import "github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"

type Options struct {
    // Existing host-only fields ...
    BearerTokenSource credentials.BearerTokenSource
}
```

The field is intentionally an interface, not a string or callback. It uses the existing credentials package contract instead of defining a second JS-specific abstraction. The Go host is responsible for constructing it.

An embedding host uses it as follows. The source itself is not JavaScript data.

```go
source := /* host-owned credentials.BearerTokenSource */
registry := require.NewRegistry()
geppetto.Register(registry, geppetto.Options{
    BearerTokenSource: source,
    // Existing profile, storage, and runtime options.
})
registry.Enable(vm)
```

### Internal wiring

Add the same private interface field to `moduleRuntime`, set it in `newRuntime`, and use it only in the `build()` closure. A tiny private helper keeps the choice explicit and avoids duplicating construction logic in future builder paths.

```go
func (m *moduleRuntime) newEngineFromSettings(
    settings *aistepssettings.InferenceSettings,
) (engine.Engine, error) {
    if m.bearerTokenSource == nil {
        return enginefactory.NewEngineFromSettings(settings)
    }
    return enginefactory.NewStandardEngineFactory(
        enginefactory.WithBearerTokenSource(m.bearerTokenSource),
    ).CreateEngine(settings)
}
```

`api_engine_builder.go` changes only its final construction call:

```go
eng, err := m.newEngineFromSettings(settings)
```

This preserves factory ownership and makes the no-source path use the existing helper. An alternative is always constructing a standard factory with `WithBearerTokenSource(nil)`; retaining the helper makes compatibility intent obvious and minimizes behavior changes.

### Runtime flow pseudocode

```text
function JSBuild(settingsWrapper):
    require a native InferenceSettings wrapper
    settings = clone(settingsWrapper.settings)
    normalize provider defaults(settings)

    if moduleRuntime.bearerTokenSource is nil:
        engine = NewEngineFromSettings(settings)
    else:
        factory = NewStandardEngineFactory(
            WithBearerTokenSource(moduleRuntime.bearerTokenSource)
        )
        engine = factory.CreateEngine(settings)

    return native JavaScript engine wrapper(engine, safe metadata)
```

At request time, no JavaScript call is involved:

```text
function providerRequest(ctx):
    bearer = source.BearerToken(ctx)        // Go interface
    response = send Authorization bearer     // Go HTTP provider
    if response.status == 401:
        source.Invalidate(ctx, bearer)       // Go interface
        bearer = source.BearerToken(ctx)     // bounded retry, provider-owned
        response = send once more
    return response
```

The exact method names in the second sketch are conceptual; use the existing `credentials.BearerTokenSource` contract and provider implementation instead of recreating refresh logic in the JS package.

## Decision records

### Decision: registration-level Go interface injection

- **Context:** A JavaScript builder needs a renewable credential, but its credential source must not become script data.
- **Options considered:** JavaScript bearer string; JavaScript refresh callback; a source selector string; one Go source in module options; a Go-prebuilt engine only.
- **Decision:** Add one `credentials.BearerTokenSource` to `geppetto.Options`, copied to `moduleRuntime` and used during native engine construction.
- **Rationale:** It reuses the existing generic credential contract and puts authority at the same host-registration layer as registries, middleware, stores, and event sinks.
- **Consequences:** One embedded module runtime has one host-selected source. Multi-tenant source selection needs a later, separately designed host resolver rather than script selection.
- **Status:** accepted.

### Decision: no JavaScript bearer API

- **Context:** It is tempting to add `.bearer(token)` or `.credentialSource(callback)` to the fluent API.
- **Options considered:** expose a token string; permit a JavaScript callback; expose an opaque source handle; expose nothing.
- **Decision:** Expose nothing.
- **Rationale:** Strings can be logged or retained; callbacks mix untrusted script execution with refresh/persistence; opaque handles invite selection and discovery APIs without solving ownership. The module already passes native engine references into JS without exporting their private Go fields.
- **Consequences:** Hosts configure the source before scripts load. Scripts cannot dynamically switch OAuth identities, which is an intentional security constraint.
- **Status:** accepted.

### Decision: reuse StandardEngineFactory rather than modify settings

- **Context:** `NewEngineFromSettings` accepts no options, while the standard factory already understands bearer source precedence and provider routing.
- **Options considered:** add credentials to `InferenceSettings`; extend the helper with a global; construct the factory in the module; create a duplicate JS provider path.
- **Decision:** Construct `NewStandardEngineFactory(WithBearerTokenSource(source))` in the module only when a source exists.
- **Rationale:** The factory already enforces static-key fallback, provider routing, and request-time source injection. It keeps settings secret-free.
- **Consequences:** The JS module imports the credentials package for the interface. This is an internal Go dependency only and adds no JavaScript API.
- **Status:** accepted.

## Implementation plan

### Phase 1: document the boundary

1. Record the current engine-builder call chain and the factory behavior with line references.
2. Write this design and an initial diary entry.
3. Add ticket tasks and relate the builder, options, factory, credentials, and provider files.
4. Commit the ticket documentation separately so a reviewer can approve the security model before code changes.

### Phase 2: add host-only plumbing

1. Import `credentials` in `module.go`.
2. Add the documented `BearerTokenSource` field to public `Options` and private `moduleRuntime`.
3. Copy the field in `newRuntime` without wrapping it in JavaScript values.
4. Add `newEngineFromSettings` in the engine-builder package or adjacent package-local code.
5. Replace the direct helper call in `build()` with that helper.
6. Run `gofmt` and package tests.
7. Commit this narrow wiring change.

### Phase 3: add regression tests

Create a focused test in `pkg/js/modules/geppetto`, using `newJSRuntime` from `module_hardcut_test.go`.

- Create source-enabled OpenAI settings with an empty `APIKeys` map.
- Expose only a native inference-settings wrapper to JavaScript; do not export the source.
- Execute `engine().inference(settings).build()` and assert it succeeds.
- Assert the JavaScript module exports no bearer/source property and the built engine metadata does not contain source material.
- Add a no-source test proving the same missing-key settings fail, preserving static validation.
- Where practical, assert the concrete OpenAI engine uses the provided source by issuing a test-server request and verifying its `Authorization` header inside Go test code. Never include bearer material in failure messages.

### Phase 4: publish and deliver

1. Update the relevant Geppetto help page if the registration API needs operator-facing discoverability.
2. Relate modified code and tests to the design/diary.
3. Run package tests, focused credential tests, lint, and the repository pre-push checks as applicable.
4. Update diary, tasks, and changelog with commit IDs.
5. Run `docmgr doctor` until clean.
6. Upload the ticket bundle to reMarkable with a table of contents.

## Test and validation strategy

### Unit-level assertions

| Scenario | Setup | Expected result |
| --- | --- | --- |
| Host source + OpenAI settings without key | `Options.BearerTokenSource` is non-nil, empty API key map | JS `build()` succeeds |
| No source + same settings | zero-value `Options` | JS `build()` fails validation for missing static key |
| Source is hidden | source-enabled runtime | no module export, builder property, engine property, metadata entry, or error includes the source |
| Source reaches provider | test HTTP server and source | Go-owned HTTP request carries the source's bearer; JavaScript never observes it |
| Existing static configuration | nil source + valid static configuration | existing engine-builder examples and tests still pass |

### Commands

Run focused checks while iterating:

```bash
gofmt -w pkg/js/modules/geppetto/module.go pkg/js/modules/geppetto/api_engine_builder.go
go test ./pkg/js/modules/geppetto -count=1
go test ./pkg/inference/engine/factory ./pkg/steps/ai/credentials -count=1
git diff --check
```

Before pushing, run the repository's configured pre-commit and pre-push checks. Preserve `GOWORK=off` where the repository hook defines it for isolated release/security checks.

## Risks, alternatives, and future work

### Risks

- A future contributor could expose `Options.BearerTokenSource` via metadata or a debug dump. Tests should check the JavaScript-visible surface and review should treat new reflection/serialization code as credential-sensitive.
- One source per module runtime may not support a multi-account product. Do not solve that by making script-selected source names; design a host-owned resolver with authorization boundaries first.
- The source is currently meaningful only for the OpenAI-compatible factory branches. Calling it a universal OAuth feature would be misleading.

### Rejected alternatives

- **Static API-key injection:** violates refresh persistence and secret exposure constraints.
- **JS refresh callback:** permits scripts to handle refresh values and introduces runtime/reentrancy/cancellation hazards.
- **JS source registry:** makes credential identity selectable and discoverable from scripts.
- **Change every factory helper:** broadens public API without need. The JS module needs one controlled construction path.
- **Require hosts to always prebuild engines:** remains a valid workaround, but prevents the declarative JavaScript profile-to-engine flow that the module already offers.

### Open questions

1. If a host later needs multiple identities, should it register separate module runtimes, or add an explicitly authorized Go resolver keyed by a non-secret profile identity?
2. Should the TypeScript declaration test add a permanent negative assertion for credential-related exports, or is the Go public-surface test sufficient?
3. After the generic injection API lands, which host should add the first integration test using a real profile-owned OAuth source? That test must use a fake/local provider and no real credentials.

## File reference map

- `pkg/js/modules/geppetto/module.go:37-59,80-165` — public registration options and runtime initialization; add and copy the source here.
- `pkg/js/modules/geppetto/api_engine_builder.go:10-69` — fluent builder and final engine construction; route through the source-aware helper here.
- `pkg/inference/engine/factory/helpers.go:9-14` — no-option convenience helper that explains the current gap.
- `pkg/inference/engine/factory/factory.go:39-148,195-225` — source option, provider propagation, and static-key bypass behavior.
- `pkg/steps/ai/credentials/bearer.go` — source interface and renewable bearer semantics.
- `pkg/steps/ai/openai/chat_stream.go` and `pkg/steps/ai/openai_responses/provider_settings.go` — provider-side request-time bearer resolution and bounded unauthorized replay.
- `pkg/js/modules/geppetto/module_hardcut_test.go` — Goja test harness for native module tests.
- `pkg/doc/playbooks/08-use-renewable-bearer-credentials.md` — generic credential ownership guidance.

## Review checklist

- [ ] `Options.BearerTokenSource` is a Go interface only; it is not converted to a JavaScript value.
- [ ] Nil source retains `NewEngineFromSettings` behavior.
- [ ] Non-nil source uses `WithBearerTokenSource` and no static key is injected into settings.
- [ ] Tests cover source/no-source behavior and absence from the JS public surface.
- [ ] No tests, fixtures, logs, docs, commit messages, or uploads contain a real credential.
- [ ] Ticket docs, diary, changelog, and reMarkable bundle reflect final commit IDs and validation results.
