---
Title: Geppetto JS API Feature Validation Scripts via geppetto-js-lab
Ticket: GP-17-JS-LAB-API-EXAMPLES
Status: complete
Topics:
    - geppetto
    - inference
    - tools
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/design/01-js-api-feature-validation-guide-and-geppetto-js-lab-scripts.md
      Note: Detailed feature validation guide and execution instructions
    - Path: ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/01_handles_consts_and_turns.js
      Note: Handle hiding and constants validation
    - Path: ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/02_context_hooks_and_run_options.js
      Note: Context and run options validation
    - Path: ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/03_async_surface_smoke.js
      Note: Async API surface smoke validation
    - Path: ttmp/2026/02/20/GP-17-JS-LAB-API-EXAMPLES--geppetto-js-api-feature-validation-scripts-via-geppetto-js-lab/scripts/run_all.sh
      Note: Batch runner for all validation scripts
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-25T17:31:16.653689996-05:00
WhatFor: ""
WhenToUse: ""
---



# Geppetto JS API Feature Validation Scripts via geppetto-js-lab

## Overview

Create and maintain a reproducible JS API validation pack that runs with `geppetto-js-lab`.

The ticket adds:

- a feature-by-feature validation guide document,
- ticket-local scripts that verify the newest JS API additions,
- a one-command runner script to execute all validations.

## Key Links

- [Validation Guide](./design/01-js-api-feature-validation-guide-and-geppetto-js-lab-scripts.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- inference
- tools

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
