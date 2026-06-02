#!/usr/bin/env bash
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
OUT="$ROOT/ttmp/2026/06/01/GP-GOJA-STREAM-EVENTS-2026-06-01--design-geppetto-js-streaming-events-via-go-go-goja-event-emitter/sources/01-code-evidence.md"
GOGOGOGA="$(go env GOPATH)/pkg/mod/github.com/go-go-golems/go-go-goja@v0.7.0"
{
  echo "---"
  echo 'title: "Code evidence"'
  echo "doc_type: reference"
  echo "topics:"
  echo "  - geppetto"
  echo "  - goja"
  echo "  - js-bindings"
  echo "  - streaming"
  echo "  - events"
  echo "status: active"
  echo "intent: evidence"
  echo "owners:"
  echo "  - manuel"
  echo "created: 2026-06-01"
  echo "updated: 2026-06-01"
  echo "---"
  echo
  echo "# Code Evidence"
  echo
  echo "Generated: $(date -Iseconds)"
  echo "Repository: $ROOT"
  echo "go-go-goja module: $GOGOGOGA"
  echo
  section() { echo; echo "## $1"; echo; }
  excerpt() {
    local title="$1" path="$2" start="$3" end="$4"
    section "$title"
    echo '```text'
    nl -ba "$path" | sed -n "${start},${end}p"
    echo '```'
  }
  excerpt "Geppetto JS module exports" "$ROOT/pkg/js/modules/geppetto/module.go" 140 180
  excerpt "Geppetto JS agent stream/run path" "$ROOT/pkg/js/modules/geppetto/api_agent.go" 220 380
  excerpt "Existing JS event collector" "$ROOT/pkg/js/modules/geppetto/api_events.go" 1 240
  excerpt "Owner bridge helpers" "$ROOT/pkg/js/modules/geppetto/api_owner_bridge.go" 1 80
  excerpt "Session StartInference lifecycle" "$ROOT/pkg/inference/session/session.go" 185 280
  excerpt "ExecutionHandle wait/cancel" "$ROOT/pkg/inference/session/execution.go" 1 80
  excerpt "Enginebuilder event sink injection" "$ROOT/pkg/inference/toolloop/enginebuilder/builder.go" 140 240
  excerpt "EventSink interface" "$ROOT/pkg/events/sink.go" 1 40
  excerpt "Event context helpers" "$ROOT/pkg/events/context.go" 1 60
  excerpt "Canonical event type constants" "$ROOT/pkg/events/chat-events.go" 1 120
  excerpt "Canonical text/provider events" "$ROOT/pkg/events/canonical_events.go" 1 220
  excerpt "Canonical tool events" "$ROOT/pkg/events/canonical_tool_events.go" 1 130
  excerpt "go-go-goja EventEmitter module" "$GOGOGOGA/modules/events/events.go" 1 260
  excerpt "go-go-goja EventEmitter Go adoption test" "$GOGOGOGA/modules/events/events_test.go" 126 190
  excerpt "Runner streaming example sink" "$ROOT/cmd/examples/runner-streaming/main.go" 1 120
} > "$OUT"
echo "$OUT"
