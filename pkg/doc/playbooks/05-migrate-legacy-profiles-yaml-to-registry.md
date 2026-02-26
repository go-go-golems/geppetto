---
Title: "Migration playbook: legacy profiles.yaml to runtime registry YAML"
Slug: migrate-legacy-profiles-to-registry
Short: Step-by-step guide to convert legacy profile maps into runtime single-registry YAML and adopt profile-registry stacks.
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

# Migration playbook: legacy profiles.yaml to runtime registry YAML

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

into runtime single-registry YAML:

```yaml
slug: default
profiles:
  default:
    slug: default
    runtime:
      step_settings_patch:
        ai-chat:
          ai-engine: gpt-4o-mini
  agent:
    slug: agent
    runtime:
      step_settings_patch:
        ai-chat:
          ai-engine: gpt-4.1
```

Runtime is hard-cut to one-file-one-registry YAML (`slug` + `profiles`).

## Before you start

- Keep a backup copy of your current `profiles.yaml`.
- Confirm your runtime default path:
  - `${XDG_CONFIG_HOME:-~/.config}/pinocchio/profiles.yaml`.

## Step 1: Inspect your current file

Legacy shape:

- top-level keys are profile names (`default`, `agent`, ...),
- each value is a section/field patch map.

Runtime shape (required):

- top-level `slug`,
- top-level `profiles`,
- no top-level `registries:`,
- no `default_profile_slug`.

## Step 2: Run the migration command

Dry run:

```bash
pinocchio profiles migrate-legacy --dry-run
```

Write to a new file:

```bash
pinocchio profiles migrate-legacy \
  --input ~/.config/pinocchio/profiles.yaml \
  --output ~/.config/pinocchio/profiles.runtime.yaml
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

- top-level `slug` exists,
- top-level `profiles` exists,
- each migrated profile has `runtime.step_settings_patch`,
- file does **not** contain `registries:` or `default_profile_slug:`.

Quick check:

```bash
rg -n "slug:|profiles:|step_settings_patch|registries:|default_profile_slug:" ~/.config/pinocchio/profiles.runtime.yaml
```

## Step 4: Activate runtime source selection

Use explicit profile registry stack config:

```yaml
profile-settings:
  profile-registries: ~/.config/pinocchio/profiles.runtime.yaml
  profile: default
```

or env:

```bash
export PINOCCHIO_PROFILE_REGISTRIES=~/.config/pinocchio/profiles.runtime.yaml
export PINOCCHIO_PROFILE=default
```

If `${XDG_CONFIG_HOME:-~/.config}/pinocchio/profiles.yaml` exists, pinocchio can also load it as implicit default source.

## Step 5: Validate effective settings

Run:

```bash
pinocchio --print-parsed-parameters <your-command>
```

Verify:

- `profile-settings.profile-registries` resolves to your runtime YAML path,
- `profile-settings.profile` resolves to the expected profile,
- profile-derived runtime fields appear in parsed output.

## Optional: Multi-registry stack

Stack sources in order (later entries are higher precedence):

```yaml
profile-settings:
  profile-registries: ~/.config/pinocchio/provider.yaml,~/.config/pinocchio/private.db
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `invalid profile slug` during migration | legacy top-level key is not a valid slug | rename offending profile key to slug-safe value |
| output file exists error | command protects existing outputs | pass `--force`, choose another `--output`, or use `--in-place` |
| runtime rejects YAML | file still has legacy map shape | rerun migration with explicit `--input` and `--output` |
| runtime rejects YAML with `registries`/`default_profile_slug` | non-runtime YAML format | rewrite to runtime single-registry YAML (`slug` + `profiles`) |
| runtime ignores expected profile | active stack points elsewhere | check `--profile-registries`, `PINOCCHIO_PROFILE_REGISTRIES`, and config `profile-settings.profile-registries` |

## See Also

- [Profile Registry in Geppetto](../topics/01-profiles.md)
- [Geppetto documentation index](../topics/00-docs-index.md)
- [Migration playbook: move to Session/EngineBuilder/ExecutionHandle](04-migrate-to-session-api.md)
