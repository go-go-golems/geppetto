# Tasks

## Planning and Design

- [x] Create GP-47 ticket workspace
- [x] Write implementation-plan document for runtime metadata cleanup
- [x] Relate the key JS API files to the ticket index and implementation plan
- [x] Run `docmgr doctor --ticket GP-47-RUNTIME-METADATA-CLEANUP --stale-after 30`

## Execution Tasks

- [x] Define the canonical runtime metadata categories:
  - execution metadata (`systemPrompt`, `middlewares`, `toolNames`)
  - identity metadata (`runtimeKey`, `runtimeFingerprint`, `profileVersion`)
  - inspection-only metadata (`registrySlug`, `profileSlug`, raw profile metadata)
- [x] Add internal helper(s) that materialize execution metadata from resolved profile output
- [x] Add internal helper(s) that stamp runtime identity metadata onto prepared seed turns and run/event context
- [x] Stop requiring callers to manually translate `effectiveRuntime.system_prompt` into middleware
- [x] Stop requiring callers to manually translate profile middleware uses into Go middleware instances
- [x] Stop requiring callers to manually align runtime `tools` metadata with the execution registry
- [x] Add focused tests for runtime metadata materialization and stamping
- [x] Update JS docs so `gp.profiles.resolve(...)` is framed as inspection/advanced resolution, not the normal execution path

## Integration with Opinionated JS API

- [x] Decide whether GP-47 lands before `gp.runner` or as the first slice inside GP-46 implementation
- [x] If kept separate, expose the new internal helpers so GP-46 can reuse them directly
- [x] Add a short migration note showing how GP-47 simplifies the future `gp.runner` implementation
