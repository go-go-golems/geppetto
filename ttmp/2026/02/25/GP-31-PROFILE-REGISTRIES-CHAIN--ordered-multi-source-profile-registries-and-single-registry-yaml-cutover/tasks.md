# Tasks

## Phase 0 - Scope lock

- [ ] Confirm GP-31 hard-cut scope:
  - single-registry YAML runtime format,
  - stack-top-first `--profile-registries` chain,
  - no `default_profile_slug` in runtime YAML sources,
  - no overlay abstraction,
  - no runtime registry selector path,
  - CRUD read exposure for all registries (temporary),
  - write routing by owner source.
- [ ] Confirm duplicate registry slug policy (`startup error`).

## Phase 1 - Settings and parsing surface

- [ ] Add `profile-registries` setting to profile selection chain in geppetto middleware wiring.
- [ ] Wire env/config/flag precedence for `profile-registries`.
- [ ] Remove runtime `profile-file` fallback path from this flow and fail startup when no registry sources are configured.

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

- [ ] Implement stack-top-first profile-name resolution.
- [ ] Remove runtime registry selector path in CLI/web-chat runtime resolution.
- [ ] Update CLI runtime resolution to use ordered source-chain lookup.

## Phase 5 - YAML format hard cut

- [ ] Make runtime YAML loader reject multi-registry bundle format.
- [ ] Make runtime YAML loader reject legacy profile-map format.
- [ ] Make runtime YAML loader reject `default_profile_slug`.
- [ ] Remove legacy runtime `profile-file` loading path from this flow in wiring/docs.

## Phase 6 - Web-chat CRUD behavior

- [ ] Ensure list/get include all loaded registries.
- [ ] Ensure create/update/delete/default-set route by registry owner.
- [ ] Ensure write attempts against read-only sources fail consistently.

## Phase 7 - Tests and validation

- [ ] Add unit tests for source parsing/detection.
- [ ] Add unit tests for duplicate registry slug detection.
- [ ] Add unit tests for stack-top-first profile search behavior.
- [ ] Add integration tests for mixed YAML+SQLite chain loading.
- [ ] Add web-chat tests for stack-top-first resolution with runtime registry selector removed.
- [ ] Add pinocchio `--print-parsed-fields` test coverage with `--profile-registries`.

## Phase 8 - Docs and operator tooling

- [ ] Update geppetto profile docs to single-registry YAML runtime format.
- [ ] Update pinocchio docs for `--profile-registries` behavior.
- [ ] Update smoke scripts for multi-source runs.
- [ ] Document temporary risk of CRUD exposure for YAML-backed private registries.
