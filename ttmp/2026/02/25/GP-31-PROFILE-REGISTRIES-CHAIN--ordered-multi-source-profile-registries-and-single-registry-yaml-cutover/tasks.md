# Tasks

## Phase 0 - Scope lock

- [x] Confirm GP-31 hard-cut scope:
  - single-registry YAML runtime format,
  - stack-top-first `--profile-registries` chain,
  - no `default_profile_slug` in runtime YAML sources,
  - no overlay abstraction,
  - no runtime registry selector path,
  - CRUD read exposure for all registries (temporary),
  - write routing by owner source.
- [x] Confirm duplicate registry slug policy (`startup error`).

## Phase 1 - Settings and parsing surface

- [x] Add `profile-registries` setting to profile selection chain in geppetto middleware wiring.
- [x] Wire env/config/flag precedence for `profile-registries`.
- [x] Remove runtime `profile-file` fallback path from this flow and fail startup when no registry sources are configured.

## Phase 2 - Source detection and loading

- [x] Implement source spec parser for ordered entries.
- [x] Implement auto-detect for YAML vs SQLite sources.
- [x] Load YAML source as exactly one registry.
- [x] Load SQLite source as N registries.
- [x] Fail startup on duplicate registry slugs across sources.

## Phase 3 - Chain registry service (router)

- [x] Implement chained registry read path for all loaded registries.
- [x] Implement write routing by registry owner source.
- [x] Return explicit read-only write errors for non-writable owners.
- [x] Reuse existing stack merge/provenance/fingerprint resolution contracts.

## Phase 4 - Resolution semantics

- [x] Implement stack-top-first profile-name resolution.
- [x] Remove runtime registry selector path in CLI/web-chat runtime resolution.
- [x] Update CLI runtime resolution to use ordered source-chain lookup.

## Phase 5 - YAML format hard cut

- [x] Make runtime YAML loader reject multi-registry bundle format.
- [x] Make runtime YAML loader reject legacy profile-map format.
- [x] Make runtime YAML loader reject `default_profile_slug`.
- [x] Remove legacy runtime `profile-file` loading path from this flow in wiring/docs.

## Phase 6 - Web-chat CRUD behavior

- [x] Ensure list/get include all loaded registries.
- [x] Ensure create/update/delete/default-set route by registry owner.
- [x] Ensure write attempts against read-only sources fail consistently.

## Phase 7 - Tests and validation

- [x] Add unit tests for source parsing/detection.
- [x] Add unit tests for duplicate registry slug detection.
- [x] Add unit tests for stack-top-first profile search behavior.
- [x] Add integration tests for mixed YAML+SQLite chain loading.
- [x] Add web-chat tests for stack-top-first resolution with runtime registry selector removed.
- [ ] Add pinocchio `--print-parsed-fields` test coverage with `--profile-registries`.

## Phase 8 - Docs and operator tooling

- [ ] Update geppetto profile docs to single-registry YAML runtime format.
- [ ] Update pinocchio docs for `--profile-registries` behavior.
- [ ] Update smoke scripts for multi-source runs.
- [ ] Document temporary risk of CRUD exposure for YAML-backed private registries.
