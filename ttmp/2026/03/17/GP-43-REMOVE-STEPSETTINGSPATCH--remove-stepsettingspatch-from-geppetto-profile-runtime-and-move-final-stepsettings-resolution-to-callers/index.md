---
Title: Remove StepSettingsPatch from Geppetto profile runtime and move final StepSettings resolution to callers
Ticket: GP-43-REMOVE-STEPSETTINGSPATCH
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - config
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design and implementation ticket for deleting RuntimeSpec.StepSettingsPatch, removing EffectiveStepSettings/BaseStepSettings and RuntimeKeyFallback from Geppetto profile resolution, and pushing final runtime resolution to callers."
LastUpdated: 2026-03-17T23:35:00-04:00
WhatFor: "Use this ticket to understand why StepSettingsPatch and RuntimeKeyFallback should be deleted, what surfaces depend on them today, and how to migrate Geppetto, Pinocchio, GEC-RAG, and Temporal Relationships to caller-owned final runtime resolution."
WhenToUse: "Use when implementing or reviewing removal of StepSettingsPatch, EffectiveStepSettings, BaseStepSettings, and RuntimeKeyFallback from the profile-resolution path."
---

# Remove StepSettingsPatch from Geppetto profile runtime and move final StepSettings resolution to callers

## Overview

This ticket captures the architectural cleanup needed to remove `RuntimeSpec.StepSettingsPatch` from Geppetto entirely. The main design goal is to stop treating profile resolution as a partial engine-configuration pipeline. Instead, callers should resolve and cache final `*settings.StepSettings` on their own side, then hand Geppetto already-concrete engine configuration.

Today, Geppetto profile resolution still merges `runtime.step_settings_patch`, applies it to `BaseStepSettings`, returns `EffectiveStepSettings`, and exposes that merged shape to downstream apps. That makes the profile subsystem responsible for an engine-configuration concern that is better owned by the application layer.

The target end state is:

- Geppetto profiles resolve profile metadata, system prompt, middleware uses, and tool names only.
- Callers resolve final `*settings.StepSettings` before or alongside profile resolution.
- Callers own runtime identity (`runtime key`, `fingerprint`) as well as final engine settings.
- Engine creation uses final settings directly.
- `StepSettingsPatch`, `BaseStepSettings`, `EffectiveStepSettings`, `RuntimeKeyFallback`, and the patch merge/apply helpers are removed from the main Geppetto profile API.

## Key Links

- Primary guide: `design-doc/01-remove-stepsettingspatch-and-move-final-stepsettings-resolution-to-callers-design-and-implementation-guide.md`
- Ideal API + hard-cut plan: `design-doc/02-ideal-app-facing-api-and-hard-cut-implementation-plan.md`
- Diary: `reference/01-manuel-investigation-diary.md`
- Inventory script: `scripts/01-stepsettingspatch-surface-inventory.sh`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

The hard cut is implemented across Geppetto, Pinocchio, GEC-RAG, and Temporal Relationships. Remaining ticket work is bookkeeping validation plus reMarkable refresh.

## Topics

- geppetto
- profile-registry
- architecture
- settings

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

## Scope

In scope:

- Remove `RuntimeSpec.StepSettingsPatch`.
- Remove `ResolveInput.BaseStepSettings`.
- Remove `ResolveInput.RuntimeKeyFallback`.
- Remove `ResolvedProfile.EffectiveStepSettings`.
- Remove `ApplyRuntimeStepSettingsPatch` and `MergeRuntimeStepSettingsPatches` from Geppetto profile runtime flow.
- Migrate downstream callers to caller-owned final runtime resolution (`StepSettings` plus runtime identity).
- Update examples, docs, tests, and migration tooling that still mention `step_settings_patch`.

Out of scope:

- Redesigning provider-specific `StepSettings` themselves.
- Removing profile-based `SystemPrompt`, `Middlewares`, or `Tools`.
- Redesigning the entire profile registry format beyond this cleanup.
