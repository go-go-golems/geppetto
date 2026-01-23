---
Title: 'Turn creation: cloning, IDs, and block propagation'
Ticket: GP-05-TURN-CREATION
Status: complete
Topics:
    - geppetto
    - turns
    - inference
    - design
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T20:53:21.124259353-05:00
WhatFor: ""
WhenToUse: ""
---


# Turn creation: cloning, IDs, and block propagation

## Overview

You want a deep, code-accurate analysis of **how a “follow-up” turn is created for subsequent
inference executions within the same session** in Geppetto, specifically:

- Where and why `*turns.Turn` is **cloned** vs **mutated in-place** vs **rebuilt/transformed**.
- When a **new `Turn.ID` should be generated** (or not), and what the current code actually does.
- How turn-/block-level **attribution metadata** should work (block metadata, event metadata), now
  that `Block.TurnID` has been removed.

The core correctness requirement is:

- Each inference should have stable correlation identifiers (`SessionID`, `InferenceID`, and
  `Turn.ID` as used by event metadata), and
- blocks/events should be attributable to the inference execution that produced them (preferably via
  metadata keys, not dedicated struct fields).

The output should be an analysis document that maps all relevant paths (session, tool loop runner,
engines, helpers, serde) and identifies:

- the current behavior
- the intended behavior
- any gaps (e.g. “turn id stays constant across multiple inferences” or “blocks keep old turn id”)
- concrete fixes/proposals (but only after the analysis is crisp)

Questions to sanity-check my understanding:

1. “Subsequent turns” means subsequent inference executions within the same SessionID.
2. Reused blocks should keep whatever attribution metadata they already carry.

Additional analysis request:

- If blocks are created as part of a specific inference execution, capture whether (and where) block
  metadata stores an `InferenceID` today. If it does not, include a proposal for how/where to attach
  it so blocks can be attributed to the inference that produced them.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- turns
- inference
- design

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
