---
Title: Investigate missing Together Qwen thinking stream in OpenAI-compatible chat completions
Ticket: GP-57-TOGETHER-THINKING
Status: active
Topics:
    - geppetto
    - together
    - reasoning
    - streaming
    - openai-compatibility
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Investigation ticket for the missing Together Qwen thinking stream, now including a detailed postmortem, experiment artifacts, and the implemented fix for the missing stream=true runtime regression in Geppetto.
LastUpdated: 2026-03-28T16:16:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Investigate missing Together Qwen thinking stream in OpenAI-compatible chat completions

## Overview

This ticket investigates why the `together-qwen-3.5-9b` profile did not expose visible thinking deltas through the Geppetto and Pinocchio stack. The current ticket contents now cover the full lifecycle of the bug: initial reproduction, raw-provider controls, SDK comparison, the Geppetto-side runtime fix, and a postmortem that explains the system for a new intern.

## Key Links

- [Initial intern guide](./design-doc/01-intern-guide-to-investigating-and-fixing-together-qwen-thinking-stream-support-in-geppetto.md)
- [Postmortem and intern guide](./design-doc/02-postmortem-and-intern-guide-to-the-together-qwen-thinking-stream-bug.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Task list](./tasks.md)
- [Changelog](./changelog.md)
- **Related Files**: See frontmatter `RelatedFiles`
- **External Sources**: See frontmatter `ExternalSources`

## Status

Current status: **active**

## Topics

- geppetto
- together
- reasoning
- streaming
- openai-compatibility

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
