# Changelog

## 2026-03-20

Created the Geppetto ticket and wrote the initial architecture/design/implementation guide for extracting shared inference debug printing and trace helpers into `geppetto/pkg/cli/bootstrap`.

Simplified the target design to a single `--print-inference-settings` path that includes provenance inline, masks secrets as `***`, and does not carry a dedicated debug-output test workstream.

Expanded `tasks.md` into a granular execution checklist spanning the Geppetto helper extraction, the Pinocchio clean cut, the downstream CozoDB backend migration, and the final documentation/upload pass.
