## Detailed session write-up: refactoring events (removing StepMetadata), stabilizing Run/Turn IDs, agent-mode improvements, and SQLite debug storage for the simple agent

### Purpose and scope

This document summarizes the refactors and cleanups implemented during the session, with a focus on:
- Simplifying and strengthening the event model by removing the old step abstraction and inlining correlation IDs.
- Making RunID and TurnID stable and ubiquitous across the entire app stack.
- Improving agent-mode middleware instructions and UI visibility of modes and mode switches.
- Adding embedded SQL schemas and debugging views, and logging events into SQLite for robust inspection.
- Capturing what changed, why it changed, where the changes live (file/function references), and what to do next.

This document also provides a concrete next-steps plan with checklists, example queries, and code pointers so you can continue the work seamlessly.

### High-level overview of changes

- Event metadata was simplified around `EventMetadata`, adding a general-purpose `Extra` map for correlation and context instead of the now-removed `StepMetadata`.
  - New constants for correlation keys: `run_id` and `turn_id`.
  - All our components (engines, UI, SQLite logger) now use `EventMetadata.Extra` for RunID/TurnID.

- Stable RunID/TurnID middleware was added in the simple agent to guarantee these identifiers are always set for every `RunInference` call.

- Agent Mode middleware was wired into the simple agent, and its injected user prompt was made more composable using tags:
  - `<currentMode>…</currentMode>` for the current mode guidance
  - `<modeSwitchGuidelines>…</modeSwitchGuidelines>` for in-band YAML mode switching instructions
  - The middleware also emits Info/Log events with `run_id` and `turn_id` set in `EventMetadata.Extra`.

- A robust SQLite debug store was added using go:embed-ed SQL files, with KV tables for payloads/metadata, an events table, and documented views to make queries trivial.

### Detailed changes by area

#### Events: removal of StepMetadata; EventMetadata.Extra and keys

- File: `geppetto/pkg/events/chat-events.go`
  - Removed `StepMetadata` and all its usages from event constructors and interface.
  - Added `EventMetadata.Extra map[string]interface{}` as a general-purpose context map.
  - Added constants for correlation keys:
    - `MetaKeyRunID = "run_id"`
    - `MetaKeyTurnID = "turn_id"`
  - Zerolog marshal now includes `extra` when present.
  - All event constructors have updated signatures. Examples:
    - `NewStartEvent(metadata EventMetadata)`
    - `NewFinalEvent(metadata EventMetadata, text string)`
    - `NewPartialCompletionEvent(metadata EventMetadata, delta string, completion string)`
    - `NewToolCallEvent(metadata EventMetadata, toolCall ToolCall)`
    - `NewToolCallExecuteEvent(metadata EventMetadata, toolCall ToolCall)`
    - `NewToolCallExecutionResultEvent(metadata EventMetadata, toolResult ToolResult)`
    - `NewErrorEvent(metadata EventMetadata, err error)`
    - `NewLogEvent(metadata EventMetadata, level, message string, fields map[string]interface{})`
    - `NewInfoEvent(metadata EventMetadata, message string, data map[string]interface{})`

Why: this consolidates correlation and provider metadata in one place, makes the event model simpler, and removes the need for a step layer that we no longer use.

#### Engines and tool executor updated to new events

- File: `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - Removed `StepMetadata` creation and usage; attached provider settings into `EventMetadata.Extra` when relevant.
  - Updated all event publishing calls to use the new constructors (no step args).

- File: `geppetto/pkg/steps/ai/claude/engine_claude.go`
  - Removed step metadata; added engine settings into `EventMetadata.Extra`.
  - Updated the streaming flow to publish events with the new constructors.

- File: `geppetto/pkg/steps/ai/claude/content-block-merger.go`
  - Refactored to use `EventMetadata.Extra` for model, message_id, role, stop reason, etc.
  - Updated intermediate event generation to new constructors.

- File: `geppetto/pkg/inference/tools/executor.go`
  - `NewToolCallExecuteEvent` and `NewToolCallExecutionResultEvent` updated to new signatures.

#### Printers and Bobatea

- File: `geppetto/pkg/events/printer.go`
  - Removed StepMetadata printing; compact metadata now derived from `EventMetadata` only.

- File: `bobatea/pkg/chat/conversation/model.go`
  - `StreamMetadata` no longer contains `StepMetadata`.
  - All references to step metadata removed; `EventMetadata` remains.

- File: `pinocchio/pkg/ui/backend.go`
  - The UI forwarder that builds `StreamMetadata` now omits `StepMetadata` and only forwards `EventMetadata`.

#### Agent Mode middleware improvements

- File: `geppetto/pkg/inference/middleware/agentmode/middleware.go`
  - Injected agent-mode prompt now uses composable tags:
    - `<currentMode>…</currentMode>`
    - `<modeSwitchGuidelines>…</modeSwitchGuidelines>`
  - The list of available modes is embedded in the switch instructions for clarity.
  - On insertion and on detected mode switch, Log/Info events are emitted with `EventMetadata.Extra` populated:
    - `{"run_id": t.RunID, "turn_id": t.ID}`
  - On mode switch, emits an additional simple Info event (“Mode changed”) meant for REPL/UI visibility.

#### Simple agent: stable RunID/TurnID, snapshots, event logging

- File: `pinocchio/cmd/agents/simple-chat-agent/main.go`
  - Wrapped the engine with a middleware ensuring non-empty `RunID` and `TurnID` for every `*turns.Turn`.
  - Added a snapshot middleware that stores `pre` and `post` turn snapshots to SQLite.
  - Wired a handler to persist incoming events (tool/log/info) into SQLite via the shared store.

- Files: `pinocchio/cmd/agents/simple-chat-agent/pkg/store/schema.sql`, `views.sql`, `sqlstore.go`
  - All SQL is embedded with go:embed.
  - Tables:
    - `runs`, `run_metadata_kv`
    - `turns`, `turn_kv` (sections: `metadata` / `data`)
    - `blocks`, `block_payload_kv`, `block_metadata_kv` (per-phase `pre`/`post`)
    - `turn_snapshots` (full JSON snapshots for convenience)
    - `chat_events` (tool-call/execute/results, info/log, with run_id/turn_id)
  - Views:
    - `v_turn_modes`: join Turn.Data agent_mode and injected block metadata `agentmode` per turn
    - `v_injected_mode_prompts`: the actual injected agent-mode prompt text per turn
    - `v_recent_events`: quick feed of recent events
    - `v_tool_activity`: aggregated lifecycle per `tool_id`

Example queries:

```sql
-- Agent mode recorded on Turn.Data and injected block metadata
SELECT * FROM v_turn_modes ORDER BY turn_id;

