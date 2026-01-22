---
Title: Turn creation + TurnID propagation
Ticket: GP-05-TURN-CREATION
Status: active
Topics:
    - geppetto
    - turns
    - inference
    - design
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: Caller-side cloneTurn keeps Turn.ID stable across prompts
    - Path: pkg/doc/playbooks/04-migrate-to-session-api.md
      Note: Documented UI pattern for creating follow-up turns
    - Path: pkg/doc/topics/08-turns.md
      Note: Conceptual model Turn=one inference cycle; Block.TurnID semantics
    - Path: pkg/inference/session/session.go
      Note: StartInference clones turns; current Turn.ID/backfill behavior
    - Path: pkg/inference/session/tool_loop_builder.go
      Note: Runner invariants for Turn.ID/InferenceID (currently only-if-empty)
    - Path: pkg/turns/keys.go
      Note: Turn meta keys vs block meta keys (no block inference id key yet)
    - Path: pkg/turns/types.go
      Note: Turn/Block structs; AppendBlock currently does not stamp TurnID
ExternalSources: []
Summary: Where follow-up turns are cloned/transformed; TurnID generation and propagation to blocks; block-level inference attribution.
LastUpdated: 2026-01-22T17:45:00-05:00
WhatFor: ""
WhenToUse: ""
---



# Turn creation + TurnID propagation

## Goal

Map (with concrete code references) how Geppetto creates “follow-up turns” for subsequent inferences
within the same session, and how identity is generated and propagated:

- `SessionID` (long-lived session correlation)
- `InferenceID` (unique per inference execution)
- `Turn.ID` (the “turn id” used for correlation in events and snapshots)
- `Block.TurnID` and block metadata attribution (especially “which inference produced this block”)

This document is intentionally **analysis-first**: it aims to make current behavior and invariants
explicit, identify gaps, and propose a consistent contract and implementation strategy.

## Mental model: “snapshot turns” with per-block attribution

In this model, each inference cycle produces a new `Turn` snapshot that contains the full history so
far, and blocks are attributed to the cycle that created them.

Example timeline (same SessionID, two inferences):

```
SessionID = S

Inference #1:
  Turn snapshot T1 (Turn.ID = T1, InferenceID = I1)
    Blocks: [U1, A1]
    Block attribution:
      U1.TurnID = T1, U1.inference_id = I1
      A1.TurnID = T1, A1.inference_id = I1

Inference #2:
  Turn snapshot T2 (Turn.ID = T2, InferenceID = I2)
    Blocks: [U1, A1, U2, A2]
    Block attribution:
      U1.TurnID = T1, U1.inference_id = I1   (reused)
      A1.TurnID = T1, A1.inference_id = I1   (reused)
      U2.TurnID = T2, U2.inference_id = I2   (new)
      A2.TurnID = T2, A2.inference_id = I2   (new)
```

This matches your intent: “subsequent turns in the same session” + “reused blocks keep the TurnID /
InferenceID of the inference that created them”.

## Definitions (current structs)

### Turn and blocks

- `turns.Turn` (`geppetto/pkg/turns/types.go`)
  - `ID string`
  - `Blocks []turns.Block`
  - `Metadata turns.Metadata` (typed key wrapper)
  - `Data turns.Data` (typed key wrapper)
- `turns.Block`
  - `ID string` (set by most block constructors)
  - `TurnID string` (**currently often empty unless backfilled**)
  - `Metadata turns.BlockMetadata` (typed key wrapper)

### Turn-level correlation keys (typed)

From `geppetto/pkg/turns/keys.go`:

- `turns.KeyTurnMetaSessionID` (`geppetto.session_id@v1`)
- `turns.KeyTurnMetaInferenceID` (`geppetto.inference_id@v1`)

### Block-level metadata keys (typed)

From `geppetto/pkg/turns/keys.go`:

