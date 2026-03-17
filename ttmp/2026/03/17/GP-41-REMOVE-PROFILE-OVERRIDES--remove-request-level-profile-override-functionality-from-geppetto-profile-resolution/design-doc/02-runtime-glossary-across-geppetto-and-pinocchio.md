---
Title: Runtime glossary across Geppetto and Pinocchio
Ticket: GP-41-REMOVE-PROFILE-OVERRIDES
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - pinocchio
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/types.go
      Note: Defines RuntimeSpec, the main profile-level meaning of runtime in Geppetto
    - Path: geppetto/pkg/profiles/registry.go
      Note: Defines ResolveInput, EffectiveRuntime, RuntimeKeyFallback, and RuntimeFingerprint
    - Path: geppetto/pkg/profiles/service.go
      Note: Shows how runtime resolution actually happens
    - Path: pinocchio/pkg/inference/runtime/composer.go
      Note: Defines the app-owned conversation runtime composition contract
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Concrete example of converting a resolved profile runtime into a runnable engine
    - Path: geppetto/pkg/js/runtime/runtime.go
      Note: Defines Geppetto's JavaScript runtime bootstrap
    - Path: geppetto/pkg/inference/tools/scopedjs/runtime.go
      Note: Defines the scoped JavaScript runtime builder used by scoped tools
    - Path: pinocchio/pkg/webchat/timeline_js_runtime.go
      Note: Defines the timeline JavaScript runtime for SEM projection
    - Path: pinocchio/pkg/webchat/timeline_registry.go
      Note: Defines the timeline runtime bridge API
    - Path: geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go
      Note: Defines how runtime attribution is copied into event metadata
    - Path: geppetto/pkg/turns/keys_gen.go
      Note: Defines KeyTurnMetaRuntime
ExternalSources: []
Summary: Detailed glossary of the major Runtime-named symbols across Geppetto and Pinocchio, with subsystem grouping, rationale, diagrams, and example flows.
LastUpdated: 2026-03-17T15:05:00-04:00
WhatFor: Use this document to disambiguate the many different meanings of runtime across Geppetto and Pinocchio before changing profile resolution or conversation composition code.
WhenToUse: Use when reading runtime-related code, reviewing GP-41 and GP-40 changes, onboarding a new engineer, or trying to understand which Runtime* names belong to profiles, JS execution, timeline projection, or persistence.
---

# Runtime glossary across Geppetto and Pinocchio

## Executive Summary

`Runtime` does not mean one thing in this codebase. It means at least four different things:

1. A resolved profile preset in Geppetto, such as `profiles.RuntimeSpec`.
2. An app-owned composed conversation runtime in Pinocchio, such as `ConversationRuntimeRequest` and `ComposedRuntime`.
3. A JavaScript execution environment, such as `js/runtime.NewRuntime`, `scopedjs.BuildRuntime`, and `JSTimelineRuntime`.
4. Runtime attribution metadata written onto turns and copied into provider event payloads, such as `KeyTurnMetaRuntime` and `AddRuntimeAttributionToExtra`.

That overloading is one reason GP-41 matters. A new engineer can read `RuntimeSpec`, `ComposedRuntime`, `moduleRuntime`, and `JSTimelineRuntime` in the same day and mistakenly assume they all describe the same layer. They do not.

This glossary groups the Runtime-named symbols by subsystem, explains what each one actually represents, and gives concrete examples of how the names interact in real flows. The main recommendation is simple: when you see `Runtime*`, first ask which layer you are in before you read the code any further.

## Problem Statement

GP-41 is about removing request-level profile overrides from Geppetto. That work is easy to get wrong if the reader does not understand the current naming landscape. The term `runtime` appears in:

- profile storage and resolution,
- app-level engine composition,
- JavaScript VM ownership,
- timeline SEM projection,
- turn metadata and persistence.

Without a glossary, an intern will usually make one of these mistakes:

- assume `RuntimeSpec` is a concrete runnable engine,
- assume `RuntimeFingerprint` is the same thing as `RuntimeKey`,
- assume Pinocchio's `ComposedRuntime` lives in Geppetto core,
- assume `NewRuntime` and `BuildRuntime` are profile APIs rather than JS APIs,
- assume `KeyTurnMetaRuntime` is the same thing as profile resolution.

