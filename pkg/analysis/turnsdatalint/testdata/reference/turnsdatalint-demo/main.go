package main

import (
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// Reference-only demo for turnsdatalint.
//
// This file intentionally violates turnsdatalint rules to show the analyzer in action.
// It lives under a `testdata/` directory so `go list ./...` / `make lint` do not pick it up.
//
// To run the linter on this file explicitly:
//
//	cd geppetto
//	make turnsdatalint-build
//	go vet -vettool=/tmp/turnsdatalint pkg/analysis/turnsdatalint/testdata/reference/turnsdatalint-demo/main.go
//
// Expected output (violations):
//   - calling turns.DataK outside key-definition files
//   - indexing a typed-key map using raw string / untyped const
//   - indexing Block.Payload using raw string / variable
type legacyTurn struct {
	Data map[turns.TurnDataKey]any
}

func main() {
	turn := &turns.Turn{
		ID: "demo-turn",
	}

	// GOOD: wrapper stores + key methods
	_ = turns.KeyAgentMode.Set(&turn.Data, "test-mode")
	mode, _, _ := turns.KeyAgentMode.Get(turn.Data)
	fmt.Printf("Good pattern: agent mode=%q\n", mode)
	_ = engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{Enabled: true})

	// BAD: ad-hoc key constructor call (should be flagged)
	_ = turns.DataK[string]("demo", "ad_hoc", 1)

	// BAD: typed-key map index with raw string (this compiles due to implicit conversion, but is flagged)
	lt := legacyTurn{Data: map[turns.TurnDataKey]any{}}
	lt.Data["raw_string_literal"] = "this compiles but is flagged"

	// BAD: untyped string const identifier used as key (also flagged)
	const k = "raw_untyped_const"
	lt.Data[k] = "also flagged"

	// Payload rule demo (Block.Payload is map[string]any)
	b := &turns.Block{Payload: map[string]any{}}
	_ = b.Payload[turns.PayloadKeyText] // GOOD: const string key

	_ = b.Payload["text"] // BAD: raw string literal (flagged)
	payloadKey := turns.PayloadKeyText
	_ = b.Payload[payloadKey] // BAD: variable (flagged)

	fmt.Println("Demo complete - run turnsdatalint on this file explicitly to see violations")
}
