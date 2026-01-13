## Migration analysis: Align tools with unified streaming events (OpenAI Responses, Anthropic)

### Scope
- Update downstream tools and UIs to the unified event surface introduced by the OpenAI Responses integration and the unified events spec in `ttmp/2025-10-17/11-unified-events-openai-anthropic.md`.
- Targets: `geppetto/cmd/examples/` programs and `pinocchio/cmd/agents/` (simple-chat-agent UI/backend/store).

### What changed since 6edae3894c5f164ec9ef2c65628d67612f8bb0cd
- New engine: `pkg/steps/ai/openai_responses/engine.go` (streaming SSE from OpenAI Responses) emitting Geppetto events.
- Expanded event surface in `pkg/events/chat-events.go`:
  - Reasoning: `EventThinkingPartial`, `EventReasoningTextDelta`, `EventReasoningTextDone`, plus `EventInfo` markers: `thinking-started|ended`, `output-started|ended`, `reasoning-summary-started|ended`, and final summary `EventInfo{"reasoning-summary", data.text}`.
  - Tool calls: `EventToolCall`, `EventToolResult`, `EventToolCallExecute`, `EventToolCallExecutionResult` (unchanged, but now emitted by Responses engine when `function_call` items complete).
  - Built-ins (server tools): Web search progress/result events: `EventWebSearchStarted|Searching|OpenPage|Done`; normalized results `EventToolSearchResults` (available, not yet emitted by Responses engine).
  - Citations: `EventCitation` attached to streamed output text.
  - Other capabilities present on the surface (future/optional): File search, Code interpreter, MCP, Image generation.
- Pretty/structured printing utilities extended in `pkg/events/step-printer-func.go` and `pkg/events/event-router.go`.

### How the OpenAI Responses engine maps provider events to Geppetto events
- Message lifecycle
  - Emits `NewStartEvent` when request starts; `NewFinalEvent` with `EventMetadata{Usage, StopReason, DurationMs, Extra{reasoning_tokens, thinking_text, saying_text, reasoning_summary_text}}` at end.
- Text
  - `response.output_text.delta` → `EventPartialCompletion{Delta, Completion}`.
  - `response.output_item.added: {type:"message"}` → `EventInfo("output-started")`; corresponding `output_item.done` → `EventInfo("output-ended")`.
- Reasoning
  - `response.output_item.added: {type:"reasoning"}` → `EventInfo("thinking-started")`; `output_item.done` → `EventInfo("thinking-ended")` and a `turns.Block{Kind: Reasoning, Payload.encrypted_content?}` appended.
  - `response.reasoning_summary_part.added` → `EventInfo("reasoning-summary-started")`.
  - `response.reasoning_summary_text.delta` → `EventThinkingPartial{Delta, Completion}`.
  - `response.reasoning_summary_part.done` → `EventInfo("reasoning-summary-ended")`; also one `EventInfo{"reasoning-summary", data.text}` is emitted at the end with full summary text.
- Tool calls (function tools)
  - `response.function_call_arguments.delta|done` are aggregated by `item_id`. On `response.output_item.done{type:"function_call"}` the engine emits `EventToolCall{ToolCall{ID: call_id, Name, Input: argsJSON}}` and appends a `turns.Block` for the tool call.
- Built-in/server tools: web search
  - `response.output_item.added{type:"web_search_call", action.type==search/open_page}` → `EventWebSearchStarted(query)` or `EventWebSearchOpenPage(url)`.
  - `response.web_search_call.in_progress|searching|completed` → `EventWebSearchStarted` (no query yet) | `EventWebSearchSearching` | `EventWebSearchDone`.
- Citations
  - `response.output_text.annotation.added` → `EventCitation{title,url,start_index,end_index,output_index,content_index,annotation_index}`.

### Impacted downstream code and required updates

#### geppetto/cmd/examples/
- `cmd/examples/openai-tools/main.go` already integrates `EventRouter` and:
  - Pretty prints `EventInfo` reasoning/output boundaries and `EventThinkingPartial`.
  - Supports a "server-tools" mode to attach Responses built-in `web_search` and will receive `EventWebSearch*` events.
  - Uses `StepPrinterFunc` which now prints `EventWebSearch*` and `EventCitation`.
