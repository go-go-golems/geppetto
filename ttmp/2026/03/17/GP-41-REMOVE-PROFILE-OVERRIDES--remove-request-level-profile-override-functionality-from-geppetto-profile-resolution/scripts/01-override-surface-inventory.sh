#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-$(pwd)}"

cd "$ROOT"

echo "== Geppetto core =="
rg -n "request_overrides|RequestOverrides|overrideKey|AllowedOverrideKeys|DeniedOverrideKeys|AllowOverrides" \
  geppetto/pkg/profiles geppetto/pkg/js/modules/geppetto geppetto/examples/js/geppetto

echo
echo "== Pinocchio surfaces =="
rg -n "request_overrides|RequestOverrides|buildOverrides|requestOverrides" \
  pinocchio/pkg/webchat/http pinocchio/cmd/web-chat pinocchio/pkg/doc

echo
echo "== GEC-RAG surfaces =="
rg -n "request_overrides|RequestOverrides|requestOverrides" \
  2026-03-16--gec-rag/internal/webchat 2026-03-16--gec-rag/web/src 2026-03-16--gec-rag/pkg/doc

echo
echo "== Temporal Relationships surfaces =="
rg -n "request_overrides|RequestOverrides|requestOverrides" \
  temporal-relationships/internal/extractor/httpapi temporal-relationships/cmd || true
