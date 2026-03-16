# Changelog

## 2026-03-15

- Created the GP-34 ticket workspace under `geppetto/ttmp`.
- Added an initial analysis document and investigation diary for the scoped JavaScript eval tool proposal.
- Inspected the current reusable scoped DB pattern in `geppetto/pkg/inference/tools/scopeddb` and the adjacent JS runtime surfaces in `geppetto/pkg/js`.
- Captured the main design direction: a reusable package that turns configured goja runtimes into one LLM-facing `eval_xxx` tool with bundled module/global/bootstrap documentation.

## 2026-03-16

- Added a detailed design and implementation guide for GP-34 aimed at a new intern.
- Expanded the codebase analysis to include the go-go-goja runtime factory, runtime owner, native module registry, concrete `fs` module, and the JSDocEx extraction/export/server stack.
- Documented the recommended package API, lifecycle modes, description-building strategy, execution flow, implementation phases, test matrix, and open questions.
- Uploaded the ticket bundle to reMarkable as `GP-34 scoped javascript eval tools guide` under `/ai/2026/03/16/GP-34`.
- Implemented the initial `pkg/inference/tools/scopedjs` package skeleton with core types, builder/manifest collection, description rendering, and focused unit tests.
