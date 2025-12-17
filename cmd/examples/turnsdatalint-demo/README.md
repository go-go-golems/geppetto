# turnsdatalint Demo

This program intentionally violates `turnsdatalint` rules to demonstrate the analyzer in action.

## Purpose

Shows examples of:
- ✅ **Good**: Using const `TurnDataKey` values (allowed)
- ❌ **Bad**: Variables of type `TurnDataKey` (flagged)
- ❌ **Bad**: Ad-hoc conversions like `TurnDataKey("string")` (flagged)
- ❌ **Bad**: Raw string literals like `"string"` (flagged)

**Important**: While raw string literals (`turn.Data["foo"]`) *compile* successfully in Go (due to implicit conversion of untyped string constants), our linter still flags them to enforce using typed constants.

## Run the Linter

```bash
# From geppetto/ directory

# Run turnsdatalint specifically
make turnsdatalint

# Or as part of the full lint suite
make lint
```

## Expected Output

```
cmd/examples/turnsdatalint-demo/main.go:31:11: Turn.Data key must be a const of type "..." (not a conversion or variable)
cmd/examples/turnsdatalint-demo/main.go:34:11: Turn.Data key must be a const of type "..." (not a conversion or variable)
cmd/examples/turnsdatalint-demo/main.go:39:11: Turn.Data key must be a const of type "..." (not a conversion or variable)
```

## What Gets Flagged

**Line 31**: Variable key
```go
badKey := turns.TurnDataKey("dynamic_key")
turn.Data[badKey] = "bad value"  // ❌ Flagged: variable, not const
```

**Line 34**: Inline conversion
```go
turn.Data[turns.TurnDataKey("another_bad")] = "also bad"  // ❌ Flagged: inline conversion
```

**Line 39**: Raw string literal
```go
turn.Data["raw_string_literal"] = "value"  // ❌ Flagged: raw string (even though it compiles!)
```

## What's Allowed

```go
turn.Data[turns.DataKeyToolRegistry] = "my-registry"  // ✅ Const key
turn.Data[turns.DataKeyAgentMode] = "test-mode"       // ✅ Const key
```

## See Also

- Analyzer implementation: `internal/turnsdatalint/turnsdatalint.go`
- Playbook: `moments/ttmp/2025/12/17/002-ADD-TURN-DATA-LINTING--add-turn-data-key-linting/playbooks/01-building-custom-go-linters-with-go-analysis.md`

