---
Title: 'Implementation guide: ordered --profile-registries chain and single-registry YAML cutover'
Ticket: GP-31-PROFILE-REGISTRIES-CHAIN
Status: active
Topics:
    - profile-registry
    - pinocchio
    - geppetto
    - migration
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/profile_policy.go
      Note: |-
        Request-time profile/registry selection and resolver usage in web-chat
        web-chat resolver selection behavior
    - Path: ../../../../../../../pinocchio/pkg/webchat/http/profile_api.go
      Note: CRUD/list endpoint semantics and registry selection inputs
    - Path: pkg/profiles/codec_yaml.go
      Note: |-
        YAML decode/encode behavior to hard-cut to single-registry docs for runtime loading
        YAML format hard-cut target
    - Path: pkg/profiles/file_store_yaml.go
      Note: YAML file store behavior for one-file-one-registry source loading
    - Path: pkg/profiles/service.go
      Note: Reused resolution contract and stack merge/provenance behavior
    - Path: pkg/sections/sections.go
      Note: |-
        Pinocchio middleware chain wiring for CLI profile selection
        CLI settings/middleware chain wiring
ExternalSources: []
Summary: Ordered source-chain proposal for loading profile registries from mixed YAML/SQLite inputs with single-registry YAML format and profile-name-first resolution by registry load order.
LastUpdated: 2026-02-25T16:24:30-05:00
WhatFor: Define concrete implementation details for multi-source registry loading and CLI/web-chat resolution behavior.
WhenToUse: Use when implementing or reviewing GP-31 source-chain registry loading, YAML format cutover, and CLI/web-chat registry resolution.
---


# Implementation guide: ordered --profile-registries chain and single-registry YAML cutover

## Executive Summary

This proposal adds a registry source chain to Pinocchio/Geppetto via `--profile-registries file1,file2,file3`.

Each source is auto-detected as YAML or SQLite. YAML sources are **single-registry only**; SQLite sources can contain multiple registries. Registries are loaded in source order, and profile-name resolution (`--profile <slug>`) searches registries in that exact order.

No overlay abstraction is reintroduced. We keep deterministic routing with strict ownership:

1. registry slugs are globally unique across loaded sources,
2. each registry belongs to exactly one source,
3. writes route to the owning source (and fail for read-only sources).

For GP-31 scope, CRUD endpoints expose all loaded registries.

## Problem Statement

Current behavior does not support the operator workflow requested:

1. mixing multiple registry backends in one run (for example private YAML + shared DB),
2. selecting profiles by name with deterministic fallback across registry sources,
3. a clean one-file-one-registry YAML model for layering source files.

In addition, runtime CLI profile resolution is currently tied to a default registry path rather than an ordered registry chain.

## Proposed Solution

### 1. New CLI/config surface: `profile-registries`

Add a new profile source-chain setting available in config/env/flags:

- flag: `--profile-registries <entry1,entry2,...>`
- env: `PINOCCHIO_PROFILE_REGISTRIES`
- config key: `profile-settings.profile-registries`

Behavior:

1. ordered list semantics are strict and preserved,
2. if empty, fallback to existing single `--profile-file` behavior (implemented as one YAML source),
3. if both are set, `profile-registries` wins and `profile-file` is ignored with a warning.

### 2. Source entry parsing and auto-detection

Each entry is parsed into a `RegistrySourceSpec`.

Recommended format:

- plain path: `/path/to/profiles.yaml`, `/path/to/profiles.db`
- explicit override (optional): `yaml:/path/to/file`, `sqlite:/path/to/file`, `sqlite-dsn:file:/...?...`

Auto-detect rules for plain entries:

1. if extension is `.db`/`.sqlite`/`.sqlite3`, treat as SQLite file,
2. else if file starts with `SQLite format 3`, treat as SQLite file,
3. otherwise parse as YAML and require single-registry document format.

### 3. Single-registry YAML hard cut

Runtime YAML source files must use **single-registry** shape only:

