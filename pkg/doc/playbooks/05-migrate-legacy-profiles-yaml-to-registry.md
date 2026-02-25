---
Title: "Migration playbook: legacy profiles.yaml to registry format"
Slug: migrate-legacy-profiles-to-registry
Short: Step-by-step guide to convert legacy profile maps into canonical profile registry YAML and adopt registry-first runtime selection.
Topics:
  - profiles
  - migration
  - configuration
  - pinocchio
Commands:
  - pinocchio
Flags:
  - profile
  - profile-registries
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Playbook
---

# Migration playbook: legacy profiles.yaml to registry format

## Goal

Convert legacy Pinocchio profile maps:

```yaml
default:
  ai-chat:
    ai-engine: gpt-4o-mini
agent:
  ai-chat:
    ai-engine: gpt-4.1
```

into canonical registry bundle documents:

```yaml
registries:
  default:
    slug: default
    default_profile_slug: default
    profiles:
      default:
        slug: default
        runtime:
          step_settings_patch: ...
```

Then switch your runtime flows to registry-first profile resolution.

Important runtime detail:

- pinocchio runtime YAML sources must be **single-registry** documents (`slug` + `profiles`).
- top-level `registries:` bundle docs are for migration/export/import workflows.

## Before you start

- Keep a backup copy of your current `profiles.yaml`.
- Ensure your app/runtime can read registry-format profile files (current Geppetto/Pinocchio does).
- Confirm your runtime default path:
  - `${XDG_CONFIG_HOME:-~/.config}/pinocchio/profiles.yaml`.

## Step 1: Inspect your current file

Check whether the file is legacy or already canonical.

Legacy shape:

- top-level keys are profile names (`default`, `agent`, ...),
- each value is a layer/setting map.

Canonical shapes:

- registry bundle (`registries:`) for migration/export workflows,
- single-registry runtime YAML (`slug` + `profiles`) for runtime source loading.

## Step 2: Run the migration command

Pinocchio provides a migration command for legacy map input.
Current output is a registry bundle (`registries:`); runtime YAML loading is single-registry only.

Dry run first:

```bash
pinocchio profiles migrate-legacy --dry-run
```

Write to a new output file:

```bash
pinocchio profiles migrate-legacy \
  --input ~/.config/pinocchio/profiles.yaml \
  --output ~/.config/pinocchio/profiles.registry.yaml
```

Overwrite in place with backup:

```bash
pinocchio profiles migrate-legacy \
  --input ~/.config/pinocchio/profiles.yaml \
  --in-place \
  --backup-in-place
```

## Step 3: Verify migrated output

Check that:

- top-level `registries` exists,
- expected profile slugs are present under `registries.<slug>.profiles`,
- each migrated profile has a `runtime.step_settings_patch`,
- output is treated as migration/interchange data (not direct runtime source YAML).

Quick check:

```bash
rg -n "registries:|default_profile_slug:|profiles:" ~/.config/pinocchio/profiles.registry.yaml
```

## Step 4: Produce runtime single-registry YAML

From the migrated bundle (`registries.<slug>...`), choose one registry for runtime and write it as:

```yaml
slug: default
profiles:
  default:
    slug: default
    runtime:
      step_settings_patch:
        ai-chat:
          ai-engine: gpt-4o-mini
  gpt-5:
    slug: gpt-5
    runtime:
      step_settings_patch:
        ai-chat:
          ai-engine: gpt-5
```

Do not include:

- top-level `registries:`
- `default_profile_slug`

Write that runtime YAML to:

- `~/.config/pinocchio/profiles.yaml` (or `${XDG_CONFIG_HOME}/pinocchio/profiles.yaml`).

That enables the implicit default source in pinocchio when `profile-registries` is not set.

If you prefer explicit stack wiring, set:

```yaml
profile-settings:
  profile-registries: ~/.config/pinocchio/profiles.yaml
  profile: default
```

or:

```bash
export PINOCCHIO_PROFILE_REGISTRIES=~/.config/pinocchio/profiles.yaml
export PINOCCHIO_PROFILE=default
```

## Step 5: Validate effective settings

Run a representative command with parsed-parameter output:

```bash
pinocchio --print-parsed-parameters <your-command>
```

Verify profile-derived values are still applied as expected.

## Step 6: Adopt registry-first workflow

After migration:

- Prefer selecting behavior through profiles.
- Keep direct `--ai-engine` / `--ai-api-type` flags for temporary overrides only.
- For runtime-editable apps, move to API/SQLite-backed profile registry flows.

## Optional: Multi-registry setup

If you need separate profile domains (for example `team`, `prod`, `sandbox`), keep each registry in its own YAML or SQLite source and stack them in order:

```yaml
profile-settings:
  profile-registries: ~/.config/pinocchio/team.yaml,~/.config/pinocchio/private.db
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `invalid profile slug` during migration | legacy top-level key is not a valid slug | rename offending profile key to slug-safe value |
| output file exists error | command protects existing outputs | pass `--force`, choose a different `--output`, or use `--in-place` |
| migrated file still uses legacy map shape | command not run or wrong file inspected | rerun migration with explicit `--input` and `--output` |
| runtime ignores migrated file | active `profile-registries` points to another source | update `profile-settings.profile-registries` or `PINOCCHIO_PROFILE_REGISTRIES` |
| runtime startup rejects YAML | file contains top-level `registries:` or `default_profile_slug` | rewrite to single-registry runtime YAML (`slug` + `profiles`) |

## See Also

- [Profile Registry in Geppetto](../topics/01-profiles.md)
- [Geppetto documentation index](../topics/00-docs-index.md)
- [Migration playbook: move to Session/EngineBuilder/ExecutionHandle](04-migrate-to-session-api.md)
