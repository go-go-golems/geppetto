---
Title: Thinking stream events missing with gpt-5-mini
Ticket: GP-05-THINK-MODE-BUG
Status: complete
Topics:
    - bug
    - inference
    - events
    - geppetto
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/inference/engine/factory/factory.go
      Note: Core routing fix
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Core streaming fix
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/analysis/01-bug-report-missing-thinking-stream-events.md
      Note: Primary bug analysis
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/reference/01-diary.md
      Note: Detailed implementation diary
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/scripts/repro_thinking_stream_events.go
      Note: Ticket-local reproduction script
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_after_fix_gpt5mini_openai.trace.log
      Note: |-
        Real API trace after fix (openai path auto-routed to responses)
        Post-fix live trace
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_gpt5mini_openai.trace.log
      Note: Real API trace (openai path)
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_gpt5mini_openai_responses.trace.log
      Note: Real API trace (responses with summary)
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/real_api_gpt5mini_openai_responses_no_summary.trace.log
      Note: Real API trace (responses no summary)
    - Path: ttmp/2026/02/18/GP-05-THINK-MODE-BUG--thinking-stream-events-missing-with-gpt-5-mini/sources/repro_thinking_stream_events.trace.log
      Note: Trace log artifact
ExternalSources: []
Summary: Investigation and fix ticket for missing thinking stream events with gpt-5-mini; includes deterministic/real-API reproductions, implemented code changes, and post-fix validation logs.
LastUpdated: 2026-02-25T17:31:18.471686467-05:00
WhatFor: Track reproduction, diagnosis, and reporting for missing thinking-stream behavior.
WhenToUse: Use when debugging missing reasoning/thinking stream events in OpenAI model flows.
---





# Thinking stream events missing with gpt-5-mini

## Overview

This ticket investigated and fixed missing thinking stream behavior with `--ai-engine gpt-5-mini`.

Implemented outcomes:
- Reasoning models now auto-route from `openai` to `openai-responses`.
- Responses SSE parser now handles `response.reasoning_text.delta` and `.done`.
- Regression tests added and passing.
- Real API post-fix logs confirm thinking stream events now appear in the default `openai` path for `gpt-5-mini`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Bug report**: `analysis/01-bug-report-missing-thinking-stream-events.md`
- **Diary**: `reference/01-diary.md`
- **Repro scripts**: `scripts/inspect_engine_selection.go`, `scripts/repro_thinking_stream_events.go`
- **Captured outputs**: `sources/inspect_engine_selection.out`, `sources/repro_thinking_stream_events.trace.log`
- **Real API traces**: `sources/real_api_gpt5mini_openai.trace.log`, `sources/real_api_gpt5mini_openai_responses.trace.log`, `sources/real_api_gpt5mini_openai_responses_no_summary.trace.log`, `sources/real_api_after_fix_gpt5mini_openai.trace.log`

## Status

Current status: **active**

## Topics

- bug
- inference
- events
- geppetto

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
