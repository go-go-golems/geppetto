#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
RUNNER_DIR="${ROOT_DIR}/cozo-relationship-js-runner"
MALFORMED="${ROOT_DIR}/ttmp/2026/02/21/CO-07-PLUGIN-DESCRIPTOR-API--introduce-explicit-js-plugin-descriptor-api-for-runner-scripts/scripts/02-malformed-descriptor.js"
TRANSCRIPT_FILE="$(mktemp)"
trap 'rm -f "${TRANSCRIPT_FILE}"' EXIT

cat > "${TRANSCRIPT_FILE}" <<'EOF'
Alice worked with Bob on the migration and they reviewed model outputs together.
EOF

cd "${RUNNER_DIR}"
set +e
go run . extract "${MALFORMED}" "${TRANSCRIPT_FILE}"
STATUS=$?
set -e

if [[ ${STATUS} -eq 0 ]]; then
  echo "Expected malformed descriptor run to fail, but it succeeded." >&2
  exit 1
fi

echo "Malformed descriptor correctly failed (exit ${STATUS})."
