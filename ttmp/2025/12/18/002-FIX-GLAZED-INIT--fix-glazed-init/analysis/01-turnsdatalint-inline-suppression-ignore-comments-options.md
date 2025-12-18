---
Title: 'turnsdatalint: inline suppression (ignore comments) options'
Ticket: 002-FIX-GLAZED-INIT
Status: active
Topics:
    - config
    - glaze
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/geppetto-lint/main.go
      Note: Vettool bundling (go vet -vettool=...)
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Analyzer implementation discussed (const-only keys + allowlist vs directives)
    - Path: geppetto/pkg/doc/topics/12-turnsdatalint.md
      Note: Existing documentation about turnsdatalint rules
    - Path: geppetto/pkg/turns/helpers_blocks.go
      Note: Example helper functions that accept variable keys
    - Path: geppetto/pkg/turns/types.go
      Note: Example helper functions that accept variable keys
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-18T13:18:48.194152023-05:00
---


# turnsdatalint: inline suppression (ignore comments) options

## Context

`make lintmax` runs `go vet -vettool=/tmp/geppetto-lint ./...`. That vettool is built from `geppetto/cmd/geppetto-lint/main.go` and currently bundles a single custom go/analysis analyzer: `turnsdatalint` (`geppetto/pkg/analysis/turnsdatalint/analyzer.go`).

`turnsdatalint` enforces a strict project rule: **all indexing into these “map-as-structure” fields must use a `const` key**:

- `Turn.Data[...]` (typed key `TurnDataKey`)
- `Turn.Metadata[...]` (typed key `TurnMetadataKey`)
- `Block.Metadata[...]` (typed key `BlockMetadataKey`)
- `Run.Metadata[...]` (typed key `RunMetadataKey`)
- additionally for `Block.Payload[...]` (which is `map[string]any`): key must be a **const string** (no raw string literals, no variables)

This is implemented by walking `ast.IndexExpr` and checking that the index expression resolves to a `types.Const` of the correct named key type.

### Why it failed in `pkg/turns`

Helpers like:

- `SetTurnMetadata(t *Turn, key TurnMetadataKey, value any)`
- `SetBlockMetadata(b *Block, key BlockMetadataKey, value any)`
- `HasBlockMetadata(b Block, key BlockMetadataKey, value string)`
- `RemoveBlocksByMetadata(t *Turn, key BlockMetadataKey, values ...string)`

*must* accept a key parameter. Inside the helper, `key` is a **variable**, so any `t.Metadata[key]` / `b.Metadata[key]` is flagged by design.

## Question: “Is there a clean way to allow an ignore lint comment?”

Yes, it can be clean — but it’s not “free”.

### Key constraint: `go vet` has no universal inline suppression mechanism

Unlike `golangci-lint`’s `//nolint:...`, **`go vet` does not implement any standardized per-line/per-file ignore comment**. Each analyzer can implement its own suppression directives (for example, Staticcheck supports `//lint:ignore`), but that’s analyzer-specific behavior.

So: if we want `// ... ignore` comments for `turnsdatalint`, we must implement them **inside** `turnsdatalint` (and write tests for them). There isn’t a built-in “disable vet in this file” comment that go vet will obey.

## Options

### Option A: Keep analyzer strict; add a tiny allowlist for specific helper functions (what we did)

**Idea:** Allowlist a small set of helper function names inside the analyzer, and skip diagnostics for `IndexExpr` nodes that occur within those functions’ bodies.

**Pros**
- Minimal surface area: only a few helpers are exempt.
- No new “escape hatch” that can be used everywhere.
- No need to specify/parse a directive language.
- Easy to test via `analysistest` (ensure these helpers stop producing findings, and other sites still do).

**Cons**
- Special-cases function names in the analyzer.
- Not configurable at call sites.

**Risk control**
- Keep the allowlist extremely small and documented.
- Add a unit test that proves violations outside these helpers are still caught.

### Option B: Implement a suppression directive (comment-based ignore)

