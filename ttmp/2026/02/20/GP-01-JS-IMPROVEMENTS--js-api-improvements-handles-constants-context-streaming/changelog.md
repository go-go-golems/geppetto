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

