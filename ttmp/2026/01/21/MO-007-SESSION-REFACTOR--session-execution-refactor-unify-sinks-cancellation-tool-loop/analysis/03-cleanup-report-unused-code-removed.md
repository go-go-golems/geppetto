---
Title: "Cleanup report: unused/legacy code removed during MO-007"
Ticket: MO-007-SESSION-REFACTOR
Status: active
Topics:
  - inference
  - cleanup
  - events
  - tooling
DocType: analysis
Intent: long-term
---

# Cleanup report: unused/legacy code removed during MO-007

## Why this report exists

MO-007 refactored the inference lifecycle across geppetto + pinocchio (and bobatea integration) around `session.Session` + `ExecutionHandle` + `ToolLoopEngineBuilder`, and standardized **context-only event sinks**.

That refactor left behind several “zombie” abstractions and legacy helper sections that:

- were no longer exercised by the canonical inference path,
- were confusing to new developers (multiple ways to do the same thing), and/or
- pulled dependencies in the wrong direction (conversation-manager based logic inside Turn-based flows).

This report lists what was removed, why it was removed, and what evidence was used to determine it was safe.

## How we decided something was removable

### Signals used (ordered roughly by trust)

1. **Workspace typecheck (pinocchio + geppetto together)**
   - Confirms pinocchio is compiling against the workspace geppetto module (not a released version).
   - Command (from `oak-git-db/`):
     - `VET=off GOCACHE=/tmp/go-build-cache bash scripts/typecheck-geppetto-pinocchio.sh`

2. **Tests + lint in each repo**
   - `cd geppetto && go test ./... -count=1`
   - `cd pinocchio && go test ./... -count=1`
   - plus repo hooks (golangci-lint, go vet, etc).

3. **PR-shape analysis vs `origin/main`**
   - What changed in the PR (file list, deleted/added sections) used to find “likely legacy edges”.

4. **oak-git-db SQLite evidence (per-repo + multi-repo)**
   - “What changed” queries: `pr_file`
   - Definition spans (base vs head): `oak_match`
   - Typed call edges (best effort): `go_ref(kind='call')`

### Key SQL queries (examples)

List PRs (multi-repo DB):

```sql
SELECT pr.id AS pr_id, r.name AS repo, r.root_path,
       substr(pr.merge_base_sha,1,7) AS merge_base7,
       substr(bs.sha,1,7) AS base7,
       substr(hs.sha,1,7) AS head7
FROM pr
JOIN repo r ON r.id=pr.repo_id
JOIN snapshot bs ON bs.id=pr.base_snapshot_id
JOIN snapshot hs ON hs.id=pr.head_snapshot_id
ORDER BY pr.id;
```

List changed Go files (example: geppetto PR = `pr_id=1`):

```sql
SELECT p.path, pf.change_type, old.path AS old_path
FROM pr_file pf
JOIN path p ON p.id=pf.path_id
LEFT JOIN path old ON old.id=pf.old_path_id
WHERE pf.pr_id=1
  AND p.path LIKE '%.go'
  AND p.path NOT LIKE 'ttmp/%'
ORDER BY pf.change_type, p.path;
```

Find callers of a symbol (example: `session.Session.StartInference` in geppetto head snapshot):

```sql
WITH head AS (SELECT head_snapshot_id sid FROM pr WHERE id=1)
SELECT caller.symbol_key AS caller, p.path, r.line, r.col
FROM go_ref r
JOIN go_symbol caller ON caller.id=r.from_symbol_id
LEFT JOIN path p ON p.id=r.path_id
WHERE r.snapshot_id=(SELECT sid FROM head)
  AND r.kind='call'
  AND r.to_symbol_id=1192
ORDER BY caller.symbol_key, p.path, r.line;
```

Heuristic “unexported funcs/methods with zero incoming call edges” (geppetto head snapshot):

```sql
WITH head AS (SELECT head_snapshot_id sid FROM pr WHERE id=1)
SELECT
  gs.id, gs.kind, gs.name, gs.symbol_key, p.path, gs.start_line,
  COUNT(r.to_symbol_id) AS call_incoming
FROM go_symbol gs
JOIN head h ON gs.snapshot_id=h.sid
LEFT JOIN path p ON p.id=gs.path_id
LEFT JOIN go_ref r ON r.snapshot_id=h.sid AND r.kind='call' AND r.to_symbol_id=gs.id
WHERE p.path LIKE 'pkg/%'
  AND p.path NOT LIKE '%_test.go'
  AND gs.kind IN ('func','method')
  AND gs.name GLOB '[a-z]*'
GROUP BY gs.id, gs.kind, gs.name, gs.symbol_key, p.path, gs.start_line
HAVING call_incoming = 0
ORDER BY p.path, gs.start_line
LIMIT 200;
```

Caveat: `go_ref` covers call edges only and can miss dynamic usage (reflection/config/plugin). We only used it as a starting point; changes were validated by compilation/tests.

## Items removed (by category)

### 1) Engine-config sinks / engine option plumbing

**What was removed**

- The “engine construction takes sinks/options” pathway (historically `engine.WithSink(...)`, `engine.Option`, etc).
- This included the dedicated option plumbing file:
  - `geppetto/pkg/inference/engine/options.go` (deleted earlier during MO-007).

**Why it was removed**

