### Title
Refactor Report: Turn-based Engine/Middleware integration in Geppetto

### Purpose

Document the end-to-end refactor from Conversation-slice based inference to a Turn-centric API, including what changed, why, how to work with it, remaining gaps, and concrete next steps for the next developer.

### Executive summary

- Introduced a new in-memory domain package `geppetto/pkg/turns` (Run/Turn/Block). Engines and middleware now operate on `*turns.Turn` instead of `conversation.Conversation`.
- Updated the engine interface to `RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)` and refactored OpenAI and Claude engines accordingly.
- Converted middleware framework and tool middleware to Turn-based handlers.
- Adjusted example commands to seed a `Turn` from the initial conversation, run the engine, and render output by converting back to a conversation for display.
- Added provider tool configuration in the middleware example so providers can emit structured tool calls; updated Claude engine to append tool_call/llm_text blocks from content blocks.

### What changed (high-level)

- Engine interface migrated from conversation to turn:
  - Old: `RunInference(ctx, messages conversation.Conversation) (conversation.Conversation, error)`
  - New: `RunInference(ctx, t *turns.Turn) (*turns.Turn, error)`
- Middleware framework and signatures updated:
  - Old: `type HandlerFunc func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error)`
  - New: `type HandlerFunc func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)`
- Tool calling middleware reimplemented for Turn blocks (`tool_call` → execute → `tool_use`).
- New conversion helpers provided to bridge conversation and turns for existing engines/examples.

### New/updated packages and files

- New: `geppetto/pkg/turns/types.go`
  - Types: `Run`, `Turn`, `Block`, `BlockKind`, `MetadataKV` (Block-level metadata), Turn `Metadata` (map), Turn `Data` (map)
  - Helpers: `AppendBlock`, `AppendBlocks`, `FindLastBlocksByKind`, `SetTurnMetadata`, `UpsertBlockMetadata`

- New: `geppetto/pkg/turns/conv_conversation.go`
  - `BuildConversationFromTurn(t *Turn) conversation.Conversation`
  - `BlocksFromConversationDelta(updated conversation.Conversation, startIdx int) []Block`

- Updated: `geppetto/pkg/inference/engine/engine.go`
  - Engine interface now operates on `*turns.Turn`.

- Updated: `geppetto/pkg/inference/middleware/middleware.go`
  - Handler/Middleware operate on `*turns.Turn`.
  - `EngineWithMiddleware` now composes Turn-based handlers.

- Updated: `geppetto/pkg/inference/middleware/tool_middleware.go`
  - Reworked to Turn-based processing:
    - After each engine step, extract pending `tool_call` blocks lacking corresponding `tool_use`.
    - Execute via `Toolbox` with per-call timeouts.
    - Append `tool_use` blocks with results or errors.
    - Loop up to `MaxIterations`.
  - Helpers added: `extractPendingToolCalls`, `executeToolCallsTurn`, `appendToolResultsBlocks`.
  - Kept the older conversation-oriented helpers in place but not compiled into the Turn flow.

