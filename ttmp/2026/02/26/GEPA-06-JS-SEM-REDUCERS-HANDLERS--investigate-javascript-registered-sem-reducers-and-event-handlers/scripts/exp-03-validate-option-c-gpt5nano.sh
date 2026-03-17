#!/usr/bin/env bash
set -euo pipefail

ROOT="/home/manuel/workspaces/2026-02-22/add-gepa-optimizer"
PINOCCHIO="$ROOT/pinocchio"
GEPA="$ROOT/go-go-gepa"
TICKET="$GEPA/ttmp/2026/02/26/GEPA-06-JS-SEM-REDUCERS-HANDLERS--investigate-javascript-registered-sem-reducers-and-event-handlers"
SCRIPTS="$TICKET/scripts"
OUTDIR="$SCRIPTS/exp-03-out"
SUMMARY="$SCRIPTS/exp-03-summary.txt"

mkdir -p "$OUTDIR"
rm -f "$SUMMARY" "$OUTDIR"/*.txt

{
  echo "GEPA-06 Option C Task 3 validation"
  echo "date: $(date --iso-8601=seconds)"
  echo "ticket: $TICKET"
  echo
  echo "1) Pinocchio cmd/web-chat loader tests (includes gpt-5-nano profile resolver)"
} | tee -a "$SUMMARY"

(
  cd "$PINOCCHIO"
  go test ./cmd/web-chat -run 'TimelineJSScript|GPT5NanoProfile|ProfileResolver' -v
) > "$OUTDIR/pinocchio-cmd-web-chat.txt" 2>&1
cat "$OUTDIR/pinocchio-cmd-web-chat.txt" >> "$SUMMARY"

{
  echo
  echo "2) Pinocchio JS timeline runtime tests (llm.delta semantics)"
} | tee -a "$SUMMARY"

(
  cd "$PINOCCHIO"
  go test ./pkg/webchat -run 'TestJSTimelineRuntime_NonConsumingReducerAllowsBuiltinProjection|TestJSTimelineRuntime_ReducerCreatesEntityAndConsumesEvent' -v
) > "$OUTDIR/pinocchio-pkg-webchat.txt" 2>&1
cat "$OUTDIR/pinocchio-pkg-webchat.txt" >> "$SUMMARY"

{
  echo
  echo "3) go-go-gepa streaming CLI integration (gpt-5-nano profile)"
} | tee -a "$SUMMARY"

(
  cd "$GEPA"
  go test ./cmd/gepa-runner -run 'TestDatasetGenerateStreamCLIOutput' -v
) > "$OUTDIR/go-go-gepa-streaming.txt" 2>&1
cat "$OUTDIR/go-go-gepa-streaming.txt" >> "$SUMMARY"

{
  echo
  echo "Validation completed successfully."
  echo "Summary: $SUMMARY"
} | tee -a "$SUMMARY"
