# Tasks

## TODO

- [x] Switch the `runner-glazed-registry-flags` example and shared runnerexample helper to `[]string` registry sources.
- [x] Add the reusable profile settings section migration analysis, including duplicated sections and raw-flag owners.
- [x] Publish the shared Geppetto `profile-settings` section and migrate Geppetto section owners to it.
- [x] Migrate Pinocchio section owners and most raw Cobra profile/profile-registries flags to the shared section.
- [x] Convert `geppetto/cmd/examples/runner-registry` from raw `flag` parsing to the shared profile-settings section.
- [x] Run targeted verification, update ticket bookkeeping, and summarize any remaining `flag`-package/script exceptions.
- [x] Write detailed intern-facing guide for Pinocchio/Geppetto CLI config, profile, and engine bootstrap; upload bundle to reMarkable

## Next Phase: Simplify CLI Config And Engine Profile Loading

- [x] Add a short decision note to the ticket adopting `sources/local/geppetto_cli_profile_guide.md` as the preferred direction for the next phase, and explicitly record that runtime profiles are out of scope for now.
- [x] Inventory every current Geppetto-backed CLI entrypoint in Pinocchio and Geppetto and classify each one as loaded command, Glazed/Cobra command, or lightweight/manual bootstrap path.
- [x] Document the target semantics for baseline app config versus engine profile registries:
  `config.yaml` provides baseline inference defaults and other CLI/app settings; `profiles.yaml` remains an engine profile registry and is not overloaded as a baseline config file.
- [x] Decide and document the exact default discovery rules for config and registry files, including home-directory defaults, legacy fallback paths if any, and how explicit CLI flags override discovery.
- [x] Add a focused design note for the minimal first implementation that excludes runtime profiles, tools, and middleware profile composition.
- [x] Introduce one shared helper contract for resolving CLI profile selection from parsed values:
  consume `profile` and `profile-registries` from the shared Geppetto profile-settings section, normalize discovered registry paths, and return one explicit resolved selection structure.
- [x] Introduce one shared helper contract for resolving final inference settings from a baseline config plus optional profile overlay, with a clearly documented precedence order for command defaults, config file values, profile overlay values, environment, and explicit flags.
- [x] Introduce one shared helper for fast engine creation from the resolved final inference settings so commands stop open-coding `factory.NewEngineFromParsedValues(...)` in registry-aware paths.
- [x] Implement the parsed-values path for loaded/full commands so `PinocchioCommand.RunIntoWriter(...)` preserves command-local defaults while delegating profile selection, profile overlay, and engine creation to the shared helpers.
- [x] Implement the lightweight/bootstrap path for JS, agents, and other thin commands that do not already have a fully parsed loaded-command context, using the same profile section and the same final engine resolution rules.
- [x] Refactor `pinocchio/pkg/cmds/cmd.go` to use the shared resolution helpers and remove duplicated engine bootstrap logic.
- [x] Refactor `pinocchio/pkg/cmds/helpers/profile_runtime.go` so its non-runtime responsibilities collapse into the shared config/profile/engine helpers, leaving no second partially overlapping bootstrap path.
- [x] Refactor `pinocchio/pkg/cmds/loader.go` so loader-side inference defaults are surfaced explicitly instead of re-parsing YAML blobs later just to rediscover baseline settings.
- [x] Reconcile current mismatch in no-registry behavior and make the rule explicit:
  commands must be able to run from baseline config plus direct flags even when no profile registry is present, while still supporting profile overlay when registries are provided or discovered.
- [x] Standardize how commands mount the shared Geppetto `profile-settings` section and any baseline config section so profile/config loading works the same across Pinocchio command families and Geppetto examples.
- [x] Add targeted tests for config/profile precedence and bootstrap parity:
  baseline config only, explicit `--profile`, explicit `--profile-registries`, discovered registry file, loaded-command defaults preservation, and manual/bootstrap command parity with loaded commands.
- [x] Update ticket docs and implementation notes after each refactor step so the adopted simplified model stays aligned with the imported guide and does not drift back toward runtime-profile scope.
