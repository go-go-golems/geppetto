---
Title: Implementation Plan
Ticket: GP-004-JS-SETUP-TURNS
Status: active
Topics:
    - geppetto
    - javascript
    - goja
    - turns
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-12T10:00:32-05:00
WhatFor: Implement phase-1 JS module bootstrap and turn/block mapping with generated key/kind mappers.
WhenToUse: Use when building or reviewing JS API setup and turn/block conversion implementation.
---

# Implementation Plan

## Goal

Deliver phase 1 of the JS API:

- bootstrap `require("geppetto")` module surface,
- establish stable JS<->Go turn/block conversion layer,
- anchor conversion to generated mappers (`block_kind_gen.go`, `keys_gen.go`).

## Scope

In scope:

- Module bootstrap plumbing and namespace structure.
- Turn decode/encode paths for core block kinds.
- Metadata/data conversion using generated keys.
- JS-facing smoke scripts for mapper contract checks.

Out of scope:

- Running model inference.
- Middleware composition runtime behavior.
- Tool registry and toolloop orchestration.

## Work Packages

1. Module bootstrap
2. Turn decode path (JS -> Go)
3. Turn encode path (Go -> JS)
4. Generated mapper contract checks
5. Error messages and guardrails

## Deliverables

- A concrete module bootstrap entrypoint design for `geppetto` JS module.
- Detailed conversion algorithm notes and pseudocode.
- Regression checklist against generated mapper outputs.
- Runnable JS mapper contract test script in ticket `scripts/`.

## Testing Plan

- Run mapper contract script:
  - `node geppetto/ttmp/2026/02/12/GP-004-JS-SETUP-TURNS--js-api-setup-and-turn-block-mapping/scripts/test_turn_block_mapping.js`
- Validate generated files exist and include canonical constants and functions.

## Risks and Mitigations

- Risk: handwritten string mappings diverge from generated constants.
  - Mitigation: enforce generated-table checks in JS script and CI follow-up.
- Risk: unknown block kinds break decode.
  - Mitigation: explicit fallback to `BlockKindOther` and preserve payload.

## Exit Criteria

- Tasks list is complete.
- JS script passes.
- Diary records commands/results and reviewer checklist.
