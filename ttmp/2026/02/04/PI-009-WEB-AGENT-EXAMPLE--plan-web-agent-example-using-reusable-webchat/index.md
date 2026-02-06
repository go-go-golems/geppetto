---
Title: Plan web agent example using reusable webchat
Ticket: PI-009-WEB-AGENT-EXAMPLE
Status: active
Topics:
    - webchat
    - frontend
    - backend
    - agent
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web-agent-example/pkg/discodialogue/sem_test.go
      Note: Boundary encode/decode test for disco dialogue SEM payloads
    - Path: web-agent-example/pkg/thinkingmode/sem_test.go
      Note: Boundary encode/decode test for thinking mode SEM payloads
    - Path: web-agent-example/web/src/sem/registerWebAgentSem.ts
      Note: Custom TS SEM handler decoding generated protobuf payloads
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-04T16:16:32.453860567-05:00
WhatFor: ""
WhenToUse: ""
---


# Plan web agent example using reusable webchat

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- webchat
- frontend
- backend
- agent

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
