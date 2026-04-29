---
Title: "Migration playbook: legacy profiles.yaml to runtime registry YAML"
Slug: migrate-legacy-profiles-to-registry
Short: Legacy profile-map migration has been removed after the StepSettingsPatch hard cut.
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
SectionType: Tutorial
---

# Migration playbook: legacy profiles.yaml to runtime registry YAML

Automatic migration of legacy `profiles.yaml` maps has been removed.

Why:

- legacy profile maps only described engine-setting overlays,
- `runtime.step_settings_patch` no longer exists,
- applications now own final `StepSettings` and provider credentials outside the profile registry.

What to do instead:

1. Create a new single-registry YAML file.
2. Rebuild each profile around `runtime.system_prompt`, `runtime.tools`, and `runtime.middlewares`.
3. Move model/provider/client settings into app config, env, or explicit `fromConfig(...)` calls.

Minimal example:

```yaml
slug: default
profiles:
  analyst:
    slug: analyst
    runtime:
      system_prompt: You are an analyst.
      tools:
        - calculator
```
