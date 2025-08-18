---
Title: Design: Integrating Run/Turn into Geppetto Engines and Middleware
Short: Unifying conversations, engines, and middleware around a Run/Turn data model with block-level structure and rich metadata
Tags:
- geppetto
- inference
- conversation
- runs
- turns
- middleware
---

### Purpose and scope

This document proposes how to integrate a run/turn data model into Geppetto’s inference stack so that middleware and engines operate on richer structures than a flat conversation. The end-goal is to have engines process a Turn (with ordered Blocks and metadata), return an updated Turn, and let middleware inspect/transform tool calls, metadata, and provider state at block-level granularity. For now, we will use pure Go structs (no Ent storage). Adding persistence can happen later without changing the engine/middleware APIs. This design updates the existing Engine and middleware to operate directly on Turns (breaking change), without adapters or a parallel compatibility layer.

The design builds on the current Conversation abstractions documented in `geppetto/pkg/doc/topics/05-conversation.md` and the engine-first architecture in `geppetto/pkg/doc/topics/06-inference-engines.md`, and leverages the working prototype in `vibes/2025-08-09/turn-inspector`.

### What exists today

- Conversation API (`geppetto/pkg/conversation`):
  - Tree structure of `Message` with roles and content types (chat, tool_use, tool_result, image, error).
  - `Manager` builds linear `Conversation` slices for engines.

- Engine API (`geppetto/pkg/inference/engine`):
  - `RunInference(ctx, messages Conversation) (Conversation, error)` returns a full, updated conversation. Engines focus on provider I/O; tool orchestration handled outside.

- Middleware (`geppetto/pkg/inference/middleware`):
  - Chains on top of `HandlerFunc(ctx, Conversation) (Conversation, error)`.
  - Tool middleware extracts tool calls from conversation responses, executes, appends results, then loops.

### What the prototype proves (Turn Inspector)

Located at `vibes/2025-08-09/turn-inspector`:

- Ent schema for persistent model:
  - `Run` → has many `Turn` and `RunMetadata`.
  - `Turn` → has many ordered `Block` and `TurnMetadata`.
  - `Block` → fields: `order`, `kind` (one of `user`, `llm_text`, `tool_call`, `tool_use`, `system`, `other`), optional `role`, flexible JSON `payload`, and `BlockMetadata`.
  - Uniqueness: `(turn, order)` for deterministic sequencing; `(entity, source, key)` uniqueness for metadata.

- Helper API (`pkg/ti/ti.go`):
  - Create/Upsert/Delete operations: `CreateRun`, `CreateTurnWithBlocks`, `Upsert*Metadata`, `DeleteTurnCascade`, `DeleteRunCascade`, `SwapBlockOrders`.

- CLI demonstrates:
  - Creating Turns with mixed blocks (user, tool_call/use, llm_text, system), metadata at turn and block levels.
  - Querying by metadata, text, kinds; showing stats and structure.

Key takeaways:
- Blocks provide a normalized, provider-agnostic representation of “what happened” in a single interaction.
- Rich metadata at both Turn and Block levels enables powerful middleware and analytics without parsing opaque text.
- A Run captures the full session across multiple Turns.

### Target architecture in Geppetto

1) Introduce domain types (Go structs only)

- New package: `geppetto/pkg/turns` with domain structs and interfaces decoupled from Ent:
  - `Run` { ID, Name, Metadata []MetadataKV, Turns []Turn }
  - `Turn` { ID, RunID, Metadata []MetadataKV, Blocks []Block }
  - `Block` { ID, TurnID, Order int, Kind BlockKind, Role string, Payload map[string]any, Metadata []MetadataKV }
  - `BlockKind` enum: `User`, `LLMText`, `ToolCall`, `ToolUse`, `System`, `Other`
  - `MetadataKV` { Source, Key, Value string }

- No storage layer at this stage:
  - Keep everything in memory using Go structs. Provide small utilities to manipulate blocks and metadata (append blocks, reorder, merge metadata) without any persistence concerns.

2) Engine interface operates on Turn pointers (breaking change)

- Modify the existing `engine.Engine` interface to:
  - `RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)`

- Engines are responsible for mapping a `Turn` into provider-specific request payloads (e.g., building message arrays from blocks), executing provider I/O, and appending resulting blocks (`llm_text`, optionally `tool_call`) back into the returned `Turn`. Engines may mutate the provided `*Turn` and should return the final `*Turn` (same pointer or a new instance).

- No adapters, no conversation-based signature, no backwards compatibility path.

