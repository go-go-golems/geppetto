# Tasks

## Completed

- [x] Create a new Geppetto ticket workspace for AllowedTools removal analysis.
- [x] Add a primary design document and a separate Manuel diary document.
- [x] Inventory Geppetto core AllowedTools definitions and enforcement points.
- [x] Inventory app-side registry filtering patterns in Pinocchio, GEC-RAG, and Temporal Relationships.
- [x] Write a detailed intern-oriented analysis, design, and implementation guide.
- [x] Add a ticket-local script to re-run the AllowedTools surface inventory.
- [x] Relate key code files to the main documents.
- [x] Run `docmgr doctor` and fix any documentation issues.
- [x] Upload the document bundle to reMarkable and verify the remote listing.

## Follow-up implementation work

- [x] Remove `AllowedTools` from `geppetto/pkg/inference/tools.ToolConfig`.
- [x] Remove `AllowedTools` from `geppetto/pkg/inference/engine.ToolConfig`.
- [x] Remove `WithAllowedTools`, `IsToolAllowed`, and `FilterTools` from Geppetto tool config helpers.
- [x] Remove provider-engine filtering that depends on `config.AllowedTools`.
- [x] Remove executor-time allowlist checks that depend on `ToolConfig.AllowedTools`.
- [x] Update JS builder options, examples, tests, and docs to stop exposing Geppetto-owned `allowedTools`.
- [x] Update downstream app docs and inspectors that still assume Geppetto persists `allowed_tools` in per-turn tool config.