The goal of this document is to remove that ambiguity before implementation work begins.

## Proposed Solution

The solution is not a code change by itself. The solution is a shared vocabulary.

This glossary uses one rule:

- every Runtime-named symbol must be explained in terms of its layer, its owner, its inputs, its outputs, and the file where it is defined.

It also uses one practical reading strategy:

```text
If the symbol contains Runtime:
  ask "profile runtime, composed app runtime, JS runtime, or metadata runtime?"
  then read only the files in that layer first
```

That sounds simple, but it prevents a lot of wasted time and bad refactors.

## Design Decisions

### Decision 1: group by subsystem, not alphabetically

An alphabetical glossary is worse for this ticket because it hides the architectural boundaries. A reader needs to know that `RuntimeSpec`, `ComposedRuntime`, and `JSTimelineRuntime` belong to different subsystems.

### Decision 2: include both current and awkward legacy-adjacent names

Some runtime names are clean. Others reflect migration history or duplicated app surfaces. This glossary includes both, because the confusing names are exactly the ones an intern will hit during GP-41.

### Decision 3: include example flows, not just definitions

Definitions alone are too abstract. The examples show where symbols actually meet:

- resolved profile runtime into Pinocchio conversation composition,
- JS timeline runtime into SEM handling,
- turn runtime metadata into provider event attribution.

### Decision 4: treat this as an architectural support doc for GP-41

This document is not trying to redesign all runtime naming at once. It exists to support the profile override cleanup by making the existing system legible.

## Glossary

## 1. Profile runtime in Geppetto

This is the most important meaning of `runtime` for GP-41.

At this layer, runtime means: "the runtime-facing part of a profile definition or the resolved runtime payload produced from profile resolution."

### `RuntimeSpec`

- Defined in `geppetto/pkg/profiles/types.go:13-19`.
- Owner: Geppetto profiles package.
- Represents: the runtime portion of a profile record.
- Contains:
  - `StepSettingsPatch`
  - `SystemPrompt`
  - `Middlewares`
  - `Tools`
- It is a data structure, not a runnable engine.

Think of `RuntimeSpec` as the profile's answer to the question "if this profile is selected, what runtime defaults should downstream code use?"

### `RuntimeKey`

- Defined in `geppetto/pkg/profiles/slugs.go:18`.
- Parsed by `ParseRuntimeKey` in `geppetto/pkg/profiles/slugs.go:36-42`.
- Built with `MustRuntimeKey` in `geppetto/pkg/profiles/slugs.go:60-66`.
- Represents: the externally visible stable key used by applications when they need a runtime label.

This is usually a compact identifier such as `default`, `agent`, or `planner`. It is closer to a selected runtime slot name than to a fingerprint.

### `RuntimeKeyFallback`

- Field on `ResolveInput` in `geppetto/pkg/profiles/registry.go:35-42`.
- Used in `ResolveEffectiveProfile` in `geppetto/pkg/profiles/service.go:165-171`.
- Represents: the app-supplied runtime key to use if the resolver should not default to the profile slug.

This field exists because applications sometimes want the resolved runtime to keep a stable app-facing name even if the profile slug is not the desired UI/runtime label.

### `ResolveInput`

- Defined in `geppetto/pkg/profiles/registry.go:35-42`.
- Represents: all inputs required to resolve an effective profile.
- Relevant runtime fields:
  - `RuntimeKeyFallback`
  - `BaseStepSettings`
  - `RequestOverrides`

For GP-41, this is the main request shape being simplified. `RequestOverrides` is the part slated for removal.

### `ResolveEffectiveProfile`

- Implemented in `geppetto/pkg/profiles/service.go:128-190`.
- Represents: the canonical resolver that expands the stack, merges policy, resolves runtime data, applies step settings patches, and produces the final resolved profile output.

This is the core runtime resolution path. If you want to know what "effective runtime" means in Geppetto, read this function first.

### `EffectiveRuntime`

- Field on `ResolvedProfile` in `geppetto/pkg/profiles/registry.go:45-55`.
- Produced in `geppetto/pkg/profiles/service.go:182-189`.
- Type: `RuntimeSpec`.
- Represents: the final resolved runtime payload after stack expansion, policy handling, and override application.

