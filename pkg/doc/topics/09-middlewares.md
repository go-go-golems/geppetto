---
Title: Middlewares in Geppetto (Turn-based)
Slug: geppetto-middlewares
Short: A practical guide to writing, composing, and using middlewares with Turn-based engines.
Topics:
- geppetto
- middlewares
- turns
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
- **Safety filters** — Block harmful requests before they reach the provider
- **Tracing** — Add correlation IDs for distributed tracing
- **Rate limiting** — Throttle requests per user or globally

Middlewares compose cleanly: wrap an engine once, and all calls to `RunInference` pass through the chain.

```
Request → [Logging] → [Engine] → Response
                     ↓          ↓
                     [Logging] ←
```

## What you'll learn

- The middleware interface and how it composes
- How to write a simple logging middleware
- How to attach middlewares to engines

---

## Core interfaces

```go
package middleware

import "context"

type HandlerFunc func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
type Middleware  func(HandlerFunc) HandlerFunc

// Chain composes multiple middleware into a single HandlerFunc.
func Chain(handler HandlerFunc, middlewares ...Middleware) HandlerFunc { /* ... */ }
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

builder := session.NewToolLoopEngineBuilder(
    session.WithToolLoopBase(baseEngine),
    session.WithToolLoopMiddlewares(logMw),
)
```

---

## Composing Multiple Middlewares

Middlewares run in the order they're provided:

```go
e := baseEngine
builder := session.NewToolLoopEngineBuilder(
    session.WithToolLoopBase(e),
    session.WithToolLoopMiddlewares(logMw /*, sysPromptMw, ... */),
)
// Now: RunInference -> logMw -> engine
```

For convenience, pass them as a slice once:

```go
builder := session.NewToolLoopEngineBuilder(
    session.WithToolLoopBase(baseEngine),
    session.WithToolLoopMiddlewares(logMw, safetyMw),
)
```

### Recommended Ordering

| Order | Middleware | Why |
|-------|-----------|-----|
| 1 | Logging | Capture all requests, including rejected ones |
| 2 | Rate limiting | Block before expensive operations |
| 3 | Safety filters | Block before reaching provider |
| 4 | Mode switching | Set context (e.g., agent mode) before provider call |
| 5 | (Engine) | The actual provider call |

General principle: **Middlewares that reject/filter go first; middlewares that modify/augment go last.**

---

## Guidance and best practices

- Keep middlewares stateless when possible; prefer reading/writing on the provided `*turns.Turn`
- Prefer structured Turn data (blocks + typed metadata keys) over parsing raw text when possible
- Log with context (correlation IDs in `Turn.Metadata`), but avoid leaking sensitive data
- Ensure the middleware chain always calls `next` unless you intend to short-circuit

### Lessons learned

- Prefer per-Turn data hints over global state: attach small hints on `Turn.Data` using typed keys (e.g., `turns.KeyAgentMode` from `geppetto/pkg/turns`, or application-specific keys from `moments/backend/pkg/turnkeys`) to guide downstream middlewares without tight coupling. Define keys in `*_keys.go` and reuse the canonical variables everywhere else.
- Reuse shared parsers/utilities: use a central YAML fenced-block parser to reliably extract structured content from LLM output instead of ad-hoc regex.
- Compose by concern: keep provider-specific logic in engines and cross-cutting concerns (logging, validation, mode switching) in middleware.
- Make instructions explicit: when asking models to emit structured control output (like mode switches), provide a clear fenced YAML template and ask for long analysis when needed.

---

## Troubleshooting

- Middleware not running: ensure you’re using `session.NewToolLoopEngineBuilder(... session.WithToolLoopMiddlewares(...))` (or that you’re applying `middleware.Chain(...)` in your own engine adapter).
- Wrong ordering: remember `middleware.Chain(m1, m2, m3)` runs as `m1(m2(m3(next)))`.
- Nil Turn: most middleware should be defensive if `t == nil` (either treat as empty turn or error early).

---

## See Also

- [Inference Engines](06-inference-engines.md) — The engines that middlewares wrap
- [Turns and Blocks](08-turns.md) — The Turn data model
- [Events](04-events.md) — Event publishing from middlewares
- Real-world examples: `geppetto/pkg/inference/middleware/agentmode/`, `geppetto/pkg/inference/middleware/sqlitetool/`
