#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../../.." && pwd)}"
cd "$ROOT"

echo "== module files =="
find pkg/js/modules/geppetto -maxdepth 1 -type f | sort

echo
echo "== examples using current JS surface =="
rg -n 'createBuilder\(|createSession\(|runInference\(|profiles\.resolve\(|engines\.fromConfig\(|events\.collector\(' \
  examples/js/geppetto pkg/doc/topics pkg/doc/tutorials \
  -g'*.js' -g'*.md'
