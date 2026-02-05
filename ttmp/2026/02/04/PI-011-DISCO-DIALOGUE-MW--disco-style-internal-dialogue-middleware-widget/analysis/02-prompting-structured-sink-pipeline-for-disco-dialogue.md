---
Title: Prompting + Structured Sink Pipeline for Disco Dialogue
Ticket: PI-011-DISCO-DIALOGUE-MW
Status: active
Topics:
    - backend
    - frontend
    - middleware
    - sem
    - protobuf
    - webchat
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/tutorials/04-structured-data-extraction.md
      Note: Tutorial overview of structured data extraction
    - Path: geppetto/pkg/events/structuredsink/filtering_sink.go
      Note: FilteringSink mechanics (tag detection, session lifecycle, filtering)
    - Path: geppetto/pkg/events/structuredsink/parsehelpers/helpers.go
      Note: Debounced YAML parsing via YAMLController
    - Path: go-go-mento/go/pkg/inference/middleware/thinkingmode/extractor.go
      Note: Extractor session parsing YAML into events
    - Path: go-go-mento/go/pkg/inference/middleware/thinkingmode/middleware.go
      Note: Prompt injection example with tagged YAML
    - Path: go-go-mento/go/pkg/webchat/sink_wrapper.go
      Note: FilteringSink wiring by profile
    - Path: moments/docs/backend/creating-llm-middleware-with-structured-data-extraction.md
      Note: End-to-end pattern for prompt injection + structured sink
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-05T03:22:00-05:00
WhatFor: Explain how prompt injection + structured sinks produce streaming SEM events for the Disco internal dialogue feature.
WhenToUse: When implementing or reviewing the Disco dialogue middleware, extractors, SEM events, and UI widget pipeline.
---


# Prompting + Structured Sink Pipeline for Disco Dialogue

## Goal

Clarify the exact system behavior we are building: a middleware injects **prompt instructions** that make the LLM emit **structured YAML inside tagged blocks**; the **FilteringSink** detects those tags in the streaming output, **parses YAML into typed payloads**, emits **structured events**, and the webchat front‑end renders those events in a custom Disco-style widget. The user-visible text should **not** include the structured block contents.

This document is deliberately concrete and references actual files in `moments/`, `go-go-mento/`, and `geppetto/` so the pipeline can be implemented confidently and debugged easily by a new engineer.

---

## Sources Reviewed (Deep Scan)

### Core implementation (streaming structured extraction)

- `geppetto/pkg/events/structuredsink/filtering_sink.go`
  - The actual FilteringSink implementation that scans partial/final text events, detects tags, and calls extractors.
- `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go`
  - `YAMLController` and `StripCodeFenceBytes` used by extractors to parse YAML progressively.

### Tutorials and playbooks (conceptual + practical guidance)

- `geppetto/pkg/doc/tutorials/04-structured-data-extraction.md`
- `geppetto/pkg/doc/playbooks/03-progressive-structured-data.md`
- `moments/docs/backend/creating-llm-middleware-with-structured-data-extraction.md`
  - This is the most direct guide to the “prompt injection + structured sink” pattern.

### Concrete middleware examples in production code

- `go-go-mento/go/pkg/inference/middleware/thinkingmode/middleware.go`
- `go-go-mento/go/pkg/inference/middleware/thinkingmode/extractor.go`
- `go-go-mento/go/pkg/inference/middleware/debate/middleware.go`
- `go-go-mento/go/pkg/inference/middleware/debate/extractor.go`
- `go-go-mento/go/pkg/webchat/sink_wrapper.go`
  - Shows how the FilteringSink is attached per profile.

### Integration notes for webchat

- `moments/ttmp/IMPORT-MIDDLEWARE-port-middleware-and-structured-event-emitters-from-mento-playground-go-pkg-webchat-into-webchat/design/01-migration-plan-middleware-and-structured-event-emitters.md`
  - Provides the “wrapSinkWithExtractors” ordering used by webchat.

---

## System Overview (Narrative)

A user message enters the webchat router. The router builds a `Turn` and constructs an inference engine with middleware. Our Disco middleware adds a **system block** instructing the model to emit **tagged YAML** for internal dialogue lines, checks, and state. The model streams output; those tagged sections are *not for the user*. The FilteringSink recognizes tags like `<disco:dialogue_line:v1> ... </disco:dialogue_line:v1>`, forwards the payload bytes to a type-specific **ExtractorSession**, and emits **structured SEM events** while stripping the tags from visible text. The UI listens to those events and renders a custom dialogue widget as a parallel stream.

The key is that **prompting and extraction are two halves of the same design**. Without the prompt, the LLM never emits valid tags. Without the sink+extractors, the tags never become structured events. These two pieces must be coordinated (exact tag names, payload schemas, and when they appear).

---

## High-Level Pipeline Diagram

