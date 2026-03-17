# Tasks

## Completed

- [x] Create a new Geppetto ticket workspace for override removal analysis.
- [x] Add a primary design document and a separate Manuel diary document.
- [x] Inventory Geppetto request override surfaces in core profile resolution, policy handling, validation, tests, docs, and JS bindings.
- [x] Inventory Pinocchio, GEC-RAG, and Temporal Relationships usage of resolved profiles versus request overrides.
- [x] Write a detailed intern-oriented analysis, design, and implementation guide.
- [x] Add a dedicated runtime glossary explaining the overloaded `Runtime*` terminology across Geppetto and Pinocchio with concrete examples.
- [x] Add a ticket-local script to re-run the override surface inventory.
- [x] Relate key code files to the main documents.
- [x] Run `docmgr doctor` and fix any documentation issues.
- [x] Upload the document bundle to reMarkable and verify the remote listing.

## Follow-up implementation work

- [x] Phase 1: remove `RequestOverrides` from `geppetto/pkg/profiles.ResolveInput`.
- [x] Phase 1: remove the `step_settings_patch` override key from Geppetto override parsing and policy handling.
- [x] Phase 1: remove override-specific merge logic from `resolveRuntimeSpec` in `geppetto/pkg/profiles/service.go`.
- [x] Phase 1: remove override-specific validation and helper parsing from Geppetto profile service tests.
- [x] Phase 1: make Geppetto JS bindings and generated type definitions reject or stop advertising `requestOverrides`.
- [ ] Phase 2: remove override-specific policy fields or hard-deprecate them in `PolicySpec`.
- [ ] Phase 2: update stack merge / stack trace tests and docs that still mention allowed or denied override keys.
- [ ] Phase 3: remove request override support from Geppetto JS bindings and JS examples.
- [ ] Phase 3: remove `request_overrides` from Pinocchio web chat HTTP request contracts and resolution logic.
- [ ] Phase 3: remove `request_overrides` from GEC-RAG chat request contracts and resolution logic.
- [ ] Phase 3: confirm Temporal Relationships stays unaffected because it already does not expose request overrides on its HTTP surface.
- [ ] Phase 4: update tests, README files, API docs, and playbooks that still describe override behavior.
- [ ] Phase 4: run focused validation in Geppetto, Pinocchio, and GEC-RAG once the override path is fully removed.
