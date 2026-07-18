---
Title: Implementation diary
Ticket: GEPPETTO-EMBEDDING-CACHE-YAML
Status: active
Topics:
    - embeddings
    - configuration
    - yaml
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/config/settings.go
      Note: Cache configuration YAML decode contract.
    - Path: abs:///home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/settings_factory_test.go
      Note: Profile-shaped YAML and disk-cache construction regression test.
ExternalSources: []
Summary: Chronological diagnosis and implementation record for embedding cache profile YAML decoding.
LastUpdated: 2026-07-13T15:49:20.453610952-04:00
WhatFor: Preserve the exact root cause, test boundary, validation command, and review concerns for the cache decoding fix.
WhenToUse: Read before changing embedding profiles or cache configuration loading.
---

# Implementation diary

## Goal

Record the fix that makes embedding cache settings declared in inference-profile YAML reach the cache provider factory.

## Context

The transcript RAG playground selects Geppetto embedding profiles and expects file cache configuration to avoid repeated Ollama requests. This diary captures why that configuration was silently ignored and how the test locks the behavior down.

## Quick Reference

Run the focused regression with:

```bash
go test ./pkg/embeddings/...
```

The test unmarshals profile-shaped YAML with `cache_type: file` and verifies that `NewSettingsFactoryFromInferenceSettings(...).NewProvider()` returns `*DiskCacheProvider` without sending any embedding request.

## Usage Examples

For a persistent profile cache, use `cache_type: file` and an explicit `cache_directory` under `inference_settings.embeddings`.

## Related

See `examples/js/geppetto/profiles/40-embeddings.yaml` and `pkg/doc/topics/06-embeddings.md`.

## Step 1: Decode profile cache settings and verify provider construction

The embedding provider factory already had working memory and disk cache branches. The fault was upstream: `EmbeddingsConfig` had no YAML tags for its cache fields, although examples used snake-case YAML keys. YAML decoding therefore left cache settings at zero values, and the factory returned the base provider.

The fix adds all four cache YAML tags and verifies the complete construction path with a local Ollama profile. The test deliberately stops before a network request: it only proves that the selected profile creates the correct wrapped provider and preserves its configured directory.

### Prompt Context

**User prompt (verbatim):** "ok go ahead, whatever"

**Assistant interpretation:** Continue the requested transcript-RAG prerequisite work by creating a Geppetto ticket/design and fixing the embedding-cache profile decode issue.

**Inferred user intent:** Ensure real retrieval experiments do not silently re-embed vectors and distort their cost or latency measurements.

### What I did

- Created ticket `GEPPETTO-EMBEDDING-CACHE-YAML` with implementation tasks and an intern-facing design guide.
- Added YAML tags for `cache_type`, `cache_max_size`, `cache_max_entries`, and `cache_directory`.
- Added a profile-shaped YAML regression test that checks both decoded values and `*DiskCacheProvider` construction.
- Documented the canonical profile cache keys in the embeddings user guide.

### Why

The published YAML was already the intended public interface. Adding tags restores that contract without an adapter, fallback key, environment dependency, or API migration.

### What worked

- `go fmt ./pkg/embeddings/...` completed.
- `go test ./pkg/embeddings/...` passed, including the new no-network disk-cache construction test.

### What didn't work

- Before the change, the cache fields had only `glazed` tags. YAML unmarshalling accepted the document but silently left the fields empty; this is the root cause rather than an Ollama or disk-cache implementation failure.

### What I learned

- A configuration field consumed by both Glazed and YAML needs both tag contracts explicitly declared.
- A factory construction test is stronger than a raw unmarshal test because it detects a regression in either tag decoding or value propagation.

### What was tricky to build

The factory's `file` path creates its cache directory, so the regression needs a per-test temporary directory. The test uses an Ollama provider because construction requires no credentials or request, keeping it deterministic and local.

### What warrants a second pair of eyes

- Confirm that silently treating an unknown `cache_type` as uncached remains desired behavior; this ticket intentionally preserves it.
- Confirm the public profile examples should include cache limit keys as optional values, not mandatory defaults.

### What should be done in the future

- The transcript RAG runner should report whether its resolved embedding provider is disk-cached during live experiment output.

### Code review instructions

- Start with the four tags in `pkg/embeddings/config/settings.go`.
- Read the YAML fixture and concrete provider assertion in `settings_factory_test.go`.
- Run `go test ./pkg/embeddings/...`; no Ollama process is required.

### Technical details

The canonical mapping is YAML `cache_type` → `EmbeddingsConfig.CacheType` → factory switch `file` → `NewDiskCacheProvider`. The same mapping applies to the directory and two cache limits.
