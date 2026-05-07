---
Title: Anthropic Claude Observability Analysis and Implementation Guide
Ticket: GEPPETTO-ANTHROPIC-OBS-2026-05-07
Status: active
Topics:
  - observability
  - llm
  - inference
  - intern-onboarding
DocType: design-doc
Intent: long-term
Owners:
  - manuel
RelatedFiles:
  - Path: pkg/steps/ai/claude/engine_claude.go
  - Path: pkg/steps/ai/claude/content-block-merger.go
  - Path: pkg/steps/ai/claude/api/messages.go
  - Path: pkg/steps/ai/openai/observability.go
  - Path: pkg/steps/ai/openai_responses/observability.go
  - Path: pkg/observability/observer.go
  - Path: pkg/observability/config.go
  - Path: pkg/inference/engine/factory/factory.go
ExternalSources: []
Summary: Intern-oriented design guide for instrumenting the Claude/Anthropic path with Geppetto observability records.
LastUpdated: 2026-05-07T15:25:00-04:00
WhatFor: "Use before implementing or reviewing Claude/Anthropic observability instrumentation."
WhenToUse: "When editing pkg/steps/ai/claude or factory option plumbing for observer/config injection."
---

# Anthropic Claude Observability Analysis and Implementation Guide

## Executive Summary

This guide describes how to add Geppetto observability support to the Anthropic Claude streaming path. The OpenAI Chat Completions and OpenAI Responses paths now share the same high-level policy:

- provider-level records are emitted at trace level `provider`;
- Geppetto publish-boundary records are compact `StageGeppettoPublishStarted` records only;
- publish records do not include full `EventJSON` or `MetadataJSON` payloads;
- observer delivery is best-effort and panic-safe through `observability.Notify`.

Claude should follow that policy. The implementation is straightforward because `ClaudeEngine.RunInference` already receives typed provider stream events from `api.Client.StreamMessage` before passing them to `ContentBlockMerger.Add`. The main work is to add the same engine option surface as OpenAI, record each typed Claude stream event, record compact publish-started events, and wire `StandardEngineFactory` so callers can attach observers to Claude engines.

## Problem Statement

`pkg/steps/ai/claude/engine_claude.go` currently emits product-facing Geppetto events through `events.PublishEventToContext`, but it does not emit neutral observability records. This makes Claude harder to debug than OpenAI providers when the web-chat debug recorder is enabled.

A user looking at a failed or surprising Claude response can currently inspect:

- final Geppetto events seen by event sinks;
- logs from the Claude client and merger;
- final `Turn` blocks and persisted inference metadata.

But they cannot easily answer:

- Which Anthropic streaming event arrived before a Geppetto partial/final/tool event?
- What was the Claude message ID, content block index, or delta type on that provider event?
- Did the engine publish a compact Geppetto event boundary record for the normalized event?
- Did observer delivery remain safe when an observer panicked?

## Current Claude Runtime Flow

```mermaid
flowchart TD
    Caller[Caller] --> Engine[ClaudeEngine.RunInference]
    Engine --> Request[MakeMessageRequestFromTurn]
    Request --> Client[api.NewClient + StreamMessage]
    Client --> EventCh[<-chan api.StreamingEvent]
    EventCh --> Loop[Claude streaming loop]
    Loop --> Merger[ContentBlockMerger.Add]
    Merger --> ProductEvents[[]events.Event]
    ProductEvents --> Publish[publishEvent]
    Publish --> Sinks[events.EventSink]
    Merger --> Response[MessageResponse]
    Response --> Turn[Append assistant/tool blocks]
    Turn --> Result[Persist inference_result]
```

Target observability flow:

```mermaid
flowchart TD
    EventCh[api.StreamingEvent] --> ProviderObs[observeProviderEvent]
    EventCh --> Merger[ContentBlockMerger.Add]
    Merger --> ProductEvents[[]events.Event]
    ProductEvents --> PublishStarted[observePublishStarted]
    PublishStarted --> Observer[geppettoobs.Observer]
    ProviderObs --> Observer
    ProductEvents --> Sinks[events.EventSink]
```

## Key Files

### `pkg/steps/ai/claude/engine_claude.go`

Owns `ClaudeEngine`, request construction, streaming event consumption, publishing, and final turn mutation. This is the primary implementation target.

Current shape:

```go
type ClaudeEngine struct {
    settings    *settings.InferenceSettings
    toolAdapter *tools.ClaudeToolAdapter
}

func NewClaudeEngine(settings *settings.InferenceSettings) (*ClaudeEngine, error)

func (e *ClaudeEngine) publishEvent(ctx context.Context, event events.Event) {
    events.PublishEventToContext(ctx, event)
}
```

Target shape:

```go
type ClaudeEngine struct {
    settings            *settings.InferenceSettings
    toolAdapter         *tools.ClaudeToolAdapter
    observer            geppettoobs.Observer
    observabilityConfig geppettoobs.Config
}

func NewClaudeEngine(settings *settings.InferenceSettings, opts ...EngineOption) (*ClaudeEngine, error)

func (e *ClaudeEngine) publishEvent(ctx context.Context, event events.Event) {
    e.observePublishStarted(ctx, event)
    events.PublishEventToContext(ctx, event)
}
```

