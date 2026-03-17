#!/usr/bin/env bash
set -euo pipefail

ROOT="/home/manuel/workspaces/2026-02-22/add-gepa-optimizer"
PINOCCHIO="$ROOT/pinocchio"
TICKET="$ROOT/go-go-gepa/ttmp/2026/02/26/GEPA-06-JS-SEM-REDUCERS-HANDLERS--investigate-javascript-registered-sem-reducers-and-event-handlers"
SCRIPTS="$TICKET/scripts"
OUT="$SCRIPTS/exp-04-harness-output.txt"

rm -f "$OUT"

{
  echo "GEPA-06 exp-04 llm.delta projection harness"
  echo "date: $(date --iso-8601=seconds)"
  echo "command: go test ./cmd/web-chat -run 'LLMDeltaProjectionHarness' -v"
  echo
} > "$OUT"

(
  cd "$PINOCCHIO"
  go test ./cmd/web-chat -run 'LLMDeltaProjectionHarness' -v
) >> "$OUT" 2>&1

echo "wrote: $OUT"