- There are some block metadata keys (`middleware`, `tool_calls`, etc),
  but there is **no block-level inference id key** today.

## Where follow-up turns are created / cloned today

There are two broad “turn creation” mechanisms in the current design:

1) **Core session clones the last turn at `StartInference` time** (Geppetto-owned).
2) **Callers clone the last turn to build a new seed turn** (Pinocchio-owned patterns).

### 1) `Session.StartInference` clones the latest session turn

In `geppetto/pkg/inference/session/session.go`, `(*Session).StartInference`:

- picks `input := s.Turns[len(s.Turns)-1]`
- creates a shallow copy: `inputCopy := *input`
- clones `inputCopy.Metadata` and `inputCopy.Data`
- deep-copies each block’s payload map and clones each block’s metadata
- best-effort backfills `Block.TurnID` **only when empty**:
  - `if b.TurnID == "" { b.TurnID = inputCopy.ID }`
- stamps session + inference metadata on the copied turn:
  - `KeyTurnMetaSessionID` = `s.SessionID`
  - `KeyTurnMetaInferenceID` = `uuid.NewString()`

Critical detail: `inputCopy.ID` is only generated if empty:

```go
if inputCopy.ID == "" {
    inputCopy.ID = uuid.NewString()
}
```

So **Turn.ID is stable across subsequent inferences** as long as the stored session turn already has
an ID.

### What a “Turn” represents in practice (snapshot, not delta)

The common runtime pattern today is that each inference execution produces a **new turn snapshot**
that contains the *entire conversation so far* as an ever-growing `[]Block`:

- seed turn contains the initial system/user blocks
- inference appends assistant/tool blocks to the turn
- the next user prompt is created by cloning the entire prior turn and appending a new user block

So “subsequent turns in the same session” means “subsequent snapshots of the conversation state”,
not “a brand-new empty turn with only the delta”.

This is why the “reused blocks keep their original TurnID/InferenceID” model is coherent: a later
turn snapshot contains blocks from multiple prior inference executions.

### 2) Callers clone turns when building “seed for prompt”

Pinocchio’s webchat and agent backends build a new “seed” turn for each prompt by cloning the
previous `Session.Latest()` turn and appending a new user block:

- `pinocchio/pkg/webchat/router.go` (`seedForPrompt`, `cloneTurn`)
- `pinocchio/pkg/ui/backend.go` (`snapshotForPrompt`, `cloneTurn`)
- `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go` (`snapshotForPrompt`, `cloneTurn`)

These helpers currently copy `Turn.ID` forward:

```go
return &turns.Turn{
    ID:       t.ID,
    Blocks:   append([]turns.Block(nil), t.Blocks...),
    Metadata: t.Metadata.Clone(),
    Data:     t.Data.Clone(),
}
```

So the dominant “follow-up turn” construction pattern today also keeps Turn.ID stable across
inference executions.

### Docs encode the same “clone latest + append prompt + append to session” pattern

`geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md` recommends the following shape for UIs:

```go
seed := clone(b.sess.Latest())
turns.AppendBlock(seed, turns.NewUserTextBlock(prompt))
b.sess.Append(seed)
handle, err := b.sess.StartInference(ctx)
```

This is consistent with the Pinocchio implementations above and is the primary mechanism by which
“subsequent inference within a session” is triggered in production code.

## Where Turn.ID is generated today (or not)

### Session path

- `Session.NewSession()` generates a `SessionID` immediately.
- `Session.StartInference` generates `Turn.ID` only when empty.
- `ToolLoopEngineBuilder` runner (`geppetto/pkg/inference/session/tool_loop_builder.go`) generates:
  - `Turn.ID` only when empty,
  - `InferenceID` only when missing from `Turn.Metadata`.

### Block creation path

- Block constructors in `geppetto/pkg/turns/helpers_blocks.go` set `Block.ID` but do **not** set
  `Block.TurnID`.
- `turns.AppendBlock` is a plain append and does **not** set `Block.TurnID`.

