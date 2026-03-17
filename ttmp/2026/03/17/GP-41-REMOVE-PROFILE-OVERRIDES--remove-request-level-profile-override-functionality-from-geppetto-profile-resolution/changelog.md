# Changelog

## 2026-03-17

- Created ticket `GP-41-REMOVE-PROFILE-OVERRIDES` for removing request-level profile override functionality from Geppetto profile resolution.
- Added the primary design doc and a separate Manuel diary.
- Completed an evidence pass over Geppetto core profile resolution, policy merge, validation, JS bindings, Pinocchio web chat, GEC-RAG chat resolution, and Temporal Relationships run-chat.
- Wrote a detailed implementation guide explaining the current architecture, why request overrides add complexity, what should be removed, and how to execute the change safely.
- Added a dedicated runtime glossary document explaining the major `Runtime*` symbols across Geppetto and Pinocchio, plus concrete example flows for profile resolution, JS timeline loading, and runtime attribution.
- Added a ticket-local inventory script for quickly re-scanning override-related code paths.
- Related the key evidence files to the ticket documents.
- Validated the ticket docs with `docmgr doctor`.
- Uploaded the bundle to reMarkable and verified the remote destination.
- Implemented GP-41 Phase 1 in Geppetto core by removing `ResolveInput.RequestOverrides`, deleting request-override runtime mutation and helper parsing from the profile service, and pruning the override-specific profile service tests.
- Updated Geppetto JS bindings to reject `requestOverrides` at runtime and removed the stale `requestOverrides` fields from the generated TypeScript declaration surface.
- Recorded a GP-41 implementation pivot: the target architecture is now a read-only profile registry service, not just override removal plus a smaller `PolicySpec`.
- Added a dedicated implementation-plan document for the read-only registry pivot and rewrote the task board around the new slice structure.
- Implemented the first read-only-registry code slice in Geppetto core and committed it as `afa1ca8` (`make profile registries read-only`).
- Removed writable registry abstractions and service methods from Geppetto, including `RegistryWriter`, `WriteOptions`, `ProfilePatch`, and the service-level profile CRUD/default APIs.
- Deleted `PolicySpec`, `Profile.Policy`, policy-specific errors, and the corresponding stack merge, trace, validation, JS surface, and test coverage that only existed for mutable or override-gated profile behavior.
- Reduced the Geppetto JS profile namespace to a read-only surface while keeping stack-connect helpers available.
- Validated the Geppetto slice with focused package tests, `make lint`, and full `go test ./...`.
- Updated Pinocchio and committed `6f28f1d` (`make webchat profile APIs read-only`), reducing the shared webchat profile API to list/get/current-profile routes only, deleting request-override handling from the request resolver path, and removing writable/profile-policy tests.
- Added a small Pinocchio follow-up commit `ec06d8d` (`clean up read-only profile api lint fallout`) after `make lintmax` surfaced dead helpers and formatting drift left behind by the large profile API deletion.
- Updated GEC-RAG and committed `3115931` (`remove request overrides from coinvault webchat`), deleting request-override plumbing from CoinVault request resolution, runner payloads, and test doubles.
- Confirmed focused downstream validation with `go test ./pkg/webchat/... ./cmd/web-chat/...` in Pinocchio and `go test ./internal/webchat/...` in GEC-RAG.
