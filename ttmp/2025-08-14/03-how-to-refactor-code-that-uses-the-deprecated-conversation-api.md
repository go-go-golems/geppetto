## How to refactor code that uses the deprecated Conversation API to Turns/Blocks

This guide explains how to migrate Geppetto code from the old `conversation` API (messages, roles, etc.) to the new provider-agnostic `Turn`/`Block` model in `pkg/turns`. It provides a focused, mechanical checklist with examples, common pitfalls, and file references.

See also the canonical overview: `pkg/doc/topics/08-turns.md`.

### What changes conceptually

- **Before (Conversation API)**: You built `conversation.Conversation` (ordered messages), engines read/write message content, tool usage was often encoded in provider-specific ways.
- **After (Turns/Blocks)**: You construct a `turns.Turn` containing ordered `turns.Block` records. Engines append `llm_text`, `tool_call`, and `tool_use` blocks. Middleware inspects and mutates blocks (e.g., executes tools) and can feed back into engines. Tools for a single step live in `Turn.Data`.

### Core types and helpers

- `turns.Turn`: `{ ID, RunID, Blocks []Block, Metadata map[string]any, Data map[string]any }`
- `turns.Block`: `{ ID, TurnID, Kind, Role, Payload map[string]any, Metadata map[string]any }`
- Block kinds: `User`, `LLMText`, `ToolCall`, `ToolUse`, `System`, `Other`
- Common payload keys: `turns.PayloadKeyText`, `turns.PayloadKeyID`, `turns.PayloadKeyName`, `turns.PayloadKeyArgs`, `turns.PayloadKeyResult`
- Turn data keys for tools: `turns.DataKeyToolRegistry`, `turns.DataKeyToolConfig`
- Builders/printing:
  - `turns.NewTurnBuilder()` to seed turns
  - `turns.FprintTurn(w, turn)` to print a transcript-like view

### Migration checklist (mechanical steps)

1) Replace ID helpers that used `conversation.NodeID`
- Search: `conversation.NewNodeID(` or types that expect `conversation.NodeID`.
- Replace with `uuid.New()` (type `uuid.UUID`), and import `github.com/google/uuid`.
- Example (engines already updated):
```go
// before
metadata := events.EventMetadata{ ID: conversation.NewNodeID() }
// after
metadata := events.EventMetadata{ ID: uuid.New() }
```
Files updated in this repo: `pkg/steps/ai/openai/engine_openai.go`, `pkg/steps/ai/claude/engine_claude.go`, `pkg/steps/ai/openai/helpers.go`, `pkg/steps/ai/claude/helpers.go`, `pkg/steps/ai/claude/api/messages.go`.

2) Stop constructing `conversation.Conversation` and seed a Turn instead
- Use the builder in `pkg/turns`:
```go
seed := turns.NewTurnBuilder().
  WithSystemPrompt("You are a helpful assistant.").
  WithUserPrompt(prompt).
  Build()
```
- Example usage: `cmd/examples/simple-streaming-inference/main.go`.

3) Run engines with a Turn (not a Conversation)
- Old:
```go
// conversation := ...
// engine.RunInference(context, conversation)
```

```result
```
- New:
```go
updatedTurn, err := engine.RunInference(ctx, seedTurn)
```
- The engine appends blocks: assistant text as `LLMText`, tool invocations as `ToolCall`, and tool results as `ToolUse`.

4) Printing results without a Conversation
- Use `turns.FprintTurn(w, updatedTurn)` to display a readable transcript, or
- Use existing event printer handlers via the event router for structured output (`pkg/events`) as done in the streaming example.

5) Tools are provided per-turn (not globally via conversations)
- Attach a registry and configuration via `Turn.Data`:
```go
// reg := tools.NewInMemoryToolRegistry() // register tools
turn.Data[turns.DataKeyToolRegistry] = reg
turn.Data[turns.DataKeyToolConfig] = engine.ToolConfig{ Enabled: true, ToolChoice: engine.ToolChoiceAuto }
```
- Engines read these keys to include provider tool definitions for that step.

6) Remove conversation conversions unless explicitly bridging
- If you used helpers to convert between conversations and provider messages, remove them.
- Transitional helpers exist in `pkg/turns/conv_conversation.go` but are meant only for bridging; prefer operating directly on blocks.

7) Events and metadata
- Event metadata lives in `pkg/events`; it already uses `uuid.UUID` for message IDs.
- Engines should publish start/partial/final/tool-call/tool-result events during streaming as they append blocks.

### Typical code changes (pseudocode with references)

