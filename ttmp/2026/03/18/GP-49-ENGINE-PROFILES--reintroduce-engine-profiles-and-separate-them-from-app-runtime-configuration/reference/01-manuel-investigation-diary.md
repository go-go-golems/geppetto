---
Title: Manuel investigation diary
Ticket: GP-49-ENGINE-PROFILES
Status: active
Topics:
    - geppetto
    - architecture
    - inference
    - profile-registry
    - config
    - javascript
    - pinocchio
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological record of the GP-49 analysis that led to the EngineProfile plus InferenceSettings split and the Glazed migration playbook."
LastUpdated: 2026-03-18T21:05:00-04:00
WhatFor: "Use this diary to understand how the GP-49 architecture direction was derived, which files were inspected, and why engine-only profiles are now preferred over mixed runtime profiles."
WhenToUse: "Use when reviewing the ticket, validating the rationale for EngineProfile plus InferenceSettings, or reconstructing the analysis and documentation work."
---

# Manuel investigation diary

## Goal

This diary records the investigation that led to the GP-49 proposal:

- bring back a Geppetto-owned engine-profile concept
- rename `StepSettings` to `InferenceSettings`
- remove mixed runtime behavior from Geppetto profiles
- document the hard cut with no compatibility layer
- add a downstream migration playbook in Glazed docs

The diary is intentionally chronological. It is meant for later review by another engineer who needs to understand why this ticket exists, not just what the final recommendation was.

## 2026-03-18 13:40 - 14:10: problem framing after Pinocchio JS inspection

The immediate trigger was the Pinocchio JS path. The example script selected a profile such as `gpt-5-mini`, but the engine bootstrap still came from `pinocchio.engines.fromDefaults(...)`. That produced two separate flows:

- profile/runtime resolution through Geppetto and Pinocchio config helpers
- engine construction through hidden base settings plus explicit JS overrides

This was the concrete signal that the current architecture was conflating two concepts:

1. "Which engine/provider/model should I use?"
2. "What prompt, middleware, tool, and runtime metadata should I apply around that engine?"

The current system treated both as "profiles" at different times, which is why the JS path felt inconsistent.

Files inspected during this phase:

- [settings-step.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/settings-step.go)
- [api_runner.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runner.go)
- [api_runtime_metadata.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go)
- [profile_runtime.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/profile_runtime.go)
- [module.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/js/modules/pinocchio/module.go)

## 2026-03-18 14:10 - 14:35: revisiting GP-43 critically

I evaluated whether GP-43 had simply been wrong. The conclusion was more specific:

- removing `runtime.step_settings_patch` was still correct
- but removing the patch mechanism did not answer the separate question of how Geppetto should model engine presets

The old mixed model had one real benefit: it gave Geppetto a reusable way to express engine configuration in profile registries. That benefit was hidden inside a poor abstraction.

So the right correction is not "undo GP-43" and not "push everything to app code." The right correction is:

- restore an engine-only profile abstraction in Geppetto
- keep app runtime behavior outside Geppetto

This is the core design pivot for GP-49.

## 2026-03-18 14:35 - 15:10: establishing the split

I mapped the current responsibilities and forced each concern into either "Geppetto core" or "application layer."

Geppetto-owned concerns:

- provider selection
- model selection
- client configuration
- inference parameters
- engine creation from those settings
- registry loading and stacking for engine presets

Application-owned concerns:

- system prompts
- middleware selection
- tool registries and tool filtering
- runtime keys and runtime fingerprints
- session and conversation policy
- event and cache semantics

This split held up across:

- Go runner APIs
- JS runner APIs
- Pinocchio CLI and webchat bootstrap

That was the strongest sign that the split is architectural, not incidental.

## 2026-03-18 15:10 - 15:45: naming decision

I reviewed the legacy `StepSettings` name in the live engine path. The object is still the primary configuration object for building provider engines, but the old "step" terminology comes from an older lifecycle model that no longer reflects the code well.

Options considered:

- `EngineSettings`
- `InferenceSettings`
- `ModelSettings`
- `LLMSettings`

The user explicitly preferred `InferenceSettings`, and that choice also fits the scope best because the object includes:

- provider settings
- model settings
- client settings
- inference tuning settings

So the ticket now assumes a hard rename:

