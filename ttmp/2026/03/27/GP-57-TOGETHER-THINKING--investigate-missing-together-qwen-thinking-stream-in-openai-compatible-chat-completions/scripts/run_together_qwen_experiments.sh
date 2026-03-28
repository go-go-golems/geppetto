#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TICKET_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$TICKET_DIR/../../../../../.." && pwd)"
OUT_DIR="$TICKET_DIR/sources/experiments"

mkdir -p "$OUT_DIR"

PROFILE="${PROFILE:-together-qwen-3.5-9b}"
PROFILES_FILE="${PROFILES_FILE:-$HOME/.config/pinocchio/profiles.yaml}"
PROMPT="${PROMPT:-What is 17 * 23? Think step by step, then give a short final answer.}"
SYSTEM_PROMPT="${SYSTEM_PROMPT:-You are a careful reasoning assistant.}"
MAX_TOKENS="${MAX_TOKENS:-128}"
TIMEOUT_BIN="${TIMEOUT_BIN:-timeout}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-40}"

run_probe() {
  local mode="$1"
  local out="$OUT_DIR/$2"

  echo "==> $mode -> $out"
  (
    cd "$REPO_ROOT/geppetto"
    "$TIMEOUT_BIN" "$TIMEOUT_SECONDS"s go run "./ttmp/2026/03/27/GP-57-TOGETHER-THINKING--investigate-missing-together-qwen-thinking-stream-in-openai-compatible-chat-completions/scripts/together_qwen_probe.go" \
      --mode "$mode" \
      --profile "$PROFILE" \
      --profiles "$PROFILES_FILE" \
      --prompt "$PROMPT" \
      --system "$SYSTEM_PROMPT" \
      --max-tokens "$MAX_TOKENS"
  ) | tee "$out"
}

run_probe raw-sse raw-sse.txt
run_probe go-openai go-openai.txt
run_probe geppetto geppetto.txt

echo "Wrote experiment outputs under $OUT_DIR"
