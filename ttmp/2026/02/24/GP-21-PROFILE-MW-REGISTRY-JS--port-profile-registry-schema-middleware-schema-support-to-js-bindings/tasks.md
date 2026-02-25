# Tasks

## Completed

- [x] Create GP-21 ticket workspace and seed required docs.
- [x] Perform source-level architecture scan for profile registry, middleware config schema support, and JS bindings.
- [x] Capture runtime evidence via reproducible scripts under `scripts/`.
- [x] Write comprehensive design doc with gap analysis, API sketches, phased implementation plan, and risks.
- [x] Produce unified inference-first final JS API research doc merging GP-21 and OS-09 proposals.
- [x] Convert final recommendation to hard cutover (remove legacy profile semantics).
- [x] Add a large JS scripts cookbook document covering old/current and new/hard-cutover APIs.
- [x] Maintain chronological investigation diary with exact commands/findings.
- [x] Run `docmgr doctor` and resolve ticket-level validation issues.
- [x] Upload bundled ticket docs to reMarkable after dry-run and verify remote listing.

## Follow-up Implementation (Not Executed in this research ticket)

- [ ] Implement `gp.profiles` namespace in `pkg/js/modules/geppetto`.
- [ ] Implement `gp.schemas` namespace in `pkg/js/modules/geppetto`.
- [ ] Extend JS module `Options` to inject profile registry and schema providers.
- [ ] Extend JS type declarations and docs for new namespaces.
- [ ] Add JS module unit/integration tests for profile/schema APIs.