- Updated: `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - Accepts a `*turns.Turn`; converts to conversation for existing request building.
  - Streams events; collects usage and stop reason into event metadata.
  - Appends `llm_text` and `tool_call` blocks to the incoming `Turn`.
  - Returns the updated `Turn`.

- Updated: `geppetto/pkg/steps/ai/claude/engine_claude.go`
  - Accepts a `*turns.Turn`; converts to conversation for request building.
  - Streams and merges content blocks; converts Claude content blocks to `turns.Block`:
    - Text → `llm_text` block
    - ToolUseContent → `tool_call` block (with id/name/args)
  - Returns the updated `Turn`.

- New (temporary): `geppetto/pkg/steps/ai/openai/helpers_ids.go`
  - Was introduced during refactor; no longer used after switching back to `conversation.NewNodeID()`. Safe to remove.

- Updated: `geppetto/cmd/examples/simple-inference/main.go`
  - Seeds a `Turn` from the initial system/user messages.
  - Runs the engine with the Turn; converts the result Turn back to a conversation for printing.
  - Optional logging middleware adapted to Turn.

- Updated: `geppetto/cmd/examples/middleware-inference/main.go`
  - Adds `--with-tools` and configures a demo `echo` tool in a `Toolbox`.
  - Configures provider engines with a matching `ToolDefinition` (OpenAI/Claude) so they can emit structured tool calls.
  - Uppercase demo middleware adapted to Turn.
  - Seeds Turn; renders output by converting back to conversation.

### Notable function signatures after refactor

- Engine interface (Turn-based):
  - `RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)`

- Middleware:
  - `type HandlerFunc func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)`
  - `type Middleware func(HandlerFunc) HandlerFunc`
  - Chain and `EngineWithMiddleware` updated accordingly.

- Tool middleware API:
  - `NewToolMiddleware(toolbox Toolbox, config ToolConfig) Middleware`
  - Toolbox: `ExecuteTool(ctx, name, args) (interface{}, error)`
  - Block semantics: `tool_call` blocks consumed; `tool_use` blocks appended with results.

### Commands/tools run

- Code search and diagnostics (ripgrep):
  - `rg -n "type Engine interface" geppetto/pkg`
  - `rg -n "NewToolMiddleware\(" geppetto`
  - `rg -n "UpsertTurnMetadata\(" geppetto`

- Builds:
  - `go build ./geppetto/pkg/turns`
  - `go build ./geppetto/cmd/examples/simple-inference`
  - `go build ./geppetto/cmd/examples/middleware-inference`

- Runs (Claude profile):
  - `PINOCCHIO_PROFILE=sonnet-4 go run ./geppetto/cmd/examples/middleware-inference middleware-inference --with-logging "Say hi in one sentence"`
  - `PINOCCHIO_PROFILE=sonnet-4 go run ./geppetto/cmd/examples/middleware-inference middleware-inference --with-logging --with-uppercase --with-tools --log-level debug --with-caller "Say hi, then consider using the echo tool with text 'hello tools'."`

### Errors encountered and fixes

- Engine/middleware type mismatches:
  - After changing interfaces to Turn-based, conversation-typed code in middleware and examples failed to compile.
  - Fixed by updating handler signatures, removing conversation cloning, and switching to Turn-based plumbing.

- Tool middleware initially still used conversation types in internal flow:
  - Removed conversation code from active path; reimplemented Turn-based loop (`extractPendingToolCalls`, execution, append `tool_use`).

- OpenAI engine: metadata and usage types mismatch:
  - Adjusted to set `metadata.Usage = &conversation.Usage{...}` and `metadata.StopReason` as `*string`.
  - Removed unused variables and simplified the conversion path.

- Claude engine: initially appended only text or relied on inline XML function calls:
  - Updated to map response content blocks into `turns.Block` (`llm_text` and `tool_call`).
  - Removed unused variables (e.g., `hasText`).

- jsonschema property map type error when configuring provider tool schema:
  - `jsonschema.Schema.Properties` expects an ordered map; changed to `jsonschema.NewProperties()` and used `Properties.Set("text", ...)`.

- Temporary `helpers_ids.go` became unused:
  - Left in tree; can be deleted in cleanup.

### Current status

- `turns` domain package is in place with metadata and data maps on `Turn`.
- Engines (OpenAI, Claude) implement Turn-based `RunInference`; they convert Turn→Conversation internally where needed.
- Middleware framework is Turn-based; Tool middleware runs an execution loop based on `tool_call`/`tool_use` blocks.
- Examples (`simple-inference`, `middleware-inference`) compile and run.
- With `--with-tools`, the example configures a demo tool and provider tool definitions; Claude conversion now appends `tool_call` blocks based on content blocks. The middleware is prepared to execute tools when providers emit structured tool calls.

### Gaps and risks

- Other engines/providers (if any) not yet migrated to Turn-based signature.
- Some existing commands or steps outside the two examples may still depend on conversation-based `Engine` and will break.
- Claude currently expresses tool usage in content blocks; we convert those to `tool_call` blocks, but richer provider metadata may require further parsing.
- `helpers_ids.go` is dead code and should be removed to avoid confusion.
- Tests for Turn-based middleware and engines are not yet added; existing middleware tests are conversation-based.

### What went wrong (why we had to “TOUCH GRASS”)

While wiring proper tool calling end-to-end, a few issues caused churn and back-and-forth fixes:

- Semantics mismatch for MaxIterations in tool middleware tests:
  - The legacy conversation-based test `TestToolMiddleware_MaxIterationsLimit` expected an error once `MaxIterations` is reached, regardless of whether any pending tool calls remain.
  - The new Turn middleware intentionally loops “until no more tool calls or limits reached.” In practice, when providers emit a tool_call and we append the corresponding tool_use, there are no pending calls on the next iteration, so the loop exits cleanly with no error even if we performed multiple iterations.
  - We briefly oscillated between “error vs no error” expectations. The new design is to finish successfully when no pending tool calls remain. Action: update or replace the old test to align with Turn semantics (no error when tool calls are exhausted). If we still want to enforce an upper bound, introduce an explicit policy that errors only when the iteration bound is hit while tool calls remain pending.

- Partial test migration and stale fixtures:
  - `pkg/inference/middleware/tool_middleware_test.go` still held conversation-style fixtures (e.g., `createMessageWithToolCall`, `ToolUseContent`), while the runtime moved to `turns.Turn` and block-level tool logic.
  - We ported tests and added `tool_middleware_turns_test.go` for Turn-focused coverage, but some conversation fixtures lingered and caused compilation noise and expectation mismatches. Action: finish pruning or fully migrate the remaining conversation-based tests, keeping only Turn-based assertions (or isolate conversation tests behind a dedicated adapter test if needed).

- Duplicate Turn tool-call logic between middleware and helpers:
  - To get provider-agnostic harnesses working, `pkg/inference/toolhelpers/helpers.go` now includes Turn-based versions of extract/execute/append helpers similar to those in the middleware. This avoids import cycles for now, but duplicates logic and risks drift.
  - Action: consolidate the Turn tool-call primitives into a small internal/shared package (e.g., `pkg/inference/toolruntime` or `pkg/turns/tooling`) and have both the middleware and helpers depend on it. This removes duplication and stabilizes semantics across paths.

- Example migrations created mixed states:
  - We updated `openai-tools`, `claude-tools`, `middleware-inference`, and `simple-streaming-inference` to Turn semantics. `generic-tool-calling` relies on the helper loop (now Turn-based under the hood). These compile, but the docs and some tests still reflect conversation flows.
  - Action: finish aligning examples and docs to a single mental model: Turn as the runtime structure; Conversation only for rendering and input seeding.

Net effect: We briefly went too deep into debugging a failing test with unclear desired semantics. The correct path is to codify the intended Turn behavior (no error when tool calls are exhausted) and update tests/docs, then centralize Turn tool helpers to eliminate duplication.

### Next steps (checklist)

1. Engines
   - [ ] Audit all engine implementations and migrate to Turn-based signature.
   - [ ] Ensure each engine maps provider responses to `Block` consistently (text/tool_call/system).

2. Middleware
   - [ ] Add unit tests for `extractPendingToolCalls`, `executeToolCallsTurn`, `appendToolResultsBlocks`.
   - [ ] Add a Turn-based logging middleware that includes run/turn/block IDs in logs.

3. Examples and docs
   - [ ] Update `geppetto/pkg/doc/topics/05-conversation.md` and `06-inference-engines.md` with Turn sections.
   - [ ] Add an example that shows a full Run with multiple Turns and windowing helpers.

4. Cleanup and polish
   - [ ] Remove `geppetto/pkg/steps/ai/openai/helpers_ids.go`.
   - [ ] Grep for any remaining conversation-based `RunInference` usages and remove/port.
   - [ ] Consider adding `RunManager` (`turns`) for multi-turn orchestration and context window utilities.

### Lessons learned

- Perform interface refactors in layers: update the core interface, adapt middleware framework, then engines, then examples.
- Provide conversion helpers (Turn↔Conversation) to reduce risk during migration.
- Provider-specific differences (Claude content blocks, OpenAI tool calls) should be normalized into block kinds as early as possible in engines so middleware remains provider-agnostic.
- Keep example commands runnable with clear flags (`--with-logging`, `--with-uppercase`, `--with-tools`) to validate behaviors end-to-end during refactors.

### Appendix: Key files touched

- `geppetto/pkg/turns/types.go`
- `geppetto/pkg/turns/conv_conversation.go`
- `geppetto/pkg/inference/engine/engine.go`
- `geppetto/pkg/inference/middleware/middleware.go`
- `geppetto/pkg/inference/middleware/tool_middleware.go`
- `geppetto/pkg/steps/ai/openai/engine_openai.go`
- `geppetto/pkg/steps/ai/claude/engine_claude.go`
- `geppetto/cmd/examples/simple-inference/main.go`
- `geppetto/cmd/examples/middleware-inference/main.go`
- (cleanup) `geppetto/pkg/steps/ai/openai/helpers_ids.go`


