---
Title: Implementation Plan
Ticket: RDX-008-REMOVE-CONVERSATION
Status: active
Topics:
    - refactor
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/examples/middleware-inference/main.go
      Note: |-
        Example using conversation builder
        Example using conversation
    - Path: pkg/conversation/builder/builder.go
      Note: |-
        Builder to remove with conversation package
        Builder to remove
    - Path: pkg/conversation/manager-impl.go
      Note: Manager implementation to delete
    - Path: pkg/conversation/manager.go
      Note: Conversation manager interface slated for removal
    - Path: pkg/js/conversation-js.go
      Note: JS bridge to remove
ExternalSources: []
Summary: Plan to remove legacy conversation package and JS bridge usage
LastUpdated: 2026-01-27T00:00:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Implementation Plan

## Executive Summary

Remove the legacy `pkg/conversation` package and the JS bridge in `pkg/js/conversation-js.go`, along with any examples that rely on them. Update any references to avoid compile failures and ensure the repo builds cleanly without the conversation API.

## Problem Statement

The conversation manager and JS bridge appear unused by the current inference pipeline, which now operates on `turns.Turn` blocks. Keeping the legacy API adds maintenance overhead and confuses the active architecture.

## Proposed Solution

1. Inventory all in-repo imports of `pkg/conversation` and `pkg/js/conversation-js.go`.
2. Remove the JS bridge (`pkg/js/conversation-js.go`) and any registration hooks that import it.
3. Remove the `pkg/conversation` package (manager, tree, messages, builder, etc.).
4. Update example(s) that used the conversation manager (or remove them) to use the current Turn-based flow.
5. Run tests to confirm the repository builds.

## Design Decisions

- Prefer removal over deprecation to avoid lingering dead code.
- Update or remove examples that still use conversation to align with the Turn-based API.
- Avoid introducing compatibility shims.

## Alternatives Considered

- Deprecate and keep code in place: rejected because it continues to create drift and confusion.
- Keep only the JS bridge: rejected because it depends on conversation internals.

## Implementation Plan

1. Create tasks and diary entries for each removal step.
2. Remove JS bridge and any registration call sites.
3. Remove conversation package files and builder.
4. Update or delete conversation-based example(s).
5. Run `go test ./...` and fix any fallout.
6. Update docs/diary/changelog and close tasks.