```yaml
slug: private-provider
default_profile_slug: provider-openai
profiles:
  provider-openai:
    slug: provider-openai
    runtime:
      step_settings_patch:
        ai-chat:
          ai-api-type: openai
        openai-chat:
          openai-base-url: https://api.openai.com/v1
          openai-api-key: ${OPENAI_API_KEY}
```

Hard-cut rules:

1. top-level `registries:` bundle format is rejected for runtime loading,
2. legacy profile map format is rejected for runtime loading,
3. migration tooling remains responsible for converting old docs.

### 4. Registry source chain and ownership model

Introduce a chain service (router, not overlay) with these invariants:

1. registries are loaded from each source in source order,
2. registry slug uniqueness is enforced globally across chain,
3. duplicate registry slugs across sources are startup errors,
4. each registry has one owner source for write routing,
5. source can be marked read-only (YAML default read-only; SQLite read-write).

Suggested types:

```go
type RegistrySourceSpec struct {
  Raw string
  Kind SourceKind // yaml|sqlite|sqlite-dsn
  Path string
  DSN  string
}

type LoadedRegistryDescriptor struct {
  RegistrySlug profiles.RegistrySlug
  SourceIndex  int
  SourceLabel  string
  Writable     bool
}
```

### 5. Resolution semantics (ordered profile-name search)

When resolving runtime selection:

1. if explicit registry is provided (`registry_slug` in web-chat, future `--registry` in CLI), resolve profile in that registry only,
2. otherwise, if a profile slug is provided, search registries in chain order and pick first match,
3. if no profile slug is provided, use first registry in chain and its default profile,
4. if profile appears in multiple registries, first match wins (deterministic by chain order).

This directly matches the requested “resolution is done in the order of registries loaded.”

### 6. Stack resolution across registries

Cross-registry stack refs continue using existing `ProfileRef.registry_slug` semantics:

1. empty `registry_slug` means same registry as current layer,
2. explicit `registry_slug` resolves through global chain registry map,
3. missing registry/profile errors remain validation errors with precise path context.

No merge overlay between registries is added.

### 7. CRUD behavior for GP-31 scope

For now, expose all loaded registries in CRUD/list APIs.

Write behavior:

1. list/get operations work for all loaded registries,
2. create/update/delete/default-set route to owner source,
3. writes against read-only source return `403` (or policy error mapped consistently).

Note: this means YAML-backed registries are visible over CRUD in this phase.

### 8. Fingerprint/provenance behavior

Keep existing stack provenance and fingerprint contracts unchanged:

1. `profile.stack.lineage`,
2. `profile.stack.trace`,
3. `runtime_fingerprint`.

Only root registry/profile selection changes based on chain-order search.

## Design Decisions

1. **No overlay abstraction**
Rationale: stack merge/provenance already provide deterministic composition; source chaining is routing, not field-level merge.

2. **Registry slug uniqueness across chain**
Rationale: prevents ambiguous explicit-registry reads/writes and simplifies CRUD/write routing.

3. **Single-registry YAML format for runtime**
Rationale: directly models one file as one source registry and avoids “bundle file as mini database” ambiguity.

4. **Profile-name-first ordered search**
Rationale: matches requested operator UX (`--profile` only, ordered fallback across registries).

5. **Expose all registries in CRUD for now**
Rationale: requested temporary behavior; privacy filtering can be added later as an explicit access policy layer.

## Alternatives Considered

1. **Keep canonical multi-registry YAML for runtime**
Rejected: conflicts with one-file-one-registry layering model and complicates source ownership semantics.

2. **Allow duplicate registry slugs and “first source wins”**
Rejected: explicit registry ops and writes become error-prone and non-obvious.

3. **Reintroduce store overlay merge**
Rejected: duplicates stack composition semantics and reintroduces unnecessary complexity removed in GP-28.

4. **Require explicit `--registry` always**
Rejected: does not satisfy requested profile-name-only ordered resolution workflow.

## Implementation Plan

### Phase 0: Contracts and settings

1. Add `profile-registries` setting to profile selection section wiring.
2. Add parsing helpers for `PINOCCHIO_PROFILE_REGISTRIES` and CLI flag.
3. Document precedence:
   - `--profile-registries` > env > config > `--profile-file` fallback.

