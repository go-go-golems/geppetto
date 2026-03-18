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
- Landed downstream fallout fix `pinocchio` `82bf805` `drop dead profile flag bridge call`:
  - removed the leftover `GatherFlagsFromProfileRegistry` call from Pinocchio command helpers after the Geppetto bridge deletion,
  - revalidated `go build ./...` in Pinocchio.
- Landed `geppetto` `c37f952` `clean gp-45 stale docs and js typings`:
  - removed stale profile-mutation and migration-escape-hatch wording from docs and help text,
  - removed the dead `allowedTools` option from JS typings and API reference docs,
  - clarified the SQLite playbook around read-only registry operation and schema inspection.
- Landed `geppetto` `5ec524f` `remove dead chained registry fields`:
  - removed the unused `aggregateStore` and `registryOwners` fields from `ChainedRegistry`,
  - confirmed the fields were construction-only bookkeeping with no live readers.
- Landed `geppetto` `6f08791` `remove dead source owner scaffolding`:
  - removed the unused `label` and `service` fields from `sourceOwner`,
  - removed the now-dead `StoreRegistry` construction from YAML and SQLite source opening paths.
- Landed `geppetto` `70aa268` `remove dead store registry codec hook`:
  - removed the unused `StoreRegistry.extensionCodecs` field and `WithExtensionCodecRegistry(...)` option,
  - removed the now-dead test that only asserted service-level codec wiring.
- Landed `geppetto` `a68c313` `remove dead normalize profile extensions helper`:
  - removed `NormalizeProfileExtensions(...)` from `pkg/profiles/extensions.go`,
  - removed the dedicated tests that only exercised that dead normalization/codec-decode path.
- Landed `geppetto` `a1f2f56` `simplify extension codec registry to lister`:
  - removed the dead `Lookup(...)` method from `ExtensionCodecRegistry`,
  - collapsed the Geppetto schema-listing path onto the lister-only contract.
- Landed downstream follow-up `pinocchio` `29e25c7` `simplify extension schema registry contract`:
  - removed the old "registry without lister" test shape,
  - simplified the profile API schema listing path to the same lister-only contract.
- Confirmed that the inference/runtime metadata compatibility paths are not dead yet:
  - `MirrorLegacyInferenceKeys` is still part of active engine persistence flow,
  - `AddRuntimeAttributionToExtra` is still used by provider engines,
  - Pinocchio still reads older runtime metadata variants in persistence code.
