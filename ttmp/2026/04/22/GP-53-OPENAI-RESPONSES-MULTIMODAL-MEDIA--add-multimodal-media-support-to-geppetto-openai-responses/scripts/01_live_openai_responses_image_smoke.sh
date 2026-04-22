#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
TICKET_DIR=$(cd -- "$SCRIPT_DIR/.." && pwd)
REPO_ROOT=$(git -C "$TICKET_DIR" rev-parse --show-toplevel)
FIXTURE_DIR="$TICKET_DIR/sources/01-live-responses-image-smoke"
OUTPUT_DIR="$TICKET_DIR/various/01-live-responses-image-smoke"
export IMAGE_PATH="$FIXTURE_DIR/passcode-card.png"
OUTPUT_LOG="$OUTPUT_DIR/output.log"

mkdir -p "$FIXTURE_DIR" "$OUTPUT_DIR"

python3 - <<'PY'
from PIL import Image, ImageDraw, ImageFont
import os
image_path = os.environ['IMAGE_PATH']
img = Image.new('RGB', (900, 480), 'white')
d = ImageDraw.Draw(img)
try:
    font_big = ImageFont.truetype('/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf', 72)
    font_med = ImageFont.truetype('/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf', 42)
except Exception:
    font_big = None
    font_med = None
pts = [(90,380),(240,120),(390,380)]
d.polygon(pts, fill=(40,90,220))
d.text((460,80), 'PASSCODE', fill='black', font=font_med)
d.text((455,145), '4319', fill='black', font=font_big)
d.text((455,280), 'Shape: blue triangle', fill='black', font=font_med)
img.save(image_path)
print(image_path)
PY

if [[ -z "${OPENAI_API_KEY:-}" && -f "$HOME/.config/pinocchio/profiles.yaml" ]]; then
  export OPENAI_API_KEY
  OPENAI_API_KEY=$(python3 - <<'PY'
import os, yaml
path=os.path.expanduser('~/.config/pinocchio/profiles.yaml')
with open(path) as f:
    data=yaml.safe_load(f)
result=None
def walk(obj):
    global result
    if result is not None:
        return
    if isinstance(obj, dict):
        if obj.get('slug') == 'gpt-5-nano-low':
            result = obj
            return
        for v in obj.values():
            walk(v)
    elif isinstance(obj, list):
        for v in obj:
            walk(v)
walk(data)
key = result['inference_settings']['api']['api_keys']['openai-api-key']
print(key, end='')
PY
)
fi

if [[ -z "${OPENAI_API_KEY:-}" ]]; then
  echo "OPENAI_API_KEY is not set and no fallback key could be resolved from ~/.config/pinocchio/profiles.yaml" >&2
  exit 1
fi

(
  cd "$REPO_ROOT"
  IMAGE_PATH="$IMAGE_PATH" go run "$TICKET_DIR/scripts/01_live_openai_responses_image_smoke.go"
) | tee "$OUTPUT_LOG"

echo "Saved image fixture to: $IMAGE_PATH"
echo "Saved output log to: $OUTPUT_LOG"