Important distinction:

- `Profile.Runtime` is stored profile data.
- `ResolvedProfile.EffectiveRuntime` is post-resolution runtime output.

### `RuntimeFingerprint`

- Field on `ResolvedProfile` in `geppetto/pkg/profiles/registry.go:45-55`.
- Computed in `geppetto/pkg/profiles/service.go:188`.
- Represents: a stable fingerprint of the resolved runtime plus relevant step settings and stack lineage.

This is not the same as `RuntimeKey`.

- `RuntimeKey` is a human-meaningful label.
- `RuntimeFingerprint` is a content-derived identity used for cache/rebuild/change detection.

### `ApplyRuntimeStepSettingsPatch`

- Defined in `geppetto/pkg/profiles/runtime_settings_patch_resolver.go:64`.
- Used in `ResolveEffectiveProfile` in `geppetto/pkg/profiles/service.go:160`.
- Represents: the bridge from profile runtime patch data into typed `settings.StepSettings`.

This matters to GP-41 because removing request overrides should not remove profile-based runtime patch application.

### `MergeRuntimeStepSettingsPatches`

- Defined in `geppetto/pkg/profiles/runtime_settings_patch_resolver.go:115`.
- Represents: deep merge behavior for `runtime.step_settings_patch`.

This is profile patch infrastructure, not JS runtime infrastructure and not app runtime composition.

### `ValidateRuntimeSpec`

- Defined in `geppetto/pkg/profiles/validation.go:37`.
- Represents: validation of the stored/resolved runtime profile payload.

This function belongs to profile data hygiene. It does not build engines.

### Profile runtime diagram

```text
ProfileRegistry
  -> Profile.Runtime (stored RuntimeSpec)
  -> ResolveEffectiveProfile(...)
      -> merge stack
      -> apply policy / overrides
      -> ApplyRuntimeStepSettingsPatch(...)
      -> produce ResolvedProfile
          -> EffectiveRuntime
          -> EffectiveStepSettings
          -> RuntimeKey
          -> RuntimeFingerprint
```

## 2. App-owned conversation runtime in Pinocchio

At this layer, runtime means: "the concrete runtime ingredients Pinocchio needs to serve a conversation."

This layer is not Geppetto profile storage. It is application composition.

### `ConversationRuntimeRequest` in `pinocchio/pkg/inference/runtime`

- Defined in `pinocchio/pkg/inference/runtime/composer.go:11-18`.
- Represents: the app-owned request passed into runtime composition.
- Important fields:
  - `ProfileKey`
  - `ProfileVersion`
  - `ResolvedProfileRuntime`
  - `ResolvedProfileFingerprint`

This is Pinocchio saying: "I already have profile-related facts; now compose me an actual runnable conversation runtime."

### `ComposedRuntime`

- Defined in `pinocchio/pkg/inference/runtime/composer.go:20-31`.
- Represents: the concrete runtime products consumed by conversation code.
- Contains:
  - `Engine`
  - `Sink`
  - `RuntimeFingerprint`
  - `RuntimeKey`
  - `SeedSystemPrompt`
  - `AllowedTools`

This is much closer to "what the app will run now" than `profiles.RuntimeSpec`.

Important distinction:

- `RuntimeSpec` is profile data.
- `ComposedRuntime` is application-ready runtime assembly.

### `RuntimeBuilder`

- Defined in `pinocchio/pkg/inference/runtime/composer.go:33-36`.
- Represents: the interface for composing a conversation runtime from a `ConversationRuntimeRequest`.

### `RuntimeBuilderFunc`

- Defined in `pinocchio/pkg/inference/runtime/composer.go:38-42`.
- Represents: adapter to let functions satisfy `RuntimeBuilder`.

This is ordinary Go API ergonomics, not a new runtime concept.

### `ProfileRuntimeComposer`

- Defined in `pinocchio/cmd/web-chat/runtime_composer.go:17-21`.
- Concrete implementation of runtime composition for the Pinocchio web chat app.
- Reads `ResolvedProfileRuntime`, resolves middlewares, applies profile `StepSettingsPatch`, builds an engine, and derives a runtime fingerprint.

This is the best concrete example of how Geppetto profile runtime data gets turned into an application runtime.

