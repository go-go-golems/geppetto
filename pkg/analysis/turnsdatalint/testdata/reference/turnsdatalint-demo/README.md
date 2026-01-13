# turnsdatalint demo (reference)

This is a **reference-only** demo program that intentionally violates `turnsdatalint` rules.

It lives under `pkg/analysis/turnsdatalint/testdata/` so `make lint` (which runs `go vet ./...`) **won’t see it**.

## What it demonstrates

- **Good**: using wrapper stores + canonical typed keys (e.g. `turns.KeyAgentMode.Set(&t.Data, ...)`)
- **Bad (flagged)**:
  - calling `turns.DataK/TurnMetaK/BlockMetaK` outside key-definition files (`*_keys.go`)
  - indexing a typed-key map field with a raw string / untyped const (these can compile due to implicit conversion)
  - indexing `Block.Payload` with a raw string or variable (must use const strings like `turns.PayloadKeyText`)

## Run the linter on this demo explicitly

```bash
cd geppetto && make turnsdatalint-build && go vet -vettool=/tmp/turnsdatalint pkg/analysis/turnsdatalint/testdata/reference/turnsdatalint-demo/main.go
```

## See also

- Analyzer implementation: `pkg/analysis/turnsdatalint/analyzer.go`
- Ticket docs (moments): `moments/ttmp/2025/12/17/002-ADD-TURN-DATA-LINTING--add-turn-data-key-linting/`

