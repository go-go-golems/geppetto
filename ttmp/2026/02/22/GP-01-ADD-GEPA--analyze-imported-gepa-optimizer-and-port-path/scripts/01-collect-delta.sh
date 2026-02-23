#!/usr/bin/env bash
set -euo pipefail

WORKSPACE_ROOT="${WORKSPACE_ROOT:-/home/manuel/workspaces/2026-02-22/add-gepa-optimizer}"
BASE_REPO="$WORKSPACE_ROOT/geppetto"
IMPORTED_REPO="$WORKSPACE_ROOT/imported/geppetto-main"
COZO_REPO="$WORKSPACE_ROOT/2026-02-18--cozodb-extraction/cozo-relationship-js-runner"
TICKET_DIR="$WORKSPACE_ROOT/geppetto/ttmp/2026/02/22/GP-01-ADD-GEPA--analyze-imported-gepa-optimizer-and-port-path"
SOURCES_DIR="$TICKET_DIR/sources"
DIFFS_DIR="$SOURCES_DIR/diffs"

mkdir -p "$SOURCES_DIR" "$DIFFS_DIR"

summary_file="$SOURCES_DIR/01-tree-delta-summary.txt"
modified_file="$SOURCES_DIR/02-modified-files.txt"
inventory_file="$SOURCES_DIR/03-gepa-symbol-inventory.txt"

tmp_base="$(mktemp)"
tmp_imported="$(mktemp)"
trap 'rm -f "$tmp_base" "$tmp_imported"' EXIT

(cd "$BASE_REPO" && find . -type f | sed 's|^./||' | sort) >"$tmp_base"
(cd "$IMPORTED_REPO" && find . -type f | sed 's|^./||' | sort) >"$tmp_imported"

{
  echo "Generated: $(date -Iseconds)"
  echo "Workspace: $WORKSPACE_ROOT"
  echo
  echo "== Baseline Repos =="
  echo "Base:     $BASE_REPO"
  echo "Imported: $IMPORTED_REPO"
  echo "COZO:     $COZO_REPO"
  echo
  echo "== Git / Revision Context =="
  echo "geppetto HEAD: $(cd "$BASE_REPO" && git rev-parse --short HEAD)"
  echo "cozo extraction HEAD: $(cd "$WORKSPACE_ROOT/2026-02-18--cozodb-extraction" && git rev-parse --short HEAD)"
  echo "imported/geppetto-main has .git: $(test -e "$IMPORTED_REPO/.git" && echo yes || echo no)"
  echo
  echo "== File Set Delta =="
  echo "only_in_geppetto=$(comm -23 "$tmp_base" "$tmp_imported" | wc -l)"
  echo "only_in_imported=$(comm -13 "$tmp_base" "$tmp_imported" | wc -l)"
  echo "common=$(comm -12 "$tmp_base" "$tmp_imported" | wc -l)"
  echo
  echo "== Files Only In Imported (first 200) =="
  comm -13 "$tmp_base" "$tmp_imported" | sed -n '1,200p'
  echo
  echo "== Files Only In Base Geppetto =="
  comm -23 "$tmp_base" "$tmp_imported"
} >"$summary_file"

> "$modified_file"
while IFS= read -r f; do
  if ! cmp -s "$BASE_REPO/$f" "$IMPORTED_REPO/$f"; then
    printf '%s\n' "$f" >>"$modified_file"
  fi
done < <(comm -12 "$tmp_base" "$tmp_imported")

while IFS= read -r f; do
  [[ -z "$f" ]] && continue
  safe_name="$(echo "$f" | tr '/' '__')"
  diff -u "$BASE_REPO/$f" "$IMPORTED_REPO/$f" >"$DIFFS_DIR/$safe_name.diff" || true
done <"$modified_file"

{
  echo "Generated: $(date -Iseconds)"
  echo
  echo "== Imported GEPA-related Symbols =="
  (cd "$IMPORTED_REPO" && rg -n "gepa|optimizer|defineOptimizerPlugin|OPTIMIZER_PLUGIN_API_VERSION|Reflector|ParetoFront|loadOptimizerPlugin")
  echo
  echo "== COZO Runner Plugin Symbols =="
  (cd "$COZO_REPO" && rg -n "defineExtractorPlugin|extractorPluginAPIVersion|loadAndRunExtractorPlugin|run_recorder|eval-report|RELATIONSHIP_")
  echo
  echo "== GEPPETTO JS Module Registration (base vs imported) =="
  echo "-- base module.go"
  rg -n "RegisterNativeModule|geppetto/plugins|pluginsLoader" "$BASE_REPO/pkg/js/modules/geppetto/module.go"
  echo "-- imported module.go"
  rg -n "RegisterNativeModule|geppetto/plugins|pluginsLoader" "$IMPORTED_REPO/pkg/js/modules/geppetto/module.go"
} >"$inventory_file"

echo "Wrote:"
echo "  $summary_file"
echo "  $modified_file"
echo "  $inventory_file"
echo "  $DIFFS_DIR/*.diff"
