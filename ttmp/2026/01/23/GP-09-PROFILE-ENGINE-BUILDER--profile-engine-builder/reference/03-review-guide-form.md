---
Title: Review Guide + Form
Ticket: GP-09-PROFILE-ENGINE-BUILDER
Status: active
Topics:
    - architecture
    - backend
    - go
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/02-postmortem.md
      Note: |-
        Engineering narrative + review checklist (what shipped, why, commits, tests)
        Postmortem context
    - Path: moments/backend/pkg/webchat/engine_from_req.go
      Note: |-
        Moments request-policy implementation (primary item to review for precedence)
        Moments precedence implementation
    - Path: moments/backend/pkg/webchat/engine_from_req_test.go
      Note: Moments precedence tests
    - Path: moments/backend/pkg/webchat/router.go
      Note: Moments Router WS behavior (token injection + draft_bundle_id) and delegation boundary
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: Pinocchio override enforcement and config building (Tools + AllowOverrides)
    - Path: pinocchio/pkg/webchat/engine_config.go
      Note: Pinocchio signature semantics (Tools included, StepSettings metadata sanitized)
    - Path: pinocchio/pkg/webchat/engine_from_req.go
      Note: |-
        Pinocchio request-policy implementation (primary item to review for precedence)
        Pinocchio precedence implementation
    - Path: pinocchio/pkg/webchat/engine_from_req_test.go
      Note: Pinocchio precedence tests
    - Path: pinocchio/pkg/webchat/router.go
      Note: Pinocchio Router now delegates request policy and filters tools by config
ExternalSources: []
Summary: 'Fill-in review guide + form for GP-09: explains what changed, why it matters, what to scrutinize, and provides decision prompts to validate precedence/override/tool semantics.'
LastUpdated: 2026-01-23T13:36:56.091592497-05:00
WhatFor: Use as the reviewer’s worksheet to understand GP-09 changes and make informed decisions about precedence rules, override policy, tool availability, and signature semantics.
WhenToUse: Use during code review of GP-09-related commits or when deciding follow-up work (e.g., align Moments with Pinocchio semantics).
---


# Review Guide + Form

## Goal

Help you (as reviewer/owner) rapidly:

1. Understand the architecture change (what moved out of Router, what didn’t).
2. Validate that behavior is correct (especially precedence rules).
3. Make explicit decisions about the remaining “policy” questions (overrides, tools, signature).
4. Record those decisions in a way that is actionable for future follow-ups.

## Context

Ticket `GP-09-PROFILE-ENGINE-BUILDER` began as a design analysis for extracting request/profile policy out of webchat Routers. We implemented the Phase 1 pattern (request policy builder) in:

- Pinocchio (commit `3b8cae7`)
- Moments (commit `fe3e9dcf`)

In Pinocchio we also implemented a “tools are driven by config” pattern and enforced `Profile.AllowOverrides`, because Pinocchio already had the right seams (explicit EngineBuilder + config signature).

### What changed conceptually

Before, the Router handlers each contained their own logic to decide:

- `conv_id` (create if missing)
- `profile` (path/query/cookie/existing/default precedence)
- `overrides` (what shape is allowed, and whether it’s allowed at all)

After, that logic is centralized behind a request-policy builder:

```text
Router (HTTP/WS handler)
  -> EngineFromReqBuilder.BuildEngineFromReq(req)
      -> returns EngineBuildInput{convID, profileSlug, overrides} (+ parsed body for HTTP)
  -> Router orchestrates run (build/get conversation, registry, start inference)
```

This should reduce drift (HTTP vs WS precedence mismatches), improve testability, and clarify what remains “policy”.

## Quick Reference

### Artifacts to review (minimal set)

Pinocchio:

- `pinocchio/pkg/webchat/engine_from_req.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/engine_builder.go`
- `pinocchio/pkg/webchat/engine_config.go`
- `pinocchio/pkg/webchat/engine_from_req_test.go`

Moments:

- `moments/backend/pkg/webchat/engine_from_req.go`
- `moments/backend/pkg/webchat/router.go`
- `moments/backend/pkg/webchat/engine_from_req_test.go`

### How to validate quickly

