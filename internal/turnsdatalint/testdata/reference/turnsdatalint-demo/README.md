# turnsdatalint demo (reference)

This is a **reference-only** demo program that intentionally violates `turnsdatalint` rules.

It lives under `internal/turnsdatalint/testdata/` so `make lint` (which runs `go vet ./...`) **won’t see it**.

## What it demonstrates

- **Good**: using canonical `const` keys like `turns.DataKeyToolRegistry`
- **Bad (flagged)**:
  - variable keys of type `TurnDataKey`
  - ad-hoc conversions like `turns.TurnDataKey("...")`
  - raw string literals like `"..."` (these compile due to implicit conversion, but are still flagged)

## Run the linter on this demo explicitly

```bash
cd geppetto && make turnsdatalint-build && go vet -vettool=/tmp/turnsdatalint internal/turnsdatalint/testdata/reference/turnsdatalint-demo/main.go
```

## See also

- Analyzer implementation: `internal/turnsdatalint/turnsdatalint.go`
- Ticket docs (moments): `moments/ttmp/2025/12/17/002-ADD-TURN-DATA-LINTING--add-turn-data-key-linting/`


