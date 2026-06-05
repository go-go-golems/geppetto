#!/usr/bin/env bash
set -euo pipefail

# Probe whether the legacy and modern Gemini Go SDKs expose the fields needed
# for Gemini 3 thinking, thought signatures, provider function-call IDs, and
# usage metadata. The probe runs in an isolated temporary module so it does not
# edit Geppetto's go.mod.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../../.." && pwd)"
ARTIFACT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/artifacts"
OUT="$ARTIFACT_DIR/sdk-capability-probe.json"
WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/gemini-sdk-probe.XXXXXX")"
trap 'rm -rf "$WORKDIR"' EXIT

cd "$WORKDIR"
go mod init gemini-sdk-capability-probe >/dev/null

cat > old_baseline.go <<'GO'
package main

import oldgenai "github.com/google/generative-ai-go/genai"

func main() {
	_ = oldgenai.Text("hello")
	_ = oldgenai.FunctionCall{Name: "lookup", Args: map[string]any{"q": "x"}}
	_ = oldgenai.FunctionResponse{Name: "lookup", Response: map[string]any{"ok": true}}
}
GO

cat > old_modern_fields.go <<'GO'
package main

import oldgenai "github.com/google/generative-ai-go/genai"

func main() {
	budget := int32(128)
	_ = &oldgenai.ThinkingConfig{IncludeThoughts: true, ThinkingBudget: &budget}
	_ = oldgenai.Part{Thought: true, ThoughtSignature: []byte("sig")}
	_ = oldgenai.FunctionCall{ID: "call-1", Name: "lookup", Args: map[string]any{"q": "x"}}
	_ = oldgenai.FunctionResponse{ID: "call-1", Name: "lookup", Response: map[string]any{"ok": true}}
	_ = oldgenai.GenerateContentResponse{ResponseID: "resp-1"}
}
GO

cat > new_modern_fields.go <<'GO'
package main

import newgenai "google.golang.org/genai"

func main() {
	budget := int32(128)
	_ = &newgenai.ThinkingConfig{IncludeThoughts: true, ThinkingBudget: &budget}
	_ = &newgenai.Part{Text: "thinking", Thought: true, ThoughtSignature: []byte("sig")}
	_ = &newgenai.FunctionCall{ID: "call-1", Name: "lookup", Args: map[string]any{"q": "x"}}
	_ = &newgenai.FunctionResponse{ID: "call-1", Name: "lookup", Response: map[string]any{"ok": true}}
	_ = &newgenai.GenerateContentResponse{
		ResponseID:    "resp-1",
		UsageMetadata: &newgenai.GenerateContentResponseUsageMetadata{ThoughtsTokenCount: 7},
	}
	_ = &newgenai.GenerateContentConfig{ThinkingConfig: &newgenai.ThinkingConfig{IncludeThoughts: true}}
}
GO

# Resolve modules inside the temp module after all probe files exist so go.sum
# contains transitive dependencies needed by the legacy SDK as well.
go get github.com/google/generative-ai-go/genai@v0.20.1 google.golang.org/genai@v1.58.0 >/dev/null 2>&1
go mod tidy >/dev/null 2>&1

run_probe() {
  local name="$1"
  local file="$2"
  local stdout="$WORKDIR/${name}.stdout"
  local stderr="$WORKDIR/${name}.stderr"
  if go build -o "$WORKDIR/${name}.bin" "$file" >"$stdout" 2>"$stderr"; then
    printf '%s\ttrue\t%s\t%s\n' "$name" "$stdout" "$stderr"
  else
    printf '%s\tfalse\t%s\t%s\n' "$name" "$stdout" "$stderr"
  fi
}

RESULTS_TSV="$WORKDIR/results.tsv"
{
  run_probe old_baseline old_baseline.go
  run_probe old_modern_fields old_modern_fields.go
  run_probe new_modern_fields new_modern_fields.go
} > "$RESULTS_TSV"

python3 - "$RESULTS_TSV" "$OUT" "$WORKDIR" <<'PY'
import json, sys, subprocess, pathlib, datetime
results_tsv, out, workdir = sys.argv[1:]
rows = []
for line in pathlib.Path(results_tsv).read_text().splitlines():
    name, ok, stdout_path, stderr_path = line.split('\t')
    rows.append({
        "name": name,
        "build_ok": ok == "true",
        "stdout": pathlib.Path(stdout_path).read_text(errors="replace"),
        "stderr": pathlib.Path(stderr_path).read_text(errors="replace"),
    })

def mod_version(path):
    try:
        return subprocess.check_output(["go", "list", "-m", path], cwd=workdir, text=True).strip()
    except Exception as e:
        return f"ERROR: {e}"

out_obj = {
    "generated_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "purpose": "Compare legacy github.com/google/generative-ai-go/genai and modern google.golang.org/genai support for Gemini 3 fields.",
    "modules": {
        "legacy": mod_version("github.com/google/generative-ai-go"),
        "modern": mod_version("google.golang.org/genai"),
    },
    "results": rows,
    "interpretation": {
        "old_baseline": "Legacy SDK compiles for basic text/function call/function response types.",
        "old_modern_fields": "Expected to fail: legacy SDK should not expose ThinkingConfig, Part struct thought fields, FunctionCall.ID, FunctionResponse.ID, or GenerateContentResponse.ResponseID.",
        "new_modern_fields": "Expected to pass: modern SDK exposes Gemini 3 thinking/signature/function ID/usage fields."
    }
}
pathlib.Path(out).write_text(json.dumps(out_obj, indent=2) + "\n")
print(out)
PY
