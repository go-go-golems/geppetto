---
Title: Interface/Embedding (Inheritance-like) variant of an Extensible Tool Executor
Status: draft
Topics:
- geppetto
- tools
- tool-executor
- extensibility
- interface
- embedding
DocType: design
LastUpdated: 2025-10-22
---

Applying Software Architecture Document guideline

## Executive summary

This document presents an alternative to the options/strategy-based design from 01-…: an interface/embedding (inheritance-like) approach suitable for Go. We define a base executor struct that encapsulates orchestration (parallelism, retries, event publishing) and exposes overridable lifecycle methods through a self-referential interface. Downstream users embed the base type and override fine-grained hooks (e.g., auth injection, masking, custom retries) without re-implementing the orchestration.

## Table of contents

1. Design overview
2. Core interfaces and base executor
3. Overriding behavior via embedding
4. Example: Authorized executor (auth injection + masking)
5. Discussion: trade-offs vs options/strategy design
6. Migration and compatibility

## 1) Design overview

Go lacks class inheritance, but we can achieve inheritance-like extensibility using:

- A small interface (ToolExecutorExt) describing lifecycle hooks
- A base struct (BaseToolExecutor) that:
  - Implements default hooks
  - Owns the orchestration logic (ExecuteToolCall/ExecuteToolCalls)
  - Holds a self reference of type ToolExecutorExt to allow dynamic dispatch to overridden methods
- A downstream executor struct that embeds the base and overrides selected methods

This pattern avoids copy/paste of orchestration and enables targeted overrides.

## 2) Core interfaces and base executor

