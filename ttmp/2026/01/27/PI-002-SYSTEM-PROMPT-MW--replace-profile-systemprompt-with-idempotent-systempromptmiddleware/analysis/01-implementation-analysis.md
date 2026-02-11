---
Title: Implementation Analysis
Ticket: PI-002-SYSTEM-PROMPT-MW
Status: active
Topics:
    - analysis
    - webchat
    - refactor
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/middlewares/planning/lifecycle_engine.go
      Note: |-
        Planner system prompt uses systemprompt metadata guard
        Planning prompt uses systemprompt metadata
    - Path: ../../../../../../../pinocchio/pkg/webchat/engine.go
      Note: |-
        Injects SystemPrompt middleware in webchat engine composition
        Where system prompt middleware is attached
    - Path: ../../../../../../../pinocchio/pkg/webchat/engine_config.go
      Note: |-
        Stores SystemPrompt in EngineConfig signature
        SystemPrompt config/signature
    - Path: ../../../../../../../pinocchio/pkg/webchat/types.go
      Note: |-
        Profile.DefaultPrompt source of system prompt
        Profile DefaultPrompt source
    - Path: pkg/inference/middleware/systemprompt_middleware.go
      Note: |-
        Current system prompt middleware (append behavior)
        Current append-based system prompt middleware
ExternalSources: []
Summary: Plan to replace profile system prompt behavior with an idempotent system prompt middleware that replaces existing system prompts.
LastUpdated: 2026-01-27T20:15:00-05:00
WhatFor: Define the desired semantics and safe integration points for the new middleware.
WhenToUse: Before refactoring webchat system prompt handling.
---


# Implementation Analysis: Idempotent System Prompt Middleware

## 1) Problem Statement

The webchat profile’s `DefaultPrompt` is currently injected via `middleware.NewSystemPromptMiddleware`, which **appends** to the first system block or inserts a new system block if none exists. The desired behavior is **idempotent replacement**: when a system prompt is already present, it should be replaced rather than appended.

This change matters because:
- Appending can produce duplicated system prompts over time.
- A “replace” semantic allows a single authoritative system prompt to dominate the run.

The user request specifically states:

> “Replace SystemPrompt in profile with a custom SystemPromptMiddleware that inserts a system prompt (idempotently, if one is already present, replace it).”

---

## 2) Current Behavior (Baseline)

**Where the system prompt comes from**
- `pinocchio/pkg/webchat/types.go` → `Profile.DefaultPrompt` (profile-level string)
- `pinocchio/pkg/webchat/engine_builder.go` → `SystemPrompt` is placed into `EngineConfig`
- `pinocchio/pkg/webchat/engine.go` → `composeEngineFromSettings` uses:
  - `middleware.NewSystemPromptMiddleware(sysPrompt)`

**Current system prompt middleware behavior**
File: `geppetto/pkg/inference/middleware/systemprompt_middleware.go`

Behavior summary:
- If a block is already tagged with metadata `systemprompt`, skip (idempotency by tag).
- Otherwise:
  - If a system block exists, append `\n\n` + prompt to the first system block.
  - If no system block exists, prepend a new system block.

This yields *additive* behavior, not replacement.

---

## 3) Desired Behavior

### 3.1 Functional Requirements

- **Idempotent replacement**: if any system block exists, replace the prompt content (do not append).
- **Insert if missing**: if no system block exists, prepend a new one.
- **Avoid duplication**: multiple runs of the middleware should not change the turn after the first replacement.

### 3.2 Policy Boundaries

Two practical edge cases must be handled:

1) **Legacy planning middleware** (removed in PI-003)
   - Previously, planning set its own system prompt and relied on metadata to avoid rewrites.
   - With planning removed, this is no longer a constraint.

2) **User-supplied system blocks**
   - If a system block exists that the user intentionally set, replacement might be too aggressive.
   - We need a clear rule for whether to always replace or only replace blocks that were inserted by the middleware itself.

---

## 4) Design Decision

We will **modify the existing `NewSystemPromptMiddleware`** to use replacement semantics. There will be **one** system prompt middleware in the codebase.

**Rationale**
- Matches the requirement: “single SystemPromptMiddleware which overwrites a present system prompt block, otherwise inserts one up front.”
- Avoids duplicate APIs and confusion.
- Keeps call sites unchanged.

---

## 5) Proposed Semantics (Final)

The system prompt middleware will do the following, every time it runs:

1) **Find the first system block** (by order) in the turn.
2) If found:
   - Replace its text **exactly** with the provided prompt.
3) If not found:
   - Prepend a new system block with the prompt.
4) **Idempotency**:
   - If the first system block already contains the same prompt text, the middleware is a no‑op.

There is no append behavior and no additional metadata gate. The middleware simply makes the system prompt deterministic.

---

## 6) Integration Points in Webchat

**Where to plug in**
- `pinocchio/pkg/webchat/engine.go` currently calls:
  `middleware.NewSystemPromptMiddleware(sysPrompt)`

**Recommended change**
- No call site change. Update `NewSystemPromptMiddleware` implementation to replace rather than append.

**Profile interface**
- `Profile.DefaultPrompt` remains the same string source.
- The change is only in how it is applied, not how it is stored.

---

## 7) Testing Strategy

**Unit tests** (preferred location: geppetto/pkg/inference/middleware)

Cases to test:
1) No system block → new system block inserted.
2) First system block exists → replaced, not appended.
3) Replacement is idempotent across multiple middleware invocations.

**Integration tests** (optional):
- Add a webchat turn run that includes a system block and verify it is replaced in the middleware stage.

---

## 8) Migration Notes and Risks

**Risk: Behavior change for existing tools**
- If this middleware is used globally instead of webchat-only, some prompts may now be replaced rather than appended.

**Mitigation**
- Roll out with clear release notes; consider a short deprecation window if external consumers rely on append behavior.

---

## 9) Summary Recommendation

- **Modify `NewSystemPromptMiddleware`** to replace the first system block (or insert one if none exists).
- **Keep the API surface unchanged** so all current call sites pick up the new behavior.
- **Document the behavior change** since append semantics will no longer apply.
