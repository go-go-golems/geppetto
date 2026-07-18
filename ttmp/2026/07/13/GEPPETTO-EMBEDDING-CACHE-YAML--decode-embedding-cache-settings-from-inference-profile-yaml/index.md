---
Title: Decode embedding cache settings from inference-profile YAML
Ticket: GEPPETTO-EMBEDDING-CACHE-YAML
Status: active
Topics:
    - embeddings
    - configuration
    - yaml
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/config/settings.go
      Note: EmbeddingsConfig currently omits the YAML tags that profile decoding requires.
    - Path: abs:///home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/settings_factory.go
      Note: Factory consumes decoded cache fields to construct memory or disk cache providers.
ExternalSources: []
Summary: Fix the mismatch between profile YAML cache keys and EmbeddingsConfig decoding so file embedding caching is actually enabled.
LastUpdated: 2026-07-13T15:49:19.456681347-04:00
WhatFor: Ensure inference profiles can reliably select none, memory, or file embedding cache behavior.
WhenToUse: Read before changing embedding profiles, configuration decoding, or cache provider construction.
---

# Decode embedding cache settings from inference-profile YAML

## Overview

Inference profiles already document and use `cache_type`, `cache_max_size`, `cache_max_entries`, and `cache_directory` under `inference_settings.embeddings`. The Go configuration type supplied matching Glazed CLI tags but omitted YAML tags. Consequently profile loading decodes the core provider settings while silently dropping cache configuration and creating an uncached provider.

This ticket adds the missing decode contract and a regression that loads profile-shaped YAML, converts it through the inference-settings factory, and verifies that `file` produces a disk cache provider.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- embeddings
- configuration
- yaml

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
