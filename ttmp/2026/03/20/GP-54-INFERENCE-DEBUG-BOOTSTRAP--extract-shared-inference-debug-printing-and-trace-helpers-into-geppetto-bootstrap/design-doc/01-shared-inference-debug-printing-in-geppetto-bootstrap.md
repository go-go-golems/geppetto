---
Title: Shared inference debug printing in geppetto bootstrap
Ticket: GP-54-INFERENCE-DEBUG-BOOTSTRAP
Status: active
Topics:
    - architecture
    - geppetto
    - pinocchio
    - glazed
    - profiles
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: 2026-03-14--cozodb-editor/backend/main.go
      Note: Downstream evidence that hidden-base trace reconstruction and debug rendering are not centralized
    - Path: geppetto/pkg/cli/bootstrap/config.go
      Note: Defines the generic bootstrap contract that should own the moved helper
    - Path: geppetto/pkg/cli/bootstrap/engine_settings.go
      Note: Shows that resolved engine settings already live in Geppetto
    - Path: geppetto/pkg/cli/bootstrap/profile_selection.go
      Note: Shows how visible profile selection is already owned by Geppetto bootstrap
    - Path: pinocchio/cmd/pinocchio/cmds/js.go
      Note: Current duplicated print branch in the JS command
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: Current duplicated print branch in the main Pinocchio command runner
    - Path: pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go
      Note: Generic trace implementation to be moved
ExternalSources: []
Summary: Detailed design and implementation guide for extracting a single generic inference debug output path into geppetto/pkg/cli/bootstrap.
LastUpdated: 2026-03-20T15:25:00-04:00
WhatFor: Onboard a new engineer to the current bootstrap/debug architecture and provide a precise plan for moving shared inference debug behavior into Geppetto.
WhenToUse: Use when implementing the extraction, reviewing ownership boundaries between Geppetto and Pinocchio, or onboarding a new engineer to this part of the bootstrap stack.
---


# Shared inference debug printing in geppetto bootstrap

## Executive Summary

This ticket proposes moving the generic inference debug functionality out of `pinocchio/pkg/cmds/profilebootstrap` and `pinocchio/pkg/cmds/cmdlayers`, and into `geppetto/pkg/cli/bootstrap`. The current arrangement is functionally workable, but architecturally split in the wrong place. Geppetto already owns the reusable bootstrap primitives: application bootstrap configuration, config-file resolution, base inference settings, profile resolution, and the final resolved engine settings object. However, the logic for exposing debug output, building a per-field source trace, reconstructing hidden bootstrap inputs for trace fidelity, and printing YAML is still scattered across Pinocchio and downstream applications.

For a new engineer, the most important mental model is this: the bootstrap system has two layers. The first layer computes resolved inference settings from defaults, environment, config files, and optional profile registries. The second layer optionally explains that result to a human via debug output. Today the computation is mostly in Geppetto and the explanation layer is partly in Pinocchio and partly in app call sites. That split creates duplication, inconsistent security behavior, and confusing package ownership.

The recommended target architecture is:

1. Geppetto owns generic debug functionality in `geppetto/pkg/cli/bootstrap`.
2. Pinocchio keeps only Pinocchio-specific bootstrap configuration and config-file mapping.
3. Pinocchio imports the new Geppetto debug helper directly instead of re-exporting it.
4. Downstream apps also import the Geppetto helper directly.
5. The shared interface is one flag, `--print-inference-settings`, and one output shape that includes both effective values and their sources.

That is the clean cut requested by the user: move as much as possible into Geppetto, do not paper over the move by re-exporting the new helper from Pinocchio, and leave Pinocchio with only app-specific concerns.

## Problem Statement

The system already supports useful inference debugging, but the behavior is distributed across packages in a way that makes reuse awkward and encourages downstream copy-paste.

Observed facts:

1. `geppetto/pkg/cli/bootstrap` already defines `AppBootstrapConfig`, config-file resolution, profile selection, base settings resolution, and `ResolvedCLIEngineSettings` as the canonical generic bootstrap API. See:
   - `geppetto/pkg/cli/bootstrap/config.go:13-19`
   - `geppetto/pkg/cli/bootstrap/profile_selection.go:48-87`
   - `geppetto/pkg/cli/bootstrap/engine_settings.go:17-24`
   - `geppetto/pkg/cli/bootstrap/engine_settings.go:61-151`
