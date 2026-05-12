---
Title: Add Model Metadata to Engine Profiles
Ticket: GP-70
Status: active
Topics:
    - geppetto
    - engine-profiles
    - inference
    - model-metadata
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/01-profiles.md
      Note: Permanent Geppetto engine profile docs now document inference_settings.model_info
    - Path: geppetto/pkg/doc/topics/13-js-api-reference.md
      Note: JS API docs now document resolved.modelInfo and engine.modelInfo
    - Path: geppetto/pkg/doc/types/geppetto.d.ts
      Note: Generated TypeScript declarations for ModelInfo/ModelCost JS API
    - Path: geppetto/pkg/engineprofiles/inference_settings_merge.go
      Note: ModelInfo stack merge and cost replacement semantics
    - Path: geppetto/pkg/engineprofiles/stack_merge.go
      Note: Stack merge for InferenceSettings where ModelInfo will merge
    - Path: geppetto/pkg/engineprofiles/types.go
      Note: Core EngineProfile types where ModelInfo will be consumed
    - Path: geppetto/pkg/inference/engine/factory/factory.go
      Note: isReasoningModel heuristic to replace with ModelInfo.Reasoning
    - Path: geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go
      Note: JS resolved profile modelInfo exposure
    - Path: geppetto/pkg/steps/ai/settings/model_info.go
      Note: New typed model metadata
    - Path: geppetto/pkg/steps/ai/settings/settings-inference.go
      Note: InferenceSettings where ModelInfo field will be added
    - Path: geppetto/pkg/turns/inference_result.go
      Note: InferenceResult where Cost field will be added
    - Path: pinocchio/README.md
      Note: Pinocchio engine-profile YAML example now includes model_info
    - Path: pinocchio/cmd/web-chat/README.md
      Note: Web-chat docs now mention model_info in profile API responses
    - Path: pinocchio/cmd/web-chat/profiles/api.go
      Note: Web-chat profile API model_info exposure
    - Path: pinocchio/pkg/ui/profileswitch/picker.go
      Note: Profile picker model capability summary rendering
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-06T10:23:28.668065926-04:00
WhatFor: ""
WhenToUse: ""
---
















# Add Model Metadata to Engine Profiles

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- geppetto
- engine-profiles
- inference
- model-metadata

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