Provider engines (OpenAI/Claude/Gemini/Responses) populate *event* metadata (`EventMetadata.TurnID =
t.ID`) but do not stamp `Block.TurnID` on newly appended blocks.

As a consequence, blocks created during inference are typically missing `Block.TurnID` unless some
other code stamps it.

## Where Turn.ID is propagated to blocks today

### Current behavior

The only place in Geppetto core that backfills `Block.TurnID` is inside
`Session.StartInference`, and it only applies to the **input copy before inference starts**.

There is no symmetric “after inference” normalization that stamps `Block.TurnID` on blocks appended
during the inference execution (assistant text, tool calls, tool uses, etc).

### Implication

For a typical flow:

1. `seedForPrompt` clones prior turn and appends a new user block (block has `TurnID == ""`)
2. `sess.Append(seed)`
3. `sess.StartInference()` clones `seed` to `inputCopy` and fills missing block TurnIDs
4. engine/tool-loop appends new blocks to the turn during the run (these blocks have `TurnID == ""`)
5. session appends output turn snapshot without post-stamping block IDs

…blocks produced by inference may end up with empty `Block.TurnID`.

## Block-level InferenceID attribution (requested)

You want blocks created by an inference to be attributable to that inference execution.

Today:

- Turns have `KeyTurnMetaInferenceID` (turn-level).
- Blocks have `Block.Metadata`, but there is no `KeyBlockMetaInferenceID` and no code that stamps
  inference identity into blocks.

## Desired contract (proposed)

### 1) Turn.ID should be fresh per inference execution

Within one long-lived session (same SessionID), each call to `StartInference` should run against a
turn snapshot with a **new `Turn.ID`**.

This implies Turn.ID becomes “per inference execution turn snapshot id”, not “stable session id”.

### 2) Block.TurnID should represent the creator turn snapshot

- Reused/copied blocks keep their existing `Block.TurnID` (and should not be rewritten).
- Blocks created as part of the current inference should have `Block.TurnID == current Turn.ID`.

### 3) Blocks should carry a block-level InferenceID (new key)

Add a block metadata key (name TBD) that stores the inference id that created the block, e.g.:

- `turns.KeyBlockMetaInferenceID` (backed by `BlockMetaK[string]("geppetto", "inference_id", 1)`)

Then:

- blocks created during the current inference get `block.metadata["geppetto.inference_id@v1"] == TurnMetaInferenceID`
- reused blocks keep their original value (or remain unset if legacy)

## Recommended implementation strategy (high-level)

### A) Decide where Turn.ID is generated (two viable policies)

There are two coherent ways to make `Turn.ID` fresh per inference while keeping it stable throughout
that inference (so events, snapshots, and block attribution all correlate):

#### Policy A (recommended): generate fresh Turn.ID when constructing the “prompt seed turn”

This matches the documented UI pattern (“clone latest + append prompt + append seed + start
inference”) and makes “Turn = one inference cycle” literal:

- When creating the follow-up seed turn for the next prompt, assign `seed.ID = uuid.NewString()`
  unconditionally.
- `Session.StartInference` should then treat a non-empty `Turn.ID` as authoritative and preserve it
  (today it already does).
- The inference runner and the engines keep using `t.ID` as the event `TurnID`.

Pros:

- Seed and output for the same inference cycle share the same `Turn.ID`.
- No “hidden” ID changes inside `StartInference`; the caller controls the cycle boundary.

Cons:

- Requires updating all caller clone helpers (`cloneTurn`/`seedForPrompt` patterns).

#### Policy B: generate fresh Turn.ID inside `Session.StartInference` on the internal copy

- `Session.StartInference` always assigns `inputCopy.ID = uuid.NewString()`, regardless of the
  input’s existing ID.

Pros:

- Caller code doesn’t need to be aware of Turn.ID changes to get per-inference correlation.