```bash
cd pinocchio && go test ./... -count=1
cd moments/backend && go test ./... -count=1
```

### Key review lens: “policy vs orchestration”

Use this table to decide if something belongs in the request-policy builder or the Router:

| Category | Examples | Belongs in | Why |
|---|---|---|---|
| Request policy | pick `profile`, defaulting, cookie fallback, existing-conv fallback, `conv_id` generation | request-policy builder | should be testable and consistent across HTTP/WS |
| Transport glue | websocket upgrade, token-injection, draft_bundle_id parsing, response frames | Router handler | tied to transport and request lifecycle |
| Engine config policy | profile defaults + engine-shaping overrides, signature inputs | engine builder/config | should drive rebuild decisions and runtime behavior |
| Runtime orchestration | build tool registry, start inference, publish events, attach sinks | Router orchestration | side effects and run lifecycle |

### Pseudocode: what you should expect after refactor

**Pinocchio**

```pseudo
BuildEngineFromReq(req):
  if req is WS (GET):
    convID = query.conv_id (required)
    profile = query.profile or cookie.chat_profile or existingConv.profile or "default"
    assert profile exists
    return {convID, profile, overrides=nil}

  if req is HTTP chat (POST):
    body = decode JSON
    convID = body.conv_id or new UUID
    profile = path /chat/{profile} or existingConv.profile or cookie.chat_profile or "default"
    assert profile exists
    return {convID, profile, overrides=body.overrides}, body
```

**Moments**

```pseudo
BuildEngineFromReq(req):
  if WS (GET):
    convID required
    profile = query.profile or cookie.chat_profile or existingConv.profile or "default"
  if HTTP chat (POST):
    body = decode JSON
    convID = body.conv_id or new UUID
    profile = explicit (query then path) or cookie.chat_profile or existingConv.profile or "default"
```

If your intended precedence differs, that’s the most important decision point to record below.

## Usage Examples

### How to use this document

1. Skim the “Textbook” sections (below) to remind yourself why each policy matters.
2. Fill out the form sections.
3. If you decide on changes, treat the form answers as the follow-up task list.

---

# Textbook: What to review (and why it matters)

## 1) Precedence rules (highest risk of subtle regressions)

Precedence bugs are painful because they look like “random profile switching” to users.

Typical symptoms:

- WS uses profile A while HTTP uses profile B for the same `conv_id`.
- Cookie unexpectedly overrides explicit profile selection.
- Existing conversations silently keep an old profile when you expected a new explicit request to win.

Where to look:

- Pinocchio: `pinocchio/pkg/webchat/engine_from_req.go`
- Moments: `moments/backend/pkg/webchat/engine_from_req.go`

How to validate manually:

- Send a `/chat/{profile}` request while a cookie is set to a different profile and confirm which wins.
- Reuse an existing `conv_id` that was previously created under another profile and confirm the new request does (or does not) migrate it.

## 2) Override policy (Pinocchio enforced; Moments not yet)

Overrides are a security/behavior policy:

- If overrides can alter `system_prompt`, tool availability, or middleware stack, they can materially change the system’s behavior.
- Therefore, if you want per-profile stability, you usually need an explicit allow/deny gate.

Where to look (Pinocchio):

- `pinocchio/pkg/webchat/engine_builder.go` (`AllowOverrides` enforcement)

Decision point:

- Do you want Moments to eventually support engine-shaping overrides?
  - If yes, Moments likely needs an explicit `AllowOverrides` flag in its profile registry first.

## 3) Tools driven by config (Pinocchio implemented)

This is about preventing drift between:

- “what the config signature says the engine is” and
- “what tools are actually usable at runtime”

If tools aren’t part of the signature and aren’t filtered from a config source-of-truth, you can get:

- cache/rebuild decisions that ignore tool changes
- “tool not found” errors that depend on which code path built the registry

Where to look:

- `pinocchio/pkg/webchat/engine_config.go` (Tools in signature)
- `pinocchio/pkg/webchat/router.go` (tool registry filtered by config)

## 4) Signature semantics and secrets

We want signatures to be:

- deterministic
- sensitive to behavior changes
- safe to log/debug (no secrets like API keys)

Where to look:

