# Changelog

## 2026-03-17

- Initial workspace created.
- Added the broad cleanup analysis and phased implementation guide.
- Added the investigation diary capturing the evidence-gathering process and the classification of findings.
- Organized the follow-up work into low-risk hard cuts, compatibility cleanup, migration-shim cleanup, and over-complex/lightly integrated machinery review.

## 2026-03-17

Added broad legacy and unused functionality cleanup analysis, phased task plan, and investigation diary.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions.go — Extension cleanup risk and uncertainty documented
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go — Compatibility cleanup candidate documented

## 2026-03-17

- Converted the broad GP-45 audit into a concrete implementation queue with low-risk removals first and compatibility cleanup explicitly gated by live usage checks.
- Landed `geppetto` `e10d584` `remove js engines fromprofile api`:
  - removed the dead no-op `GatherFlagsFromProfileRegistry` bridge and its tests,
  - removed `pkg/profiles/adapters.go`,
  - removed `gp.engines.fromProfile` entirely from the JS module surface,
  - removed the dedicated dead-API example and updated JS docs/tests/examples accordingly.
- Landed `geppetto` `1fd2013` `narrow mutable profile store api`:
  - removed `DeleteProfile` and `SetDefaultProfile` from `ProfileStore`,
  - removed those methods from memory/sqlite/yaml store implementations,
  - rewrote the affected store tests around explicit registry upserts instead of hidden mutation helpers.
- Confirmed that the inference/runtime metadata compatibility paths are not dead yet:
  - `MirrorLegacyInferenceKeys` is still part of active engine persistence flow,
  - `AddRuntimeAttributionToExtra` is still used by provider engines,
  - Pinocchio still reads older runtime metadata variants in persistence code.
