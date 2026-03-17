# Tasks

## Ticket Setup

- [x] Create the `GP-38` ticket workspace under `geppetto/ttmp`
- [x] Draft the cleanup design and implementation plan
- [x] Add a chronological implementation diary

## Implementation

- [x] Introduce a small explicit shared-runtime wrapper type in `pkg/inference/tools/scopedjs`
- [x] Ensure the wrapper serializes `RunEval(...)` across all phases for one reused runtime
- [x] Expose the wrapper from `BuildResult` without breaking existing callers that still use `BuildResult.Runtime`
- [x] Switch `RegisterPrebuilt(...)` to use the wrapper instead of calling `RunEval(...)` on the raw runtime directly
- [x] Reuse the wrapper in the lazy path where it improves API consistency without changing lazy semantics

## Tests

- [x] Add regression coverage for concurrent prebuilt calls on one runtime
- [x] Prove same-runtime overlapping calls serialize instead of interleaving
- [x] Preserve existing prebuilt/lazy behavior assertions

## Docs

- [x] Update the scopedjs developer tutorial to explain the new wrapper and when to use it
- [x] Update ticket diary and changelog with the exact implementation and validation results

## Validation

- [x] Run `go test ./pkg/inference/tools/scopedjs`
- [x] Run `go test ./pkg/doc`
- [x] Run `docmgr doctor --root geppetto/ttmp --ticket GP-38 --stale-after 30`
- [ ] Commit the cleanup slice with a focused message
