---
Title: Prompt Resolution in Router and Middlewares
Ticket: MO-003-UNIFY-INFERENCE
Status: active
Topics:
    - inference
    - architecture
    - webchat
    - prompts
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: moments/backend/pkg/drive1on1/chat/chat.go
      Note: ResolveTemplateForTurnWithPrefix usage for document context
    - Path: moments/backend/pkg/inference/middleware/current_user_middleware.go
      Note: ResolveTemplateForTurn usage for user context prompt
    - Path: moments/backend/pkg/inference/middleware/thinkingmode/middleware.go
      Note: ResolvePromptForTurnWithPrefix usage for thinking mode
    - Path: moments/backend/pkg/prompts/manifest/manifest.go
      Note: Prompt slot templates and profile override slug derivation
    - Path: moments/backend/pkg/promptutil/resolve.go
      Note: Shared prompt resolution helpers and fallback logic
    - Path: moments/backend/pkg/webchat/loops.go
      Note: Step mode pauses and explicit tool result events in ToolCallingLoop
    - Path: moments/backend/pkg/webchat/moments_global_prompt_middleware.go
      Note: ResolvePromptForTurn call for system prompt injection
    - Path: moments/backend/pkg/webchat/router.go
      Note: Profile prompt slug resolution and injection in the router
    - Path: moments/backend/pkg/webchat/step_controller.go
      Note: Step mode pause/continue controller
ExternalSources: []
Summary: Explains why profile prompt slugs are resolved in the router, documents resolver usage in each middleware, and clarifies prompt slots/manifest behavior.
LastUpdated: 2026-01-16T12:28:40-05:00
WhatFor: Use when refactoring prompt resolution or unifying inference flows across apps.
WhenToUse: When touching prompt slugs, promptutil, or middleware prompt injection order.
---










# Prompt Resolution in Router and Middlewares (Moments)

## Goal

Explain (1) what it means that prompt slugs are resolved in the router, (2) why that is done, (3) the exact prompt-resolver call sites in each middleware, and (4) how prompt slots are declared and used.

## Definitions (so the rest is unambiguous)

- **Prompt slug**: A stable string key used to resolve text from the prompt store (ex: `profile.default.base`, `mw.global.team_suggestions.main`).
- **Profile prompt**: The base system prompt for a profile. It is resolved once per conversation (or per run) and inserted into the Turn as a system block.
- **Middleware prompt**: A prompt resolved inside a middleware during inference, typically per turn.
- **Prompt slot**: A named prompt requirement for a middleware (ex: `thinking_mode: exploring`), declared in the prompt manifest for admin UI and tooling.

## What it means that slugs are resolved in the router

When we say *“slugs are resolved in the router”*, we mean:

- The webchat router looks up the **profile base prompt slug** and resolves it to actual prompt text **before inference**.
- This resolution happens outside the middleware chain.
- The router then injects a system block into the Turn using that resolved text.

This is different from middleware prompts, which are resolved inside each middleware and may run on every turn.

### The actual router code path

Relevant file: `moments/backend/pkg/webchat/router.go`

#### Step 1: Resolve the profile slug

```
func (r *Router) resolveProfilePrompt(ctx context.Context, p profilesregistry.ProfileDescriptor, opts prompts.ResolveOptions) (string, error) {
    slug, err := profilesregistry.ResolvePromptSlug(p)
    parsedSlug, err := prompts.ParseSlug(slug)
    scope := r.resolutionScopeFromContext(ctx)
    record, err := r.promptResolver.Resolve(ctx, parsedSlug, scope, opts)
    ...
    return record.Text, nil
}
```

This is where the router turns `ProfileDescriptor` into a slug and resolves it using `prompts.Resolver`.

#### Step 2: Inject the resolved prompt

```
if resolved, err := r.resolveProfilePrompt(req.Context(), prof, resolveOpts); err == nil {
    EnsureProfileSystemPromptBlock(conv.Turn, resolved)
}
```

The router injects the resolved text into the Turn directly. This is done:

- on websocket connect (early), and
- again at run-loop start (after attaching identity/draft context).

### Why do this in the router?

- **Profile selection lives in the router**: the router chooses the profile slug and owns the HTTP/WS lifecycle, so it naturally owns the base prompt.
- **Base prompt is not a middleware**: it is a static, profile-level prompt. It must exist even if no middlewares are configured.
- **Identity/draft bundle context is attached in the router**: `resolutionScopeFromContext` and draft bundle resolution are only available in the router at the time of chat submission or WS connect.

The tradeoff is that the base prompt is injected outside the middleware ordering rules, which is why unification work is needed.

## Where prompt resolver is used in each middleware (exact code excerpts)

All of the following call `promptutil.Resolve*` helpers, which in turn call the `prompts.Resolver`. The exact call site per middleware is shown below.

