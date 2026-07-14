---
Title: Host-owned renewable bearer sources for JavaScript engines
Ticket: GEP-JS-RENEWABLE-BEARER-INJECTION
Status: complete
Topics:
    - javascript
    - oauth
    - inference
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-13T20:30:09.5896359-04:00
WhatFor: ""
WhenToUse: ""
---


# Host-owned renewable bearer sources for JavaScript engines

## Overview

This ticket adds a registration-level, Go-host-only `BearerTokenSource` to the Geppetto JavaScript native module. JavaScript can continue to resolve profile settings and build OpenAI-compatible engines, while the host retains OAuth refresh, persistence, and bearer values outside JavaScript and `InferenceSettings.APIKeys`.

The implementation is complete: source wiring, source/no-source behavioral tests, public guidance, and ticket delivery records are in place. The design document is the onboarding entrypoint for future changes, especially multi-identity requirements.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **implemented and validated**

## Topics

- javascript
- oauth
- inference

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/01-host-owned-renewable-bearer-source-injection-for-javascript-engines.md` — intern-facing architecture, API, pseudocode, diagrams, and review plan.
- `reference/01-investigation-diary.md` — chronological implementation and validation evidence.
