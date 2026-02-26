# Changelog

## 2026-02-20

- Initial workspace created
- Added detailed analysis of the `pkg/turns/keys.go` merge conflict and root-cause (manual constants vs generated constants source of truth).
- Added recommended unification strategy: schema-first updates in `pkg/turns/spec/turns_codegen.yaml` with regenerated `keys_gen.go` and `turns.d.ts`.
- Added implementation task list focused on merge conflict resolution and validation.
- Added separate analysis comparing `cmd/gen-js-api` and `cmd/gen-turns`, with a concrete unification design and generated `BlockMetadataKeys`/turns-key exports for `gp.consts`.
- Resolved `pkg/turns/keys.go` merge conflict by keeping manual payload/run constants and moving inference key additions into `turns_codegen.yaml`.
- Regenerated turns artifacts and validated turns/inference/provider packages.
- Extended `cmd/gen-js-api` with `--turns-schema` import support and generated turns-domain const groups in `gp.consts`: `BlockKind`, `TurnDataKeys`, `MetadataKeys`, `TurnMetadataKeys`, `BlockMetadataKeys`.
- Updated JS module tests and JS API reference docs for new const groups.
- Added contributor note in `pkg/turns/spec/README.md` documenting schema/codegen ownership and required regeneration commands.
- Added exhaustive analysis document defining a no-backward-compat single-generator + single-manifest architecture covering all key families (data/turn_meta/block_meta/run_meta/payload), JS enums, Go/TS outputs, engine turnkeys generation, and codec map generation.
- Added and executed ticket-local experiment script to inventory current generated vs manual surfaces: `scripts/analyze_codegen_overlap.go`.

## 2026-02-20

Implemented unified codegen migration (commit 78dfc79): added cmd/gen-meta + pkg/spec/geppetto_codegen.yaml as single source for turns/engine/js outputs; removed cmd/gen-turns/cmd/gen-js-api and legacy split schemas; removed manual duplicate key files and moved canonical key ownership to generated files; updated linter allowlist, JS const tests, and docs; regenerated artifacts and passed full test/lint/vet suite.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/cmd/gen-meta/main.go — Unified generator entrypoint and emitters
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/inference/engine/turnkeys_gen.go — Generated engine-owned typed keys
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/consts_gen.go — Generated JS const installer for all groups
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/spec/geppetto_codegen.yaml — Authoritative manifest for all key families and JS exports
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/turns/keys_gen.go — Generated turns-owned constants and typed keys


## 2026-02-25

Ticket closed

