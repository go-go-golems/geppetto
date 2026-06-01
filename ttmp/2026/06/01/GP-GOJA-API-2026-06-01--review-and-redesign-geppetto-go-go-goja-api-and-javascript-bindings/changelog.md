# Changelog

## 2026-06-01

- Initial workspace created


## 2026-06-01

Created evidence-backed review and intern-facing design guide for a Go-backed fluent Geppetto JS API, including current-state analysis, gaps, proposed agent/turn/engine/embedding/schema builders, phased implementation plan, and diary.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Primary design deliverable
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Chronological investigation diary
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/scripts/01-collect-evidence.sh — Reproducible evidence collection script


## 2026-06-01

Validated ticket with docmgr doctor, committed docs (321fb82), and uploaded design+diary bundle to reMarkable at /ai/2026/06/01/GP-GOJA-API-2026-06-01.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Included in uploaded bundle
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Included in uploaded bundle


## 2026-06-01

Revised the JS API design to a hard-cut ideal model: JavaScript manipulates direct Go-owned wrappers, hidden __geppetto_ref is only transitional implementation detail, map-first constructors leave the public contract, and unsafe raw-object import is explicitly isolated.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Hard-cut API model redesign
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Diary step recording redesign rationale


## 2026-06-01

Re-uploaded the updated hard-cut redesign bundle to reMarkable with --force at /ai/2026/06/01/GP-GOJA-API-2026-06-01.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Updated hard-cut redesign included in reMarkable bundle
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Delivery note for forced re-upload


## 2026-06-01

Clarified API naming and boundaries: profiles become inferenceProfiles only, gp.inferenceSettings builds provider/model settings, Pinocchio can back the default inference profile resolver, agents own system prompt/tools/middleware, and JS-side env/API-key methods are forbidden.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Updated naming and credential policy
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Diary step for naming clarification


## 2026-06-01

Added analysis/design guide for extracting Pinocchio inline inference profile registry/config-document behavior into Geppetto so goja JS can resolve inference settings without importing Pinocchio.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/02-reusable-geppetto-inference-profile-registry-extraction-guide.md — New profile registry extraction design doc


## 2026-06-01

Uploaded updated reMarkable bundle including the new reusable Geppetto inference profile registry extraction guide.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/02-reusable-geppetto-inference-profile-registry-extraction-guide.md — Included in updated bundle
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Delivery note for updated bundle


## 2026-06-01

Uploaded the current design+diary bundle as reMarkable v2 under /ai/2026/06/01/GP-GOJA-API-2026-06-01.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Recorded v2 upload details


## 2026-06-01

Updated the primary JS API plan to use Geppetto registry sources directly through gp.inferenceProfiles.load(...), added a detailed phased implementation task list, and uploaded just that document as reMarkable v3.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Updated v3 JS API plan and task list
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Recorded v3 upload and design pivot


## 2026-06-01

Updated the primary JS API plan to remove gp.chat(), agent.ask(), and agent.system(); system/user/multimodal content now belongs in explicit Turn objects, agent.run(turn) is the execution API, tasks.md now contains detailed phased implementation tasks, and the doc was uploaded as reMarkable v4.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Updated v4 explicit-turn API plan
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Recorded v4 update and upload
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/tasks.md — Detailed phased implementation task list


## 2026-06-01

Phase 0: added build-tagged hard-cut JS API contract test, corrected no-inferenceSettings-builder implementation plan, and recorded baseline/expected contract-test results

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/hardcut_contract_test.go — Build-tagged hard-cut API contract
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Step 12 diary entry
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/tasks.md — Phase 0 checklist marked complete


## 2026-06-01

Implemented Phase 1-3 hard-cut JS slice: registry-resolved InferenceSettings wrappers, inferenceProfiles.load/resolve, engine().inference(settings).build(), real goja tests, and example scripts

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/25_inference_profiles_load_resolve_settings.js — Example script for registry-resolved settings
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/26_engine_builder_from_registry_profile.js — Example script for engine builder
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_engine_builder.go — Engine builder integration
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_inference_profiles.go — Registry loader/resolver
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_inference_settings.go — Read-only settings wrapper and redaction


