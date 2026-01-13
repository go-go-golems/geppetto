---
Title: Moments Follow-up Plan
Ticket: MO-002-FIX-UP-THINKING-MODELS
Status: active
Topics:
    - bug
    - geppetto
    - go
    - inference
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../moments/backend/pkg/inference/middleware/conversation_compression_middleware.go
      Note: Compression mutation to convert into ConversationState step.
    - Path: ../../../../../../../moments/backend/pkg/webchat/loops.go
      Note: Tool loop turn updates to migrate to ConversationState.
    - Path: ../../../../../../../moments/backend/pkg/webchat/ordering_middleware.go
      Note: Ordering rules that must map to section-based mutations.
    - Path: ../../../../../../../moments/backend/pkg/webchat/router.go
      Note: Moments webchat entry point and Turn mutation.
    - Path: ../../../../../../../moments/backend/pkg/webchat/system_prompt_middleware.go
      Note: Idempotent system prompt insertion to preserve.
ExternalSources: []
Summary: Deferred Moments webchat migration plan for unified conversation handling.
LastUpdated: 2026-01-13T17:41:18.518955638-05:00
WhatFor: Track deferred Moments webchat changes needed for unified conversation handling.
WhenToUse: Use when planning the Moments migration or validating conversation ordering.
---






# Moments Follow-up Plan

## Goal

Record what must change in Moments once we start the shared conversation-state migration, and keep this plan updated as geppetto/pinocchio work lands.

## Context

We are not touching Moments yet. The immediate work is limited to geppetto and pinocchio. This document captures the deferred Moments steps so the eventual migration stays consistent with the unified conversation design.

## Current Observations (from prior analysis)

- Moments webchat mutates a single `conv.Turn` per request, but ordering and compression middlewares can reorder and drop blocks.
- System prompts are injected via idempotent middleware, but the exact ordering contract differs from pinocchio.
- Tool loop updates the conversation state in `moments/backend/pkg/webchat/loops.go`, not via a shared conversation API.

## Proposed Future Work (Moments)

### Phase A: Prep and parity checks

- Audit Moments `Turn` metadata keys to ensure they can support section-based ordering.
- Confirm tool call + tool result pairing logic survives ordering/compression mutations.
- Add lightweight logging around ordering/compression to capture block order before and after.

### Phase B: ConversationState integration

- Replace `conv.Turn` with the shared `ConversationState` (geppetto package).
- Convert ordering middleware into a deterministic mutation step, driven by section metadata.
- Convert compression middleware into a mutation (or snapshot config) so it runs in a single place.
- Preserve idempotent system/global prompts with metadata keys and stable section placement.

### Phase C: Validation and tests

- Add multi-turn Responses tests with reasoning blocks to verify adjacency rules.
- Add regression tests for tool call pairing and ordering stability.
- Validate webchat rendering and SSE event ordering under Responses mode.

## Risks and gotchas

- Ordering middleware that runs after compression may reorder blocks into invalid reasoning adjacency.
- Tool call metadata could be lost during compression if metadata is not preserved.
- Snapshot mode differences (Responses vs legacy chat) need explicit flags to avoid surprises.

## Open questions

- Do Moments tool loops need a snapshot hook to persist intermediate turns?
- Should Moments adopt the same ConversationManager lifecycle as pinocchio webchat?

## Update checklist

- If geppetto/pinocchio code changes alter the conversation API, update this doc.
- If a new ordering contract is finalized, update the "Phase B" tasks accordingly.