3) Middleware operates on Turn pointers (replacing Conversation)

- Update the existing middleware types in `geppetto/pkg/inference/middleware` to use `*Turn` instead of `Conversation`:
  - `type HandlerFunc func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)`
  - `type Middleware func(HandlerFunc) HandlerFunc`
  - Keep the same `Chain` function name and behavior; it now chains Turn-based handlers. There is no separate `ChainTurn`.

- Tool middleware updated to block-level semantics:
  - Extract `tool_call` blocks from assistant output blocks within the Turn.
  - Execute tools, append `tool_use` blocks (and errors) with proper ordering and metadata.
  - Loop until no more tool calls or limits reached, as today, but working on `Turn` and `Block`.

- Tool middleware updated to block-level semantics:
  - Extract `tool_call` blocks from the last assistant block(s) in the Turn (no need to parse provider-specific metadata).
  - Execute tools, append `tool_use` blocks (and errors) with proper ordering and metadata.
  - Re-enter the engine loop until no more tool calls or limits are reached, mirroring current logic but in terms of `Turn` and `Block` instead of `Conversation` and `Message`.

4) Provider request helpers (optional)

- Engines may use helper functions to build provider-specific message arrays from a `Turn`’s blocks (e.g., flattening multiple recent Turns into context). Place these as utilities, not as a public compatibility layer:
  - `BuildProviderMessagesFromTurn(t Turn, window WindowSpec) ([]ProviderMessage, error)`
  - Keep these internal to engine implementations or in a provider/helper package.

5) Run orchestration

- Add a `RunManager` in `geppetto/pkg/turns` to manage multi-turn sessions:
  - Tracks active `Run` (ID, optional name, metadata)
  - Appends new `Turn` for each interaction (in memory)
  - Provides helpers to query previous turns and compute context windows for the next inference (e.g., synthesize provider-ready messages from selected blocks across recent turns).

6) Events and streaming

- Keep existing event sink model but enrich events with turn/block context:
  - Emit events when a `Turn` is created, when `Block`s are appended, when tool calls and tool uses are produced.
  - Middleware can publish structured events referencing `run_id`, `turn_id`, `block_id`, and `order`.

### Detailed design

#### A) Domain types (no storage)

- Define `turns` package with Go structs only. Provide pure in-memory helpers:
  - `AppendBlock(t *Turn, b Block)`, `AppendBlocks(t *Turn, ...Block)` ensure increasing order
  - `FindLastBlocksByKind(t Turn, kinds ...BlockKind) []Block`
  - `UpsertTurnMetadata(t *Turn, kv MetadataKV)` and `UpsertBlockMetadata(b *Block, kv MetadataKV)` by `(source,key)`
  - `ComputeContextWindow(run Run, spec WindowSpec) []Block` (for engines to turn into provider messages)

These helpers enable middleware and engines to work without persistence. A storage implementation can be added later behind separate packages without changing these APIs.

#### B) Engine changes (breaking)

- Update `engine.Engine` to `RunInference(ctx, t *turns.Turn) (*turns.Turn, error)`.
- Provider engines map `Turn` blocks to provider request payloads and back to `Turn` blocks.
- The existing factory continues to construct engines, but they now conform to the Turn-based signature. No adapters provided.

#### C) Middleware refactor

- Replace Conversation-based middleware with Turn-based middleware (same names where possible, different types). Example:
  - Logging: `func NewLoggingMiddleware(logger zerolog.Logger) Middleware` now operates on `*Turn`.
  - Tools: `func NewToolMiddleware(toolbox Toolbox, cfg TurnToolConfig) Middleware` operates on `*Turn`.
  - Metrics/Tracing: emit events keyed by `run_id`, `turn_id`, `block_id`.

- Turn tool workflow changes versus current `tool_middleware.go`:
  - Extraction: instead of parsing `Message.Metadata`, read the latest assistant `Block`(s) of kind `tool_call` and parse payload args (already JSON).
  - Execution: call toolbox; produce `tool_use` blocks with results/errors; append to the same `Turn` with increasing `order`.
  - Loop: re-run the next engine step if tool results imply the model should continue; cap by `MaxIterations` and timeouts.

#### D) Provider context building (optional helpers)

- Provide utilities for engines to flatten recent Turns into provider-ready messages. Keep this optional and implementation-specific, not as part of any public conversation compatibility surface.

### Mapping rules (Turn ↔ Conversation)

