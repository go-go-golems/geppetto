# Tasks

## TODO

- [x] Change `pinocchio profiles migrate-legacy` output to runtime single-registry YAML (`slug` + `profiles`) by default.
- [x] Remove bundle-format encode/decode paths from `geppetto/pkg/profiles/codec_yaml.go` (`registries:` document support).
- [x] Remove legacy profile-map conversion support from `DecodeYAMLRegistries` / `ConvertLegacyProfilesMapToRegistry`.
- [x] Update `YAMLFileProfileStore` to strict single-registry YAML read/write semantics (or remove the store if no runtime caller remains).
- [x] Update/replace tests that currently validate legacy map or bundle behavior (`codec_yaml_test.go`, `file_store_yaml_test.go`, migrate-legacy tests).
- [ ] Remove old profile-file helper flow in `pinocchio/pkg/cmds/helpers/parse-helpers.go` (`WithProfileFile`, `GatherFlagsFromProfiles` usage).
- [ ] Remove Clay legacy `profiles` command wiring and `pinocchioInitialProfilesContent` template from `cmd/pinocchio/main.go`.
- [ ] Remove migration-parity tests against `sources.GatherFlagsFromProfiles` that no longer reflect supported runtime behavior.
- [ ] Update `pinocchio/scripts/profile_registry_cutover_smoke.sh` to operate on single-registry runtime YAML output.
- [ ] Update docs to eliminate remaining references to bundle output as canonical behavior.
- [ ] Run full validation matrix (`go test ./...` in `geppetto`, targeted + full tests in `pinocchio`, smoke scripts).
- [ ] Capture final hard-cut decision and cleanup outcomes in ticket changelog/diary.
