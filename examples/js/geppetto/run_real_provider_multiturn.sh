#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

PROFILE_REGISTRIES="${GEPPETTO_PROFILE_REGISTRIES:-${HOME}/.config/pinocchio/profiles.yaml}"
PROFILE="${GEPPETTO_PROFILE:-default}"
TIMEOUT_MS="${GEPPETTO_TIMEOUT_MS:-120000}"

cd "${REPO_ROOT}"
exec go run ./cmd/examples/geppetto-js-run run \
  --script examples/js/geppetto/30_real_provider_multiturn.js \
  --profile-registries "${PROFILE_REGISTRIES}" \
  --profile "${PROFILE}" \
  --timeout-ms "${TIMEOUT_MS}"