### `RuntimeFingerprintInput` and `buildRuntimeFingerprint`

- Defined in `pinocchio/cmd/web-chat/runtime_composer.go:269-290`.
- Represents: the payload Pinocchio hashes when it needs to derive a runtime fingerprint itself rather than trusting an upstream resolver-owned fingerprint.

This is similar in spirit to Geppetto's profile `RuntimeFingerprint`, but it belongs to the app composition layer.

### `ConversationRuntimeRequest` in `pinocchio/pkg/webchat`

- Defined in `pinocchio/pkg/webchat/conversation_service.go:42-50`.
- Represents: a webchat-service-facing request contract.
- Fields:
  - `RuntimeKey`
  - `RuntimeFingerprint`
  - `ResolvedRuntime`
  - `ResolvedProfileMetadata`
  - `Overrides`

This is one of the awkward surfaces in the current code. It overlaps conceptually with the newer `pinocchio/pkg/inference/runtime` vocabulary but is not the same type.

For an intern, the key lesson is:

- if you are in `pinocchio/pkg/inference/runtime`, you are looking at the composition abstraction,
- if you are in `pinocchio/pkg/webchat`, you may be looking at an app-facing request/handle contract that carries similar fields.

### App runtime diagram

```text
ResolvedProfile from Geppetto
  -> ResolvedProfile.EffectiveRuntime
  -> Pinocchio ConversationRuntimeRequest
  -> ProfileRuntimeComposer.Compose(...)
      -> middleware resolution
      -> step settings patch application
      -> engine build
      -> runtime fingerprint
  -> ComposedRuntime
  -> conversation lifecycle code
```

## 3. JavaScript runtime in Geppetto core

At this layer, runtime means: "a JavaScript VM and its owner/bridge/bootstrap state."

This has nothing to do with profile selection.

### `NewRuntime`

- Defined in `geppetto/pkg/js/runtime/runtime.go:31-76`.
- Represents: constructor for a Geppetto-owned JavaScript runtime exposing `require("geppetto")`.

This runtime is about JS execution and module registration, not about profile runtime resolution.

### `Options.RuntimeInitializers`

- Field on `geppetto/pkg/js/runtime/runtime.go:15-29`.
- Represents: callbacks executed after module registration to customize the VM.

These initializers run against a JS runtime context, not a profile runtime payload.

### `RuntimeInitializer`

- Type comes from `go-go-goja/engine`, but it is used directly in `geppetto/pkg/js/runtime/runtime.go:27-29,58-73`.
- Represents: an initializer hook for a JS runtime.

### `RuntimeContext`

- Built in `geppetto/pkg/js/runtime/runtime.go:58-64`.
- Represents: the JavaScript runtime context passed into initializers.
- Contains:
  - `VM`
  - `Require`
  - `Loop`
  - `Owner`

### `moduleRuntime`

- Defined in `geppetto/pkg/js/modules/geppetto/module.go:61-85`.
- Represents: the internal state holder for the native JS `geppetto` module.
- Carries:
  - the JS VM,
  - runtime owner,
  - bridge,
  - Go tool registry,
  - profile registry handles,
  - middleware schemas,
  - default sinks and hooks.

Despite the name, `moduleRuntime` is not a user-facing runtime selection API. It is internal JS module state.

### JS runtime diagram

```text
js/runtime.NewRuntime(...)
  -> go-go-goja factory.NewRuntime(...)
  -> register require("geppetto")
  -> build RuntimeContext
  -> run RuntimeInitializers
  -> return owned JS runtime
```

## 4. Scoped JavaScript tool runtime

At this layer, runtime means: "a scoped JavaScript execution environment built for a specific tool or environment."

### `BuildRuntime`

- Defined in `geppetto/pkg/inference/tools/scopedjs/runtime.go:50-94`.
- Represents: builder for a scoped JS runtime plus executor, manifest, cleanup, and metadata.

This is a specialized runtime factory used for scoped tool environments. It is not part of profile resolution.

### `runtimeInitFunc`

- Defined in `geppetto/pkg/inference/tools/scopedjs/runtime.go:34-48`.
- Represents: adapter that turns a function into a `gojengine.RuntimeInitializer`.

### `runtimeInitializers`