2. `pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go` contains a generic trace builder that depends only on Geppetto types plus `values.Values`. It is not inherently Pinocchio-specific. See:
   - `pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go:42-95`
   - `pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go:98-178`
3. The inference debug flag section currently lives in `pinocchio/pkg/cmds/cmdlayers/helpers.go`, even though the small debug-only section is just two generic flags. See:
   - `pinocchio/pkg/cmds/cmdlayers/helpers.go:165-182`
4. Pinocchio command call sites each implement their own “if debug flag, print settings or trace and exit” branch:
   - `pinocchio/pkg/cmds/cmd.go:318-341`
   - `pinocchio/cmd/pinocchio/cmds/js.go:159-180`
5. A downstream app had to add its own custom debug section, hidden-base trace reconstruction, and redaction logic on top:
   - `2026-03-14--cozodb-editor/backend/main.go:211-228`
   - `2026-03-14--cozodb-editor/backend/main.go:266-299`

This split creates three concrete problems:

### 1. Ownership is inconsistent

Geppetto owns the resolved settings model, but Pinocchio owns part of the most reusable way to inspect it.

### 2. Call-site behavior is duplicated

Every call site has to remember:

- which flags to decode,
- how to build the trace,
- when to exit,
- whether to redact secrets,
- and whether hidden base sections must be reconstructed before tracing.

### 3. Security behavior is not centralized

The original Pinocchio command paths write raw YAML. The downstream backend had to add its own redaction logic. That means secret handling is not guaranteed to be consistent across callers.

## Scope

### In Scope

This ticket covers the generic debug-printing functionality that belongs in Geppetto:

- the debug settings type,
- the debug section builder,
- the inference settings trace builder,
- helpers to reconstruct trace-ready parsed values from hidden base sections,
- YAML rendering for a combined settings-and-sources document,
- simple secret masking with `***`,
- and a small orchestration helper that answers “was debug output requested, and if so, print and exit.”

### Out of Scope

This ticket does not change the underlying inference settings model itself. It also does not redesign profile registries or config-file discovery. Those systems are inputs to the debug helper, not the focus of this extraction.

This ticket also does not introduce extra debug variants such as separate source-only flags or redaction modes. The design goal is one obvious debug path.

## System Orientation for a New Intern

This section explains the major packages involved before we talk about the refactor.

### What Geppetto is responsible for

Geppetto is the generic runtime/configuration layer. In this context it owns:

- bootstrap configuration via `AppBootstrapConfig`,
- config-file discovery,
- profile selection,
- base inference settings resolution,
- profile merge logic,
- and the final `ResolvedCLIEngineSettings` object.

The important point is that Geppetto is already the system that answers “what inference settings are active?” The proposed extraction simply makes Geppetto also answer “how do I debug and print those settings safely?”

### What Pinocchio is responsible for

Pinocchio is an application and app-framework layer on top of Geppetto. It provides:

- Pinocchio-specific app bootstrap defaults such as app name and env prefix,
- Pinocchio-specific config-file mapping,
- concrete command surfaces,
- and application runtime behavior.

Pinocchio should keep app-specific bootstrap wrappers. It should not keep generic debug/trace logic once that logic can be expressed purely in terms of Geppetto bootstrap types.

### What the downstream backend is doing

The CozoDB backend is a useful stress test because it is neither Geppetto nor Pinocchio itself. It consumes the bootstrap system from outside and had to add its own glue for:

- default profile behavior,
- hidden-base parsed-value reconstruction,
- and redacted debug output.

That downstream duplication is a strong signal that the generic helper is currently too low-level.

## Current-State Architecture

### Core Bootstrap Ownership

`geppetto/pkg/cli/bootstrap/config.go` defines the central contract:

```go
type AppBootstrapConfig struct {
    AppName           string
    EnvPrefix         string
    ConfigFileMapper  sources.ConfigFileMapper
    NewProfileSection func() (schema.Section, error)
    BuildBaseSections func() ([]schema.Section, error)
}
```

This contract matters because it is already the correct abstraction boundary. A caller provides:

- how to find its app config,
- how to interpret its config shape,
- how to expose profile selection,
- and which hidden base sections should participate in inference resolution.

Everything needed for generic inference debug output can be derived from this contract.

### Profile Selection Flow

`ResolveCLIProfileSelection` in `geppetto/pkg/cli/bootstrap/profile_selection.go:48-87` resolves visible profile-selection state by layering:

