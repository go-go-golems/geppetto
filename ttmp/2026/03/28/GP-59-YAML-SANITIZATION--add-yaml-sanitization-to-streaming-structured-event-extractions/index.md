---
Title: Add YAML sanitization to streaming structured event extractions
Ticket: GP-59-YAML-SANITIZATION
Status: active
Topics:
    - geppetto
    - events
    - streaming
    - yaml
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/structuredsink/filtering_sink.go
      Note: FilteringSink owns tag scanning and extractor session routing
    - Path: geppetto/pkg/events/structuredsink/parsehelpers/helpers.go
      Note: parsehelpers is the proposed default-on sanitization insertion point
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: OpenAI engine emits streaming text events upstream of FilteringSink
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Responses engine emits the same event types upstream of FilteringSink
    - Path: sanitize/pkg/yaml/sanitize.go
      Note: YAML sanitizer used by the shipped parsehelpers implementation
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: Pinocchio is downstream SEM translation and not the parsing layer
ExternalSources: []
Summary: Geppetto owns the structured-sink and YAML parsing helpers; Pinocchio only translates already-emitted events to SEM. This ticket now captures both the design rationale and the shipped implementation for optional-but-default-on YAML sanitization at the Geppetto parsehelpers layer, including tests and doc updates.
LastUpdated: 2026-03-30T18:30:00-04:00
WhatFor: Give a new engineer enough context to implement default-on YAML sanitization for streaming structured event extraction without accidentally placing the change in the wrong layer.
WhenToUse: Use when implementing or reviewing YAML extraction behavior in Geppetto structured sinks, or when deciding whether a structured-streaming change belongs in Geppetto or Pinocchio.
---


# Add YAML sanitization to streaming structured event extractions

## Overview

This ticket concludes that the change belongs in `geppetto`, not `pinocchio`. Provider engines emit text streaming events, `FilteringSink` extracts tagged payloads, and extractor helpers parse YAML. Pinocchio only translates emitted Geppetto events into SEM frames for UI delivery. The implementation in this ticket adds optional-but-default-on YAML sanitization in `geppetto/pkg/events/structuredsink/parsehelpers`, backed by `github.com/go-go-golems/sanitize/pkg/yaml`.

## Key Links

- Primary design doc: `design-doc/01-intern-guide-to-adding-optional-by-default-yaml-sanitization-to-streaming-structured-event-extractions.md`
- Investigation diary: `reference/01-investigation-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

Completed in this ticket:

- Confirmed ownership boundary: Geppetto, not Pinocchio.
- Wrote a detailed intern-oriented design and implementation guide.
- Recorded the investigation diary and supporting evidence.
- Added sanitize-backed default-on parsing to `parsehelpers`.
- Added helper-focused tests for sanitize-on and sanitize-off behavior.
- Updated structured-sink docs and tutorials to the shipped helper API.

Open implementation work:

- Consider whether a future helper should expose richer sanitize metadata (for example, whether the YAML was modified).
- Optionally add an end-to-end extractor smoke test on top of the helper-focused coverage if future regressions justify it.

## Structure

- `design-doc/` contains the main implementation guide.
- `reference/` contains the diary and continuation notes.
- `tasks.md` tracks the recommended implementation sequence.
- `changelog.md` records ticket-level progress updates.
