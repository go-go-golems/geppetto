---
Title: Middlewares in Geppetto (Turn-based)
Slug: geppetto-middlewares
Short: A practical guide to writing, composing, and using middlewares with Turn-based engines, including logging and tool execution.
Topics:
- geppetto
- middlewares
- turns
- tools
- architecture
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

# Middlewares in Geppetto (Turn-based)

## Why Middlewares?

Middlewares let you add behavior **around** inference calls without modifying the engine itself. They're the standard pattern for:

- **Logging** — Record every inference call with timing and block counts
- **Tool execution** — Detect `tool_call` blocks, run tools, append results
- **Safety filters** — Block harmful requests before they reach the provider
- **Tracing** — Add correlation IDs for distributed tracing
- **Rate limiting** — Throttle requests per user or globally

Middlewares compose cleanly: wrap an engine once, and all calls to `RunInference` pass through the chain.

```
Request → [Logging] → [Tool Execution] → [Engine] → Response
                                              ↓
                     [Logging] ← [Tool Execution] ←
```

## What you'll learn

- The middleware interface and how it composes
- How to write a simple logging middleware
- How to use the built-in tool middleware
- How to attach middlewares to engines

---

## Core interfaces

```go
package middleware

import "context"

type HandlerFunc func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
type Middleware  func(HandlerFunc) HandlerFunc

// EngineWithMiddleware wraps an engine so that calls to RunInference pass through the chain.
func NewEngineWithMiddleware(e engine.Engine, mws ...Middleware) engine.Engine { /* ... */ }
```

Conceptually, a middleware takes a `HandlerFunc` (the next step) and returns a new `HandlerFunc` that adds behavior before and/or after calling `next`.

---

## Example: Logging middleware

```go
logMw := func(next middleware.HandlerFunc) middleware.HandlerFunc {
    return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
        logger := log.With().Int("block_count", len(t.Blocks)).Logger()
        logger.Info().Msg("Starting inference")
        res, err := next(ctx, t)
        if err != nil {
            logger.Error().Err(err).Msg("Inference failed")
        } else {
            logger.Info().Int("result_block_count", len(res.Blocks)).Msg("Inference completed")
        }
        return res, err
    }
}

e := middleware.NewEngineWithMiddleware(baseEngine, logMw)
```

---

## Example: Tool middleware

The tool middleware detects `tool_call` blocks and executes tools via a `Toolbox`, then appends matching `tool_use` blocks. It loops up to `MaxIterations` or until there are no pending calls.

```go
tb := middleware.NewMockToolbox()
tb.RegisterTool("echo", "Echo back the input text", map[string]any{
    "text": {"type": "string"},
}, func(ctx context.Context, args map[string]any) (any, error) {
    return args["text"], nil
})

cfg := middleware.ToolConfig{ MaxIterations: 5 }
toolMw := middleware.NewToolMiddleware(tb, cfg)
e := middleware.NewEngineWithMiddleware(baseEngine, toolMw)
```

Note: Providers learn about tools to advertise from `context.Context` (tool registry) plus `Turn.Data` (tool config, e.g. `engine.KeyToolConfig`). The middleware executes tools independent of provider.

---

## Composing Multiple Middlewares

Middlewares run in the order they're provided:

```go
e := baseEngine
e = middleware.NewEngineWithMiddleware(e, logMw)
e = middleware.NewEngineWithMiddleware(e, toolMw)
// Now: RunInference -> logMw -> toolMw -> engine
```

For convenience, pass them as a slice once:

```go
e = middleware.NewEngineWithMiddleware(baseEngine, logMw, toolMw)
```

### Recommended Ordering

| Order | Middleware | Why |
|-------|-----------|-----|
| 1 | Logging | Capture all requests, including rejected ones |
| 2 | Rate limiting | Block before expensive operations |
| 3 | Safety filters | Block before reaching provider |
| 4 | Mode switching | Set context (e.g., agent mode) before tools |
| 5 | Tool execution | Execute tools, may re-invoke engine |
| 6 | (Engine) | The actual provider call |

General principle: **Middlewares that reject/filter go first; middlewares that modify/augment go last.**

---

## Guidance and best practices

- Keep middlewares stateless when possible; prefer reading/writing on the provided `*turns.Turn`
- Use provider-agnostic block semantics (`tool_call`/`tool_use`, `llm_text`) rather than parsing text
- Log with context (correlation IDs in `Turn.Metadata`), but avoid leaking sensitive data
- Ensure the middleware chain always calls `next` unless you intend to short-circuit

### Lessons learned (agent-mode and tools)

- Prefer per-Turn data hints over global state: attach small hints on `Turn.Data` using typed keys (e.g., `turns.KeyAgentMode`, `turns.KeyAgentModeAllowedTools` from `geppetto/pkg/turns`, or application-specific keys from `moments/backend/pkg/turnkeys`) to guide downstream middlewares without tight coupling. Define keys in `*_keys.go` and reuse the canonical variables everywhere else.
- Separate declarative advertisement from imperative execution: providers read a declarative registry (schemas) from `context.Context`, while execution happens via a runtime `Toolbox` (function pointers) in the tool middleware. This separation improves safety and testability.
- Reuse shared parsers/utilities: use a central YAML fenced-block parser to reliably extract structured content from LLM output instead of ad-hoc regex.
- Compose middlewares by concern: a mode middleware can set allowed tools; the tool middleware enforces the filter and handles execution; engines remain provider-focused.
- Make instructions explicit: when asking models to emit structured control output (like mode switches), provide a clear fenced YAML template and ask for long analysis when needed.

---

## Troubleshooting

- Tool calls not executing: Ensure you attached the registry to context (e.g., `ctx = toolcontext.WithRegistry(ctx, reg)`) and that the engine/provider emits `tool_call` blocks
- Infinite loops: Set `MaxIterations` and verify that new `tool_call` blocks eventually stop
- Missing results: Confirm `tool_use` blocks carry the same `id` as the originating `tool_call`

---

## Full example (logging + tools)

```go
func buildEngineWithMiddlewares(base engine.Engine, tb middleware.Toolbox) engine.Engine {
    logMw := func(next middleware.HandlerFunc) middleware.HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            log.Info().Int("blocks", len(t.Blocks)).Msg("Run start")
            res, err := next(ctx, t)
            if err != nil { log.Error().Err(err).Msg("Run error") }
            return res, err
        }
    }
    toolMw := middleware.NewToolMiddleware(tb, middleware.ToolConfig{MaxIterations: 5})
    return middleware.NewEngineWithMiddleware(base, logMw, toolMw)
}
```

## Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/inference/middleware" // Core middleware
    "github.com/go-go-golems/geppetto/pkg/inference/engine"     // Engine interface
    "github.com/go-go-golems/geppetto/pkg/turns"                // Turn/Block types
)
```

## See Also

- [Inference Engines](06-inference-engines.md) — The engines that middlewares wrap
- [Tools](07-tools.md) — Tool definitions and execution
- [Turns and Blocks](08-turns.md) — The Turn data model
- [Events](04-events.md) — Event publishing from middlewares
- Real-world examples: `geppetto/pkg/inference/middleware/agentmode/`, `geppetto/pkg/inference/middleware/sqlitetool/`
