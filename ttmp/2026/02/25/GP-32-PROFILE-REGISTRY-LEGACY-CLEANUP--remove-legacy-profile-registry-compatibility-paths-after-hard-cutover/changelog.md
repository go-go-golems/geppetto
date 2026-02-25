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
