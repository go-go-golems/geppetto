# Changelog

## 2026-02-20

- Initial workspace created
- Added analysis document with detailed mechanical split plan and function inventory.
- Mechanically split `pkg/js/modules/geppetto/api.go` into domain files and removed monolith (`440b1a9`).
- Fixed one extraction miss by restoring `start`, `buildRunContext`, and `parseRunOptions` verbatim from `HEAD` before final validation.
- Validation executed:
  - `go test ./pkg/js/modules/geppetto -count=1`
  - `go test ./pkg/js/modules/geppetto -race -count=1`
  - pre-commit: `go test ./...`, `go generate ./...`, `go build ./...`, `golangci-lint run`, `go vet`