- `StepSettings` -> `InferenceSettings`

No wrappers. No aliases. No transition package.

## 2026-03-18 15:45 - 16:20: designing the new profile model

I then translated the split into a concrete data model.

Proposed Geppetto concepts:

- `InferenceSettings`
- `EngineProfile`
- `EngineProfileRegistry`
- `ResolvedEngineProfile`

Important negative decision:

- do not recreate `StepSettingsPatch` under a new name
- do not use generic patch maps
- do not keep a `runtime:` section in Geppetto engine profile YAML

Instead, the profile format should describe engine configuration only.

That means no fields for:

- `system_prompt`
- `middlewares`
- `tools`
- runtime keys or runtime fingerprints

Those move fully to app code.

## 2026-03-18 16:20 - 16:50: writing the ticket design doc

I wrote the main design document as an intern-oriented guide rather than a terse architecture memo. The goal was to make the future implementation straightforward for a new engineer.

The guide includes:

- current-state analysis
- gap analysis
- target architecture
- YAML format proposal
- API proposals
- pseudocode
- migration sequencing
- testing requirements
- rationale for the `InferenceSettings` rename

The final design doc is:

- [01-engine-profiles-architecture-and-migration-guide.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/18/GP-49-ENGINE-PROFILES--reintroduce-engine-profiles-and-separate-them-from-app-runtime-configuration/design-doc/01-engine-profiles-architecture-and-migration-guide.md)

## 2026-03-18 16:50 - 17:10: adding the migration-playbook requirement

The user specifically wanted a migration playbook in Glazed docs, not just ticket-local design notes.

I checked existing Glazed help-entry conventions and existing migration tutorial style:

- [SKILL.md](/home/manuel/.codex/skills/glazed-help-page-authoring/SKILL.md)
- [migrating-to-facade-packages.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/glazed/pkg/doc/tutorials/migrating-to-facade-packages.md)

That existing tutorial confirmed the preferred style:

- explicit hard-cut wording
- no compatibility sugar
- step-by-step checklist
- import/type/file migration guidance

Based on that, GP-49 now explicitly requires a Glazed tutorial-style migration playbook as part of rollout.

## 2026-03-18 17:10 - 18:05: writing the Glazed migration playbook and closing the documentation pass

I added the first version of the migration playbook directly under Glazed docs:

- [migrating-from-mixed-runtime-profiles-to-engine-profiles.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/glazed/pkg/doc/tutorials/migrating-from-mixed-runtime-profiles-to-engine-profiles.md)

This matters because downstream engineers are more likely to find a tutorial in Glazed docs than a design note buried in a Geppetto ticket workspace.

I then updated the GP-49 task board and changelog so the ticket accurately reflects:

- what is already documented
- what is still planned
- which migration artifact now exists outside the ticket

## 2026-03-18 18:15 - 18:30: clarifying what happens to the current registry subsystem

The next question was whether the current `pkg/profiles` code should be deleted wholesale. I checked the live package structure and separated the value-bearing mechanics from the mixed-model payload semantics.

Files inspected in this pass:

- [registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/registry.go)
- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/types.go)
- [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/service.go)
- [source_chain.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/source_chain.go)
- [sqlite_store.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/sqlite_store.go)

The conclusion was:

- keep registry loading and stacked-read mechanics
- replace `Profile`, `RuntimeSpec`, and `ResolvedProfile`
- decide explicitly whether the package remains `pkg/profiles` or becomes `pkg/engineprofiles`

That distinction is now captured in the main design doc so the ticket is no longer ambiguous about whether "profiles" means delete everything or narrow the subsystem.

## 2026-03-18 18:35 - 18:50: choosing the implementation order

Once the design was stable, I mapped the implementation blast radius. Two rename-heavy changes dominate:

- `pkg/profiles` -> `pkg/engineprofiles`
- `StepSettings` -> `InferenceSettings`

I decided to do the package rename first for a practical reason: it changes import boundaries without changing payload semantics. That gives a clean first slice with lower conceptual risk.

Then the `InferenceSettings` rename can happen on top of the new package name, instead of mixing package and type renames into one harder-to-review commit.

So the execution order for GP-49 is now:

1. package rename
2. settings rename
3. engine-profile API rename
4. runtime-payload deletion
5. YAML/codec migration
6. downstream runtime cleanup

