---
Title: Documentation Improvement Proposal for Geppetto Core Docs
Ticket: PI-013-TURN-MW-DEBUG-UI
Status: active
Topics:
    - middleware
    - turns
    - events
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/06-inference-engines.md
      Note: Current inference engine doc that needs bigger-picture motivation section
    - Path: geppetto/pkg/doc/topics/08-turns.md
      Note: Current turns doc that needs conceptual grounding section
    - Path: geppetto/pkg/doc/topics/09-middlewares.md
      Note: Current middlewares doc that needs richer explanation of prompting composition
    - Path: geppetto/pkg/doc/topics/04-events.md
      Note: Current events doc that should be cross-linked to structured sink explanation
    - Path: geppetto/pkg/events/structuredsink/filtering_sink.go
      Note: Structured sink implementation that deserves its own topic doc
    - Path: geppetto/pkg/inference/session/session.go
      Note: Session management code that is underdocumented in topics
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Tool loop with snapshot phases that should be documented in topics
ExternalSources: []
Summary: Proposes specific improvements to geppetto/pkg/doc/topics/ documents to better explain the bigger picture of why turns replace messages, how blocks accumulate during inference, how middlewares compose prompting techniques, and how the full runtime flow connects.
LastUpdated: 2026-02-07T01:30:00-05:00
WhatFor: Guide improvements to geppetto core documentation so that new developers and cross-functional collaborators can quickly understand the system's design philosophy.
WhenToUse: Reference when planning documentation sprints or when onboarding new team members and finding the existing docs insufficient.
---


# Documentation Improvement Proposal for Geppetto Core Docs

## Purpose

The existing geppetto topic docs (`08-turns.md`, `09-middlewares.md`, `06-inference-engines.md`, `04-events.md`) are technically accurate and useful as API references. However, they lack a crucial layer: **conceptual motivation**. They explain *what* the abstractions are and *how* to use them, but not *why* they were designed this way or *how they connect* into a coherent system.

This proposal identifies specific gaps and suggests concrete additions to make the documentation tell a complete story.

## Audience for this proposal

- Documentation authors (yourself, future contributors)
- Anyone planning a docs improvement sprint
- Reviewers assessing whether docs are "onboarding ready"

## General principles for the improvements

1. **Lead with "why", then "what", then "how".** Every topic doc should open with the problem being solved, not the API surface.
2. **Connect the pieces.** Each doc should explain where its concept sits in the larger runtime flow, with forward/backward references to related docs.
3. **Use concrete examples from the actual system.** Abstract descriptions are less useful than walking through what happens when a user sends a prompt.
4. **Distinguish audiences.** Some readers want API reference, others want conceptual understanding. Section headers should make it easy to skip to the right depth.

---

## Document-specific proposals

### 1. `08-turns.md` -- Turns and Blocks

#### Current state

The document is a solid API/tutorial reference. It covers type definitions, block kinds, typed keys, serialization, and engine mapping. It includes code examples and a "Multi-turn Sessions" section.

#### What is missing

**A. The "why not messages?" motivation section.**

The document jumps straight into "Why Turns?" with "Every AI conversation is a sequence of messages" and then says turns provide a unified model. But it doesn't explain the deeper design decision:

- Not all LLM inference is conversational. Turns are a general-purpose container for inference cycles, not chat messages.
- A turn contains the *entire context window*, not a single message. It holds system prompts, prior conversation history, tool calls, tool results, and model output -- all in one ordered list.
- The word "turn" was chosen deliberately to avoid the chat-centric connotations of "message."

**Proposed addition:** A new section called "Why 'Turn' instead of 'Message'" between the current "Why Turns?" and "Core Concepts" sections. 2-3 paragraphs explaining the design philosophy.

**Suggested content outline:**