```
User prompt
   ↓
Pinocchio Webchat Router (/chat)
   ↓ builds Turn + middleware chain
DiscoDialogueMiddleware (Before hook)
   ↓ injects system instructions (tagged YAML format)
Inference Engine runs (streaming tokens)
   ↓ Event stream (partial + final text)
FilteringSink (structuredsink)
   ├─ strips <disco:...> blocks from visible text
   └─ routes payload bytes to Extractors
         ↓
         ExtractorSessions parse YAML incrementally
         ↓
         SEM events (disco.dialogue.*)
         ↓
Timeline projection + UI widget
```

---

## What “Prompt Injection” Means Here

Middleware injects a **system block** that explains the data contract to the LLM. You can see how this is done in `go-go-mento/go/pkg/inference/middleware/thinkingmode/middleware.go`:

- It builds a large instruction string with explicit tag formats.
- It prepends that block to `Turn.Blocks`.
- It includes examples of `<thinking:mode:v1> ... </thinking:mode:v1>` inside triple-backtick YAML fences.

This is critical: the FilteringSink looks for **exact tag triples** `(package, type, version)`. That’s why the prompt uses the same triple. If the prompt says `<disco:dialogue_line:v1>` but the extractor is registered for `disco:dialogue-line:v1`, you will not get structured events.

### Practical pattern (from thinking mode)

1. Middleware builds instructions (string).
2. Add as system block at the start of the Turn.
3. The LLM writes structured blocks while also continuing normal response text.

In our case, the system block will include multiple **mini-contracts**:

- `dialogue_line`: “a line of inner speech from a persona”
- `dialogue_check`: “a passive/active check result with roll + success”
- `dialogue_state`: “start/completed signals or summary of the internal exchange”

We should embed **short YAML examples** for each. The thinking mode middleware shows exactly how to do that, including a note that those blocks are removed from user-visible output.

---

## What the FilteringSink Actually Does

The FilteringSink (`geppetto/pkg/events/structuredsink/filtering_sink.go`) sits inside the event stream. It watches for **partial completion** and **final completion** events, scans the text as it streams, and detects tags of the form:

```
<package:type:version>
```yaml
...
```
</package:type:version>
```

Internally it:

- Tracks **per-stream state** (it is keyed by `meta.ID` and manages an internal buffer).
- On open tag detection, it creates an `ExtractorSession` from the registered extractor.
- While inside the tag, it streams raw bytes to `OnRaw`.
- When it finds the closing tag, it calls `OnCompleted` with the raw payload.
- It forwards filtered text downstream (the tags and payload are stripped from the stream).

This is why the user-visible text is clean, while we still get structured SEM events.

### Parsing YAML in the extractor

The YAML parsing is intentionally done **by the extractor**, not by the sink. Most extractors use `parsehelpers.YAMLController[T]`, which supports **debounced incremental parsing**:

- `FeedBytes` returns partial parsed snapshots on newline boundaries or after N bytes.
- `FinalBytes` attempts a final parse once the full block is available.

This is what makes it possible to stream progressive updates to the UI.

---

## Concrete Example: Thinking Mode in go-go-mento

The thinking mode middleware and extractor are a direct pattern match for what we need.

### Prompt Injection (middleware)

In `go-go-mento/go/pkg/inference/middleware/thinkingmode/middleware.go` the prompt includes:

```
<thinking:mode:v1>
```yaml
mode: <name>
phase: <current_phase>
reasoning: <why>
```
</thinking:mode:v1>
```

This is exactly the type of contract we will mirror for Disco dialogue lines.

### Structured Sink + Extractor

In `go-go-mento/go/pkg/inference/middleware/thinkingmode/extractor.go`, each tag has its own extractor (e.g., `thinking:mode:v1`, `thinking:mode_evaluation:v1`, `thinking:inner_thoughts:v1`). Each extractor session:

- Emits `started`, `delta`, and `update` events while streaming.
- Calls YAML parsing in `OnRaw` for best-effort incremental snapshots.
- Emits a `completed` event at the end, with success/failure flags.

### Sink wiring

In `go-go-mento/go/pkg/webchat/sink_wrapper.go`, the FilteringSink is attached **per profile**. That file shows a robust pattern for adding multiple extractors and wrapping the base sink.

This should be emulated for the Disco profile so that our structured events are emitted during streaming.

---

## Proposed Disco Dialogue Tag Set

We should keep the tag count small, but expressive. I recommend 2–3 tags:

1. **Line output**

```
<disco:dialogue_line:v1>
```yaml
line_id: "..."
persona: "Empathy"
text: "You can feel it—the sadness is not yours."
tone: "tender"
trigger: "passive"  # passive|active|antipassive|thought
progress: 0.4
```
</disco:dialogue_line:v1>
```

2. **Checks / Rolls**

```
<disco:dialogue_check:v1>
```yaml
check_type: "passive"   # passive|active|antipassive
skill: "Empathy"
difficulty: 12
roll: 10
success: false
```
</disco:dialogue_check:v1>
```

3. **Dialogue summary or lifecycle event** (optional)