- Block.kind → Message:
  - `user` → `ChatMessageContent` with `RoleUser` and `payload.text`.
  - `llm_text` → `ChatMessageContent` with `RoleAssistant` and `payload.text` (and optional `payload.code` as additional formatted text or attachment).
  - `system` → `ChatMessageContent` with `RoleSystem` using `payload.text`.
  - `tool_call` → `ToolUseContent` with `ToolID` (from payload.id or name), `Name`, `Input` (payload.args marshaled to `json.RawMessage`).
  - `tool_use` → `ToolResultContent` with `ToolID` and `payload.result` serialized to string/JSON.

- Message → Block.kind:
  - `ChatMessageContent` (roles user/assistant/system) → `user`/`llm_text`/`system` with `payload.text` from content text.
  - `ToolUseContent` → `tool_call` with `payload.tool` and `payload.args`.
  - `ToolResultContent` → `tool_use` with `payload.result`.

- Metadata:
  - Carry select `Message.Metadata` to `BlockMetadata` with a `source` convention (e.g., `provider`, `runtime`, `trace`).
  - Turn-level metadata can capture request-level parameters like temperature, tool config, prompts used, etc.

### API changes summary

- Add new packages:
  - `geppetto/pkg/turns` (domain types, RunManager, small provider-request helper utils)

- Modify existing:
  - `engine.Engine` to operate on `*turns.Turn`
  - `inference/middleware` `HandlerFunc` and `Chain` to operate on `*turns.Turn`

- Remove/avoid:
  - No storage for now (no Ent); no adapters; no conversation-based compatibility surface.

### Refining middleware

- Logging middleware: include run/turn/block identifiers; log block kinds and sizes; summarize payloads.
- Tool middleware: shift to operating on blocks; respect `ToolFilter`, `MaxIterations`, and per-call timeouts; emit `tool_call` and `tool_use` events with structured data.
- Policy middleware: inspect `Turn.Metadata` to enforce limits (max blocks, max tool calls, redaction) before engine calls.
- Metrics/tracing: export counters and timings per block kind and per tool; add hooks for Watermill sinks already present in Geppetto.

### Data lifecycle (in memory)

- On each user interaction:
  1) `RunManager` creates a new `*Turn` with initial `user` (and `system`) blocks (in memory).
  2) Middleware stack calls `engine.Engine.RunInference(ctx, turnPtr)`.
     - Engine appends provider outputs as `llm_text` and any `tool_call` blocks to the provided `*Turn` (or returns a new `*Turn`).
     - Tool middleware executes tools and appends `tool_use` results (loop with limits/timeouts).
  3) The updated `*Turn` remains in memory and is attached to the current `Run`.

Persistence (e.g., Ent) can be added later without changing engine/middleware signatures.

### Migration strategy

- Single-step breaking change in the inference layer:
  - Change Engine interface and middleware to Turn-based APIs.
  - Update provider engines, middleware, and examples accordingly.

### Risks and mitigations

- Dual path complexity (Conversation vs Turn):
  - Mitigate via thin, well-tested adapters and a single source of truth in conversion utilities.

- Provider differences for tool calls:
  - Keep conversation-based extraction as fallback; prefer explicit `tool_call` blocks produced by provider-specific engines when possible.

### Implementation outline (high level)

1) Domain
   - [ ] Create `geppetto/pkg/turns` with types and `RunManager` scaffolding (pure Go)
   - [ ] Add in-memory helpers for blocks and metadata

2) Provider request helpers
   - [ ] Implement helpers to build provider-ready messages from a Turn and recent context (window spec)

3) Engine and middleware
   - [ ] Modify `engine.Engine` to use `Turn`
   - [ ] Update middleware HandlerFunc/Chain to use `Turn`
   - [ ] Port tool middleware to Turn-based execution

4) Events/metrics
   - [ ] Enrich event payloads with run/turn/block IDs
   - [ ] Provide printers and sample subscribers for Turn events

5) Examples and docs
   - [ ] Add example that runs a full Run with several Turns via updated `engine.Engine`
   - [ ] Update `05-conversation.md` and `06-inference-engines.md` with Turn sections

### Appendix: Key references

- Conversation tutorial: `geppetto/pkg/doc/topics/05-conversation.md`
- Engine architecture: `geppetto/pkg/doc/topics/06-inference-engines.md`
- Existing middleware shape: `geppetto/pkg/inference/middleware/middleware.go`, `tool_middleware.go`
- Prototype model and helpers: `vibes/2025-08-09/turn-inspector/ent/schema/*.go`, `pkg/ti/ti.go`, CLI under `cmd/`


