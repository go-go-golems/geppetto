---
Title: Diary
Ticket: GP-010-JS-TOOLLOOP-HOOKS
Status: active
Topics:
    - geppetto
    - goja
    - javascript
    - tools
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Implementation diary for JS toolloop lifecycle hooks
LastUpdated: 2026-02-12T11:23:48.537478366-05:00
WhatFor: Track implementation and verification details for JS toolloop lifecycle hooks and hook policies.
WhenToUse: Use when reviewing before/after/error hook behavior, retry semantics, and executor wiring for GP-010.
---

# Diary

## Goal

Implement JS lifecycle hooks around toolloop execution:

- `beforeToolCall`
- `afterToolCall`
- `onToolError`

with configurable hook error handling policy and retry/abort controls.

## Step 1: Add Tool Executor Injection in Engine Builder

The existing builder pipeline needed an executor override path so JS hooks could run in tool execution.

### Changes

- Updated `pkg/inference/toolloop/enginebuilder/builder.go`:
  - new `ToolExecutor tools.ToolExecutor` field
  - runner stores executor and passes it into `toolloop.WithExecutor(...)`
- Updated `pkg/inference/toolloop/enginebuilder/options.go`:
  - added `WithToolExecutor(exec tools.ToolExecutor)`

## Step 2: Add Current Tool Call Context for Hook Visibility

`onToolError` needed tool-call details at retry decision time.

### Changes

- Updated `pkg/inference/tools/base_executor.go`:
  - added context helpers:
    - `WithCurrentToolCall(ctx, call)`
    - `CurrentToolCallFromContext(ctx)`
  - `ExecuteToolCall` now stores current call in context after `PreExecute`.

## Step 3: Implement JS Hook Parsing and Hook-Aware Executor

### Changes in `pkg/js/modules/geppetto/api.go`

- Added builder state:
  - `toolHooks`
  - `toolExecutor`
- Added builder API:
  - `withToolHooks(hooks)`
- Hook config parsing:
  - callbacks:
    - `beforeToolCall`
    - `afterToolCall`
    - `onToolError`
  - policy:
    - `failOpen` / `hookErrorPolicy` / `onHookError`
  - limit:
    - `maxHookRetries`
- Extended toolloop option mapping:
  - `toolErrorHandling`
  - `retryMaxRetries`
  - `retryBackoffMs`
  - `retryBackoffFactor`
- Added `jsToolHookExecutor` that hooks into:
  - `PreExecute` for pre-call mutation/abort
  - `PublishResult` for post-call result/error mutation
  - `ShouldRetry` for retry/abort decisions

## Step 4: Fix Runtime/Test Issues Discovered During Implementation

### Issue A

- Panic in hook parser due `.Export()` on missing JS properties.

### Fix A

- Added nil/undefined/null guards for all optional hook fields in `parseToolHooks`.

### Issue B

- Panic in `tools.extractResults` debug logging when `result` was `nil` and error non-nil.

### Fix B

- Updated `pkg/inference/tools/definition.go` to handle nil result types safely in logs (`<nil>` fallback).

## Step 5: Tests and Smoke Script

### Unit test added

- `pkg/js/modules/geppetto/module_test.go`:
  - `TestToolLoopHooksMutationRetryAbortAndHookPolicy`

Covered scenarios:

- arg rewrite via `beforeToolCall`
- result rewrite via `afterToolCall`
- retry via `onToolError`
- abort via `onToolError`
- fail-open hook errors
- fail-closed hook errors

### Smoke script added

- `geppetto/ttmp/2026/02/12/GP-010-JS-TOOLLOOP-HOOKS--js-api-toolloop-lifecycle-hooks/scripts/test_toolloop_hooks_smoke.js`

### Commands run

```bash
go test ./pkg/js/modules/geppetto -run 'TestToolLoopHooksMutationRetryAbortAndHookPolicy|TestBuilderToolsAndGoToolInvocationFromJS' -count=1 -v
go test ./pkg/inference/toolloop/... ./pkg/inference/tools/... -count=1
node geppetto/ttmp/2026/02/12/GP-010-JS-TOOLLOOP-HOOKS--js-api-toolloop-lifecycle-hooks/scripts/test_toolloop_hooks_smoke.js
go test ./pkg/js/modules/geppetto ./pkg/inference/... -count=1
```

### Outcomes

- All targeted tests passed.
- Hook smoke script output: `PASS: toolloop hooks smoke test completed`.
- Broader inference package suite passed.

## Review Pointers

- `pkg/js/modules/geppetto/api.go`
- `pkg/inference/toolloop/enginebuilder/builder.go`
- `pkg/inference/toolloop/enginebuilder/options.go`
- `pkg/inference/tools/base_executor.go`
- `pkg/inference/tools/definition.go`
- `pkg/js/modules/geppetto/module_test.go`
- `geppetto/ttmp/2026/02/12/GP-010-JS-TOOLLOOP-HOOKS--js-api-toolloop-lifecycle-hooks/scripts/test_toolloop_hooks_smoke.js`
