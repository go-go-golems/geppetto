---
Title: Prompt Resolver Analysis and Middleware Replacement (Moments)
Ticket: MO-003-UNIFY-INFERENCE
Status: active
Topics:
    - prompts
    - moments
    - webchat
    - llm-inference
    - middleware
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: moments/backend/pkg/app/app.go
      Note: Prompt resolver initialization and webchat wiring
    - Path: moments/backend/pkg/app/profiles.go
      Note: Profile base prompt slugs and prompt prefix configuration
    - Path: moments/backend/pkg/inference/middleware/thinkingmode/middleware.go
      Note: Prompt resolution with explicit prefix override
    - Path: moments/backend/pkg/prompts/resolver.go
      Note: Resolver interface + scope/draft bundle logic
    - Path: moments/backend/pkg/promptutil/resolve.go
      Note: Prompt resolution logic + prefix fallback used by middlewares
    - Path: moments/backend/pkg/webchat/moments_global_prompt_middleware.go
      Note: Global prompt middleware with promptutil resolution
    - Path: moments/backend/pkg/webchat/router.go
      Note: Profile prompt resolution and prompt prefix injection
ExternalSources: []
Summary: Analysis of the Moments prompt resolver flow and design options for tag-based prompt resolution middleware.
LastUpdated: 2026-01-16T10:41:00-05:00
WhatFor: Understand where prompt resolution happens today and how to unify it behind a single middleware.
WhenToUse: When touching prompt resolution, profile prompt injection, or middleware prompt slugs.
---









# Prompt Resolver Analysis and Tag-Based Middleware Alternative

## Goal

This document explains how prompt resolution currently works in Moments, where the resolver is used, and which prompts are resolved. It then proposes a tag-based middleware approach that could replace the scattered promptutil calls and make prompt resolution a single, centralized step.

## Current architecture (Moments)

### Key components

- `moments/backend/pkg/prompts/resolver.go` defines `prompts.Resolver` and implements scope-aware, draft-aware resolution.
- `moments/backend/pkg/promptutil/resolve.go` wraps prompt resolution for middleware usage by reading `Turn.Data` and performing prefix + fallback logic.
- `moments/backend/pkg/webchat/router.go` resolves **profile base prompts** directly (outside the middleware chain).
- `moments/backend/pkg/prompts/manifest/manifest.go` registers known profile + middleware prompt slots for admin UI and prompt tooling.

### Prompt resolver wiring

The prompt resolver is created on app startup and required by webchat:

- `moments/backend/pkg/app/app.go`:
  - `App.Initialize` -> `prompts.NewResolver(repo)` stored as `App.PromptResolver`
  - `App.BuildWebchatRegistries` -> `RegisterPromptMiddlewaresWithRegistry(a.PromptResolver)`
  - `App.InitWebChat` -> `webchat.NewRouterFromRegistries(..., a.PromptResolver, ...)`

If `PromptResolver` is nil, webchat refuses to start.

## How resolution works today

### 1) Profile base prompt resolution (router)

Profile base prompts are resolved in the webchat router, not by middleware:

- `moments/backend/pkg/webchat/router.go`:
  - `resolveProfilePrompt(ctx, profile, opts)`
  - uses `profilesregistry.ResolvePromptSlug(profile)` to compute slug
  - uses `resolutionScopeFromContext(ctx)` to compute person/org scope
  - calls `promptResolver.Resolve(ctx, slug, scope, opts)`
  - injects the text via `EnsureProfileSystemPromptBlock(turn, text)`

Where it is called:

- On WS connect (`wsHandler`): resolves profile prompt early using the request context.
- On run loop start (`handleChatRequest` goroutine): resolves again after attaching identity session + draft bundle.

Implication: the base prompt is not part of middleware ordering and can be injected twice if not idempotent.

### 2) Middleware prompt resolution (promptutil)

Middleware prompt resolution uses the `promptutil` helpers:

- `ResolvePromptForTurn` and `ResolveTemplateForTurn` inspect `Turn.Data` and then call `prompts.Resolver`.
- `promptResolutionDataFromTurn` extracts:
  - `turnkeys.PersonID`
  - `turnkeys.OrgID`
  - `turnkeys.PromptSlugPrefix`
  - `turnkeys.DraftBundleID`
- `resolvePromptText`:
  - uses prefix from `Turn.Data[PromptSlugPrefix]` or default `mw.global.`
  - calls `resolver.Resolve(ctx, slug, scope, opts)`
  - if prefix != `mw.global.` and resolution fails, falls back to `mw.global.`

