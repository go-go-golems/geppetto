#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-$(pwd)}"

rg -n \
  "StepSettingsPatch|step_settings_patch|ApplyRuntimeStepSettingsPatch|MergeRuntimeStepSettingsPatches|EffectiveStepSettings|BaseStepSettings" \
  "$ROOT/geppetto" \
  "$ROOT/pinocchio" \
  "$ROOT/2026-03-16--gec-rag" \
  "$ROOT/temporal-relationships" \
  --glob '!**/ttmp/**' \
  --glob '!**/node_modules/**' \
  --glob '!**/dist/**' \
  --glob '!**/vendor/**'