Cons:

- If the caller appends a “seed turn” into session history before `StartInference`, that seed turn’s
  ID won’t match the inference’s Turn.ID (unless we also rewrite/stamp it).
- The “turn boundary” becomes implicit, and it becomes harder to reason about seed vs output
  snapshots in session history.

Given your stated intent (“subsequent turns in the same session” + “Turn = one inference cycle”),
Policy A is the better fit: it keeps the cycle boundary explicit and keeps seed/output aligned.

### B) Normalize blocks after inference completes

After `runner.RunInference` returns `out` (still within the session goroutine, before `s.Append(out)`):

1) Ensure `out.ID` and `out.Metadata` have `{SessionID, InferenceID}` (already done today).
2) For each block in `out.Blocks`:
   - if `block.TurnID == ""`, set it to `out.ID`
   - if `block.TurnID == out.ID` and `KeyBlockMetaInferenceID` missing, set it to the inference id

This “post-stamp” is what makes blocks produced during the inference self-identifying.

Concrete pseudocode (post-inference):

```go
sid, _ := turns.KeyTurnMetaSessionID.Get(out.Metadata)    // best-effort
iid, _ := turns.KeyTurnMetaInferenceID.Get(out.Metadata)  // best-effort
_ = sid

for i := range out.Blocks {
    b := out.Blocks[i]
    if b.TurnID == "" {
        b.TurnID = out.ID
    }
    if b.TurnID == out.ID {
        if _, ok, _ := turns.KeyBlockMetaInferenceID.Get(b.Metadata); !ok {
            _ = turns.KeyBlockMetaInferenceID.Set(&b.Metadata, iid)
        }
    }
    out.Blocks[i] = b
}
```

### C) Consider moving some stamping to `turns.AppendBlock`

If we want to eliminate “empty `Block.TurnID`” by construction, the most centralized approach is to
make `turns.AppendBlock` do:

```go
if b.TurnID == "" && t != nil && t.ID != "" {
    b.TurnID = t.ID
}
t.Blocks = append(t.Blocks, b)
```

This preserves the “reused blocks keep their old TurnID” rule (only fills when empty) while ensuring
new blocks appended during inference carry the current Turn.ID.

Whether `AppendBlock` should also stamp `KeyBlockMetaInferenceID` depends on whether the turn already
has `KeyTurnMetaInferenceID` at the time of append (it might not for user blocks created before
`StartInference`).

## Sites to revisit if implementing (checklist)

- Geppetto core
  - `geppetto/pkg/inference/session/session.go` (generate fresh Turn.ID + post-stamp blocks)
  - `geppetto/pkg/inference/session/tool_loop_builder.go` (align runner invariants with new policy)
  - `geppetto/pkg/turns/types.go` (`AppendBlock` behavior)
  - `geppetto/pkg/turns/keys.go` (add `KeyBlockMetaInferenceID`)
- Callers (likely need changes if Turn.ID becomes per-inference)
  - `pinocchio/pkg/webchat/router.go` (`cloneTurn` should not copy ID forward for follow-up seeds)
  - `pinocchio/pkg/ui/backend.go` (same)
  - `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go` (same)

## Open questions / decisions to make explicitly

1) If a historical block has empty `Block.TurnID`, do we:
   - backfill it to the *previous* turn id (if present),
   - leave it empty (legacy), or
   - backfill it to the current new turn id (risk mis-attribution)?
2) Should `Turn.ID` be treated as “per inference turn id” everywhere (including filenames, stores,
   UI entity IDs), and if so, what is the stable identifier for “conversation/thread” (SessionID)?
3) Do we want a block-level `SessionID` too, or is block-level `InferenceID` sufficient when paired
   with session-level metadata?

## Appendix: Where blocks are added/modified, and how `TurnID` is (not) stamped

This section inventories the main code paths that (a) append new blocks to a turn, or (b) modify or
reorder existing blocks, and documents how `Block.TurnID` is currently set (and where it is missing).

