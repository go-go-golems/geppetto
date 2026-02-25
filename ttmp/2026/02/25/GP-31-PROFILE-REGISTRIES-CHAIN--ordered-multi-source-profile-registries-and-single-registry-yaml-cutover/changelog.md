# Changelog

## 2026-02-25

- Initial workspace created

## 2026-02-25

Added the first complete implementation guide for ordered multi-source profile registry loading.

### What changed

- Created design doc defining:
  - `--profile-registries` ordered source-chain behavior,
  - YAML vs SQLite source autodetection,
  - one-file-one-registry YAML runtime format hard cut,
  - profile-name-first resolution by loaded registry order,
  - CRUD behavior (read exposure for all registries, owner-based write routing).
- Added granular phase-based task plan for implementation and validation.
- Updated ticket index metadata and overview for review readiness.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-31-PROFILE-REGISTRIES-CHAIN--ordered-multi-source-profile-registries-and-single-registry-yaml-cutover/design-doc/01-implementation-guide-ordered-profile-registries-chain-and-single-registry-yaml-cutover.md — Detailed GP-31 design and rollout plan
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-31-PROFILE-REGISTRIES-CHAIN--ordered-multi-source-profile-registries-and-single-registry-yaml-cutover/tasks.md — Granular implementation tasks
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-31-PROFILE-REGISTRIES-CHAIN--ordered-multi-source-profile-registries-and-single-registry-yaml-cutover/index.md — Ticket summary and scope framing

## 2026-02-25

Applied hard-cut scope correction: stack-only registry resolution, no runtime registry switching, no `default_profile_slug` in runtime YAML.

### What changed

- Updated GP-31 design doc to enforce:
  - no runtime `profile-file` fallback path,
  - no runtime registry selector path (`registry_slug`/`--registry`) in this flow,
  - stack-top-first profile resolution,
  - runtime YAML single-registry shape without `default_profile_slug`.
- Updated GP-31 tasks to match the corrected hard-cut scope and validation matrix.
- Updated ticket index wording to make stack-top-first profile resolution explicit and avoid selector-style phrasing.

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-31-PROFILE-REGISTRIES-CHAIN--ordered-multi-source-profile-registries-and-single-registry-yaml-cutover/design-doc/01-implementation-guide-ordered-profile-registries-chain-and-single-registry-yaml-cutover.md — Scope corrected to stack-only runtime semantics
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-31-PROFILE-REGISTRIES-CHAIN--ordered-multi-source-profile-registries-and-single-registry-yaml-cutover/tasks.md — Tasks aligned with no-fallback/no-registry-switch model

## 2026-02-25

Implemented GP-31 core runtime behavior across geppetto and pinocchio with phase-by-phase commits.

### What changed

- Geppetto profiles:
  - added source spec parsing/autodetection and chain loading (`yaml`/`sqlite`/`sqlite-dsn`),
  - added chained registry routing with owner-based writes and read-only enforcement,
  - added strict runtime YAML decoder rejecting bundle/legacy/default-profile-slug formats.
- Geppetto sections:
  - switched profile bootstrap to `profile-settings.profile-registries`,
  - removed runtime `profile-file` fallback from middleware wiring.
- Pinocchio:
  - added root `--profile-registries` persistent flag for CLI commands,
  - switched web-chat startup to chained profile registries from `--profile-registries`,
  - removed request-time runtime registry switching from web-chat resolver,
  - mapped read-only registry write failures to `403` in profile API.

### Commits

- `c88a1e3` (geppetto) — `profiles: add source-chain registry service and strict runtime YAML loader`
- `683fc10` (geppetto) — `sections: require profile-registries and load profile stack middleware`
- `d070241` (geppetto) — `profiles: expose chained registry default slug accessor`
- `0108628` (pinocchio) — `web-chat: load profile registry chains and remove runtime registry switching`

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/source_chain.go — Chain service, source autodetection, owner write routing
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/codec_yaml_runtime.go — Strict runtime YAML loader
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/sections/sections.go — `profile-registries` bootstrap and hard-cut wiring
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/sections/profile_registry_source.go — Profile middleware now backed by chained registries
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/main.go — Web-chat chain startup from `profile-registries`
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go — Runtime resolver without registry switching
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/profile_api.go — Read-only write error mapping

## 2026-02-25

Follow-up implementation completed remaining near-term GP-31 validation/API behavior gaps.

### What changed

- Added explicit duplicate registry slug test coverage for chain startup validation.
- Updated web-chat profile API to:
  - list profiles across all loaded registries when `registry` is omitted,
  - get profile by slug across loaded registries when `registry` is omitted,
  - preserve existing response contract shape for list items.
- Updated GP-31 tasks status to mark completed test and list/get coverage items.

### Commits

- `bc338dd` (geppetto) — `profiles: add duplicate registry slug chain test`
- `10815ea` (pinocchio) — `web-chat: list and get profiles across loaded registries`

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/source_chain_test.go — Duplicate slug rejection test
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/profile_api.go — Cross-registry list/get behavior
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy_test.go — API behavior assertions for loaded multi-registry sources

## 2026-02-25

Completed GP-31 Phase 7/8 closeout: parsed-fields coverage, docs hard-cut updates, and multi-source smoke script validation.

### What changed

- Added pinocchio integration coverage for `--print-parsed-fields` with `--profile-registries`.
- Updated cutover smoke script to:
  - migrate legacy profile maps,
  - import bundle output into SQLite,
  - generate and layer a top runtime YAML registry source,
  - run web-chat with `--profile-registries <db>,<top-yaml>`,
  - validate `/chat` runtime metadata and `--print-parsed-fields` source metadata.
- Updated geppetto and pinocchio docs for hard-cut semantics:
  - no runtime `profile-file` path in this flow,
  - no request-time `registry_slug` runtime selector,
  - `--profile-registries` as canonical source stack input.
- Documented current CRUD read exposure risk for YAML-backed private registries.

### Commits

- `c8fcdef` (pinocchio) — `profiles: add profile-registries parsed-fields coverage and cutover smoke script`
- `9a172dd` (geppetto) — `docs(gp-31): finalize hard-cut profile-registry docs and close remaining tasks`

### Validation

- `go test ./cmd/pinocchio ./cmd/web-chat ./pkg/webchat/http -count=1`
- `go test ./pkg/profiles ./pkg/sections -count=1`
- `scripts/profile_registry_cutover_smoke.sh --port 18125` (pass)

### Related Files

- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/pinocchio/main_profile_registries_test.go — Parsed-fields integration coverage for `--profile-registries`
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/scripts/profile_registry_cutover_smoke.sh — Multi-source cutover smoke workflow
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/doc/topics/01-profiles.md — Hard-cut profile registry docs
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md — Migration playbook updates for stack sources
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/doc/topics/webchat-profile-registry.md — Web-chat registry behavior and CRUD exposure note
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/doc/topics/webchat-http-chat-setup.md — Runtime request contract update
- /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/README.md — Updated CLI examples and request payload

## 2026-02-25

Ticket closed