1. environment,
2. config files,
3. defaults,
4. and explicit parsed values.

This is the visible bootstrap side. It knows about:

- `profile`
- `profile-registries`
- and config-file inputs via `CommandSettings`.

### Hidden Base Resolution Flow

`ResolveBaseInferenceSettings` in `geppetto/pkg/cli/bootstrap/engine_settings.go:26-58` resolves hidden base sections using the same app bootstrap config. This is the hidden bootstrap side. It is not limited to visible CLI flags; it loads the full base inference surface.

That distinction is crucial:

- profile selection may be narrow and user-facing,
- but final inference settings depend on hidden base sections too.

### Final Merge Flow

`ResolveCLIEngineSettings` and `ResolveCLIEngineSettingsFromBase` in `engine_settings.go:61-151` merge:

1. base inference settings,
2. selected profile registries,
3. resolved profile settings,
4. and profile merges.

The result is `ResolvedCLIEngineSettings`, which contains:

- the base settings,
- the final settings,
- the resolved profile metadata,
- config files,
- and a cleanup hook.

This is the exact object a debug helper should operate on.

### Current Trace Builder

`pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go` converts the resolved settings into a nested YAML-friendly trace of sources.

Conceptually it works like this:

```text
resolved settings
    -> flatten every final field path
    -> collect source logs from:
         command baseline (optional)
         parsed values
         resolved profile settings
    -> if no source exists for a field
         mark it as implicit-defaults
    -> rebuild nested YAML tree
```

The implementation is generic:

- `BuildInferenceSettingsSourceTrace(...)` orchestrates the work.
- `applyParsedValueSources(...)` maps parsed sections/fields to inference paths.
- `applyProfileSource(...)` annotates fields with profile metadata.
- `applySettingsSource(...)` applies one source block to all leaves in a settings object.

There is no Pinocchio-only data model in this file. The package placement is historical rather than architectural.

### Current Debug Flag Section

`pinocchio/pkg/cmds/cmdlayers/helpers.go:165-182` defines `NewInferenceDebugParameterLayer()`, which exposes only:

- `--print-inference-settings`
- `--print-inference-settings-sources`

This section is already small and generic. Its placement in `cmdlayers` is convenient for Pinocchio, but it does not need Pinocchio-specific runtime state. The simplification requested by the user is to collapse this down to a single shared flag, `--print-inference-settings`, and make that output include source provenance inline instead of through a second flag.

### Current Call Sites

There are at least three distinct call-site patterns today.

#### 1. Pinocchio generic command runner

`pinocchio/pkg/cmds/cmd.go:318-341`:

- reads `HelpersSettings`,
- optionally synthesizes a minimal `ResolvedCLIEngineSettings`,
- builds the trace,
- writes YAML,
- exits before engine creation.

#### 2. Pinocchio JS command

`pinocchio/cmd/pinocchio/cmds/js.go:159-180`:

- decodes the debug booleans,
- calls `BuildInferenceSettingsSourceTrace(...)`,
- writes raw YAML,
- exits.

#### 3. Downstream backend

`2026-03-14--cozodb-editor/backend/main.go:211-228` and `:266-299`:

- defines a custom local debug section,
- reconstructs hidden-base parsed values so the trace is accurate,
- redacts secrets,
- writes YAML,
- exits.

This is the most compelling evidence that the generic helper is incomplete today.

## Gap Analysis

The current architecture has reusable pieces, but not a reusable workflow.

### Gap 1: No single entrypoint

There is no function shaped like:

```go
handled, err := bootstrap.HandleInferenceDebugOutput(...)
if handled { return err }
```

Without that helper, every command has to rebuild the same branch logic.

### Gap 2: Trace fidelity requires hidden-base reconstruction, but that is not shared

The downstream backend needed `buildInferenceTraceParsedValues(...)` because the visible parsed values alone do not show config/default provenance for hidden base sections. That functionality is exactly bootstrap-specific and belongs near the bootstrap package.

### Gap 3: Output policy is more complicated than it needs to be

Pinocchio current command paths emit raw YAML. The backend added custom redaction after discovering secret leakage risk. The system should have one shared default-safe path. For this ticket, the policy can be deliberately simple: sensitive values become `***`.

### Gap 4: Pinocchio wrapper package mixes app config with generic helpers

