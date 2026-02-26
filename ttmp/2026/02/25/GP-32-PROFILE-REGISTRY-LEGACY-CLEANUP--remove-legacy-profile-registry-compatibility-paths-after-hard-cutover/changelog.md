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
- Implemented Phase 3 pinocchio hard-cut cleanup:
  - removed legacy `WithProfileFile`/`GatherFlagsFromProfiles` flow from `pkg/cmds/helpers/parse-helpers.go`.
  - parse helper now resolves profile registries from `PINOCCHIO_PROFILE_REGISTRIES` (or default `~/.config/pinocchio/profiles.yaml` when present) and uses registry-stack middleware.
  - removed Clay legacy `profiles` command initialization and the embedded legacy template from `cmd/pinocchio/main.go`.
  - added native `profiles` command group and kept `profiles migrate-legacy` under it.
  - updated `scripts/profile_registry_cutover_smoke.sh` to migrate/import runtime single-registry YAML directly (no bundle assumptions).
- Implemented Phase 4 docs/validation closure:
  - rewrote `pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md` to hard-cut runtime single-registry semantics.
  - validation matrix completed:
    - `go test ./...` in `geppetto` (pre-commit),
    - `go test ./...` in `pinocchio` (pre-commit),
    - manual smoke script run: `scripts/profile_registry_cutover_smoke.sh` passed end-to-end.
- Follow-up hard-cut cleanup:
  - removed Glazed profile settings layer wiring (`cli.WithProfileSettingsSection`) from pinocchio and geppetto example command setup.
  - moved profile flag ownership to geppetto sections (`profile` + `profile-registries`) and ensured legacy `profile-file` is not exposed.
  - added regression coverage for flag surface (`--profile` / `--profile-registries` present, `--profile-file` absent).
