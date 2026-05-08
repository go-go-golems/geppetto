---
Title: Split provider, run, and text segment event vocabulary
Ticket: GP-EVENT-VOCABULARY
Status: active
Topics:
  - geppetto
  - pinocchio
  - streaming
  - observability
  - events
DocType: index
Intent: long-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Design ticket for replacing overloaded inference/text finalization names with explicit run, provider-call, text-segment, reasoning, and tool lifecycle events plus typed correlation IDs.
LastUpdated: 2026-05-08T07:20:00-04:00
WhatFor: Track the design and implementation plan for a clean event vocabulary across Geppetto, Pinocchio, Sessionstream, and CoinVault debug exports.
WhenToUse: Use when changing event names, provider stream adapters, Pinocchio chatapp protobufs, or SQLite correlation exports.
---

# Split provider, run, and text segment event vocabulary

## Overview

This ticket designs a clean replacement vocabulary for the overloaded `EventFinal` / `ChatInferenceFinished` path. The goal is to separate:

- chat run lifecycle;
- provider-call lifecycle;
- text segment lifecycle;
- reasoning segment lifecycle;
- tool lifecycle.

The design also specifies typed correlation IDs for every new event so consumers can join provider events, Geppetto events, Pinocchio backend events, Sessionstream frames, frontend records, and SQLite rows without heuristics over generic metadata maps.

## Key documents

- [Provider run and text segment event vocabulary design guide](./design-doc/01-provider-run-and-text-segment-event-vocabulary-design-guide.md)
- [Gemini canonical event migration analysis](./analysis/01-gemini-canonical-event-migration-analysis.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Evidence

Line-numbered source excerpts are stored in [`sources/`](./sources/) for:

- Geppetto events and observability;
- Claude, OpenAI Chat Completions, OpenAI Responses, and Gemini provider mappings;
- Pinocchio runtime sink, runtime inference, protobuf messages, plugins, and SQLite reconcile code.

## Status

Current status: **active implementation; Gemini/tool-executor migration added before legacy event deletion**.

## Structure

- `design-doc/` — architecture and implementation guide.
- `analysis/` — focused migration analyses such as the Gemini provider cutover.
- `reference/` — diary and supporting references.
- `sources/` — line-numbered source evidence.
- `scripts/` — future temporary code and tooling.
- `various/` — working notes and research.