- Method on `Builder` in `geppetto/pkg/inference/tools/scopedjs/runtime.go:117-130`.
- Represents: the collected JS runtime initializers derived from configured globals and explicit initializers.

### `RuntimeLabel`

- Not defined in this file, but it is part of the scoped environment spec passed into `BuildRuntime`.
- Represents: a human-readable label for the scoped JS runtime being built.

This is another example of runtime meaning "JS execution environment identity," not "profile runtime."

## 5. Timeline runtime in Pinocchio

At this layer, runtime means: "the optional runtime bridge that can consume SEM events before builtin projection."

### `TimelineSemRuntime`

- Defined in `pinocchio/pkg/webchat/timeline_registry.go:24-28`.
- Represents: the interface for runtime-backed SEM event handling.
- Method:
  - `HandleSemEvent(ctx, p, ev, now) (handled bool, err error)`

The important semantic rule is documented in the same file:

- if `handled` is true, builtin projection is skipped.

### `SetTimelineRuntime`

- Defined in `pinocchio/pkg/webchat/timeline_registry.go:46-53`.
- Represents: install a timeline runtime bridge.

### `ClearTimelineRuntime`

- Defined in `pinocchio/pkg/webchat/timeline_registry.go:55-62`.
- Represents: uninstall the timeline runtime bridge.

### `JSTimelineRuntime`

- Defined in `pinocchio/pkg/webchat/timeline_js_runtime.go:37-48`.
- Represents: a JavaScript-backed implementation of `TimelineSemRuntime`.
- Holds:
  - owned JS runtime,
  - VM,
  - runtime owner,
  - reducer registry,
  - handler registry.

### `JSTimelineRuntimeOptions`

- Defined in `pinocchio/pkg/webchat/timeline_js_runtime.go:50-52`.
- Represents: constructor options for the JS timeline runtime, mainly require options.

### `NewJSTimelineRuntime`

- Defined in `pinocchio/pkg/webchat/timeline_js_runtime.go:54-60`.
- Represents: panic-on-error convenience constructor.

### `NewJSTimelineRuntimeWithOptions`

- Defined in `pinocchio/pkg/webchat/timeline_js_runtime.go:62-97`.
- Represents: explicit constructor for a timeline JS runtime with dependency configuration.

### Timeline runtime execution order

`pinocchio/pkg/webchat/timeline_registry.go:64-99` is especially important:

- runtime executes first,
- runtime may consume the event,
- list handlers run only if runtime did not consume,
- runtime errors are treated as handled so the error propagates.

That means this runtime is a projection-control hook, not just a helper.

### Timeline runtime diagram

```text
configureTimelineJSScripts(...)
  -> ClearTimelineRuntime()
  -> NewJSTimelineRuntimeWithOptions(...)
  -> LoadScriptFile(...)
  -> SetTimelineRuntime(runtime)

SEM event arrives
  -> handleTimelineHandlers(...)
      -> timeline runtime first
      -> if handled, skip builtin/list handlers
      -> else continue projection
```

## 6. Runtime attribution in turns and provider metadata

At this layer, runtime means: "runtime-related metadata stored on a turn and later normalized into event metadata."

### `KeyTurnMetaRuntime`

- Defined in `geppetto/pkg/turns/keys_gen.go:67-77`.
- Represents: the typed metadata key used to store runtime attribution on a turn.

The value can be:

- a string runtime key,
- or a richer `map[string]any` payload including:
  - `runtime_key`
  - `runtime_fingerprint`
  - `profile_slug`
  - `registry_slug`
  - `profile_version`

### `AddRuntimeAttributionToExtra`

- Defined in `geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go:10-72`.
- Represents: normalization of turn runtime metadata into provider event `extra` metadata.

It copies runtime/profile facts from `KeyTurnMetaRuntime` into the dotted keys used downstream by persistence and tracing.

This is not runtime selection. It is runtime attribution.

### Runtime attribution diagram

```text
Turn.Metadata[KeyTurnMetaRuntime]
  -> AddRuntimeAttributionToExtra(extra, turn)
      -> extra["runtime_key"]
      -> extra["runtime_fingerprint"]
      -> extra["profile.slug"]
      -> extra["profile.registry"]
      -> extra["profile.version"]
```