```
## Why 'Turn' instead of 'Message'

Most LLM frameworks model interactions as a list of chat messages. This works for
simple chatbots but breaks down for:

- Document processing (one input, one output, no conversation)
- Agent loops (model calls tools repeatedly without human input)
- Multi-mode agents (different instructions and tools per mode)
- Reasoning/planning (internal steps that aren't "messages" to anyone)

A Turn is a general-purpose container for one inference cycle. It holds everything
the model needs to see (input blocks) and everything it produces (output blocks),
regardless of whether the interaction is a chat, a batch job, or an agent loop.

The word "Turn" avoids the conversational connotations of "message" and correctly
implies that the model takes a turn (like in a board game) -- it receives context,
reasons, and produces output.
```

**B. How blocks accumulate during inference.**

The document describes blocks statically (type definitions, kinds, payload keys) but does not walk through how blocks grow over the course of a single inference cycle. This is critical for anyone building middleware or debugging tools.

**Proposed addition:** A new section called "How Blocks Accumulate" after "Block Kinds". A step-by-step walkthrough showing a turn's block list at different moments:

```
## How Blocks Accumulate During Inference

A turn starts with input blocks and grows as inference proceeds:

1. Application creates turn: [system, user]
2. Engine runs, model responds with tool call: [system, user, tool_call]
3. Tool executor runs tool, appends result: [system, user, tool_call, tool_use]
4. Engine runs again, model gives final answer: [system, user, tool_call, tool_use, llm_text]

This happens in-place on a single Turn object. The same pointer is mutated
throughout the inference cycle. Middlewares see and can modify the evolving
context at each step.
```

**C. How turns grow across a conversation.**

The "Multi-turn Sessions" section mentions `AppendNewTurnFromUserPrompt` but doesn't explain the "clone previous turn + append" mental model visually.

**Proposed addition:** A visual example showing how Turn N+1 relates to Turn N:

```
## Turn Growth in Multi-Turn Sessions

Turn 1 (after inference): [system, user_1, llm_text_1]
Turn 2 = clone(Turn 1) + user_2: [system, user_1, llm_text_1, user_2]
Turn 2 (after inference): [system, user_1, llm_text_1, user_2, llm_text_2]
Turn 3 = clone(Turn 2) + user_3: [system, user_1, llm_text_1, user_2, llm_text_2, user_3]

Each turn is a complete snapshot. You can look at any turn in isolation and
see the full context the model had at that point.
```

---

### 2. `09-middlewares.md` -- Middlewares

#### Current state

The document covers the interface, a logging example, composition order, and best practices. It is brief and focused on API mechanics.

#### What is missing

**A. Middlewares as composable prompting techniques.**

The document frames middleware primarily as cross-cutting concerns (logging, safety, tracing, rate limiting). These are valid use cases but miss the most powerful and distinctive capability: **middleware as composable prompting techniques**.

In this system, middlewares don't just observe or gate inference -- they actively shape what the model sees and how its output is processed. The system-prompt middleware is a prompting technique (ensuring the system prompt is correct). The agent-mode middleware is a prompting technique (injecting mode-specific instructions). The structured sink is a prompting technique (asking the model to output structured data and parsing it).

**Proposed addition:** A new section called "Middleware as Composable Prompting" after "Why Middlewares?". This should explain:

- The difference between "infrastructure middleware" (logging, rate limiting) and "prompting middleware" (system prompt injection, mode switching, block reordering, structured output parsing).
- Why composition matters: you can develop and test each prompting technique independently, then combine them.
- How this is different from monolithic prompt-construction functions.

**Suggested content outline:**

```
## Middleware as Composable Prompting

Most frameworks treat prompt construction as a single function that builds a string.
Middleware inverts this: each prompting technique is a separate, composable wrapper
that adds its contribution to the Turn.

Infrastructure middleware (logging, rate limiting, safety) wraps inference for
operational concerns. Prompting middleware shapes what the model sees:

- System-prompt middleware: ensures the right system instructions are present
- Agent-mode middleware: injects mode-specific guidance and tool restrictions
- Tool-reorder middleware: satisfies provider ordering constraints
- Structured-output middleware: asks the model to embed structured data and parses it

Each technique is:
- Independent: develop and test in isolation
- Composable: stack with other techniques without interference
- Observable: tags blocks with provenance metadata for debugging

This is the primary design innovation of the middleware system.
```

