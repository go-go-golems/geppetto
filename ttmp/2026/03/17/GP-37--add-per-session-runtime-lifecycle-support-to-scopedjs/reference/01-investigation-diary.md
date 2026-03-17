---
Title: Investigation diary
Ticket: GP-37
Status: active
Topics:
    - geppetto
    - tools
    - architecture
    - js-bindings
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/tool.go
      Note: Current registrar behavior reviewed as the starting point
    - Path: /home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/session/context.go
      Note: Default session identifier source considered for the future design
ExternalSources: []
Summary: Chronological notes for the per-session scopedjs runtime ticket.
LastUpdated: 2026-03-17T08:35:07.139341274-04:00
WhatFor: Preserve the exact reasoning path, commands, and conclusions used to scope the future per-session runtime feature.
WhenToUse: Use when reviewing how the ticket was framed, what existing code was inspected, and why the proposed design is shaped the way it is.
---

# Investigation diary

## Goal

Create a future-facing ticket for true per-session runtime reuse in `scopedjs`, with enough design detail that a new intern can understand both the current system and the planned implementation path.

## Context

The current `scopedjs` package already exposes two honest runtime behaviors:

- `RegisterPrebuilt(...)` uses one existing runtime for all calls
- `NewLazyRegistrar(...)` builds a fresh runtime for each call

That resolved the earlier misleading `StateMode` situation, but it left an intentionally missing middle option: session-scoped runtime reuse. The goal of this ticket is to describe that missing feature without pretending it is already simple.

## Quick Reference

### Commands run during ticket setup

```bash
docmgr ticket create-ticket --root geppetto/ttmp --ticket GP-37 --title "Add per-session runtime lifecycle support to scopedjs" --topics geppetto,tools,architecture,js-bindings
docmgr doc add --root geppetto/ttmp --ticket GP-37 --doc-type design-doc --title "Per-session scopedjs runtime lifecycle analysis design and intern implementation guide" --summary "Detailed design and implementation guide for adding true per-session runtime reuse to scopedjs."
docmgr doc add --root geppetto/ttmp --ticket GP-37 --doc-type reference --title "Investigation diary" --summary "Chronological notes for the per-session scopedjs runtime ticket."
docmgr doc add --root geppetto/ttmp --ticket GP-37 --doc-type sources --title "GitHub issue body" --summary "Exact GitHub issue body filed for the per-session scopedjs runtime ticket."
docmgr doc relate --root geppetto/ttmp --ticket GP-37 --file-note "/home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/tool.go:Current registration strategies define runtime reuse behavior and are the main seam for a future per-session mode" --file-note "/home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/runtime.go:BuildRuntime returns the owned runtime handle that a future session pool would cache" --file-note "/home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/tools/scopedjs/eval.go:RunEval assumes practical exclusive access to one runtime during evaluation" --file-note "/home/manuel/workspaces/2026-03-15/add-scoped-js/geppetto/pkg/inference/session/context.go:Provides the default session identifier source for a future per-session registrar"
gh issue create --repo go-go-golems/geppetto --title "Add true per-session runtime lifecycle support to scopedjs" --body-file geppetto/ttmp/2026/03/17/GP-37--add-per-session-runtime-lifecycle-support-to-scopedjs/sources/01-github-issue-body.md
docmgr doctor --root geppetto/ttmp --ticket GP-37 --stale-after 30
```

### Code paths inspected

- `geppetto/pkg/inference/tools/scopedjs/schema.go`
- `geppetto/pkg/inference/tools/scopedjs/tool.go`
- `geppetto/pkg/inference/tools/scopedjs/runtime.go`
- `geppetto/pkg/inference/tools/scopedjs/eval.go`
- `geppetto/pkg/inference/tools/scopedjs/tool_test.go`
- `geppetto/pkg/inference/session/context.go`
- `geppetto/pkg/inference/session/session.go`

### Main conclusions

- There is no remaining live `StateMode` API in `scopedjs`; that cleanup already happened.
- A true per-session mode is not just a description tweak. It needs runtime pooling, session key resolution, synchronization, cleanup, and tests.
- Geppetto already has a good default session identity source: `session.SessionIDFromContext(ctx)`.
- The correct home for lifecycle configuration is the registration or runtime-manager layer, not per-call eval options.
- The upstream tracker for this future work is `go-go-golems/geppetto#304`.

## Usage Examples

### Example reasoning model

When planning the future feature, reason about it as a third lifecycle strategy:

```text
prebuilt     => one runtime reused for everybody
lazy         => one fresh runtime per call
per_session  => one runtime per session key
```

### Example future execution flow

```text
tool call
  -> read session id from context
  -> look up runtime entry in pool
  -> if missing, build runtime once for that session
  -> lock session runtime
  -> run eval
  -> unlock session runtime
  -> update last-used timestamp
```

## Detailed Notes

### 1. Verified the current `scopedjs` lifecycle surface

I reviewed `pkg/inference/tools/scopedjs/tool.go` first. The important observation is that current lifecycle behavior is registration-driven:

- prebuilt registration closes over `handle.Runtime`
- lazy registration resolves scope and calls `BuildRuntime(...)` per invocation

That means the next feature should be introduced in the same part of the system. Reintroducing lifecycle semantics in `EvalOptions` would repeat the original mistake, because `EvalOptions` configures a single eval call, not runtime ownership.

### 2. Verified the runtime-construction seam

`pkg/inference/tools/scopedjs/runtime.go` already has the right construction seam for future pooling:

- `BuildRuntime(...)` takes `context.Context`, `EnvironmentSpec`, and `scope`
- it returns a `BuildResult` with `Runtime`, `Manifest`, `Meta`, and `Cleanup`

That means a session pool does not need a second runtime-construction path. It can cache `BuildResult`-like entries or wrap them in a pool entry struct.

### 3. Checked whether session identity already exists in Geppetto

I inspected `pkg/inference/session/context.go` and `pkg/inference/session/session.go`. The answer is yes:

- `session.WithSessionMeta(ctx, sessionID, inferenceID)` injects session metadata into context
- `session.SessionIDFromContext(ctx)` reads the current session ID back out
- `session.Session` itself already represents a long-lived session model with a stable `SessionID`

This is the strongest argument against inventing a new `scopedjs`-specific session concept. The future feature should default to existing Geppetto session plumbing.

### 4. Identified the real hard parts

The missing work is not runtime creation. The missing work is runtime ownership:

- which calls map to the same runtime
- how concurrent calls for one session are serialized
- what happens when a runtime becomes invalid
- when idle session runtimes are evicted
- how the feature behaves if no session ID is present

Those are the design points the main guide focuses on.

### 5. Filed the upstream GitHub issue

I created:

- `https://github.com/go-go-golems/geppetto/issues/304`

The issue body was taken from the local `sources/01-github-issue-body.md` file so the local ticket and upstream tracker stay synchronized.

### 6. Fixed the saved issue-body doc and re-ran validation

`docmgr doctor` initially failed because `sources/01-github-issue-body.md` had been overwritten with plain issue text and no YAML frontmatter. I restored proper frontmatter, kept the issue text under a heading, and reran:

```bash
docmgr doctor --root geppetto/ttmp --ticket GP-37 --stale-after 30
```

Result:

- all checks passed

## Related

- Main guide: [../design-doc/01-per-session-scopedjs-runtime-lifecycle-analysis-design-and-intern-implementation-guide.md](../design-doc/01-per-session-scopedjs-runtime-lifecycle-analysis-design-and-intern-implementation-guide.md)
- GitHub issue body: [../sources/01-github-issue-body.md](../sources/01-github-issue-body.md)
