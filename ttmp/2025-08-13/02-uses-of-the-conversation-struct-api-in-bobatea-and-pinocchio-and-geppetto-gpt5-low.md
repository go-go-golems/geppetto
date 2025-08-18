### Scope

- **Goal**: Identify usages of the `conversation` abstraction (notably `geppetto/pkg/conversation/message.go` and related types like `Message`, `Conversation`, `Manager`, `ChatMessageContent`, `ToolUseContent`, `ToolResultContent`, roles) across `bobatea/`, `pinocchio/`, and `geppetto/` to inform removal/migration.

### High-level findings

- The abstraction is used across UI layers (Bobatea conversation UI), Pinocchio command/runtime glue, and Geppetto engines/middleware/events. Bridges exist between the newer `turns` API and `conversation` (both directions) and will need removal.

---

### bobatea/

- `bobatea/pkg/chat/conversation/model.go`
  - Imports `conversation` as `conversation2`.
  - Uses `conversation2.Manager`, `conversation2.Message`, `conversation2.NodeID` to render and cache the conversation tree in the Bubble Tea sub-UI.
  - Maintains cache keyed by `NodeID` and renders message content.

- `bobatea/pkg/chat/model.go`
  - Imports `geppetto_conversation`.
  - Root chat UI integrates with conversation-based sub-UI and events; handles `*conversation.Message` for selection, copy/export, and counts.

- `bobatea/cmd/conversation/main.go`
  - Demo wiring `conversation.NewManager()` to `pkg/chat/conversation` UI; simulates streaming by appending conversation messages.

- `bobatea/cmd/chat/fake-backend.go`
  - Imports `conversation2`; emits `*conversation.Message` to drive UI for testing.

- `bobatea/pkg/chat/conversation/README.md`
  - Documents streaming message types assuming a conversation-tree model.

- `bobatea/prompto/bobatea/conversation-ui`
  - Script showcasing conversation UI and docs.

Implication: Replace conversation-backed Bobatea sub-UI with Timeline/Turns-native models; remove `Manager`, `Message`, `NodeID` usage and demo/fake-backend or port them.

---

### pinocchio/

- `pinocchio/pkg/cmds/cmd.go`
  - Imports `conversation` and returns `[]*conversation.Message` in some code paths (prompt printing).
  - `buildInitialTurn(systemPrompt string, msgs []*conversation.Message, userPrompt string)` converts legacy messages into `turns` blocks via `turns.BlocksFromConversationDelta(conversation.Conversation(msgs), 0)`.
  - In blocking/chat flows, converts `Turn` results back to `conversation` via `turns.BuildConversationFromTurn`, appends `*conversation.Message`, prints last `ChatMessageContent`.

- `pinocchio/pkg/ui/backend.go`
  - Imports `conv` (conversation) with `turns` and `timeline`; backend bridges still refer to conversation types.

- `pinocchio/pkg/chatrunner/chat_runner.go`
  - Reads `Conversation` from a manager, runs inference on a `Turn`, converts back with `turns.BuildConversationFromTurn`, appends `*conversation.Message`, prints last `ChatMessageContent`.

- `pinocchio/pkg/cmds/run/context.go`
  - Imports `geppetto_conversation`; run context/types still reference conversation in signatures/state.

- `pinocchio/cmd/agents/simple-chat-agent/main.go`
  - Uses `github.com/go-go-golems/geppetto/pkg/conversation/builder` to construct conversation content for a simple agent example.

- `pinocchio/ttmp/2025-08-13/pkg-diff.txt`
  - Diff evidence of ongoing migration; multiple files still import conversation and convert between `turns` and `conversation`.

Implication: Update CLI entrypoints and chat runtime to be Turn/Timeline-native, stop returning `[]*conversation.Message`, and remove all conversion shims at call sites.

---

### geppetto/

- `geppetto/pkg/turns/conv_conversation.go`
  - Bridge functions: `BuildConversationFromTurn(*Turn) conversation.Conversation` and `BlocksFromConversationDelta(conversation.Conversation, startIdx)`.

- `geppetto/pkg/js/conversation-js.go`
  - Exposes `conversation.Manager` to Goja; constructs `NewChatMessage` with options and appends via `manager.AppendMessages`.

- Engines/helpers referencing conversation content types directly:
  - `geppetto/pkg/steps/ai/claude/engine_claude.go`
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `geppetto/pkg/steps/ai/claude/content-block-merger.go`
  - `geppetto/pkg/steps/ai/claude/helpers.go`
  - `geppetto/pkg/inference/toolhelpers/helpers.go`
  - `geppetto/pkg/inference/middleware/tool_middleware.go`

- Events integration:
  - `geppetto/pkg/events/chat-events.go` formats/emits chat events with conversation message data.

- Examples/docs/prompts:
  - `geppetto/cmd/examples/simple-inference/main.go`
  - `geppetto/cmd/examples/middleware-inference/main.go`
  - `geppetto/pkg/doc/topics/05-conversation.md`
  - `geppetto/pkg/conversation/README.md`
  - `geppetto/prompto/geppetto/conversation`

Implication: Replace content-type usage in engines/middleware/events with Turn-native structures and remove JS exposure and docs for conversation.

---

### Representative import references

- bobatea
  - `bobatea/pkg/chat/conversation/model.go`: `conversation2 "github.com/go-go-golems/geppetto/pkg/conversation"`
  - `bobatea/pkg/chat/model.go`: `geppetto_conversation "github.com/go-go-golems/geppetto/pkg/conversation"`
  - `bobatea/cmd/conversation/main.go`: `conversation2 "github.com/go-go-golems/geppetto/pkg/conversation"`
  - `bobatea/cmd/chat/fake-backend.go`: `conversation2 "github.com/go-go-golems/geppetto/pkg/conversation"`

- pinocchio
  - `pinocchio/pkg/cmds/cmd.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`
  - `pinocchio/pkg/ui/backend.go`: `conv "github.com/go-go-golems/geppetto/pkg/conversation"`
  - `pinocchio/pkg/chatrunner/chat_runner.go`: `geppetto_conversation "github.com/go-go-golems/geppetto/pkg/conversation"`
  - `pinocchio/pkg/cmds/run/context.go`: `geppetto_conversation "github.com/go-go-golems/geppetto/pkg/conversation"`
  - `pinocchio/cmd/agents/simple-chat-agent/main.go`: `"github.com/go-go-golems/geppetto/pkg/conversation/builder"`

- geppetto
  - `geppetto/pkg/turns/conv_conversation.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`
  - `geppetto/pkg/js/conversation-js.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`
  - `geppetto/pkg/steps/ai/claude/content-block-merger.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`
  - `geppetto/pkg/steps/ai/claude/helpers.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`
  - `geppetto/pkg/steps/ai/claude/engine_claude.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`
  - `geppetto/pkg/inference/toolhelpers/helpers.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`
  - `geppetto/pkg/inference/middleware/tool_middleware.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`
  - `geppetto/pkg/events/chat-events.go`: `"github.com/go-go-golems/geppetto/pkg/conversation"`

---

### Notes for deprecation/removal

- Replace Bobatea conversation UI with Timeline/Turns-native models; remove `Manager`, `Message`, `NodeID` references and related demos.
- Update Pinocchio command APIs to stop returning `[]*conversation.Message`; eliminate conversion helpers at call sites.
- Refactor engines/middleware/events to operate on `turns` blocks and Turn-native structures.
- Remove JS exposure (`pkg/js/conversation-js.go`) and docs describing conversation.

This list should serve as a checklist to verify all imports/usages are removed during the migration.
