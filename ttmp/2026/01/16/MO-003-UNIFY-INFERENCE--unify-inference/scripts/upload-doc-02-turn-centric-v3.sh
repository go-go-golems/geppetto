#!/usr/bin/env bash
set -euo pipefail

DOC="/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference/design-doc/02-turn-centric-conversation-state-and-runner-api-v3.md"

python3 /home/manuel/.local/bin/remarkable_upload.py \
  --ticket-dir /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/16/MO-003-UNIFY-INFERENCE--unify-inference \
  --mirror-ticket-structure \
  "$DOC"
