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

1) **Planning middleware** (pinocchio/pkg/middlewares/planning/lifecycle_engine.go)
   - It sets system prompts for the planning call and marks them with `systemprompt` metadata to avoid being overwritten by the normal system prompt middleware.
   - A “replace always” system prompt middleware could overwrite planner prompts, which would break planning execution.

2) **User-supplied system blocks**
   - If a system block exists that the user intentionally set, replacement might be too aggressive.
   - We need a clear rule for whether to always replace or only replace blocks that were inserted by the middleware itself.

---

## 4) Design Options

### Option A: Modify Existing `NewSystemPromptMiddleware`

**Pros**
- Single change, no duplication.
- All current uses adopt the new semantics.

**Cons**
- Risky: planning middleware currently relies on the `systemprompt` tag to prevent rewriting. If we change logic globally, we must preserve a “do not override” mechanism.

**Mitigation**
- Add a “lock” metadata key (e.g., `systemprompt:locked`) to prevent replacement.

---

### Option B: Create a New Middleware (e.g., `NewReplaceSystemPromptMiddleware`)

**Pros**
- Minimal blast radius. Webchat can opt into replacement behavior without changing other users.
- Planning middleware can continue to rely on the existing appending semantics.

**Cons**
- Two system prompt middlewares may confuse future maintainers if not documented.

**Recommendation**
Option B is safer and aligns with the user request (“custom SystemPromptMiddleware”).

---

### Option C: Parameterized Middleware

Add a parameter to the existing middleware:

```
NewSystemPromptMiddleware(prompt, SystemPromptModeReplace)
```

**Pros**
- Single API with explicit behavior.

**Cons**
- Requires signature change across existing call sites.
- Still riskier than Option B.

---

## 5) Proposed Semantics (Recommended)

Implement a **replacement system prompt middleware** with these rules:

1) Find the first system block (by order) in the turn.
2) If found:
   - If the block has metadata `systemprompt_lock = true`, skip replacement.
   - Otherwise, set its text **exactly** to the new prompt.
   - Mark metadata `systemprompt_source = "profile"` (or similar).
3) If not found:
   - Prepend a new system block with the prompt.
   - Mark metadata `systemprompt_source = "profile"`.
4) **Idempotency**: if the first system block already has the same prompt and metadata, do nothing.

This approach ensures:
- Replacement happens exactly once.
- Planning prompts can opt out via metadata (explicit lock).
- The middleware can be safely applied multiple times.

---

## 6) Integration Points in Webchat

**Where to plug in**
- `pinocchio/pkg/webchat/engine.go` currently calls:
  `middleware.NewSystemPromptMiddleware(sysPrompt)`

**Recommended change**
- Replace with new middleware (Option B), e.g.:

```
mws = append(mws, middleware.NewReplaceSystemPromptMiddleware(sysPrompt))
```

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
4) `systemprompt_lock` present → skip replacement.

**Integration tests** (optional):
- Add a webchat turn run that includes a system block and verify it is replaced in the middleware stage.

---

## 8) Migration Notes and Risks

**Risk: Planning middleware override**
- Must ensure new middleware respects a lock to avoid overriding planner prompts.

**Risk: Behavior change for existing tools**
- If this middleware is used globally instead of webchat-only, some prompts may now be replaced rather than appended.

**Mitigation**
- Keep the new middleware scoped to webchat until proven safe.

---

## 9) Summary Recommendation

- **Create a new middleware** in `geppetto/pkg/inference/middleware` with replace semantics.
- **Use it in webchat** instead of the current append-based middleware.
- **Add a lock metadata key** for cases where replacement should not occur (planning middleware).
- Keep the existing `NewSystemPromptMiddleware` unchanged to avoid breaking other call sites.
