# Inference Step Refactor – From `Message` to `Conversation`

## Purpose

The Geppetto middleware already works on `conversation.Conversation` objects, allowing chains of middleware to add tool-call messages, partial completions, and other artifacts.  **Engines still return a single `*conversation.Message`, creating an impedance mismatch.**  This refactor eliminates that gap so that every layer of the stack manipulates the same data-type.

## Background

### Current State

* **Engine API** – `RunInference(...) (*conversation.Message, error)`
* **Middleware API** – expects `(conversation.Conversation, error)`
* **Adapters** – `EngineHandler` rebuilds a conversation by appending the single response, adding unnecessary allocation & complexity.
* **Limitations** – Cannot represent tool-calls, function-calls or incremental deltas originating from engines; requires ad-hoc patches.

### Desired State

* Unified type across engines, middleware, commands and UI.
* Engines directly return the **full updated conversation**, preserving ordering and metadata.
* Middleware chains become identity-preserving transformations (no hidden cloning / stitching).

With that context, the rest of the document details the migration without any backward-compatibility shims.

Align **all** inference APIs with the middleware contract so that engines return the *entire* updated `conversation.Conversation`.  This enables:

* Tool-call support (multiple messages per request)
* Cleaner middleware chains (no extra adapter)
* Consistent data-flow across engines, UI, and command layer

Backward-compatibility shims **will not be provided** – the change is **breaking** and will ship in the next minor version.

---

## Task Matrix (4 Weeks)

| Wk | Area | Tasks |
|----|------|-------|
| 1  | **Interfaces** | ① Update `Engine` + `SimpleChatStep` signatures.<br/>② Update godocs & compile-fix. |
| 1  | **Middleware** | ③ Remove `EngineHandler` adapter.<br/>④ Remove `EngineWithMiddleware` – engines expose `WithMiddleware(...)` or `Use(...)` registration directly. |
| 2  | **Engines** | ⑤ Refactor OpenAI, Claude, Gemini engines: clone → run → append response(s) → return conversation. |
| 2  | **Steps (legacy)** | ⑥ Update `chat-step` files to new signature (OpenAI, Claude, Gemini).  Replace tool-call logic to push messages into slice. |
| 3  | **Consumers** | ⑦ Update Pinocchio cmd, ChatRunner, simple example cmds.  Extract `newMessages := result[len(original):]` and append. |
| 3  | **Tests** | ⑧ Unit: interface returns full convo.<br/>⑨ Integration: middleware passes convo unmodified except additions.<br/>⑩ Tool-call path: expect ≥ 3 new msgs. |
| 4  | **Docs & Cleanup** | ⑪ Update guide & help topics.<br/>⑫ Remove deprecated code & docs. |

---

## Interface Changes

### `engine.Engine`
```go
type Engine interface {
    // returns full updated conversation
    RunInference(ctx context.Context, msgs conversation.Conversation) (conversation.Conversation, error)
}
```

### `chat.SimpleChatStep`
```go
type SimpleChatStep interface {
    RunInference(ctx context.Context, msgs conversation.Conversation) (conversation.Conversation, error)
}
```

> **BREAKING** – All callers must handle a `Conversation`, not a single message.

---

## Implementation Hints

### Engines
```go
func (e *OpenAIEngine) RunInference(ctx context.Context, in conversation.Conversation) (conversation.Conversation, error) {
    convo := append(conversation.Conversation(nil), in...) // clone
    respMsg, err := callOpenAI(...)
    if err != nil { return nil, err }
    convo = append(convo, respMsg)
    return convo, nil
}
```
*Tool calls* should push **call**, **result**, and **final** messages before return.

### Consumers
```go
updated, _ := engine.RunInference(ctx, orig)
for _, m := range updated[len(orig):] {
    _ = manager.AppendMessages(m)
}
```

### Middleware & Engine Consolidation

#### Design
* **Remove** the separate `EngineWithMiddleware` wrapper.
* Every engine owns a `Config` containing an ordered `[]middleware.Middleware` slice.
* Engines expose:
  ```go
  func (e *OpenAIEngine) Use(m middleware.Middleware)
  ```
  or factory option `engine.WithMiddleware(middleware.Middleware)`.
* `RunInference` executes the internal chain:
  ```go
  return middleware.Chain(e.rawRun)(ctx, msgs)
  ```
  where `e.rawRun` is the original implementation.

#### Handler Example
```go
eng := factory.NewOpenAIEngine(settings)
eng.Use(loggingMiddleware)
eng.Use(tracingMiddleware)

updated, _ := eng.RunInference(ctx, msgs)
```

Middleware helpers stay in `pkg/inference/middleware`, but now operate on the engine instance directly.


---