`pinocchio/pkg/cmds/profilebootstrap` currently contains:

- valid Pinocchio-specific wrapper responsibilities,
- plus a generic inference trace builder that should live in Geppetto.

This is the main ownership cleanup target.

## Proposed Architecture

### Target Package

Move the generic functionality to:

```text
geppetto/pkg/cli/bootstrap/inference_debug.go
```

### What Moves

Move these concepts into Geppetto bootstrap:

1. `InferenceDebugSettings`
2. `NewInferenceDebugSection(...)`
3. `BuildInferenceSettingsSourceTrace(...)`
4. helper functions used only by the trace builder
5. shared trace-ready parsed-values reconstruction from hidden base sections
6. YAML writer for a combined settings-and-sources document
7. simple masking of sensitive values with `***`
8. orchestration helper that:
   - reads or accepts debug settings
   - prints output if requested
   - returns `(handled bool, err error)`

### What Stays in Pinocchio

Pinocchio should keep:

1. `pinocchioBootstrapConfig()` and the wrappers that inject:
   - `AppName: "pinocchio"`
   - `EnvPrefix: "PINOCCHIO"`
   - Pinocchio config-file mapping
2. any broader helper settings struct that contains many non-debug flags
3. any app-specific command wiring that is unrelated to inference debug output

### What Should Not Happen

Do not move the functionality and then re-export it from Pinocchio. That would preserve the wrong dependency direction. The point of this ticket is to make Geppetto the direct home for shared behavior.

## Proposed API Surface

The exact names can vary, but the following shape is a good target:

```go
package bootstrap

type InferenceDebugSettings struct {
    PrintInferenceSettings bool `glazed:"print-inference-settings"`
}

func NewInferenceDebugSection() (schema.Section, error)

func BuildInferenceTraceParsedValues(
    cfg AppBootstrapConfig,
    parsed *values.Values,
) (*values.Values, error)

type InferenceDebugOutputOptions struct {
    CommandBase    *aisettings.InferenceSettings
    ParsedForTrace *values.Values
}

func WriteInferenceSettingsDebugYAML(
    w io.Writer,
    resolved *ResolvedCLIEngineSettings,
    opts InferenceDebugOutputOptions,
) error

func HandleInferenceDebugOutput(
    w io.Writer,
    cfg AppBootstrapConfig,
    parsed *values.Values,
    settings InferenceDebugSettings,
    resolved *ResolvedCLIEngineSettings,
    opts InferenceDebugOutputOptions,
) (bool, error)
```

### Why this API shape works

It preserves a clear layering:

- `ResolvedCLIEngineSettings` remains the resolved data object.
- `AppBootstrapConfig` remains the contract that knows how to rebuild hidden base inputs.
- `HandleInferenceDebugOutput(...)` becomes the small convenience API.
- the writer emits one combined document, so callers do not need to choose between “values” and “sources.”

## Detailed Execution Flow

### Current desired generic flow

```text
caller parses flags
    -> caller resolves ResolvedCLIEngineSettings
    -> caller decodes InferenceDebugSettings
    -> bootstrap.HandleInferenceDebugOutput(...)
        -> if print settings:
             rebuild hidden-base parsed values if needed
             build source trace
             mask sensitive values as ***
             write combined settings-and-sources YAML
             return handled=true
        -> else:
             return handled=false
```

### ASCII Diagram

```text
                    +--------------------------------------+
                    | caller command / server entrypoint   |
                    +-------------------+------------------+
                                        |
                                        v
                    +--------------------------------------+
                    | ResolveCLIEngineSettings(...)        |
                    |  - base hidden sections              |
                    |  - env/config/defaults               |
                    |  - profile selection                 |
                    |  - profile registry merge            |
                    +-------------------+------------------+
                                        |
                                        v
                    +--------------------------------------+
                    | HandleInferenceDebugOutput(...)      |
                    |  - check debug flags                 |
                    |  - rebuild trace parsed values       |
                    |  - BuildInferenceSettingsSourceTrace |
                    |  - mask sensitive fields as ***      |
                    |  - emit YAML and exit if handled     |
                    +-------------------+------------------+
                                        |
                              handled? / \ not handled
                                      /   \
                                     v     v
                          return early     continue normal run
```

## Pseudocode for the Refactor

### New shared helper

