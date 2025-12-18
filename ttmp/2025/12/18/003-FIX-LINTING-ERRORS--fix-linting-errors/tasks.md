# Tasks

## Completed

- [x] Create ticket `003-FIX-LINTING-ERRORS`
- [x] Fix `gosec`/SSA panic caused by `turnsdatalint` testdata importing real `turns` package missing `RunMetaKeyTraceID`
- [x] Verify `make lintmax gosec` passes in `geppetto/`
- [x] Update Watermill from v1.5.0 to v1.5.1 to support `AddConsumerHandler` API
- [x] Update `docker-lint` target to use golangci-lint v2.1.0 to match GitHub Actions workflow

## Files

- [x] `geppetto/pkg/turns/keys.go` (add `RunMetaKeyTraceID`)
- [x] `geppetto/go.mod` (update Watermill v1.5.0 → v1.5.1)
- [x] `geppetto/Makefile` (update docker-lint golangci-lint v2.0.2 → v2.1.0)

