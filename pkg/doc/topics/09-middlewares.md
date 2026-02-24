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

## Middleware as Composable Prompting

The examples above (logging, safety, tracing) are **infrastructure middleware** — they observe or gate inference for operational concerns. But the most powerful use of middleware in Geppetto is as **composable prompting techniques**.

Most LLM frameworks treat prompt construction as a single function that builds a string. If you want a system prompt, you concatenate it. If you want tool instructions, you concatenate more. If you want mode-specific guidance, you add more text. The result is a fragile, monolithic prompt builder.

Middleware inverts this: each prompting technique is a separate, composable wrapper that adds its contribution to the Turn. Real examples in the codebase:

| Middleware | What it does | Type of change |
|-----------|-------------|----------------|
| **System prompt** | Ensures the correct system block exists; adds or replaces it | Block insertion/replacement |
| **Tool reorder** | Moves `tool_use` blocks to sit adjacent to their `tool_call` blocks | Block reordering |
| **Agent mode** | Injects mode-specific guidance blocks; parses model output for mode switches | Block insertion + output parsing |
| **SQLite tool** | Registers a database query tool into the runtime registry | Configuration change (no text change) |

Each technique is:
- **Independent** — develop and test in isolation
- **Composable** — stack with other techniques without interference
- **Observable** — tags blocks with provenance metadata (`Block.Metadata`) for debugging

Not all middleware effects are visible as text changes. Some modify Turn configuration (`Turn.Data`), register tools, or emit events. A debugging UI must surface these "invisible" changes alongside content diffs.

## What you'll learn

- The middleware interface and how it composes
- How to write middlewares that modify Turn content (not just log)
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

builder := enginebuilder.New(
    enginebuilder.WithBase(baseEngine),
    enginebuilder.WithMiddlewares(logMw),
)
```

---

## Example: Block-mutating middleware (system prompt)

Unlike the logging example above, this middleware **modifies the Turn's content** before inference — it ensures a system block is always present with the correct text:

```go
systemPromptMw := func(prompt string) middleware.Middleware {
    return func(next middleware.HandlerFunc) middleware.HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            // Check if a system block already exists
            found := false
            for i, b := range t.Blocks {
                if b.Kind == turns.BlockKindSystem {
                    // Update existing system block
                    t.Blocks[i].Payload[turns.PayloadKeyText] = prompt
                    _ = turns.KeyBlockMetaMiddleware.Set(&t.Blocks[i].Metadata, "systemprompt")
                    found = true
                    break
                }
            }
            if !found {
                // Insert system block at the beginning
                block := turns.NewSystemTextBlock(prompt)
                _ = turns.KeyBlockMetaMiddleware.Set(&block.Metadata, "systemprompt")
                turns.PrependBlock(t, block)
            }
            return next(ctx, t)
        }
    }
}
```

Note how the middleware tags the block with `KeyBlockMetaMiddleware` — this records provenance (which middleware touched this block), enabling debugging tools to show attribution.

---

## Example: Post-processing middleware (output parsing)

Middlewares can also inspect and act on the model's output **after** inference. This pattern is used by the agent-mode middleware to detect mode-switch signals in the model's response:

```go
postProcessMw := func(next middleware.HandlerFunc) middleware.HandlerFunc {
    return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
        // Call the next handler (or engine) first
        result, err := next(ctx, t)
        if err != nil {
            return result, err
        }

        // Examine model output blocks
        for _, b := range result.Blocks {
            if b.Kind == turns.BlockKindLLMText {
                text, _ := b.Payload[turns.PayloadKeyText].(string)
                // Parse structured content from model output,
                // update Turn.Data, emit events, etc.
                _ = text
            }
        }
        return result, nil
    }
}
```

This two-phase capability (pre-processing + post-processing) is what makes middleware a full prompting technique rather than just a request filter.

---

## Composing Multiple Middlewares

Middlewares run in the order they're provided:

```go
e := baseEngine
builder := enginebuilder.New(
    enginebuilder.WithBase(e),
    enginebuilder.WithMiddlewares(logMw /*, sysPromptMw, ... */),
)
// Now: RunInference -> logMw -> engine
```

For convenience, pass them as a slice once:

```go
builder := enginebuilder.New(
    enginebuilder.WithBase(baseEngine),
    enginebuilder.WithMiddlewares(logMw, safetyMw),
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

## Profile-Scoped Middleware Configuration

In current app integrations, middleware selection and config are profile-scoped runtime data:

```json
{
  "runtime": {
    "middlewares": [
      {
        "name": "agentmode",
        "id": "default",
        "config": {
          "default_mode": "financial_analyst"
        }
      }
    ]
  }
}
```

The profile controls:

- middleware ordering,
- per-instance identity (`id`),
- enabled/disabled flags,
- config payload values.

## Write-Time Validation Model

Profile write APIs validate middleware entries before persistence:

- unknown middleware names fail hard (`400` + validation error),
- config payloads are coerced and validated against middleware JSON schema,
- invalid shape/types fail hard (`400` + validation error with field path).

This avoids storing profile data that only fails later at compose-time.

## Schema Discovery for Frontends

Schema catalogs can be exposed by app APIs:

- `GET /api/chat/schemas/middlewares` returns middleware names + JSON schema payloads,
- `GET /api/chat/schemas/extensions` returns extension keys + JSON schema payloads.

Frontend editors can use these endpoints to build profile forms and validate payloads before sending CRUD writes.

---

## Troubleshooting

- Middleware not running: ensure you’re using `enginebuilder.New(... enginebuilder.WithMiddlewares(...))` (or that you’re applying `middleware.Chain(...)` in your own engine adapter).
- Wrong ordering: remember `middleware.Chain(m1, m2, m3)` runs as `m1(m2(m3(next)))`.
- Nil Turn: most middleware should be defensive if `t == nil` (either treat as empty turn or error early).
- `validation error (runtime.middlewares[*].name)`: middleware definition is not registered in the application runtime definition registry.
- `validation error (runtime.middlewares[*].config)`: payload does not satisfy the middleware JSON schema. Fetch schema from `/api/chat/schemas/middlewares` and fix payload shape/types.

---

## See Also

- [Inference Engines](06-inference-engines.md) — The engines that middlewares wrap; see "Complete Runtime Flow"
- [Turns and Blocks](08-turns.md) — The Turn data model; see "How Blocks Accumulate"
- [Sessions](10-sessions.md) — Multi-turn session management
- [Events](04-events.md) — Event publishing from middlewares
- [Structured Sinks](11-structured-sinks.md) — How middleware and sinks compose for structured output extraction
- Real-world examples: `geppetto/pkg/inference/middleware/agentmode/`, `geppetto/pkg/inference/middleware/sqlitetool/`
