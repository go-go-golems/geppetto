---
Title: How to build an extensible Tool Executor for Geppetto (third‑party friendly)
Status: draft
Topics:
- geppetto
- tools
- tool-executor
- extensibility
- architecture
DocType: design
LastUpdated: 2025-10-22
---

Applying Software Architecture Document guideline

## Executive summary

We currently have two parallel implementations for executing tools: the built-in `DefaultToolExecutor` in Geppetto and an identity-aware `AuthorizedToolExecutor` living downstream. Both implement very similar orchestration (lookup, allowlist checks, event publishing, retries/backoff, parallelism) but differ in one cross-cutting concern: injecting authentication into tool arguments and masking sensitive values for events/logs.

This document proposes an Extensible Tool Executor that centralizes the common orchestration while exposing first-class, well-scoped extension points (hooks/strategies). Third-party users can inject custom behaviors—like auth argument injection, per-call logging, custom event payload formatting, and retry policies—without reimplementing the executor. Existing code keeps working with no breaking changes.

## Table of contents

1. Current state and pain points
2. Overlap analysis (what is duplicated today)
3. Design goals
4. Proposed architecture and APIs
   - Core interfaces and types
   - Hooks and strategies (pre/post-call, masking, retry, concurrency, events)
   - Functional options and composition
5. Example: Implementing an identity-aware extension (auth injection)
6. Migration and compatibility
7. Alternatives considered
8. Next steps

## 1) Current state and pain points

Geppetto ships a default executor with retries, parallelism, and event publishing:

```14:18:geppetto/pkg/inference/tools/executor.go
type ToolExecutor interface {
    ExecuteToolCall(ctx context.Context, toolCall ToolCall, registry ToolRegistry) (*ToolResult, error)
    ExecuteToolCalls(ctx context.Context, toolCalls []ToolCall, registry ToolRegistry) ([]*ToolResult, error)
}
```

It publishes pre/post execution events:

```68:71:geppetto/pkg/inference/tools/executor.go
events.PublishEventToContext(ctx, events.NewToolCallExecuteEvent(
        events.EventMetadata{},
        events.ToolCall{ID: toolCall.ID, Name: toolCall.Name, Input: argStr},
))
```

```98:101:geppetto/pkg/inference/tools/executor.go
events.PublishEventToContext(ctx, events.NewToolCallExecutionResultEvent(
        events.EventMetadata{},
        events.ToolResult{ID: toolCall.ID, Result: resultStr},
))
```

It supports bounded parallel execution and retry with backoff:

```149:169:geppetto/pkg/inference/tools/executor.go
func (e *DefaultToolExecutor) executeInParallel(ctx context.Context, toolCalls []ToolCall, registry ToolRegistry, maxParallel int) ([]*ToolResult, error) {
    results := make([]*ToolResult, len(toolCalls))
    errors := make([]error, len(toolCalls))
    sem := make(chan struct{}, maxParallel)
    var wg sync.WaitGroup
    for i, toolCall := range toolCalls {
        wg.Add(1)
        go func(index int, tc ToolCall) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            result, err := e.ExecuteToolCall(ctx, tc, registry)
            results[index] = result
            errors[index] = err
        }(i, toolCall)
    }
    wg.Wait()
    // ... error checks ...
    return results, nil
}
```

Downstream, an `AuthorizedToolExecutor` duplicates the orchestration but injects auth context into the call and masks sensitive fields before publishing events:

```26:46:go-go-mento/go/pkg/identity/tools/authorized_executor.go
func (e *AuthorizedToolExecutor) ExecuteToolCall(ctx context.Context, toolCall geptools.ToolCall, registry geptools.ToolRegistry) (*geptools.ToolResult, error) {
    start := time.Now()
    injectedCall, maskedArgs := e.injectAuthIntoCall(toolCall)
    toolDef, err := registry.GetTool(injectedCall.Name)
    if err != nil { /* ... */ }
    if !e.config.IsToolAllowed(injectedCall.Name) { /* ... */ }
    events.PublishEventToContext(ctx, events.NewToolCallExecuteEvent(events.EventMetadata{}, events.ToolCall{ID: injectedCall.ID, Name: injectedCall.Name, Input: maskedArgs}))
    result, execErr := e.executeWithRetry(ctx, injectedCall, toolDef)
    if result != nil { /* ... */ }
```

