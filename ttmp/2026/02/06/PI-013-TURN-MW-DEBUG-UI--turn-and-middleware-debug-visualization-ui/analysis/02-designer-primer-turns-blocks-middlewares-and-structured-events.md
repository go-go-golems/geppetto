---
Title: 'Designer Primer: Turns, Blocks, Middlewares, and Structured Events'
Ticket: PI-013-TURN-MW-DEBUG-UI
Status: active
Topics:
    - websocket
    - middleware
    - turns
    - events
    - frontend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/04-events.md
      Note: Primary conceptual source for event and sink model
    - Path: geppetto/pkg/doc/topics/08-turns.md
      Note: Primary conceptual source for turn and block model
    - Path: geppetto/pkg/doc/topics/09-middlewares.md
      Note: Primary conceptual source for middleware composition
    - Path: geppetto/pkg/events/structuredsink/filtering_sink.go
      Note: Structured sink extraction behavior explained for designers
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Defines snapshot phases and inference/tool iteration lifecycle
    - Path: pinocchio/pkg/webchat/router.go
      Note: Exposes /turns and /timeline and wires snapshot hook
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Event to SEM translation behavior referenced in primer
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: SEM to timeline projection behavior referenced in primer
    - Path: pinocchio/pkg/webchat/timeline_store.go
      Note: Timeline projection persistence interface referenced in primer
    - Path: pinocchio/pkg/webchat/turn_store.go
      Note: Turn snapshot persistence interface referenced in primer
ExternalSources: []
Summary: Plain-language, detailed primer for web designers to understand the runtime concepts behind turn/middleware/event debugging UIs. Revised to include deeper conceptual motivation explaining why turns replace messages, how blocks accumulate, and how middlewares compose prompting techniques.
LastUpdated: 2026-02-07T01:00:00-05:00
WhatFor: Give design collaborators enough technical context to make strong UX decisions for a turn and middleware debugging interface.
WhenToUse: Read before designing or reviewing UI flows for middleware/turn/event observability.
---


# Designer Primer: Turns, Blocks, Middlewares, and Structured Events

## Why this primer exists

The main PI-013 specification defines requirements and UX directions for a debugging web application. This companion primer answers a simpler but critical question:

"What are we actually looking at?"

If you are a web designer, your job is not to become a Go engineer. But to design a powerful debugging interface, you need a practical mental model of the objects and flows that developers are debugging.

This primer explains four core concepts in plain language, with enough depth for design decisions:

- Turns
- Blocks
- Middlewares
- Structured events

It also explains why these concepts matter for layout, interaction, information hierarchy, and visual semantics.

## The problem this system solves

Before diving into concepts, it helps to understand the problem.

Large Language Models (LLMs) are accessed through APIs. You send a request with context (instructions, previous messages, available tools) and the model responds with text, tool invocations, or both. Different providers (OpenAI, Anthropic/Claude, Google/Gemini) each have their own wire format for these requests and responses.

Most frameworks model this as a "chat": a list of messages with roles like "user", "assistant", "system". But **not every LLM interaction is a chat**. An LLM might:

- Summarize a document (one input, one output, no conversation).
- Run a multi-step agent loop where it calls tools repeatedly without any human input in between.
- Switch between different "modes" mid-conversation, each with different instructions and tool sets.
- Reason internally before responding (thinking/planning steps that aren't visible to the user).

The word "message" implies human communication. But what we really have is a **sequence of inference cycles**, each producing structured data. That is why this system uses the word **Turn** instead of "message" and **Block** instead of "content".

## One-sentence mental model

An interaction is a sequence of **turns**; each turn is an ordered list of **blocks** that grows during inference; **middlewares** are composable transformation steps that can inspect and modify turns before and after the model runs; **structured events** are the streaming telemetry emitted as all this happens.

## The big picture first

Imagine the system as two parallel tracks:

1. **State track** (slow, authoritative snapshots)
- A turn is a growing container of blocks. It starts with input blocks (system prompt, user prompt, prior context) and accumulates output blocks (model text, tool calls, tool results) as inference proceeds.
- The system takes snapshots at named phases (`pre_inference`, `post_inference`, `post_tools`, `final`) so you can see exactly what the turn looked like at each stage.

2. **Event track** (fast, streaming telemetry)
- While blocks are being added, the system also emits a stream of fine-grained events: `llm.start`, `llm.delta` (each word/token as it arrives), `tool-call`, `tool-result`, `final`, etc.
- These events are translated into a normalized format (SEM frames) and projected into timeline entities for the frontend to consume.

Design implication:

- A great debug UI should not pick one track only.
- It should help users understand *how state snapshots and event stream relate*.
- The state track shows "what is the state at this moment"; the event track shows "what happened to get there."

## Part 1: Turns (the container for one inference cycle)

### What is a turn?

A turn is a single container that holds everything relevant to one inference cycle. Think of it as a "working document" that the system reads from and writes to.

Before inference starts, a turn contains input blocks: the system prompt, any conversation history, and the latest user prompt. After inference completes, the same turn also contains the model's output: text responses, tool calls, tool results, and possibly reasoning traces.

Critically, **a turn is not a single message**. It is the full context window -- all the blocks that the model needs to see to produce its response, plus all the blocks it produces.

### How turns grow during a conversation

In a multi-turn conversation (like a chatbot), each new turn starts as a **clone of the previous turn** with the new user prompt appended. This means:

- Turn 1 might contain: `[system, user]` -> after inference: `[system, user, llm_text]`
- Turn 2 starts as a copy of Turn 1's final state with a new user block: `[system, user, llm_text, user]` -> after inference: `[system, user, llm_text, user, llm_text]`
- Turn 3 starts as a copy of Turn 2's final state with another user block, and so on.

Each turn is a complete snapshot. You can look at any turn in isolation and see the full context the model had at that point.

Design implication for the debug UI:

- A "conversation view" is really a list of turn snapshots, each longer than the last.
- A diff between Turn N and Turn N+1 shows what was added (new user prompt + model response).
- The designer should provide a way to view a single turn's full block list, and also to compare turns side-by-side.

### Why turns instead of messages

The word "Turn" was chosen deliberately over "message" because:

1. **Not all inference is conversational.** A document summarizer, a code generator, or an agent loop all use turns but have nothing to do with "chat messages."
2. **A turn contains many things, not one thing.** A single turn might hold a system prompt, three user messages, two assistant responses, a tool call, and a tool result. Calling that a "message" would be misleading.
3. **Provider neutrality.** OpenAI calls them "messages", Claude calls them "content blocks", Gemini has its own format. "Turn" is our neutral term that maps to all of them.

Design implication:

- Designers should think in terms of domain objects (turns and blocks), not provider-specific JSON.
- The UI should remain stable even if the backend switches between OpenAI, Claude, or Gemini.

### Turn structure

A turn has four parts:

- **ID**: unique identifier for this turn.
- **Blocks**: ordered list of content units (see Part 2).
- **Metadata**: observational data -- what happened during this turn (which model was used, how many tokens were consumed, what the stop reason was, correlation IDs for tracing).
- **Data**: configuration data -- what this turn was set up to do (tool configuration, agent mode, allowed tools).

Design implication:

- In an inspector UI, Metadata and Data should be separate tabs or sections.
- Metadata answers "what happened." Data answers "what this turn was configured to do."
- Both are important for debugging, but they serve different audiences and questions.

### Turn identity and correlation

Turn-level IDs are not cosmetic. They are the glue between different panes of the debug UI:

- `session_id`: identifies the overall interaction (a conversation, a batch run, etc.)
- `inference_id`: identifies one specific engine call within a turn
- `turn_id`: identifies this specific snapshot

These IDs should always be visible and copyable. Correlation chips at the top of an inspector panel are high-value UX.

## Part 2: Blocks (what a turn is made of)

### What is a block?

A block is the smallest atomic piece of content inside a turn. Each block has a **kind** that tells you what it represents:

| Kind | What it is | Who creates it | Example content |
|------|-----------|---------------|----------------|
| `system` | System instructions | Your application | "You are a helpful assistant. Use tools when needed." |
| `user` | User input | Your application | "What's the weather in Paris?" |
| `llm_text` | Model's text response | The inference engine | "The weather in Paris is currently 22C and sunny." |
| `tool_call` | Model asks to use a tool | The inference engine | `{name: "get_weather", args: {city: "Paris"}}` |
| `tool_use` | Result of running a tool | The tool executor | `{result: {temp: 22, conditions: "sunny"}}` |
| `reasoning` | Model's internal reasoning | The inference engine | Thinking/planning traces (often encrypted) |
| `other` | Catch-all | Various | Anything that doesn't fit the above |

### How blocks accumulate

This is one of the most important things to understand for UI design. Blocks are **appended to the turn as things happen**. Here is what a turn might look like at different moments during a single inference cycle with tool use:

**Before inference starts:**
```
[system, user]
```
The turn has the system prompt and the user's question.

**After the model responds with a tool call:**
```
[system, user, tool_call]
```
The engine appended a `tool_call` block asking to invoke a tool.

**After the tool runs and returns a result:**
```
[system, user, tool_call, tool_use]
```
The tool executor appended a `tool_use` block with the result.

**After the model sees the tool result and gives a final answer:**
```
[system, user, tool_call, tool_use, llm_text]
```
The engine ran inference again (seeing the tool result) and appended its final text.

This accumulation happens **within a single turn**. The turn is mutated in place -- blocks are added to the end as the inference cycle progresses.

Design implication:

- The debug UI should show this growth over time, not just the final state.
- A timeline or animation showing blocks appearing one by one would be very powerful.
- The snapshot phases (`pre_inference`, `post_inference`, `post_tools`) capture the turn at specific moments during this growth.

### Why block order matters

Order is semantic, not merely visual.

- A `tool_use` block must correspond to a previous `tool_call` by ID. If the IDs don't match, something is broken.
- Some providers require tool results to be placed immediately after tool calls (adjacency constraints).
- Middleware may reorder blocks to satisfy these provider constraints.

Design implication:

- The debug UI must show block order explicitly (numbered positions, not just a visual stack).
- Reordering by middleware should be visible as a first-class diff, not hidden.

### Block payloads and metadata

Each block carries:

- **Payload**: a map of content fields. The keys depend on the block kind:
  - Text blocks: `text`
  - Tool calls: `id`, `name`, `args`
  - Tool results: `id`, `result`, possibly `error`
- **Metadata**: annotations about where this block came from. For example, if a middleware inserted or modified a block, the metadata records which middleware did it.

Design implication:

- For payload display, use kind-specific renderers first (a nice tool-call card, a text display, etc.), with raw JSON as a fallback.
- A subtle "provenance badge" on each block card showing its origin (e.g., "inserted by system-prompt middleware") is valuable for debugging.

## Part 3: Middlewares (composable prompting techniques)

### The core idea

Here is where many frameworks fall short and where this system shines.

In most LLM frameworks, prompting is just "build a string and send it." If you want to add a system prompt, you concatenate it. If you want to inject tool instructions, you concatenate more. If you want to enforce formatting, you add more text. The result is a fragile, monolithic prompt-construction function that is hard to test, debug, or compose.

**Middlewares solve this.** A middleware is a wrapper that can inspect and transform a turn before it goes to the model, and also inspect and transform the result after the model responds. Middlewares compose: you can stack multiple middlewares and each one does its specific job.

### What middleware can do

A middleware is not just "logging around an API call." Middlewares are **composable prompting techniques**. They can:

1. **Add or replace blocks.** The system-prompt middleware ensures a system block exists and has the right content. If the system block is missing, it adds one. If it exists but has wrong content, it replaces it.

2. **Reorder blocks.** The tool-result-reorder middleware rearranges blocks so that tool results sit immediately after their corresponding tool calls. Some LLM providers reject requests where tool results are out of order.

3. **Inject contextual guidance.** The agent-mode middleware inserts a guidance block near the end of the turn telling the model what "mode" it's currently in (e.g., "research mode" vs. "coding mode") and what tools are available.

4. **Parse model output and trigger actions.** Some middlewares examine the model's response after inference. For example, the agent-mode middleware looks for specially formatted YAML blocks in the model's output that signal a mode switch, then updates the turn's configuration accordingly.

5. **Register or modify tool availability.** The SQLite-tool middleware registers a database query tool into the runtime. This doesn't change the turn's text at all -- it changes what tools are available for the model to call.

6. **Annotate blocks with provenance.** When a middleware inserts or modifies a block, it tags the block's metadata with its own name. This creates an audit trail of "who changed what."

### The middleware chain

Middlewares execute in a specific order, nested like layers of an onion:

```
Request arrives
  -> Middleware 1 (e.g., logging) sees the turn, maybe modifies it
    -> Middleware 2 (e.g., system prompt) adds/updates system block
      -> Middleware 3 (e.g., tool reorder) reorders blocks
        -> Engine: calls the LLM API, gets response, appends output blocks
      <- Middleware 3 sees the result
    <- Middleware 2 sees the result
  <- Middleware 1 sees the result
Response returned
```

Each middleware wraps the next one. A middleware can:
- Modify the turn **before** passing it inward (pre-processing).
- Modify the turn **after** getting it back (post-processing).
- Short-circuit and not call the next middleware at all (e.g., a safety filter that rejects harmful input).

Design implication:

- The UI should show the middleware chain order clearly.
- For each middleware, a "before vs. after" diff view is ideal.
- Some middleware effects are invisible in the text (e.g., registering a tool, updating Data fields). The UI must surface these "invisible" changes too.

### Why middleware debugging is hard today

Today, the system takes snapshots at tool-loop phases (`pre_inference`, `post_inference`, `post_tools`), but it does **not** take snapshots between individual middlewares. This means developers see the turn before the entire middleware chain runs and after it finishes, but they cannot see what each individual middleware did.

This causes ambiguity:

- "Was this system block inserted by the system-prompt middleware or the agent-mode middleware?"
- "Did the reordering happen because of the tool-reorder middleware, or was the model's output already in this order?"
- "A block appeared between pre_inference and post_inference -- was it added by a middleware or by the engine?"

Design implication:

- The debug UI should make middleware attribution explicit when available (via block metadata provenance tags).
- Where attribution is inferred rather than explicit, the UI should label it accordingly (e.g., "likely added by system-prompt middleware" vs. "confirmed: added by system-prompt middleware").

## Part 4: Structured events (runtime telemetry stream)

### What is an event?

While blocks accumulate inside a turn (the state track), the system also emits a parallel stream of **events** (the telemetry track). These are timestamped signals that describe what is happening in real time:

- `llm.start`: the model started generating
- `llm.delta`: the model produced another chunk of text (streaming)
- `tool-call`: the model requested a tool invocation
- `tool-result`: a tool returned its result
- `final`: inference is complete
- `agent_mode_switch`: the agent changed modes
- `debugger.pause`: the step controller paused execution

Events carry correlation metadata (session ID, inference ID, turn ID) so they can be linked back to the specific turn and inference cycle they belong to.

### Why events are separate from turns

Turns and events answer different questions:

| | Turns (state track) | Events (telemetry track) |
|---|---|---|
| **Question answered** | "What is the state right now?" | "What happened over time?" |
| **Granularity** | Coarse (snapshot phases) | Fine (per-token, per-action) |
| **Persistence** | Durable (stored in DB) | Often ephemeral (streamed) |
| **Use case** | Comparing before/after states | Watching real-time progress |

Design implication:

- The debug UI needs both: a state inspector for turns/blocks and a timeline/log for events.
- The two should be linked: clicking a turn snapshot should highlight the events that occurred during that phase.

### Structured sink extraction: when the model outputs structured data inside prose

One particularly interesting pattern is the **structured sink**. Sometimes we ask the model to embed structured data (like YAML) inside its prose output, wrapped in special XML-like tags:

```
Here is my analysis of the situation...

<$myapp:ModeSwitch:v1>
```yaml
new_mode: research
reason: "Need to gather more information before coding"
```
</$myapp:ModeSwitch:v1>

Based on this, I'll switch to research mode and look for...
```

The **FilteringSink** watches the streaming text, detects these tagged blocks, extracts the YAML payload, and:
1. Emits the clean prose text (without the tags) downstream for display.
2. Emits a typed structured event (`ModeSwitch` with the parsed YAML data) for programmatic handling.

This is a composable prompting technique: you can ask the model to output structured control signals inside natural language, and the middleware/sink infrastructure handles parsing and routing automatically.

Design implication:

- The most revealing UI for this is a three-column view:
  - **Raw text**: exactly what the model produced (tags and all)
  - **Filtered text**: what the user sees (tags removed)
  - **Extracted events**: the structured data that was parsed out
- This helps developers verify that extraction is working correctly and that the model is producing valid structured output.

### SEM translation and timeline projection

Events are translated into a normalized envelope format called **SEM** (Structured Event Model) for the frontend:

- `event.type`: what kind of event
- `event.id`: unique event identifier
- `event.seq`: sequence number (for ordering)
- `event.stream_id`: which stream this belongs to
- `event.data`: the event payload

SEM events are then **projected** into timeline entities -- higher-level objects like `message`, `tool_call`, `tool_result`, `thinking_mode`. These projections are stored with monotonic versioning, which supports:

- **Hydration**: loading the full timeline state when a client connects
- **Incremental updates**: "give me everything since version N"
- **Replay**: reconstructing the timeline from persisted events

Design implication:

- The event inspector should show both the original event meaning and the wire-level SEM envelope.
- The projection layer is a separate "truth" from both raw events and turn state -- it deserves its own lane in the UI.

## Putting it together: one concrete walkthrough

Here is a complete flow for one user prompt in a multi-turn conversation with tool use. Follow along to see how turns, blocks, middlewares, and events interact:

**Step 1: New turn created.**
The session clones the previous turn (preserving full history) and appends a new `user` block with the latest prompt. The turn now contains all previous context plus the new question.

**Step 2: Tool loop takes `pre_inference` snapshot.**
The turn is captured as-is before any middleware or engine processing.

**Step 3: Middleware chain runs (pre-processing).**
- System-prompt middleware checks that a system block exists, adds or updates it.
- Agent-mode middleware reads the current mode from Turn.Data and injects a guidance block.
- Tool-reorder middleware ensures any existing tool results are in the right position.

Each middleware tags the blocks it touches with provenance metadata.

**Step 4: Engine calls the LLM API.**
The engine translates the turn's blocks into the provider's wire format (OpenAI messages, Claude content blocks, etc.), sends the request, and streams the response. During streaming:
- Events emit: `llm.start`, then many `llm.delta` (one per text chunk).
- If the model decides to call a tool: `tool-call` event emitted, `tool_call` block appended to the turn.

**Step 5: Tool loop takes `post_inference` snapshot.**
The turn now includes the model's output blocks.

**Step 6: Tool execution (if tool calls are pending).**
The loop extracts pending tool calls (tool_call blocks without matching tool_use blocks), executes the tools, and appends `tool_use` blocks with results.
Events emit: `tool-call-execute`, `tool-call-execution-result`.

**Step 7: Tool loop takes `post_tools` snapshot.**
The turn now includes tool results.

**Step 8: Loop repeats.**
If the model made tool calls, the loop goes back to Step 2 with the updated turn (the model needs to see the tool results and respond). This can repeat multiple times (up to a configured limit).

**Step 9: Final response.**
When the model responds with just text (no more tool calls), the loop exits. The `final` event is emitted. The turn is persisted.

**Step 10: Frontend consumption.**
Events are translated to SEM envelopes, projected into timeline entities, and delivered to the frontend via WebSocket. The frontend hydrates its state and renders the conversation.

### What a developer wants to know when debugging this

- Which middleware changed which blocks? (attribution)
- Did tool call IDs match tool result IDs? (correctness)
- Did the model see the right context? (turn content at pre_inference)
- Did structured extraction parse the YAML correctly? (sink behavior)
- Did the projection sequence stay monotonic? (ordering invariant)
- Why did the agent switch modes? (middleware post-processing)

### What the designer should provide

- Clear causal chain across these steps.
- Fast drill-down from summary anomaly to raw evidence.
- Visual correlation between the state track and event track.

## Why this matters for interface design quality

A weak debug UI will show "lots of JSON" and force users to manually compare logs.

A strong debug UI will:

- **Tell a coherent story.** The user should see "input -> middleware transformations -> model response -> tool execution -> next iteration" as a clear visual flow, not a pile of data.
- **Keep IDs and correlations always accessible.** Every block, event, and snapshot should show its correlation IDs so users can trace connections across panes.
- **Show diffs as first-class visual objects.** When a middleware changes a block, the change should be highlighted like a code diff, not require manual JSON comparison.
- **Surface hidden state changes.** Not all changes are visible in text. Metadata updates, Data configuration changes, and tool registration changes should have their own display.
- **Separate snapshot state from event flow without disconnecting them.** Two parallel tracks, visually linked by time and correlation IDs.

This is exactly why the PI-013 spec asks for multi-lane synchronized design.

## Practical design checklist

Use this as a quick handoff guide while sketching.

### Must-have visual primitives

- Timeline lanes (state snapshots + streaming events + projection entities)
- Block cards with kind-specific rendering (text display, tool-call card, tool-result card)
- Diff badges on blocks (added, removed, changed, reordered)
- Provenance badges showing which middleware touched a block
- Correlation chips (`session_id`, `inference_id`, `turn_id`) -- always visible, always copyable
- Expandable payload tree for raw inspection

### Must-have interactions

- Click any node -> cross-highlight correlated nodes in all lanes
- Compare any two snapshots (A/B diff of block lists)
- Filter by event type, snapshot phase, middleware, block kind
- Toggle between raw / decoded / semantic payload views
- Pin anomalies for sharing or export

### Must-have trust signals

- Explicit vs. inferred attribution labels ("confirmed" vs. "likely")
- Unknown/unhandled event type indicators
- Ordering/monotonicity warnings (out-of-sequence events)
- Truncation/throttling markers (when data was dropped or summarized)

## Glossary (designer-oriented)

- **Session / Run**: one interaction execution context (a conversation, a batch job, etc.).
- **Turn**: the container for one inference cycle. Holds all blocks (input + output) as an ordered list. In a conversation, each turn is a growing snapshot of the full context.
- **Block**: the smallest content unit inside a turn. Has a kind (system, user, llm_text, tool_call, tool_use, reasoning, other), a payload, and metadata.
- **Middleware**: a composable wrapper that can inspect and transform a turn before and/or after inference. Used for prompting techniques, safety filters, block reordering, mode switching, etc.
- **Middleware chain**: the ordered sequence of middlewares that a turn passes through. Nested execution: outermost runs first on the way in, last on the way out.
- **Snapshot phase**: a named moment when the turn state is captured: `pre_inference`, `post_inference`, `post_tools`, `final`.
- **Event**: a timestamped runtime signal emitted during processing (text delta, tool call, mode switch, etc.).
- **Structured sink / FilteringSink**: a component that watches streaming text for tagged YAML blocks, extracts them, and emits typed events.
- **SEM frame**: the normalized event envelope format used by the frontend (type, id, seq, stream_id, data).
- **Projection**: a materialized timeline entity (message, tool_call, etc.) derived from SEM events and stored with monotonic versioning.
- **Hydration**: loading persisted projection state into the frontend store when a client connects.
- **Upsert**: insert a new entity or update an existing one, identified by stable ID.
- **Provenance**: metadata on a block recording which middleware or component created/modified it.

## Final note for designers

You are designing an interface for expert users who debug causality, not only content. The highest value is reducing time-to-understanding when behavior looks wrong.

If a developer can answer "what changed, where, and why" in under 30 seconds, the design succeeded.
