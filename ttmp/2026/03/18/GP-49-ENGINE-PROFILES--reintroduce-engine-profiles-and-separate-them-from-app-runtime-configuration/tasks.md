# Tasks

## Done

- [x] Finalized the architecture decision: Geppetto owns `EngineProfile` + `InferenceSettings`; apps own prompts, middlewares, tools, and runtime metadata.
- [x] Defined the new Geppetto data model at the design level:
  - `InferenceSettings` rename from `StepSettings`
  - `EngineProfile`
  - `EngineProfileRegistry`
  - `ResolvedEngineProfile`
- [x] Defined the hard-cut YAML format direction for engine profile registries with no `runtime:` section and no patch maps.
- [x] Defined the app-owned runtime boundary explicitly:
  - system prompt
  - middlewares
  - tools / filtered registries
  - runtime key / runtime fingerprint
  - cache and identity policy
- [x] Wrote the main intern-oriented architecture and migration guide.
- [x] Added a dedicated migration playbook to `glazed/pkg/doc/tutorials/` describing the hard cut.
- [x] Updated diary, changelog, and ticket index to reflect the architecture pivot and documentation deliverables.

## Next Implementation Planning Tasks

## Implementation Slices

- [x] Slice 1: hard-rename `pkg/profiles` to `pkg/engineprofiles`
  - move the package directory
  - rename the Go package clause
  - update imports in geppetto, pinocchio, gec-rag, and temporal-relationships
  - keep behavior unchanged in this slice
  - validate with focused package tests plus cross-repo compile/test smoke
- [x] Slice 2: hard-rename `StepSettings` to `InferenceSettings`
  - rename the core type in Geppetto
  - rename constructors such as `NewStepSettings` and `NewStepSettingsFromParsedValues`
  - rename engine factory helpers that still mention step settings
  - update imports and call sites in all downstream repos
  - update docs/examples/tests that still teach `StepSettings`
- [x] Slice 3: rename profile-resolution surface to engine-profile terminology
  - `Profile` -> `EngineProfile`
  - `ProfileRegistry` -> `EngineProfileRegistry`
  - `ResolvedProfile` -> `ResolvedEngineProfile`
  - `ResolveEffectiveProfile(...)` -> `ResolveEngineProfile(...)`
  - keep mixed runtime payload temporarily if needed until Slice 4
- [ ] Slice 4: remove mixed runtime payload from the renamed engineprofiles package
  - delete `RuntimeSpec`
  - remove runtime stack merge semantics
  - remove runtime fingerprinting from Geppetto profile resolution
  - change resolution output to final `InferenceSettings`
- [ ] Slice 5: rewrite engine profile YAML and codecs
  - define engine-only YAML shape
  - update YAML decode/encode tests
  - update SQLite store payload expectations if needed
- [ ] Slice 6: move remaining prompt/middleware/tool runtime semantics fully to applications
  - Pinocchio runtime config
  - GEC-RAG runtime config
  - Temporal Relationships runtime config
  - JS host/runtime glue
- [ ] Slice 7: doc and migration cleanup
  - Geppetto docs
  - Pinocchio docs
  - JS docs/examples
  - Glazed migration playbook follow-up

## Validation Requirements

- [x] Geppetto unit tests pass after each slice
- [x] Pinocchio focused tests or compile smoke pass after cross-repo rename slices
- [x] GEC-RAG focused tests or compile smoke pass after cross-repo rename slices
- [x] Temporal Relationships focused tests or compile smoke pass after cross-repo rename slices
- [x] `docmgr doctor --ticket GP-49-ENGINE-PROFILES --stale-after 30` passes after each documentation update
- [ ] Commit each completed slice separately and record it in the diary and changelog
