---
Title: Diary
Ticket: GP-05-TURN-CREATION
Status: active
Topics:
    - geppetto
    - turns
    - inference
    - design
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: Primary caller pattern for follow-up turn creation
    - Path: pkg/inference/session/session.go
      Note: Primary site investigated for turn cloning and stamping
ExternalSources: []
Summary: Investigation diary for turn cloning/creation and per-inference TurnID/block attribution.
LastUpdated: 2026-01-22T17:45:00-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Track investigation steps for GP-05: where follow-up turns are created/cloned for subsequent
inferences within a session, how `Turn.ID` is generated, and how (and whether) IDs propagate to
blocks (`Block.TurnID` + block metadata such as `InferenceID` attribution).

## Update (2026-01-23)

This diary captures the investigation that led to the simplifications in this ticket, but it
predates the final implementation decisions:

- `turns.Block` no longer has a `TurnID` field.
- Prompt turns are created via `Session.AppendNewTurnFromUserPrompt(s)`.
- `Session.StartInference` now runs against the latest appended turn in-place (no internal cloning,
  no “append output turn” step).

## Step 1: Restate intent + locate current Turn/Block ID behaviors

I first captured the intent precisely in the ticket `index.md` and then traced the actual “turn
creation” and “turn cloning” paths in the core session runner (`geppetto/pkg/inference/session`) and
in the most important downstream caller patterns (Pinocchio webchat + agent backends).

The immediate goal of this step was to answer three questions with code references: (1) do we
generate a fresh `Turn.ID` per inference today, (2) when do blocks get a non-empty `Block.TurnID`,
and (3) do blocks carry an `InferenceID` attribution at all.

**Commit (code):** N/A

### What I did
- Updated ticket intent writeup:
  - `geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/index.md`
- Added an initial task breakdown:
  - `geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/tasks.md`
- Read core definitions:
  - `geppetto/pkg/turns/types.go` (`Turn`, `Block`, `BlockMetadata`)
  - `geppetto/pkg/turns/helpers_blocks.go` (block constructors)
  - `geppetto/pkg/turns/keys.go` (Turn meta keys vs Block meta keys)
- Read core session execution path:
  - `geppetto/pkg/inference/session/session.go` (turn cloning, Turn.ID generation, inference metadata)
  - `geppetto/pkg/inference/session/tool_loop_builder.go` (runner invariants)
- Traced follow-up “seed for prompt” patterns in Pinocchio (important because it is the common
  caller pattern for “subsequent inference within a session”):
  - `pinocchio/pkg/webchat/router.go` (`seedForPrompt`, `cloneTurn`)
  - `pinocchio/pkg/ui/backend.go` (same `snapshotForPrompt` pattern)
  - `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go` (same pattern)
- Commands used:
  - `rg -n "StartInference\\(|cloneTurn\\b|TurnID" -S geppetto pinocchio`
  - `rg -n "type Turn struct|type Block struct|BlockMetadata" -S geppetto/pkg/turns`

### Why
- We need a code-accurate map before changing invariants; otherwise we risk “fixing” Turn IDs while
  leaving block attribution inconsistent (or vice versa).

### What worked
- Found that `Session.StartInference` explicitly clones the last session turn and tags per-inference
  metadata (`geppetto/pkg/inference/session/session.go`), which is the right place to reason about
  Turn.ID / InferenceID semantics.

### What didn't work
- N/A (this step was repository reading + mapping).

### What I learned
- **Turn.ID is not fresh per inference today**: `Session.StartInference` only assigns a new `Turn.ID`
  when the input turn’s `ID` is empty; otherwise it reuses the existing ID.
- **Blocks don’t automatically get TurnID at creation time**: block constructors set `Block.ID` but
  do not set `Block.TurnID`, and `turns.AppendBlock` is a plain append with no ID propagation.
- The only current “best effort” `Block.TurnID` backfill I found is on the *input copy* inside
  `Session.StartInference` (it sets `b.TurnID` only when empty). There is no symmetric “after
  inference” backfill for blocks appended during the run.
- **Blocks do not appear to carry an `InferenceID` attribution** today (no `KeyBlockMetaInferenceID`
  exists in `geppetto/pkg/turns/keys.go`).

### What was tricky to build
- N/A (analysis-only step), but the subtlety is that Turn and Block IDs currently mean different
  things in different places (some callers treat Turn.ID as stable session identity).

### What warrants a second pair of eyes
- Whether `Turn.ID` should become “per inference execution” (fresh every StartInference) versus stay
  “stable conversation snapshot id” (current behavior), because downstream systems currently assume
  stability in a few helper functions (e.g. Pinocchio’s `cloneTurn` copies `ID` forward).

