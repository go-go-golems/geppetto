Title: How events are implemented in the streaming inference + tool-calling setup

Purpose
- Define the complete event lifecycle across engines and tool-calling helpers
- Specify when each event is emitted and from where in code
- Introduce context-carried sinks so downstream components (like tools) can publish events without plumbing

Event types (see `geppetto/pkg/events/chat-events.go`)
- start: inference started for a given engine step
- partial: streamed delta and accumulated completion
- tool-call: model requested a tool invocation (id, name, input)
- tool-result: tool returned a result (id, result)
- error: error condition in streaming or execution
- interrupt: context cancelled; includes partial text
- final: inference finished; includes final text

Event sinks
- Interface: `events.EventSink` with `PublishEvent(event Event) error`
- Implementations: e.g., watermill publisher (`middleware.WatermillSink`)
- Engine config (`engine.Config`) holds a list of sinks via `engine.WithSink`
- NEW: Context-carried sinks to empower downstream publishers
  - `events.WithEventSinks(ctx, sinks...)`
  - `events.GetEventSinks(ctx)`
  - `events.PublishEventToContext(ctx, event)`

Where events are published

OpenAI engine (file `geppetto/pkg/steps/ai/openai/engine_openai.go`)
- On start: `NewStartEvent` before opening the stream
- During stream loop:
  - partial: `NewPartialCompletionEvent` for each delta
  - error: on stream receive failures
  - interrupt: on context cancellation
  - tool-call (implicit): we currently log tool-call deltas and preserve them as `tool-use` messages; dedicated tool-call events can be published at block completion time if needed in future
- On completion: `NewFinalEvent` after assembling assistant text and tool-use messages
- Events are sent to both configured engine sinks and context-carried sinks

Claude engine (files `geppetto/pkg/steps/ai/claude/engine_claude.go` + `content-block-merger.go`)
- Streaming routed through `ContentBlockMerger`, which emits events:
  - start on `message_start`
  - partial on `text_delta`
  - tool-call on `content_block_stop` for tool_use blocks
  - error on API error
  - final on `message_stop`
- Engine publishes intermediate events yielded by the merger and final event

Tool-calling helper (file `geppetto/pkg/inference/toolhelpers/helpers.go`)
- After extracting tool calls:
  - publish `tool-call` events (best-effort serialization of args)
- After executing tools:
  - publish `tool-result` events (best-effort serialization of result)
- Uses `events.PublishEventToContext(ctx, ...)` so any caller that added sinks to the context sees these

Context-carried sinks design
- Motivation: allow tools and helpers to publish events without needing to modify engine structs or plumb sinks everywhere
- Pattern:
  1) At call site, combine sinks from engine config and attach to ctx using `events.WithEventSinks`
  2) Pass that ctx down to engines and helpers
  3) Downstream code calls `events.PublishEventToContext(ctx, e)` opportunistically
- Backward compatible: engines still publish via their own config sinks

Recommended integration path
1) When creating engines, also build an event router and a watermill sink
2) Attach the sink to the engine via `engine.WithSink`
3) Before invoking `RunInference`, attach the same sink(s) to the ctx:
   - `ctx = events.WithEventSinks(ctx, theSink)`
4) Run inference and, if using tool-calling helpers, pass the same ctx

Notes on ordering and idempotence
- Engines publish authoritative start/partial/final/error/interrupt events
- Helpers publish tool-call/tool-result events during orchestration
- Consumers should tolerate duplicates if multiple sinks or paths exist

Future enhancements
- OpenAI engine could emit explicit tool-call events once a tool-call block is fully merged (parity with Claude merger)
- Add more granular block-level events (block-start, block-stop) for UI fidelity
- Extend sinks with async/buffered behavior where needed