**Idea:** Teach `turnsdatalint` to ignore specific diagnostics when it sees a suppression directive in the AST token stream.

Common patterns used by go/analysis tools:

- **Line-level ignore**:
  - `//lint:ignore turnsdatalint reason...`
  - applied to the *next* statement or expression
- **Function-level ignore**:
  - a doc comment on a `func`:
    - `//lint:ignore turnsdatalint reason...`
  - suppresses diagnostics anywhere inside that function

**Pros**
- “Clean” UX: local, readable, and explicit.
- No hard-coded function-name allowlist in the analyzer.
- Can carry a required reason string.

**Cons / complexity**
- Requires choosing and documenting an exact directive syntax (and keeping it stable).
- Requires careful implementation so it’s not brittle:
  - multi-line comments
  - comments attached as `*ast.CommentGroup` vs “free-floating” file comments (`ast.File.Comments`)
  - “applies to next node” semantics need a well-defined target (e.g., next `ast.Node` by position)
  - performance: don’t scan *all comments* for *every IndexExpr* naively in large codebases
- Requires test coverage:
  - positive: directive suppresses the intended diagnostic
  - negative: directive does not suppress unrelated diagnostics
  - edge cases: gofmt moving comments, block comments, multiple directives

**Implementation sketch (high level)**
- Precompute a per-file suppression index once:
  - map from line → list of directives
  - map from function body ranges → function-level directives
- For each `IndexExpr` candidate:
  - check if its enclosing function has suppression
  - else check if there is a directive on the line immediately above the expression (or same line)
- If suppressed: return early (no `pass.Reportf`)

**Recommendation if we want this option**
- Reuse Staticcheck’s “shape” for familiarity:
  - `//lint:ignore turnsdatalint <reason>`
- Require `<reason>` (enforced by parsing) so suppressions are reviewable.

### Option C: Add CLI flags to ignore paths/packages in the vettool invocation

**Idea:** Add flags to `turnsdatalint` like:
- `-turnsdatalint.ignore-files='regex'`
- `-turnsdatalint.ignore-pkgs='regex'`

**Pros**
- Simple to implement.
- Useful for `testdata/`, generated code, or known “dirty” directories.

**Cons**
- Coarse-grained (not line-level).
- Easy to accidentally suppress too much.
- Doesn’t solve “I want to ignore only this one expression”.

### Option D: Disable `turnsdatalint` entirely in `make lintmax`

The vettool exposes `-turnsdatalint` (enable/disable). So you can run:

```bash
go vet -vettool=/tmp/geppetto-lint -turnsdatalint=false ./...
```

**Pros**
- Zero code changes.

**Cons**
- Removes the core safety property across the whole repo; not recommended for CI.

## “Proper” choice for this repo

Given why `turnsdatalint` exists (prevent key drift and enforce canonical typed constants), a suppression mechanism should be **hard to abuse**:

- If we want “ignore comments” primarily to support a few core helpers that accept key parameters, then an **allowlist in the analyzer** (Option A) is actually the smallest and safest change.
- If we anticipate many legitimate dynamic-key operations (e.g., generic middleware that must take a key), then the model may need to change:
  - either make those operations live behind *well-known helpers* that are allowlisted, or
  - add a tightly-scoped directive (Option B) with required reasons + tests + code review expectations.

## Current state (as of this ticket)

- We used **Option A**: allowlist the helper functions inside `turnsdatalint` so `make lintmax` passes, while keeping strict enforcement everywhere else.
- We also captured the work retroactively in the ticket diary.

## Suggested next step (if you still want comment-based suppression)

If we decide the allowlist is too “cumbersome” and want explicit call-site control, implement Option B with:

- directive syntax: `//lint:ignore turnsdatalint <reason>`
- tests in `geppetto/pkg/analysis/turnsdatalint/testdata/`
- documentation updates in:
  - `geppetto/pkg/doc/topics/12-turnsdatalint.md`

