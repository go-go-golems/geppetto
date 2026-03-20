# Changelog

## 2026-03-20

Created the Geppetto ticket and wrote the initial architecture/design/implementation guide for extracting shared inference debug printing and trace helpers into `geppetto/pkg/cli/bootstrap`.

Simplified the target design to a single `--print-inference-settings` path that includes provenance inline, masks secrets as `***`, and does not carry a dedicated debug-output test workstream.

Expanded `tasks.md` into a granular execution checklist spanning the Geppetto helper extraction, the Pinocchio clean cut, the downstream CozoDB backend migration, and the final documentation/upload pass.

Implemented the shared helper in `geppetto/pkg/cli/bootstrap/inference_debug.go`, switched Pinocchio to use it directly, removed the old Pinocchio trace file, and switched the CozoDB backend to the same helper with a single `--print-inference-settings` debug surface.

Recorded the implementation commits:
- `geppetto`: `ac0872e` (`feat(bootstrap): add shared inference debug helper`)
- `pinocchio`: `4df0346` (`refactor(cmds): use shared inference debug helper`)
- `2026-03-14--cozodb-editor`: `07cdd50` (`refactor(backend): use shared inference debug helper`)

Validated the final ticket docs with `docmgr doctor` and refreshed the reMarkable bundle at `/ai/2026/03/20/GP-54-INFERENCE-DEBUG-BOOTSTRAP`.
