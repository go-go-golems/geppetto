---
Title: Instrument OpenAI chat completions observability
Ticket: GEPPETTO-OPENAI-OBS-2026-05-07
Status: active
Topics:
    - observability
    - openai
    - chat
    - inference
    - intern-onboarding
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: pkg/events/context.go
      Note: Context EventSink publish mechanism wrapped by observability
    - Path: pkg/inference/engine/factory/factory.go
      Note: Factory option plumbing for provider engines
    - Path: pkg/inference/engine/factory/factory_observability_test.go
      Note: Factory OpenAI option plumbing regression test
    - Path: pkg/observability/config.go
      Note: Trace level policy for off/events/provider
    - Path: pkg/observability/observer.go
      Note: Neutral Record shape and panic-safe Notify delivery
    - Path: pkg/steps/ai/openai/chat_stream.go
      Note: Streaming SSE decoder; chatStreamEvent.RawPayload source for provider records
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: OpenAI Chat Completions RunInference loop and publishEvent wrapper target
    - Path: pkg/steps/ai/openai/observability.go
      Note: OpenAI Chat Completions observability options and record helpers
    - Path: pkg/steps/ai/openai/observability_test.go
      Note: OpenAI Chat Completions observability trace-level and panic-safety tests
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Responses constructor and publishEvent wrapper pattern
    - Path: pkg/steps/ai/openai_responses/observability.go
      Note: Existing observability implementation used as template
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-07T14:23:57.755401723-04:00
WhatFor: ""
WhenToUse: ""
---



# Instrument OpenAI chat completions observability

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- observability
- openai
- chat
- inference
- intern-onboarding

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
