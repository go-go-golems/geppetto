# Middleware and Chained Responses: The Context Coherence Problem

## Overview

The OpenAI Responses API's `previous_response_id` chaining feature creates a tension with Geppetto's middleware architecture. When using chained responses, the server expects the conversation to be a strict continuation of the previous response. However, Geppetto's middleware can manipulate the Turn (add/remove/modify blocks) between engine calls, potentially creating a mismatch between what the server thinks happened and what the Turn says happened.

This document analyzes the problem and explores solutions that maintain middleware flexibility while ensuring coherence with server-side state.

---

## The Problem Statement

### Scenario: Tool Middleware + Chained Responses

```
Initial state: Turn with [user: "What's the weather in SF?"]

Engine Call #1:
  → Engine sends Turn to Responses API
  → Server creates response_abc123
  → Server appends: [tool_call(get_weather, SF)]
  → Engine returns Turn with blocks: [user: "...", tool_call(...)]
  → Engine marks blocks as "from response_abc123"
  
Middleware Intercepts:
  → Tool middleware sees tool_call block
  → Executes get_weather locally
  → Appends tool_use block with result: "72°F"
  → Turn now: [user: "...", tool_call(...), tool_use(result: "72°F")]
  
Engine Call #2 (same Turn, next middleware iteration):
  → Engine needs to continue conversation
  → Option A: Use previous_response_id=response_abc123
    - Problem: Server's state is [user, tool_call]
    - We need to send the NEW block: [tool_use]
    - But which previous_response_id to use?
  → Option B: Send full Turn
    - Sends: [user, tool_call, tool_use]
    - But tool_call came FROM the server (response_abc123)
    - Sending it back is redundant/confusing
```

### Core Issue (Corrected Understanding)

**The challenge**: When using `previous_response_id`, we need to:

1. **Identify the "anchor point"**: Which blocks came from which server response?
2. **Send only NEW blocks**: Blocks added AFTER the last server response (by middleware)
3. **Maintain ordering**: Ensure we don't accidentally send blocks from BEFORE the anchor

**Key insight**: The server already knows about blocks it generated (via `previous_response_id`). We should only send blocks that were added CLIENT-SIDE after that response.

Example:
```
Turn state: [user, tool_call(from response_abc), tool_use(from middleware), user(new)]

If using previous_response_id=response_abc:
  - Server knows: [user, tool_call]
  - We should send: [tool_use, user(new)]
  - NOT: [user, tool_call, tool_use, user(new)] (redundant)
  - NOT: [user(new)] (missing tool_use that server needs to see)
```

### Why This Matters

1. **Chained responses assume strict continuation**: Using `previous_response_id` tells the server "continue from exactly where we left off". If we've added tool results client-side, the server doesn't see them.

2. **Middleware is core to Geppetto**: The tool middleware, logging middleware, safety filters, etc. all operate on Turns between engine calls. We can't easily "freeze" the Turn.

3. **Provider semantics leak**: The concept of "previous_response_id must match Turn state" is OpenAI-specific and breaks the provider-agnostic middleware model.

---

## Analysis: What Can We Send with Chained Responses?