### `pkg/steps/ai/claude/api/messages.go`

Owns the HTTP/SSE request and decoder. `StreamMessage` returns `<-chan StreamingEvent>`. For this phase, do not modify the decoder; the engine can observe decoded typed events.

### `pkg/steps/ai/claude/content-block-merger.go`

Turns Claude provider events into Geppetto product events:

- `message_start` -> `start`
- `content_block_delta` with text -> `partial`
- `content_block_stop` with tool_use -> `tool-call`
- `message_stop` -> `final`
- `error` -> `error`

Observability should not change merger semantics.

### `pkg/observability`

Provides the shared record shape and trace-level policy:

- `TraceOff`: no records.
- `TraceEvents`: compact publish-started records only.
- `TraceProvider`: provider records plus compact publish-started records.

## Proposed API

Add `pkg/steps/ai/claude/observability.go`:

```go
type EngineOption func(*ClaudeEngine)

func WithObserver(obs geppettoobs.Observer) EngineOption
func WithObservabilityConfig(cfg geppettoobs.Config) EngineOption
```

Add helpers:

```go
func (e *ClaudeEngine) observe(ctx context.Context, rec geppettoobs.Record)
func (e *ClaudeEngine) observePublishStarted(ctx context.Context, event events.Event)
func (e *ClaudeEngine) observeProviderEvent(ctx context.Context, metadata events.EventMetadata, model string, ev api.StreamingEvent)
```

## Provider Record Mapping

Claude `api.StreamingEvent` should map to `geppettoobs.Record` as follows:

| Claude field | Record field |
|---|---|
| `event.Type` | `EventType` |
| `event.Message.ID` | `ResponseID` |
| `event.Message.Model` or request model | `Model` |
| `event.Index` | `OutputIndex` |
| serialized event | `ObjectJSON` |
| text delta length | `DeltaLen` |
| error message | `Error` for error events |

Claude does not naturally use OpenAI-style `item_id` / `summary_index`. Keep those empty unless future Claude content block IDs need to be modeled.

## Publish Record Policy

Emit only compact `StageGeppettoPublishStarted` records:

```go
func (e *ClaudeEngine) publishEvent(ctx context.Context, event events.Event) {
    e.observePublishStarted(ctx, event)
    events.PublishEventToContext(ctx, event)
}
```

Do not emit `StageGeppettoPublishDone`. Do not attach full event JSON or metadata JSON. This matches the final OpenAI policy and prevents trace-size ballooning on streamed partials.

## Implementation Plan

1. Add `pkg/steps/ai/claude/observability.go`.
2. Add observer/config fields to `ClaudeEngine`.
3. Make `NewClaudeEngine` accept variadic options while preserving existing callers.
4. Emit compact publish-started records in `publishEvent`.
5. In the streaming loop, call `e.observeProviderEvent(ctx, metadata, req.Model, event)` immediately after receiving each provider event and before `completionMerger.Add(event)`.
6. Add `claudeOptions []claude.EngineOption` and `WithClaudeOptions(...)` to `StandardEngineFactory`.
7. Pass `f.claudeOptions...` into `claude.NewClaudeEngine` for both `claude` and `anthropic` providers.
8. Add tests for:
   - trace off emits no records;
   - `TraceEvents` emits publish-started records and no provider records;
   - `TraceProvider` emits provider records and publish-started records;
   - observer panic is ignored;
   - factory options reach Claude engines.
9. Run targeted tests, then broad validation.
10. Update diary/changelog and commit source/docs at appropriate checkpoints.

## Test Strategy

Use existing Claude fake HTTP transport patterns from `pkg/steps/ai/claude/helpers_test.go`. A minimal SSE body should include:

```text
event: message_start
data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-test","stop_reason":"","stop_sequence":"","usage":{"input_tokens":1,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"pong"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_stop
data: {"type":"message_stop"}
```

Expected product result: assistant text `pong` appended to the turn.

Expected observability at `TraceProvider`:

- provider records for the above Claude event types;
- compact publish-started records for `start`, `partial`, and `final`;
- no publish-done records.

## Safety Checklist

- Do not record request bodies or API keys.
- Do not change Claude merger behavior.
- Do not emit full product event JSON on publish records.
- Use `observability.Notify` so observer panics do not fail inference.
- Keep constructor changes variadic for source compatibility.
- Keep non-Claude factory paths unchanged.

## Open Questions

- Should Claude content block indexes be stored in `OutputIndex` permanently, or should Geppetto add a provider-neutral `ContentIndex` field later?
- Should ping events be recorded at `TraceProvider`? Initial recommendation: yes, because they are cheap and help explain stream gaps; they can be filtered later if too noisy.
- Should `EventJSON` / `MetadataJSON` fields remain on the shared record type? Yes, for backward compatibility and possible non-stream future use, even though OpenAI/Claude publish records no longer populate them.

## References

- OpenAI Chat Completions implementation: `pkg/steps/ai/openai/observability.go`
- OpenAI Responses implementation: `pkg/steps/ai/openai_responses/observability.go`
- Claude engine: `pkg/steps/ai/claude/engine_claude.go`
- Claude merger: `pkg/steps/ai/claude/content-block-merger.go`
- Shared observability: `pkg/observability/observer.go`, `pkg/observability/config.go`
