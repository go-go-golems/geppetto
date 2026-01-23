---
Title: Step controller integration for webchat
Ticket: GP-07-STEP-CONTROLLER-INTEGRATION
Status: active
Topics:
    - geppetto
    - backend
    - conversation
    - events
    - websocket
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/01/21/MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/reference/01-diary.md
      Note: Background on the refactored session execution model that StepController integration should target
ExternalSources: []
Summary: Research how the Moments backend "StepController" pattern works and how to integrate equivalent stepping into the current Geppetto `session.Session` abstraction (as used by Pinocchio webchat) post-MO-007.
LastUpdated: 2026-01-22T17:47:19.026778905-05:00
WhatFor: Capture the intent, constraints, and design options for adding step-by-step execution control ("stepping") to webchat, leveraging prior StepController work in moments/backend and the recent session execution refactor in Geppetto.
WhenToUse: Use when implementing stepping (pause/resume/continue) for tool-loop execution in Pinocchio webchat and when evolving Geppetto’s session execution API (breaking changes allowed).
---


# Step controller integration for webchat

## Overview

We want to evaluate the existing **StepController** approach in `moments/backend` (how it coordinates streaming, tool loops, and “advance one step” semantics) and determine how to backport the *useful behavior* into **Geppetto/Pinocchio’s webchat** now that session handling/execution was refactored (MO-007).

Constraint update (per Jan 22, 2026): **we explicitly want to integrate stepping into the core session abstraction** (no backwards-compatibility requirement for APIs).

This ticket is *research/design-first*: document what Moments does, map it to the refactored Geppetto session execution model, and propose integration options + next steps.

## Key Links

- MO-007 session execution refactor diary: `../MO-007-SESSION-REFACTOR--session-execution-refactor-unify-sinks-cancellation-tool-loop/reference/01-diary.md`
- Analysis: `analysis/01-stepcontroller-deep-dive-session-integration.md`
- Diary: `reference/01-diary.md`
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Intent / Questions

- What is the **contract** of StepController in Moments (inputs/outputs, lifecycle, ownership)?
- What “step” means there (token chunk, assistant message, tool call boundary, turn boundary, something else)?
- Where does it sit (session vs executor vs websocket handler), and what does it *own* (buffering, scheduling, cancellation, backpressure)?
- How does it interact with **tool loops** (tool selection, tool execution, tool results, continuation)?
- How does it interact with **streaming sinks/events** (buffer until step, stream but gate commits, send markers)?
- What parts are essential for UX (e.g., “Run”, “Step”, “Pause”, “Stop”, “Continue after tool”) vs incidental?

## Working hypothesis (to validate)

Backporting “stepping” cleanly likely means:
- modeling stepping as an **execution-level** concept (active inference) but exposing it via the **session** so webchat can control it
- inserting pause points at **tool-loop boundaries** (after inference when tools are pending; after tools) and emitting explicit pause events

## Non-goals

- Do not preserve legacy APIs or compatibility shims; prefer a clean, breaking-change integration.
- Do not fully implement UI/UX here; only capture requirements and integration points.

## Status

Current status: **active**

## Topics

- geppetto
- backend
- conversation
- events
- websocket
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
