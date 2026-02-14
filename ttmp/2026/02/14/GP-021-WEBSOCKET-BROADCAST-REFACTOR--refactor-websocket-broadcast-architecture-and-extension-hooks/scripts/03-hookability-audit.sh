#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../../../../.." && pwd)"
OUT_DIR="$ROOT/geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/sources"
mkdir -p "$OUT_DIR"

OUT="$OUT_DIR/ws-hookability-audit.txt"
{
  echo "# Hookability Audit"
  echo
  echo "## Existing explicit extension hooks in webchat"
  rg -n "WithTimelineUpsertHook|timelineUpsertHookOverride|RegisterTimelineHandler|WithEventSinkWrapper|WithBuildSubscriber|EngineFromReqBuilder|RouterOption" "$ROOT/pinocchio/pkg/webchat" -g'*.go'
  echo
  echo "## Areas where direct ConnectionPool access is currently required"
  rg -n "pool\\.Broadcast|pool\\.SendToOne|type ConnectionPool|dropClient|TrySend" "$ROOT/pinocchio/pkg/webchat" -g'*.go'
  echo
  echo "## Debug API data sources relevant to websocket bootstrap/catch-up"
  rg -n "HandleFunc\\(\\\"/api/debug/(timeline|events|turns)|since_version|since_seq|since_ms|semBuf\\.Snapshot" "$ROOT/pinocchio/pkg/webchat/router_debug_routes.go"
} > "$OUT"

echo "wrote $OUT"