```go
func HandleInferenceDebugOutput(
    w io.Writer,
    cfg AppBootstrapConfig,
    parsed *values.Values,
    debugSettings InferenceDebugSettings,
    resolved *ResolvedCLIEngineSettings,
    opts InferenceDebugOutputOptions,
) (bool, error) {
    if resolved == nil || resolved.FinalInferenceSettings == nil {
        return false, nil
    }

    if debugSettings.PrintInferenceSettings {
        parsedForTrace := opts.ParsedForTrace
        if parsedForTrace == nil {
            parsedForTrace = BuildInferenceTraceParsedValues(cfg, parsed)
        }
        return true, WriteInferenceSettingsDebugYAML(
            w, resolved, InferenceDebugOutputOptions{
                CommandBase:    opts.CommandBase,
                ParsedForTrace: parsedForTrace,
            })
    }

    return false, nil
}
```

### Pinocchio command runner after extraction

```go
debugSettings := bootstrap.InferenceDebugSettings{
    PrintInferenceSettings: helpersSettings.PrintInferenceSettings,
}

handled, err := bootstrap.HandleInferenceDebugOutput(
    w,
    pinocchioBootstrapConfig(),
    parsedValues,
    debugSettings,
    resolvedEngineSettings,
    bootstrap.InferenceDebugOutputOptions{
        CommandBase: g.BaseInferenceSettings,
    },
)
if err != nil { return err }
if handled { return nil }
```

### JS command after extraction

```go
debugSection, _ := bootstrap.NewInferenceDebugSection()
// mount section

debugSettings := &bootstrap.InferenceDebugSettings{}
_ = parsed.DecodeSectionInto(bootstrap.InferenceDebugSectionSlug, debugSettings)

handled, err := bootstrap.HandleInferenceDebugOutput(
    w,
    pinocchioBootstrapConfig(),
    parsed,
    *debugSettings,
    runtimeBootstrap.ResolvedEngineSettings,
    bootstrap.InferenceDebugOutputOptions{},
)
if err != nil { return err }
if handled { return nil }
```

### Downstream backend after extraction

```go
debugSettings := &bootstrap.InferenceDebugSettings{}
_ = parsed.DecodeSectionInto(bootstrap.InferenceDebugSectionSlug, debugSettings)

handled, err := bootstrap.HandleInferenceDebugOutput(
    w,
    appBootstrapConfig(),
    effectiveParsed,
    *debugSettings,
    resolved,
    bootstrap.InferenceDebugOutputOptions{},
)
if err != nil { return err }
if handled { return nil }
```

## Implementation Phases

### Phase 1: Move the generic trace logic into Geppetto

Files:

- create `geppetto/pkg/cli/bootstrap/inference_debug.go`

Actions:

1. Move `InferenceSettingSource` and the trace helper functions from Pinocchio to Geppetto bootstrap.
2. Rename package imports so the code compiles purely against Geppetto bootstrap types.
3. Preserve behavior before adding the combined output helper.

### Phase 2: Add a shared debug section and combined output helper

Files:

- `geppetto/pkg/cli/bootstrap/inference_debug.go`

Actions:

1. Add `InferenceDebugSettings`.
2. Add `NewInferenceDebugSection()`.
3. Add a single combined YAML rendering helper.
4. Add `BuildInferenceTraceParsedValues(cfg, parsed)`.
5. Mask sensitive values as `***`.

### Phase 3: Add the orchestration helper

Files:

- `geppetto/pkg/cli/bootstrap/inference_debug.go`

Actions:

1. Add `HandleInferenceDebugOutput(...)`.
2. Keep it intentionally small: one flag in, one combined debug document out.

### Phase 4: Switch Pinocchio call sites with no re-export

Files:

- `pinocchio/pkg/cmds/cmd.go`
- `pinocchio/cmd/pinocchio/cmds/js.go`
- `pinocchio/pkg/cmds/cmdlayers/helpers.go`
- `pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go`

Actions:

1. Replace direct trace-building and YAML-printing branches with the Geppetto helper.
2. Switch the JS command from `cmdlayers.NewInferenceDebugParameterLayer()` to `bootstrap.NewInferenceDebugSection()`.
3. Delete `pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go`.
4. Remove `cmdlayers.NewInferenceDebugParameterLayer()` if no longer needed anywhere.

### Phase 5: Switch downstream backend

Files:

- `2026-03-14--cozodb-editor/backend/main.go`

Actions:

