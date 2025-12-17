package main

import (
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

// This demo intentionally violates turnsdatalint rules to show the analyzer in action.
//
// Run `make lint` or `make turnsdatalint` to see the violations reported.
//
// Expected output (3 violations):
//   Line 31: variable key (badKey)
//   Line 34: inline conversion TurnDataKey("another_bad")
//   Line 39: raw string literal "raw_string_literal"

func main() {
	turn := &turns.Turn{
		ID: "demo-turn",
	}

	// GOOD: Using a const TurnDataKey (should NOT be flagged)
	turn.Data = map[turns.TurnDataKey]interface{}{
		turns.DataKeyToolRegistry: "my-registry",
	}
	fmt.Printf("Good pattern: %v\n", turn.Data[turns.DataKeyToolRegistry])

	// BAD: Ad-hoc conversion (should be flagged by turnsdatalint)
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

	fmt.Println("Demo complete - run 'make lint' to see violations")
}
