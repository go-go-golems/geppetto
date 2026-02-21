#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR" && git rev-parse --show-toplevel)"

cd "$ROOT_DIR"

echo "Running GP-17 JS API validation scripts with geppetto-js-lab"
for script in \
  "$SCRIPT_DIR/01_handles_consts_and_turns.js" \
  "$SCRIPT_DIR/02_context_hooks_and_run_options.js" \
  "$SCRIPT_DIR/03_async_surface_smoke.js"
do
  echo "---- $(basename "$script")"
  go run ./cmd/examples/geppetto-js-lab --script "$script"
done

echo "All GP-17 scripts passed."
