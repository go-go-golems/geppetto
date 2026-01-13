---
Title: Diary
Ticket: MO-002-FIX-UP-THINKING-MODELS
Status: active
Topics:
    - bug
    - geppetto
    - go
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/reasoning_validator.go
      Note: Upstream reasoning model constraints (max tokens + sampling).
    - Path: geppetto/cmd/examples/openai-tools/main.go
      Note: Repro steps for GPT-5 runs.
    - Path: geppetto/pkg/steps/ai/openai/helpers.go
      Note: Chat-mode request parameter gating for reasoning models.
    - Path: geppetto/pkg/steps/ai/openai_responses/helpers.go
      Note: Responses request parameter gating (sampling params).
ExternalSources: []
Summary: Track fixes for thinking-model parameter handling in chat vs responses engines.
LastUpdated: 2026-01-13T00:00:00Z
WhatFor: Capture investigation and code changes for GPT-5/o-series parameter gating.
WhenToUse: Use when validating reasoning model support and engine request building.
---


# Diary

## Goal

Document the investigation and fixes for GPT-5/o-series (thinking) model parameter handling across OpenAI chat and Responses engines.

## Step 1: Identify thinking-model parameter failures and start gating

I created the ticket workspace and traced the failures reported when running GPT-5. Chat-mode failed fast because go-openai rejects `max_tokens` for GPT-5, while Responses-mode rejected `temperature` for GPT-5. The plan is to explicitly detect reasoning-capable models (o1/o3/o4/gpt-5) and gate parameters accordingly: use `max_completion_tokens` for chat mode and omit sampling params for Responses.

I began implementing the first part by adding a reasoning-model detector in the OpenAI chat helper, using it to move `max_tokens` into `max_completion_tokens` and to reset sampling params to supported values. This keeps the request aligned with go-openai's validator and GPT-5 constraints while preserving standard behavior for non-reasoning models.

**Commit (code):** N/A (not committed)

### What I did
- Created ticket MO-002-FIX-UP-THINKING-MODELS and added a diary doc.
- Captured reported errors:
  - `this model is not supported MaxTokens, please use MaxCompletionTokens`
  - `Unsupported parameter: 'temperature' is not supported with this model.`
- Added a reasoning-model detector in `openai/helpers.go` and started gating max tokens + sampling parameters for chat-mode requests.

### Why
- GPT-5 and o-series models have stricter parameter requirements; chat-mode must use `max_completion_tokens` and omit unsupported sampling knobs.
- Responses-mode should omit sampling params for these models as well.

### What worked
- Identified that go-openai's reasoning validator flags GPT-5 when `max_tokens` is set.

### What didn't work
- `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5` failed with:
  - `this model is not supported MaxTokens, please use MaxCompletionTokens`
- `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5 --ai-api-type=openai-responses` failed with:
  - `responses api error: status=400 body=map[error:map[code:<nil> message:Unsupported parameter: 'temperature' is not supported with this model. param:temperature type:invalid_request_error]]`

### What I learned
- GPT-5 is treated as a reasoning model by go-openai and must follow the same parameter restrictions as o-series.

### What was tricky to build
- Ensuring we only change request parameters for reasoning-capable models without breaking defaults for standard chat models.

### What warrants a second pair of eyes
- Confirm the gating logic matches OpenAI's current parameter constraints for GPT-5/o-series across both chat and responses.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai/helpers.go` and review `isReasoningModel` and request parameter changes.
- Validate with `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5 --ai-api-type=openai-responses`.

### Technical details
- go-openai validator: `/home/manuel/go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/reasoning_validator.go`
- Command that failed:
  - `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5`

## Step 2: Gate sampling params in Responses for GPT-5/o-series

I extended the Responses request builder to treat GPT-5 and o1/o3/o4 as reasoning models when deciding whether to include sampling parameters. This aligns the Responses path with the same model constraints that the chat engine now enforces, and directly addresses the 400 error about unsupported `temperature`.

This is a focused change to `openai_responses/helpers.go`, leaving the rest of the request shape intact. The aim is to keep tool and reasoning behavior unchanged while dropping parameters that the API rejects for these model families.

**Commit (code):** N/A (not committed)

### What I did
- Updated `allowSampling` to exclude `o1`, `o3`, `o4`, and `gpt-5` in the Responses helper.

### Why
- GPT-5 rejects `temperature` (and related sampling params) in the Responses API.

### What worked
- The request builder now omits sampling params for GPT-5/o-series models.

### What didn't work
- N/A (no new failures recorded yet).

### What I learned
- Responses and chat paths both need explicit reasoning-model gating; assumptions based on o3/o4 alone are insufficient.

### What was tricky to build
- Keeping model-family matching consistent between chat and Responses helpers without changing behavior for standard models.

### What warrants a second pair of eyes
- Confirm the model prefix list is complete and that omitting sampling params is correct for all GPT-5 variants.

### What should be done in the future
- N/A

### Code review instructions
- Review `allowSampling` in `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/helpers.go`.
- Validate with `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5 --ai-api-type=openai-responses`.

### Technical details
- Responses helper: `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/helpers.go`

## Step 3: Validate GPT-5 Responses run via tmux

I ran the Responses-mode CLI example in tmux using GPT-5 and the server-tools mode to validate the parameter gating changes. The run completed successfully and produced tool calls, search results, and the final response, which suggests the previous `temperature` rejection is resolved. It did take about a minute to finish, so the prior “hang” likely reflected long tool activity rather than a deadlock.

This step focused on runtime behavior rather than code changes, confirming that the streaming loop can complete with GPT-5 and server tools enabled.

**Commit (code):** N/A (no code changes)

### What I did
- Ran `go run ./cmd/examples/openai-tools test-openai-tools --ai-api-type=openai-responses --ai-engine gpt-5 --mode server-tools --log-level info` inside tmux.
- Observed multiple reasoning summary phases, tool calls, and final output.

### Why
- Validate that GPT-5 Responses requests no longer fail on unsupported `temperature` and that the flow completes.

### What worked
- The run completed with a final response and tool results; total runtime ~1m17s.

### What didn't work
- N/A

### What I learned
- GPT-5 Responses with server tools can be slow but completes; the “hang” likely reflects long tool activity.

### What was tricky to build
- Distinguishing a slow Responses run from a genuine hang without explicit timeouts.

### What warrants a second pair of eyes
- Confirm if we should add a user-facing timeout or progress indicator for long server-tool runs.

### What should be done in the future
- N/A

### Code review instructions
- N/A (runtime validation only).

### Technical details
- tmux session: `gpt5-resp` (captured output shows completion).