-- The injected mode prompt text
SELECT * FROM v_injected_mode_prompts;

-- Recent events
SELECT * FROM v_recent_events LIMIT 50;

-- Tool lifecycle
SELECT * FROM v_tool_activity ORDER BY last_seen DESC;
```

### Why this refactor matters

- Removing `StepMetadata` removes a leaky abstraction and centralizes all runtime/correlation/provider metadata in `EventMetadata`.
- Stable RunID/TurnID makes agent-mode persistence, debugging, and analytics deterministic.
- KV tables and views in SQLite make root cause analysis and “what did we send to the provider” queries easy and fast.
- Tagged prompts for agent-mode make the system prompt composable and machine-parsable across different consumers.

### What to do next (detailed plan)

1) Generate real IDs
   - Replace placeholder session run id and turn id with real UUIDs.
   - File: `pinocchio/cmd/agents/simple-chat-agent/main.go`
   - Pseudocode:
     ```go
     sessionRunID := uuid.NewString()
     if t.RunID == "" { t.RunID = sessionRunID }
     if t.ID == "" { t.ID = uuid.NewString() }
     ```

2) Ensure events always include run/turn
   - Engines should set `metadata.Extra[run_id]` and `metadata.Extra[turn_id]` for every emitted event.
   - Files: `geppetto/pkg/steps/ai/openai/engine_openai.go`, `geppetto/pkg/steps/ai/claude/engine_claude.go`.
   - Pseudocode:
     ```go
     if metadata.Extra == nil { metadata.Extra = map[string]any{} }
     metadata.Extra[events.MetaKeyRunID] = t.RunID
     metadata.Extra[events.MetaKeyTurnID] = t.ID
     ```

3) Prevent tool-call 400s (input during tool loop)
   - Gate REPL input while `toolhelpers.RunToolCallingLoop` is running.
   - File: `pinocchio/cmd/agents/simple-chat-agent/pkg/ui/app.go`
   - Approach: disable Enter or queue user input while `isStreaming` and pending tool calls exist.

4) Persist agent-mode across restarts
   - Option A: keep `StaticService` + stable RunID (mode persists per run in memory).
   - Option B: switch to `SQLiteService` and store mode history for restarts.
   - Files: `geppetto/pkg/inference/middleware/agentmode/service.go` (use `SQLiteService`).

5) Improve snapshot timing
   - Take the `pre` snapshot after agent-mode injection but before provider call (pre-provider truth).
   - Optionally add a `post_tools` snapshot after tool execution.
   - File: `pinocchio/cmd/agents/simple-chat-agent/main.go` (adjust middleware order or add a pre-provider hook).

6) Expand SQL views for faster debugging
   - New view `v_turns_with_modes` with `run_id`, `turn_id`, Turn.Data mode, injected mode, and last agent-mode Info event.
   - New view `v_provider_messages` extracting user/assistant texts per turn from `block_payload_kv`.
   - File: `pinocchio/cmd/agents/simple-chat-agent/pkg/store/views.sql`.

### Quick grep patterns for logs

- Agent mode switches and insertions: `agentmode:`
- Mode switched: `agentmode: mode switched`
- Inserted prompt: `agentmode: user prompt inserted`
- YAML switch proposals: `mode_switch`

### Key files and symbols

- Events core: `geppetto/pkg/events/chat-events.go`
- OpenAI engine: `geppetto/pkg/steps/ai/openai/engine_openai.go`
- Claude engine + merger: `geppetto/pkg/steps/ai/claude/engine_claude.go`, `geppetto/pkg/steps/ai/claude/content-block-merger.go`
- Tool executor: `geppetto/pkg/inference/tools/executor.go`
- Agent mode middleware: `geppetto/pkg/inference/middleware/agentmode/middleware.go`
- Simple agent main: `pinocchio/cmd/agents/simple-chat-agent/main.go`
- UI app and sidebar: `pinocchio/cmd/agents/simple-chat-agent/pkg/ui/app.go`, `pinocchio/cmd/agents/simple-chat-agent/pkg/ui/sidebar.go`
- SQLite store: `pinocchio/cmd/agents/simple-chat-agent/pkg/store/sqlstore.go`, `schema.sql`, `views.sql`

### Closing notes

The event model is now simpler and more robust, the agent’s mode switching is visible and queryable, and we have a solid foundation to reason about each turn’s inputs/outputs. The next batch (UUIDs, input gating, snapshot timing, richer views, tests) will consolidate these gains and make the debugging and analytics experience first-class.