I updated the task board to reflect that exact sequence.

## 2026-03-18 18:50 - 19:05: Slice 1 implementation - `pkg/profiles` to `pkg/engineprofiles`

I started implementation with the narrowest hard cut: rename the package first and keep behavior stable.

Actions taken:

- moved `geppetto/pkg/profiles` to `geppetto/pkg/engineprofiles`
- renamed `package profiles` to `package engineprofiles`
- rewrote import paths across:
  - Geppetto
  - Pinocchio
  - GEC-RAG
  - Temporal Relationships
- fixed files that still referenced `profiles.*` by aliasing the renamed package import where needed

Validation run for this slice:

- `cd geppetto && go test ./pkg/engineprofiles/... ./pkg/js/modules/geppetto ./cmd/examples/geppetto-js-lab -count=1`
- `cd pinocchio && go test ./cmd/pinocchio/cmds ./pkg/cmds/helpers ./cmd/web-chat/... ./pkg/webchat/... -count=1`
- `cd 2026-03-16--gec-rag && go test ./internal/webchat/... -count=1`
- `cd temporal-relationships && go test ./internal/extractor/httpapi/... ./internal/extractor/gorunner/... -count=1`

All of those passed.

Ticket maintenance after the rename:

- updated related file paths from `pkg/profiles/...` to `pkg/engineprofiles/...`
- marked Slice 1 done in the task board
- prepared the ticket for the next slice: `StepSettings` to `InferenceSettings`

## 2026-03-18 19:05 - 20:05: Slice 2 implementation - `StepSettings` to `InferenceSettings`

The second slice was the hard naming cut. The goal was to remove `StepSettings` from the live codebase, not just add a new alias that would keep the old term around.

I started by moving the main Geppetto settings file:

- [settings-step.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/settings-step.go) -> [settings-inference.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/settings-inference.go)

Then I hard-renamed the primary symbols:

- `StepSettings` -> `InferenceSettings`
- `NewStepSettings` -> `NewInferenceSettings`
- `NewStepSettingsFromYAML` -> `NewInferenceSettingsFromYAML`
- `NewStepSettingsFromParsedValues` -> `NewInferenceSettingsFromParsedValues`
- `NewEngineFromStepSettings` -> `NewEngineFromSettings`

The first real compile break after the broad rename pass was inside the settings package itself. The file move had left the inference section helper absent. I restored that directly in [settings-inference.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/settings-inference.go):

- embedded `flags/inference.yaml`
- reintroduced `InferenceValueSection`
- restored `AiInferenceSlug`
- reintroduced `NewInferenceValueSection(...)`

That gave me a stable inner compile ring:

- `cd geppetto && go test ./pkg/steps/ai/settings ./pkg/inference/engine/factory ./pkg/inference/runner ./pkg/sections -count=1`

Once that passed, I cleaned the remaining rename fallout in the application repos:

- Pinocchio bootstrap errors, comments, and tests
- GEC-RAG bootstrap and resolver wording
- Temporal Relationships bootstrap wording
- example helper files and filenames that still used `step_settings.go`

I also renamed live helper filenames that still encoded the legacy term:

- [cmd/examples/internal/runnerexample/inference_settings.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/internal/runnerexample/inference_settings.go)
- [internal/cmdruntime/inference_settings.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/cmdruntime/inference_settings.go)

Validation for this slice was intentionally broad:

- `cd geppetto && go test ./...`
- `cd pinocchio && go test ./cmd/pinocchio/... ./cmd/web-chat/... ./pkg/... ./scripts/...`
- `cd 2026-03-16--gec-rag && go test ./internal/...`
- `cd temporal-relationships && go test ./internal/...`

All of these passed.

One operational note: the rename touched a large amount of historical `ttmp/` material. I deliberately left those files unstaged because they are not part of the live code slice and would create a noisy commit. The live-code rename itself is validated and ready for separate commits.

## 2026-03-18 20:10 - 21:05: Slice 3 implementation - engine-profile API rename

This slice renamed the public resolution surface inside the newly moved package. The goal was to make the live API match the architecture language from the ticket before deleting the remaining mixed runtime payload in Slice 4.

Core symbols changed in [pkg/engineprofiles](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles):

