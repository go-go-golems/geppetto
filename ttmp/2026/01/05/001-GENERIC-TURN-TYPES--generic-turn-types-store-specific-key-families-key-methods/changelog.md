# Changelog

## 2026-01-05

- Initial workspace created


## 2026-01-05

Bootstrap: create ticket workspace, add analysis+diary docs, relate key files, and seed task list.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2026/01/05/001-GENERIC-TURN-TYPES--generic-turn-types-store-specific-key-families-key-methods/analysis/01-analysis-implement-store-specific-key-families-key-methods.md — Defines target API and migration plan
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2026/01/05/001-GENERIC-TURN-TYPES--generic-turn-types-store-specific-key-families-key-methods/reference/01-diary.md — Records ongoing investigation and decisions


## 2026-01-05

Plan: replace Key[T]+function API with store-specific key families + key methods; no backwards compatibility; add detailed task breakdown.

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/ttmp/2026/01/05/001-GENERIC-TURN-TYPES--generic-turn-types-store-specific-key-families-key-methods/tasks.md — Defines no-BC implementation/migration steps


## 2026-01-05

Implement DataKey/TurnMetaKey/BlockMetaKey + key.Get/key.Set methods (tasks 2-3) (commit 583343b)

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/key_families.go — New production key families + methods
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/poc_split_key_types_test.go — Use production key families; keep behavior contract tests


## 2026-01-05

Migrate canonical keys to DataK/TurnMetaK/BlockMetaK and rewrite geppetto call sites to key methods (commit c07a9f1)

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/inference/engine/turnkeys.go — Engine escape-hatch key now uses turns.DataK
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/inference/middleware/systemprompt_middleware.go — Example call site migrated to key.Set/key.Get
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/keys.go — Canonical keys now use store-specific key families
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/types.go — Legacy function API now takes store-specific key families


## 2026-01-05

Moments: migrate to turns wrapper stores + key methods; run turnsrefactor; go test + make lint (commits af80d5f, 08707ed)

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/moments/backend/pkg/memory/middleware.go — Refactor legacy turns.*Get/*Set calls to key methods
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/moments/backend/pkg/webchat/moments_global_prompt_middleware.go — Replace Has/WithBlockMetadata; use BlockMetaWebchatID/Section
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/moments/backend/pkg/webchat/router.go — Remove map-style Turn.Data usage; use turnkeys + wrapper stores


## 2026-01-05

Validation: geppetto go test ./... -count=1 (task 22)

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/types.go — Turns API changes validated by full test run


## 2026-01-05

Turns: add NewKeyString + store-typed key constructors; check off task 5; confirm no turns.K call sites remain

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/key_families.go — TurnMetaK/BlockMetaK now use typed constructors
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/types.go — NewKeyString + typed key-id constructors


## 2026-01-05

Turns: delete legacy Key/K and turns.*Get/*Set APIs (task 16); update moments tests; keep go test green

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/turns/types.go — Remove legacy Key/K and DataGet/DataSet/etc
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/moments/backend/pkg/promptutil/resolve_draft_test.go — Replace turns.DataSet/DataGet with key methods


## 2026-01-05

Post-cut cleanup: check off tasks 17-21 (docs/tooling/lint). Update turnsrefactor verify (82b1913), extend turnsdatalint constructor policy + tests (c275286), refresh turns docs (1dc3760).

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/analysis/turnsdatalint/analyzer.go — Ban turns.DataK/TurnMetaK/BlockMetaK outside key files
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/analysis/turnsrefactor/refactor.go — Verify scans all compiled files post-cut
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/pkg/doc/topics/08-turns.md — Docs updated for wrapper stores + key families


## 2026-01-05

Exclude testdata/ from golangci-lint + gosec (geppetto 92d077c; pinocchio baad607)

### Related Files

- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/.golangci.yml — Skip testdata dirs for golangci-lint
- /home/manuel/workspaces/2025-12-19/use-strong-turn-data-access/geppetto/Makefile — Exclude testdata from gosec

