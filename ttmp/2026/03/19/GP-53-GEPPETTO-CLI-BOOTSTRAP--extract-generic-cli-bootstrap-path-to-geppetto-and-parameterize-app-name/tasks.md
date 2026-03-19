# Tasks

## TODO

- [ ] Audit the current Geppetto bootstrap surfaces and classify them into direct parsed-AI paths, registry-only helper paths, and older middleware/bootstrap helper paths.
- [ ] Define the target Geppetto package boundary for generic CLI bootstrap and record the proposed package name, public types, and public functions.
- [ ] Design the app-parameterization surface for app name, env prefix, config-file mapper, profile section construction, and baseline section construction.
- [ ] Port the generic `ResolvedCLIProfileSelection` and `ResolvedCLIEngineSettings` contracts from Pinocchio into the new Geppetto package without Pinocchio imports.
- [ ] Port the generic config-file discovery, profile-selection, base-settings, and final-settings resolution flow into Geppetto and make it use the app-parameterization config.
- [ ] Add focused Geppetto tests for config discovery, env prefix handling, no implicit registry fallback, profile-without-registries validation, base-only mode, and profile overlay merge behavior.
- [ ] Migrate `pinocchio/pkg/cmds/profilebootstrap` to wrap or call the new Geppetto package and remove duplicated implementation.
- [ ] Re-run Pinocchio loaded-command, thin-command, and web-chat bootstrap verification against the Geppetto-owned implementation.
- [ ] Decide whether `geppetto/pkg/sections/sections.go` and `geppetto/pkg/sections/profile_sections.go` should be slimmed down, deprecated, or partially rewritten after the new bootstrap package exists.
- [ ] Decide which Geppetto examples should continue using `factory.NewEngineFromParsedValues(...)` and which should adopt the new bootstrap package for config/profile-registry-aware behavior.
