# Tasks

## Setup

- [x] Create ticket `GP-019-OPENAI-CACHE-USAGE`
- [x] Create design doc + diary
- [x] Confirm current usage/caching behavior in OpenAI Responses streaming and non-streaming paths

## Implementation Plan

- [x] Add shared usage parsing helper(s) for Responses usage payloads
- [x] Streaming path: parse and forward `input_tokens_details.cached_tokens`
- [x] Streaming path: keep existing reasoning-token behavior (`output_tokens_details.reasoning_tokens`)
- [x] Non-streaming path: parse usage and forward cached tokens + usage metadata to final event
- [x] Ensure metadata maps to existing `events.Usage` fields (`InputTokens`, `OutputTokens`, `CachedTokens`)
- [x] Ensure existing metadata extras remain intact (`thinking_text`, `reasoning_tokens`, etc.)

## Tests

- [x] Extend streaming test coverage to assert cached-token forwarding on final metadata
- [x] Add non-streaming test coverage for usage + cached-token forwarding
- [x] Keep existing OpenAI Responses tests green

## Validation

- [x] Run targeted tests for `pkg/steps/ai/openai_responses`
- [x] Manually verify code path into `EventFinal(metadata, ...)` contains cached tokens
- [x] Confirm no regressions to error handling behavior

## Documentation

- [x] Update ticket design doc with final implementation notes and exact code references
- [x] Append diary steps with command outputs, failures, and review notes
- [x] Update ticket changelog with code + docs commits