So middleware prompts are **profile-aware** by default, based on the prefix injected into `Turn.Data` by the router.

### 3) Draft bundle preview

Draft bundle resolution is supported in two places:

- `promptutil.ResolvePromptForTurn` reads `turnkeys.DraftBundleID` and `turnkeys.PersonID` and passes `ResolveOptions{DraftBundleID, UserID}`.
- `webchat.Router.resolveProfilePrompt` also passes draft bundle options when available on the request or in `Turn.Data`.

## Where the resolver is used (and which prompts it resolves)

### A) Profile base prompts (resolved in router)

These slugs are resolved for base profile system prompts:

- `profile.default.base`
- `profile.doclens.base`
- `profile.team-select.base`
- `profile.drive1on1.base`
- `profile.drive1on1-summary.base`
- `profile.find-coaching-transcripts.base`
- `profile.thinking-mode.base`
- `profile.presidential-debate.base`
- `profile.coaching-conversation-summary.base`
- `profile.coaching-conversations-summary.base`

Source: `moments/backend/pkg/app/profiles.go` (via `DefaultPromptSlug`).

### B) Middleware prompts (resolved via promptutil)

These are the known promptutil-backed prompt slots, with slug suffixes and files that resolve them:

- `moments_global_prompt.main`
  - `moments/backend/pkg/webchat/moments_global_prompt_middleware.go`
  - Resolved via `ResolvePromptForTurn` (prefix from `Turn.Data`, fallback to `mw.global.`).
- `thinking_mode.{exploring|coaching|onboarding}`
  - `moments/backend/pkg/inference/middleware/thinkingmode/middleware.go`
  - Resolved via `ResolvePromptForTurnWithPrefix` with prefix `mw.thinking_mode.`.
- `summary_chunk_prompt.main`
  - `moments/backend/pkg/inference/middleware/summary/summary_prompt_middleware.go`
- `coaching_conversation_summary.main`
  - `moments/backend/pkg/inference/middleware/coachingconversationsummary/middleware.go`
- `coaching_guidelines.{mento|icf}`
  - `moments/backend/pkg/inference/middleware/coachingguidelines/middleware.go`
- `debate.main`
  - `moments/backend/pkg/inference/middleware/debate/middleware.go`
- `current_user.main` (templated)
  - `moments/backend/pkg/inference/middleware/current_user_middleware.go`
- `team_suggestions.main`
  - `moments/backend/pkg/inference/middleware/team_suggestions_middleware.go`
- `document_context` (templated)
  - `moments/backend/pkg/drive1on1/chat/chat.go` (prefix override `mw.drive1on1.`)

Most of these are registered in the prompt manifest for admin UI with global + override slug templates:

- `moments/backend/pkg/prompts/manifest/manifest.go`
- Middleware `init()` blocks call `manifest.RegisterMiddlewarePromptSlots`.

## Key observations / constraints

- Prompt resolution logic is duplicated across many middlewares.
- Profile base prompt resolution happens outside middleware ordering, creating a separate codepath.
- Profile-specific overrides depend on `Turn.Data[PromptSlugPrefix]` and consistent router behavior.
- Thinking mode is special: it uses a fixed prefix (`mw.thinking_mode.`), bypassing the profile prefix.
- Some middlewares have custom post-processing (e.g., `moments_global_prompt` inserts date).
- Draft bundle resolution is split across router and middleware paths.

## Tag-based middleware alternative

### Goal

Replace per-middleware `promptutil.Resolve*` calls with a single middleware that:

- scans the turn for prompt references (tags)
- resolves them via `prompts.Resolver` using shared logic
- replaces the placeholder with resolved prompt text

This centralizes prompt resolution, unifies fallback rules, and eliminates duplicated slug logic.

### Option A: PromptRef block metadata (recommended)

Define a prompt reference as block metadata rather than raw text:

```
Block.Metadata["moments.prompt_ref.v1"] = {
  "slug_suffix": "coaching_guidelines.mento",
  "prefix_override": "",          // optional
  "use_profile_prefix": true,      // default true
  "template_values": {"name":"..."},
  "insert_role": "system",
  "idempotency_key": "coaching_guidelines"
}
```

A central `prompt_resolution` middleware would:

1) Walk `t.Blocks` and find any blocks tagged with `prompt_ref`.
2) Compute the effective slug:
   - If `prefix_override` is set -> `prefix_override + slug_suffix`
   - Else if `use_profile_prefix` -> `Turn.Data[PromptSlugPrefix] + slug_suffix`
   - Else -> `defaultPrefix + slug_suffix`
