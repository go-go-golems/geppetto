---
Title: Implementation Plan
Ticket: GP-007-JS-BUILDER-TOOLS
Status: active
Topics:
    - geppetto
    - javascript
    - goja
    - tools
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-12T10:00:32-05:00
WhatFor: Implement phase-4 JS builder and tool orchestration APIs, including JS/Go tool interoperability.
WhenToUse: Use when implementing or reviewing builder, registry, and toolloop integration from JS.
---

# Implementation Plan

## Goal

Deliver phase 4 builder + tools integration:

- builder-style API from JS for assembling engines/middlewares/tools,
- JS tool registration,
- toolloop configuration from JS,
- invocation of Go-registered tools and JS-registered tools through one registry contract.

Note: wording/detail for phase 4 may be refined later, but this plan captures implementation-ready baseline tasks.

## Scope

In scope:

- Builder object model and defaults.
- Tool registry bridge (`JS + Go` origin support).
- Toolloop configuration mapping from JS options.
- JS smoke script for tool-enabled example execution.

Out of scope:

- Provider-specific advanced server tool features.
- Long-horizon plugin marketplace concerns.

## Work Packages

1. Builder contract and options normalization
2. Tool registry bridge and collision policy
3. JS tool registration and schema validation
4. Direct tool invocation API (`reg.call`)
5. Toolloop config wiring and smoke tests

## Deliverables

- Builder and tools API contract draft.
- Detailed collision/allowlist policy.
- Runnable JS script that validates builder/toolloop example command.

## Testing Plan

- Run builder+tools smoke script:
  - `node geppetto/ttmp/2026/02/12/GP-007-JS-BUILDER-TOOLS--js-api-builder-and-tools-orchestration/scripts/test_builder_tools_smoke.js`
- Inference/toolloop tests use:
  - `PINOCCHIO_PROFILE=gemini-2.5-flash-lite`

## Risks and Mitigations

- Risk: duplicate tool names across JS and Go registries.
  - Mitigation: deny by default, explicit override policy.
- Risk: toolloop behavior diverges based on tool origin.
  - Mitigation: normalize into shared `tools.ToolDefinition` and shared executor path.

## Exit Criteria

- Detailed tasks completed.
- Tool smoke script executed and logged in diary.
- Bridge policy and toolloop mapping clearly documented.
