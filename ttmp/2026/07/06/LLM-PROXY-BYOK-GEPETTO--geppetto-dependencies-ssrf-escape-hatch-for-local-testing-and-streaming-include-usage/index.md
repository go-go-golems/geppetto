---
Title: 'Geppetto dependencies: SSRF escape hatch for local testing and streaming include_usage'
Ticket: LLM-PROXY-BYOK-GEPETTO
Status: active
Topics:
    - byok
    - geppetto
    - llm-proxy
    - security
    - inference
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/security/outbound_url.go
      Note: ValidateOutboundURL + OutboundURLOptions — the SSRF guard primitive
    - Path: pkg/steps/ai/claude/api/completion.go
      Note: Claude API client carries outbound URL options
    - Path: pkg/steps/ai/openai/chat_stream.go
      Note: OpenAI-compatible Chat URL validation uses profile-owned outbound options
    - Path: pkg/steps/ai/openai_responses/provider_settings.go
      Note: Responses alias-aware outbound option resolution
    - Path: pkg/steps/ai/settings/outbound_url.go
      Note: New helper mapping profile API settings to security.OutboundURLOptions
    - Path: pkg/steps/ai/settings/settings-inference.go
      Note: APISettings exposes allow_http and allow_local_networks profile maps
ExternalSources: []
Summary: 'Geppetto-side dependencies for BYOK: expose a secure local-provider testing escape hatch around outbound URL validation and implement on-wire streaming usage support via stream_options.include_usage.'
LastUpdated: 2026-07-06T11:45:00-04:00
WhatFor: Track geppetto changes that unblock local fake-provider smoke tests and streaming usage visibility for llm-proxy BYOK.
WhenToUse: Read before implementing the Geppetto SSRF escape hatch or llm-proxy include_usage plumbing.
---











# Geppetto dependencies: SSRF escape hatch for local testing and streaming include_usage

## Overview

This ticket was moved from the llm-proxy docmgr workspace into the Geppetto repo because the first implementation step belongs in Geppetto: provider URL validation already supports `OutboundURLOptions`, but LLM provider call sites hard-code `AllowHTTP: false` and expose no profile-level test escape hatch.

The second workstream, `stream_options.include_usage`, is cross-repo: Geppetto must continue returning authoritative `result.Usage`; llm-proxy must serialize it into a final SSE frame when requested.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- byok
- geppetto
- llm-proxy
- security
- inference

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
