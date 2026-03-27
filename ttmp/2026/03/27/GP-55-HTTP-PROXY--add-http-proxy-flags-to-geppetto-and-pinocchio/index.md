---
Title: Add HTTP proxy flags to Geppetto and Pinocchio
Ticket: GP-55-HTTP-PROXY
Status: active
Topics:
    - geppetto
    - pinocchio
    - glazed
    - config
    - inference
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/steps/ai/claude/engine_claude.go
      Note: Claude engine now injects the shared HTTP client into the API client.
    - Path: geppetto/pkg/steps/ai/gemini/engine_gemini.go
      Note: Gemini engine now routes proxy-aware/custom HTTP clients through SDK-compatible transport options.
    - Path: geppetto/pkg/steps/ai/openai/helpers.go
      Note: |-
        Representative provider seam that shows parsing is not the main gap; HTTP client construction is
        OpenAI client-construction seam now wired to the shared HTTP client helper.
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Responses engine now uses the shared HTTP client in both streaming and non-streaming paths.
    - Path: geppetto/pkg/steps/ai/settings/flags/client.yaml
      Note: Shared Glazed ai-client flag surface where proxy options should be added
    - Path: geppetto/pkg/steps/ai/settings/http_client.go
      Note: Shared proxy-aware HTTP client helper used by provider engines.
    - Path: geppetto/pkg/steps/ai/settings/settings-client.go
      Note: Shared transport settings object that should own explicit proxy configuration
ExternalSources: []
Summary: Ticket workspace for the explicit HTTP proxy support work in Geppetto and Pinocchio, including section ownership, runtime wiring analysis, intern-facing guides, the Pinocchio base-settings work, and the Geppetto provider transport implementation.
LastUpdated: 2026-03-27T11:05:00-04:00
WhatFor: Track the analysis, documentation, and implementation work that added first-class proxy flags through Geppetto's shared settings system and carried them into Pinocchio runtime flows.
WhenToUse: Use when reviewing or continuing proxy-support work, especially to understand where the new flags belong, how Pinocchio base settings are built, how provider engines now consume the shared HTTP client, and what follow-up consistency work may still remain.
---



# Add HTTP proxy flags to Geppetto and Pinocchio

## Overview

This ticket started as the design work for explicit HTTP proxy support in Geppetto-backed Pinocchio commands. The core design outcome is a detailed guide that identifies `ai-client` as the correct shared Glazed section, traces how Pinocchio inherits that section, and pinpoints the provider engine seams that must use a proxy-aware `*http.Client`.

The ticket now also includes the two main implementation steps that followed the design work: the Pinocchio-side parsed-base helper plus `web-chat` CLI/base-settings wiring, and the Geppetto-side shared HTTP-client helper plus provider transport wiring across OpenAI, Claude, OpenAI Responses, and Gemini.

The ticket also includes a second deep-dive document that explains the base-settings/profile-overlay/runtime-switch lifecycle in detail for onboarding and debugging. At this point the main proxy path is implemented; remaining follow-up work is mostly consistency and scope expansion, such as normalizing adjacent non-engine HTTP paths if desired.

## Key Links

- Design guide: [design-doc/01-http-proxy-design-and-implementation-guide-for-geppetto-and-pinocchio.md](./design-doc/01-http-proxy-design-and-implementation-guide-for-geppetto-and-pinocchio.md)
- Base/profile lifecycle guide: [design-doc/02-base-settings-profile-overlays-and-runtime-profile-switching-in-pinocchio.md](./design-doc/02-base-settings-profile-overlays-and-runtime-profile-switching-in-pinocchio.md)
- ai-client exposure analysis: [design-doc/03-analysis-of-ai-client-cli-exposure-in-pinocchio-and-web-chat.md](./design-doc/03-analysis-of-ai-client-cli-exposure-in-pinocchio-and-web-chat.md)
- Diary: [reference/01-diary.md](./reference/01-diary.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- pinocchio
- glazed
- config
- inference
- documentation

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
