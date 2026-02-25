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
Summary: Stack-ordered source-chain proposal for loading profile registries from mixed YAML/SQLite inputs with single-registry YAML format, top-of-stack profile precedence, and no runtime registry switching.
LastUpdated: 2026-02-25T16:31:05-05:00
WhatFor: Define concrete implementation details for multi-source registry loading and CLI/web-chat resolution behavior.
WhenToUse: Use when implementing or reviewing GP-31 source-chain registry loading, YAML format cutover, and CLI/web-chat registry resolution.
---


# Implementation guide: ordered --profile-registries chain and single-registry YAML cutover

## Executive Summary

This proposal adds a registry source chain to Pinocchio/Geppetto via `--profile-registries file1,file2,file3`.

Each source is auto-detected as YAML or SQLite. YAML sources are **single-registry only**; SQLite sources can contain multiple registries. Sources form a pure stack; later entries are the top of stack (higher precedence), and profile-name resolution (`--profile <slug>`) searches top -> bottom.

No overlay abstraction is reintroduced. We keep deterministic routing with strict ownership:

1. registry slugs are globally unique across loaded sources,
2. each registry belongs to exactly one source,
3. writes route to the owning source (and fail for read-only sources).

For GP-31 scope, CRUD endpoints expose all loaded registries.

## Problem Statement

Current behavior does not support the operator workflow requested:

1. mixing multiple registry backends in one run (for example private YAML + shared DB),
2. selecting profiles by name with deterministic ordered resolution across registry sources,
3. a clean one-file-one-registry YAML model for layering source files.

In addition, runtime selection currently exposes/assumes registry switching semantics that are not needed for this stack-only model.

## Proposed Solution

### 1. New CLI/config surface: `profile-registries`

Add a new profile source-chain setting available in config/env/flags:

- flag: `--profile-registries <entry1,entry2,...>`
- env: `PINOCCHIO_PROFILE_REGISTRIES`
- config key: `profile-settings.profile-registries`

Behavior:

1. ordered list semantics are strict and preserved,
2. value must be non-empty; startup fails when no registry sources are configured,
3. `--profile-file`/`PINOCCHIO_PROFILE_FILE` legacy runtime loading path is removed from this flow (hard cut),
4. runtime registry selector inputs are removed from this flow; source stack order is authoritative.

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
3. `default_profile_slug` is rejected for runtime YAML source files,
4. runtime startup fails fast with actionable validation errors when old formats are provided.

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

### 5. Resolution semantics (stack-top-first profile search)

When resolving runtime selection:

1. runtime registry switching is not supported,
2. if a profile slug is provided, search registries from stack top -> bottom and pick first match,
3. if no profile slug is provided, resolve literal slug `default` using stack top -> bottom search,
4. if profile appears in multiple registries, top-most match wins.

This directly matches the requested stack behavior (“the one on top is used”).

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

Only root profile selection changes based on stack-top-first search.

## Design Decisions

1. **No overlay abstraction**
Rationale: stack merge/provenance already provide deterministic composition; source chaining is routing, not field-level merge.

2. **Registry slug uniqueness across chain**
Rationale: prevents ambiguous explicit-registry reads/writes and simplifies CRUD/write routing.

3. **Single-registry YAML format for runtime**
Rationale: directly models one file as one source registry and avoids “bundle file as mini database” ambiguity.

4. **Profile-name-first ordered search**
Rationale: matches requested operator UX (`--profile` only, top-of-stack precedence).

5. **No runtime registry selector path**
Rationale: registry chain is a runtime source stack; explicit registry switching is unnecessary complexity.

6. **Expose all registries in CRUD for now**
Rationale: requested temporary behavior; privacy filtering can be added later as an explicit access policy layer.

## Alternatives Considered

1. **Keep canonical multi-registry YAML for runtime**
Rejected: conflicts with one-file-one-registry layering model and complicates source ownership semantics.

2. **Allow duplicate registry slugs and “first source wins”**
Rejected: explicit registry ops and writes become error-prone and non-obvious.

3. **Reintroduce store overlay merge**
Rejected: duplicates stack composition semantics and reintroduces unnecessary complexity removed in GP-28.

4. **Keep runtime registry selector (`registry_slug` / `--registry`)**
Rejected: does not fit the stack-only model and adds avoidable branching semantics.

## Implementation Plan

### Phase 0: Contracts and settings

1. Add `profile-registries` setting to profile selection section wiring.
2. Add parsing helpers for `PINOCCHIO_PROFILE_REGISTRIES` and CLI flag.
3. Remove legacy runtime profile-file fallback path from this flow and fail startup when no registry sources are configured.

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
   - resolve `--profile` by stack-top-first search.
2. web-chat resolver path:
   - remove runtime registry selector path from request resolution,
   - resolve `runtime_key` by stack-top-first search.

### Phase 4: YAML format hard cut

1. Make runtime YAML loaders reject:
   - top-level `registries` bundles,
   - legacy profile-map format,
   - `default_profile_slug`.
2. Remove legacy runtime profile-file loading path from docs/code surface in this flow.
3. Update docs/examples/scripts to one-file-one-registry + `--profile-registries`.

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

1. Update smoke scripts to accept multiple registry sources.
2. Publish operator playbook with expected startup failures and remediation.

## Detailed Task Breakdown

1. Add new settings field(s) in geppetto profile middleware selection path.
2. Implement `ParseProfileRegistries(raw []string) ([]RegistrySourceSpec, error)`.
3. Implement `LoadRegistrySource(spec RegistrySourceSpec) ([]*ProfileRegistry, SourceHandle, error)`.
4. Implement duplicate registry slug detection with actionable error messages.
5. Implement `ChainedRegistry` (read routing + owner write dispatch).
6. Refactor CLI profile selection middleware to call chained resolver.
7. Refactor web-chat startup to accept chain sources and build chained registry service.
8. Update web-chat resolver to remove runtime registry selector and use stack-top-first profile search.
9. Add read-only error mapping for write attempts against YAML sources.
10. Change runtime YAML decode path to single-registry-only.
11. Remove runtime `profile-file` fallback path and runtime registry selector path from this flow (flags/env/config wiring and docs).
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
