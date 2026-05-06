---
Title: Audit Responses API reasoning parsing and replay schema
Ticket: GP-RESPONSES-REPLAY
Status: active
Topics:
    - responses
    - openai
    - reasoning
    - replay
    - turns
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: "Audit and implementation plan for aligning Geppetto's OpenAI Responses reasoning parsing/replay with the official schema, including reasoning_text, encrypted_content, provider item IDs, grouping, and diagnostics."
LastUpdated: 2026-05-06T15:05:00-04:00
WhatFor: "Use this ticket to implement and review Responses API reasoning item parsing and replay hardening in geppetto."
WhenToUse: "When touching pkg/steps/ai/openai_responses request construction, SSE parsing, reasoning block persistence, or turn replay."
---

# Audit Responses API reasoning parsing and replay schema

## Overview

This ticket captures the audit of Geppetto's hand-written OpenAI Responses API integration after a persisted reasoning block replayed with an invalid internal UUID as `input[].id`. The main deliverable is an intern-oriented design and implementation guide that explains the Turn model, incoming parser, replay builder, official API references, discrepancies, and a staged remediation plan.

Primary document: [design/01-responses-reasoning-parsing-replay-audit.md](./design/01-responses-reasoning-parsing-replay-audit.md)

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- responses
- openai
- reasoning
- replay
- turns

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
