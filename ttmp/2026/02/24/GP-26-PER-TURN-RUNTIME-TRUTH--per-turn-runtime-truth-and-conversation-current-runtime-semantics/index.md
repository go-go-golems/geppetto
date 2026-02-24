---
Title: Per-turn Runtime Truth and Conversation Current Runtime Semantics
Ticket: GP-26-PER-TURN-RUNTIME-TRUTH
Status: active
Topics:
    - architecture
    - backend
    - chat
    - persistence
    - pinocchio
    - migration
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-24T15:34:17.578257518-05:00
WhatFor: Define and implement the data-model correction where runtime is authoritative per turn, while conversation stores only the current runtime pointer.
WhenToUse: Use when implementing or reviewing runtime persistence, debug APIs, and profile-switch behavior for multi-runtime conversations.
---

# Per-turn Runtime Truth and Conversation Current Runtime Semantics

## Overview

This ticket corrects runtime data semantics for chat persistence.

If a conversation can switch profiles/runtimes mid-stream, then runtime can no longer be modeled as a stable conversation-level truth. The correct model is:

- per-turn runtime is authoritative historical truth,
- conversation runtime is a denormalized "current runtime" pointer for list and bootstrap scenarios.

This ticket delivers schema changes, persistence wiring, API shape updates, migration, and tests to make that model explicit and enforced.

## Key Links

- Design: [Implementation Plan - Per-turn Runtime Truth and Conversation Current Runtime](./design-doc/01-implementation-plan-per-turn-runtime-truth-and-conversation-current-runtime.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- backend
- chat
- persistence
- pinocchio
- migration

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
