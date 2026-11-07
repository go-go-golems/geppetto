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
    - Path: repo://pkg/steps/ai/credscope/credscope_test.go
      Note: the regression matrix (invariant test)
    - Path: repo://pkg/steps/ai/providers/openaicodex/codex.go
      Note: existing origin pinning precedent
    - Path: repo://pkg/steps/ai/providers/pi/endpoint.go
      Note: provider-keyed endpoint resolution
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

The original implementation pins credential-bearing HTTP clients to their configured origins. Design 02 completes the runtime pattern: one opaque connection owns the endpoint, routes, authentication, headers, URL policy and client for every supported AI API family. Credential overrides remain atomic through embedding settings and profile stacks.

Implementation is committed on `task/geppetto-bound-connections-001` in the writable sibling checkout `geppetto-bound-connections`. The 64-case regression matrix and recorded validation gate pass. See design 02 for API changes and the precise validation limits.

## Key Links

- [Design 01: original analysis, scoping and pinning](design-doc/01-intern-guide-credential-scoping-in-geppetto-analysis-design-and-implementation.md)
- [Design 02: bound API connections and implementation results](design-doc/02-bound-connections-atomic-credential-ownership-and-request-execution.md)
- [Detailed implementation diary](reference/01-diary.md), steps 7–13 for this continuation
- [Validation evidence](sources/05-bound-connection-validation.txt) and [lint evidence](sources/06-bound-connection-lint.txt)

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
