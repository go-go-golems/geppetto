#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <workspace-root>" >&2
  echo "example: $0 /home/manuel/workspaces/2026-03-01/generate-js-types" >&2
  exit 1
fi

WORKSPACE_ROOT="$1"
TICKET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$TICKET_DIR/sources/experiments"

mkdir -p "$OUT_DIR"

DTS="$WORKSPACE_ROOT/geppetto/pkg/doc/types/geppetto.d.ts"
TYPES_GO="$WORKSPACE_ROOT/go-go-goja/pkg/tsgen/spec/types.go"
RENDERER_GO="$WORKSPACE_ROOT/go-go-goja/pkg/tsgen/render/dts_renderer.go"
VALIDATOR_GO="$WORKSPACE_ROOT/go-go-goja/pkg/tsgen/validate/validate.go"

"$TICKET_DIR/scripts/01_probe_dts_surface.py" --dts "$DTS" > "$OUT_DIR/01-dts-surface-report.md"
"$TICKET_DIR/scripts/02_probe_tsgen_capabilities.py" \
  --types-go "$TYPES_GO" \
  --renderer-go "$RENDERER_GO" \
  --validator-go "$VALIDATOR_GO" > "$OUT_DIR/02-tsgen-capability-report.md"

{
  echo "# True Replacement Gap Experiment Bundle"
  echo
  echo "Generated: $(date -Iseconds)"
  echo
  echo "## Inputs"
  echo
  echo "- geppetto d.ts: $DTS"
  echo "- tsgen types: $TYPES_GO"
  echo "- tsgen renderer: $RENDERER_GO"
  echo "- tsgen validator: $VALIDATOR_GO"
  echo
  echo "## Artifacts"
  echo
  echo "- [01-dts-surface-report.md](./01-dts-surface-report.md)"
  echo "- [02-tsgen-capability-report.md](./02-tsgen-capability-report.md)"
} > "$OUT_DIR/README.md"

echo "Wrote experiment artifacts to $OUT_DIR"
