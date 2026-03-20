---
Title: Extract shared inference debug printing and trace helpers into geppetto bootstrap
Ticket: GP-54-INFERENCE-DEBUG-BOOTSTRAP
Status: active
Topics:
    - architecture
    - geppetto
    - pinocchio
    - glazed
    - profiles
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: 2026-03-14--cozodb-editor/backend/main.go
      Note: Downstream consumer that had to add its own debug section redaction and hidden-base trace reconstruction
    - Path: geppetto/pkg/cli/bootstrap/config.go
      Note: Defines AppBootstrapConfig
    - Path: geppetto/pkg/cli/bootstrap/engine_settings.go
      Note: Owns hidden base section resolution and final merged engine settings
    - Path: geppetto/pkg/cli/bootstrap/profile_selection.go
      Note: Resolves visible profile selection and config file inputs
    - Path: pinocchio/cmd/pinocchio/cmds/js.go
      Note: Second Pinocchio call site that should switch directly to the Geppetto helper
    - Path: pinocchio/pkg/cmds/cmd.go
      Note: Pinocchio command call site that currently duplicates debug output handling
    - Path: pinocchio/pkg/cmds/cmdlayers/helpers.go
      Note: Current home of the generic inference debug flag section
    - Path: pinocchio/pkg/cmds/profilebootstrap/inference_settings_trace.go
      Note: Current generic trace builder living in the wrong package
ExternalSources: []
Summary: Detailed research ticket for moving a single reusable inference-debug output path from Pinocchio and downstream apps into geppetto/pkg/cli/bootstrap.
LastUpdated: 2026-03-20T15:25:00-04:00
WhatFor: Give implementers a concrete, file-anchored plan for extracting shared inference debug functionality into Geppetto without leaving wrapper re-exports behind in Pinocchio.
WhenToUse: Use when implementing or reviewing the extraction of inference debug printing and provenance tracing into Geppetto bootstrap.
---


# Extract shared inference debug printing and trace helpers into geppetto bootstrap

## Overview

This ticket documents how to move the reusable parts of inference debug output out of Pinocchio and into `geppetto/pkg/cli/bootstrap`. The goal is a clean ownership split: Geppetto owns generic bootstrap-time debug behavior, Pinocchio owns only Pinocchio-specific app bootstrap configuration, and downstream apps such as the CozoDB backend can consume the shared helper directly instead of copying logic.

The deliverable for this ticket is a detailed, intern-friendly design and implementation guide. It maps the current package boundaries, shows where behavior is duplicated today, proposes the target API shape, and outlines a phased migration plan with file-level edits. The simplified target is one `--print-inference-settings` flag that prints effective values together with their provenance, masking secrets as `***`.

## Key Links

- Primary analysis: `design-doc/01-shared-inference-debug-printing-in-geppetto-bootstrap.md`
- Investigation diary: `reference/01-diary.md`
- Task checklist: `tasks.md`
- Change log: `changelog.md`

## Current Status

Current status: **active**

Research and design are documented. The implementation has not been started in `geppetto/` yet.

## Main Conclusion

The reusable pieces are currently split across packages:

- Geppetto already owns bootstrap configuration and resolved engine settings.
- Pinocchio currently owns the inference debug flag section and the source-trace builder.
- Pinocchio command call sites and at least one downstream app each carry their own debug-output branch logic.

The clean cut is to move the generic debug section, trace building, hidden-base parsed-value reconstruction, and combined debug YAML output into `geppetto/pkg/cli/bootstrap`, then update Pinocchio and downstream callers to use that package directly.

## Structure

- `design-doc/` contains the primary architecture and implementation guide.
- `reference/` contains the chronological diary of how the design was derived.
- `tasks.md` tracks what is done in the research ticket and what remains for implementation.
- `changelog.md` records major ticket milestones.
