---
Title: 'Credentials scoped to their origin: no key reaches a host it was not issued for'
Ticket: GEPPETTO-CREDSCOPE-001
Status: active
Topics:
    - security
    - credentials
    - providers
    - geppetto
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/security/outbound_url.go
      Note: existing outbound URL policy, home of the new Origin/PinnedTransport
    - Path: repo://pkg/steps/ai/credentials/bearer.go
      Note: BearerTokenSource and Request{Provider,BaseURL}
    - Path: repo://pkg/steps/ai/providers/openaicodex/codex.go
      Note: existing origin pinning precedent
    - Path: repo://pkg/steps/ai/providers/pi/resolve.go
      Note: Pi bridge credential and endpoint resolution
    - Path: repo://ttmp/2026/11/05/GEPPETTO-CREDSCOPE-001--credentials-scoped-to-their-origin-no-key-reaches-a-host-it-was-not-issued-for/scripts/01-leak-probe/main.go
      Note: in-process probe of every credential path
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-05T21:50:27.048000827-05:00
WhatFor: ""
WhenToUse: ""
---

# Credentials scoped to their origin: no key reaches a host it was not issued for

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- security
- credentials
- providers
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
