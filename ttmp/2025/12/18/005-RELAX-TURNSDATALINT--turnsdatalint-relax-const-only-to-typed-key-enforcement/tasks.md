# Tasks

## TODO

- [ ] Implement typed-key enforcement in `turnsdatalint` (core change)
  - [x] Update `geppetto/pkg/analysis/turnsdatalint/analyzer.go`
    - [x] Replace const-identity check with type-based check for typed-key maps
    - [x] Keep `Block.Payload` as const-string only (no literals, no vars)
    - [x] Remove helper allowlist (avoid bypass holes)
  - [x] Update tests in `geppetto/pkg/analysis/turnsdatalint/testdata/src/a/a.go`
    - [x] Allow typed conversions, typed vars, typed params for typed-key maps
    - [x] Add failing tests for raw string literals + untyped string consts
  - [x] Update docs in `geppetto/pkg/doc/topics/12-turnsdatalint.md`
    - [x] Describe the new rule (typed-key enforcement)
    - [x] Update examples
    - [x] Fix flag name docs (current analyzer has multiple keytype flags)
  - [x] Validate by running:
    - [x] `cd geppetto && make build`
    - [x] `cd geppetto && go test ./... -count=1`
    - [x] `cd geppetto && make linttool`

