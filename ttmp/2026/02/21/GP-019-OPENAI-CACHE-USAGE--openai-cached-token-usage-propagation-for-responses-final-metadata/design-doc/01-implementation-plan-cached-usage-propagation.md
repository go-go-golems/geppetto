---
Title: 'Implementation plan: cached usage propagation'
Ticket: GP-019-OPENAI-CACHE-USAGE
Status: active
Topics:
    - geppetto
    - openai
    - events
    - metadata
    - usage
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Implements streaming and non-streaming usage parsing and cached token forwarding
    - Path: geppetto/pkg/steps/ai/openai_responses/engine_test.go
      Note: Verifies cached token propagation in both streaming and non-streaming paths
ExternalSources: []
Summary: Add reliable cached-token usage propagation for OpenAI Responses in both streaming and non-streaming inference paths.
LastUpdated: 2026-02-21T18:01:44-05:00
WhatFor: Ensure llm.final metadata includes usage.caching information when provider returns it.
WhenToUse: Use when implementing or reviewing OpenAI Responses token-usage metadata handling.
---


# Implementation plan: cached usage propagation

## Executive Summary

OpenAI Responses currently forwards input/output usage in streaming mode, but does not forward cached-token usage (`input_tokens_details.cached_tokens`) and does not parse usage in the non-streaming path. This ticket implements usage parsing in both paths and forwards cached usage through `events.EventMetadata.Usage` so downstream SEM/LLM-final metadata surfaces it consistently.

## Problem Statement

The UI/SEM layer can display cached token usage only if `EventFinal` metadata includes `Usage.CachedTokens` (and related totals). In the current code:

- Streaming Responses path parses `input_tokens` and `output_tokens`, plus reasoning tokens, but drops cached tokens.
- Non-streaming Responses path does not parse usage at all before publishing `EventFinal`.

Result: `llm.final` metadata often lacks cached usage even when the provider response includes it.

## Proposed Solution

Implement a small shared usage parser for Responses usage payloads and use it in both paths:

1. Parse usage from streaming `response.completed` event payload.
2. Parse usage from non-streaming HTTP JSON response body.
3. Map parsed values to `events.EventMetadata.Usage`:
   - `input_tokens` -> `Usage.InputTokens`
   - `output_tokens` -> `Usage.OutputTokens`
   - `input_tokens_details.cached_tokens` -> `Usage.CachedTokens`
4. Preserve existing reasoning-token forwarding in `metadata.Extra["reasoning_tokens"]`.
5. Keep existing event model unchanged (`events.Usage` already contains cached fields).

## Design Decisions

- Reuse existing `events.Usage` schema rather than adding new event fields.
- Keep parser permissive:
  - accept nested usage under either `usage` or `response.usage`
  - tolerate absent details objects.
- Keep non-streaming behavior backward-compatible:
  - no change to message assembly semantics
  - only metadata enrichment.

## Alternatives Considered

- Add provider-specific cached fields into `metadata.Extra` only: rejected because downstream already expects canonical usage fields.
- Parse only streaming path: rejected because user requested both paths and non-streaming would remain inconsistent.
- Add new protobuf/SEM fields first: rejected because usage schema already supports cached tokens.

## Implementation Plan

1. Add usage parser helper(s) in `pkg/steps/ai/openai_responses/engine.go`.
2. Update streaming `response.completed` parsing to capture cached tokens.
3. Update non-streaming response handling to parse usage and set metadata usage.
4. Add/extend tests in `pkg/steps/ai/openai_responses/engine_test.go`:
   - streaming cached-token assertion
   - non-streaming cached-token assertion.
5. Run targeted tests and document outcomes in diary/changelog.

## Final Implementation Notes

Implemented in commit `970b936cec31e07c2928af9c491638ed59974991`.

- Streaming path now parses and forwards cached usage:
  - `pkg/steps/ai/openai_responses/engine.go:629` parses totals from `response.completed`.
  - `pkg/steps/ai/openai_responses/engine.go:713` writes `InputTokens`, `OutputTokens`, and `CachedTokens` to `metadata.Usage`.
- SSE loop now flushes terminal buffered payloads on EOF:
  - `pkg/steps/ai/openai_responses/engine.go:660`
  - `pkg/steps/ai/openai_responses/engine.go:675`
  - `pkg/steps/ai/openai_responses/engine.go:694`
  - `pkg/steps/ai/openai_responses/engine.go:701`
- Non-streaming path now parses usage and forwards cached totals:
  - `pkg/steps/ai/openai_responses/engine.go:792` reads raw response body.
  - `pkg/steps/ai/openai_responses/engine.go:834` parses usage envelope and sets `metadata.Usage`.
- Shared permissive parser helpers added:
  - `pkg/steps/ai/openai_responses/engine.go:865`
  - `pkg/steps/ai/openai_responses/engine.go:881`
  - `pkg/steps/ai/openai_responses/engine.go:906`

## Test Coverage Added

- Streaming final metadata cached usage assertion:
  - `pkg/steps/ai/openai_responses/engine_test.go:113`
  - `pkg/steps/ai/openai_responses/engine_test.go:231`
- New non-streaming cached usage test:
  - `pkg/steps/ai/openai_responses/engine_test.go:343`
  - `pkg/steps/ai/openai_responses/engine_test.go:428`
- Validation commands:
  - `GOCACHE=/tmp/go-build-cache go test ./pkg/steps/ai/openai_responses -count=1`
  - pre-commit hook also ran `go test ./...` and lint/vet successfully.

## Open Questions

- Should non-streaming also forward stop reason from Responses root object if present? (not required for cached-token scope)
- Should we add a dedicated parser file if more usage fields are added later?

## References

- `geppetto/pkg/steps/ai/openai_responses/engine.go`
- `geppetto/pkg/events/metadata.go`
- `geppetto/pkg/steps/ai/openai_responses/engine_test.go`
