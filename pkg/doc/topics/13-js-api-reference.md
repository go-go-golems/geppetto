---
Title: Geppetto JavaScript API Reference
Slug: geppetto-js-api-reference
Short: Reference for the native `require("geppetto")` API exposed through goja.
Topics:
- geppetto
- javascript
- goja
- api-reference
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page documents the current hard-cut JavaScript API in [pkg/js/modules/geppetto](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto).

## Execution Harness

Run scripts with:

```bash
go run ./cmd/examples/geppetto-js-lab --script <path-to-script.js>
```

## Top-Level Namespaces

- `turns`
- `engines`
- `profiles`
- `runner`
- `middlewares`
- `schemas`
- `events`
- `tools`

## `engines`

Current constructors:

- `echo({ reply? })`
- `fromFunction(fn)`
- `fromConfig(options)`
- `fromResolvedProfile(resolvedProfile)`
- `fromProfile({ registrySlug?, profileSlug? })`

Recommended meaning:

- use `fromConfig(...)` when the script already knows the provider/model/settings
- use `fromProfile(...)` or `fromResolvedProfile(...)` when the script wants Geppetto to resolve engine settings from an engine profile registry

`fromProfile(...)` and `fromResolvedProfile(...)` build the engine only. They do not apply prompts, tool policy, or middleware policy.

## `profiles`

The `profiles` namespace is now really an **engine profiles** namespace, even though the JS name is still `profiles`.

Available functions:

- `listRegistries()`
- `getRegistry(registrySlug?)`
- `listProfiles(registrySlug?)`
- `getProfile(profileSlug, registrySlug?)`
- `resolve({ registrySlug?, profileSlug? })`
- `connectStack(sources)`
- `disconnectStack()`
- `getConnectedSources()`

`resolve(...)` returns:

- `registrySlug`
- `profileSlug`
- `inferenceSettings`
- `metadata`

It no longer returns:

- `effectiveRuntime`
- `runtimeKey`
- `runtimeFingerprint`

## `runner`

`runner` is now purely app/runtime-oriented.

Available functions:

- `resolveRuntime(input?)`
- `prepare(options)`
- `run(options, runOptions?)`
- `start(options, runOptions?)`

`runner.resolveRuntime(...)` accepts only direct runtime input such as:

- `systemPrompt`
- `middlewares`
- `toolNames`
- `runtimeKey`
- `runtimeFingerprint`
- `profileVersion`
- `metadata`

It no longer accepts `profile`.

If you need engine profiles, resolve them separately:

```javascript
const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
const engine = gp.engines.fromResolvedProfile(resolved);
const runtime = gp.runner.resolveRuntime({
  systemPrompt: "App-owned prompt",
  runtimeKey: "assistant",
});
```

## `createBuilder` / `createSession`

Builder/session construction is now lower-level again.

Supported builder inputs:

- `engine`
- `middlewares`
- `tools`
- `toolLoop`
- `toolHooks`

Removed from the builder path:

- `resolvedProfile`
- `useResolvedProfile(...)`

If you want profile-derived engine settings, build the engine first with `gp.engines.fromProfile(...)` or `gp.engines.fromResolvedProfile(...)`, then pass the engine into the builder or runner.

## Files

Relevant implementation files:

- [api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go)
- [api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go)
- [api_runner.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runner.go)
- [api_runtime_metadata.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go)
- [api_builder_options.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_builder_options.go)
