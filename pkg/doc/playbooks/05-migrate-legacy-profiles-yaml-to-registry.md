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
  - profile-file
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

into canonical registry documents:

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

## Before you start

- Keep a backup copy of your current `profiles.yaml`.
- Ensure your app/runtime can read registry-format profile files (current Geppetto/Pinocchio does).
- Confirm you know your effective profile path:
  - typically `~/.config/pinocchio/profiles.yaml`
  - or `~/.pinocchio/profiles.yaml` in older setups.

## Step 1: Inspect your current file

Check whether the file is legacy or already canonical.

Legacy shape:

- top-level keys are profile names (`default`, `agent`, ...),
- each value is a layer/setting map.

Canonical shape:

- top-level `registries:` key, or
- single registry document with `slug`, `profiles`, etc.

## Step 2: Run the migration command

Pinocchio provides a command that converts legacy map input into canonical registry YAML.

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

- `registries.default` exists,
- `default_profile_slug` is set,
- expected profile slugs are present under `profiles`,
- each migrated profile has a `runtime.step_settings_patch`.

Quick check:

```bash
rg -n "registries:|default_profile_slug:|profiles:" ~/.config/pinocchio/profiles.registry.yaml
```

## Step 4: Point runtime to migrated file

Set profile file in config:

```yaml
profile-settings:
  profile-file: ~/.config/pinocchio/profiles.registry.yaml
  profile: default
```

Or via environment:

```bash
export PINOCCHIO_PROFILE_FILE=~/.config/pinocchio/profiles.registry.yaml
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

If you need separate profile domains (for example `team`, `prod`, `sandbox`), split into multiple entries under `registries:` and set explicit registry selection where your app resolver supports it.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `invalid profile slug` during migration | legacy top-level key is not a valid slug | rename offending profile key to slug-safe value |
| output file exists error | command protects existing outputs | pass `--force`, choose a different `--output`, or use `--in-place` |
| migrated file has no `registries` key | command not run or wrong file inspected | rerun migration with explicit `--input` and `--output` |
| runtime ignores migrated file | active `profile-file` still points to old path | update `profile-settings.profile-file` or `PINOCCHIO_PROFILE_FILE` |
| behavior changed unexpectedly | default profile slug differs after conversion | set `default_profile_slug` explicitly and verify selected profile |

## See Also

- [Profile Registry in Geppetto](../topics/01-profiles.md)
- [Geppetto documentation index](../topics/00-docs-index.md)
- [Migration playbook: move to Session/EngineBuilder/ExecutionHandle](04-migrate-to-session-api.md)