### What should be done in the future
- Define and enforce a single consistent contract:
  - when a new Turn.ID is generated for a follow-up inference
  - how block attribution (`Block.TurnID` and a new block-level `InferenceID`) is stamped

### Code review instructions
- Start with `geppetto/pkg/inference/session/session.go` for current cloning + ID behavior.
- Then cross-check caller patterns in `pinocchio/pkg/webchat/router.go` (`seedForPrompt` + `cloneTurn`).

### Technical details
- N/A (no implementation changes yet).

## Step 2: Trace the “snapshot turn” model and identify mismatched invariants

I traced how “one inference cycle” maps onto the actual runtime data model: each inference produces
a new *snapshot* of the conversation (a `Turn` whose `Blocks` contains the full history so far), and
subsequent inferences clone that snapshot and append a new user block. This makes block-level
attribution important: the same snapshot contains blocks from multiple inferences.

This step focused on pinpointing where the system currently violates the intended invariants: Turn
IDs are stable across inferences (not fresh), and blocks appended during inference do not reliably
get a `TurnID` (or any `InferenceID`) stamped onto them.

**Commit (code):** N/A

### What I did
- Read and cross-checked the “turn snapshot” caller patterns:
  - `pinocchio/pkg/webchat/router.go` (`seedForPrompt`, `cloneTurn`)
  - `pinocchio/pkg/ui/backend.go` (`snapshotForPrompt`, `cloneTurn`)
  - `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go` (`snapshotForPrompt`, `cloneTurn`)
- Verified that these helpers preserve `Turn.ID` across prompts/inferences (they copy `ID: t.ID`).
- Read the tool loop runner invariants:
  - `geppetto/pkg/inference/session/tool_loop_builder.go` (ensures `Turn.ID` only if empty; ensures `InferenceID` only if missing; does not stamp blocks)
- Cross-checked the documentation model:
  - `geppetto/pkg/doc/topics/08-turns.md` describes Turn as “one inference cycle” and `Block.TurnID`
    as a “parent turn reference” (this implies Turn.ID changes per inference, and blocks can be
    attributed).

### Why
- If we make Turn.ID “fresh per inference”, we need to ensure the snapshot model still works: old
  blocks in new snapshots must retain their original attribution, while new blocks must be stamped
  correctly.

### What worked
- The snapshot model aligns naturally with block-level attribution:
  - later snapshots contain blocks from multiple inferences, so “reused blocks keep old TurnID /
    InferenceID” is coherent.

### What didn't work
- Current code does not fully support the attribution model:
  - `turns.AppendBlock` does not stamp `Block.TurnID`.
  - Session-level stamping only backfills `Block.TurnID` on the *input copy*, not on blocks appended
    during inference.
  - There is no block-level `InferenceID` metadata key today.

### What I learned
- Today, the system behaves as if `Turn.ID` is “stable session/conversation id”, despite docs
  describing it as “one inference cycle”.
- Fixing this is not just a session change: callers that clone turns (Pinocchio) currently preserve
  `Turn.ID`, and the core turn helper (`AppendBlock`) does not guarantee block attribution.

### What was tricky to build
- Separating three different “identity layers” that are currently conflated in practice:
  - Session identity (`SessionID`) — stable across the whole chat thread
  - Inference identity (`InferenceID`) — stable across one tool loop execution
  - Turn snapshot identity (`Turn.ID`) — should be fresh per inference, but is currently stable

### What warrants a second pair of eyes
- The choice of where to implement block stamping:
  - “post-stamp” after inference in `Session.StartInference` versus
  - moving it into `turns.AppendBlock` (which would change behavior globally, likely desirable but
    needs review)

### What should be done in the future
- Make `Turn.ID` fresh per inference execution (within a Session), and ensure every block created
  during that inference is stamped with:
  - `Block.TurnID == Turn.ID`
  - `Block.Metadata[geppetto.inference_id@v1] == Turn.Metadata[geppetto.inference_id@v1]` (new key)

### Code review instructions
- Start at `geppetto/pkg/inference/session/session.go` and `geppetto/pkg/inference/session/tool_loop_builder.go`.
- Then verify caller-side cloning in `pinocchio/pkg/webchat/router.go`.

### Technical details
- N/A (no implementation changes yet).

## Step 3: Cross-check documentation patterns vs current ID semantics

I cross-checked Geppetto’s own documentation around turns/sessions to ensure the analysis is aligned
with intended usage. This matters because the “clone latest turn and append prompt” pattern is
documented as the canonical way UIs create follow-up turns, and any TurnID changes must keep this
pattern safe and non-surprising.

This step primarily confirmed that documentation already *conceptually* treats a Turn as “one
inference cycle”, which matches your desired semantics of “fresh Turn.ID per inference” and block
attribution — but the current implementation does not fully enforce it.

