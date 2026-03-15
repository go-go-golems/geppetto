# Tasks

## Completed Analysis And Delivery

- [x] Create GP-33 ticket workspace under `geppetto/ttmp`.
- [x] Map the current scoped database tool pattern in `temporal-relationships`.
- [x] Map the relevant Geppetto and Pinocchio tool/runtime abstractions.
- [x] Update the design guide to keep `Meta` in the proposed build result and make the tool-definition shape explicit.
- [x] Record the investigation in a ticket diary.
- [x] Validate the documentation ticket with `docmgr doctor`.
- [x] Upload the documentation bundle to reMarkable.

## Geppetto Extraction Tasks

- [x] Create `geppetto/pkg/inference/tools/scopeddb` package skeleton.
- [x] Port generic SQLite schema/bootstrap helpers out of `temporal-relationships/internal/extractor/scopeddb`.
- [x] Add shared `QueryInput`, `QueryOutput`, `QueryOptions`, and `BuildResult[Meta]` types.
- [x] Implement shared read-only query validation and SQLite authorizer enforcement.
- [x] Implement shared tool-description rendering from grouped tool-definition metadata plus allowlisted objects.
- [x] Implement `BuildInMemory` and `BuildFile` helpers that keep `Meta` and cleanup behavior.
- [x] Implement `RegisterPrebuilt` helper for already-built scoped DBs.
- [x] Implement `NewLazyRegistrar` helper for context-resolved scopes.
- [x] Add Geppetto tests for schema bootstrap, read-only query safety, description rendering, prebuilt registration, and lazy registration.

## Temporal Migration Tasks

- [x] Replace `internal/extractor/scopeddb` imports in `transcripthistory` with the new Geppetto package.
- [x] Replace `internal/extractor/scopeddb` imports in `entityhistory` with the new Geppetto package.
- [x] Replace `internal/extractor/scopeddb` imports in `runturnhistory` with the new Geppetto package.
- [x] Refactor `transcripthistory/query.go` to wrap the shared Geppetto query runner.
- [x] Refactor `entityhistory/query.go` to wrap the shared Geppetto query runner.
- [x] Refactor `runturnhistory/query.go` to wrap the shared Geppetto query runner.
- [x] Refactor `transcripthistory/tool.go` to register through the shared Geppetto helper.
- [x] Refactor `entityhistory/tool.go` to register through the shared Geppetto helper.
- [x] Refactor `runturnhistory/tool.go` to render descriptions and lazy registration through the shared Geppetto helper.
- [x] Refactor `BuildScopedDBFile` callers to use the shared Geppetto file-build helper where appropriate.
- [x] Replace duplicated run-chat tool factory logic with the shared lazy registrar helper.
- [x] Replace duplicated gorunner tool registration logic with the shared prebuilt registration helper.
- [x] Remove or delete the now-obsolete `temporal-relationships/internal/extractor/scopeddb` package once no imports remain.

## Verification Tasks

- [x] Run targeted Geppetto tests for the new package.
- [x] Run targeted `temporal-relationships` scoped-db/query/runtime tests after migration.
- [x] Run any additional smoke tests needed for build-file snapshot generation.
- [x] Re-run `docmgr doctor` for GP-33 after implementation notes are written.

## Delivery Tasks

- [x] Update the GP-33 design guide with the implemented API and any deviations from the proposal.
- [x] Add detailed diary entries for each implementation phase, including exact commands and failures.
- [x] Commit the Geppetto extraction in coherent repository-local commits.
- [x] Commit the `temporal-relationships` migration in coherent repository-local commits.
- [x] Commit the GP-33 ticket/doc updates in the Geppetto repository.
- [x] Refresh the reMarkable bundle with the implementation updates.