```46:62:go-go-mento/go/pkg/identity/tools/authorized_executor.go
resultPayload := ""
// marshal result and append error text if present
events.PublishEventToContext(ctx, events.NewToolCallExecutionResultEvent(
    events.EventMetadata{}, events.ToolResult{ID: injectedCall.ID, Result: resultPayload},
))
```

Auth injection itself happens by rewriting the JSON args with `person_id` and `bearer_token`:

```186:204:go-go-mento/go/pkg/identity/tools/authorized_executor.go
func (e *AuthorizedToolExecutor) injectAuthIntoCall(call geptools.ToolCall) (geptools.ToolCall, string) {
    if e.sess == nil { return call, compactJSONString(call.Arguments, true) }
    token := strings.TrimSpace(e.sess.Bearer())
    person := "" // from GetPerson/GetCurrentUser
    // ... decide person and token, then inject into args map ...
```

Pain points:
- Duplicated orchestration logic across executors
- No first-class hook point to mutate arguments before execution (e.g., inject auth, expand paths)
- No way to customize event payload masking/formatting without copy/paste
- Retry/backoff and concurrency are built-in but not swappable

## 2) Overlap analysis (duplicated concerns)

- Registry lookup and allowlist checks
- Pre-execution event publish (with optionally masked args)
- Execution with `ExecuteWithContext`
- Retry with exponential backoff
- Parallel/sequential execution with bounded concurrency
- Post-execution event publish (stringified result plus error tail)
- JSON argument marshaling and compacting for events/logs

## 3) Design goals

- Keep `ToolExecutor` interface stable
- Centralize orchestration in one executor implementation
- Provide explicit, composable extension points for third parties
- Zero-cost defaults for current behavior
- No cross-package dependency on downstream identity types

## 4) Proposed architecture and APIs

### 4.1 Core: Extensible executor

Introduce an `ExtensibleToolExecutor` that drives a pipeline:

```go
type ExtensibleToolExecutor struct {
    config           ToolConfig
    // strategies / hooks
    preHooks         []PreCallHook
    postHooks        []PostCallHook
    argMasker        ArgumentMasker
    retryPolicy      RetryPolicy
    concurrency      ConcurrencyPolicy
    authorizer       AuthorizationPolicy
    eventPublisher   EventPublisher
}

func NewExtensibleToolExecutor(cfg ToolConfig, opts ...ExecutorOption) *ExtensibleToolExecutor
```

`NewDefaultToolExecutor` wraps this with the built-in strategies to preserve current behavior.

### 4.2 Extension points

- Pre-call hook (mutate/augment calls before execution):

```go
type PreCallHook func(ctx context.Context, call ToolCall) (ToolCall, error)
// Example uses: inject auth, add tracing IDs, expand environment variables
```

- Post-call hook (observe/transform results):

```go
type PostCallHook func(ctx context.Context, call ToolCall, def *ToolDefinition, res *ToolResult, execErr error) (*ToolResult, error)
// Example uses: redact fields from results, attach metrics, translate errors
```

- Argument masker (controls what we put in `ToolCallExecute` event payload):

```go
type ArgumentMasker func(ctx context.Context, call ToolCall) string // returns compact+masked JSON string
```

- Retry policy (pluggable backoff/decision):

```go
type RetryDecision struct { Retry bool; Backoff time.Duration }
type RetryPolicy interface {
    Next(attempt int, res *ToolResult, err error) RetryDecision
}
// Default: exponential backoff using ToolConfig.RetryConfig
```

- Concurrency policy:

```go
type ConcurrencyPolicy interface { MaxParallel(calls []ToolCall) int }
// Default: fixed cfg.MaxParallelTools
```

- Authorization policy (beyond `AllowedTools`):

```go
type AuthorizationPolicy interface { IsAllowed(call ToolCall) bool }
// Default: matches existing IsToolAllowed/AllowedTools
```

- Event publisher (for pre/post events; can be swapped or disabled):

```go
type EventPublisher interface {
    PublishStart(ctx context.Context, call ToolCall, maskedArgs string)
    PublishResult(ctx context.Context, call ToolCall, res *ToolResult)
}
// Default: uses events.PublishEventToContext with ToolCallExecute / ToolCallExecutionResult
```

### 4.3 Functional options

