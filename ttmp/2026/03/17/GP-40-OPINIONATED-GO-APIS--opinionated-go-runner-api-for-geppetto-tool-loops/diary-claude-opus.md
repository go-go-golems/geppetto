---
Title: "Diary: Opinionated Runner API Analysis (Claude Opus Session)"
DocType: diary
Ticket: GP-40-OPINIONATED-GO-APIS
Topics:
  - geppetto
  - pinocchio
  - api-design
Status: active
---

# Diary: Opinionated Runner API Analysis

**Session:** Claude Opus, 2026-03-17
**Author:** Claude Opus (separate diary to avoid conflicts with colleague's session)

---

## Entry 1: Initial Exploration (2026-03-17, start)

### What I did

Launched 5 parallel exploration agents to deeply analyze:
1. **geppetto/** — core architecture, turns, engine, middleware, toolloop, session, events
2. **pinocchio/** — opinionated layer, PinocchioCommand, RunContext, webchat, tools
3. **2026-03-14--cozodb-editor/** — usage patterns, pain points, structured extraction
4. **2026-03-16--gec-rag/** — CoinVault project, tool catalog, profile resolution, webchat
5. **temporal-relationships/** — batch extraction GoRunner, interactive run-chat, tool persistence

### Key Findings

**Architecture layers (bottom-up):**
- `geppetto/pkg/turns` — Turn/Block data model with typed keys
- `geppetto/pkg/inference/engine` — Provider-agnostic Engine interface: `RunInference(ctx, Turn) → Turn`
- `geppetto/pkg/inference/middleware` — Functional middleware chain: `HandlerFunc → HandlerFunc`
- `geppetto/pkg/inference/tools` — ToolRegistry, ToolDefinition, ToolExecutor with retry/parallel
- `geppetto/pkg/inference/toolloop` — Loop orchestration: inference → extract tool calls → execute → repeat
- `geppetto/pkg/inference/session` — Session lifecycle with async ExecutionHandle
- `geppetto/pkg/inference/toolloop/enginebuilder` — Builder wires engine+middleware+tools+loop+events
- `pinocchio/pkg/cmds` — PinocchioCommand, RunContext with RunMode (blocking/interactive/chat)

**Pain points identified across all 3 consumer projects:**

1. **Settings ceremony** — 20-30 lines to configure StepSettings with pointer allocations for every field
2. **Session/Turn/Builder boilerplate** — Every inference requires: NewTurnBuilder → Build → NewSession → enginebuilder.New → Append → StartInference → Wait
3. **Tool registration pattern duplication** — Same create-registry → register-tool → cleanup pattern in every project
4. **Event sink orchestration** — Manual chain construction: streaming sink → filtering sink → extractors
5. **Profile resolution complexity** — 200+ lines in gec-rag just for profile negotiation
6. **Engine creation cascade** — Multiple fallback functions for resolving engine from profiles/settings
7. **Cleanup management** — Manual reverse-iteration cleanup slices instead of defer-based patterns
8. **Context plumbing** — Manual injection/extraction of session IDs, metadata, sinks

### What was tricky

The biggest insight was realizing there are actually **two distinct use cases** that need opinionated APIs:

1. **Batch/CLI tools** — Single-shot or few-turn inference with tools (cozodb-editor, temporal-relationships GoRunner). Need: quick setup, run, get result.
2. **Interactive/Web tools** — Long-lived sessions with streaming, persistence, profile switching (gec-rag webchat, temporal-relationships run-chat). Need: conversation management, WebSocket integration, event routing.

The opinionated runner should primarily target use case #1 (batch/CLI) since that's the "scaffold a powerful tool in a couple of lines" scenario. Use case #2 builds on it but requires more infrastructure.

### Next steps

- Write the current-state analysis document
- Design 3-4 API proposals with increasing opinionation levels
- Write experiment scripts showing practical usage
- Compare trade-offs

---

## Entry 2: Analyzing Common Patterns (2026-03-17)

### What I noticed

All three consumer projects follow essentially the same skeleton:

```
1. Create StepSettings (model, API key, streaming, max tokens)
2. Build engine from settings
3. Create tool registry + register tools
4. Create enginebuilder.Builder with engine + registry + config
5. Create session with seed turn
6. Attach builder to session
7. Start inference → wait → get result
8. Extract text from result turn
```

Steps 1-8 are ~60-80 lines of boilerplate in every project. The only things that vary:
- Which model/provider
- Which tools are registered
- The system prompt
- The user prompt
- How results are processed

This is a classic case for an opinionated wrapper that handles the common path and provides escape hatches for customization.

### Design principles emerging

1. **Zero-to-working in 5 lines** — The simplest case (prompt + model → text) should be trivially easy
2. **Tools as first-class citizens** — Adding tools should be one line each
3. **Middleware composability preserved** — Don't sacrifice geppetto's middleware flexibility
4. **Escape hatches everywhere** — Users must be able to drop to lower-level APIs at any point
5. **Profile-aware by default** — Profile registries should "just work" if configured
6. **Event sinks pluggable** — Streaming/logging should be opt-in but easy

---

## Entry 3: Writing API Proposals (2026-03-17)

Starting on the design document with multiple API proposals. Will focus on:
- Design A: Minimal functional API (single function calls)
- Design B: Builder pattern with method chaining
- Design C: Struct-based configuration with functional options
- Design D: Convention-over-configuration with sensible defaults

Each will have rationale, trade-offs, and real-world examples drawn from the 3 consumer projects.

---

## Entry 4: Design Document Complete (2026-03-17)

### What I produced

Wrote `design/opinionated-runner-api-designs.md` with 4 complete API proposals:

- **Design A: Minimal Functional API** — Single `Run()` function with functional options. Like `http.Get`. Simplest possible API but no multi-turn support.
- **Design B: Builder Pattern** — `New() → *Runner` with `Run()` and `Chat()` methods. Supports multi-turn conversations and session reuse.
- **Design C: Struct Config + Options** — Serializable `Config` struct for YAML/JSON configuration, plus `Option` functions for non-serializable things (tools, sinks). Like `http.Server`.
- **Design D: Convention-over-Configuration** — Most opinionated. Auto-detects provider from env vars, short option names (`Tool`, `System`, `Stream`), zero config for 80% case.

### Key design decision

Recommended a **hybrid of A + D with B's multi-turn**: Design D's short names for the common path, Design A's functional options for flexibility, Design B's Runner struct for multi-turn chat. This gives:

```go
// Simplest case (Design D)
runner.Run(ctx, "hello", runner.System("Be helpful"))

// With tools (Design D + A)
runner.Run(ctx, "analyze", runner.Tool("read", "Read file", readFn))

// Multi-turn (Design B)
r := runner.New(runner.Tool("sql", "Query DB", sqlFn))
r.Chat(ctx, "What tables exist?")
r.Chat(ctx, "Show me the data")
```

### What was tricky

Deciding where the package should live. Options considered:
- `geppetto/pkg/runner` — Chosen because it wraps geppetto types and doesn't need pinocchio
- `pinocchio/pkg/runner` — Rejected because pinocchio adds CLI/UI concerns not needed here
- `geppetto/pkg/inference/runner` — Too deep in the package hierarchy

The profile integration was also tricky — it needs to be optional but seamless. Settled on `runner.Profile("name", "registry.yaml")` which is clean but hides the complexity of profile resolution.

---

## Entry 5: Experiment Scripts (2026-03-17)

### What I wrote

6 experiment scripts in `scripts/`:

1. **experiment_01_minimal_runner.go** — Complete CLI tool with file reading and code search tools in ~35 lines. Demonstrates the "zero to working" promise.

2. **experiment_02_multiturn_chat.go** — Interactive multi-turn chat with SQL and calculator tools. Shows how `Runner.Chat()` maintains conversation state.

3. **experiment_03_cozodb_rewrite.go** — Side-by-side comparison of current CozoDB engine.go (~80 lines) vs proposed (~10 lines). The most dramatic simplification.

4. **experiment_04_temporal_extraction.go** — Batch extraction pipeline showing the outer-loop/inner-loop pattern. Application owns the outer iteration loop; runner handles the inner tool loop.

5. **experiment_05_runner_internals_sketch.go** — Implementation sketch showing how the runner wraps geppetto's types internally. Demonstrates that the simplification is real — it's wrapping existing abstractions, not replacing them.

6. **experiment_06_glazed_integration.go** — How a runner + glazed CLI tool would look. Shows the path from "quick script" to "production CLI tool" using glazed's output formatting.

### Key insight from writing experiments

The temporal-relationships experiment (04) revealed an important design consideration: some applications have an **outer loop** (multiple extraction passes) on top of geppetto's **inner tool loop** (inference → tools → re-inference). The runner should handle the inner loop transparently but NOT try to own the outer loop — that's application-specific logic.

This means:
- `runner.Run()` = single inference with tool loop (handles inner loop)
- `runner.Chat()` = appends to conversation (also handles inner loop per turn)
- Application code wraps `Run()` or `Chat()` in its own iteration logic

### How to validate

To review the designs:
1. Read `design/current-state-analysis.md` for the problem statement
2. Read `design/opinionated-runner-api-designs.md` for the proposals
3. Read `scripts/experiment_05_runner_internals_sketch.go` to see the implementation would work
4. Compare `scripts/experiment_03_cozodb_rewrite.go` to `2026-03-14--cozodb-editor/backend/pkg/hints/engine.go` for the before/after

### Open questions

1. Should `runner.Run()` return a `Result` struct or just `(string, error)` for the simplest case?
   - I chose `*Result` because it carries usage info and tool call records, which are almost always useful
   - But a `runner.RunText()` convenience function could return just the string

2. How should errors from individual tool calls surface?
   - Currently: tool errors go back to the LLM which decides how to handle them
   - Should we also expose them in `Result.ToolCalls[i].Error`?
   - Decision: yes, for observability, but don't short-circuit the loop on tool errors

3. Should the runner support context-based tools (tools that receive `context.Context`)?
   - geppetto already supports `func(ctx, Input) (Output, error)` signatures
   - The runner should transparently pass its context through
   - This enables tools that respect cancellation, deadlines, etc.

---

## Entry 6: Summary and Next Steps (2026-03-17)

### Documents produced

| Document | Type | Purpose |
|----------|------|---------|
| `design/current-state-analysis.md` | Analysis | Problem statement: architecture, usage patterns, pain points |
| `design/opinionated-runner-api-designs.md` | Design doc | 4 API proposals with rationale, trade-offs, examples |
| `scripts/experiment_01_minimal_runner.go` | Sketch | Minimal CLI tool demo |
| `scripts/experiment_02_multiturn_chat.go` | Sketch | Multi-turn chat demo |
| `scripts/experiment_03_cozodb_rewrite.go` | Sketch | Before/after comparison for CozoDB |
| `scripts/experiment_04_temporal_extraction.go` | Sketch | Batch extraction pipeline demo |
| `scripts/experiment_05_runner_internals_sketch.go` | Sketch | Implementation internals |
| `scripts/experiment_06_glazed_integration.go` | Sketch | Glazed CLI integration demo |

### Recommended next steps

1. **Review with Manuel**: Get feedback on the hybrid recommendation (D + A + B)
2. **Prototype Phase 1**: Implement `runner.Run()` + `Tool()` + `System()` + `Model()` + `Stream()` — the 80% solution
3. **Port CozoDB**: Use the runner to simplify cozodb-editor's engine.go as proof of concept
4. **Phase 2**: Add `runner.New()` → `Runner.Chat()` for multi-turn
5. **Phase 3**: Profile integration, YAML config support
6. **Phase 4**: Glazed integration helpers
