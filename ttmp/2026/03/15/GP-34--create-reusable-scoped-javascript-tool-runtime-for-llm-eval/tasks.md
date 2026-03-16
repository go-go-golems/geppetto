# Tasks

## TODO

### Discovery and design

- [x] Create the GP-34 ticket workspace under `geppetto/ttmp`.
- [x] Inspect the freshly created scoped DB package and GP-33 ticket as the structural precedent.
- [x] Inspect the current JS runtime, goja module registration, and JS tool registry code paths in Geppetto.
- [x] Write the intern-facing design guide covering the Geppetto tool system, JS runtime stack, go-go-goja engine, and JSDocEx documentation flow.
- [x] Upload the finalized GP-34 document bundle to reMarkable.

### Slice 1: package skeleton and pure-data API

- [x] Create `geppetto/pkg/inference/tools/scopedjs/`.
- [x] Define `ToolDescription`, `ToolDefinitionSpec`, `EvalOptions`, `StateMode`, `EnvironmentSpec`, and `BuildResult`.
- [x] Define manifest doc types for modules, globals, helpers, and bootstrap files.
- [x] Define the builder data model and additive builder methods for modules, globals, initializers, helpers, and bootstrap sources/files.
- [x] Implement safe normalization helpers and manifest copying helpers.
- [x] Implement description rendering for runtime capabilities and state mode.
- [x] Add unit tests for defaults, builder validation, manifest rendering, and description formatting.

### Slice 2: runtime construction and eval execution

- [x] Implement runtime build path on top of `go-go-goja/engine`.
- [x] Convert builder module entries into engine module specs.
- [x] Convert builder global bindings into runtime initializers.
- [x] Implement bootstrap source/file loading on the runtime owner thread.
- [x] Implement `EvalInput` and `EvalOutput`.
- [x] Reuse or adapt owned-runtime promise-settling logic from the go-go-goja JavaScript evaluator.
- [x] Implement console capture and output truncation behavior.
- [x] Add unit tests for runtime creation, bootstrap loading, promise fulfillment/rejection, and timeout handling.

### Slice 3: tool registration and integration

- [x] Implement `RegisterPrebuilt`.
- [x] Implement `NewLazyRegistrar`.
- [x] Add tests proving provider-safe JSON schema generation for eval input.
- [x] Add tests proving lazy scope resolution and cleanup behavior.
- [ ] Decide and document the package location and public naming in the design doc if implementation deviates.

### Slice 4: examples, docs, and finish

- [ ] Add an example covering `fs` plus a simple scoped global.
- [ ] Add an example for the motivating `db + webserver + obsidian + fs` composition shape, even if some pieces are fake/test doubles.
- [ ] Add a playbook or topic doc showing how an app should adopt `scopedjs`.
- [x] Update the GP-34 changelog and diary with implementation checkpoints and exact commands.
- [x] Run targeted tests for the new package and any affected JS runtime packages.
- [x] Commit the work in small logical checkpoints.
