# Changelog

## 2026-02-25

- Added runtime stack-binding APIs to JS `gp.profiles` namespace:
  - `connectStack(sources)` for loading profile registries from YAML/SQLite sources at runtime.
  - `disconnectStack()` for explicit teardown (restores host-provided registry when one was injected in module options).
  - `getConnectedSources()` for stack inspection/debugging.
- Added regression tests covering:
  - YAML stack precedence and lifecycle (`connectStack` -> `resolve` -> `disconnectStack`),
  - writable sqlite CRUD through runtime-connected stacks.
- Updated JS type declarations (`geppetto.d.ts`) for new stack APIs and source/result types.
- Added runnable script example `examples/js/geppetto/19_profiles_connect_stack_runtime.js` and wired it into the profile-registry example suite runner.
- Re-validated:
  - `go test ./pkg/js/modules/geppetto -count=1`
  - `./examples/js/geppetto/run_profile_registry_examples.sh`

- Updated profile-registry docs and runtime loading notes to reflect current pinocchio behavior:
  - fallback default source: `${XDG_CONFIG_HOME:-~/.config}/pinocchio/profiles.yaml` when present,
  - explicit source precedence (`flag > env > config > default-file-if-present`),
  - clarified runtime YAML shape (`slug` + `profiles`, no `registries:` / `default_profile_slug`).
- Updated user-facing pinocchio README with profile-registry loading and legacy conversion guidance.
- Updated geppetto AI flag help text to remove legacy `--profile-file` references.
- Added/validated section middleware fallback behavior enabling:
  - `PINOCCHIO_PROFILE=<slug>` with default `~/.config/pinocchio/profiles.yaml` runtime registry.

## 2026-02-25

- Rebased GP-21 design/tasks to GP-31 runtime surface:
  - stack-first profile resolution across `--profile-registries`,
  - no runtime registry selector path for JS runtime APIs.
- Updated final design doc to remove runtime `registry`/`registrySlug` selectors from factory/runtime inputs.
- Added explicit follow-up tasks to remove `registrySlug` from `ProfileEngineOptions` and align cookbook/examples with stack-only runtime selection.
- Updated index metadata/status to mark GP-21 as implementation-pending but GP-31-aligned.

## 2026-02-25

- Completed deep codebase analysis for profile registry + middleware/extension schema parity against JS bindings.
- Added unified inference-first final API research document that merges GP-21 findings with OS-09 comprehensive JS API design.
- Updated the final design to hard cutover mode: removed legacy compatibility recommendation for `engines.fromProfile` and made registry-first behavior mandatory.
- Added dedicated reference document `02-geppetto-js-api-scripts-cookbook-old-and-new.md` with 30 script examples spanning current and new hard-cutover JS APIs.
- Added new reproducible experiments:
  - `scripts/inspect_inference_surface.js`
  - `scripts/inspect_from_profile_semantics.js`
- Captured additional runtime evidence outputs:
  - `various/inspect_inference_surface.out`
  - `various/inspect_from_profile_semantics.out`
- Added reproducible ticket-local experiments:
  - `scripts/inspect_geppetto_exports.js`
  - `scripts/inspect_geppetto_plugins_exports.js`
- Captured runtime evidence outputs under `various/` proving missing `profiles`/`schemas` JS namespaces.
- Authored design doc with evidence-backed architecture map, gaps, pseudocode, and phased implementation plan.
- Authored detailed chronological diary with commands, one failed attempt, and resolution trail.
- Completed docmgr bookkeeping/validation and uploaded verified reMarkable bundle at `/ai/2026/02/25/GP-21-PROFILE-MW-REGISTRY-JS`.

## 2026-02-24

- Initial workspace created.

## 2026-02-24

Completed deep parity research for Go profile-registry + schema capabilities vs JS bindings, including reproducible export-inventory experiments and a phased implementation plan.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/design-doc/01-profile-registry-middleware-schema-parity-analysis-for-js-bindings.md — Primary architecture and gap analysis deliverable
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/reference/01-investigation-diary.md — Chronological command-level evidence log
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_geppetto_exports.js — Runtime proof of top-level JS API surface
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_geppetto_plugins_exports.js — Runtime proof of plugin-module surface

## 2026-02-24

Added final merged inference-first JS API research doc combining GP-21 parity findings with OS-09 API design; added runtime experiments for inference surface and fromProfile semantics

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/design-doc/02-unified-final-js-api-design-inference-first.md — Primary final-design deliverable before implementation
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_from_profile_semantics.js — Evidence script for engines.fromProfile behavior
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_inference_surface.js — Evidence script for current inference API

## 2026-02-24

Applied hard-cutover directive: removed legacy compatibility recommendation and made engines.fromProfile registry-first in final design guidance

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/design-doc/01-profile-registry-middleware-schema-parity-analysis-for-js-bindings.md — Supersession note pointing to hard-cutover final doc
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/design-doc/02-unified-final-js-api-design-inference-first.md — Hard-cutover final recommendation and implementation phases
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/reference/01-investigation-diary.md — Step-14 record of hard-cutover user directive

## 2026-02-24

Added comprehensive JS script cookbook document with broad current API coverage and hard-cutover target examples; prepared artifact for reMarkable delivery

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/reference/02-geppetto-js-api-scripts-cookbook-old-and-new.md — Primary cookbook deliverable with 30 example scripts