### Summary: there is no “stamp TurnID on new blocks” invariant today

- `turns.AppendBlock` is a raw append and does **not** set `Block.TurnID`.
- Block constructors (`turns.NewUserTextBlock`, `turns.NewAssistantTextBlock`, etc.) set `Block.ID`
  but do **not** set `Block.TurnID`.
- The only best-effort backfill of `Block.TurnID` in core code happens **before inference starts**
  (inside `Session.StartInference`, on the input copy, and only when empty).

So blocks appended during inference (assistant/tool blocks) often end up with `Block.TurnID == ""`.

### A) User prompt entrypoint: “follow-up seed turn” creation

The most common place a new *user* block is added for a follow-up inference is in the caller/UI
layer, not in Geppetto core:

- Pattern (documented): `clone(sess.Latest()) → AppendBlock(user) → sess.Append(seed) → StartInference`
  - `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md`
- Pinocchio webchat: `seedForPrompt` appends a new user block:
  - `pinocchio/pkg/webchat/router.go` (see `seedForPrompt` + `turns.AppendBlock(seed, turns.NewUserTextBlock(prompt))`)

TurnID behavior:

- The user block is created with `TurnID == ""` (constructors don’t stamp it).
- There is no stamping at append time (`turns.AppendBlock` does not stamp).
- `Session.StartInference` later backfills `Block.TurnID` on the input copy **only if empty**, and
  it uses the copied turn’s `ID` (which is often stable across inferences today).

### B) Middleware that inserts/edits blocks (pre-inference)

#### 1) System prompt middleware: insert or edit first system block

`geppetto/pkg/inference/middleware/systemprompt_middleware.go`:

- If a system block exists, it mutates the existing block’s payload text:
  - `t.Blocks[firstSystemIdx].Payload[text] = ...`
- If no system block exists, it creates a new system block and prepends it:
  - `newBlock := turns.NewSystemTextBlock(prompt)`
  - `t.Blocks = append([]turns.Block{newBlock}, t.Blocks...)`
- It also sets `KeyBlockMetaMiddleware = "systemprompt"` on the inserted/edited block.

TurnID behavior:

- When inserting a new system block, `newBlock.TurnID` is empty and is not stamped by the middleware.
- If `Session.StartInference` clones and backfills input blocks, it can fill the missing TurnID on
  the *input copy* (again, only pre-inference and only if empty).

#### 2) Tool result reorder middleware: reorder blocks for provider adjacency constraints

`geppetto/pkg/inference/middleware/reorder_tool_results_middleware.go`:

- Rebuilds the block slice (`newBlocks := ...; t.Blocks = newBlocks`) to move matching `tool_use`
  blocks immediately after contiguous `tool_call` runs.

TurnID behavior:

- The middleware copies blocks as values into a new slice; it does not set or adjust `Block.TurnID`.
- It preserves whatever `Block.TurnID` values already exist.

### C) Tool loop execution: append `tool_use` blocks

Tool results are appended as new `tool_use` blocks in a few places:

- Turn-based toolblocks helper:
  - `geppetto/pkg/inference/toolblocks/toolblocks.go` (`AppendToolResultsBlocks`)
  - It appends via `turns.AppendBlock(t, turns.NewToolUseBlock(...))`.
- Tool middleware (stub) uses the helper:
  - `geppetto/pkg/inference/middleware/tool_middleware.go` (`toolblocks.AppendToolResultsBlocks(updated, shared)`)

TurnID behavior:

- `turns.NewToolUseBlock` does not set `Block.TurnID`.
- `turns.AppendBlock` does not stamp `Block.TurnID`.
- There is no post-tool stamping pass in toolblocks/tool middleware, so `tool_use` blocks typically
  have empty TurnID unless later normalized elsewhere.

### D) Provider engines: append assistant/tool_call/reasoning blocks

