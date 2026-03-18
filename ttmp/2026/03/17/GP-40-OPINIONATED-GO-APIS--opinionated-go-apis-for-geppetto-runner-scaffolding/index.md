---
Title: Opinionated Go APIs for Geppetto Runner Scaffolding
Ticket: GP-40-OPINIONATED-GO-APIS
Status: complete
Topics:
    - geppetto
    - pinocchio
    - go-api
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Manuel-specific GP-40 workspace covering the current Geppetto runtime stack, downstream usage, and a proposed opinionated runner API, updated after the hard cuts that moved runtime resolution and policy out of Geppetto core."
LastUpdated: 2026-03-18T03:12:00-04:00
WhatFor: "Detailed architecture and API design work for a new opinionated Geppetto runner API, kept separate from a colleague's parallel GP-40 workspace and updated to reflect app-owned resolved runtime preparation rather than Geppetto-owned profile/runtime patching."
WhenToUse: "Use this ticket when reviewing how Geppetto, Pinocchio, and downstream applications currently assemble inference loops and when planning a simpler public Go API with app-owned resolved runtime inputs and registry filtering."
---

# Opinionated Go APIs for Geppetto Runner Scaffolding

## Overview

This workspace captures Manuel's GP-40 analysis for an opinionated Go runner API on top of Geppetto. The goal is to make common CLI and chat-style tool-loop programs easy to scaffold without forcing every application to hand-wire `session.Session`, `enginebuilder.Builder`, tool registries, middleware chains, event sinks, snapshot hooks, and persistence.

The analysis is evidence-backed and focuses on `geppetto/` first, then on how the same wiring is repeated in `pinocchio/`, `2026-03-14--cozodb-editor`, `2026-03-16--gec-rag`, and `temporal-relationships`.

Important collaboration note: there is already a separate GP-40 ticket directory owned by another session. This workspace intentionally keeps a separate Manuel-specific diary and does not overwrite the colleague's materials.

## Key Links

- Design doc:
  `design-doc/01-opinionated-geppetto-runner-design-and-implementation-guide.md`
- Concrete implementation plan:
  `design-doc/02-concrete-runner-implementation-plan.md`
- Diary:
  `reference/01-manuel-investigation-diary.md`
- API sketch:
  `scripts/01-opinionated-runner-api-sketch.go`

## Status

Current status: **complete**

## Topics

- geppetto
- pinocchio
- go-api
- architecture

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

## Outcome Summary

- Documented the current Geppetto runtime stack from `session.Session` down to `toolloop.Loop` and the tools/middleware subsystems.
- Mapped how Pinocchio and downstream apps repeat similar composition logic with small variations.
- Proposed a recommended opinionated runner layer that preserves Geppetto's low-level primitives while making common CLI and chat flows substantially easier to scaffold.
- Updated the recommendation after GP-41, GP-42, GP-43, and GP-45 so the runner consumes app-owned resolved runtime inputs instead of Geppetto-owned profile patches, override paths, or compatibility policy knobs.
- Added a concrete package-level implementation plan for `pkg/inference/runner`, including phased tasks, test slices, example migration targets, and commit sequencing for implementation work.
