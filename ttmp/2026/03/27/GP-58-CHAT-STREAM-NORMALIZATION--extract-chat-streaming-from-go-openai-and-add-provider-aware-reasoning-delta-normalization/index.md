---
Title: Extract chat streaming from go-openai and add provider-aware reasoning delta normalization
Ticket: GP-58-CHAT-STREAM-NORMALIZATION
Status: active
Topics:
    - inference
    - streaming
    - reasoning
    - geppetto
    - chat
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: Ticket overview centers on the current chat streaming engine
    - Path: pkg/steps/ai/openai/helpers.go
      Note: Ticket overview references the existing request builder scope
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Ticket overview references the direct SSE reference implementation
ExternalSources:
    - https://github.com/openai/openai-go
    - https://docs.together.ai/docs/openai-api-compatibility
    - https://docs.together.ai/docs/deepseek-faqs
Summary: Plan and design package for removing go-openai from the OpenAI-compatible chat streaming path while keeping it for embeddings and transcription.
LastUpdated: 2026-03-27T19:07:20.662445711-04:00
WhatFor: ""
WhenToUse: ""
---


# Extract chat streaming from go-openai and add provider-aware reasoning delta normalization

## Overview

This ticket captures the design work for a narrow refactor: Geppetto should stop relying on `github.com/sashabaranov/go-openai` for chat streaming, because that typed stream boundary drops provider-specific reasoning deltas such as Together's `delta.reasoning`. The scope is intentionally limited. Embeddings and transcription continue to use `go-openai` for now.

The primary deliverable is an intern-facing design and implementation guide that explains the current architecture, the failure mode, the migration shape, the proposed package boundaries, the normalization rules, and the tests needed to land the change safely.

## Key Links

- Design doc: [design-doc/01-intern-guide-to-extracting-chat-streaming-from-go-openai-and-normalizing-provider-reasoning-deltas.md](./design-doc/01-intern-guide-to-extracting-chat-streaming-from-go-openai-and-normalizing-provider-reasoning-deltas.md)
- Diary: [reference/01-diary.md](./reference/01-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)
- Prior evidence ticket: `GP-57-TOGETHER-THINKING`

## Status

Current status: **active**

## Topics

- ai
- inference
- streaming
- openai
- together

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Current Decision

The recommended next engineering step is:

1. Replace `CreateChatCompletionStream` usage in `pkg/steps/ai/openai/engine_openai.go` with a direct HTTP + SSE reader.
2. Normalize provider delta variants at the raw stream boundary.
3. Keep `go-openai` in `pkg/embeddings/openai.go` and `pkg/steps/ai/openai/transcribe.go`.
4. Revisit a broader SDK migration separately after the chat-streaming path is covered by regression tests.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
