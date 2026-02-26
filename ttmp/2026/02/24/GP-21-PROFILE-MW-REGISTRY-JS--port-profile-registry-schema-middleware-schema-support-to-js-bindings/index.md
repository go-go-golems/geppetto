---
Title: Port profile registry schema/middleware schema support to JS bindings
Ticket: GP-21-PROFILE-MW-REGISTRY-JS
Status: active
Topics:
    - profile-registry
    - js-bindings
    - go-api
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/design-doc/01-profile-registry-middleware-schema-parity-analysis-for-js-bindings.md
      Note: Primary design document
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/design-doc/02-unified-final-js-api-design-inference-first.md
      Note: |-
        Final merged inference-first API recommendation
        Final merged inference-first design decision document
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/reference/01-investigation-diary.md
      Note: Chronological investigation log
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/reference/02-geppetto-js-api-scripts-cookbook-old-and-new.md
      Note: Dedicated script cookbook covering old and new JS APIs
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_from_profile_semantics.js
      Note: fromProfile semantics experiment script
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_geppetto_exports.js
      Note: Experiment script
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_geppetto_plugins_exports.js
      Note: Experiment script
    - Path: ttmp/2026/02/24/GP-21-PROFILE-MW-REGISTRY-JS--port-profile-registry-schema-middleware-schema-support-to-js-bindings/scripts/inspect_inference_surface.js
      Note: Inference API inventory experiment script
ExternalSources: []
Summary: Research deliverable documenting Go-vs-JS parity gaps for profile registry and schema discovery, updated to GP-31 stack-first runtime semantics for JS port planning.
LastUpdated: 2026-02-25T18:05:00-05:00
WhatFor: Track GP-21 research outputs and implementation guidance.
WhenToUse: Use when planning or reviewing JS parity work for profile/schema APIs.
---




# Port profile registry schema/middleware schema support to JS bindings

## Overview

This ticket captures a deep, evidence-backed analysis of parity gaps between:

1. Go-side profile registry + schema discovery capabilities, and
2. current `require("geppetto")` JS bindings.

The main conclusion is that Go primitives exist (`pkg/profiles`, `pkg/inference/middlewarecfg`), but JS bindings currently do not expose profile registry or schema catalog operations.

Final recommendation now uses a hard cutover approach (no legacy profile-semantic compatibility path).

## Key Links

- Design doc:
  - `design-doc/01-profile-registry-middleware-schema-parity-analysis-for-js-bindings.md`
  - `design-doc/02-unified-final-js-api-design-inference-first.md`
- Investigation diary:
  - `reference/01-investigation-diary.md`
  - `reference/02-geppetto-js-api-scripts-cookbook-old-and-new.md`
- Repro scripts:
  - `scripts/inspect_geppetto_exports.js`
  - `scripts/inspect_geppetto_plugins_exports.js`
  - `scripts/inspect_inference_surface.js`
  - `scripts/inspect_from_profile_semantics.js`
- Repro outputs:
  - `various/inspect_geppetto_exports.out`
  - `various/inspect_geppetto_plugins_exports.out`
  - `various/inspect_inference_surface.out`
  - `various/inspect_from_profile_semantics.out`

## Status

Current status: **active** (research complete; implementation follow-up pending, rebased to GP-31 runtime surface).

## Tasks

See [tasks.md](./tasks.md) for completed research tasks and remaining implementation tasks.

## Changelog

See [changelog.md](./changelog.md) for timestamped updates.

## Structure

- `design-doc/` - primary technical analysis and implementation plan
- `reference/` - chronological investigation diary
- `scripts/` - reproducible experiments
- `various/` - captured experiment outputs
- `archive/` - future archival material
