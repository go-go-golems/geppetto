---
Title: Diary
Ticket: 002-IMPLEMENT-TYPE-DATA-ACCESSOR
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Step-by-step implementation diary for typed Turn.Data/Metadata accessors migration"
LastUpdated: 2025-12-22T16:00:00-05:00
WhatFor: "Track implementation progress, document decisions, failures, and learnings"
WhenToUse: "Reference during implementation to understand context and avoid repeating mistakes"
---

# Diary

## Goal

Document the step-by-step implementation of typed Turn.Data/Metadata accessors. This diary captures what changed, why it changed, what happened (including failures), and what we learned during the migration from `map[TurnDataKey]any` to opaque wrapper with typed `Get[T]`/`Set[T]` accessors.

---

## Step 1: Create Ticket and Initial Research

This step established the implementation ticket and performed comprehensive codebase research to map all locations where Turn.Data, Turn.Metadata, and Block.Metadata are accessed across geppetto, moments, pinocchio, and bobatea repositories.

**Commit (code):** N/A — documentation only

### What I did

- Created ticket `002-IMPLEMENT-TYPE-DATA-ACCESSOR` using docmgr
- Performed systematic grep searches for `.Data[`, `.Metadata[` patterns across all repositories
- Used codebase semantic search to find Turn creation/initialization sites
- Read key files:
  - `geppetto/pkg/turns/types.go` (core type definitions)
  - `geppetto/pkg/turns/keys.go` (canonical keys)
  - `moments/backend/pkg/turnkeys/*.go` (moments-specific keys)
  - `geppetto/pkg/turns/serde/serde.go` (serialization)
  - `geppetto/pkg/analysis/turnsdatalint/analyzer.go` (linter)
  - Multiple middleware files showing access patterns
- Created comprehensive analysis document mapping ~136 access sites

### Why

- Need complete inventory before migration to avoid missing locations
- Design doc (`001-REVIEW-TYPED-DATA-ACCESS`) specifies breaking change — all direct map access must migrate
- Understanding access patterns helps prioritize migration order and identify special cases

### What worked

- Grep searches found ~130+ files with Turn.Data/Metadata access
- Semantic search identified initialization patterns
- File-by-file review revealed helper function usage patterns (`SetTurnMetadata`, `HasBlockMetadata`)
- Identified special case: compression middleware takes `map[string]any` (needs refactoring)

### What didn't work

- Initial grep for `.Data\[` missed some patterns (needed broader regex)
- Some access sites hidden in helper functions (had to trace call sites)

### What I learned

- **Access distribution:**
  - Geppetto: ~16 sites (core types, helpers, tool helpers)
  - Moments: ~115 sites (heavy middleware usage)
  - Pinocchio: ~5 sites (agentmode middleware)
  - Bobatea: 0 sites (TUI library, no Turn interaction)

- **Common patterns:**
  1. Nil map initialization: `if t.Data == nil { t.Data = map[...]any{} }`
  2. Type assertion reads: `value, _ := t.Data[key].(string)`
  3. Direct writes: `t.Data[key] = value`
  4. Helper function usage: `turns.HasBlockMetadata(b, key, value)` (very common in moments)

- **Special cases identified:**
  - Compression middleware (`turn_data_compressor.go`) takes `map[string]any` — needs API redesign
  - Serialization (`serde.go`) normalizes nil maps — wrapper handles this internally
  - Linter (`turnsdatalint`) allows typed conversions `TurnDataKey("...")` — loophole to close

### What was tricky to build

- **Counting access sites accurately:** Helper functions like `HasBlockMetadata` are called 30+ times but counted as one site. Need to distinguish "definition sites" vs "usage sites" for migration planning.

- **Identifying all initialization patterns:** Some Turn structs created via struct literals `&Turn{Data: map[...]any{}}`, others initialized later. Both patterns need migration.

### What warrants a second pair of eyes