- `Profile` -> `EngineProfile`
- `ProfileRegistry` -> `EngineProfileRegistry`
- `ResolvedProfile` -> `ResolvedEngineProfile`
- `ProfileSlug` -> `EngineProfileSlug`
- `ProfileRef` -> `EngineProfileRef`
- `ResolveEffectiveProfile(...)` -> `ResolveEngineProfile(...)`
- `ListProfiles(...)` -> `ListEngineProfiles(...)`
- `GetProfile(...)` -> `GetEngineProfile(...)`
- parsing helpers such as `ParseProfileSlug(...)` -> `ParseEngineProfileSlug(...)`

I used a broad mechanical rename pass inside the Geppetto package first, then cleaned the fallout manually. The main failure mode was doubled names like `ParseEngineEngineProfileSlug`, which I corrected by hand before validating the package in isolation.

The first safe inner-ring validation for this slice was:

- `cd geppetto && go test ./pkg/engineprofiles -count=1`
- `cd geppetto && go test ./pkg/sections ./pkg/js/modules/geppetto ./cmd/examples/internal/runnerexample -count=1`

After the core package compiled again, I updated the downstream callers.

Pinocchio fixes:

- aligned [js.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/cmd/pinocchio/cmds/js.go) with `gp.Options.EngineProfileRegistry`
- updated helper and webchat code to use `ResolveInput.EngineProfileSlug` and `ResolveEngineProfile(...)`
- fixed the legacy migration command to use Pinocchio's own default config path convention rather than a removed Geppetto helper

GEC-RAG fixes:

- updated the webchat resolver and tests in [resolver.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/resolver.go) and [resolver_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/resolver_test.go)
- renamed the registry-loading helper calls in [profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/profiles.go)

Temporal Relationships fixes:

- updated run-chat and gorunner registry resolution to use `ParseEngineProfileRegistrySourceEntries(...)`, `ParseEngineProfileSlug(...)`, and `ResolveEngineProfile(...)`
- fixed resolved-profile field access such as `resolved.EngineProfileSlug`

I also cleaned the remaining Geppetto example fallout:

- [geppetto-js-lab/main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/geppetto-js-lab/main.go)
- [runner-registry/main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-registry/main.go)
- [runner-glazed-registry-flags/main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/cmd/examples/runner-glazed-registry-flags/main.go)

Broad validation for Slice 3:

- `cd geppetto && go test ./cmd/... ./pkg/...`
- `cd pinocchio && go test ./pkg/ui/profileswitch ./pkg/webchat/http ./cmd/web-chat ./cmd/pinocchio/cmds ./pkg/cmds/helpers ./scripts/... -count=1`
- `cd 2026-03-16--gec-rag && go test ./internal/...`
- `cd temporal-relationships && go test ./internal/...`

All of these passed.

This leaves the package/API terminology aligned for the next structural cut. The remaining conceptual mismatch is that `EngineProfile` still carries `RuntimeSpec`, `EffectiveRuntime`, and runtime fingerprinting. Slice 4 is now the point where that mixed-model payload finally gets removed.

## 2026-03-18 15:30 - 16:10 Hard-cut engine-only semantics in Geppetto

I implemented the first real semantic cut after the earlier rename slices.

The goal for this pass was narrow and explicit:

- finish the Geppetto-side hard cut first
- do not preserve the old mixed runtime model
- accept that downstream applications can be migrated afterward

### What changed in core `engineprofiles`

I removed the remaining mixed runtime semantics from:

- [types.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/types.go)
- [service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/service.go)
- [stack_merge.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/stack_merge.go)
- [validation.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/engineprofiles/validation.go)

The key architectural result is:

```text
ResolveEngineProfile(...)
  -> registrySlug
  -> profileSlug
  -> InferenceSettings
  -> profile metadata
```

not:

```text
ResolveEngineProfile(...)
  -> effectiveRuntime
  -> runtimeKey
  -> runtimeFingerprint
  -> prompts / middlewares / tools
```

I also deleted the leftover helper code that only existed for the mixed runtime design:

- `middleware_extensions.go`
- `stack_trace.go`

### What changed in the JS API

The JS layer was the main place where the old mixed model was still visible.

I changed it so that:

- `gp.profiles.resolve(...)` now returns engine-only data:
  - `registrySlug`
  - `profileSlug`
  - `inferenceSettings`
  - `metadata`
