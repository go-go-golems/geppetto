#!/usr/bin/env bash
set -euo pipefail

WORKSPACE_ROOT="${WORKSPACE_ROOT:-/home/manuel/workspaces/2026-02-22/add-gepa-optimizer}"
BASE_REPO="$WORKSPACE_ROOT/geppetto"
IMPORTED_REPO="$WORKSPACE_ROOT/imported/geppetto-main"
TICKET_DIR="$WORKSPACE_ROOT/geppetto/ttmp/2026/02/22/GP-01-ADD-GEPA--analyze-imported-gepa-optimizer-and-port-path"
SOURCES_DIR="$TICKET_DIR/sources"

mkdir -p "$SOURCES_DIR"

build_file="$SOURCES_DIR/04-build-and-test-results.txt"
harness_file="$SOURCES_DIR/05-offline-optimizer-harness.txt"

{
  echo "Generated: $(date -Iseconds)"
  echo
  echo "== Environment =="
  go version
  echo
  echo "== Imported geppetto-main: build cmd/gepa-runner =="
  (
    cd "$IMPORTED_REPO"
    set +e
    GOWORK=off go build ./cmd/gepa-runner
    status=$?
    set -e
    echo "exit_code=$status"
    exit 0
  )
  echo
  echo "== Imported geppetto-main: compile relevant packages =="
  (
    cd "$IMPORTED_REPO"
    GOWORK=off go test ./pkg/optimizer/gepa ./pkg/js/modules/geppetto
  )
  echo
  echo "== Base geppetto: compile js module package =="
  (
    cd "$BASE_REPO"
    go test ./pkg/js/modules/geppetto
  )
} >"$build_file" 2>&1

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

cat >"$tmpdir/go.mod" <<EOF
module local/gepa-harness

go 1.25.7

require github.com/go-go-golems/geppetto v0.0.0

replace github.com/go-go-golems/geppetto => $IMPORTED_REPO
EOF

cat >"$tmpdir/main.go" <<'EOF'
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	gepa "github.com/go-go-golems/geppetto/pkg/optimizer/gepa"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type fakeEngine struct {
	n int
}

var _ engine.Engine = (*fakeEngine)(nil)

func (f *fakeEngine) RunInference(_ context.Context, _ *turns.Turn) (*turns.Turn, error) {
	f.n++
	// Produce a unique mutated prompt each call so optimizer budget advances.
	proposal := fmt.Sprintf("```Answer with OPT-%d and be concise.```", f.n)
	return &turns.Turn{
		Blocks: []turns.Block{
			turns.NewAssistantTextBlock(proposal),
		},
	}, nil
}

func main() {
	ctx := context.Background()
	examples := []any{
		map[string]any{"id": 1, "target": "x"},
		map[string]any{"id": 2, "target": "y"},
		map[string]any{"id": 3, "target": "z"},
	}

	evalFn := func(_ context.Context, c gepa.Candidate, _ int, _ any) (gepa.EvalResult, error) {
		p := c["prompt"]
		score := 0.0
		if strings.Contains(p, "OPT") {
			score = 1.0
		}
		return gepa.EvalResult{
			Score:    score,
			Feedback: fmt.Sprintf("contains_OPT=%v", strings.Contains(p, "OPT")),
		}, nil
	}

	cfg := gepa.Config{
		MaxEvalCalls: 12,
		BatchSize:    2,
		RandomSeed:   7,
	}
	fe := &fakeEngine{}
	opt := gepa.NewOptimizer(cfg, evalFn, &gepa.Reflector{
		Engine: fe,
	})

	res, err := opt.Optimize(ctx, gepa.Candidate{
		"prompt": "Base prompt without optimization marker.",
	}, examples)
	if err != nil {
		fmt.Fprintf(os.Stderr, "optimize error: %v\n", err)
		os.Exit(1)
	}

	best := res.BestCandidate["prompt"]
	fmt.Printf("calls_used=%d\n", res.CallsUsed)
	fmt.Printf("best_mean_score=%.3f\n", res.BestStats.MeanScore)
	fmt.Printf("best_prompt=%q\n", best)
	fmt.Printf("candidate_count=%d\n", len(res.Candidates))

	if !strings.Contains(best, "OPT") {
		fmt.Fprintln(os.Stderr, "expected best prompt to contain OPT")
		os.Exit(2)
	}
}
EOF

(
  cd "$tmpdir"
  GOWORK=off go mod tidy
  GOWORK=off go run .
) >"$harness_file" 2>&1

echo "Wrote:"
echo "  $build_file"
echo "  $harness_file"
