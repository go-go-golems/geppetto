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