### 1) `moments_global_prompt` middleware

File: `moments/backend/pkg/webchat/moments_global_prompt_middleware.go`

```
systemPrompt, err := promptutil.ResolvePromptForTurn(ctx, resolver, t, "moments_global_prompt.main")
if err != nil {
    return nil, err
}
```

Context: a system block is injected with a date placeholder replacement.

### 2) `thinking_mode` middleware

File: `moments/backend/pkg/inference/middleware/thinkingmode/middleware.go`

```
text, err := promptutil.ResolvePromptForTurnWithPrefix(ctx, resolver, t, thinkingModePrefix, modeName)
if err != nil {
    ...
}
```

Context: uses a fixed prefix `mw.thinking_mode.` instead of the profile prefix.

### 3) `summary_chunk_prompt` middleware

File: `moments/backend/pkg/inference/middleware/summary/summary_prompt_middleware.go`

```
msg, err := promptutil.ResolvePromptForTurn(ctx, resolver, t, "summary_chunk_prompt.main")
if err != nil {
    return nil, err
}
```

### 4) `coaching_conversation_summary` middleware

File: `moments/backend/pkg/inference/middleware/coachingconversationsummary/middleware.go`

```
msg, err := promptutil.ResolvePromptForTurn(ctx, resolver, t, "coaching_conversation_summary.main")
if err != nil {
    return nil, err
}
```

### 5) `coaching_guidelines` middleware

File: `moments/backend/pkg/inference/middleware/coachingguidelines/middleware.go`

```
mento, err := promptutil.ResolvePromptForTurn(ctx, resolver, t, "coaching_guidelines.mento")
if err != nil {
    return nil, err
}
icf, err := promptutil.ResolvePromptForTurn(ctx, resolver, t, "coaching_guidelines.icf")
if err != nil {
    return nil, err
}
```

### 6) `debate` middleware

File: `moments/backend/pkg/inference/middleware/debate/middleware.go`

```
instructions, err := promptutil.ResolvePromptForTurn(ctx, resolver, t, "debate.main")
if err != nil {
    return nil, err
}
```

### 7) `current_user` middleware (templated)

File: `moments/backend/pkg/inference/middleware/current_user_middleware.go`

```
msg, err := promptutil.ResolveTemplateForTurn(ctx, resolver, t, "current_user.main", map[string]string{
    "current_user_summary": summary,
})
if err != nil {
    return nil, err
}
```

### 8) `team_suggestions` middleware

File: `moments/backend/pkg/inference/middleware/team_suggestions_middleware.go`

```
header, err := promptutil.ResolvePromptForTurn(ctx, resolver, t, "team_suggestions.main")
if err != nil {
    return nil, err
}
```

### 9) `document_context` middleware (drive1on1)

File: `moments/backend/pkg/drive1on1/chat/chat.go`

```
msg, err := promptutil.ResolveTemplateForTurnWithPrefix(ctx, opts.Resolver, t, "mw.drive1on1.", "document_context", map[string]string{
    "document_list": docList,
})
if err != nil {
    return nil, err
}
```

### Common resolver path (shared logic)

All of the above calls run through `promptutil.ResolvePromptForTurn` / `ResolveTemplateForTurn`, which:

- reads Turn.Data (person/org/prefix/draft bundle)
- builds `prompts.ResolveOptions`
- calls `resolver.Resolve`
- falls back from profile prefix to `mw.global.`

File: `moments/backend/pkg/promptutil/resolve.go`

```
const defaultPrefix = "mw.global."
...
record, err := resolver.Resolve(ctx, firstSlug, scope, opts)
if (err != nil || record == nil) && prefix != defaultPrefix {
    fallbackSlug := prompts.MustSlug(defaultPrefix + slugSuffix)
    record, err = resolver.Resolve(ctx, fallbackSlug, scope, opts)
}
```

## Prompt slots: what they are and how they are used

Prompt slots are a *manifest-level description* of which prompts a middleware expects. They are not used to resolve prompts at runtime; they are used by tooling (profile editor, admin UI, validation) to show which prompt slugs exist and which are profile-overridable.

### Slot declaration

Slots are registered by middleware packages in their `init()` functions. Example from `thinking_mode`:

File: `moments/backend/pkg/inference/middleware/thinkingmode/middleware.go`

```
manifest.RegisterMiddlewarePromptSlots("thinking_mode", []manifest.PromptSlotTemplate{
    {Key: "exploring", GlobalSlug: "mw.thinking_mode.exploring", OverrideSlugSuffix: "thinking_mode.exploring"},
    {Key: "coaching", GlobalSlug: "mw.thinking_mode.coaching", OverrideSlugSuffix: "thinking_mode.coaching"},
    {Key: "onboarding", GlobalSlug: "mw.thinking_mode.onboarding", OverrideSlugSuffix: "thinking_mode.onboarding"},
})
```

