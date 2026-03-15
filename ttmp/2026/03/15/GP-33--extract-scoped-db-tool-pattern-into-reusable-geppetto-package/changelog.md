# Changelog

## 2026-03-15

- Created the GP-33 ticket workspace under `geppetto/ttmp`.
- Added the primary design document describing the current pattern, the extraction boundary, the proposed Geppetto package, and the file-level migration plan.
- Added the investigation diary documenting the research steps, command trail, and delivery details.
- Validated the ticket with `docmgr doctor --root geppetto/ttmp --ticket GP-33 --stale-after 30`.
- Uploaded the final bundle to reMarkable at `/ai/2026/03/15/GP-33`.
- Implemented the reusable Geppetto `pkg/inference/tools/scopeddb` package in commit `f79f77b`.
- Migrated `temporal-relationships` history packages onto the shared Geppetto package in commits `ba7cfcb` and `eaad1be`.
- Removed the obsolete `temporal-relationships/internal/extractor/scopeddb` package after all imports were eliminated.
- Replaced run-chat tool factory registration with shared lazy scoped-db registrars while keeping the direct query helpers for testability.

## 2026-03-15

Completed the GP-33 analysis package: created the ticket, mapped the current scoped database tool pattern across temporal-relationships/geppetto/pinocchio, wrote the intern-facing design guide, and prepared the ticket bundle for validation and reMarkable delivery.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/inference/tools/definition.go — Framework tool surface analyzed for extraction target
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/pinocchio/pkg/webchat/router.go — App-level tool registration integration point
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/httpapi/run_chat_transport.go — Current lazy scoped DB factory call site
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/scopeddb/schema.go — Current repo-private scoped DB helper layer

## 2026-03-15

Implemented the shared Geppetto scopeddb package, migrated temporal-relationships onto thin dataset specs over it, removed the old internal scopeddb package, and switched run-chat tool registration to the shared lazy registrar path (commits f79f77b, ba7cfcb, eaad1be).

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/inference/tools/scopeddb/schema.go — Implemented shared scopeddb API
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/entityhistory/spec.go — App-owned dataset spec migration
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/httpapi/run_chat_transport.go — Shared lazy registrar adoption in run-chat transport