Provider engines generally mutate the passed-in `*turns.Turn` in place and append new blocks to it:

- OpenAI: `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - appends `llm_text` and `tool_call` blocks via `turns.AppendBlock(...)`
- Claude: `geppetto/pkg/steps/ai/claude/engine_claude.go`
  - appends `llm_text` and `tool_call` blocks via `turns.AppendBlock(...)`
- Gemini: `geppetto/pkg/steps/ai/gemini/engine_gemini.go`
  - appends `llm_text` and `tool_call` blocks via `turns.AppendBlock(...)`
- OpenAI Responses: `geppetto/pkg/steps/ai/openai_responses/engine.go`
  - appends `llm_text` and `tool_call` blocks via `turns.AppendBlock(...)`
  - may also append `reasoning` blocks (explicit `turns.Block{Kind: BlockKindReasoning, Payload: ...}` then `AppendBlock`)
  - may add provider payload fields (e.g. `PayloadKeyItemID`, `PayloadKeyEncryptedContent`) before appending

TurnID behavior:

- Engines publish events whose `EventMetadata.TurnID` is set from `t.ID`, but the blocks they append
  do not get `Block.TurnID` assigned (constructors don’t set it, and `turns.AppendBlock` doesn’t).
- So, unless some other normalization step runs after inference, blocks produced by engines will
  frequently have `Block.TurnID == ""`.

### E) Block modification patterns (not just append)

In addition to appending blocks, some code paths modify existing blocks:

- **System prompt middleware** mutates the payload of an existing system block:
  - `t.Blocks[firstSystemIdx].Payload[text] = ...`
- **Reorder middleware** rebuilds the entire block slice and assigns `t.Blocks = newBlocks` (reorder only).

These operations preserve any existing `Block.TurnID` values, but they do not correct missing
TurnIDs and they can insert new blocks with empty TurnID (systemprompt insertion).

### F) Current “TurnID stamping” coverage (what is actually guaranteed today)

Today, the only place that attempts to set `Block.TurnID` in core code is:

- `geppetto/pkg/inference/session/session.go` (`Session.StartInference`):
  - clones the last session turn into an input copy
  - for each block on the **input copy**, if `b.TurnID == ""`, sets it to `inputCopy.ID`

What this does *not* cover:

- blocks appended during inference (assistant text, tool calls, tool uses, reasoning blocks)
- blocks inserted by middleware via direct slice operations (`t.Blocks = append(...)`)
- block-level inference attribution (there is no block-level inference id key today)

### G) Implications for the “snapshot turn + per-block attribution” model

With the snapshot model, a later turn snapshot contains blocks created across multiple inferences.
That means:

- **Reused blocks must keep their original attribution** (TurnID + block-level inference id).
- **New blocks must be stamped deterministically** with the current cycle’s TurnID and inference id.

Today, neither of these is guaranteed:

- reused blocks may have empty `Block.TurnID` (legacy/missing)
- new blocks appended by engines/toolblocks generally have empty `Block.TurnID`
- blocks carry no `InferenceID` attribution at all

### H) Recommended enforcement points (where to “make it impossible”)

There are two practical “choke points” that cover the majority of block creation sites:

1) **Normalize blocks after inference completes** (Session-level post-pass; see earlier pseudocode)
   - ensures blocks appended during inference are stamped
   - can also stamp a new block-level inference id key (proposed `turns.KeyBlockMetaInferenceID`)
2) **Stamp on append** by changing `turns.AppendBlock` (and related helpers) to fill missing `Block.TurnID`
   from `t.ID`
   - reduces the chance of producing unstamped blocks in the first place
   - still needs special-case thinking for blocks inserted by direct slice manipulation (e.g. systemprompt)

In practice, doing both is robust:

- `AppendBlock` prevents most “empty TurnID” blocks going forward
- the session post-pass backfills any remaining edge cases and sets block-level inference attribution
