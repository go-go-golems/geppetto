# Changelog

## 2026-03-17

- Initial workspace created
- Added a detailed design and implementation ticket for removing `StepSettingsPatch` from Geppetto profile runtime.
- Added an explicit phase-by-phase task list covering Geppetto core changes, downstream caller migration, docs, and validation.
- Added a second design document that defines the ideal app-facing API and argues for a single hard cut that removes both `StepSettingsPatch` and `RuntimeKeyFallback`.
- Updated the original GP-43 guide, index, and task board to reflect the new decision: no backward-compatibility adapter, caller-owned runtime identity, and `RuntimeKeyFallback` folded into the same implementation series.
- Completed the first implementation slice in downstream callers: Pinocchio and GEC-RAG no longer populate `RuntimeKeyFallback` in their live webchat resolver paths.
- Removed stale GEC-RAG runtime-composition dependence on `ComposedRuntime.AllowedTools`; tool registry filtering now reads directly from the incoming resolved runtime request instead of from composed engine artifacts.