**Commit (code):** N/A

### What I did
- Reviewed:
  - `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md` (UI pattern: clone latest, append prompt, append to session, start inference)
  - `geppetto/pkg/doc/topics/08-turns.md` (conceptual model: Run/Turn/Block; Turn = one inference cycle)
- Updated the analysis document to include the documented UI pattern as part of the “follow-up turn creation” surface area:
  - `geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/analysis/01-turn-creation-turnid-propagation.md`

### Why
- If docs and code disagree on what Turn.ID means, downstream systems will drift and refactors will
  keep reintroducing ambiguity.

### What worked
- The docs already describe the mental model we want: Turn = one inference cycle and blocks can be attributed via TurnID.

### What didn't work
- The current implementation patterns (session + callers) keep Turn.ID stable across cycles, which weakens the model.

### What I learned
- We should treat this ticket as “make implementation match the documented model”, not as inventing a new one.

### What was tricky to build
- N/A (analysis-only step).

### What warrants a second pair of eyes
- Whether any UI/storage code currently assumes Turn.ID is stable across the whole session and uses it as a “conversation id”.

### What should be done in the future
- After the analysis is complete, update the docs/snippets to explicitly state:
  - Turn.ID is per inference cycle
  - blocks carry the TurnID/InferenceID of the cycle that created them

### Code review instructions
- Start at `geppetto/pkg/doc/playbooks/04-migrate-to-session-api.md` and compare to `pinocchio/pkg/webchat/router.go`.

### Technical details
- N/A (no implementation changes yet).

## Step 4: Identify two Turn.ID generation policies and prefer explicit “seed turn owns the cycle”

I distilled the findings into two viable policies for making `Turn.ID` “fresh per inference” while
keeping it stable throughout one inference cycle (so events and blocks correlate): either the caller
generates a fresh Turn.ID when building the prompt seed turn, or the session generates it internally
on its input copy.

This step is important because it affects how we reason about session history and what a “turn”
means: if Turn.ID is generated at seed creation time, then seed/output for the cycle can share the
same Turn.ID and match the documented “Turn = one inference cycle” model.

**Commit (code):** N/A

### What I did
- Updated the analysis doc to explicitly describe both policies and their tradeoffs:
  - `geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/analysis/01-turn-creation-turnid-propagation.md`
- Preferred the “caller generates fresh Turn.ID when constructing the prompt seed turn” policy as the default recommendation (aligned with documented UI pattern).

### Why
- If we pick the “session generates Turn.ID internally” policy, the seed turn stored in session
  history can end up with an ID that does not match the inference’s TurnID, which complicates
  debugging and attribution.

### What worked
- The documented UI pattern already has a natural “cycle boundary” (seed turn construction), so it’s
  a good place to define the Turn.ID boundary explicitly.

### What didn't work
- N/A (analysis-only step).

### What I learned
- The biggest conceptual unlock is treating Turn snapshots as “full-history snapshots” and using
  `Block.TurnID` + block-level inference metadata to attribute blocks to the cycle that created them.

### What was tricky to build
- Keeping the analysis honest about the current behavior: many call sites implicitly treat Turn.ID
  as stable, so changing it requires coordinated refactors across caller helpers.

### What warrants a second pair of eyes
- Whether any existing store/log/UI logic uses Turn.ID as a stable conversation key (it should be
  SessionID instead).

### What should be done in the future
- If we proceed to implementation, update all “clone latest turn” helper functions to reset Turn.ID
  for each new inference cycle and ensure block attribution is stamped consistently.

### Code review instructions
- Read the “Policy A vs Policy B” section in the analysis doc and confirm it matches desired semantics.

### Technical details
- N/A (no implementation changes yet).

## Step 5: Write the “snapshot turn + per-block attribution” model and stamping pseudocode

I wrote down the target mental model explicitly with a timeline diagram and then sketched concrete
pseudocode for the post-inference “stamping” pass that ensures blocks produced during an inference
cycle have `Block.TurnID == Turn.ID` and carry a new block-level `InferenceID` metadata value.

This is meant to make the eventual implementation mechanical: once we decide the Turn.ID generation
policy, the rest becomes “stamp missing fields deterministically”.

**Commit (code):** N/A

### What I did
- Expanded the analysis doc with:
  - a 2-inference timeline diagram showing block attribution across snapshots
  - post-inference stamping pseudocode (TurnID + proposed block-level inference id key)
  - `geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/analysis/01-turn-creation-turnid-propagation.md`

### Why
- Without a crisp target model, it’s too easy to implement “fresh Turn.ID” while accidentally
  rewriting historical blocks or leaving new blocks unstamped.