## 7. Raw runtime lexicon by subsystem

This appendix is the compact inventory of the architecture-significant Runtime-named symbols across Geppetto and Pinocchio. It intentionally excludes trivial local variables and test function names.

### Geppetto profile/runtime selection lexicon

- `RuntimeSpec`
- `RuntimeKey`
- `ParseRuntimeKey`
- `MustRuntimeKey`
- `RuntimeKeyFallback`
- `ResolveEffectiveProfile`
- `EffectiveRuntime`
- `RuntimeFingerprint`
- `ApplyRuntimeStepSettingsPatch`
- `MergeRuntimeStepSettingsPatches`
- `ValidateRuntimeSpec`

### Geppetto JS/runtime lexicon

- `NewRuntime`
- `RuntimeInitializers`
- `RuntimeInitializer`
- `RuntimeContext`
- `moduleRuntime`

### Geppetto scoped JS/runtime lexicon

- `BuildRuntime`
- `runtimeInitFunc`
- `runtimeInitializers`
- `RuntimeLabel`

### Geppetto runtime metadata lexicon

- `KeyTurnMetaRuntime`
- `TurnMetaRuntimeValueKey`
- `AddRuntimeAttributionToExtra`

### Pinocchio app runtime lexicon

- `ConversationRuntimeRequest`
- `ComposedRuntime`
- `RuntimeBuilder`
- `RuntimeBuilderFunc`
- `ProfileRuntimeComposer`
- `RuntimeFingerprintInput`
- `buildRuntimeFingerprint`

### Pinocchio timeline runtime lexicon

- `TimelineSemRuntime`
- `SetTimelineRuntime`
- `ClearTimelineRuntime`
- `JSTimelineRuntime`
- `JSTimelineRuntimeOptions`
- `NewJSTimelineRuntime`
- `NewJSTimelineRuntimeWithOptions`

## Example 1: profile runtime resolution feeding Pinocchio web chat

This is the single most important example for GP-41.

### What happens

1. Geppetto resolves a profile through `ResolveEffectiveProfile`.
2. The result contains:
   - `EffectiveRuntime`
   - `EffectiveStepSettings`
   - `RuntimeKey`
   - `RuntimeFingerprint`
3. Pinocchio turns the resolved runtime into a `ConversationRuntimeRequest`.
4. `ProfileRuntimeComposer.Compose` converts that request into `ComposedRuntime`.
5. Conversation lifecycle code runs the composed engine.

### Example pseudocode

```go
resolved, err := profileRegistry.ResolveEffectiveProfile(ctx, profiles.ResolveInput{
    RegistrySlug:       profiles.MustRegistrySlug("private"),
    ProfileSlug:        profiles.MustProfileSlug("agent"),
    RuntimeKeyFallback: profiles.MustRuntimeKey("chat"),
    BaseStepSettings:   base,
})
if err != nil {
    return err
}

req := infruntime.ConversationRuntimeRequest{
    ProfileKey:                 resolved.RuntimeKey.String(),
    ProfileVersion:             extractVersion(resolved.Metadata),
    ResolvedProfileRuntime:     &resolved.EffectiveRuntime,
    ResolvedProfileFingerprint: resolved.RuntimeFingerprint,
}

runtime, err := composer.Compose(ctx, req)
if err != nil {
    return err
}

// runtime.Engine is now what the conversation code actually runs.
_ = runtime
```

### Why this example matters

This flow shows the exact handoff point between:

- profile runtime data,
- and app-owned composed runtime.

That handoff is where people often confuse `RuntimeSpec` with `ComposedRuntime`.

## Example 2: loading timeline JavaScript reducers into the timeline runtime

This is the clearest example of runtime meaning "JavaScript execution environment."

### What happens

1. The web chat command gathers JS script paths.
2. It clears any existing timeline runtime bridge.
3. It constructs a `JSTimelineRuntime`.
4. It loads scripts into that runtime.
5. It installs the runtime bridge with `SetTimelineRuntime`.
6. Later, `handleTimelineHandlers` invokes the runtime before builtin projection.

### Example pseudocode

