#!/usr/bin/env bash
set -euo pipefail

DOC="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/analysis/02-postmortem-inferencestate-session-enginebuilder-unification.md"
REMOTE_DIR="/ai/2026/01/20/MO-004-UNIFY-INFERENCE-STATE/analysis"

remarquee upload md --force --remote-dir "${REMOTE_DIR}" "${DOC}"

