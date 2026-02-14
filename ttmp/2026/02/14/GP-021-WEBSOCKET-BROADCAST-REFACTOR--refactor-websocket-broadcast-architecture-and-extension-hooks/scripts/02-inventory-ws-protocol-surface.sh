#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../../../../.." && pwd)"
OUT_DIR="$ROOT/geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/sources"
mkdir -p "$OUT_DIR"

OUT="$OUT_DIR/ws-protocol-surface.txt"
{
  echo "# WS Protocol Surface Inventory"
  echo
  echo "## /ws handler and query params"
  rg -n "HandleFunc\\(\\\"/ws\\\"|URL\\.Query\\(\\)\\.Get|ws\\.hello|ws\\.pong|conn\\.ReadMessage|BuildEngineFromReq" "$ROOT/pinocchio/pkg/webchat/router.go"
  echo
  echo "## Current SEM event types emitted by translator"
  rg -n "\"type\": \"[a-zA-Z0-9._-]+\"" "$ROOT/pinocchio/pkg/webchat/sem_translator.go"
  echo
  echo "## Frontend websocket consumers"
  rg -n "new WebSocket|/ws\\?conv_id|registerSem\\(|timeline\\.upsert|wsManager" "$ROOT/pinocchio/cmd/web-chat/web/src" -g'*.ts' -g'*.tsx'
} > "$OUT"

echo "wrote $OUT"
