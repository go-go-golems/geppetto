#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../../../../.." && pwd)"
OUT_DIR="$ROOT/geppetto/ttmp/2026/02/14/GP-021-WEBSOCKET-BROADCAST-REFACTOR--refactor-websocket-broadcast-architecture-and-extension-hooks/sources"
mkdir -p "$OUT_DIR"

OUT="$OUT_DIR/ws-broadcast-paths.txt"
{
  echo "# WebSocket Broadcast Path Inventory"
  echo
  echo "## Broadcast and targeted send callsites"
  rg -n "Broadcast\\(|SendToOne\\(" "$ROOT/pinocchio/pkg/webchat" -g'*.go'
  echo
  echo "## Timeline upsert emission path"
  rg -n "emitTimelineUpsert|TimelineUpsertHook|timelineUpsertHook|timeline\.upsert" "$ROOT/pinocchio/pkg/webchat" -g'*.go'
  echo
  echo "## Stream coordinator callback path"
  rg -n "NewStreamCoordinator|onFrame|SemanticEventsFromEventWithCursor|pool\\.Broadcast|timelineProj\\.ApplySemFrame" "$ROOT/pinocchio/pkg/webchat" -g'*.go'
} > "$OUT"

echo "wrote $OUT"
