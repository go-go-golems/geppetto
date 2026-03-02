#!/usr/bin/env bash
set -euo pipefail

ROOT="/home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships"
CFG="/tmp/structured-event-extraction.lowtokens.ticket-men-tr-005.yaml"
OUT="/tmp/men-tr-005-lowtokens.stdout"
ERR="/tmp/men-tr-005-lowtokens.stderr"
DB="/tmp/men-tr-005-lowtokens.db"

cd "$ROOT"

read -r BYTES LONGEST < <(find anonymized -type f -name '*.txt' -printf '%s %p\n' | sort -n | tail -n1)

cat > "$CFG" <<'YAML'
engine:
  mode: profile
  apiType: claude
  maxResponseTokens: 32

prompt:
  structuredEventText: ""

profiles:
  registrySources: []

loop:
  maxIterations: 3
  continuePrompt: Continue extraction until STOP token is emitted.
  timeoutMs: 120000
  tags:
    app: temporal-relationships
    mode: men-tr-005-lowtokens-repro

stopPolicy:
  stopSequences:
    - __STOP__
  acceptedStopReasons:
    - stop_sequence
    - end_turn
    - max_tokens
  failOnMaxIterations: false
  continueOnFirstMaxTokens: true
YAML

rm -f "$OUT" "$ERR" "$DB"

go run ./cmd/temporal-relationships \
  --profile-registries "yaml://$HOME/.config/pinocchio/profiles.yaml" \
  --db-path "$DB" \
  go extract \
  --config "$CFG" \
  --input-file "$LONGEST" \
  --profile mento-haiku \
  --timeline-printer=false \
  --print-result=true \
  --result-format=json \
  > "$OUT" 2> "$ERR"

echo "longest_file=$LONGEST"
echo "bytes=$BYTES"
jq '{status,reason,iterations,eventCount,relationshipCount,extractionSource,structuredParseFailuresCount:(.structuredParseFailures|length)}' "$OUT"
echo "run_inference_starts=$(rg -o 'Claude RunInference started' "$ERR" | wc -l | tr -d ' ')"
echo "message_delta_max_tokens=$(rg -o '"stop_reason":"max_tokens"' "$ERR" | wc -l | tr -d ' ')"
echo "stdout=$OUT"
echo "stderr=$ERR"
