---
Title: 'Rationale: relax turnsdatalint to typed-key enforcement'
Ticket: 005-RELAX-TURNSDATALINT
Status: active
Topics:
    - infrastructure
    - inference
    - bug
DocType: analysis
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: "turnsdatalint currently enforces const-identity for Turn/Block map keys; relax it to accept any expression typed as the key type while still rejecting raw string literals and untyped string constants."
LastUpdated: 2025-12-18T18:11:25.694650523-05:00
---

## Executive summary

`turnsdatalint` currently enforces **const identity** for typed Turn/Block key maps. That implementation choice is stricter than the actual safety goal (“avoid raw string drift”) and it blocks normal Go patterns:

- typed conversions (`turns.TurnDataKey(k)`)
- typed variables (`key := turns.DataKeyFoo`)
- typed parameters (`func Set(key turns.TurnDataKey) { t.Data[key] = v }`)

We should relax the rule to **typed-key enforcement**:

- **Allow** any key expression whose type is the expected named key type (TurnDataKey, TurnMetadataKey, BlockMetadataKey, RunMetadataKey)
- **Reject** raw string literals (e.g. `t.Data["foo"]`) and “untyped string constants disguised as keys” (e.g. `const k = "foo"; t.Data[k]`)
- Keep `Block.Payload` (`map[string]any`) as **const-string only** (no literals / no vars), since it is not a typed-key map

This keeps the drift protection (no raw string literals) while removing downstream friction and “workaround APIs”.

## Current behavior (problem)

In `geppetto/pkg/analysis/turnsdatalint/analyzer.go`, keys are accepted only when the index expression resolves to a `*types.Const` of the exact named key type.

This rejects:
- `turns.TurnDataKey("raw")` (conversion call)
- `k := turns.DataKeyFoo; t.Data[k]` (variable)
- `func f(k turns.TurnDataKey) { t.Data[k] = v }` (parameter)

Downstream impact: once repos integrate Geppetto’s vettool into `lintmax`/pre-commit, this strictness forces awkward patterns or encourages `--no-verify` commits.

## Proposed rule (typed-key enforcement)

For typed-key maps (map key is one of the configured key types):

- **Allowed**: key expression has the expected named type (regardless of const/var/param/call)
- **Rejected**:
  - raw string literals (AST `BasicLit` string)
  - untyped string const identifiers/selectors used as keys (e.g. `const k = "foo"`)

For `Block.Payload` (`map[string]any`):

- **Allowed**: const strings (typed or untyped const), via identifiers/selectors only
- **Rejected**: raw string literals and variables

## Implementation sketch (analyzer)

- Replace `isAllowedConstKey` with `isAllowedTypedKeyExpr`:
  - `unwrapParens`
  - reject `*ast.BasicLit` string literals
  - reject `Ident`/`SelectorExpr` that refer to a `*types.Const` whose declared type is “untyped string”
  - otherwise accept if `pass.TypesInfo.Types[e].Type` is the expected named key type
- Remove the helper allowlist (`isInsideAllowedHelperFunction`) to avoid “holes” where raw literals slip through inside allowed helpers

## Test updates

Update `pkg/analysis/turnsdatalint/testdata/src/a/a.go`:

- flip existing “badConversion / badVar” to **allowed** for typed-key maps
- add new failing cases:
  - `t.Data["raw"]` (raw string literal)
  - `const k = "raw"; t.Data[k]` (untyped string const)
- keep payload tests const-only as-is

## References

- Analyzer implementation: `/home/manuel/workspaces/2025-11-18/fix-pinocchio-profiles/geppetto/pkg/analysis/turnsdatalint/analyzer.go`
- Analyzer tests: `/home/manuel/workspaces/2025-11-18/fix-pinocchio-profiles/geppetto/pkg/analysis/turnsdatalint/analyzer_test.go`
- Linting docs: `/home/manuel/workspaces/2025-11-18/fix-pinocchio-profiles/geppetto/pkg/doc/topics/12-turnsdatalint.md`
- Downstream analysis (Pinocchio): `/home/manuel/workspaces/2025-11-18/fix-pinocchio-profiles/pinocchio/ttmp/2025/12/18/001-FIX-KEY-TAG-LINTING--fix-key-tag-linting-errors/analysis/01-turnsdatalint-why-dynamic-keys-conversions-fail-options-to-fix-pinocchio-geppetto.md`