- Seed a turn (replace conversation construction)
```go
// file: cmd/examples/simple-streaming-inference/main.go
seed := turns.NewTurnBuilder().
  WithSystemPrompt("You are a helpful assistant. Answer briefly.").
  WithUserPrompt(inputPrompt).
  Build()

updated, err := engine.RunInference(ctx, seed)
if err != nil { /* handle */ }

turns.FprintTurn(w, updated)
```

- Attach tools per step
```go
// file: engine caller (any)
reg := tools.NewInMemoryToolRegistry()
// reg.Register(...)
seed.Data = map[string]any{}
seed.Data[turns.DataKeyToolRegistry] = reg
seed.Data[turns.DataKeyToolConfig] = engine.ToolConfig{ Enabled: true, ToolChoice: engine.ToolChoiceAuto }
```

- Engine emits events and appends blocks
```go
// file: pkg/steps/ai/openai/engine_openai.go or claude/engine_claude.go
metadata := events.EventMetadata{ ID: uuid.New(), /* ... */ }
// during streaming: publish partials and finally append blocks on the Turn
turns.AppendBlock(t, turns.NewAssistantTextBlock(text))
turns.AppendBlock(t, turns.NewToolCallBlock(id, name, args))
turns.AppendBlock(t, turns.NewToolUseBlock(id, result))
```

### Mapping guide: Conversation → Turns/Blocks

- System message → `BlockKindSystem` with payload `{ text }`
- User message → `BlockKindUser` with payload `{ text, images? }`
- Assistant text → `BlockKindLLMText` with payload `{ text }`
- Assistant tool_calls → `BlockKindToolCall` with payload `{ id, name, args }`
- Tool results (formerly `tool` messages) → `BlockKindToolUse` with payload `{ id, result }`

### Common pitfalls and fixes

- “Where did message IDs come from?”
  - Use `uuid.New()` for event/message IDs; block IDs typically come from the provider (for tool calls) or `uuid.NewString()` in helpers.

- “Where are tools configured now?”
  - On `Turn.Data` using `turns.DataKeyToolRegistry` and `turns.DataKeyToolConfig`.

- “I still have code calling `BuildConversationFromTurn`”
  - Remove it if possible. Only use bridging functions when interfacing with legacy components.

- “How do I print a result without a conversation?”
  - Use `turns.FprintTurn(w, turn)` for a chat-like view or structured event printers.

- “Claude/OpenAI role strings without `conversation` types?”
  - Use string literals: `"system"`, `"user"`, `"assistant"`, `"tool"`. For example, `claude/helpers.go` uses `"assistant"`/`"user"` directly.

- “Leftover conversation utilities in provider API packages?”
  - In `claude/api/messages.go`, the `ToMessage` helper (which returned a `conversation.Message`) is deprecated and the `conversation` import removed. Prefer Turn/Block events and builders.

### End-to-end example references

- Example command (seed/stream/print): `cmd/examples/simple-streaming-inference/main.go`
- Engines (OpenAI/Claude) appending blocks and emitting events:
  - `pkg/steps/ai/openai/engine_openai.go`
  - `pkg/steps/ai/claude/engine_claude.go`
- Turn helpers:
  - Builder: `pkg/turns/builders.go`
  - Printing: `pkg/turns/helpers_print.go`
  - Block constructors and keys: `pkg/turns/helpers_blocks.go`, `pkg/turns/keys.go`
- Events: `pkg/events/chat-events.go`

### Testing tips

- Assert on the structure and order of blocks:
```go
// want: system, user, assistant
if len(t.Blocks) < 3 { t.Fatalf("expected >=3 blocks") }
if t.Blocks[0].Kind != turns.BlockKindSystem { t.Fatalf("want system first") }
if t.Blocks[1].Kind != turns.BlockKindUser { t.Fatalf("want user second") }
if t.Blocks[2].Kind != turns.BlockKindLLMText { t.Fatalf("want assistant third") }
```
- Verify tool call/result pairing by matching `Payload["id"]` across `ToolCall` and `ToolUse` blocks.

### Appendix: Search patterns to start a refactor

- Find conversation usage:
  - `conversation.Conversation`
  - `conversation.NewChatMessage`
  - `conversation.NewNodeID`
  - `RoleUser|RoleAssistant|RoleSystem|RoleTool`

- Replace with turns equivalents:
  - `turns.NewUserTextBlock`, `turns.NewAssistantTextBlock`, `turns.NewSystemTextBlock`
  - `turns.NewToolCallBlock`, `turns.NewToolUseBlock`
  - `turns.NewTurnBuilder`
  - `uuid.New()` (event IDs)