- **Compression middleware refactoring:** Design doc suggests `CompressKnownKeys` approach, but current code iterates generically. Need review to ensure compression logic can work with typed keys.

- **Helper function migration strategy:** `SetTurnMetadata`, `HasBlockMetadata` are used extensively. Should we:
  1. Update helpers to use wrapper API internally (backward compatible)?
  2. Deprecate helpers, force direct wrapper API usage?
  3. Keep helpers as convenience wrappers?

- **Linter enhancement scope:** Design doc specifies 6 new rules. Need to verify linter can detect wrapper API violations (ban `t.Data[key]` after migration).

### What should be done in the future

- **Migration phases:** Break into phases:
  1. Implement wrapper types + API (geppetto only)
  2. Migrate geppetto code (low risk, core library)
  3. Migrate moments middleware (high impact, test thoroughly)
  4. Migrate pinocchio (low impact)
  5. Enhance linter (enforce wrapper API)

- **Test coverage:** Create test suite for wrapper API before migration:
  - Nil map handling
  - Type assertion errors
  - Serialization round-trip
  - Error handling (Set with non-serializable value)

- **Documentation:** Update middleware creation guide to show new API patterns

### Code review instructions

- Start with analysis document: `analysis/01-codebase-analysis-turn-data-metadata-access-locations.md`
- Verify access site counts match actual codebase (grep results may have false positives)
- Review special cases (compression, serialization, linter) for migration feasibility

### Technical details

**Grep commands used:**
```bash
grep -r '\.Data\[' geppetto/ moments/ pinocchio/ bobatea/
grep -r '\.Metadata\[' geppetto/ moments/ pinocchio/ bobatea/
grep -r 'SetTurnMetadata\|SetBlockMetadata\|HasBlockMetadata' geppetto/ moments/ pinocchio/
```

**Key files identified:**
- Core types: `geppetto/pkg/turns/types.go`
- Keys: `geppetto/pkg/turns/keys.go`, `moments/backend/pkg/turnkeys/*.go`
- Serialization: `geppetto/pkg/turns/serde/serde.go`
- Linter: `geppetto/pkg/analysis/turnsdatalint/analyzer.go`
- Middleware examples: `moments/backend/pkg/inference/middleware/current_user_middleware.go`, `pinocchio/pkg/middlewares/agentmode/middleware.go`

**Access site breakdown:**
- Turn.Data: ~50 sites
- Turn.Metadata: ~18 sites  
- Block.Metadata: ~68 sites
- Total: ~136 sites

### What I'd do differently next time