### What worked
- The diagram makes it obvious which blocks should and should not change when producing the next
  snapshot.

### What didn't work
- N/A (analysis-only step).

### What I learned
- Post-inference stamping is the simplest choke point for correctness: engines/toolhelpers can remain
  dumb appenders, and the session runner can normalize the result.

### What was tricky to build
- Choosing the stamp condition for the block-level inference id:
  - I used `block.TurnID == out.ID` as the “this block was created in this cycle” predicate, which
    depends on getting `Block.TurnID` correct first.

### What warrants a second pair of eyes
- Whether `block.TurnID == out.ID` is the correct predicate in all cases (especially for legacy
  blocks with missing TurnID that might be backfilled later).

### What should be done in the future
- If we implement this, add tests that simulate:
  - old blocks with pre-existing TurnID/inference id,
  - new blocks appended during inference with empty TurnID,
  - and confirm stamping affects only the new blocks.

### Code review instructions
- Review the pseudocode in the analysis doc and sanity-check the attribution predicate.

### Technical details
- N/A (no implementation changes yet).

## Step 6: Inventory block creation/mutation sites (middlewares + engines + tool loop) and TurnID stamping gaps

I enumerated the production code paths that add or modify blocks in a turn (not tests), focusing on
where `Block.TurnID` is supposed to come from. The recurring theme is that block constructors and
`turns.AppendBlock` do not stamp TurnID, and only a single pre-inference backfill exists in the
session code. That means blocks appended during inference (assistant/tool blocks) often have empty
TurnID today, which makes per-block attribution impossible without a new invariant.

This step also covered the “user input becomes a block” entrypoint: in practice that happens in the
caller layer (Pinocchio/webchat, Bubble Tea backend), where a new user block is appended to a cloned
turn snapshot and then appended to the session before starting inference.

**Commit (code):** N/A

### What I did
- Searched for where blocks are appended or where `t.Blocks` is rewritten:
  - `rg -n "AppendBlock\\(|Blocks\\s*=\\s*append\\(|t\\.Blocks\\s*=" geppetto/pkg -S`
- Read the key middlewares that insert/reorder blocks:
  - `geppetto/pkg/inference/middleware/systemprompt_middleware.go` (inserts or edits a system block)
  - `geppetto/pkg/inference/middleware/reorder_tool_results_middleware.go` (reorders tool_use blocks)
  - `geppetto/pkg/inference/middleware/tool_middleware.go` (appends tool_use blocks via toolblocks)
  - `geppetto/pkg/inference/toolblocks/toolblocks.go` (`AppendToolResultsBlocks`)
- Read provider engine append sites:
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `geppetto/pkg/steps/ai/claude/engine_claude.go`
  - `geppetto/pkg/steps/ai/gemini/engine_gemini.go`
  - `geppetto/pkg/steps/ai/openai_responses/engine.go`
- Updated the analysis doc with a new appendix section describing these paths and their TurnID behavior:
  - `geppetto/ttmp/2026/01/22/GP-05-TURN-CREATION--turn-creation-cloning-ids-and-block-propagation/analysis/01-turn-creation-turnid-propagation.md`

### Why
- We can’t safely enforce “blocks created in an inference cycle carry TurnID + InferenceID” unless we
  know exactly which layers append/insert blocks and which ones bypass helper APIs.

### What worked
- The inventory confirmed that most new blocks are appended via `turns.AppendBlock`, which gives us a
  clean enforcement point if we choose to stamp on append.

### What didn't work
- System prompt middleware inserts blocks by direct slice manipulation (`t.Blocks = append(...)`),
  bypassing `turns.AppendBlock`, so “stamp on append” alone would not cover all insertion cases.

### What I learned
- Current stamping is asymmetric:
  - pre-inference: session backfills missing `Block.TurnID` on the input copy (only-if-empty)
  - post-inference: there is no normalization pass, so new engine/tool blocks can remain unstamped
- Block metadata currently has no inference id attribution; we likely need a new typed key for that.

### What was tricky to build
- Separating “event metadata TurnID” (always set from `t.ID`) from “block attribution TurnID” (often empty).
  It’s easy to assume they match, but they currently don’t.

### What warrants a second pair of eyes
- Whether changing `turns.AppendBlock` to stamp `Block.TurnID` would have any unintended consequences
  for legacy code that appends old blocks with empty TurnID (potential mis-attribution).

### What should be done in the future
- If we implement the contract, enforce it in two places:
  - stamp-on-append for the common case, and
  - a session post-inference normalization pass that stamps remaining blocks and sets a block-level inference id key.

### Code review instructions
- Start with the “Appendix” section added to the analysis doc.
- Cross-check the specific middlewares and toolblocks helpers cited there.

### Technical details
- N/A (analysis-only step).