1. Replace local debug section, trace reconstruction, and redaction helpers with the Geppetto helper.
2. Keep app-specific bootstrap config local.

## Verification Strategy

This ticket does not need a dedicated debug-output test workstream. The user explicitly asked to keep this debug surface simple. The practical validation path is:

1. compile Geppetto after the move,
2. switch Pinocchio and verify the commands still print one combined debug document,
3. switch the downstream backend and verify it no longer needs local debug helpers.

The CozoDB backend remains a useful smoke test because it exercises:

- app-owned bootstrap config,
- default registry behavior,
- hidden-base trace reconstruction,
- and simple `***` masking.

## Risks

### Risk 1: Package-boundary confusion

If the extraction moves too much app-specific behavior, Geppetto could become opinionated about application UX instead of just bootstrap behavior.

Mitigation:

- move only the generic section and debug behavior,
- keep application-specific command setup outside the helper.

### Risk 2: Silent change in source-trace semantics

If the new shared helper reconstructs trace inputs differently, source labels could shift from `implicit-defaults` to `defaults` or `config`.

Mitigation:

- treat the downstream backend’s recent fix as the desired behavior,
- verify the combined debug output manually at each migrated call site.

### Risk 3: Trying to keep multiple debug modes alive

If the implementation tries to preserve separate “settings” and “sources” outputs or configurable redaction strategies, the extraction gets harder without real benefit.

Mitigation:

- use one flag,
- emit one combined document,
- mask secrets as `***`,
- and defer any more advanced policy until a real need appears.

## Alternatives Considered

### Alternative 1: Keep the logic in Pinocchio and let apps copy it

Rejected. This is exactly what caused the downstream backend duplication.

### Alternative 2: Move only the trace builder to Geppetto

Rejected. That would still leave:

- local debug section definitions,
- local hidden-base trace reconstruction,
- and local output code

at every call site.

### Alternative 3: Move everything and re-export from Pinocchio

Rejected. The user explicitly asked for a clean cut instead of re-exporting. Re-exporting would preserve the wrong dependency direction and make future cleanup harder.

## Open Questions

1. Should `BuildInferenceTraceParsedValues(...)` be public, or should it stay an implementation detail hidden behind `HandleInferenceDebugOutput(...)`?
2. Is `InferenceDebugSettings` best placed in `bootstrap`, or should Geppetto create a small dedicated `bootstrap/debug` subpackage if the file grows too large?

## Implementation Guide for an Intern

If you are new to the codebase, work in this order:

1. Read `geppetto/pkg/cli/bootstrap/config.go` and `engine_settings.go` first.
   This tells you what bootstrap owns already.
2. Read `pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go`.
   This shows the generic trace logic that is misplaced today.
3. Read the two Pinocchio call sites:
   - `pinocchio/pkg/cmds/cmd.go`
   - `pinocchio/cmd/pinocchio/cmds/js.go`
4. Read the downstream backend call site:
   - `2026-03-14--cozodb-editor/backend/main.go`
5. Implement the Geppetto helper first before touching any call site.
6. Migrate Pinocchio next.
7. Only after Pinocchio compiles and the combined debug output looks correct, update the downstream backend.

Do not start by editing all callers at once. The safest path is:

1. land the new Geppetto helper,
2. switch one Pinocchio caller,
3. switch the second Pinocchio caller,
4. delete the obsolete Pinocchio trace file,
5. switch the downstream backend.

## References

Primary code references:

- `geppetto/pkg/cli/bootstrap/config.go:13-19`
- `geppetto/pkg/cli/bootstrap/profile_selection.go:48-87`
- `geppetto/pkg/cli/bootstrap/engine_settings.go:17-24`
- `geppetto/pkg/cli/bootstrap/engine_settings.go:26-58`
- `geppetto/pkg/cli/bootstrap/engine_settings.go:61-151`
- `pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go:42-178`
- `pinocchio/pkg/cmds/cmdlayers/helpers.go:165-182`
- `pinocchio/pkg/cmds/cmd.go:318-341`
- `pinocchio/cmd/pinocchio/cmds/js.go:159-180`
- `pinocchio/pkg/cmds/cmd_profile_registry_test.go:320-377`
- `pinocchio/pkg/cmds/cmd_profile_registry_test.go:380-495`
- `pinocchio/cmd/pinocchio/cmds/js_test.go:42-58`
- `2026-03-14--cozodb-editor/backend/main.go:211-299`
