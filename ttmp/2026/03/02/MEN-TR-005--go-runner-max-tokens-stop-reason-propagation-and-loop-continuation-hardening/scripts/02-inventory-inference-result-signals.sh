#!/usr/bin/env bash
set -euo pipefail

ROOT="/home/manuel/workspaces/2026-03-02/deliver-mento-1"
OUT="/tmp/men-tr-005-inference-signals.txt"

{
  echo "# Inference Result Signaling Inventory"
  echo
  echo "Generated: $(date -Iseconds)"
  echo

  echo "## Engine contract"
  rg -n "type Engine interface|RunInference\(" "$ROOT/geppetto/pkg/inference/engine/engine.go" -n -S
  echo

  echo "## Session execution handle"
  rg -n "type ExecutionHandle|SessionID|InferenceID|Input|Wait\(|Cancel\(" "$ROOT/geppetto/pkg/inference/session/execution.go" -S
  echo

  echo "## Turn metadata canonical keys"
  rg -n "TurnMeta.*ValueKey|KeyTurnMeta" "$ROOT/geppetto/pkg/turns/keys_gen.go" -S
  echo

  echo "## Event metadata fields"
  rg -n "type LLMInferenceData|StopReason|Usage|DurationMs|type EventMetadata" "$ROOT/geppetto/pkg/events/metadata.go" "$ROOT/geppetto/pkg/events/chat-events.go" -S
  echo

  echo "## Provider-specific turn stop_reason setters"
  rg -n "KeyTurnMetaStopReason\.Set" "$ROOT/geppetto/pkg/steps/ai" -S
  echo

  echo "## Claude stream stop_reason flow"
  rg -n "MessageDeltaType|StopReason|message_delta|message_stop" "$ROOT/geppetto/pkg/steps/ai/claude/content-block-merger.go" "$ROOT/geppetto/pkg/steps/ai/claude/api/streaming.go" "$ROOT/geppetto/pkg/steps/ai/claude/engine_claude.go" -S
  echo

  echo "## Runner-side stop-reason read path"
  rg -n "stopReasonOfTurn|shouldStop|AcceptedStopReasons|continueOnFirstMaxTokens" "$ROOT/temporal-relationships/internal/extractor/gorunner/run.go" -S
  echo

  echo "## Toolloop snapshot and pause channels"
  rg -n "SnapshotHook|WithTurnSnapshotHook|StepPhase|NewDebuggerPauseEvent|PauseMeta" "$ROOT/geppetto/pkg/inference/toolloop" -S
  echo

  echo "## JS persistence stop_reason and turns/timeline tables"
  rg -n "stop_reason|temporal_relationship_turns|temporal_relationship_timeline|persistIteration|completeRun" "$ROOT/temporal-relationships/js/extractor/persistence.js" -S
} > "$OUT"

echo "wrote $OUT"
