# Changelog

## 2026-03-17

- Initial workspace created
- Added a detailed design and implementation ticket for removing `StepSettingsPatch` from Geppetto profile runtime.
- Added an explicit phase-by-phase task list covering Geppetto core changes, downstream caller migration, docs, and validation.
- Added a second design document that defines the ideal app-facing API and argues for a single hard cut that removes both `StepSettingsPatch` and `RuntimeKeyFallback`.
- Updated the original GP-43 guide, index, and task board to reflect the new decision: no backward-compatibility adapter, caller-owned runtime identity, and `RuntimeKeyFallback` folded into the same implementation series.
- Completed the first implementation slice in downstream callers: Pinocchio and GEC-RAG no longer populate `RuntimeKeyFallback` in their live webchat resolver paths.
- Removed stale GEC-RAG runtime-composition dependence on `ComposedRuntime.AllowedTools`; tool registry filtering now reads directly from the incoming resolved runtime request instead of from composed engine artifacts.
- Completed the second implementation slice in Geppetto core and JS surfaces: `RuntimeKeyFallback` is removed from `profiles.ResolveInput`, profile resolution always derives `runtimeKey` from `profileSlug`, and the JS profile/engine APIs plus examples/docs were updated to match the hard cut.
- Completed the third implementation slice in downstream callers: Pinocchio webchat and GEC-RAG now carry fully resolved `StepSettings` through request/runtime contracts, and their runtime composers no longer apply `StepSettingsPatch` themselves.
- Completed downstream-prep cleanup so no remaining live caller path expects Geppetto-owned `EffectiveStepSettings`:
  - `pinocchio` `64d2f39` `own runtime step settings outside profile resolution`
  - `2026-03-16--gec-rag` `722f41d` `stop expecting profile-resolved step settings`
  - `temporal-relationships` `94b0c8a` `build extractor engines from app settings`
- Completed the Geppetto hard cut:
  - `geppetto` `6aecbcc` `remove profile step settings patches`
  - removed `RuntimeSpec.StepSettingsPatch`, `ResolveInput.BaseStepSettings`, `ResolvedProfile.EffectiveStepSettings`, and `runtime_settings_patch_resolver.go`
  - rewrote stack merge/trace, sections glue, JS bindings, generated types, examples, tests, and docs around runtime-metadata-only profiles
- Completed downstream fallout cleanup and legacy-migration hard errors:
  - `pinocchio` `b75c069` `drop legacy profile patch assumptions`
  - `temporal-relationships` `d5ac736` `update runtime profile fixtures`
  - `2026-03-16--gec-rag` `b444744` `document runtime metadata-only profiles`