Each slot template provides:

- **Key**: semantic name (ex: `exploring`)
- **GlobalSlug**: canonical default slug (used if no profile override)
- **OverrideSlugSuffix**: suffix combined with a profile prefix to produce a per-profile override slug

### Slot materialization by profile

The manifest builds concrete slot lists per profile. When a profile has `PromptSlugPrefix`, the manifest derives an override slug from the suffix.

File: `moments/backend/pkg/prompts/manifest/manifest.go`

```
if data.PromptSlugPrefix != "" && tpl.OverrideSlugSuffix != "" {
    slot.OverrideSlug = data.PromptSlugPrefix + tpl.OverrideSlugSuffix
}
```

### Where slots are used

Slots are surfaced in the profile editor and prompt management UI. The profile service builds a view that includes slot definitions and resolved prompt records.

File: `moments/backend/pkg/prompts/profile_service.go`

Conceptually:

```
profiles := manifest.AllProfiles()
for each profile:
  show base prompt + middleware prompt slots
  resolve global + override prompt records for display
```

This is why the manifest is separate from promptutil: one is for **administration and visibility**, the other is for **runtime resolution**.

## Summary of current split

- **Router** resolves *profile base prompts* because profile selection + session context are owned by the router.
- **Middlewares** resolve their own prompts on every turn using `promptutil`.
- **Prompt slots** exist only for tooling and admin UI; they do not drive runtime resolution.

This split is the precise reason a unification layer is needed: today there are two prompt-resolution entry points (router + middleware) with different ordering constraints.

## Moments step mode in the inference loop (context for event emission)

Step mode is a per-conversation pause/continue mechanism used for debugging tool calls and results. It lives in the webchat tool loop, not the engine or middleware layers.

### Where it is implemented

- `moments/backend/pkg/webchat/step_controller.go` defines the pause/continue controller.
- `moments/backend/pkg/webchat/loops.go` checks the controller and pauses at explicit points.

### How it works (key call sites)

The tool loop pauses after inference when tool calls are pending:

```
calls := toolblocks.ExtractPendingToolCalls(updated)
if len(calls) > 0 && conv.StepCtrl.IsEnabled() {
    pauseID := uuid.NewString()
    deadline := conv.StepCtrl.Pause(pauseID)
    ev := mentoevents.NewEventDebuggerPause(..., pauseID, "after_inference", ...)
    gepevents.PublishEventToContext(ctx, ev)
    conv.StepCtrl.Wait(pauseID, 30*time.Second)
}
```

And pauses again after tool results are appended:

```
toolblocks.AppendToolResultsBlocks(updated, appended)
if conv.StepCtrl.IsEnabled() {
    pauseID := uuid.NewString()
    deadline := conv.StepCtrl.Pause(pauseID)
    ev := mentoevents.NewEventDebuggerPause(..., pauseID, "after_tools", ...)
    gepevents.PublishEventToContext(ctx, ev)
    conv.StepCtrl.Wait(pauseID, 30*time.Second)
}
```

### Why this matters for event emission

Step mode depends on explicit event emission in the loop so the UI can render pause prompts. This is also where explicit tool result events are emitted.

## “Explicit tool result events” in Moments

In `moments/backend/pkg/webchat/loops.go`, after tool execution, the loop **publishes explicit `tool.result` events** rather than relying on downstream consumers to parse Turn blocks:

```
gepevents.PublishEventToContext(ctx, &gepevents.EventToolResult{
    EventImpl: gepevents.EventImpl{
        Type_:     gepevents.EventTypeToolResult,
        Metadata_: makeMeta(),
    },
    ToolResult: gepevents.ToolResult{
        ID:     tr.ID,
        Result: resultJSON,
    },
})
```

This is explicit because the event is emitted even though the tool result is already added to the Turn via `AppendToolResultsBlocks`. It ensures UI and analytics can react immediately without re-parsing the Turn structure.

## Should tool result events be emitted lower-level?

Short answer: **not with the current architecture**.

Reasons:

- Tool execution happens in the webchat loop (`ToolCallingLoop`), not inside the inference engine.
- The engine only knows about Turn input/output and emits inference-related events; it does not run tools or build tool results.
- Lower-level layers do not have access to the tool registry + executor + authorization context needed to run and event tool results.

### Could this move lower in the stack?

Only if the tool execution layer is moved into a shared engine-level component that has:

- access to tool registry (and auth context), and
- the same event bus/sink context used by webchat.

At the moment, emitting tool result events at a lower level would require major architectural changes, not just a refactor.

### Do lower levels emit events at all?

Yes, but only for **inference** events. The engine publishes partial/final output events to configured sinks and context sinks. Tool call/result events are emitted by the loop, not the engine.