**B. Richer real-world examples.**

The current examples are a logging middleware and brief mention of ordering. The document should include at least one "prompting middleware" example that shows block mutation, not just logging.

**Proposed addition:** Add a concrete example of a middleware that modifies blocks, such as a simplified system-prompt middleware:

```go
systemPromptMw := func(next middleware.HandlerFunc) middleware.HandlerFunc {
    return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
        // Check if system block exists
        found := false
        for i, b := range t.Blocks {
            if b.Kind == turns.BlockKindSystem {
                t.Blocks[i].Payload[turns.PayloadKeyText] = "You are a helpful assistant."
                turns.KeyBlockMetaMiddleware.Set(&t.Blocks[i].Metadata, "systemprompt")
                found = true
                break
            }
        }
        if !found {
            block := turns.NewSystemTextBlock("You are a helpful assistant.")
            turns.KeyBlockMetaMiddleware.Set(&block.Metadata, "systemprompt")
            turns.PrependBlock(t, block)
        }
        return next(ctx, t)
    }
}
```

**C. Post-processing middleware pattern.**

The document doesn't show a middleware that modifies the result after inference. This is important for understanding agent-mode middleware (which parses model output) and structured sinks.

**Proposed addition:** A section or example showing the post-processing pattern:

```go
postProcessMw := func(next middleware.HandlerFunc) middleware.HandlerFunc {
    return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
        result, err := next(ctx, t)
        if err != nil { return result, err }

        // Examine model output and take action
        for _, b := range result.Blocks {
            if b.Kind == turns.BlockKindLLMText {
                // Parse structured content, update Turn.Data, emit events, etc.
            }
        }
        return result, nil
    }
}
```

---

### 3. `06-inference-engines.md` -- Inference Engines

#### Current state

The document is thorough: 30-second overview, engine interface, factories, basic inference, tool calling (manual and automated), provider implementations, middleware, testing, best practices, and debugging.

#### What is missing

**A. The runtime flow as a connected story.**

The document explains each component but doesn't connect them into a single narrative flow. A reader who finishes the document understands engines, tool loops, and factories separately but may not have a clear picture of how a request flows through the entire system.

**Proposed addition:** A new section called "Complete Runtime Flow" (or improve the existing tool-loop section) that walks through a single request from session creation to final response, showing how session, middleware chain, engine, tool loop, events, and persistence interact. A diagram would be very valuable here.

**Suggested content:**

```
## Complete Runtime Flow

When a user sends a prompt in a multi-turn application, here is the full path:

1. Session.AppendNewTurnFromUserPrompt("question")
   - Clones the latest turn (preserving full history)
   - Appends a new user block
   - Assigns a new TurnID

2. Session.StartInference(ctx)
   - Creates an ExecutionHandle
   - Launches a goroutine running the engine builder's runner

3. Runner attaches event sinks to context

4. If tool registry is present, creates a toolloop.Loop

5. Tool loop iterates:
   a. Snapshot: pre_inference
   b. Middleware chain processes the turn (pre-processing)
   c. Engine calls the LLM API (streams events: start, delta, final)
   d. Engine appends output blocks (llm_text, tool_call)
   e. Middleware chain processes the result (post-processing)
   f. Snapshot: post_inference
   g. Extract pending tool calls
   h. If none: done
   i. Execute tools, append tool_use blocks
   j. Snapshot: post_tools
   k. Go to (a) for next iteration

6. Final turn persisted (if persister configured)
7. ExecutionHandle receives result
8. Caller retrieves via handle.Wait()
```

**B. Context-based dependency injection explanation.**

The document mentions `tools.WithRegistry(ctx, registry)` and `events.WithEventSinks(ctx, sink)` but doesn't explain the pattern as a design principle. This is unusual and worth calling out.

**Proposed addition:** A brief subsection in "Core Architecture Principles" explaining that the system uses `context.Context` for dependency injection of event sinks, tool registries, and snapshot hooks, and why (no global state, easy testing, supports parallel runs).

---

### 4. `04-events.md` -- Events

#### Current state