- `gp.engines.fromResolvedProfile(...)` and `gp.engines.fromProfile(...)` are the engine-profile entry points
- `gp.runner.resolveRuntime(...)` rejects `profile` input and only accepts app-owned runtime fields
- builder/session APIs no longer accept `resolvedProfile` or `useResolvedProfile(...)`

That gives a clean JS mental model:

```text
profiles -> engine settings
engines  -> build engine
runner   -> app-owned runtime behavior
```

### A small but important detail

The raw Go `InferenceSettings` struct does not carry useful JSON tags for the nested shape, so simply exposing the Go struct into JS produced bad property names like `Chat` instead of `chat`.

To fix that, I added YAML-backed encoding in:

- [api_runtime_metadata.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go)

so resolved JS profile objects expose `inferenceSettings` in the same lower-case shape as the YAML examples.

### Example and docs cleanup

I rewrote the shipped example registries under:

- [examples/js/geppetto/profiles/10-provider-openai.yaml](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/profiles/10-provider-openai.yaml)
- [examples/js/geppetto/profiles/20-team-agent.yaml](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/profiles/20-team-agent.yaml)
- [examples/js/geppetto/profiles/30-user-overrides.yaml](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/profiles/30-user-overrides.yaml)

They now use `inference_settings` only.

I also rewrote the JS examples and docs that were still teaching:

- `effectiveRuntime`
- profile-derived runtime keys
- `useResolvedProfile(...)`
- `runner.resolveRuntime({ profile: ... })`

Main doc files updated:

- [01-profiles.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/01-profiles.md)
- [13-js-api-reference.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md)
- [14-js-api-user-guide.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/14-js-api-user-guide.md)

### Validation

The validation sequence for this hard cut was:

- `cd geppetto && go test ./pkg/engineprofiles/...`
- `cd geppetto && go test ./pkg/js/modules/geppetto -count=1`
- `cd geppetto && go test ./cmd/examples/geppetto-js-lab -count=1`
- `cd geppetto && go test ./cmd/... ./pkg/...`
- `cd geppetto && ./.bin/golangci-lint run ./cmd/... ./pkg/...`

All of those passed after the final cleanup.

### Result

Geppetto is now on the intended stable footing:

- engine profiles are engine-only
- runtime behavior is no longer a Geppetto profile concern
- JS reflects that split directly

The next remaining work under this ticket is downstream migration, not more Geppetto core simplification.

### Review packet refresh

I replaced the earlier short PR 308 review note with a much more explicit review packet in:

- [02-pr-308-review-guide-for-tired-reviewer.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/18/GP-49-ENGINE-PROFILES--reintroduce-engine-profiles-and-separate-them-from-app-runtime-configuration/design-doc/02-pr-308-review-guide-for-tired-reviewer.md)

The new version is intentionally written for a tired reviewer who does not want to reconstruct the migration from memory. It now explains:

- the architectural rule behind the whole line of work
- the ticket and commit map
- the final public contracts in Geppetto and JavaScript
- the downstream migration story in Pinocchio, CoinVault, and Temporal Relationships
- before/after pseudocode
- the highest-signal files to inspect
- the specific risk areas to pay more attention to
- the remaining deferred boundary in `GP-51`

That document should now be sufficient as a stand-alone review packet for the migration line.

## Quick Reference

Architecture rule:

```text
Geppetto owns engine configuration.
Applications own runtime behavior.
```

Hard-cut rename:

```text
StepSettings -> InferenceSettings
```

Target Geppetto profile concepts:

- `EngineProfile`
- `EngineProfileRegistry`
- `ResolvedEngineProfile`

Concepts that should not remain in Geppetto core profile resolution:

- system prompt
- middleware lists
- tool-name policy
- runtime keys
- runtime fingerprints

## Related

- [index.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/18/GP-49-ENGINE-PROFILES--reintroduce-engine-profiles-and-separate-them-from-app-runtime-configuration/index.md)
- [tasks.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/18/GP-49-ENGINE-PROFILES--reintroduce-engine-profiles-and-separate-them-from-app-runtime-configuration/tasks.md)
- [changelog.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/18/GP-49-ENGINE-PROFILES--reintroduce-engine-profiles-and-separate-them-from-app-runtime-configuration/changelog.md)
