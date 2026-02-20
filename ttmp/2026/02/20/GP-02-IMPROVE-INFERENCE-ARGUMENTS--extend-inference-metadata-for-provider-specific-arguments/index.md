---
Title: Extend inference metadata for provider-specific arguments
Ticket: GP-02-IMPROVE-INFERENCE-ARGUMENTS
Status: active
Topics:
    - inference
    - metadata
    - architecture
    - geppetto
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-20T11:52:25-05:00
WhatFor: Track GP-02 implementation status, key decisions, and links to supporting analysis/diary artifacts.
WhenToUse: Use this page first when onboarding or reviewing the current state of inference metadata work.
---

# Extend inference metadata for provider-specific arguments

## Overview

GP-02 extends inference metadata so provider-specific and cross-provider inference arguments can be set at engine creation time (`StepSettings.Inference`) and overridden per turn (`Turn.Data` typed keys).

Current state:
- Core inference config types/keys and provider wiring are implemented.
- Field-level merge semantics are implemented in `MergeInferenceConfig`.
- Reasoning-model sanitization and provider validation guards are in place.
- Explicit empty stop override (`Stop: []string{}`) now clears inherited chat stop sequences across OpenAI, Claude, and OpenAI Responses (commit `2e0b55e`).

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Design Analysis**: [design/01-analysis-inference-arguments.md](./design/01-analysis-inference-arguments.md)
- **Merge/Validation Analysis**: [analysis/03-rigorous-merge-and-validation-for-inferenceconfig.md](./analysis/03-rigorous-merge-and-validation-for-inferenceconfig.md)
- **Implementation Diary**: [reference/01-diary.md](./reference/01-diary.md)

## Status

Current status: **active**

## Topics

- inference
- metadata
- architecture
- geppetto

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
