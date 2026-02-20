---
Title: Detailed analysis and mechanical split execution plan for geppetto api.go
Ticket: GP-15-JS-API-MECHANICAL-SPLIT
Status: active
Topics:
    - architecture
    - geppetto
    - go
    - inference
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/js/modules/geppetto/api_engines.go
      Note: Engine/profile/config move target
    - Path: pkg/js/modules/geppetto/api_middlewares.go
      Note: Middleware adapter move target
    - Path: pkg/js/modules/geppetto/api_sessions.go
      Note: Primary session/builder/run lifecycle split target
    - Path: pkg/js/modules/geppetto/api_tool_hooks.go
      Note: Tool hook executor move target
    - Path: pkg/js/modules/geppetto/api_tools_registry.go
      Note: Tool registry move target
ExternalSources: []
Summary: Detailed implementation analysis and step-by-step plan for mechanically splitting pkg/js/modules/geppetto/api.go into domain files without behavior changes.
LastUpdated: 2026-02-20T12:04:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Detailed analysis and mechanical split execution plan for geppetto `api.go`

## Goal

Mechanically split `pkg/js/modules/geppetto/api.go` into multiple smaller files in the same package, preserving behavior and exported surface exactly. This ticket does not include redesign, API changes, semantic edits, naming changes, or business-logic rewrites.

## Context and problem statement

`api.go` currently has ~2,079 lines and combines many domains:

- session object wiring and run lifecycle,
- async runtime-owner bridging,
- event collector streaming,
- builder options and tool-loop parsing,
- engine factories/profile/config translation,
- middleware adapters,
- tool registry adapters,
- tool hook executor logic,
- turns helper functions,
- many internal helper functions/types.

This file shape is difficult to review safely because unrelated changes collide in one diff and ownership boundaries are not visible at file level.

The immediate objective is to reduce cognitive load and improve maintainability by moving code into domain-focused files while preserving all symbols and behavior.

## Non-goals for this ticket

- No interface changes.
- No behavior fixes.
- No API renaming.
- No package reorganization outside `pkg/js/modules/geppetto`.
- No test logic changes except minimal compile-adjustment if needed for moved symbols.

## Mechanical split constraints

These invariants must hold after the split:

1. Same package name: all new files stay in package `geppetto`.
2. Same function/type names and signatures.
3. Same control flow and error strings.
4. Same registration/wiring from `module.go` (`installExports` call graph unchanged).
5. Same test outcomes (including race test behavior in existing suite).

## Proposed file map (mechanical)

Target files (all under `pkg/js/modules/geppetto`):

- `api_types.go`
  - internal refs/types currently declared near top of `api.go`
- `api_sessions.go`
  - session/builder constructor wiring and run lifecycle methods
- `api_owner_bridge.go`
  - `requireBridge`, `callOnOwner`, `postOnOwner`
- `api_builder_options.go`
  - builder options parsing/tool loop settings + primitive coercion helpers
- `api_tool_hooks.go`
  - hook executor and hook payload mutation helpers
- `api_engines.go`
  - engine/profile/config conversion and engine object constructors
- `api_middlewares.go`
  - middleware adapters and default Go middleware factories
- `api_tools_registry.go`
  - JS tool registry object, register/useGoTools methods
- `api_turns.go`
  - `turns.*` helper methods
- `api_events.go`
  - `jsEventCollector` functions and event encoding payload builder

The original `api.go` is expected to be removed once all functions/types are relocated.

## Exact function move inventory

Function inventory captured from current file:

- `createBuilder`, `createSession`, `runInference`, `newBuilderObject`, `buildSession`, `newSessionObject`
- `runSync`, `runAsync`, `start`, `buildRunContext`, `parseRunOptions`
- `newJSEventCollector`, `subscribe`, `close`, `PublishEvent`, `encodeEventPayload`
- `requireBridge`, `callOnOwner`, `postOnOwner`
- `applyBuilderOptions`, `applyToolLoopSettings`, `parseToolHooks`
- `hookError`, `PreExecute`, `PublishResult`, `ShouldRetry`
- helper funcs: `toBool`, `toInt`, `toString`, `toFloat64`, `decodeToolCallArgs`, `cloneJSONMap`, `addSessionMetaFromContext`, `applyCallMutation`
- engine path helpers: `requireEngineRef`, `requireToolRegistry`, `inferAPIType`, `inferAPIKeyFromEnv`, `profileFromPrecedence`, `stepSettingsFromEngineOptions`, `engineFromStepSettings`, `newEngineObject`, `engineEcho`, `engineFromProfile`, `engineFromConfig`, `engineFromFunction`
- middleware path: `middlewareFromJS`, `middlewareFromGo`, `resolveMiddleware`, `resolveGoMiddleware`, `jsMiddleware`
- tools path: `toolsCreateRegistry`, `register`, `useGoTools`
- `defaultGoMiddlewareFactories`
- turns path: `turnsNormalize`, `turnsNewTurn`, `turnsAppendBlock`, `turnsNewUserBlock`, `turnsNewSystemBlock`, `turnsNewAssistantBlock`, `turnsNewToolCallBlock`, `turnsNewToolUseBlock`

## Step-by-step execution procedure

### Step 1: Freeze baseline

- Capture current test baseline for module package.
- Confirm clean compile before splitting.

Commands:

```bash
cd geppetto
go test ./pkg/js/modules/geppetto -count=1
go test ./pkg/js/modules/geppetto -race -count=1
```

### Step 2: Create destination files and move code blocks verbatim

- Move type declarations to `api_types.go`.
- Move function blocks by domain into target files.
- Keep function bodies unchanged except import fixes and comments if required for readability.
- Delete moved blocks from `api.go`.

Important: do not interleave semantic edits while moving.

### Step 3: Remove `api.go`

- Once all symbols are present in new files and package compiles, remove old monolith file.

### Step 4: Formatting and compile validation

- `gofmt -w` all new/changed files.
- Run package tests.
- Run package race tests.

### Step 5: Ticket documentation updates

- Record exact source→destination move map in diary/changelog.
- Mark mechanical split tasks complete.

## Risk analysis

Primary risks in a mechanical split:

1. Missing helper function moved to wrong file or omitted.
2. Import drift (unused/missing imports) in new files.
3. Duplicate symbol during partial move state.
4. Hidden initialization ordering assumptions.

Mitigations:

- Move symbols in coherent domain batches.
- Compile/test after major move steps.
- Keep same package to avoid visibility changes.
- Use `rg '^func '` inventory before/after.

## Validation checklist

- [ ] `api.go` removed.
- [ ] All prior functions present in new files with identical signatures.
- [ ] `go test ./pkg/js/modules/geppetto -count=1` passes.
- [ ] `go test ./pkg/js/modules/geppetto -race -count=1` passes.
- [ ] No behavioral diffs beyond file structure movement.

## Implementation notes for reviewers

Review this as a “move-only” change:

- Focus on symbol continuity and import correctness.
- Ignore file-name churn; compare function bodies if uncertain.
- Verify no changes to exported JS API or runtime behavior.

The follow-up ticket can propose rearchitecture once this mechanical split lands safely.