- Start with helper function analysis (they're used everywhere, understanding them first would have saved time)
- Use AST-based search tools (like `gopls` or `go/ast`) for more accurate pattern matching
- Create migration checklist with file-by-file status tracking

---

## Step 2: Pull docmgr tasks + confirm baseline code shape

This step turned the high-level checklist into an actionable, checkable work plan by pulling the ticket’s docmgr tasks (with stable IDs). It also confirmed the exact “before” state in the code: `Turn.Data`, `Turn.Metadata`, and `Block.Metadata` are still raw maps, helper functions still exist, and serde currently normalizes nil maps — all of which must change per the final design.

**Commit (code):** N/A — documentation only

### What I did

- Ran `docmgr task list --ticket 002-IMPLEMENT-TYPE-DATA-ACCESSOR` to fetch task IDs (1–113)
- Read the current implementations:
  - `geppetto/pkg/turns/types.go` (Turn/Block structs and metadata helper setters)
  - `geppetto/pkg/turns/key_types.go` (key type aliases)
  - `geppetto/pkg/turns/keys.go` (current “flat” key constants without namespace/version)
  - `geppetto/pkg/turns/helpers_blocks.go` (helper functions like `HasBlockMetadata`, `WithBlockMetadata`)
  - `geppetto/pkg/turns/serde/serde.go` (normalizes nil maps on Turn and Block)

### Why

- The implementation ticket is large; docmgr task IDs let us move in small, reviewable steps while keeping progress visible.
- Understanding the exact baseline state reduces “surprise churn” when we start migrating call sites.

### What worked

- docmgr tasks are already fully enumerated and match the design doc’s phases (Core API → Linter → Geppetto cleanup).
- The baseline is consistent with the analysis doc: raw map usage + helper functions + serde nil-map normalization.

### What didn't work

- N/A

### What I learned

- The current `turns` package still exposes map fields directly:
  - `Turn.Data map[TurnDataKey]interface{}`
  - `Turn.Metadata map[TurnMetadataKey]interface{}`
  - `Block.Metadata map[BlockMetadataKey]interface{}`
- This means we must do a **breaking API shift** to opaque wrapper fields to prevent bypassing.
- `serde.NormalizeTurn` currently initializes nil maps to empty maps; with wrappers, that responsibility moves into `Set` and YAML (un)marshal helpers.

### What was tricky to build

- N/A (this was a discovery/grounding step)

### What warrants a second pair of eyes

- Whether we should implement wrapper types in `types.go` (per tasks) vs split into dedicated files for readability, without making review harder.
- Ensuring the eventual wrapper/YAML behavior still matches the current `omitempty` behavior expectations across consumers.

### What should be done in the future

- As each major core API piece lands, immediately `docmgr task check` the corresponding IDs and keep diary entries tight to those step boundaries.

### Code review instructions

- Review `geppetto/pkg/turns/types.go` first to confirm the “before” public API is raw-map based.
- Review `geppetto/pkg/turns/serde/serde.go` to confirm current nil-map normalization behavior that we’ll be changing.

### Technical details

**docmgr tasks:** `docmgr task list --ticket 002-IMPLEMENT-TYPE-DATA-ACCESSOR` (IDs 1–113)

### What I'd do differently next time

- Keep a small “before/after API sketch” in the diary early, so subsequent steps can refer to it without re-explaining.

---

## Step 3: Introduce opaque wrappers for Turn.Data/Metadata and Block.Metadata

This step implemented the core “opaque wrapper” move: `Turn.Data`, `Turn.Metadata`, and `Block.Metadata` are no longer public maps. Instead, they became wrapper fields that own the underlying map and enforce access via a typed-key API. This is the structural guardrail the design doc relies on: callers can’t bypass invariants by writing directly into maps.

**Commit (code):** b86cb63ab746c3049cbdf9bd6d8804356026ec5a — "turns: typed wrappers + namespaced keys; migrate geppetto" (`geppetto/`)

### What I did

- Updated `geppetto/pkg/turns/types.go`:
  - Changed structs:
    - `Turn.Data` → `turns.Data`
    - `Turn.Metadata` → `turns.Metadata`
    - `Block.Metadata` → `turns.BlockMetadata`
  - Added key identity + constructor:
    - `Key[T]` (typed wrapper)
    - `NewTurnDataKey(namespace, value, version)` producing `"namespace.value@vN"`
    - `K[T](namespace, value, version)` helper
  - Implemented wrapper functionality:
    - `Len`, `Range`, `Delete`
    - YAML marshal/unmarshal (`MarshalYAML`, `UnmarshalYAML`) converting to/from `map[string]any`
  - Added JSON serializability validation on writes via `json.Marshal(value)` before storing

### Why

- Opaque wrappers are the only way to make “no bypasses” true in Go without relying solely on linting.
- Centralizing nil initialization and validation avoids the codebase-wide pattern `if t.Data == nil { ... }`.
- YAML behavior must stay human-friendly and stable, so wrapper marshal/unmarshal converts keys to strings.

### What worked

- The wrapper API captures the required invariants:
  - nil map is handled internally
  - serializability is checked at write time
  - YAML serialization stays `map[string]any` (readable)

### What didn't work

- Initially, I implemented wrapper methods as generic methods (e.g. `func (d *Data) Set[T any](...)`).
- The repo’s linter rejected this with:
  - `method must have no type parameters`

### What I learned

- Go (as of the toolchain used in this repo) does **not** allow methods to declare their own type parameters. The compiler rejects this with: `syntax error: method must have no type parameters`.
- We need the “typed API” semantics but must implement them as **generic functions** rather than methods.

### What was tricky to build

- Preserving `omitempty` semantics while changing from a map field to a struct wrapper:
  - we must ensure empty wrappers serialize to `nil` during YAML marshal
  - we must avoid serde logic that force-initializes empty maps

### What warrants a second pair of eyes

- Wrapper marshal/unmarshal contract:
  - does `MarshalYAML` returning `nil` correctly omit fields everywhere this Turn struct is used?
  - do we want stricter key-format validation at unmarshal time (currently we accept and rely on linting)?

### What should be done in the future

- Align repo tooling so generic methods are either supported, or we consistently stick to generic functions (current approach).

### Code review instructions

- Start in `geppetto/pkg/turns/types.go`, review:
  - the new wrapper field types on `Turn` and `Block`
  - `NewTurnDataKey`/`K`
  - YAML marshal/unmarshal behavior

### Technical details

- **Error observed:** `geppetto/pkg/turns/types.go:... method must have no type parameters`

### What I'd do differently next time

- Validate repo lint/tooling constraints on generics before implementing method-level generics.

---

## Step 4: Rework the typed API to avoid generic methods

This step kept the design semantics (typed keys + typed read/write) while adapting to a Go language constraint (enforced by the compiler): methods can’t declare their own type parameters. The fix was to switch from methods to package-level generic functions with explicit receiver arguments.

**Commit (code):** b86cb63ab746c3049cbdf9bd6d8804356026ec5a — "turns: typed wrappers + namespaced keys; migrate geppetto" (`geppetto/`)

### What I did

- Updated `geppetto/pkg/turns/types.go`:
  - Replaced generic methods with generic functions:
    - `turns.DataSet(&t.Data, key, value)` and `turns.DataGet(t.Data, key)`
    - `turns.MetadataSet(&t.Metadata, key, value)` and `turns.MetadataGet(t.Metadata, key)`
    - `turns.BlockMetadataSet(&b.Metadata, key, value)` and `turns.BlockMetadataGet(b.Metadata, key)`
  - Kept non-generic helpers (`Len`, `Range`, `Delete`, YAML marshal/unmarshal) as methods.

### Why

- We still need type inference at call sites and typed error reporting on mismatches.
- Switching to functions keeps the API enforceable while satisfying the repo’s tooling.

### What worked

- Lints for `geppetto/pkg/turns/types.go` cleared after the change.
- Call sites can still read cleanly and type-safely:
  - `v, ok, err := turns.DataGet(t.Data, turns.KeyAgentMode)`
  - `err := turns.DataSet(&t.Data, engine.KeyToolConfig, cfg)`

### What didn't work

- N/A (this step directly addressed the blocking lint error)

### What I learned

- The “opaque boundary” can still work without generic methods; generic functions are sufficient.

### What was tricky to build

- Avoiding churn: a lot of future migrations are easier if function names are short and consistent.

### What warrants a second pair of eyes

- Naming and API ergonomics: are `DataGet/DataSet` the right shape vs methods, given this repo’s style?

### What should be done in the future

- If we ever allow generic methods in tooling, we could evaluate switching back to a method-based API for ergonomics.

### Code review instructions

- Review `geppetto/pkg/turns/types.go` for the final “typed API surface” and make sure no generic methods remain.

---

## Step 5: Migrate geppetto canonical keys without introducing an import cycle

This step migrated geppetto’s canonical keys to the new namespace/value/version scheme and introduced typed `Key[T]` values for the common primitives. A key detail: `engine` already imports `turns` (Engine interface uses `*turns.Turn`), so `turns` cannot import `engine` for `ToolConfig` without a cycle. The solution was to place the `ToolConfig` typed key in `engine` instead.

**Commit (code):** b86cb63ab746c3049cbdf9bd6d8804356026ec5a — "turns: typed wrappers + namespaced keys; migrate geppetto" (`geppetto/`)

### What I did

- Updated `geppetto/pkg/turns/keys.go`:
  - Introduced `turns.GeppettoNamespaceKey = "geppetto"`
  - Added per-key “value key” consts
  - Added typed keys:
    - Turn.Data: `turns.KeyAgentMode`, `turns.KeyAgentModeAllowedTools`, `turns.KeyResponsesServerTools`
    - Turn.Metadata: `turns.KeyTurnMetaModel`, `turns.KeyTurnMetaProvider`, etc.
    - Block.Metadata: `turns.KeyBlockMetaMiddleware`, `turns.KeyBlockMetaClaudeOriginalContent`, etc.
  - Removed the old `DataKey*` / `TurnMetaKey*` / `BlockMetaKey*` flat constants
- Added `geppetto/pkg/inference/engine/turnkeys.go`:
  - `engine.KeyToolConfig = turns.K[engine.ToolConfig](turns.GeppettoNamespaceKey, turns.ToolConfigValueKey, 1)`
  - This avoids the `turns -> engine -> turns` cycle.

### Why

- The design doc requires stable, versioned, namespaced identities for keys.
- Import cycles are a hard constraint; the engine-owned typed key is the simplest resolution.

### What worked

- Key identity is now explicit and versioned (`@v1`) rather than implicit.
- We have a clean place to keep keys that depend on `engine` types.

### What didn't work

- Attempting to define `KeyToolConfig` in `turns` would create an import cycle due to `engine` importing `turns`.

### What I learned

- Key ownership must sometimes follow type ownership: if the value type lives in `engine`, the typed key likely lives in `engine`.

### What was tricky to build

- Balancing “canonical key definition lives in turns” vs “avoid cycles”: we can keep the namespace/value consts in `turns` while placing typed keys that require engine types in `engine`.

### What warrants a second pair of eyes

- Whether `ToolConfigValueKey` belongs in `turns` or should also be moved into `engine` (currently: const in `turns`, typed key var in `engine`).

### What should be done in the future

- Apply the same “avoid cycles” rule for other cross-package typed keys.

### Code review instructions

- Review `geppetto/pkg/turns/keys.go` and `geppetto/pkg/inference/engine/turnkeys.go` together.

---

## Step 6: Migrate geppetto call sites (and remove helper functions)

This step migrated all geppetto call sites we touched away from direct map access and removed now-banned helper functions (`WithBlockMetadata`, `HasBlockMetadata`, `RemoveBlocksByMetadata`, plus the `Set*Metadata` setters in `types.go`). The goal was to keep geppetto compiling once the wrappers and keys changed.

**Commit (code):** b86cb63ab746c3049cbdf9bd6d8804356026ec5a — "turns: typed wrappers + namespaced keys; migrate geppetto" (`geppetto/`)

### What I did

- Removed helper functions in `geppetto/pkg/turns/helpers_blocks.go`:
  - `WithBlockMetadata`
  - `HasBlockMetadata`
  - `RemoveBlocksByMetadata`
  - (and their direct-map usage)
- Updated core code to wrapper API:
  - `geppetto/pkg/turns/serde/serde.go`: removed nil-map normalization for Data/Metadata/Block.Metadata; kept payload stabilization and role synthesis
  - `geppetto/pkg/inference/middleware/systemprompt_middleware.go`: replaced metadata writes with `turns.BlockMetadataSet`
  - `geppetto/pkg/inference/middleware/tool_middleware.go`: reads allowed tools via `turns.DataGet(..., turns.KeyAgentModeAllowedTools)`
  - `geppetto/pkg/inference/toolhelpers/helpers.go`: sets tool config via `turns.DataSet(..., engine.KeyToolConfig, ...)`
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`: reads tool config via `turns.DataGet(..., engine.KeyToolConfig)`
  - `geppetto/pkg/steps/ai/claude/helpers.go`: reads Claude original content via `turns.BlockMetadataGet(..., turns.KeyBlockMetaClaudeOriginalContent)`
- Updated examples to stop constructing `Turn{Data: map[...]...}` and to use typed setters:
  - `geppetto/cmd/examples/middleware-inference/main.go`
  - `geppetto/cmd/examples/claude-tools/main.go`
  - `geppetto/cmd/examples/openai-tools/main.go`
  - `geppetto/cmd/examples/generic-tool-calling/main.go`
- Updated serde tests to use wrapper API:
  - `geppetto/pkg/turns/serde/serde_test.go`

### Why

- Once `Turn.Data` is no longer a map, old patterns simply can’t compile.
- We want geppetto to migrate first (smaller surface area), then proceed to moments/pinocchio.

### What worked

- The call-site migrations mostly became mechanical:
  - map assignment → `turns.DataSet/MetadataSet/BlockMetadataSet`
  - map read + type assert → `turns.DataGet/MetadataGet/BlockMetadataGet`

### What didn't work

- Any remaining references to old `DataKey*` / `TurnMetaKey*` / `BlockMetaKey*` constants became immediate compile errors after key migration.

### What I learned

- Tests are a great canary: once serde tests compile and pass, the wrapper/YAML contract is likely close to correct.

### What was tricky to build

- Ensuring serde doesn’t “force materialize” empty maps:
  - the wrapper must stay nil internally unless written to
  - YAML marshal should omit empty wrappers

### What warrants a second pair of eyes

- The OpenAI engine path:
  - confirm `ToolConfig` default behavior is still correct when config is missing
  - confirm error wrapping provides enough context without being noisy

### What should be done in the future

- Add focused unit tests for `DataSet/DataGet` serializability and type mismatch errors (beyond serde round-trip).

### Code review instructions

- Start at `geppetto/pkg/turns/types.go` and `geppetto/pkg/turns/keys.go`
- Then review the migrated call sites:
  - `geppetto/pkg/inference/toolhelpers/helpers.go`
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `geppetto/pkg/inference/middleware/systemprompt_middleware.go`

---

## Step 7: Migrate pinocchio to the wrapper API (agentmode + server-tools + sqlitetool)

This step brought pinocchio back into alignment with the new API by removing direct map usage and migrating middleware patterns that relied on removed helper functions. The biggest change here was agentmode, which used `RemoveBlocksByMetadata` and `WithBlockMetadata` heavily.

**Commit (code):** f48a9bd81507bf5e1323f7e1d8e6b52d57b1e057 — "turns: migrate pinocchio to typed Data/Metadata wrappers" (`pinocchio/`)

### What I did

- Updated `pinocchio/pkg/middlewares/agentmode/middleware.go`:
  - Replaced direct Turn.Data reads/writes with `turns.DataGet/DataSet`
  - Replaced helper usage (`RemoveBlocksByMetadata`, `WithBlockMetadata`) with inline logic using `turns.BlockMetadataGet/Set`
  - Switched to canonical geppetto keys for agentmode tags/mode names:
    - `turns.KeyBlockMetaAgentModeTag`, `turns.KeyBlockMetaAgentMode`
- Updated server-tools seed writes to use typed setter:
  - `pinocchio/cmd/examples/simple-chat/main.go`
  - `pinocchio/cmd/agents/simple-chat-agent/main.go`
- Updated tool-loop backend to stop using map-based initialization and removed a convenience method that performed dynamic key conversion:
  - `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go`
- Updated sqlitetool middleware to read DSN via wrapper API and introduced pinocchio-local typed keys:
  - `pinocchio/pkg/middlewares/sqlitetool/middleware.go`

### Why

- pinocchio is a consumer of geppetto’s `turns` types, so it must follow the new API boundary.
- Removing dynamic key conversion is aligned with the “no escape hatches” design.

### What worked

- Agentmode logic migrated cleanly by:
  - reading tag values via `BlockMetadataGet`
  - filtering blocks by tag values
  - writing new metadata via `BlockMetadataSet`

### What didn't work

- N/A (but this step is sensitive to correctness because it changes block filtering semantics)

### What I learned

- The wrapper API makes “metadata tagging” more explicit and less error-prone than raw map access.

### What was tricky to build

- Replacing `RemoveBlocksByMetadata` without reintroducing hidden behavior:
  - we must preserve idempotency by removing prior inserted blocks based on the tag value.

### What warrants a second pair of eyes

- Agentmode block removal logic:
  - confirm we’re filtering exactly the same blocks as before (tag values and conditions).

### What should be done in the future

- Add a regression test for agentmode ensuring the same blocks are removed/inserted as before.

### Code review instructions

- Focus on `pinocchio/pkg/middlewares/agentmode/middleware.go` and validate the removal + insertion logic.

---

## Step 8: Start moments key migration (turnkeys package)

This step began the moments-side key migration by converting `moments/backend/pkg/turnkeys/*` to typed keys `turns.K[...]` with an explicit `MentoNamespaceKey`. At this point, call-site migration in moments is still in progress.

**Commit (code):** 520ad4ee327c408a3f7a0b065b6320d7f309aea2 — "turnkeys: migrate moments keys to typed Key[T]" (`moments/`)

### What I did

- Updated moments key definition files:
  - `moments/backend/pkg/turnkeys/data_keys.go`:
    - introduced `MentoNamespaceKey` and per-key value consts
    - replaced old `turns.TurnDataKey` consts with typed `var` keys
    - removed legacy/compatibility keys (per spec: no backward compatibility)
  - `moments/backend/pkg/turnkeys/turn_meta_keys.go`:
    - switched to typed keys and reused `turns.KeyTurnMetaModel` for the shared model key
  - `moments/backend/pkg/turnkeys/block_meta_keys.go`:
    - switched to typed keys for memory context/extraction markers

### Why

- moments has the largest surface area of `Turn.Data` and `Block.Metadata` usage; moving keys first sets us up for mechanical call-site updates.

### What worked

- We now have a single namespace/value/version scheme in moments keys, aligned with the design doc.

### What didn't work

- moments call sites are still referencing the old `turnkeys.*` constants as raw map keys; those must now migrate to wrapper API and new typed keys.

### What I learned

- moments has a lot of “typed map helpers” that accept `map[turns.TurnDataKey]any` — those need API redesign to accept wrapper types or use `Range`.

### What was tricky to build

- Some moments “keys” previously used dotted strings like `"mento.team.suggestions"`.
  - We switched to underscore-ish value keys under the `mento` namespace (e.g. `team_suggestions`) to match the design doc’s `namespace.value@vN` convention.

### What warrants a second pair of eyes

- Key renames in moments:
  - confirm that any persisted YAML snapshots will not require migration in this repository context (breaking change is acceptable per spec, but we should understand the blast radius).

### What should be done in the future

- Migrate moments call sites next, starting with `current_user_middleware`, `thinkingmode`, `promptutil`, and memory middleware.

### Code review instructions

- Review the new key definitions in `moments/backend/pkg/turnkeys/*.go` for naming, types, and consistency.

---

## Git state (important)

We created the foundational checkpoint commits (even though the overall workspace may not compile yet, per “massive fundamentals PR” workflow):

- `geppetto/`: b86cb63ab746c3049cbdf9bd6d8804356026ec5a — "turns: typed wrappers + namespaced keys; migrate geppetto"
- `pinocchio/`: f48a9bd81507bf5e1323f7e1d8e6b52d57b1e057 — "turns: migrate pinocchio to typed Data/Metadata wrappers"
- `moments/`: 520ad4ee327c408a3f7a0b065b6320d7f309aea2 — "turnkeys: migrate moments keys to typed Key[T]"

Next: commit documentation updates (diary + design-doc corrections), then proceed with moments call-site migration in small, frequent commits.