Looking at the OpenAI Responses API docs (https://platform.openai.com/docs/guides/conversation-state?api-mode=responses), when using `previous_response_id`:

### Option A: Send nothing (pure chain)
```json
{
  "model": "o4-mini",
  "previous_response_id": "resp_abc123",
  "input": []  // or omit entirely?
}
```
**Assumption**: Server replays conversation from its state. No new input needed unless user adds new message.

### Option B: Send only new user input
```json
{
  "model": "o4-mini",
  "previous_response_id": "resp_abc123",
  "input": [
    {"role": "user", "content": [{"type": "input_text", "text": "Follow up question"}]}
  ]
}
```
**Assumption**: Server appends new input to its conversation state.

### Option C: Send full Turn (breaks chain?)
```json
{
  "model": "o4-mini",
  "previous_response_id": "resp_abc123",
  "input": [
    {"role": "user", "content": [...]},
    {"role": "assistant", "content": [...]},  // What server already knows
    {"role": "tool", "content": [...]},        // What middleware added
    ...
  ]
}
```
**Risk**: Does `previous_response_id` + full `input` array conflict? Does server merge or replace?

### Unknown (needs testing)
- Can we use `previous_response_id` AND send tool results in `input`?
- Does server validate that new input is a strict append?
- What happens if we send input that contradicts server state?

---

## Solution Space

### Solution 0: Track Response Boundaries (Leverage Existing Helpers)

**Concept**: Use existing `SnapshotBlockIDs` and `NewBlocksNotIn` helpers. Engine marks which response_id generated which blocks via Block.Metadata. When building next request, find the latest response_id that doesn't have middleware modifications after it.

```go
// In turns/keys.go (add new metadata key)
const (
    MetaKeyResponseID = "response_id"  // Which server response created this block
)

// Engine side (openai_responses/engine.go)
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    state := GetState(t)
    
    // Take snapshot of blocks BEFORE engine call
    snapshotIDs := middleware.SnapshotBlockIDs(t)
    
    if state.Mode == "chained" {
        // Find best anchor: latest response_id where blocks are unmodified
        anchorResponseID := findBestAnchor(t)
        
        if anchorResponseID != "" {
            // Send only blocks AFTER the anchor
            newBlocks := getBlocksAfterResponse(t, anchorResponseID)
            req := buildChainedRequest(anchorResponseID, newBlocks)
        } else {
            // No valid anchor, fall back to stateless
            req := buildFullRequest(t)
        }
    } else {
        req := buildFullRequest(t)
    }
    
    resp := e.sendRequest(req)
    
    // After engine call: mark NEW blocks with response_id
    newBlocks := middleware.NewBlocksNotIn(t, snapshotIDs)
    for i := range newBlocks {
        newBlocks[i].Metadata[MetaKeyResponseID] = resp.ID
    }
    
    state.PreviousResponseID = resp.ID
    SetState(t, state)
    return t, nil
}

// Helper: Find the best previous_response_id to use
// Returns the LATEST response_id such that:
// 1. All blocks from that response are present and unmodified
// 2. All blocks AFTER that response are new (from middleware)
func findBestAnchor(t *turns.Turn) string {
    // Collect all response IDs in order
    var responseIDs []string
    seenIDs := make(map[string]bool)
    
    for _, b := range t.Blocks {
        if respID, ok := b.Metadata[MetaKeyResponseID].(string); ok && respID != "" {
            if !seenIDs[respID] {
                responseIDs = append(responseIDs, respID)
                seenIDs[respID] = true
            }
        }
    }
    
    if len(responseIDs) == 0 {
        return ""  // No previous responses
    }
    
    // Scan from latest to earliest, find first valid anchor
    for i := len(responseIDs) - 1; i >= 0; i-- {
        candidateID := responseIDs[i]
        
        // Check if blocks from this response are contiguous and unmodified
        if isValidAnchor(t, candidateID) {
            return candidateID
        }
    }
    
    return ""  // No valid anchor found
}

// Helper: Check if response_id is a valid anchor
// Valid if: all blocks with this response_id form a contiguous sequence
// and no middleware has inserted blocks in the middle
func isValidAnchor(t *turns.Turn, responseID string) bool {
    firstIdx := -1
    lastIdx := -1
    
    for i, b := range t.Blocks {
        if respID, ok := b.Metadata[MetaKeyResponseID].(string); ok && respID == responseID {
            if firstIdx == -1 {
                firstIdx = i
            }
            lastIdx = i
        }
    }
    
    if firstIdx == -1 {
        return false  // No blocks from this response
    }
    
    // Check all blocks in range [firstIdx, lastIdx] have same response_id
    for i := firstIdx; i <= lastIdx; i++ {
        respID, ok := t.Blocks[i].Metadata[MetaKeyResponseID].(string)
        if !ok || respID != responseID {
            return false  // Middleware inserted a block in the middle
        }
    }
    
    return true
}

// Helper: Get all blocks added AFTER the anchor response
func getBlocksAfterResponse(t *turns.Turn, responseID string) []turns.Block {
    // Find last block with this response_id
    lastIdx := -1
    for i := len(t.Blocks) - 1; i >= 0; i-- {
        if respID, ok := t.Blocks[i].Metadata[MetaKeyResponseID].(string); ok && respID == responseID {
            lastIdx = i
            break
        }
    }
    
    if lastIdx == -1 || lastIdx == len(t.Blocks)-1 {
        return nil  // No blocks after anchor
    }
    
    // Return all blocks after lastIdx
    return t.Blocks[lastIdx+1:]
}
```

**Pros**:
- Leverages existing `SnapshotBlockIDs` and `NewBlocksNotIn` helpers
- Handles reordering: middleware can rearrange blocks, we still find the anchor
- Handles insertions: middleware adds blocks anywhere, we detect them
- No middleware changes needed: middleware is transparent
- Automatic fallback: if no valid anchor, use stateless mode

**Cons**:
- Requires Block.Metadata (already exists)
- Engine must mark blocks with response_id (one-time implementation)
- Complexity in `findBestAnchor` logic (needs careful testing)

**Edge cases handled**:
1. Middleware reorders blocks → `isValidAnchor` checks contiguity
2. Middleware removes blocks → Next anchor in the past is used
3. Middleware inserts in middle → Anchor before insertion is used
4. No valid anchor → Falls back to stateless

---

### Solution 1: Middleware Transparency (Track Deltas)

**Concept**: Middleware marks which blocks it added. Engine only sends middleware-added blocks (deltas) when using chained responses.

```go
// In turns package
type Block struct {
    Kind     BlockKind
    Payload  map[string]any
    Metadata BlockMetadata  // NEW
}

type BlockMetadata struct {
    Source     string  // "engine" | "middleware" | "user"
    AddedAfter *uuid.UUID  // Response ID after which this was added
}

// Middleware side
func (m *ToolMiddleware) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    prevResponseID := getResponseID(t)  // from Turn.Data
    
    res, err := m.next(ctx, t)
    if err != nil { return nil, err }
    
    // Mark new blocks as middleware-added
    for i := len(t.Blocks); i < len(res.Blocks); i++ {
        res.Blocks[i].Metadata.Source = "middleware"
        res.Blocks[i].Metadata.AddedAfter = &prevResponseID
    }
    return res, nil
}

// Engine side (openai_responses)
func (e *Engine) buildInputForChainedResponse(t *turns.Turn, prevRespID string) []responsesInput {
    var items []responsesInput
    
    // Only include blocks added AFTER the previous response
    for _, b := range t.Blocks {
        if b.Metadata.AddedAfter != nil && *b.Metadata.AddedAfter == prevRespID {
            items = append(items, convertBlock(b))
        }
    }
    
    // If no new blocks, send empty input (pure continuation)
    return items
}
```

**Pros**:
- Maintains middleware flexibility: middleware can still add blocks
- Engine knows what's "new" since last server interaction
- Composable: multiple middlewares can mark their additions

**Cons**:
- Requires BlockMetadata (schema change to Turn)
- Middleware must cooperate (mark blocks correctly)
- Assumes server accepts delta-style input with `previous_response_id`

---

### Solution 2: Conversation Snapshot (Explicit Sync Points)

**Concept**: Store a "server snapshot" of the Turn at each engine call. On next call, compute diff.

```go
// In openai_responses/state.go
type ConversationState struct {
    Mode              string
    PreviousResponseID string
    ServerSnapshot    *turns.Turn  // Turn as server last saw it
}

// Engine side
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    state := GetState(t)
    
    if state.Mode == "chained" && state.PreviousResponseID != "" {
        // Compute delta: blocks in t that aren't in snapshot
        delta := computeDelta(t, state.ServerSnapshot)
        req := buildChainedRequest(state.PreviousResponseID, delta)
    } else {
        // Stateless: send full Turn
        req = buildFullRequest(t)
    }
    
    resp := e.sendRequest(req)
    
    // Update snapshot: current Turn (with middleware additions) becomes new baseline
    state.ServerSnapshot = cloneTurn(t)
    state.PreviousResponseID = resp.ID
    
    return t, nil
}

func computeDelta(current, snapshot *turns.Turn) []turns.Block {
    // Simple: blocks in current that aren't in snapshot
    // Could use block IDs or index-based comparison
    // Handles appends but not removals/edits (those would break chain)
}
```

**Pros**:
- No schema changes to Turn/Block
- Clear synchronization point: snapshot = what server knows
- Engine owns the logic (middleware doesn't need to know)

**Cons**:
- Cloning Turns adds memory overhead
- Doesn't handle block removals or edits (middleware shouldn't do these anyway?)
- State management is complex (snapshot must travel with ConversationState)

---

### Solution 3: Middleware-Aware Engine (Consultation Hook)

**Concept**: Introduce a hook where middleware can "commit" or "stage" blocks, and engine asks middleware what to send.

```go
// New interface in middleware package
type StateAwareMiddleware interface {
    middleware.Middleware
    GetUnsentBlocks(t *turns.Turn) []turns.Block  // What hasn't been sent to server yet
    MarkBlocksSent(t *turns.Turn, responseID string)  // Ack that server saw these
}

// Tool middleware implements this
type ToolMiddleware struct {
    next middleware.HandlerFunc
    pendingBlocks map[uuid.UUID][]turns.Block  // per Turn ID
}

func (m *ToolMiddleware) GetUnsentBlocks(t *turns.Turn) []turns.Block {
    return m.pendingBlocks[t.ID]
}

func (m *ToolMiddleware) MarkBlocksSent(t *turns.Turn, responseID string) {
    delete(m.pendingBlocks, t.ID)
}

// Engine queries the middleware stack
func (e *Engine) buildInputForChainedResponse(ctx context.Context, t *turns.Turn) []responsesInput {
    // If middleware stack implements StateAwareMiddleware, ask for deltas
    if sam, ok := e.middlewareStack.(StateAwareMiddleware); ok {
        unseenBlocks := sam.GetUnsentBlocks(t)
        return convertBlocks(unseenBlocks)
    }
    // Fallback: send full Turn
    return convertAllBlocks(t)
}
```

**Pros**:
- Explicit contract: middleware that cares about state can implement interface
- Backward compatible: middleware that doesn't implement it still works (fallback)
- Engine and middleware collaborate explicitly

**Cons**:
- Tight coupling: engine must know about middleware capabilities
- Requires middleware refactor to implement interface
- Breaks "engine doesn't know about middleware" abstraction

---

### Solution 4: Disable Chaining with Middleware (Pragmatic)

**Concept**: If middleware is in the stack, force "stateless" mode (send full Turn every time).

```go
// In factory or engine initialization
func NewEngineWithMiddleware(base engine.Engine, mws ...Middleware) engine.Engine {
    wrapped := &wrappedEngine{base: base, middlewares: mws}
    
    // If using openai_responses with chained mode, downgrade to stateless
    if respEngine, ok := base.(*openai_responses.Engine); ok {
        respEngine.SetMiddlewarePresent(len(mws) > 0)
    }
    
    return wrapped
}

// Engine side
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    state := GetState(t)
    
    // Force stateless if middleware is present
    if e.hasMiddleware && state.Mode == "chained" {
        log.Warn().Msg("Middleware detected; forcing stateless mode for Responses API")
        state.Mode = "stateless"
    }
    
    // Proceed with stateless (full input)
    ...
}
```

**Pros**:
- Simple: no complex delta tracking
- Safe: always sends full context, no coherence issues
- Clear trade-off: middleware flexibility vs. chaining efficiency

**Cons**:
- Loses chaining benefits (reduced token usage, server-side state)
- Requires engine to know if middleware is present (breaks layering)
- User might expect chaining to work with middleware

---

### Solution 5: Two-Phase Commit (Explicit Synchronization)

**Concept**: Middleware operates in "staged" mode. Engine sends Turn to server, then notifies middleware "server ACKed", then middleware commits changes.

```go
// New middleware hook
type TwoPhaseMiddleware interface {
    PreInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)   // Before engine
    PostInference(ctx context.Context, t *turns.Turn, responseID string) (*turns.Turn, error)  // After engine, server ACKed
}

// Tool middleware
func (m *ToolMiddleware) PreInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    // Extract tool calls, execute tools, but DON'T append results yet
    m.stageTool Results[t.ID] = executeTools(t)
    return t, nil  // Return Turn unchanged
}

func (m *ToolMiddleware) PostInference(ctx context.Context, t *turns.Turn, responseID string) (*turns.Turn, error) {
    // Now append tool results (server has ACKed previous state)
    results := m.stagedToolResults[t.ID]
    for _, r := range results {
        turns.AppendBlock(t, turns.NewToolUseBlock(r.ID, r.Result))
    }
    delete(m.stagedToolResults, t.ID)
    return t, nil
}

// Engine orchestrates
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    // Phase 1: Middleware pre-processes (but doesn't modify Turn)
    if tpm, ok := e.middleware.(TwoPhaseMiddleware); ok {
        t, err = tpm.PreInference(ctx, t)
        if err != nil { return nil, err }
    }
    
    // Phase 2: Engine sends Turn to server (Turn is clean, matches previous response)
    resp := e.sendRequest(buildChainedRequest(t))
    
    // Phase 3: Middleware post-processes (now can append blocks)
    if tpm, ok := e.middleware.(TwoPhaseMiddleware); ok {
        t, err = tpm.PostInference(ctx, t, resp.ID)
    }
    
    return t, nil
}
```

**Pros**:
- Explicit synchronization: server state and Turn state stay in sync
- Middleware still executes (tools run), but timing is controlled
- Engine doesn't need delta tracking

**Cons**:
- Major middleware API change: breaks existing middleware
- Complex: middleware needs to manage staged state
- Doesn't work with middleware that must append blocks before next engine call (e.g., filters)

---

### Solution 6: Conversation Objects (Stateful Server-Side)

**Concept**: Use OpenAI's conversation objects (`POST /conversations`, then `POST /conversations/{id}/responses`) instead of `previous_response_id`. Server maintains full state; client only sends new user input.

```go
// In openai_responses/state.go
type ConversationState struct {
    Mode           string  // "conversation" instead of "chained"
    ConversationID string
    // No need for previous_response_id
}

// Engine side
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    state := GetState(t)
    
    if state.Mode == "conversation" {
        if state.ConversationID == "" {
            // Create conversation on server
            state.ConversationID = e.createConversation(ctx)
        }
        
        // Send ONLY new user input (last user block)
        newInput := extractNewUserInput(t)
        req := buildConversationRequest(state.ConversationID, newInput)
        
        // Tool results? Server handles them internally if tools are enabled.
        // Middleware can still execute tools locally, but server maintains canonical state.
    }
    
    return t, nil
}
```

**Pros**:
- Server is source of truth: tool results tracked server-side
- Client sends minimal input (just new user messages)
- Middleware can still operate on local Turn for logging/filtering

**Cons**:
- Requires conversation object API (not just responses)
- Tool execution might need to happen server-side (limits middleware control)
- Unclear how middleware-added blocks (like injected system messages) work

---

## Comparison Matrix

| Solution | Complexity | Middleware Impact | Engine Complexity | Coherence Guarantee | Performance | Uses Existing Helpers |
|----------|-----------|------------------|-------------------|---------------------|-------------|----------------------|
| 0. Response Boundaries | Medium | None | Medium (anchor logic) | High | High (sends deltas) | ✅ Yes |
| 1. Delta Tracking | Medium | Low (add metadata) | Medium (compute deltas) | High | High (sends deltas) | Partial |
| 2. Snapshot | Medium | None | High (clone Turns) | High | Medium (memory overhead) | ✅ Yes |
| 3. Aware Hook | High | High (new interface) | Medium | High | High | No |
| 4. Disable Chain | Low | None | Low | Perfect (no chain) | Low (full Turn) | N/A |
| 5. Two-Phase | High | High (new API) | High (orchestrate) | Perfect | High | No |
| 6. Conversation Objects | Medium | Low (mostly transparent) | Medium | Delegate to server | Highest | N/A |

---

## Recommendations

### Short-term (Ship MVP) - UPDATED
**Solution 0: Track Response Boundaries** ⭐ **NEW RECOMMENDATION**

Reasoning:
- **Leverages existing infrastructure**: Uses `SnapshotBlockIDs` and `NewBlocksNotIn` helpers already in middleware package
- **Correct semantics**: Properly identifies which blocks came from server vs. middleware
- **Middleware transparent**: No changes needed to existing middleware
- **Automatic fallback**: If anchor detection fails, falls back to stateless mode
- **Handles edge cases**: Reordering, insertions, removals all handled correctly
- **Performance**: Sends only delta blocks when possible

Implementation sketch:
```go
// 1. Add MetaKeyResponseID to turns/keys.go
// 2. In engine.RunInference():
//    a. Snapshot block IDs before call
//    b. Find best anchor via findBestAnchor(t)
//    c. If anchor found, send only blocks after it
//    d. After response, mark new blocks with response_id
// 3. Implement anchor detection helpers (isValidAnchor, getBlocksAfterResponse)
```

**Alternative if Solution 0 is too complex initially**:
**Solution 4: Disable Chaining with Middleware**

Reasoning:
- Simplest to implement: no schema changes, no complex logic
- Safe: guarantees coherence by always sending full Turn
- Pragmatic trade-off: users who want chaining can avoid middleware, users who want middleware get stateless mode
- Clear documentation: "Chained responses are incompatible with middleware"
- Can be replaced with Solution 0 later without breaking changes

### Mid-term (Optimize)
**Solution 6: Conversation Objects**

Reasoning:
- Cleanest separation: server manages state, client just sends user input
- Middleware can still do local work (logging, filtering) without affecting server state
- Scales to advanced use cases (multi-turn, persistent conversations)
- Requires understanding conversation object API semantics (needs research/testing)
- **Note**: May still benefit from Solution 0's anchor detection if sending deltas

### Long-term (Full Control)
**Enhance Solution 0** OR **Add Solution 5: Two-Phase Commit**

Reasoning:
- Solution 0 provides the foundation; enhancements might include:
  - More sophisticated anchor detection (handling out-of-order blocks)
  - Validation warnings when anchor cannot be found
  - Metrics/observability for anchor selection
- Solution 5 (two-phase) only if we need strict ordering guarantees
- Avoid Solution 1 (duplicate of Solution 0 but more invasive)
- Avoid Solution 3 (breaks layering)

---

## Open Questions

1. **Conversation Objects API**: How do conversation objects handle tools? Do tool results need to be sent explicitly, or does the server track them?

2. **Response Chaining Validation**: Does OpenAI validate that `input` with `previous_response_id` is a strict append? Or can we send arbitrary new blocks?

3. **Middleware Contract**: Should we formally specify "middleware must not remove/edit blocks, only append"? This would simplify delta tracking.

4. **Turn Immutability**: Should Turns become immutable (middleware returns new Turn instead of mutating)? Would simplify snapshot/delta logic but breaks existing code.

5. **Testing**: How to test conversation coherence? Need mock server that tracks state and rejects invalid chains.

6. **Fallback**: If chained request fails (coherence error), should we automatically retry with stateless? Or fail fast and let caller handle?

7. **Observability**: Should we emit events when mode switches (chained → stateless due to middleware)? Helps debug unexpected behavior.

---

## Implications for Architecture

### Middleware Design Principles (Updated)

From `09-middlewares.md`, we learned:
- Middlewares should be stateless when possible
- Use per-Turn data hints, not global state

**New principle for Responses API**:
- **Middlewares that add blocks should mark them** (if using chained responses with Solution 1)
- **OR: Accept that middleware + chaining is incompatible** (Solution 4)
- **OR: Use conversation objects** (Solution 6) and accept server is source of truth

### Turn-Based Architecture (Implications)

The Turn is meant to be a self-contained unit representing conversation state. Chained responses challenge this:
- Server maintains "true" state indexed by response_id
- Turn is client's view, potentially diverged via middleware

**Design question**: Should Turn be the canonical source of truth, or just a working copy?

**Answer** (from this analysis):
- **Stateless mode**: Turn is canonical (sent in full)
- **Chained mode without middleware**: Turn and server state are in sync
- **Chained mode with middleware**: Need synchronization mechanism (Solutions 1, 2, 5, or 6)

---

## Next Steps (Post-Design)

1. **Test conversation objects API**: Create a conversation, send responses, see how tool results are handled
2. **Test response chaining with deltas**: Try `previous_response_id` + new blocks in `input`, see if server accepts/rejects
3. **Implement Solution 4** (disable chaining with middleware) as MVP
4. **Document trade-offs** in user-facing docs (inference engines tutorial)
5. **Gather feedback**: Do users prefer middleware flexibility or chaining efficiency?
6. **Revisit** with Solution 6 (conversation objects) if server-side state is well-supported

---

## Summary: The Key Insight

**The problem is NOT** that middleware modifies the Turn and breaks coherence.

**The problem IS** that we need to:
1. **Identify which blocks came from which server response** (via response_id in metadata)
2. **Find the best "anchor point"** - the latest server response where blocks are unmodified and contiguous
3. **Send only the blocks AFTER that anchor** - these are the new blocks from middleware

**Why this matters**:
- Server already knows about blocks it generated (via `previous_response_id`)
- Sending them again is redundant and potentially confusing
- But we MUST send middleware-added blocks (like tool results) that the server hasn't seen

**The solution** (Solution 0): Mark blocks with their originating response_id, then intelligently find the best anchor and send only new blocks. This is compatible with existing middleware helpers (`SnapshotBlockIDs`, `NewBlocksNotIn`) and requires no middleware changes.

---

## Related Documents

- `04-conversation-state-management-design-proposals.md`: Six approaches to storing/passing conversation state
- `09-middlewares.md`: Core middleware architecture and principles
- `06-inference-engines.md`: Engine interface and Turn-based design
- `geppetto/pkg/inference/middleware/helpers.go`: Existing snapshot and delta helpers
- OpenAI Conversation State Guide: https://platform.openai.com/docs/guides/conversation-state?api-mode=responses

---

## Appendix: Potential API Extensions

If we pursue Solution 1 (Delta Tracking), we might add:

```go
// In turns package
type TurnDiff struct {
    Added   []Block
    Removed []BlockID  // if we allow removals
    Modified []BlockUpdate
}

func ComputeDiff(previous, current *Turn) TurnDiff { /* ... */ }

// In middleware
func (m *MyMiddleware) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    snapshot := cloneTurn(t)
    res, err := m.next(ctx, t)
    if err != nil { return nil, err }
    
    diff := turns.ComputeDiff(snapshot, res)
    res.Data["middleware_diff"] = diff  // Engine can read this
    return res, nil
}
```

If we pursue Solution 5 (Two-Phase), we might add a middleware orchestrator:

```go
// In middleware package
type Orchestrator struct {
    middlewares []TwoPhaseMiddleware
}

func (o *Orchestrator) RunWithPhases(ctx context.Context, t *turns.Turn, engineFn func(*turns.Turn) (*turns.Turn, string, error)) (*turns.Turn, error) {
    // Pre-phase
    for _, m := range o.middlewares {
        t, _ = m.PreInference(ctx, t)
    }
    
    // Engine
    t, responseID, err := engineFn(t)
    
    // Post-phase
    for _, m := range o.middlewares {
        t, _ = m.PostInference(ctx, t, responseID)
    }
    
    return t, nil
}
```

