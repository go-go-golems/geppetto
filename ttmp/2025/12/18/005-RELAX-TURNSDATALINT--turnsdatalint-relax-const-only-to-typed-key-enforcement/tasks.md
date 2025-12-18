# Tasks

## TODO

- [ ] Implement typed-key enforcement in `turnsdatalint` (core change)
  - [ ] Update `geppetto/pkg/analysis/turnsdatalint/analyzer.go`
    - [ ] Replace const-identity check with type-based check for typed-key maps
    - [ ] Keep `Block.Payload` as const-string only (no literals, no vars)
    - [ ] Remove helper allowlist (avoid bypass holes)
  - [ ] Update tests in `geppetto/pkg/analysis/turnsdatalint/testdata/src/a/a.go`
    - [ ] Allow typed conversions, typed vars, typed params for typed-key maps
    - [ ] Add failing tests for raw string literals + untyped string consts
  - [ ] Update docs in `geppetto/pkg/doc/topics/12-turnsdatalint.md`
    - [ ] Describe the new rule (typed-key enforcement)
    - [ ] Update examples
    - [ ] Fix flag name docs (current analyzer has multiple keytype flags)
  - [ ] Validate by running:
    - [ ] `cd geppetto && make build`
    - [ ] `cd geppetto && go test ./... -count=1`

