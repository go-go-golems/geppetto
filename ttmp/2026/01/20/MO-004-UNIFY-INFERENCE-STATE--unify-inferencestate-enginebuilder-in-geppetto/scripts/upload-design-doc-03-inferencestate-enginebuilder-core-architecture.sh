#!/usr/bin/env bash
set -euo pipefail

DOC="/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/design-doc/03-inferencestate-enginebuilder-core-architecture.md"

remarquee upload md \
  --force \
  --remote-dir "/ai/2026/01/20/MO-004-UNIFY-INFERENCE-STATE--unify-inferencestate-enginebuilder-in-geppetto/design-doc/" \
  "$DOC"
