#!/usr/bin/env bash
set -euo pipefail

# Smoke-test the real-provider EventEmitter examples. This script intentionally
# checks for final JSON output rather than requiring text-delta events, because
# provider streaming behavior varies.

PROFILE_REGISTRIES=${PROFILE_REGISTRIES:-${GEPPETTO_PROFILE_REGISTRIES:-$HOME/.config/pinocchio/profiles.yaml}}
PROFILE=${PROFILE:-${GEPPETTO_PROFILE:-default}}
TIMEOUT_MS=${TIMEOUT_MS:-120000}
RUNNER=${RUNNER:-go run ./cmd/examples/geppetto-js-run run}

scripts=(
  examples/js/geppetto/31_event_emitter_run_async.js
  examples/js/geppetto/32_event_emitter_progress_summary.js
  examples/js/geppetto/33_event_emitter_multiturn_run_async.js
)

for script in "${scripts[@]}"; do
  echo "==> $script" >&2
  out_file=$(mktemp)
  if ! $RUNNER \
      --script "$script" \
      --profile-registries "$PROFILE_REGISTRIES" \
      --profile "$PROFILE" \
      --timeout-ms "$TIMEOUT_MS" >"$out_file"; then
    cat "$out_file" >&2 || true
    rm -f "$out_file"
    exit 1
  fi
  cat "$out_file"
  if ! grep -q '"finalText"' "$out_file"; then
    echo "missing finalText in output for $script" >&2
    rm -f "$out_file"
    exit 1
  fi
  rm -f "$out_file"
  echo >&2
done
