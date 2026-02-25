#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

LAB=(go run ./cmd/examples/geppetto-js-lab)
STACK_REGISTRIES="${SCRIPT_DIR}/profiles/10-provider-openai.yaml,${SCRIPT_DIR}/profiles/20-team-agent.yaml,${SCRIPT_DIR}/profiles/30-user-overrides.yaml"
OPENAI_API_KEY_DEFAULT="${OPENAI_API_KEY:-example-openai-key}"

TMP_DIR="$(mktemp -d)"
SQLITE_PATH="${TMP_DIR}/workspace-profiles.db"
trap 'rm -rf "${TMP_DIR}"' EXIT

run_script() {
  local script_path="$1"
  shift
  echo "==> ${script_path}"
  (cd "${REPO_ROOT}" && OPENAI_API_KEY="${OPENAI_API_KEY_DEFAULT}" "${LAB[@]}" --script "${script_path}" "$@")
}

echo "==> Seeding sqlite profile registry: ${SQLITE_PATH}"
(cd "${REPO_ROOT}" && "${LAB[@]}" --seed-profile-sqlite "${SQLITE_PATH}")

echo "==> Running existing baseline scripts"
run_script "examples/js/geppetto/01_turns_and_blocks.js"
run_script "examples/js/geppetto/02_session_echo.js"
run_script "examples/js/geppetto/03_middleware_composition.js"
run_script "examples/js/geppetto/04_tools_and_toolloop.js"
run_script "examples/js/geppetto/05_go_tools_from_js.js"
run_script "examples/js/geppetto/06_live_profile_inference.js"
run_script "examples/js/geppetto/07_context_and_constants.js"

echo "==> Running profile/schema scripts against stacked YAML registries"
run_script "examples/js/geppetto/08_profiles_registry_inventory.js" --profile-registries "${STACK_REGISTRIES}"
run_script "examples/js/geppetto/09_profiles_resolve_stack_precedence.js" --profile-registries "${STACK_REGISTRIES}"
run_script "examples/js/geppetto/10_engines_from_profile_metadata.js" --profile-registries "${STACK_REGISTRIES}"
run_script "examples/js/geppetto/11_profiles_resolve_explicit_registry.js" --profile-registries "${STACK_REGISTRIES}"
run_script "examples/js/geppetto/12_profiles_request_overrides_policy.js" --profile-registries "${STACK_REGISTRIES}"
run_script "examples/js/geppetto/13_schemas_middlewares_catalog.js" --profile-registries "${STACK_REGISTRIES}"
run_script "examples/js/geppetto/14_schemas_extensions_catalog.js" --profile-registries "${STACK_REGISTRIES}"
run_script "examples/js/geppetto/17_from_profile_legacy_registry_option_error.js" --profile-registries "${STACK_REGISTRIES}"

echo "==> Running profile CRUD script against sqlite registry"
run_script "examples/js/geppetto/15_profiles_crud_sqlite.js" --profile-registries "${SQLITE_PATH}"

echo "==> Running mixed stack precedence script (YAML + sqlite)"
run_script "examples/js/geppetto/16_mixed_registry_precedence.js" --profile-registries "${STACK_REGISTRIES},${SQLITE_PATH}"

echo "==> Running expected error script without profile registries"
run_script "examples/js/geppetto/18_missing_profile_registry_errors.js"

echo "All geppetto JS profile/registry scripts passed."
