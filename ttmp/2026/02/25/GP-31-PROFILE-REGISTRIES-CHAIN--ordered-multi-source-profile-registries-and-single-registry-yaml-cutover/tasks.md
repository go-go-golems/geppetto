# Tasks

## Phase 0 - Scope lock

- [ ] Confirm GP-31 hard-cut scope:
  - single-registry YAML runtime format,
  - ordered `--profile-registries` chain,
  - no overlay abstraction,
  - CRUD read exposure for all registries (temporary),
  - write routing by owner source.
- [ ] Confirm duplicate registry slug policy (`startup error`).

## Phase 1 - Settings and parsing surface

- [ ] Add `profile-registries` setting to profile selection chain in geppetto middleware wiring.
- [ ] Wire env/config/flag precedence for `profile-registries`.
- [ ] Keep `profile-file` fallback when `profile-registries` is unset.

## Phase 2 - Source detection and loading

- [ ] Implement source spec parser for ordered entries.
- [ ] Implement auto-detect for YAML vs SQLite sources.
- [ ] Load YAML source as exactly one registry.
- [ ] Load SQLite source as N registries.
- [ ] Fail startup on duplicate registry slugs across sources.

## Phase 3 - Chain registry service (router)

- [ ] Implement chained registry read path for all loaded registries.
- [ ] Implement write routing by registry owner source.
- [ ] Return explicit read-only write errors for non-writable owners.
- [ ] Reuse existing stack merge/provenance/fingerprint resolution contracts.

## Phase 4 - Resolution semantics

- [ ] Implement ordered profile-name resolution when registry is unspecified.
- [ ] Keep explicit registry selection path for web-chat/API inputs.
- [ ] Update CLI runtime resolution to use ordered source-chain lookup.

## Phase 5 - YAML format hard cut

- [ ] Make runtime YAML loader reject multi-registry bundle format.
- [ ] Make runtime YAML loader reject legacy profile-map format.
- [ ] Update migration command defaults/docs for single-registry output.

## Phase 6 - Web-chat CRUD behavior

- [ ] Ensure list/get include all loaded registries.
- [ ] Ensure create/update/delete/default-set route by registry owner.
- [ ] Ensure write attempts against read-only sources fail consistently.

## Phase 7 - Tests and validation

- [ ] Add unit tests for source parsing/detection.
- [ ] Add unit tests for duplicate registry slug detection.
- [ ] Add unit tests for ordered profile search behavior.
- [ ] Add integration tests for mixed YAML+SQLite chain loading.
- [ ] Add web-chat tests for ordered resolution + explicit registry override.
- [ ] Add pinocchio `--print-parsed-fields` test coverage with `--profile-registries`.

## Phase 8 - Docs and operator tooling

- [ ] Update geppetto profile docs to single-registry YAML runtime format.
- [ ] Update pinocchio docs for `--profile-registries` behavior.
- [ ] Update migration/smoke scripts for multi-source runs.
- [ ] Document temporary risk of CRUD exposure for YAML-backed private registries.
