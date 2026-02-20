# Changelog

## 2026-02-20

- Initial workspace created
- Completed codebase analysis covering all four improvement areas (5.1-5.4)
  - 5.1: Identified `attachRef()` → `DefineDataProperty` as the fix; mapped all 6 call sites
  - 5.2: Mapped Go enum definitions to proposed `consts` export namespace; identified validation gaps
  - 5.3: Traced context drop in middleware (api.go:1285), tool handlers (api.go:1396), and tool hooks (api.go:750-868); proposed `session/context.go` for context plumbing
  - 5.4: Designed `jsEventCollector` implementing `events.EventSink`, `RunHandle` object, and `session.start()` method; recommended keeping `runAsync()` unchanged for backwards compatibility
- Created phased task list (4 phases) with dependency ordering
- Wrote comprehensive TypeScript definition sketch covering full API surface
- Updated 5.2 to use code-generation approach:
  - New YAML schema `js_api_codegen.yaml` as single source of truth for JS-exported enums
  - New `cmd/gen-js-api` generator producing both `consts_gen.go` and `geppetto.d.ts`
  - Hybrid `.d.ts.tmpl` approach: generated enum/const types + hand-maintained API surface interfaces
  - Modeled after existing `cmd/gen-turns` infrastructure
- Updated TypeScript definitions section to hybrid codegen approach
  - Template skeleton (`geppetto.d.ts.tmpl`) with `{{ range .Enums }}` blocks for consts
  - Remaining API surface types (interfaces, function signatures) hand-maintained in template
  - Rejected full Go→TS codegen as over-engineered for ~30 stable interface definitions
- Updated implementation order table with codegen-specific phases and new file listing
- Updated task list with granular codegen subtasks
- **Implemented 5.1: Opaque handles + better error messages**
  - `attachRef()` now uses two-step Set + DefineDataProperty (non-writable, non-enumerable, non-configurable)
  - Discovered that `m.vm.ToValue(ref)` wraps Go struct pointers into goja proxies whose `Export()` returns `map[string]interface{}` — must use `o.Set()` first to preserve raw pointer, then `DefineDataProperty` with `o.Get()` to change attributes
  - Discovered that `v.Export()` strips non-enumerable properties — refactored `applyBuilderOptions()` to read engine/middlewares/tools directly from goja object
  - Improved `requireEngineRef()` and `requireToolRegistry()` error messages to include `%T` and `%v`
  - Added `TestOpaqueRefHidden` test verifying non-enumerability, JSON exclusion, non-writability, continued functionality
  - All 8 tests pass, lint clean


## 2026-02-20

Phase 5.2 complete: added JS API codegen pipeline (schema + generator + template), generated consts/.d.ts outputs, wired gp.consts exports, and validated toolChoice/toolErrorHandling values (commit ac2271685f4f12a44d55f9cac09ac60861f24e12).

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/cmd/gen-js-api/main.go — New code generator for JS constants + TypeScript definitions
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/api.go — Added strict validation for toolChoice/toolErrorHandling
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/consts_gen.go — Generated consts installer wired into module exports
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/spec/js_api_codegen.yaml — Single-source enum schema for generated JS exports


## 2026-02-20

Phase 5.3 complete: added middleware/tool handler context objects, session metadata context helpers, StartInference context injection, and hook payload session/inference enrichment (commit f2acde50e3f7758b844c35cb3fd2d24c1e2248c6). Also added two new open 5.2 tasks for generating turns/blocks key .d.ts outputs via cmd/gen-turns per user request.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/inference/session/context.go — New context helpers for session and inference identifiers
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/inference/session/session.go — Injects session/inference identifiers into run context
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/api.go — Passes context to JS middleware/tool handlers and enriches hook payloads
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/module_test.go — Validates middleware/tool/hook context payloads
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/ttmp/2026/02/20/GP-01-JS-IMPROVEMENTS--js-api-improvements-handles-constants-context-streaming/tasks.md — Added new turns/blocks .d.ts generation tasks


## 2026-02-20

Completed additional 5.2 scope: extended cmd/gen-turns to generate turns/block key TypeScript declarations via template, wired go:generate, and generated pkg/doc/types/turns.d.ts (commit 7bee52dd684c89a165a3d19e994422df333f6c3c).

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/cmd/gen-turns/main.go — Added dts section with template/out rendering path
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/doc/types/turns.d.ts — Generated turns and block key declaration file
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/generate.go — Added go:generate directive for turns.d.ts output
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/spec/turns.d.ts.tmpl — Template for generated turns/block key declarations


## 2026-02-20

Completed Phase 5.4 and final polish: implemented jsEventCollector, added session.start() RunHandle API, added run/start per-run options (timeoutMs,tags), updated generated .d.ts and JS API reference, added example 07_context_and_constants.js, and added CI generated-file freshness check (commit d9d7d52af843b054752060c4e746a2f8a612bed5).

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/.github/workflows/push.yml — CI now verifies generated files are committed
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/examples/js/geppetto/07_context_and_constants.js — Demonstrates constants + context-aware callbacks
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/doc/topics/13-js-api-reference.md — Reference updated for consts/start/run options/context payloads
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/doc/types/geppetto.d.ts — Regenerated declarations for new API surface
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/api.go — RunHandle/start implementation + jsEventCollector + run options plumbing
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl — Template updated for MiddlewareContext/RunHandle/StreamEvent


## 2026-02-20

Added a long-form bug report analyzing goja runtime thread-safety issues in geppetto async inference (runAsync/start), compared with goja/goja_nodejs and go-go-goja patterns, and documented concrete fix strategies and test plan.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/api.go — Primary source analyzed for async/race path
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/ttmp/2026/02/20/GP-01-JS-IMPROVEMENTS--js-api-improvements-handles-constants-context-streaming/analysis/01-bug-report-js-async-inference-runtime-thread-safety-runasync-start.md — New bug-report analysis document


## 2026-02-20

Expanded Option D in the async runtime safety bug report into a full architecture proposal: runtime-owner actor model, message protocol, migration slices, lifecycle/cancellation semantics, deadlock prevention, observability, performance, rollout/rollback, and acceptance criteria.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/ttmp/2026/02/20/GP-01-JS-IMPROVEMENTS--js-api-improvements-handles-constants-context-streaming/analysis/01-bug-report-js-async-inference-runtime-thread-safety-runasync-start.md — Added detailed Option D analysis and implementation blueprint


## 2026-02-20

Added an intern-facing architecture guide covering a reusable async runtime-safety runner design for go-go-goja, geppetto integration strategy, and third-party app wiring with both database and geppetto modules.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/ttmp/2026/02/20/GP-01-JS-IMPROVEMENTS--js-api-improvements-handles-constants-context-streaming/planning/01-intern-guide-reusable-async-runtime-safety-runner-go-go-goja-geppetto.md — New detailed planning/implementation guide for async safety runner and integration patterns