```go
type ExecutorOption func(*ExtensibleToolExecutor)

func WithPreCallHook(h PreCallHook) ExecutorOption
func WithPostCallHook(h PostCallHook) ExecutorOption
func WithArgumentMasker(m ArgumentMasker) ExecutorOption
func WithRetryPolicy(p RetryPolicy) ExecutorOption
func WithConcurrencyPolicy(p ConcurrencyPolicy) ExecutorOption
func WithAuthorizationPolicy(p AuthorizationPolicy) ExecutorOption
func WithEventPublisher(p EventPublisher) ExecutorOption
```

`NewDefaultToolExecutor(cfg)` becomes:

```go
func NewDefaultToolExecutor(cfg ToolConfig) *DefaultToolExecutor {
    return &DefaultToolExecutor{ ExtensibleToolExecutor: NewExtensibleToolExecutor(
        cfg,
        WithArgumentMasker(defaultCompactMasker),
        WithRetryPolicy(defaultRetry(cfg.RetryConfig)),
        WithConcurrencyPolicy(staticConcurrency(cfg.MaxParallelTools)),
        WithAuthorizationPolicy(defaultAuthorizer(cfg)),
        WithEventPublisher(watermillCtxPublisher{}),
    )}
}
```

Internally `DefaultToolExecutor` just forwards to the extensible executor; public behavior stays identical.

## 5) Example: Identity-aware extension (auth injection)

Third parties can implement auth injection as a pre-call hook and a custom masker without forking the executor:

```go
// Context key for an app-defined session
type Session interface { Bearer() string; PersonID(ctx context.Context) (string, bool) }
type ctxKey struct{}

func InjectAuthFromContext() PreCallHook {
    return func(ctx context.Context, call tools.ToolCall) (tools.ToolCall, error) {
        sess, _ := ctx.Value(ctxKey{}).(Session)
        if sess == nil || strings.TrimSpace(sess.Bearer()) == "" { return call, nil }
        var args map[string]any
        _ = json.Unmarshal(call.Arguments, &args)
        if args == nil { args = map[string]any{} }
        if pid, ok := sess.PersonID(ctx); ok { args["auth"] = map[string]string{"person_id": pid, "bearer_token": sess.Bearer()} }
        b, _ := json.Marshal(args)
        call.Arguments = b
        return call, nil
    }
}

func MaskAuthArgs() ArgumentMasker {
    return func(ctx context.Context, call tools.ToolCall) string {
        var tmp map[string]any; _ = json.Unmarshal(call.Arguments, &tmp)
        if a, ok := tmp["auth"].(map[string]any); ok { a["bearer_token"] = "***" }
        b, _ := json.Marshal(tmp)
        return string(b)
    }
}

exec := tools.NewExtensibleToolExecutor(
    tools.DefaultToolConfig(),
    tools.WithPreCallHook(InjectAuthFromContext()),
    tools.WithArgumentMasker(MaskAuthArgs()),
)
```

This replicates the behavior of `AuthorizedToolExecutor` while reusing core orchestration.

## 6) Migration and compatibility

- `ToolExecutor` interface remains unchanged
- `NewDefaultToolExecutor` continues to exist and uses the new core internally
- Third-party executors should migrate to `NewExtensibleToolExecutor` with hooks instead of copy/paste
- No runtime behavior changes for existing users unless options are provided

## 7) Alternatives considered

- Inheritance/embedding without hooks: still forces copy/paste for concerns like masking and auth injection
- Global registries (like event registry) for executor stages: harder to reason about per-instance behavior and testing compared to explicit options

## 8) Next steps

- Extract orchestration into `ExtensibleToolExecutor` and wire defaults
- Add hook/strategy interfaces and functional options
- Refactor `DefaultToolExecutor` to use the extensible core
- Add a simple `examples/` program showing an auth injector built as a pre-hook
- Update docs (`geppetto-tools`) with a short “Extending tool execution” section

## Appendix: Additional code references

Retry/backoff loop in default executor:

```195:207:geppetto/pkg/inference/tools/executor.go
for attempt := 0; attempt <= e.config.RetryConfig.MaxRetries; attempt++ {
    if attempt > 0 {
        backoff := time.Duration(float64(e.config.RetryConfig.BackoffBase) *
            pow(e.config.RetryConfig.BackoffFactor, float64(attempt-1)))
        select {
        case <-ctx.Done(): /* ... */
        case <-time.After(backoff):
        }
    }
    // execute...
}
```


