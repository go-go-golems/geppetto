---
Title: add provider-native token counting APIs to geppetto and expose them via pinocchio tokens count
Ticket: PI-03-TOKEN-COUNT-APIS
Status: active
Topics:
    - geppetto
    - pinocchio
    - glazed
    - inference
    - config
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Ticket hub for the provider-native token-counting design across geppetto and pinocchio."
LastUpdated: 2026-03-16T22:08:46-04:00
WhatFor: "Research and design ticket for adding provider-native token counting in geppetto and exposing it through pinocchio tokens count."
WhenToUse: "Use this ticket when implementing or reviewing OpenAI and Claude input-token counting support, or when onboarding an engineer to the current command/runtime architecture."
---

# add provider-native token counting APIs to geppetto and expose them via pinocchio tokens count

## Overview

This ticket captures the design work for a new capability, not the implementation itself. The goal is to add provider-native input-token counting to `geppetto` for the official OpenAI and Anthropic APIs, then expose that functionality in `pinocchio` by extending the existing `tokens count` command rather than creating a separate top-level command.

Canonical ticket home: this ticket was moved from `pinocchio/ttmp` to `geppetto/ttmp` on 2026-03-16 because the primary artifact is a Geppetto capability with a Pinocchio integration layer.

The core recommendation is:

- keep the current offline tokenizer path in `pinocchio` for fast local estimates,
- add a separate provider-backed count path in `geppetto`,
- make `pinocchio tokens count` select between `estimate`, `api`, and `auto` modes,
- do not widen `geppetto`'s `engine.Engine` interface just to support preflight counting.

## Key Links

- Design doc: `design-doc/01-provider-native-token-counting-for-geppetto-and-pinocchio.md`
- Diary: `reference/01-investigation-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- pinocchio
- geppetto
- glazed
- profiles
- analysis

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

## Recommended Reading Order

1. Read the design doc first for the system walkthrough and implementation plan.
2. Read the diary second for the exact repo exploration path, commands, and dead ends.
3. Use the tasks file as the execution checklist once implementation starts.
