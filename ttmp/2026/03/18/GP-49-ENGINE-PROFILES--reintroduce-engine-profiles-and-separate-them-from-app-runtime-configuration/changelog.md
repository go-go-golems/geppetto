# Changelog

## 2026-03-18

- Initial workspace created
- Added the primary design direction:
  - reintroduce Geppetto-owned engine-only profiles
  - rename `StepSettings` to `InferenceSettings`
  - hard-cut the mixed runtime profile model with no compatibility wrappers
  - move prompts, middlewares, tools, and runtime identity fully to application-owned runtime config
- Added a detailed implementation task board.
- Added the intern-facing design and migration guide.
- Added the first diary entry documenting the architectural pivot and evidence used.
- Captured the requirement for a separate migration playbook in `glazed/pkg/doc/...` as part of rollout.
- Added the first actual Glazed migration playbook draft at `glazed/pkg/doc/tutorials/migrating-from-mixed-runtime-profiles-to-engine-profiles.md`.
- Split the task board into completed design/documentation work and remaining implementation-planning work.
- Added a concrete section on the current profile-registry subsystem explaining which mechanics survive, which mixed-model semantics are removed, and where a package rename to `pkg/engineprofiles` may make sense.
- Replaced the high-level planning bullets with concrete implementation slices, starting with the hard package rename and the `StepSettings` to `InferenceSettings` rename.
- Completed Slice 1:
  - moved `geppetto/pkg/profiles` to `geppetto/pkg/engineprofiles`
  - renamed the Go package clause to `engineprofiles`
  - updated imports across Geppetto, Pinocchio, GEC-RAG, and Temporal Relationships
  - validated with focused tests in all four repos