3) Resolve via `prompts.Resolver` using scope + draft bundle.
4) If `template_values` exist, apply templating (or error if unused/missing).
5) Replace the placeholder block content with the resolved text, preserving metadata.

Pseudo:

```
for each block in turn.blocks:
  ref := promptRefFromMetadata(block)
  if ref == nil:
    continue
  slug := resolveSlug(ref, turn.data)
  text := resolvePrompt(ctx, slug, turn.data)
  if ref.templateValues:
     text = applyTemplate(text, ref.templateValues)
  block = systemTextBlock(text)
  block.Metadata.merge(refMetadata)
```

### Option B: PromptRef list in Turn.Data

Store references in `Turn.Data[turnkeys.PromptRefs]` and let the middleware expand them into blocks. This is useful if you want to avoid placeholder blocks, but it complicates ordering because the middleware must decide where to insert them.

### Option C: Inline tag replacement in block text

Use inline tags like `{{prompt:coaching_guidelines.mento}}` and have the middleware substitute the text. This is more fragile (string parsing + collisions) and mixes control data with user-visible prompt text.

## How this replaces current resolution

### Profile base prompts

Instead of `resolveProfilePrompt(...)` in the router, insert a prompt ref block that carries the full profile slug (already computed by `profilesregistry.ResolvePromptSlug`):

```
ref := PromptRef{slug: "profile.default.base"}
turn.blocks = prepend(systemPromptRefBlock(ref))
```

Then the prompt-resolution middleware resolves it like any other prompt. This unifies ordering and idempotency with other middlewares.

### Middleware prompts

Each middleware becomes a *tagging middleware* instead of a *resolving middleware*. Example:

```
// thinking_mode middleware
mode := getModeFromTurn(t)
ref := PromptRef{
  slug_suffix: mode,
  prefix_override: "mw.thinking_mode.",
  use_profile_prefix: false,
}
insertPromptRefBlock(t, ref)
return next(ctx, t)
```

### Template + post-processing

Template support should move into the central middleware, but post-processing still needs hooks:

- `moments_global_prompt` currently injects `{{date}}`.
  - Option: treat date as a template key and let the resolver fill it from `Turn.Data` (e.g. `turnkeys.BrowserTime`).
  - This would remove custom code and make the prompt fully data-driven.

## Integration points and API hooks

### Required data sources

- `Turn.Data[PromptSlugPrefix]` (profile namespace)
- `Turn.Data[PersonID]`, `Turn.Data[OrgID]`
- `Turn.Data[DraftBundleID]`
- Session context (for draft bundle ownership checks in router path)

### Where the middleware should run

Run the prompt-resolution middleware early in the chain, before any middleware that depends on resolved prompt text (e.g., ordering, logblocks, langfuse).

Suggested ordering:

1) tagging middlewares (e.g., thinking_mode, moments_global_prompt, current_user)
2) prompt_resolution middleware
3) downstream middlewares (ordering, logging, langfuse, compression)

## Compatibility and migration strategy

1) Introduce the new prompt-resolution middleware without removing existing promptutil calls.
2) Convert one middleware at a time to emit prompt refs instead of resolving.
3) When all promptutil usage is removed, delete `promptutil.Resolve*` usage and reduce duplicated logic.
4) Move base profile prompt insertion into a prompt-ref block inserted by the router (or a profile middleware).

## Open questions

- Should the prompt-resolution middleware own `PromptSlugPrefix` fallback logic, or should it call `promptutil` internally for backwards compatibility?
- How do we store prompt ref metadata so that it remains stable across serialization and logging?
- Should the manifest be the source of truth for slug derivation (slot-based resolution) instead of raw suffixes?
- Should prompt resolution be allowed to fail softly (log + continue) or strictly (propagate error)?

## Appendix: Sequence diagram (current vs proposed)

### Current (profile prompt in router + middleware prompts)

```
HTTP/WS request
    -> Router resolves profile prompt (Resolver)
    -> Turn has base system block
    -> Middlewares run
         -> each middleware resolves promptutil (Resolver)
         -> injects block
    -> Engine inference
```

### Proposed (tag-based prompt resolution middleware)

```
HTTP/WS request
    -> Router inserts base prompt ref block
    -> Middlewares add prompt ref blocks
    -> Prompt-resolution middleware resolves all refs (Resolver)
    -> Downstream middlewares (ordering/logging)
    -> Engine inference
```
