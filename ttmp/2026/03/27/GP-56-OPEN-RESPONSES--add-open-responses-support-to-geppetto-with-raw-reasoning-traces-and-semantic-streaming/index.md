---
Title: Add Open Responses support to Geppetto with raw reasoning traces and semantic streaming
Ticket: GP-56-OPEN-RESPONSES
Status: active
Topics:
    - geppetto
    - open-responses
    - reasoning
    - streaming
    - events
    - tools
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/03/27/GP-56-OPEN-RESPONSES--add-open-responses-support-to-geppetto-with-raw-reasoning-traces-and-semantic-streaming/design-doc/01-intern-guide-to-adding-open-responses-support-and-raw-reasoning-traces-in-geppetto.md
      Note: Primary design deliverable for this ticket
    - Path: geppetto/ttmp/2026/03/27/GP-56-OPEN-RESPONSES--add-open-responses-support-to-geppetto-with-raw-reasoning-traces-and-semantic-streaming/reference/01-diary.md
      Note: Implementation diary for the research/documentation work
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-27T17:06:31.029195935-04:00
WhatFor: ""
WhenToUse: ""
---


# Add Open Responses support to Geppetto with raw reasoning traces and semantic streaming

## Overview

This ticket documents how to add provider-neutral Open Responses support to Geppetto by generalizing the existing OpenAI Responses runtime. The primary deliverable is an intern-focused design and implementation guide that explains the current architecture, identifies the gaps around raw reasoning/thinking traces, and proposes a phased implementation plan.

Current outcome:

- the ticket workspace is created,
- the main design doc is written,
- the diary is recorded,
- implementation work is intentionally left as follow-up tasks.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary design doc**: `design-doc/01-intern-guide-to-adding-open-responses-support-and-raw-reasoning-traces-in-geppetto.md`
- **Diary**: `reference/01-diary.md`

## Status

Current status: **active**

## Topics

- geppetto
- open-responses
- reasoning
- streaming
- events
- tools

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- design-doc/ - Primary implementation/design deliverables
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