## 2026-06-01

Implemented Phase 4 explicit-turn agent API: gp.agent(), minimal gp.turn(), agent.run/stream requiring Turn wrappers, RunResult traceability, tests, and examples

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/27_agent_explicit_turn_echo.js — Runnable explicit-turn agent example
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_agent.go — Agent builder and RunResult implementation
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_turn_builder.go — Go-owned explicit turn wrapper
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/module_test.go — Phase 4 goja integration tests and example execution


## 2026-06-01

Implemented Phase 5-6: schema/tool/toolRegistry wrappers, multimodal message turns, provider profileRegistries/defaultProfile/allowRegistryLoad config, tests, and examples

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/29_tools_schema_multimodal_turn.js — Phase 5 example
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_schema_builders.go — Schema builders
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_tool_builders.go — Tool builders and tool registry
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_turn_builder.go — Multimodal message builder
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/provider/provider.go — Xgoja provider registry-load config


## 2026-06-01

Added real-provider multi-turn JS example and geppetto-js-run profile-flag runner; removed test-only real-provider smoke path

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/cmd/examples/geppetto-js-run/main.go — Profile-aware JS example runner
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/30_real_provider_multiturn.js — Real provider multi-turn example
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/run_real_provider_multiturn.sh — Convenience wrapper


## 2026-06-01

Completed Phase 7 docs/examples/declarations polish: hard-cut JS reference, user guide, tutorial, numbered examples, and tested example execution

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/hardcut/01_load_registry_resolve_profile.js — Hard-cut numbered examples
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/topics/13-js-api-reference.md — Updated hard-cut API reference
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/topics/14-js-api-user-guide.md — Updated hard-cut user guide
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/tutorials/05-js-api-getting-started.md — Updated tutorial


## 2026-06-01

Completed Phase 8 clean cutover: removed legacy JS exports/examples/tests, updated DTS/docs, and kept only wrapper-first public surface

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/README.md — Hard-cut examples index
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/module.go — Legacy exports removed
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/module_hardcut_test.go — Hard-cut public surface and example tests
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/module_test.go — Deleted legacy regression test file


## 2026-06-01

Reviewed and cleaned internal dead code after hard cutover: removed legacy profiles/runner/turns/schema adapters and trimmed engines/sessions/runtime metadata to hard-cut helpers

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_engines.go — Hard-cut engine helper rewrite
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_profiles.go — Deleted legacy profiles adapter
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_runner.go — Deleted legacy runner adapter
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_sessions.go — Internal session helper rewrite


## 2026-06-01

Pruned TypeScript declaration files to the hard-cut public API and removed stale builder/session/runner declaration shapes

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/types/geppetto.d.ts — Pruned public declaration file
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl — Pruned declaration template


## 2026-06-01

Fixed pre-commit lint fallout from the JS hard-cut cleanup: removed leftover unused middleware/turn helpers and made inference settings debug snapshots nil-safe

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_inference_settings.go — Nil-safe debug snapshot
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_middlewares.go — Removed unused legacy middleware object factories
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/codec.go — Removed unused turn-slice encoder
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/module.go — Removed unused registry ownership field


## 2026-06-01

Moved the profile-aware JS example runner from cmd/examples to examples/go to keep the ad-hoc flag-based helper outside the Glazed CLI lint scope

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/go/geppetto-js-run/main.go — Profile-aware JS example runner now lives outside cmd lint scope
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/run_real_provider_multiturn.sh — Updated runner path
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/topics/13-js-api-reference.md — Updated runner path


## 2026-06-01

Converted geppetto-js-run to a Glazed example command with an explicit run subcommand, satisfying the repository CLI lint while preserving the profile-aware JS runner

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/cmd/examples/geppetto-js-run/main.go — Glazed profile-aware JS example runner
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/examples/js/geppetto/run_real_provider_multiturn.sh — Updated to call the run subcommand
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/topics/13-js-api-reference.md — Updated runner invocation

