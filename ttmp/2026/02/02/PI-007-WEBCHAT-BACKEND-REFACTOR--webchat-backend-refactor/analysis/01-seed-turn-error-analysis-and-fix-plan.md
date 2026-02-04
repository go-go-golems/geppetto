---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Profile defaults
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Seed turn built from system prompt
    - Path: pinocchio/pkg/webchat/conversation_test.go
      Note: Regression tests
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: System prompt fallback
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Seed turn error analysis and fix plan

## Root cause
`Session.StartInference` returns `ErrSessionEmptyTurn` when the latest turn has 0 blocks. The webchat session is initialized with a blank seed turn (no blocks). When `/chat` receives an empty prompt, `Session.AppendNewTurnFromUserPrompt("" )` clones the empty seed and appends nothing, leaving the latest turn empty. StartInference fails before any middleware can add a system prompt.

## Why the system prompt middleware is not enough
The system prompt middleware runs during inference, but StartInference validates the turn before running middlewares. Therefore the input turn must already contain at least one block (system or user) before StartInference begins.

## Fix strategy
1. Seed the session with a system prompt block when a conversation is created.
2. Ensure each profile supplies a system prompt so the seed is always non-empty.
3. Set the default profile prompt to exactly "You are an assistant" as requested.

## Candidate implementation
- In `webchat.getOrCreateConv`, after building `EngineConfig`, use `cfg.SystemPrompt` to build the initial seed turn:
  - Create a turn.
  - Append `turns.NewSystemTextBlock(cfg.SystemPrompt)` when non-empty.
  - Set session metadata (SessionID).
  - Use this turn as the initial element in `Session.Turns`.
- Update `cmd/web-chat/main.go` default profile prompt to `"You are an assistant"`.
- Verify other profiles have non-empty `DefaultPrompt` strings (keep existing prompt or set explicitly).

## Validation
- Run `/chat` with `prompt_len=0`; StartInference should succeed and the session should contain the system block.
- Run with a normal prompt; the first turn should contain system + user blocks.
- Ensure the middlewares still apply and no regression in normal runs.
