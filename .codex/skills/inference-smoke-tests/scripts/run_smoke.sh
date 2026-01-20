#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
run_smoke.sh --quick|--full [--ai-engine MODEL] [--profile PROFILE]

Runs inference smoke tests across geppetto + pinocchio:
- geppetto: Responses "thinking" + generic tool calling
- pinocchio: TUI agent (tmux-driven) (quick)

Examples:
  bash geppetto/.codex/skills/inference-smoke-tests/scripts/run_smoke.sh --quick
  bash geppetto/.codex/skills/inference-smoke-tests/scripts/run_smoke.sh --quick --ai-engine gpt-5-mini

Notes:
  - Requires OPENAI_API_KEY for openai-responses tests.
  - Uses tmux for TUI runs; logs to /tmp/*.log.
EOF
}

MODE=""
AI_ENGINE="gpt-5-mini"
PROFILE="4o-mini"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --quick|--full) MODE="$1"; shift ;;
    --ai-engine) AI_ENGINE="${2:-}"; shift 2 ;;
    --profile) PROFILE="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown arg: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ -z "${MODE}" ]]; then
  usage
  exit 2
fi

if [[ -z "${OPENAI_API_KEY:-}" ]]; then
  echo "OPENAI_API_KEY is not set; aborting." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GEPPETTO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
WORKSPACE_ROOT="$(cd "${GEPPETTO_ROOT}/.." && pwd)"
PINOCCHIO_ROOT="${WORKSPACE_ROOT}/pinocchio"

echo "[info] GEPPETTO_ROOT=${GEPPETTO_ROOT}"
echo "[info] PINOCCHIO_ROOT=${PINOCCHIO_ROOT}"
echo "[info] MODE=${MODE} AI_ENGINE=${AI_ENGINE} PROFILE=${PROFILE}"

echo
echo "[1/3] geppetto: OpenAI Responses thinking smoke"
(cd "${GEPPETTO_ROOT}" && \
  go run ./cmd/examples/openai-tools test-openai-tools \
    --ai-api-type openai-responses \
    --ai-engine "${AI_ENGINE}" \
    --mode thinking \
    --prompt "What is 2+2?" \
  | head -n 120)

echo
echo "[2/3] geppetto: generic tool loop smoke"
(cd "${GEPPETTO_ROOT}" && \
  go run ./cmd/examples/generic-tool-calling \
    --pinocchio-profile "${PROFILE}" \
    --prompt "What's the weather in Paris and what is 2+2?" \
    --tools-enabled \
    --max-iterations 2 \
    --log-level info \
  | head -n 120)

echo
echo "[3/3] pinocchio: agent TUI smoke (tmux, Tab submits, Alt-q quits)"
if ! command -v tmux >/dev/null 2>&1; then
  echo "tmux not found; skipping pinocchio TUI agent smoke." >&2
  exit 0
fi

AGENT_LOG="/tmp/simple-chat-agent.smoke.log"
rm -f "${AGENT_LOG}"

tmux kill-session -t mo4-agent-smoke 2>/dev/null || true
tmux new-session -d -s mo4-agent-smoke \
  "cd \"${PINOCCHIO_ROOT}\" && go run ./cmd/agents/simple-chat-agent simple-chat-agent \
    --ai-api-type openai-responses \
    --ai-engine \"${AI_ENGINE}\" \
    --ai-max-response-tokens 256 \
    --openai-reasoning-summary auto \
    --log-level debug \
    --log-file \"${AGENT_LOG}\" \
    --with-caller"

sleep 2
tmux send-keys -t mo4-agent-smoke 'hello' Tab
sleep 14
tmux send-keys -t mo4-agent-smoke M-q
sleep 1
tmux kill-session -t mo4-agent-smoke 2>/dev/null || true

echo "--- ${AGENT_LOG} (tail) ---"
tail -n 180 "${AGENT_LOG}" || true

echo
echo "[ok] smoke suite complete"

