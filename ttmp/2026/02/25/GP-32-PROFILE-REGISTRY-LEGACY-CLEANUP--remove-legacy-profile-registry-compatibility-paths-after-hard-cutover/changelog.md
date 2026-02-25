# Changelog

## 2026-02-25

- Initial workspace created
- Added initial hard-cut cleanup inventory and phased removal plan in:
  - `design/01-hard-cut-cleanup-inventory-and-removal-plan.md`
- Replaced placeholder task list with granular implementation tasks covering:
  - migration command output contract,
  - codec/store compatibility removals,
  - CLI/helper cleanup,
  - test/script/doc alignment and final validation.
- Implemented Phase 1 migration output contract update:
  - `pinocchio profiles migrate-legacy` now emits runtime single-registry YAML (`slug` + `profiles`) by default.
  - canonical bundle (`registries:`) input is now rejected in normal mode and only no-op'd with `--skip-if-not-legacy`.
  - default output path changed from `<input>.registry.yaml` to `<input>.runtime.yaml`.
  - migrate command tests updated to assert hard-cut output shape.
- Implemented Phase 2 codec/store hard-cut updates in geppetto:
  - `pkg/profiles/codec_yaml.go` no longer decodes legacy map or canonical bundle YAML; wrappers now accept runtime single-registry documents only.
  - added `EncodeRuntimeYAMLSingleRegistry(...)` and switched single-registry encoding to omit `default_profile_slug`.
  - `YAMLFileProfileStore` now reads/writes runtime single-registry YAML and rejects operations against non-default registry slugs.
  - removed multi-registry YAML expectations from YAML-store tests and updated codec tests to assert strict rejection of legacy/bundle input.
  - adjusted stack-ref parity test matrix to run multi-registry backend parity on memory/sqlite only.
