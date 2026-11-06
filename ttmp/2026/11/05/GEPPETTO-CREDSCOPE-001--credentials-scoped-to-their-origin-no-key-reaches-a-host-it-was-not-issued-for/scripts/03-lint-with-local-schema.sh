#!/usr/bin/env bash
# Run the Makefile's lint analyzers with a matching local schema when remote
# schema download is unavailable. Arguments are the binary and its source schema.
set -euo pipefail
if [ "$#" -ne 2 ]; then
    printf 'Usage: %s GOLANGCI_BINARY MATCHING_SCHEMA_FILE\n' "$0" >&2
    exit 2
fi
lint_binary=$1
schema_file=$2
export GOWORK=off
export GOCACHE=/tmp/geppetto-bound-go-cache
export GOLANGCI_LINT_CACHE=/tmp/geppetto-bound-lint-cache
export GOFLAGS=-buildvcs=false
export GOPROXY=off

"$lint_binary" config verify --schema "$schema_file"
"$lint_binary" run -v --timeout=5m ./cmd/... ./pkg/...
go build -o /tmp/geppetto-bound-lint ./cmd/tools/geppetto-lint
go vet -vettool=/tmp/geppetto-bound-lint ./cmd/... ./pkg/...
go build -o /tmp/geppetto-bound-glazed-lint github.com/go-go-golems/glazed/cmd/tools/glazed-lint
go vet -vettool=/tmp/geppetto-bound-glazed-lint -glazedclilint.allow-paths=pkg/analysis/,pkg/cli/,pkg/cmds/fields/,pkg/cmds/logging/,pkg/cmds/sources/,pkg/help/ ./cmd/... ./pkg/...
