---
Title: reintroduce engine profiles and separate them from app runtime configuration
Ticket: GP-49-ENGINE-PROFILES
Status: active
Topics:
    - geppetto
    - architecture
    - inference
    - profile-registry
    - config
    - javascript
    - pinocchio
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/js/modules/geppetto/api_runner.go
      Note: JS runner path that currently consumes resolved runtime metadata
    - Path: pkg/js/modules/geppetto/api_runtime_metadata.go
      Note: JS materialization of runtime metadata into middleware tool filtering and turn stamping
    - Path: pkg/engineprofiles/service.go
      Note: Renamed package surface still carrying the current effective profile resolution and runtime fingerprint path
    - Path: pkg/engineprofiles/stack_merge.go
      Note: Renamed package surface still carrying runtime merge semantics for prompt middleware and tools
    - Path: pkg/engineprofiles/types.go
      Note: Renamed package surface still carrying the current mixed profile data model centered on RuntimeSpec
    - Path: pkg/steps/ai/settings/settings-step.go
      Note: Current StepSettings definition and constructors targeted for hard rename to InferenceSettings
ExternalSources: []
Summary: Design ticket for reintroducing engine-only profiles in Geppetto, renaming StepSettings to InferenceSettings, and moving runtime behavior fully to application code.
LastUpdated: 2026-03-18T19:05:00-04:00
WhatFor: Use this ticket when redesigning Geppetto profiles so they configure engines only, while application runtimes own prompts, middlewares, tools, and runtime identity.
WhenToUse: Use when planning the hard cut from mixed runtime profiles to dedicated engine profiles, renaming StepSettings to InferenceSettings, or defining the migration playbook that downstream apps must follow.
---


# reintroduce engine profiles and separate them from app runtime configuration

## Overview

This ticket proposes a second-generation profile model for Geppetto.

The current post-GP-43 state removed `runtime.step_settings_patch`, which simplified a real architectural problem, but it also exposed a new one: Geppetto no longer has a first-class, reusable abstraction for engine configuration, while application-facing runtime behavior is still mixed into the Geppetto profile model through `RuntimeSpec`.

The proposal in this ticket is:

- reintroduce Geppetto-owned profiles, but only for engine configuration
- rename `StepSettings` to `InferenceSettings`
- replace mixed `Profile + RuntimeSpec` documents with dedicated `EngineProfile` documents
- move prompt, middleware, tool selection, runtime keys, and runtime fingerprints completely out of Geppetto core and into application-owned runtime resolvers
- perform a hard cut with no backwards compatibility wrappers
- publish a migration playbook in Glazed docs so downstream applications can update systematically

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

Research and design are complete in this ticket, and implementation has started. Slice 1 is now done:

- `pkg/profiles` hard-renamed to `pkg/engineprofiles`
- imports updated across Geppetto, Pinocchio, GEC-RAG, and Temporal Relationships
- behavior intentionally kept unchanged in this slice

The ticket also includes a concrete downstream migration playbook in Glazed docs:

- [migrating-from-mixed-runtime-profiles-to-engine-profiles.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/glazed/pkg/doc/tutorials/migrating-from-mixed-runtime-profiles-to-engine-profiles.md)

No code implementation has started yet.

## Topics

- geppetto
- profiles
- engine
- configuration
- javascript
- pinocchio

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
