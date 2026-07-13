---
Title: Embedding cache YAML decoding analysis, design, and implementation guide
Ticket: GEPPETTO-EMBEDDING-CACHE-YAML
Status: active
Topics:
    - configuration
    - embeddings
    - yaml
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/go-go-golems/geppetto/examples/js/geppetto/profiles/40-embeddings.yaml
      Note: Published profile syntax that must decode correctly.
    - Path: abs:///home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/config/settings.go
      Note: Configuration structure requiring cache YAML tags.
    - Path: abs:///home/manuel/code/wesen/go-go-golems/geppetto/pkg/embeddings/settings_factory.go
      Note: Cache-provider selection from InferenceSettings.
    - Path: repo://pkg/doc/topics/06-embeddings.md
      Note: User documentation for canonical profile cache keys
ExternalSources: []
Summary: Intern-ready analysis and implementation plan for making YAML embedding cache settings reach Geppetto's disk-cache provider.
LastUpdated: 2026-07-13T15:49:00-04:00
WhatFor: Define the decoding contract, regression tests, and documentation for embedding cache profile settings.
WhenToUse: Read before modifying EmbeddingsConfig, profile YAML, or SettingsFactory cache behavior.
---


# Embedding cache YAML decoding analysis, design, and implementation guide

## Executive Summary

Geppetto supports three embedding cache modes: no wrapper, in-memory LRU cache, and disk-backed cache. The provider factory already implements these modes. Profile YAML examples also expose the intended keys. The configuration object between those two layers, `pkg/embeddings/config.EmbeddingsConfig`, was missing `yaml` tags for its four cache fields.

The result is silent data loss during YAML decoding. `cache_type: file` and `cache_directory: ...` do not populate `InferenceSettings.Embeddings`. `NewSettingsFactoryFromInferenceSettings` therefore forwards empty cache fields, and `NewProvider` uses its default branch, returning the network-backed embedding provider without a cache. This ticket adds explicit tags and profile-shaped regression coverage.

## Data Flow

```text
profiles YAML
  inference_settings.embeddings.cache_type
  inference_settings.embeddings.cache_directory
                 │ yaml.Unmarshal
                 v
settings.InferenceSettings.Embeddings (*EmbeddingsConfig)
                 │ NewSettingsFactoryFromInferenceSettings
                 v
embeddings.Config { CacheType, CacheDirectory, CacheMaxSize, CacheMaxEntries }
                 │ NewProvider
                 v
none ──> provider
memory ──> CachedProvider(provider)
file ──> DiskCacheProvider(provider, directory, limits)
```

The defect lies at the first arrow. The factory is already correctly structured to copy non-zero cache fields and choose its wrapper.

## Existing API and File References

`EmbeddingsConfig` currently declares:

```go
CacheType       string `glazed:"embeddings-cache-type"`
CacheMaxSize    int64  `glazed:"embeddings-cache-max-size"`
CacheMaxEntries int    `glazed:"embeddings-cache-max-entries"`
CacheDirectory  string `glazed:"embeddings-cache-directory"`
```

The published YAML uses lower snake case:

```yaml
inference_settings:
  embeddings:
    type: ollama
    engine: nomic-embed-text
    dimensions: 768
    cache_type: file
    cache_directory: ./.geppetto/embeddings-cache/ollama-nomic-embed-text
```

The required Go contract is:

```go
CacheType       string `yaml:"cache_type,omitempty" glazed:"embeddings-cache-type"`
CacheMaxSize    int64  `yaml:"cache_max_size,omitempty" glazed:"embeddings-cache-max-size"`
CacheMaxEntries int    `yaml:"cache_max_entries,omitempty" glazed:"embeddings-cache-max-entries"`
CacheDirectory  string `yaml:"cache_directory,omitempty" glazed:"embeddings-cache-directory"`
```

No compatibility aliases are needed. These are the already-published canonical YAML keys. The change makes existing documented configuration effective.

## Design Decisions

### Add explicit YAML tags to all cache fields

Adding only `cache_type` and `cache_directory` would leave the two limit controls silently ineffective. All four configuration fields form one cache contract and receive matching lower-snake-case tags.

### Test decoding through the profile boundary

A direct `yaml.Unmarshal` test of `EmbeddingsConfig` detects the immediate tag regression. A stronger integration test loads a profile-shaped `InferenceSettings` document and sends it into `NewSettingsFactoryFromInferenceSettings`, then asserts `file` yields `*DiskCacheProvider`. This proves the real application path used by JavaScript inference profiles.

### Do not add new environment-based defaults

The transcript playground deliberately relies on selected profile configuration, not process environment variables. This fix preserves that explicit profile boundary and does not introduce `Getenv` behavior.

## Implementation Plan

1. Add the four YAML tags in `pkg/embeddings/config/settings.go`.
2. Add a YAML decoding test with `cache_type`, both limits, and directory values.
3. Add a factory test that constructs an `InferenceSettings` from decoded YAML and verifies a `*DiskCacheProvider` is returned. Use an Ollama configuration because it does not require an API key or network request to construct the provider.
4. Update the embedding workflow documentation to state the canonical profile keys and the cache verification method.
5. Run focused tests, `go test ./...`, formatting, and documentation validation. Record the commands in the diary.

## Test Pseudocode

```text
yaml := profile-shaped document with inference_settings.embeddings:
    type=ollama, engine=nomic-embed-text, dimensions=768
    cache_type=file, cache_max_size=123, cache_max_entries=7
    cache_directory=tempdir

unmarshal yaml into InferenceSettings
assert all four Embeddings cache fields retain their values

factory := NewSettingsFactoryFromInferenceSettings(settings)
provider := factory.NewProvider()
assert provider has concrete type DiskCacheProvider
assert provider model is nomic-embed-text, dimensions 768
```

The test must not send an embedding request. It verifies configuration construction only.

## Failure Behavior

- Unknown cache type remains the existing factory behavior: it returns the underlying provider. This ticket does not change cache-type validation semantics.
- An invalid disk-cache directory remains the existing `NewDiskCacheProvider` construction error.
- A `file` cache profile without a directory keeps the disk-cache provider's existing default directory policy; profile examples should still set an explicit directory for reproducible experiment storage.

## Review Guide

Review `settings.go` first: the tags should mirror the YAML used in `40-embeddings.yaml`. Then review `settings_factory.go`: values copied from `InferenceSettings.Embeddings` should reach `config.EmbeddingsConfig`, and the `file` branch should construct `DiskCacheProvider`. Finally, inspect the test to ensure it would fail if any one cache tag were removed.

For the transcript RAG user, the operational acceptance criterion is simple: resolving the `ollama-nomic-embedding` profile yields a disk cache provider configured for the profile's directory, so repeated embedding runs reuse vectors rather than contacting Ollama again.
