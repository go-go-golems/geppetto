# Tasks

## Phase 1 — Handles + Codegen Infrastructure

- [x] 5.1 Replace `attachRef()` with `DefineDataProperty` (module.go:124-126)
- [x] 5.1 Improve error messages in `requireEngineRef()` and `requireToolRegistry()` (api.go:911-933)
- [x] 5.2 Create YAML schema `spec/js_api_codegen.yaml` with all JS-exported enums
- [x] 5.2 Create `.d.ts` template `spec/geppetto.d.ts.tmpl` (generated consts + hand-maintained API surface)
- [x] 5.2 Implement `cmd/gen-js-api` generator (YAML + template → `consts_gen.go` + `geppetto.d.ts`)
- [x] 5.2 Add `generate.go` with `//go:generate` directive, run `go generate`
- [x] 5.2 Wire `installConsts()` call into `installExports()` (module.go)
- [x] 5.2 Add validation for ToolChoice/ToolErrorHandling string values (api.go:589-590)
- [x] 5.2 Extend `cmd/gen-turns` to generate `.d.ts` for BlockKind + turns key constants
- [x] 5.2 Add `pkg/turns/spec/turns.d.ts.tmpl` and generated `pkg/doc/types/turns.d.ts` via `go generate`

## Phase 2 — Context Plumbing

- [x] 5.3a Add `ctx` parameter to `jsMiddleware()` (api.go:1257-1298)
- [x] 5.3b Add `ctx` parameter to JS tool handler wrapper (api.go:1396-1401)
- [x] 5.3c Create `session/context.go` with `WithSessionMeta()`, `SessionIDFromContext()`, `InferenceIDFromContext()`
- [x] 5.3c Inject session/inference IDs into context in `StartInference()` (session.go:228)
- [x] 5.3c Enrich tool hook payloads with session/inference IDs (api.go:750-868)

## Phase 3 — RunHandle & Streaming

- [x] 5.4 Implement `jsEventCollector` (new type implementing `events.EventSink`)
- [x] 5.4 Add `session.start()` method returning `RunHandle` with `.promise`, `.cancel()`, `.on()`
- [x] 5.4 Add per-run options (timeoutMs, tags) to `session.run()` and `session.start()`

## Phase 4 — Documentation & Polish

- [x] Update `.d.ts.tmpl` with types from phases 2-3 (MiddlewareContext, StreamEvent, RunHandle), re-run `go generate`
- [x] Add example `07_context_and_constants.js` demonstrating new features
- [x] Update `13-js-api-reference.md` with new API surface
- [x] Add CI check: `go generate ./... && git diff --exit-code` to catch stale generated files

## Done

- [x] Write codebase analysis document (design/01-js-api-improvements-codebase-analysis.md)
- [x] Update analysis to use codegen approach for enumerations and TypeScript definitions
