---
Title: 'Bug report: missing thinking stream events'
Ticket: GP-05-THINK-MODE-BUG
Status: active
Topics:
    - bug
    - inference
    - events
    - geppetto
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/events/chat-events.go
      Note: Defines reasoning-text and partial-thinking event types used by the fix
    - Path: pkg/inference/engine/factory/factory.go
      Note: |-
        Auto-routes reasoning models (gpt-5/o*) from openai to openai-responses
        Implements reasoning-model auto-routing to openai-responses
    - Path: pkg/inference/engine/factory/factory_test.go
      Note: |-
        Regression test for auto-routing behavior
        Regression coverage for auto-routing
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: |-
        Added response.reasoning_text.delta/done SSE handling
        Implements reasoning_text delta/done handling
    - Path: pkg/steps/ai/openai_responses/engine_test.go
      Note: |-
        Regression test for reasoning-text event publication
        Regression coverage for reasoning_text stream events
    - Path: pkg/steps/ai/settings/flags/chat.yaml
      Note: ai-api-type default remains openai
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_after_fix_gpt5mini_openai.trace.log
      Note: |-
        Post-fix live trace showing auto-routing and partial-thinking events
        Post-fix live evidence
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_gpt5mini_openai.trace.log
      Note: Pre-fix live trace showing no thinking stream in openai mode
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/repro_thinking_stream_events.trace.log
      Note: Pre-fix deterministic trace for reasoning_text parser gap
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/repro_thinking_stream_events_after_fix.trace.log
      Note: |-
        Post-fix deterministic trace showing reasoning-text events emitted
        Post-fix deterministic evidence
ExternalSources: []
Summary: 'Implemented and validated fixes for missing thinking stream behavior: auto-routing gpt-5/o* to Responses and handling response.reasoning_text.* events.'
LastUpdated: 2026-02-18T16:19:00-05:00
WhatFor: Diagnose, fix, and validate missing thinking stream events when using gpt-5-mini.
WhenToUse: Use when investigating or validating OpenAI thinking-stream behavior in geppetto.
---


# Bug Report: Missing Thinking Stream Events (`gpt-5-mini`)

## Final outcome

The issue is fixed in code and validated with both unit tests and real API runs.

Implemented fixes:

1. **Auto-route reasoning models to Responses**  
   When provider resolves to `openai` and model is `gpt-5*`/`o1*`/`o3*`/`o4*`, engine creation now routes to `openai-responses`.

2. **Handle `response.reasoning_text.*` SSE events**  
   Responses streaming now handles:
   - `response.reasoning_text.delta`
   - `response.reasoning_text.done`

   and emits:
   - `EventReasoningTextDelta` / `EventReasoningTextDone`
   - mirrored `EventThinkingPartial` for existing UI compatibility.

## Root causes (pre-fix)

1. `--ai-engine gpt-5-mini` with default `ai-api-type=openai` used Chat Completions path, which did not emit Geppetto thinking-stream events.
2. Responses parser did not handle `response.reasoning_text.*`, causing dropped thinking text for that stream shape.
3. Thinking text visibility in Responses also depends on reasoning summary output configuration.

## Validation evidence

### Unit tests

```bash
go test ./pkg/inference/engine/factory -count=1
go test ./pkg/steps/ai/openai_responses -count=1
go test ./... -count=1
```

Both passed.

### Live API verification

- Pre-fix (`ai-api-type=openai`): no thinking stream events  
  `sources/real_api_gpt5mini_openai.trace.log`
- Post-fix (`ai-api-type=openai`): auto-routed to Responses, emits `partial-thinking`  
  `sources/real_api_after_fix_gpt5mini_openai.trace.log`

Key post-fix trace lines include:
- `Auto-routing reasoning model to OpenAI Responses engine ...`
- `Responses: sending request ... /v1/responses`
- multiple `event_type=partial-thinking`

### Deterministic parser verification

- Pre-fix mocked `response.reasoning_text.*` stream dropped reasoning-text events  
  `sources/repro_thinking_stream_events.trace.log`
- Post-fix same mocked stream emits:
  - `type:reasoning-text-delta`
  - `type:reasoning-text-done`
  - `type:partial-thinking`
  
  `sources/repro_thinking_stream_events_after_fix.trace.log`

## Code changes

- `pkg/inference/engine/factory/factory.go`
  - added `shouldAutoRouteToResponses(...)`
  - auto-routes reasoning models from `openai` to `openai-responses`
- `pkg/inference/engine/factory/factory_test.go`
  - added regression test: `TestStandardEngineFactory_CreateEngine_AutoRoutesReasoningModelsToResponses`
- `pkg/steps/ai/openai_responses/engine.go`
  - added SSE switch cases for `response.reasoning_text.delta` and `.done`
  - emits reasoning-text events plus mirrored `partial-thinking`
- `pkg/steps/ai/openai_responses/engine_test.go`
  - added regression test: `TestRunInference_StreamingReasoningTextEventsArePublished`

## Remaining considerations

1. `ai-api-type=openai-responses` with `openai-reasoning-summary ""` can still produce boundary-only thinking (no summary deltas). This is expected config-dependent behavior.
2. If product wants guaranteed visible thinking text, enforce/override summary settings or render reasoning-text events directly in all clients.

## Confidence

- High confidence: fixes and regressions are covered by tests plus real API verification.
