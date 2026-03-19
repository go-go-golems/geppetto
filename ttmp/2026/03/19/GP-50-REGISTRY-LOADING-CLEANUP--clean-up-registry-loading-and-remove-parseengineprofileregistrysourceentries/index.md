---
Title: Clean up registry loading and remove ParseEngineProfileRegistrySourceEntries
Ticket: GP-50-REGISTRY-LOADING-CLEANUP
Status: active
Topics:
    - profiles
    - glazed
    - cleanup
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/examples/internal/runnerexample/inference_settings.go
      Note: Example helper currently straddling string vs slice registry inputs
    - Path: pkg/engineprofiles/source_chain.go
      Note: Registry source spec parsing boundary and helper removal target
    - Path: pkg/js/modules/geppetto/api_profiles.go
      Note: Non-Glazed JS API path that keeps local registry-source normalization
    - Path: pkg/sections/sections.go
      Note: Glazed bootstrap profile-settings decoding to migrate to TypeStringList
ExternalSources:
    - local:geppetto_cli_profile_guide.md
Summary: ""
LastUpdated: 2026-03-19T10:16:48.15180029-04:00
WhatFor: Remove the exported string-splitting helper from engine profile registry loading, let Glazed own list decoding where available, and document follow-up migrations for remaining non-Glazed callers.
WhenToUse: Use when updating Geppetto or downstream callers that currently normalize comma-separated profile registry source strings before calling ParseRegistrySourceSpecs.
---



# Clean up registry loading and remove ParseEngineProfileRegistrySourceEntries

## Overview

This ticket tracks a cleanup of registry loading around `ParseEngineProfileRegistrySourceEntries`. The helper only trims and splits comma-separated input, which duplicates `glazed` list decoding for commands that already declare `profile-registries` as `fields.TypeStringList`.

The implementation goal is to keep `ParseRegistrySourceSpecs([]string)` as the main boundary, convert Geppetto's Glazed-backed paths to `[]string`, and remove the exported string parser. Non-Glazed entrypoints keep their own local normalization where they still accept raw strings.

## Key Links

- Analysis: [analysis/01-registry-loading-cleanup-analysis-and-migration-inventory.md](./analysis/01-registry-loading-cleanup-analysis-and-migration-inventory.md)
- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- profiles
- glazed
- cleanup

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