### Phase 1: Source parsing and loading

1. Add source parser/autodetect package under `geppetto/pkg/profiles`.
2. Implement loader for:
   - single-registry YAML source,
   - SQLite source (multi-registry).
3. Add startup validation for duplicate registry slugs across chain.

### Phase 2: Chain router service

1. Implement router registry service with:
   - ordered registry descriptors,
   - slug -> owner source routing,
   - read path for all registries,
   - write dispatch by owner source.
2. Reuse existing `StoreRegistry` resolution logic by providing a unified read store view.

### Phase 3: Resolution behavior changes

1. CLI resolver path:
   - if registry unspecified, resolve `--profile` by ordered chain search.
2. web-chat resolver path:
   - preserve explicit `registry_slug` override,
   - otherwise use ordered chain search for `runtime_key`.

### Phase 4: YAML format hard cut

1. Make runtime YAML loaders reject:
   - top-level `registries` bundles,
   - legacy profile-map format.
2. Update migration command defaults to emit single-registry format.
3. Update docs/examples/scripts to one-file-one-registry.

### Phase 5: CRUD integration (allow all registries)

1. Keep list/get across all loaded registries.
2. Write routing:
   - writable source => apply write,
   - read-only source => return explicit write-forbidden error.
3. Keep response schema unchanged except optional diagnostics metadata.

### Phase 6: Tests

1. Unit tests:
   - source autodetection,
   - single-registry YAML validation,
   - duplicate registry slug rejection,
   - ordered profile lookup.
2. Integration tests:
   - mixed YAML+SQLite load,
   - web-chat `/chat` ordered resolution,
   - CRUD read-all + write routing behavior,
   - pinocchio `--print-parsed-fields` with `--profile-registries`.

### Phase 7: Rollout and tooling

1. Provide one-shot migration utility updates for splitting/rewriting YAML.
2. Update smoke scripts to accept multiple registry sources.
3. Publish operator playbook with expected startup failures and remediation.

## Detailed Task Breakdown

1. Add new settings field(s) in geppetto profile middleware selection path.
2. Implement `ParseProfileRegistries(raw []string) ([]RegistrySourceSpec, error)`.
3. Implement `LoadRegistrySource(spec RegistrySourceSpec) ([]*ProfileRegistry, SourceHandle, error)`.
4. Implement duplicate registry slug detection with actionable error messages.
5. Implement `ChainedRegistry` (read routing + owner write dispatch).
6. Refactor CLI profile selection middleware to call chained resolver.
7. Refactor web-chat startup to accept chain sources and build chained registry service.
8. Update web-chat resolver to perform ordered profile search when registry absent.
9. Add read-only error mapping for write attempts against YAML sources.
10. Change runtime YAML decode path to single-registry-only.
11. Update migration command docs/examples and defaults.
12. Add tests for all above with mixed-source fixtures.
13. Update smoke automation script to handle multiple `--profile-registries` entries.
14. Update docs in geppetto + pinocchio for new format and resolution behavior.

## Security and Operational Notes

1. Allowing all registries in CRUD means YAML-backed profile contents are discoverable through list/get endpoints.
2. If private credentials are stored in YAML profile step settings, they will be visible unless separately redacted.
3. For GP-31 scope, this is accepted per request; a follow-up ticket should add registry visibility controls and response redaction policy.

## Open Questions

1. Should `--profile-registries` support repeating flags in addition to comma-separated values?
2. Should writes to read-only sources map to `403` or `400 policy violation` for API compatibility?
3. Should runtime metadata trace payload redact known secret-like keys by default when CRUD is open to all registries?
4. Should we add a strict startup flag requiring explicit source type prefixes (`yaml:`/`sqlite:`) in production?

## References

1. GP-28 stack profile merge/provenance contracts.
2. GP-29 pinocchio web-chat resolver/runtime cutover.
3. Existing profile YAML/SQLite store implementations in `geppetto/pkg/profiles`.