```go
func configureTimeline(paths []string) error {
    webchat.ClearTimelineRuntime()

    rt, err := webchat.NewJSTimelineRuntimeWithOptions(webchat.JSTimelineRuntimeOptions{})
    if err != nil {
        return err
    }

    for _, path := range paths {
        if err := rt.LoadScriptFile(path); err != nil {
            return err
        }
    }

    webchat.SetTimelineRuntime(rt)
    return nil
}
```

### Why this example matters

This runtime has nothing to do with profiles or step settings. It is an event-processing bridge with its own JS VM lifecycle.

## Example 3: persisting runtime attribution from turns into provider events

This is the clearest example of runtime meaning "metadata stamped on an inference run."

### What happens

1. Upstream code stores runtime attribution on `Turn.Metadata` using `KeyTurnMetaRuntime`.
2. A provider engine prepares request/event metadata.
3. `AddRuntimeAttributionToExtra` copies normalized runtime/profile fields into the event payload.
4. Persistence, logs, or downstream systems can now inspect the runtime identity that produced the event.

### Example pseudocode

```go
_ = turns.KeyTurnMetaRuntime.Set(&turn.Metadata, map[string]any{
    "runtime_key":         "planner",
    "runtime_fingerprint": "sha256:abc123",
    "profile_slug":        "planner",
    "registry_slug":       "private",
    "profile_version":     uint64(42),
})

extra := map[string]any{}
runtimeattrib.AddRuntimeAttributionToExtra(extra, turn)

// extra now contains runtime_key, runtime_fingerprint,
// profile.slug, profile.registry, and profile.version.
```

### Concrete evidence in the codebase

Provider engines call this helper directly, for example in `geppetto/pkg/steps/ai/openai_responses/engine.go:149-156`.

## Alternatives Considered

### Alternative 1: do not write a glossary, just rely on code comments

Rejected because the confusion is cross-package, not single-file. Comments inside one file do not explain why another package uses the same word differently.

### Alternative 2: rename everything before documenting it

Rejected for GP-41 because that would expand the ticket into a much larger naming/refactor effort. We need understanding before renaming.

### Alternative 3: dump a raw grep list into the ticket

Rejected because raw grep output is not an onboarding guide. The intern needs grouping, explanations, and examples.

## Implementation Plan

This glossary supports the GP-41 implementation in three practical ways.

### Phase 1: use it during review of request override removal

When reading `ResolveInput`, `ResolvedProfile`, or `RuntimeSpec`, stay in the Geppetto profile layer first. Do not jump into JS runtime or timeline runtime code unless the call graph actually crosses there.

### Phase 2: use it when simplifying Pinocchio request contracts

When touching Pinocchio request/handle types, identify whether a field belongs to:

- resolved profile runtime data,
- app-owned composed runtime data,
- or turn metadata attribution.

That helps prevent accidental duplication or loss of runtime identity fields.

### Phase 3: use it as a naming checkpoint for future cleanup

Future cleanup work can use this glossary to decide whether a symbol should be:

- kept,
- renamed,
- collapsed into another layer,
- or documented more clearly at the package boundary.

## Open Questions

1. Should Pinocchio eventually converge on one `ConversationRuntimeRequest` type instead of carrying similarly named request shapes in multiple packages?
2. Should `RuntimeKey` and `ProfileKey` naming be unified across the app/runtime boundary, or is the distinction still useful?
3. Should `RuntimeSpec` eventually become less overloaded by splitting profile storage concerns from resolved runtime output concerns?
4. After GP-41, should runtime fingerprint derivation move more consistently into resolver-owned code rather than sometimes being recomputed in the app layer?

## References

- `geppetto/pkg/profiles/types.go`
- `geppetto/pkg/profiles/registry.go`
- `geppetto/pkg/profiles/service.go`
- `geppetto/pkg/profiles/slugs.go`
- `geppetto/pkg/js/runtime/runtime.go`
- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/inference/tools/scopedjs/runtime.go`
- `geppetto/pkg/turns/keys_gen.go`
- `geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go`
- `pinocchio/pkg/inference/runtime/composer.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`
- `pinocchio/pkg/webchat/conversation_service.go`
- `pinocchio/pkg/webchat/timeline_registry.go`
- `pinocchio/pkg/webchat/timeline_js_runtime.go`
- `pinocchio/cmd/web-chat/timeline_js_runtime_loader.go`
