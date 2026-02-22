#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
RUNNER_DIR="${ROOT_DIR}/cozo-relationship-js-runner"
TRANSCRIPT="${RUNNER_DIR}/examples/transcript.txt"

if [[ ! -f "${TRANSCRIPT}" ]]; then
  echo "Transcript not found: ${TRANSCRIPT}" >&2
  echo "Pass a transcript path as first arg to override." >&2
fi

INPUT_PATH="${1:-${TRANSCRIPT}}"

cd "${RUNNER_DIR}"
go run . extract ./scripts/relation_extractor_template.js "${INPUT_PATH}" --include-metadata