(Not reviewed in detail for this proposal, but referenced by the other docs.)

#### What is missing

**A. Cross-reference to structured sink extraction.**

The events doc should explain or link to the FilteringSink pattern, where streaming text events are watched for tagged structured blocks, which are extracted and re-emitted as typed events.

**B. Event-to-turn correlation.**

The doc should explain how events carry correlation IDs (session_id, inference_id, turn_id) and how these link events back to specific turn snapshots.

---

### 5. New topic: Session management

#### Why

There is no topic doc for `session.Session`. The session is the top-level orchestrator for multi-turn interactions and is referenced in both the turns doc and the engines doc, but neither explains it fully.

**Proposed:** A new topic doc (`10-sessions.md` or similar) covering:
- What a session is (ordered history of turn snapshots + exclusive active inference)
- How turns are created via clone + append
- How inference is started and awaited
- How sessions connect to persistence
- The relationship between Session, EngineBuilder, and toolloop.Loop

---

### 6. New topic: Structured sinks and the FilteringSink

#### Why

The structured sink pattern is one of the most distinctive and powerful features of the system. It enables composable prompting techniques where the model produces structured control signals inside natural language, and the infrastructure handles parsing and routing. There is no topic doc for this.

**Proposed:** A new topic doc covering:
- The problem: getting structured data out of LLM text streams
- The tagging convention (`<$pkg:Type:vN>` ... `</$pkg:Type:vN>`)
- How the FilteringSink works (watches stream, extracts blocks, emits events)
- How to register extractors
- Malformed block handling policies
- Integration with middleware (e.g., agent-mode middleware uses structured sink to detect mode switches)

---

## Summary of proposed changes

| Document | Change type | Description |
|----------|------------|-------------|
| `08-turns.md` | New section | "Why 'Turn' instead of 'Message'" -- design philosophy |
| `08-turns.md` | New section | "How Blocks Accumulate During Inference" -- step-by-step walkthrough |
| `08-turns.md` | Enhancement | Visual example of turn growth across a conversation |
| `09-middlewares.md` | New section | "Middleware as Composable Prompting" -- key design insight |
| `09-middlewares.md` | New example | Block-mutating middleware (system prompt) |
| `09-middlewares.md` | New example | Post-processing middleware pattern |
| `06-inference-engines.md` | New section | "Complete Runtime Flow" -- connected narrative with diagram |
| `06-inference-engines.md` | Enhancement | Context-based DI explanation in architecture principles |
| `04-events.md` | Enhancement | Cross-reference to structured sink, correlation ID explanation |
| New doc | New topic | Session management (`10-sessions.md`) |
| New doc | New topic | Structured sinks and FilteringSink |

## Priority ordering

1. **High priority (blocks understanding):** "How Blocks Accumulate" in turns doc. This is the single most important gap -- without it, readers don't understand that a turn grows in place.
2. **High priority (design philosophy):** "Why Turn instead of Message" in turns doc. This sets the conceptual frame for everything else.
3. **High priority (middleware identity):** "Middleware as Composable Prompting" in middlewares doc. This reframes middleware from infrastructure plumbing to core design innovation.
4. **Medium priority (connected flow):** "Complete Runtime Flow" in engines doc. Connects all the pieces.
5. **Medium priority (new topic):** Structured sinks doc. Documents a unique feature.
6. **Lower priority (new topic):** Sessions doc. Useful but the core concepts are partially covered in turns and engines docs.
7. **Lower priority (events enhancement):** Cross-references in events doc.

## Estimated effort

Each proposed change is 1-3 paragraphs plus optional code examples. The total effort for all changes is approximately one focused documentation session (half a day). The new topic docs (sessions, structured sinks) would each take about an hour.

## Validation approach

After making these changes:
1. Have a new team member read the docs in order (turns -> blocks -> middleware -> engines -> events) and verify they can explain the runtime flow without looking at code.
2. Have a designer read the turns and middleware docs and verify they understand why the debug UI needs to show block accumulation and middleware attribution.
3. Run `docmgr doctor` to verify all cross-references are valid.
