---
Title: "Bootstrap binary StepSettings from defaults, config, registries, and profile"
Slug: bootstrap-binary-step-settings-defaults-config-registries-profile
Short: Applications now bootstrap StepSettings from defaults/config/env/flags only; profiles no longer patch them.
Topics:
  - configuration
  - profiles
  - step-settings
  - glazed
Commands:
  - geppetto
  - pinocchio
Flags:
  - config-file
  - profile
  - profile-registries
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Bootstrap binary StepSettings from defaults, config, registries, and profile

This playbook changed with the StepSettingsPatch hard cut.

Current rule:

1. Build `StepSettings` from schema defaults, config files, env, and flags.
2. Resolve profile registries separately for runtime metadata (`system_prompt`, `tools`, `middlewares`).
3. Build engines from the app-owned final `StepSettings`.

Profiles no longer produce final `StepSettings`, and `ResolveEffectiveProfile` no longer returns `effectiveStepSettings`.
