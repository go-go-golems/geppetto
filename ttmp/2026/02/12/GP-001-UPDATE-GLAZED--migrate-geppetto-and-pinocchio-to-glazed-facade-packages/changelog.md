# Changelog

## 2026-02-12

- Initial workspace created.
- Added analysis doc: `analysis/01-migration-analysis-old-glazed-to-facade-packages-geppetto-then-pinocchio.md`.
- Added diary doc: `reference/01-diary.md`.
- Captured baseline validation logs and migration inventories under `sources/local/`:
  - legacy imports/symbols/tags/signatures
  - `make test` + `make lint` outputs for geppetto and pinocchio
  - failure extracts and per-repo count breakdowns
- Documented ordered implementation strategy: geppetto migration first, pinocchio second.

## 2026-02-12

Completed baseline migration analysis: validated glazed facade APIs, captured geppetto/pinocchio test+lint failures, generated exhaustive file/symbol inventories, and documented phased plan (geppetto first, then pinocchio).

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/analysis/01-migration-analysis-old-glazed-to-facade-packages-geppetto-then-pinocchio.md — Primary migration analysis deliverable
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/reference/01-diary.md — Detailed implementation diary
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/14-failure-extracts.txt — Baseline failure evidence for planning

## 2026-02-12

Completed Pinocchio Phase 2 Task 1 by migrating `pkg/cmds/*` core command model/loader flow to `schema/fields/sources/values` in pinocchio commit `acd8533`, then recorded focused validation and blockers in ticket artifacts.

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/cmds/cmd.go — values-based command runtime and default variable extraction
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/cmds/helpers/parse-helpers.go — source middleware migration for profile/config/env/default parsing
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/23-pinocchio-pkg-cmds-focused-pass.txt — focused package test evidence
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/24-pinocchio-pkg-cmds-helpers-blocker.txt — current missing-geppetto import blocker

## 2026-02-12

Completed Pinocchio Phase 2 Task 2 by migrating command implementation files under `cmd/pinocchio/cmds/*` to facade APIs (`fields`, `values`, `sources`) in pinocchio commit `826ba63`, including a local `pkg/filefilter` shim to decouple `catter` from `clay`'s legacy layer API.

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/pinocchio/cmds/openai/openai.go — command section wiring and values-based decode migration
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/pinocchio/cmds/catter/cmds/print.go — catter command migration plus filefilter integration via local package
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/filefilter/section.go — facade-compatible section definition replacing layer-based parser dependency
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/26-pinocchio-cmd-impl-legacy-scan-after-task2.txt — post-migration local legacy-symbol scan
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/28-pinocchio-cmd-impl-aggregate-blockers.txt — aggregate command package blockers (`conversation`, `prompto`)

## 2026-02-12

Completed Pinocchio Phase 2 Task 3 by migrating webchat + redis settings decode paths to facade APIs in pinocchio commit `bc94338`.

### Related Files

- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/redisstream/redis_layer.go — redis section migrated from layer/parameter definitions to `schema.WithFields`
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/pkg/webchat/router.go — router settings and redis settings decode switched to `values.DecodeSectionInto`
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/pinocchio/cmd/web-chat/main.go — command flags migrated to `fields.New` and `WithSections`
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/29-pinocchio-webchat-redis-legacy-scan-after-task3.txt — post-migration scope scan
- /home/manuel/workspaces/2026-02-11/geppetto-glazed-bump/geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/31-pinocchio-webchat-focused-blocker.txt — remaining `toolhelpers` blocker in `pkg/webchat`
