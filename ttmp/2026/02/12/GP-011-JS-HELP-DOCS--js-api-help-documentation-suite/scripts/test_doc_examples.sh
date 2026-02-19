#!/usr/bin/env bash
set -euo pipefail

# Validate doc loading

go test ./pkg/doc -count=1 -v

# Validate script-first JS documentation examples

go run ./cmd/examples/geppetto-js-lab --list-go-tools

go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/01_turns_and_blocks.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/02_session_echo.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/03_middleware_composition.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/04_tools_and_toolloop.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/05_go_tools_from_js.js

# Optional live provider script (self-skips if keys are missing)
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/06_live_profile_inference.js
