package main

import (
	"fmt"

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
//	go vet -vettool=/tmp/turnsdatalint internal/turnsdatalint/testdata/reference/turnsdatalint-demo/main.go
//
// Expected output (3 violations):
//   - variable key (badKey)
//   - inline conversion TurnDataKey("another_bad")
//   - raw string literal "raw_string_literal" (compiles, but flagged)
func main() {
	turn := &turns.Turn{
		ID: "demo-turn",
	}

	// GOOD: Using a const TurnDataKey (should NOT be flagged)
	turn.Data = map[turns.TurnDataKey]interface{}{
		turns.DataKeyToolRegistry: "my-registry",
	}
	fmt.Printf("Good pattern: %v\n", turn.Data[turns.DataKeyToolRegistry])

	// BAD: Variable key (should be flagged)
	badKey := turns.TurnDataKey("dynamic_key")
	turn.Data[badKey] = "bad value"

	// BAD: Inline conversion (should be flagged)
	turn.Data[turns.TurnDataKey("another_bad")] = "also bad"

	// BAD: Raw string literal (compiles but flagged by linter!)
	// Go allows implicit conversion of untyped string constants to TurnDataKey,
	// so this compiles, but turnsdatalint catches it:
	turn.Data["raw_string_literal"] = "this compiles but is flagged"

	// GOOD: Using another defined const
	turn.Data[turns.DataKeyAgentMode] = "test-mode"

	fmt.Println("Demo complete - run turnsdatalint on this file explicitly to see violations")
}