- No changes needed in `geppetto/examples/` YAML/text samples.

Action: None required beyond validating that web search events render as expected in server-tools mode.

#### pinocchio/cmd/agents/simple-chat-agent (UI/backend/store)
Current handling focuses on: `EventPartialCompletion(Start)`, `EventFinal`, `EventError`, `EventInterrupt`, `EventToolCall`, `EventToolCallExecute`, `EventToolResult`, `EventToolCallExecutionResult`, `EventLog`, `EventInfo` (generic). It does not yet explicitly handle the new event kinds below.

Required updates:
- UI pretty and forwarders
  - Add explicit handling for `EventWebSearchStarted|Searching|OpenPage|Done` in:
    - `pkg/xevents/events.go` (AddPrettyHandlers): render nice status lines and URLs.
    - `pkg/ui/app.go` and `pkg/ui/host.go`: forward to sidebar/timeline; update `status` field accordingly.
  - Add handling for `EventCitation` in pretty and UI: show title/url lines during streaming; optionally persist on the final text block or a dedicated citation list.
  - Add support for `EventThinkingPartial` to stream reasoning summary text to the UI (separate area or subdued style). Also interpret `EventInfo` messages: `thinking-started|ended`, `reasoning-summary-started|ended`, `output-started|ended` for clean section headers.
- Timeline backend (`pkg/backend/tool_loop_backend.go`)
  - Forward `EventWebSearch*` to dedicated timeline entities (e.g., `Kind: "web_search"` with `props{query,url}`) and mark completion on `Done`.
  - Optional: Forward `EventCitation` into the text entity or as separate child entities.
- Sidebar aggregation (`pkg/ui/sidebar.go`)
  - Add entries for web search operations keyed by `ItemID` with live progress; show `OpenPage` URLs.
  - Keep existing tool aggregation untouched for function tools; web search is a server tool with distinct events.
- Storage (`pkg/store/sqlstore.go`)
  - Extend `LogEvent` to capture and persist `EventWebSearch*` and `EventCitation` (store `item_id`, `query`, `url`, indices when available).
- Developer diagnostics
  - Ensure `pkg/ui/debug_commands.go` lists and renders new server-tool events in the timeline/debug outputs, if relevant.

Optional upgrades (future-ready, minimal effort now):
- Recognize `EventReasoningTextDelta|Done` if we later emit raw thinking text (separate from summary), and surface in UI under a “Thinking” view.
- Recognize `EventToolSearchResults` if/when normalized results are published.
- Recognize MCP, File search, Code interpreter, Image generation events if introduced by other engines/providers.

### Backward compatibility considerations
- Existing handling of `EventToolCall*`, `EventPartialCompletion*`, `EventFinal`, `EventLog`, `EventInfo` remains valid.
- New events are additive; adding handlers is safe and won’t break older providers.
- `EventInfo` is used as phase markers; unrecognized `EventInfo.Message` values can continue to fall back to generic rendering.

### Concrete update checklist
- [ ] UI: Pretty printer (`pkg/xevents/events.go`) renders `EventWebSearch*`, `EventCitation`, and `EventThinkingPartial`.
- [ ] UI: `pkg/ui/app.go` + `pkg/ui/host.go` handle and display new events; update `status` and views accordingly.
- [ ] UI: `pkg/ui/sidebar.go` aggregates web search events per `ItemID` and shows progress/urls.
- [ ] Backend: `pkg/backend/tool_loop_backend.go` forwards new events to timeline entities.
- [ ] Store: `pkg/store/sqlstore.go` persists `EventWebSearch*` and `EventCitation` fields.
- [ ] Debug: `pkg/ui/debug_commands.go` reflects new events in its inspectors.
- [ ] Validate `cmd/examples/openai-tools` server-tools mode: confirm web search progress and citations are visible via router printers.

### References
- Unified spec: `ttmp/2025-10-17/11-unified-events-openai-anthropic.md`
- Engine: `pkg/steps/ai/openai_responses/engine.go`
- Events: `pkg/events/chat-events.go`, `pkg/events/step-printer-func.go`, `pkg/events/event-router.go`
- Example CLI: `cmd/examples/openai-tools/main.go`
- Agent UI/Backend/Store: `pinocchio/cmd/agents/simple-chat-agent/...`


