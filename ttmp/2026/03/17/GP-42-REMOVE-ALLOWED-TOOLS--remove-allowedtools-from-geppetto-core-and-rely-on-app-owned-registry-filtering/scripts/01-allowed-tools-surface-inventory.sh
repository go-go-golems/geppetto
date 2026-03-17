#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-$(pwd)}"

cd "$ROOT"

echo "== Geppetto core =="
rg -n "AllowedTools|WithAllowedTools|FilterTools\\(|IsToolAllowed\\(|allowed_tools" \
  geppetto/pkg geppetto/cmd/examples

echo
echo "== Pinocchio app surfaces =="
rg -n "AllowedTools|allowed_tools" \
  pinocchio/pkg pinocchio/cmd || true

echo
echo "== GEC-RAG app surfaces =="
rg -n "AllowedTools|allowed_tools" \
  2026-03-16--gec-rag/internal 2026-03-16--gec-rag/web || true

echo
echo "== Temporal Relationships app surfaces =="
rg -n "AllowedTools|allowed_tools" \
  temporal-relationships/internal temporal-relationships/ui || true
