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