- The canonical design is now: **engines publish events via context only**.
- Sinks are attached at runtime (e.g. `events.WithEventSinks(ctx, sinks...)`) or set on `ToolLoopEngineBuilder.EventSinks`.
- Keeping engine-config sinks would preserve two competing patterns and reintroduce ordering problems with stricter providers.

**Evidence**

- Session-based examples and pinocchio frontends run with context-only sinks.
- Docs were updated to stop teaching engine-config sinks.

### 2) EngineConfig (legacy builder “fingerprint”)

**What was removed**

- `geppetto/pkg/inference/builder.EngineConfig` (and the entire `geppetto/pkg/inference/builder/` package).
- Parsed-layers builder “split API” that returned `(engine, sink, EngineConfig, error)`.

**Why it was removed**

- It existed to support signature-based recomposition in an older builder style (`BuildConfig`/`BuildFromConfig`).
- After MO-007, `session.ToolLoopEngineBuilder` is the canonical composition point.
- Keeping EngineConfig increased conceptual surface area without providing value to the current flow.

**Impact / follow-up**

- Updated example builders and pinocchio’s parsed-layers builder to return only `(engine.Engine, events.EventSink, error)`.

### 3) Unused ConversationState snapshot/mutation layer (inside `pkg/conversation`)

**What was removed**

- `geppetto/pkg/conversation/state.go` (ConversationState + SnapshotConfig)
- `geppetto/pkg/conversation/mutations.go`
- `geppetto/pkg/conversation/state_test.go`

**Why it was removed**

- The new canonical multi-turn container is `session.Session` holding `[]*turns.Turn` snapshots.
- The ConversationState layer encouraged provider-specific ordering validation at the wrong abstraction level.
- It was not used by the new Session-based flows.

### 4) Small, clearly unused helpers

**What was removed**

- `geppetto/pkg/helpers/jsonschema.go`: `checkFirstArgContext` (had `//nolint:unused`).
- `geppetto/pkg/inference/session/session.go`: `ErrSessionNoTurns` (unused error var).

**Why it was removed**

- Both were dead code and increased cognitive load.
- Removing them reduced lint noise and made the public surface tighter.

**Evidence**

- No workspace references via ripgrep.
- Tests + workspace typecheck passed.

### 5) Legacy conversation-based tool helpers hiding inside Turn-based toolhelpers

**What was removed**

From `geppetto/pkg/inference/toolhelpers/helpers.go`:

- `ExtractToolCalls(conv conversation.Conversation) []ToolCall`
- `ExecuteToolCalls(ctx, toolCalls, registry) []ToolResult`
- `AppendToolResults(conv, results) conversation.Conversation`

…and the associated imports that kept the conversation-manager dependency alive inside toolhelpers:

- `github.com/go-go-golems/geppetto/pkg/conversation`
- `github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api`

**Why it was removed**

- The canonical tool calling orchestration is `RunToolCallingLoop(ctx, eng, turn, reg, cfg)` working over **Turn blocks**, not conversation messages.
- These helpers were a partial, earlier experiment for conversation-message tool parsing and were not used by the current code paths.

**Notes**

- `RunToolCallingLoop` remains and uses `toolblocks.ExtractPendingToolCalls` and the context registry.

### 6) Legacy conversation parsing and unused local helpers inside Turn tool middleware

**What was removed**

From `geppetto/pkg/inference/middleware/tool_middleware.go`:

- `ToolResult.ToMessage()` (conversation Message conversion)
- `extractToolCalls(*conversation.Message)` and its OpenAI/Claude metadata parsing
- local helpers duplicated elsewhere:
  - `extractPendingToolCalls` (duplicate of `toolblocks.ExtractPendingToolCalls`)
  - `appendToolResultsBlocks` (duplicate of `toolblocks.AppendToolResultsBlocks`)

Also removed the corresponding tests that validated the deleted conversation parsing function.

**Why it was removed**

- After MO-007, tool calling is standardized around **Turn blocks** and tool loop orchestration in helpers/session.
- Keeping conversation-message parsing inside the middleware created two tool-call extraction strategies.
- The “local duplicates” were simply redundant.

**Test changes**

- Updated turn middleware tests to use shared `toolblocks` helpers.

## Non-goals / things intentionally not removed

- `geppetto/pkg/conversation/manager` APIs still exist and are used by JS bindings and some docs/examples. This report only covers the unused ConversationState snapshot/mutation add-on we already stopped using.
- The `middleware.NewToolMiddleware` itself remains (examples still use it), even though `ToolLoopEngineBuilder` is preferred for production chat apps.

## Risk / correctness notes

- Typed call edges (`go_ref(kind='call')`) are not a proof of reachability. Dynamic usage is possible.
- For every deletion we still required:
  - `go test ./...` passing in geppetto
  - and the combined workspace typecheck passing (pinocchio compiling against workspace geppetto).

## Appendix: commits that performed cleanup

Geppetto:
- `d6a0f54` — remove engine option/config sink plumbing
- `f9d61c4` — docs: migrate to Session API playbook
- `e4518d7` — cleanup: drop EngineConfig and unused ConversationState
- `4691f89` — cleanup: drop unused helpers
- `e1cfea2` — cleanup: drop legacy conversation tool helpers

Pinocchio:
- `554d947` — cleanup: simplify ParsedLayersEngineBuilder

Bobatea:
- `c2a08dc` — remove AutoStartBackend/StartBackendMsg (lifecycle cleanup; affects how pinocchio starts inference)