```go
package tools

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"
)

// ToolExecutorExt defines lifecycle hooks that can be overridden.
type ToolExecutorExt interface {
    // PreExecute may mutate the call (e.g., inject auth) or reject it.
    PreExecute(ctx context.Context, call ToolCall, registry ToolRegistry) (ToolCall, error)

    // IsAllowed adds authorization beyond AllowedTools/IsToolAllowed in config.
    IsAllowed(ctx context.Context, call ToolCall) bool

    // MaskArguments returns a compact and masked JSON string for event payloads.
    MaskArguments(ctx context.Context, call ToolCall) string

    // PublishStart/PublishResult control event publishing.
    PublishStart(ctx context.Context, call ToolCall, maskedArgs string)
    PublishResult(ctx context.Context, call ToolCall, result *ToolResult)

    // ShouldRetry decides retry and backoff after a failed attempt.
    ShouldRetry(ctx context.Context, attempt int, res *ToolResult, execErr error) (retry bool, backoff time.Duration)

    // MaxParallel decides concurrency for a batch.
    MaxParallel(ctx context.Context, calls []ToolCall) int
}

// BaseToolExecutor hosts orchestration and default hook implementations.
type BaseToolExecutor struct {
    ToolExecutorExt              // self reference used for dynamic dispatch
    config         ToolConfig
}

func NewBaseToolExecutor(cfg ToolConfig) *BaseToolExecutor {
    b := &BaseToolExecutor{config: cfg}
    b.ToolExecutorExt = b // default to self; outer types overwrite this
    return b
}

// Default hooks
func (b *BaseToolExecutor) PreExecute(ctx context.Context, call ToolCall, _ ToolRegistry) (ToolCall, error) {
    return call, nil
}

func (b *BaseToolExecutor) IsAllowed(_ context.Context, call ToolCall) bool {
    return b.config.IsToolAllowed(call.Name)
}

func (b *BaseToolExecutor) MaskArguments(_ context.Context, call ToolCall) string {
    // Compact JSON for events by default
    if len(call.Arguments) == 0 { return "" }
    var tmp any
    if err := json.Unmarshal(call.Arguments, &tmp); err == nil {
        if b, err := json.Marshal(tmp); err == nil { return string(b) }
    }
    return string(call.Arguments)
}

func (b *BaseToolExecutor) PublishStart(ctx context.Context, call ToolCall, masked string) {
    events.PublishEventToContext(ctx, events.NewToolCallExecuteEvent(events.EventMetadata{}, events.ToolCall{ID: call.ID, Name: call.Name, Input: masked}))
}

func (b *BaseToolExecutor) PublishResult(ctx context.Context, call ToolCall, res *ToolResult) {
    payload := ""
    if res != nil && res.Result != nil {
        if bts, err := json.Marshal(res.Result); err == nil { payload = string(bts) } else { payload = fmt.Sprintf("%v", res.Result) }
    }
    if res != nil && res.Error != "" {
        if payload == "" { payload = fmt.Sprintf("Error: %s", res.Error) } else { payload = fmt.Sprintf("%s | Error: %s", payload, res.Error) }
    }
    events.PublishEventToContext(ctx, events.NewToolCallExecutionResultEvent(events.EventMetadata{}, events.ToolResult{ID: call.ID, Result: payload}))
}

func (b *BaseToolExecutor) ShouldRetry(_ context.Context, attempt int, res *ToolResult, execErr error) (bool, time.Duration) {
    if b.config.ToolErrorHandling != ToolErrorRetry { return false, 0 }
    if attempt >= b.config.RetryConfig.MaxRetries { return false, 0 }
    // exponential backoff
    backoff := time.Duration(float64(b.config.RetryConfig.BackoffBase) * pow(b.config.RetryConfig.BackoffFactor, float64(attempt)))
    return true, backoff
}

func (b *BaseToolExecutor) MaxParallel(_ context.Context, _ []ToolCall) int {
    if b.config.MaxParallelTools <= 1 { return 1 }
    return b.config.MaxParallelTools
}

// Orchestration using dynamic dispatch to hooks
func (b *BaseToolExecutor) ExecuteToolCall(ctx context.Context, call ToolCall, registry ToolRegistry) (*ToolResult, error) {
    start := time.Now()
    // PreExecute (allow mutation)
    var err error
    call, err = b.ToolExecutorExt.PreExecute(ctx, call, registry)
    if err != nil { return &ToolResult{ID: call.ID, Error: err.Error(), Duration: time.Since(start)}, nil }

    // Lookup + allow checks
    def, err := registry.GetTool(call.Name)
    if err != nil { return &ToolResult{ID: call.ID, Error: fmt.Sprintf("tool not found: %s", call.Name), Duration: time.Since(start)}, nil }
    if !b.ToolExecutorExt.IsAllowed(ctx, call) { return &ToolResult{ID: call.ID, Error: fmt.Sprintf("tool not allowed: %s", call.Name), Duration: time.Since(start)}, nil }

    // Publish start
    b.ToolExecutorExt.PublishStart(ctx, call, b.ToolExecutorExt.MaskArguments(ctx, call))

    // Execute with retries
    var result *ToolResult
    var execErr error
    for attempt := 0; ; attempt++ {
        r, e := b.executeOnce(ctx, call, def)
        if r != nil { result = r }
        execErr = e
        if execErr == nil && (result == nil || result.Error == "") { break }
        retry, backoff := b.ToolExecutorExt.ShouldRetry(ctx, attempt, result, execErr)
        if !retry { break }
        select {
        case <-ctx.Done():
            return &ToolResult{ID: call.ID, Error: "context cancelled during retry backoff", Duration: time.Since(start)}, ctx.Err()
        case <-time.After(backoff):
        }
    }

    if result != nil { result.ID = call.ID; result.Duration = time.Since(start) }

    // Publish result
    b.ToolExecutorExt.PublishResult(ctx, call, result)
    return result, execErr
}

func (b *BaseToolExecutor) ExecuteToolCalls(ctx context.Context, calls []ToolCall, registry ToolRegistry) ([]*ToolResult, error) {
    if len(calls) == 0 { return nil, nil }
    if len(calls) == 1 { r, err := b.ExecuteToolCall(ctx, calls[0], registry); return []*ToolResult{r}, err }

    maxPar := b.ToolExecutorExt.MaxParallel(ctx, calls)
    if maxPar <= 1 { return b.executeSequential(ctx, calls, registry) }
    return b.executeParallel(ctx, calls, registry, maxPar)
}

// Internal helpers
func (b *BaseToolExecutor) executeOnce(ctx context.Context, call ToolCall, def *ToolDefinition) (*ToolResult, error) {
    select { case <-ctx.Done(): return &ToolResult{Error: "execution cancelled"}, ctx.Err(); default: }
    out, err := def.Function.ExecuteWithContext(ctx, call.Arguments)
    if err != nil { return &ToolResult{Error: err.Error()}, nil }
    return &ToolResult{Result: out}, nil
}

func (b *BaseToolExecutor) executeSequential(ctx context.Context, calls []ToolCall, registry ToolRegistry) ([]*ToolResult, error) {
    results := make([]*ToolResult, len(calls))
    for i, c := range calls {
        r, err := b.ExecuteToolCall(ctx, c, registry)
        if err != nil { return results, err }
        results[i] = r
        if r != nil && r.Error != "" && b.config.ToolErrorHandling == ToolErrorAbort {
            return results, fmt.Errorf("tool execution aborted due to error in %s: %s", c.Name, r.Error)
        }
    }
    return results, nil
}

func (b *BaseToolExecutor) executeParallel(ctx context.Context, calls []ToolCall, registry ToolRegistry, maxParallel int) ([]*ToolResult, error) {
    results := make([]*ToolResult, len(calls))
    errs := make([]error, len(calls))
    sem := make(chan struct{}, maxParallel)
    var wg sync.WaitGroup
    for i, c := range calls {
        wg.Add(1)
        go func(idx int, call ToolCall) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            r, err := b.ExecuteToolCall(ctx, call, registry)
            results[idx] = r
            errs[idx] = err
        }(i, c)
    }
    wg.Wait()
    for i, err := range errs {
        if err != nil { return results, err }
        if results[i] != nil && results[i].Error != "" && b.config.ToolErrorHandling == ToolErrorAbort {
            return results, fmt.Errorf("tool execution aborted due to error in %s: %s", calls[i].Name, results[i].Error)
        }
    }
    return results, nil
}
```

