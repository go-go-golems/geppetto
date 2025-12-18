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

## turnsdatalint (Turn.Data key linting)

`turnsdatalint` is a fast `go/analysis`-based linter that enforces a project convention: **access `turns.Turn.Data[...]` using typed `const` keys** (for example `turns.DataKeyToolRegistry`) instead of ad-hoc keys. This prevents subtle drift where different packages use different string keys for the same concept, and it makes key discovery (via `turns/keys.go`) reliable.

### What it enforces

`turnsdatalint` enforces **const-only keys** for the most important “map-as-structure” fields in Turns/Blocks:

- **Turn.Data** (`map[turns.TurnDataKey]any`): key must be a `const turns.TurnDataKey`
- **Turn.Metadata** (`map[turns.TurnMetadataKey]any`): key must be a `const turns.TurnMetadataKey`
- **Block.Metadata** (`map[turns.BlockMetadataKey]any`): key must be a `const turns.BlockMetadataKey`
- **Run.Metadata** (`map[turns.RunMetadataKey]any`): key must be a `const turns.RunMetadataKey`
- **Block.Payload** (`map[string]any`): key must be a **const string** (no `"literal"` and no variables)

**Allowed examples (high-level):**
- `t.Data[turns.DataKeyToolRegistry]`
- `t.Metadata[turns.TurnMetaKeyModel]`
- `b.Metadata[turns.BlockMetaKeyMiddleware]`
- `b.Payload[turns.PayloadKeyText]`

**Flagged examples (high-level):**
- `t.Data[turns.TurnDataKey("raw")]` (ad-hoc conversion)
- `t.Metadata["model"]` (raw string literal)
- `b.Metadata["middleware"]` (raw string literal)
- `b.Payload["text"]` (raw string literal)
- `k := turns.PayloadKeyText; b.Payload[k]` (variable key)

**Note:** `t.Data["raw"]` can compile in Go because untyped string constants may be implicitly converted to a defined string type. The linter exists specifically to prevent these from creeping in.

### How it works (internals, high level)

`turnsdatalint` is implemented with `golang.org/x/tools/go/analysis`. It:

- **Finds** AST `IndexExpr` nodes that look like `<something>.Data[<key>]`
- **Verifies** the selector is a **field** named `Data` and the field type is a `map[...]...`
- **Scopes** the rule to `Data` maps whose **key type** is the configured named type (defaults to Geppetto’s `turns.TurnDataKey`)
- **Allows** only keys that are `types.Const` of that named type (identifiers like `Foo` or selectors like `turns.Foo`)
- **Reports** a diagnostic for everything else

Pseudocode:

```text
for each IndexExpr idx:
  if idx.X is not SelectorExpr ".Data": continue
  if selected field is not map[TurnDataKey]...: continue
  if idx.Index is (Ident or Selector) AND resolves to const TurnDataKey: ok
  else: report diagnostic at '['
```

### Configuration

`turnsdatalint` supports a flag for the key type:

- **Flag**: `-turnsdatalint.keytype`
- **Default**: `github.com/go-go-golems/geppetto/pkg/turns.TurnDataKey`

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