- Pinocchio: `pinocchio/pkg/webchat/engine_config.go` uses StepSettings metadata (sanitized) + Tools.

---

# Form: Review worksheet (fill this out)

## Section A — Your high-level understanding

- [ ] I understand what “request policy extraction” means in this context.
- [ ] I understand why precedence rules are the primary risk.
- [ ] I understand the difference between request policy and engine config policy.

**In one sentence, what problem is GP-09 solving?**

<WRITE YOUR TEXT HERE>

## Section B — Precedence decision record (must fill)

### B1) Intended precedence for HTTP chat (Moments + Pinocchio)

Check the intended order (top wins):

- [ ] Explicit profile (path/query) wins
- [ ] Cookie `chat_profile` wins
- [ ] Existing conversation profile wins
- [ ] Default `"default"` wins

**If your intended order differs, write the exact order here:**

<WRITE YOUR TEXT HERE>

### B2) Intended precedence for WS join

- [ ] `?profile=...` wins
- [ ] Cookie `chat_profile` wins
- [ ] Existing conversation profile wins
- [ ] Default `"default"` wins

**If your intended order differs, write the exact order here:**

<WRITE YOUR TEXT HERE>

### B3) Migration semantics when reusing `conv_id`

When a request supplies a different explicit profile than the existing conversation’s profile:

- [ ] We should migrate the conversation to the new profile
- [ ] We should reject with an error
- [ ] We should ignore explicit profile and keep existing

Decision:

<WRITE YOUR TEXT HERE>

## Section C — Overrides (security/behavior policy)

### C1) Should clients be allowed to supply engine-shaping overrides?

Define “engine-shaping overrides” as any of:

- `system_prompt`
- `middlewares`
- `tools`

Decision:

- [ ] Yes, but only for some profiles
- [ ] Yes, for all profiles
- [ ] No, never (remove the feature)

**Reasoning:**

<WRITE YOUR TEXT HERE>

### C2) For Pinocchio: do you agree with the enforcement boundary?

Pinocchio currently rejects engine-shaping overrides when `AllowOverrides=false`.

- [ ] Yes
- [ ] No (explain what should be allowed/blocked instead)

Notes:

<WRITE YOUR TEXT HERE>

### C3) For Moments: do we want parity with Pinocchio?

- [ ] Yes: add `AllowOverrides` to Moments profile registry and enforce it
- [ ] No: keep overrides limited (and possibly remove engine-shaping overrides support)
- [ ] Unsure: leave as-is for now

Notes:

<WRITE YOUR TEXT HERE>

## Section D — Tools semantics (Pinocchio implemented; Moments TBD)

### D1) Pinocchio: should `overrides[\"tools\"]` exist?

- [ ] Yes (keep + document supported shape)
- [ ] No (remove feature; rely only on profile defaults)

Notes:

<WRITE YOUR TEXT HERE>

### D2) Moments: do we want tools overrides in the future?

- [ ] Yes (requires explicit profile flag + enforcement)
- [ ] No

Notes:

<WRITE YOUR TEXT HERE>

## Section E — What to inspect in code (check off)

### Pinocchio

- [ ] Request policy precedence: `pinocchio/pkg/webchat/engine_from_req.go`
- [ ] Router delegation: `pinocchio/pkg/webchat/router.go`
- [ ] Override enforcement: `pinocchio/pkg/webchat/engine_builder.go`
- [ ] Signature semantics: `pinocchio/pkg/webchat/engine_config.go`
- [ ] Tests encode intended behavior: `pinocchio/pkg/webchat/engine_from_req_test.go`

### Moments

- [ ] Request policy precedence: `moments/backend/pkg/webchat/engine_from_req.go`
- [ ] WS handler still does token injection and draft bundle parsing: `moments/backend/pkg/webchat/router.go`
- [ ] Tests encode intended behavior: `moments/backend/pkg/webchat/engine_from_req_test.go`

## Section F — Follow-up tasks (generated from your answers)

Once you fill this out, create follow-up tasks here:

- [ ] <TASK 1>
- [ ] <TASK 2>
- [ ] <TASK 3>

## Related

- Postmortem: `reference/02-postmortem.md`
- Diary: `reference/01-diary.md`
