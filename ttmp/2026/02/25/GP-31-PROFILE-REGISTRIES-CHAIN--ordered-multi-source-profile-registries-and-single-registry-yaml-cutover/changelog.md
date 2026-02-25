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