Notes:
- All orchestration calls hook methods through `b.ToolExecutorExt`, enabling dynamic dispatch to overridden methods when an outer type sets itself as the `ToolExecutorExt`.
- Public `ToolExecutor` interface remains compatible (same method set as today).

## 3) Overriding behavior via embedding

```go
// Downstream executor embeds the base and overrides selected hooks.
type MyExecutor struct {
    *BaseToolExecutor
    // custom fields…
}

func NewMyExecutor(cfg ToolConfig) *MyExecutor {
    b := NewBaseToolExecutor(cfg)
    me := &MyExecutor{BaseToolExecutor: b}
    // Important: set self-ref to outer type to enable dynamic dispatch
    b.ToolExecutorExt = me
    return me
}

// Override hooks
func (m *MyExecutor) PreExecute(ctx context.Context, call ToolCall, reg ToolRegistry) (ToolCall, error) {
    // mutate args, inject headers, etc.
    return call, nil
}

func (m *MyExecutor) MaskArguments(ctx context.Context, call ToolCall) string {
    // redact sensitive fields
    return m.BaseToolExecutor.MaskArguments(ctx, call)
}
```

## 4) Example: Authorized executor (auth injection + masking)

```go
type Session interface {
    Bearer() string
    PersonID(ctx context.Context) (string, bool)
}

type AuthorizedToolExecutor struct {
    *BaseToolExecutor
    sess Session
}

func NewAuthorizedToolExecutor(cfg ToolConfig, sess Session) *AuthorizedToolExecutor {
    b := NewBaseToolExecutor(cfg)
    a := &AuthorizedToolExecutor{BaseToolExecutor: b, sess: sess}
    b.ToolExecutorExt = a // enable overrides
    return a
}

func (a *AuthorizedToolExecutor) PreExecute(ctx context.Context, call ToolCall, _ ToolRegistry) (ToolCall, error) {
    if a.sess == nil || strings.TrimSpace(a.sess.Bearer()) == "" { return call, nil }
    var args map[string]any
    _ = json.Unmarshal(call.Arguments, &args)
    if args == nil { args = map[string]any{} }
    if pid, ok := a.sess.PersonID(ctx); ok {
        args["auth"] = map[string]string{"person_id": pid, "bearer_token": a.sess.Bearer()}
    }
    b, _ := json.Marshal(args)
    call.Arguments = b
    return call, nil
}

func (a *AuthorizedToolExecutor) MaskArguments(ctx context.Context, call ToolCall) string {
    var tmp map[string]any
    _ = json.Unmarshal(call.Arguments, &tmp)
    if auth, ok := tmp["auth"].(map[string]any); ok { auth["bearer_token"] = "***" }
    b, _ := json.Marshal(tmp)
    return string(b)
}
```

## 5) Discussion: trade-offs vs options/strategy design

- Pros
  - Simple for downstreams familiar with embedding/overrides
  - No separate strategy types to wire; override only what you need
  - Orchestration remains centralized
- Cons
  - Self-referential interface pattern requires careful constructor wiring (`b.ToolExecutorExt = outer`)
  - Less explicit composition than strategy types; hard to mix behaviors from multiple sources

If you plan to mix many orthogonal behaviors at runtime, prefer the strategy/options variant (01-…). If you want a subclass-like experience with targeted overrides, the interface/embedding pattern fits well.

## 6) Migration and compatibility

- Keep `ToolExecutor` interface as-is; implement it by embedding `BaseToolExecutor` and exposing methods
- `NewDefaultToolExecutor` can be reimplemented as a thin wrapper around `NewBaseToolExecutor`
- Downstreams currently re-implementing executors can migrate by embedding and overriding hooks


