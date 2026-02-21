---
Title: Linting in Geppetto (go/analysis and custom vettools)
Slug: geppetto-linting
Short: How Geppetto runs linting, how to add custom go/analysis analyzers, and how turnsdatalint works
Topics:
- geppetto
- lint
- tooling
- go-analysis
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

## Linting in Geppetto (go/analysis and custom vettools)

Linting is how we encode “team rules” that the Go compiler cannot enforce by itself. In Geppetto we combine standard linters (via `golangci-lint`) with **project-specific checks** implemented as `go vet` plugins using the `go/analysis` framework. This keeps checks fast, type-aware, and easy to run in CI and locally.

### What runs when you type `make lint`

`make lint` is the **blessed** command for contributors. It is expected to:

- **Build** the repo (including codegen) so type-based checks see up-to-date code
- **Run** `golangci-lint` for standard Go linting
- **Run** a bundled custom vettool (`cmd/geppetto-lint`) via `go vet -vettool=... ./...`

For custom analyzers only, use:

- **`make linttool`**: builds and runs the bundled vettool (multichecker)
- **`make turnsdatalint`**: builds and runs the single analyzer tool (singlechecker)

### How custom linters are packaged (so downstream repos can reuse them)

Geppetto packages analyzers under `pkg/analysis/<name>` (not `internal/`) so third-party projects can import them. We then provide two entrypoints:

- **Single analyzer** (debug-friendly): `cmd/<name>` using `singlechecker.Main(<name>.Analyzer)`
- **Bundled tool** (future-proof): `cmd/geppetto-lint` using `multichecker.Main(...)`

This allows downstream repos to either run Geppetto’s bundled tool directly, or embed Geppetto analyzers into their own multichecker alongside their own checks.

### How to add a new custom analyzer (checklist)

To add a new analyzer that scales with the bundling approach:

- **Create** `pkg/analysis/<name>/analyzer.go` exporting `var Analyzer *analysis.Analyzer`
- **Add tests** with `analysistest` under `pkg/analysis/<name>/testdata/src/...`
- **Register** the analyzer in `cmd/geppetto-lint/main.go`
- **Optionally** add `cmd/<name>/main.go` as a singlechecker wrapper
- **Document** it under `pkg/doc/topics/`
- **Put reference demos under `*/testdata/reference/`** so `go vet ./...` doesn’t pick them up during normal lint runs

---

## turnsdatalint (key + payload linting)

`turnsdatalint` is a fast `go/analysis`-based linter that enforces two conventions:

- **Block payload keys** must be const strings (use `turns.PayloadKeyText`, etc; no raw literals and no variables).
- **Typed key constructors** (`turns.DataK`, `turns.TurnMetaK`, `turns.BlockMetaK`) must only be called in key-definition files so the rest of the codebase reuses canonical key variables.

In current Geppetto, `Turn.Data`, `Turn.Metadata`, and `Block.Metadata` are wrapper stores (not `map[...]...` fields), so direct indexing is not possible; typed-key access is done via `key.Get/key.Set`. The linter still applies typed-key index rules to any remaining map fields (notably `Run.Metadata`).

### What it enforces

`turnsdatalint` enforces:

- **Run.Metadata** (`map[turns.RunMetadataKey]any`): key expression must have type `turns.RunMetadataKey` (no raw string literals; no untyped string const identifiers)
- **Block.Payload** (`map[string]any`): key must be a **const string** (no raw literals; no variables)
- **Key constructor locality**: `turns.DataK/TurnMetaK/BlockMetaK` calls are only allowed in generated canonical key files (`geppetto/pkg/turns/keys_gen.go`, `geppetto/pkg/inference/engine/turnkeys_gen.go`), app-level `*_keys.go`, and tests.

**Allowed examples (high-level):**
- `b.Payload[turns.PayloadKeyText]`
- `run.Metadata[turns.RunMetaKeyTraceID]`
- `turns.KeyTurnMetaProvider.Get(t.Metadata)` (wrapper store access via typed key)
- `engine.KeyToolConfig.Set(&t.Data, engine.ToolConfig{Enabled: true})` (wrapper store access via typed key)

**Flagged examples (high-level):**
- `b.Payload["text"]` (raw string literal)
- `k := turns.PayloadKeyText; b.Payload[k]` (variable key)
- `turns.DataK[string]("app", "some_key", 1)` in a non-`*_keys.go` file

**Note:** `b.Payload["text"]` can compile because payload keys are strings; the linter exists specifically to force canonical const keys for stability and searchability.

### How it works (internals, high level)

`turnsdatalint` is implemented with `golang.org/x/tools/go/analysis`. It:

- **Finds** AST `IndexExpr` nodes that look like `<something>.Payload[<key>]` and enforces const string keys.
- **Finds** calls to `turns.DataK/TurnMetaK/BlockMetaK` and enforces they only appear in key-definition files.
- **Optionally** enforces typed key expressions when indexing map fields whose key type matches configured named key types (notably `Run.Metadata` in Geppetto).

Pseudocode:

```text
for each IndexExpr idx:
  if idx.X is not SelectorExpr ".Data": continue
  if selected field is not map[TurnDataKey]...: continue
  if idx.Index has type TurnDataKey AND is not a raw literal and not an untyped-string const ident: ok
  else: report diagnostic at '['
```

### Configuration

`turnsdatalint` supports flags for the named key types (fully-qualified):

- **Flags**:
  - `-turnsdatalint.data-keytype`
  - `-turnsdatalint.turn-metadata-keytype`
  - `-turnsdatalint.block-metadata-keytype`
  - `-turnsdatalint.run-metadata-keytype`
- **Defaults**: the corresponding `github.com/go-go-golems/geppetto/pkg/turns.<...>` named types

This is mainly useful if you reuse the analyzer logic for a different struct/map key type in another project.

### Running it (in Geppetto and in third-party repos)

If another repo uses Geppetto (for example it imports `github.com/go-go-golems/geppetto/pkg/turns`), it can reuse the analyzer in two standard ways:

#### Option 1: Build Geppetto’s bundled lint tool and run it as a vettool

This is the lowest-friction approach when you’re happy to run the analyzers that Geppetto bundles:

```bash
go build -o /tmp/geppetto-lint github.com/go-go-golems/geppetto/cmd/geppetto-lint@<version>
go vet -vettool=/tmp/geppetto-lint ./...
```

#### Option 2: Embed the analyzer in your own multichecker

If you want to bundle Geppetto’s analyzers plus your own analyzers, create your own lint binary:

- **Main idea**:
  - `multichecker.Main(turnsdatalint.Analyzer, yourlint.Analyzer, ...)`
- **Import**:
  - `github.com/go-go-golems/geppetto/pkg/analysis/turnsdatalint`
