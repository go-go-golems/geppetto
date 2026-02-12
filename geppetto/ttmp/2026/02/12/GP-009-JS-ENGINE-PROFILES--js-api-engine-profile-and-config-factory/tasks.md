# Tasks

## TODO


- [x] Audit existing profile/config engine constructors and choose reusable Go entry points
- [x] Define JS API shape for engines.fromProfile(profile, opts) and engines.fromConfig(opts)
- [x] Define precedence rules between explicit args, opts, PINOCCHIO_PROFILE, and defaults
- [x] Implement profile resolution and error messages for unknown/invalid profiles
- [x] Implement config-to-engine translation for provider/model/timeout/tool-related settings
- [x] Add support for safe overrides (model, temperature, maxTokens, timeout)
- [x] Add tests for valid profile resolution and failure paths
- [x] Add integration test for runInference/createSession using fromProfile
- [x] Create smoke script that runs real inference with PINOCCHIO_PROFILE=gemini-2.5-flash-lite
- [x] Run tests/script and update diary/changelog with exact command outputs
