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

## Pivot

- [x] Record the GP-41 pivot toward a read-only registry architecture in a dedicated implementation-plan document.
- [x] Slice 1: remove `ProfilePatch`, `WriteOptions`, and `RegistryWriter` from Geppetto `pkg/profiles`.
- [x] Slice 1: remove `CreateProfile`, `UpdateProfile`, `DeleteProfile`, and `SetDefaultProfile` from `StoreRegistry` and `ChainedRegistry`.
- [x] Slice 1: remove mutation-specific Geppetto tests that exercise writable registry service behavior.
- [x] Slice 2: remove `PolicySpec`, `Profile.Policy`, and policy-related stack merge / trace / validation code.
- [x] Slice 2: remove policy-specific errors that only existed for writable or override-gated registry behavior.
- [x] Slice 3: remove Geppetto JS profile mutation APIs and mutation-related TypeScript declarations.
- [x] Slice 3: keep JS stack-connect helpers, but make them operate against read-only registries only.
- [x] Slice 4: remove Pinocchio profile CRUD/default HTTP endpoints or reduce the profile API to a read-only surface.
- [x] Slice 4: remove downstream uses of `PolicyViolationError`, `RegistryWriter`, and mutation-only fake registries in Pinocchio and GEC-RAG tests.
- [ ] Slice 5: remove stale JS examples, docs, and API references that still describe profile CRUD, profile policy, or request overrides.
- [ ] Slice 5: run focused validation in Geppetto, Pinocchio, and GEC-RAG once the read-only registry pivot is fully implemented.

## Superseded Tasks

- [ ] Old Phase 2 policy-shrink plan is superseded by deleting `PolicySpec` entirely.
- [ ] Old Phase 3 downstream cleanup plan is superseded by the larger read-only registry pivot.
