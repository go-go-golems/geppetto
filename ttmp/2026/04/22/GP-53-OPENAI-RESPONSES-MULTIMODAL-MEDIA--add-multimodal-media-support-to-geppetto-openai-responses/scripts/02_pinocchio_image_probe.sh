#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
TICKET_DIR=$(cd -- "$SCRIPT_DIR/.." && pwd)
FIXTURE_DIR="$TICKET_DIR/sources/01-live-responses-image-smoke"
OUTPUT_DIR="$TICKET_DIR/various/02-pinocchio-image-probe"
IMAGE_PATH="$FIXTURE_DIR/passcode-card.png"
OUTPUT_LOG="$OUTPUT_DIR/output.log"

mkdir -p "$OUTPUT_DIR"

if [[ ! -f "$IMAGE_PATH" ]]; then
  echo "Fixture image missing: $IMAGE_PATH" >&2
  echo "Run scripts/01_live_openai_responses_image_smoke.sh first." >&2
  exit 1
fi

pinocchio --profile gpt-5-nano-low code professional \
  --non-interactive \
  --full-output \
  --output yaml \
  "What four-digit passcode is shown in the image, and what shape/color appears on the left? Answer in one sentence." \
  --images "$IMAGE_PATH" | tee "$OUTPUT_LOG"

echo "Saved output log to: $OUTPUT_LOG"