```
<disco:dialogue_state:v1>
```yaml
dialogue_id: "..."
status: "completed"     # started|updated|completed
summary: "Empathy cautions you, Logic dismisses it, Volition decides." 
```
</disco:dialogue_state:v1>
```

### Why multiple tags?

- `dialogue_line` can be parsed and rendered as it streams.
- `dialogue_check` can be rendered as a special “roll” card or annotation.
- `dialogue_state` lets the UI know when to collapse or highlight the final state.

This is directly aligned with the structured sink architecture: each tag has a single extractor, and the event stream is cleanly separated.

---

## Widget + SEM Event Mapping (How the UI gets data)

Once the extractors emit structured events, they are translated into SEM events and propagated through the existing webchat event channel. The widget does **not** parse raw text. It listens to structured SEM events (or the timeline hydration results) and renders the dialogue.

Recommended SEM event types (already defined in the plan):

- `disco.dialogue.started`
- `disco.dialogue.line`
- `disco.dialogue.updated`
- `disco.dialogue.completed`
- `disco.dialogue.passive`
- `disco.dialogue.antipassive`
- `disco.dialogue.active_check`
- `disco.dialogue.thought`

Each of these should carry a `DiscoDialogueEventV1` payload (protojson). The event payload can embed the parsed YAML data (line/check/state), or the YAML can be normalized into the proto schema in the extractor.

**Implementation note:** it is normal for the YAML payload and the protobuf schema to be *aligned but not identical*. The extractor can map or coerce fields (e.g., `check_type` → enum). If we want a 1:1 mapping, then the YAML should mirror the proto shape exactly.

---

## Pseudocode (End-to-End Contract)

### Middleware (prompt injection)

```pseudo
function DiscoDialogueMiddleware.Before(turn):
  instruction = buildDiscoDialoguePrompt(personas, thresholds, max_lines)
  system_block = new SystemTextBlock(instruction)
  prepend(turn.Blocks, system_block)
  return turn
```

### FilteringSink (already implemented)

```pseudo
on EventTypePartialCompletion(text):
  scan for <disco:dialogue_line:v1>...
  if open tag detected:
     session = extractor.NewSession(meta, item_id)
     session.OnStart()
  if inside tag:
     session.OnRaw(payload_delta)
  if close tag detected:
     session.OnCompleted(full_payload, success)
  forward filtered text downstream
```

### Extractor Session (YAML parsing)

```pseudo
on OnRaw(payload_chunk):
  snapshot = yaml_controller.FeedBytes(payload_chunk)
  if snapshot != nil:
     emit EventDiscoLineUpdated(snapshot)
  emit EventDiscoLineDelta(payload_chunk)

on OnCompleted(payload):
  final = yaml_controller.FinalBytes(payload)
  emit EventDiscoLineCompleted(final)
```

### UI Widget

```pseudo
on SEMEvent(discrete):
  if event.type == disco.dialogue.line:
     update internal store
  render cards grouped by persona
  show roll icons for check events
```

---

## Implementation Notes and Caveats

1. **Tag spelling must match extractor triple exactly**.
   - Example: `disco:dialogue_line:v1` must match TagPackage = `disco`, TagType = `dialogue_line`, TagVersion = `v1`.

2. **YAML blocks should be wrapped in triple backticks**
   - FilteringSink doesn’t require the fence, but `parsehelpers.StripCodeFenceBytes` expects them and avoids parsing the raw fence.

3. **Keep structured payloads small**
   - Extractors often cap `MaxBytes` (e.g., 64KB). Dialogue lines should be short.

4. **Streaming cadence**
   - If the model only emits one huge YAML block at the end, the UI won’t update until completion. Encourage incremental emission in the prompt.

5. **Malformed output behavior**
   - FilteringSink handles malformed blocks with a policy (`MalformedErrorEvents`, `MalformedReconstructText`, etc.). The default is to emit errors and avoid reinserting broken text. This is fine for dialogue, but debugging output should log errors if the model outputs invalid YAML.

6. **Avoid leaking tags to users**
   - The FilteringSink removes tags from *streaming* outputs, but ensure no other UI path (like non-streamed `Final` events without filtering) reintroduces them.

---

## How This Updates the Implementation Plan

- The middleware **must** inject a structured output prompt that uses explicit tag names and YAML examples (not optional).
- The **structured sink** is the system that connects that prompt to the UI. Implementing extractors without prompt injection (or vice versa) will fail.
- The Disco feature requires at least **two extractors** (line and check), likely three if we want dialogue summary/lifecycle.
- The UI widget should render exclusively from SEM events / timeline projection. Parsing raw text is explicitly out-of-scope.

---

## Suggested Next Steps

1. Define the exact YAML schema for `disco:dialogue_line:v1` and `disco:dialogue_check:v1` (if we need those fields).
2. Draft the system prompt that embeds those schemas.
3. Build extractor sessions for each tag.
4. Wire the extractors into the webchat sink wrapper.
5. Map extractor payloads to SEM protobuf events.
6. Render the widget and verify streaming updates + timeline hydration.
