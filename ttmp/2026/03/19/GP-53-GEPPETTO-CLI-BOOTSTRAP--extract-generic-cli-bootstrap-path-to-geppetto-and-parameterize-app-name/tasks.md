# Tasks

## TODO

- [x] Audit the current Geppetto bootstrap surfaces and classify them into direct parsed-AI paths, registry-only helper paths, and older middleware/bootstrap helper paths.
- [x] Define the target Geppetto package boundary for generic CLI bootstrap and record the proposed package name, public types, and public functions.
- [x] Design the app-parameterization surface for app name, env prefix, config-file mapper, profile section construction, and baseline section construction.
- [x] Create the new `geppetto/pkg/cli/bootstrap` package and add the generic app-config struct plus validation helpers.
- [x] Port the generic `ProfileSettings`, `ResolvedCLIProfileSelection`, `ResolvedCLIEngineSettings`, and CLI selection input contracts from Pinocchio into the new Geppetto package without Pinocchio imports.
- [x] Port the generic config-file discovery helpers into Geppetto and make them depend only on the new app-config struct.
- [x] Port the generic profile-selection resolution flow into Geppetto and make env prefix, app name, config mapper, and profile section construction caller-configurable.
- [x] Port the generic base-inference-settings resolution flow into Geppetto and make baseline section construction caller-configurable.
- [x] Port the final engine-settings resolution and engine-construction helpers into Geppetto.
- [x] Add focused Geppetto tests for config discovery, env prefix handling, no implicit registry fallback, profile-without-registries validation, base-only mode, explicit profile overlay merge behavior, and from-base parity.
- [x] Add a Pinocchio-owned wrapper config for app name `pinocchio`, env prefix `PINOCCHIO`, the Pinocchio config mapper, and the shared Geppetto section builders.
- [x] Migrate `pinocchio/pkg/cmds/profilebootstrap` to wrap or call the new Geppetto package and remove duplicated implementation.
- [x] Re-run Pinocchio loaded-command, thin-command, and web-chat bootstrap verification against the Geppetto-owned implementation.
- [x] Decide whether `geppetto/pkg/sections/sections.go` and `geppetto/pkg/sections/profile_sections.go` should be slimmed down, deprecated, or partially rewritten after the new bootstrap package exists.
- [x] Decide which Geppetto examples should continue using `factory.NewEngineFromParsedValues(...)` and which should adopt the new bootstrap package for config/profile-registry-aware behavior.
