# Implementing Response Boundary Tracking (Solution 0): Detailed Guide & Plan

## Overview

This document provides a comprehensive implementation guide for **Solution 0: Track Response Boundaries** from `05-middleware-and-chained-responses-problem.md`. This solution enables chained responses to work correctly with middleware by tracking which blocks came from which server response and intelligently selecting anchor points.

**Goal**: Enable `previous_response_id` chaining while maintaining full middleware compatibility, sending only the blocks that the server hasn't seen yet.

**Status**: Design complete, ready for implementation

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Component Breakdown](#component-breakdown)
3. [Implementation Phases](#implementation-phases)
4. [Detailed Component Specifications](#detailed-component-specifications)
5. [Testing Strategy](#testing-strategy)
6. [Edge Cases and Failure Modes](#edge-cases-and-failure-modes)
7. [Migration and Rollout](#migration-and-rollout)
8. [Performance Considerations](#performance-considerations)
9. [Observability and Debugging](#observability-and-debugging)
10. [Future Enhancements](#future-enhancements)

---

## Architecture Overview

### High-Level Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Before Engine Call                                       │
│    - Snapshot current block IDs (middleware.SnapshotBlockIDs)│
│    - If chained mode: Find best anchor (findBestAnchor)     │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. Build Request                                            │
│    - If anchor found: Include only blocks AFTER anchor      │
│    - If no anchor: Include full Turn (stateless fallback)   │
│    - Set previous_response_id to anchor (if chained)        │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. Send Request to OpenAI Responses API                     │
│    - Server continues from previous_response_id             │
│    - Server appends new blocks to its conversation state    │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. After Response                                           │
│    - Identify new blocks (middleware.NewBlocksNotIn)        │
│    - Mark each new block with response_id in Metadata       │
│    - Update ConversationState.PreviousResponseID            │
└─────────────────────────────────────────────────────────────┘
```

### Key Data Structures

```
Block.Metadata["response_id"] = "resp_abc123"
  ↳ Tracks which server response created this block

Turn.Data["openai_responses_state"] = ConversationState{
    Mode: "chained",
    PreviousResponseID: "resp_xyz789",
}
  ↳ Tracks overall conversation state

Anchor Detection:
  input: Turn with mixed blocks (some from server, some from middleware)
  output: response_id of latest valid anchor point
  
Delta Extraction:
  input: Turn + anchor response_id
  output: []Block containing only blocks AFTER anchor
```

---

## Component Breakdown

### Phase 1: Foundation (Metadata Keys & State Management)

**Components**:
1. Add `MetaKeyResponseID` constant to `turns/keys.go`
2. Ensure `ConversationState` structure exists (from doc 04)
3. Ensure `GetState/SetState` accessors exist (from doc 04)

**Dependencies**: None (can start immediately)

**Deliverable**: Constants and state management ready

---

### Phase 2: Snapshot & Tagging (Engine Integration)

**Components**:
1. Integrate `middleware.SnapshotBlockIDs` into engine's `RunInference`
2. Integrate `middleware.NewBlocksNotIn` to identify new blocks after response
3. Tag new blocks with `response_id` in their Metadata

**Dependencies**: Phase 1

**Deliverable**: Engine can snapshot and tag blocks with response IDs

---

### Phase 3: Anchor Detection (Core Logic)

**Components**:
1. Implement `findBestAnchor(t *Turn) string`
2. Implement `isValidAnchor(t *Turn, responseID string) bool`
3. Implement `getBlocksAfterResponse(t *Turn, responseID string) []Block`

**Dependencies**: Phase 2

**Deliverable**: Engine can identify valid anchor points

---

### Phase 4: Request Building (Delta Mode)

**Components**:
1. Modify `buildResponsesRequest` to accept anchor + delta blocks
2. Implement `buildInputFromBlocks(blocks []Block)` helper
3. Add fallback logic when no anchor exists

**Dependencies**: Phase 3

**Deliverable**: Engine can build chained requests with deltas

---

### Phase 5: Testing & Validation

**Components**:
1. Unit tests for anchor detection
2. Integration tests with mock middleware
3. End-to-end tests with real Responses API

**Dependencies**: Phase 4

**Deliverable**: Fully tested implementation

---

## Detailed Component Specifications

### 1. Metadata Key Definition

**File**: `geppetto/pkg/turns/keys.go`

**What to add**:
```go
const (
    // Existing keys...
    MetaKeyProvider   = "provider"
    MetaKeyRuntime    = "runtime"
    // ... etc ...
    
    // NEW: Response tracking
    MetaKeyResponseID = "response_id"  // Stores the OpenAI response_id that created this block
)
```

**Documentation**:
- Add comment explaining that `MetaKeyResponseID` is used by engines to track block provenance
- Note that middleware should NOT set this key (engine-owned)
- Clarify that absence of this key means block was user-created or from non-tracking engine

---

### 2. Snapshot Integration in RunInference

**File**: `geppetto/pkg/steps/ai/openai_responses/engine.go`

**Method**: `RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)`

**Where to add snapshot**:
- **Location**: At the very beginning of `RunInference`, before any processing
- **Purpose**: Capture the set of block IDs that exist BEFORE engine modifies the Turn

**Pseudocode structure**:
```go
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    // NEW: Take snapshot BEFORE any work
    snapshotIDs := middleware.SnapshotBlockIDs(t)
    
    // Existing: Get conversation state
    state := GetState(t)
    
    // Existing: Build request (will be modified in Phase 4)
    reqBody := buildRequest(...)
    
    // Existing: Send request, handle response
    ...
    
    // NEW: After response, identify and tag new blocks
    newBlocks := middleware.NewBlocksNotIn(t, snapshotIDs)
    for i := range newBlocks {
        // Find the actual block in t.Blocks and update its metadata
        // (need to match by ID since newBlocks is a copy)
    }
    
    return t, nil
}
```

**Important considerations**:
- Snapshot must happen BEFORE request building (captures pre-call state)
- Tagging must happen AFTER response (identifies what engine added)
- Must handle case where block IDs might change during processing

---

### 3. Anchor Detection Algorithm

**File**: `geppetto/pkg/steps/ai/openai_responses/helpers.go` (or new `anchors.go`)

#### Function: `findBestAnchor`

**Signature**:
```go
func findBestAnchor(t *turns.Turn) string
```

**Purpose**: Find the latest response_id that represents a valid anchor point

**Algorithm**:
1. Scan all blocks and collect unique response_ids in order of appearance
2. Iterate from most recent to oldest response_id
3. For each candidate, check if it's a valid anchor via `isValidAnchor`
4. Return first valid anchor found, or empty string if none

**Return values**:
- Non-empty string: Valid anchor found, use this as `previous_response_id`
- Empty string: No valid anchor, fall back to stateless mode

**Edge cases to handle**:
- Empty Turn (no blocks): Return ""
- All blocks from middleware (no response_ids): Return ""
- Multiple responses with same ID: Should work (validates contiguity)

---

#### Function: `isValidAnchor`

**Signature**:
```go
func isValidAnchor(t *turns.Turn, responseID string) bool
```

**Purpose**: Determine if a response_id can be used as an anchor

**Validation rules**:
1. **Existence**: At least one block has this response_id
2. **Contiguity**: All blocks with this response_id form a contiguous sequence
3. **No insertions**: No blocks with different (or missing) response_id in the middle

**Algorithm**:
1. Find first and last index of blocks with `responseID`
2. Check all blocks between first and last index
3. If any block in that range has a different response_id (or none), return false
4. Otherwise return true

**Why contiguity matters**:
- If middleware inserted a block in the middle, the sequence is broken
- We can't use this as an anchor because the server's view doesn't match
- Example: `[user, resp_A, MIDDLEWARE_BLOCK, resp_A]` - resp_A is not valid

---

#### Function: `getBlocksAfterResponse`

**Signature**:
```go
func getBlocksAfterResponse(t *turns.Turn, responseID string) []turns.Block
```

**Purpose**: Extract all blocks that come AFTER the anchor point

**Algorithm**:
1. Find the last block with `responseID` in the Turn
2. Return all blocks after that index
3. If no blocks after, return empty slice (not nil)

**Return value semantics**:
- Empty slice: No new blocks to send (pure continuation)
- Non-empty slice: These are the delta blocks to include in request

---

### 4. Request Building Modifications

**File**: `geppetto/pkg/steps/ai/openai_responses/helpers.go`

#### Modify: `buildResponsesRequest`

**Current behavior**: Calls `buildInputItemsFromTurn(t)` to get full Turn

**New behavior**:
- Accept additional parameters: `anchorResponseID string`, `useChained bool`
- If `useChained` and anchor is non-empty:
  - Call `getBlocksAfterResponse(t, anchorResponseID)`
  - Build input from ONLY those blocks
  - Set `previous_response_id` field in request
- Otherwise:
  - Use full Turn (existing behavior)
  - Omit `previous_response_id` field

**Signature changes**:
```go
// Before:
func buildResponsesRequest(s *settings.StepSettings, t *turns.Turn) (responsesRequest, error)

// After:
func buildResponsesRequest(s *settings.StepSettings, t *turns.Turn, anchorResponseID string) (responsesRequest, error)
```

**Request structure changes**:
```go
type responsesRequest struct {
    Model              string             `json:"model"`
    Input              []responsesInput   `json:"input"`
    PreviousResponseID string             `json:"previous_response_id,omitempty"`  // NEW
    // ... other fields ...
}
```

---

#### New Helper: `buildInputFromBlocks`

**Signature**:
```go
func buildInputFromBlocks(blocks []turns.Block) []responsesInput
```

**Purpose**: Convert a subset of blocks to Responses API input format

**Why separate from `buildInputItemsFromTurn`**:
- `buildInputItemsFromTurn` operates on full Turn
- This operates on a slice of blocks (the delta)
- Allows reuse of conversion logic

**Implementation note**: May internally call the same logic as `buildInputItemsFromTurn`, just with different input

---

### 5. Fallback Logic

**Where**: In `engine.go`'s `RunInference` method

**Fallback conditions**:
1. Anchor detection returns empty string
2. Anchor detection fails with error
3. `getBlocksAfterResponse` returns empty slice (nothing to send)

**Fallback behavior**:
- Log warning: "No valid anchor found, using stateless mode"
- Build request with full Turn
- Omit `previous_response_id` field
- Continue normally

**Why this is safe**:
- Stateless mode always works (just less efficient)
- No data loss or coherence issues
- Allows system to degrade gracefully

---

## Implementation Phases

### Phase 1: Foundation (1 day)

**Tasks**:
1. Add `MetaKeyResponseID` to `turns/keys.go`
2. Review/ensure `ConversationState` exists with fields:
   - `Mode string`
   - `PreviousResponseID string`
3. Review/ensure `GetState/SetState` accessors exist
4. Add godoc comments explaining the anchor tracking concept

**Verification**:
- Constants are accessible
- State can be get/set on a Turn
- No compilation errors

---

### Phase 2: Snapshot & Tagging (2 days)

**Tasks**:
1. Import `middleware` package in `engine.go`
2. Add snapshot at beginning of `RunInference`
3. Add tagging logic at end of `RunInference` (after response received)
4. Handle edge case: block IDs might be empty (generate if needed)
5. Add debug logging: "Snapshot captured X blocks", "Tagged Y new blocks with response_id Z"

**Verification**:
- Run existing tests, ensure no regressions
- Add unit test: mock engine call, verify blocks get tagged
- Inspect logs: confirm snapshot and tagging happen

---

### Phase 3: Anchor Detection (3 days)

**Tasks**:
1. Implement `findBestAnchor` with order-preserving response_id collection
2. Implement `isValidAnchor` with contiguity checking
3. Implement `getBlocksAfterResponse` with boundary extraction
4. Add extensive logging in each function (trace level):
   - "Found response_ids: [...]"
   - "Checking anchor: resp_X"
   - "Anchor resp_X is valid/invalid because..."
   - "Extracted N blocks after anchor"

**Verification**:
- Unit tests for each function (see Testing Strategy below)
- Test with synthetic Turns (manually constructed blocks with response_ids)
- Verify logging output makes sense

---

### Phase 4: Request Building (2 days)

**Tasks**:
1. Add `PreviousResponseID` field to `responsesRequest` struct
2. Modify `buildResponsesRequest` signature to accept `anchorResponseID`
3. Add conditional logic: if anchor, use delta; else use full Turn
4. Implement `buildInputFromBlocks` helper
5. Add logging: "Using chained mode with anchor resp_X, sending Y blocks"

**Verification**:
- Unit tests for request building (with and without anchor)
- Verify `previous_response_id` field is set correctly
- Verify `input` array contains only delta blocks when anchored

---

### Phase 5: Integration (2 days)

**Tasks**:
1. Wire anchor detection into `RunInference`
2. Pass anchor to `buildResponsesRequest`
3. Handle fallback when anchor is empty
4. Update error messages and logging
5. Test full flow: snapshot → detect → build → send → tag

**Verification**:
- Integration test with mock HTTP server
- Verify full flow works end-to-end
- Test fallback scenarios

---

### Phase 6: Testing & Validation (3 days)

**Tasks**:
- See "Testing Strategy" section below

---

### Phase 7: Documentation (1 day)

**Tasks**:
1. Update `06-inference-engines.md` with anchor tracking explanation
2. Update `05-middleware-and-chained-responses-problem.md` status
3. Add godoc to all new functions
4. Create example code snippet showing chained mode usage

---

**Total estimated time**: ~14 days (can be parallelized in some areas)

---

## Testing Strategy

### Unit Tests

#### Test: `findBestAnchor`

**Scenarios**:

1. **Empty Turn**
   - Input: Turn with 0 blocks
   - Expected: "" (empty string)

2. **Single Response, No Middleware**
   - Input: Turn with [user, llm(resp_A), tool_call(resp_A)]
   - Expected: "resp_A"

3. **Single Response, Middleware Appended**
   - Input: Turn with [user, llm(resp_A), tool_call(resp_A), tool_use(no resp_id)]
   - Expected: "resp_A" (anchor is valid, middleware appended)

4. **Multiple Responses**
   - Input: Turn with [user, llm(resp_A), tool_use(no resp_id), llm(resp_B)]
   - Expected: "resp_B" (most recent)

5. **Middleware Inserted in Middle**
   - Input: Turn with [user, llm(resp_A), INJECTED(no resp_id), tool_call(resp_A)]
   - Expected: "" (resp_A not contiguous, no valid anchor)

6. **All Middleware Blocks**
   - Input: Turn with [user, system(no resp_id), tool_use(no resp_id)]
   - Expected: "" (no response_ids at all)

---

#### Test: `isValidAnchor`

**Scenarios**:

1. **Valid Contiguous Sequence**
   - Input: Turn with [user, llm(resp_A), tool_call(resp_A)], check "resp_A"
   - Expected: true

2. **Non-contiguous (Gap)**
   - Input: Turn with [llm(resp_A), OTHER, llm(resp_A)], check "resp_A"
   - Expected: false

3. **Non-existent Response ID**
   - Input: Turn with [llm(resp_A)], check "resp_B"
   - Expected: false

4. **Single Block**
   - Input: Turn with [llm(resp_A)], check "resp_A"
   - Expected: true

---

#### Test: `getBlocksAfterResponse`

**Scenarios**:

1. **Blocks After Anchor**
   - Input: Turn with [user, llm(resp_A), tool_use, user2], anchor "resp_A"
   - Expected: [tool_use, user2]

2. **No Blocks After**
   - Input: Turn with [user, llm(resp_A)], anchor "resp_A"
   - Expected: [] (empty slice)

3. **Anchor Not Found**
   - Input: Turn with [user, llm(resp_A)], anchor "resp_B"
   - Expected: [] (or could return all blocks as fallback)

4. **Multiple Blocks with Same Anchor**
   - Input: Turn with [llm(resp_A), tool_call(resp_A), tool_use], anchor "resp_A"
   - Expected: [tool_use] (only after LAST occurrence)

---

### Integration Tests

#### Test: End-to-End Chained Request

**Setup**:
1. Mock HTTP server that tracks requests
2. Initial Turn with [user]
3. Engine configured for chained mode

**Flow**:
1. First call: Engine sends full Turn, server returns response_A with [tool_call]
2. Middleware appends [tool_use]
3. Second call: Engine sends only [tool_use] with previous_response_id=response_A
4. Server returns response_B with [llm_text]

**Assertions**:
- First request has full input, no previous_response_id
- Second request has only tool_use, previous_response_id=response_A
- Blocks are correctly tagged with response_ids
- Final Turn has all blocks in correct order

---

#### Test: Fallback to Stateless

**Setup**:
- Turn where middleware has rearranged blocks (no valid anchor)

**Flow**:
1. Engine attempts chained mode
2. `findBestAnchor` returns ""
3. Engine falls back to stateless mode
4. Request contains full Turn, no previous_response_id

**Assertions**:
- Log message indicates fallback
- Request structure is correct for stateless mode
- No errors or panics

---

#### Test: Multiple Middleware Iterations

**Setup**:
- Tool middleware that loops multiple times

**Flow**:
1. Call 1: User message → response_A (tool_call)
2. Middleware appends tool_use
3. Call 2: Only tool_use sent → response_B (tool_call2)
4. Middleware appends tool_use2
5. Call 3: Only tool_use2 sent → response_C (llm_text)

**Assertions**:
- Each call uses correct anchor
- Delta blocks are correct at each step
- Final Turn has all blocks with correct response_ids

---

### Real API Tests

#### Test: Simple Chained Request

**Setup**: Real OpenAI API key, real Responses API

**Flow**:
1. Send initial message
2. Capture response_id
3. Send follow-up with previous_response_id
4. Verify server accepts it

**Purpose**: Confirm API behavior matches expectations

---

#### Test: Tool Calling with Chaining

**Setup**: Real API, tool registry with simple tool

**Flow**:
1. User message triggers tool call
2. Engine chains with previous_response_id
3. Tool result sent as delta
4. Verify server continues conversation correctly

**Purpose**: Validate the entire chained tool calling flow

---

## Edge Cases and Failure Modes

### Edge Case 1: Empty Blocks After Anchor

**Scenario**: Anchor found, but `getBlocksAfterResponse` returns empty slice

**Behavior**: 
- Should send request with empty input array + previous_response_id
- OR fall back to stateless if API doesn't support empty input

**Test**: Verify API accepts empty input with previous_response_id

---

### Edge Case 2: Middleware Removes Blocks

**Scenario**: Middleware removes blocks that had response_ids

**Example**: Turn has [user, llm(resp_A), tool_call(resp_A)], middleware removes tool_call

**Behavior**:
- `findBestAnchor` won't find resp_A (contiguity broken)
- Falls back to stateless mode
- Sends full Turn (with removal)

**Why this is safe**: Server doesn't know about removed block anyway (it's local client-side manipulation)

---

### Edge Case 3: Block IDs Collision

**Scenario**: Multiple blocks have same ID (shouldn't happen, but could)

**Behavior**:
- `SnapshotBlockIDs` and `NewBlocksNotIn` handle this (use map, so dupes are ignored)
- Tagging might tag wrong block if IDs collide

**Mitigation**: Ensure engine generates unique IDs (use UUID)

---

### Edge Case 4: Response ID Not Returned by Server

**Scenario**: API call succeeds but response doesn't include response_id

**Behavior**:
- Engine can't tag blocks with response_id
- Next call won't find anchor (blocks have no metadata)
- Falls back to stateless automatically

**Mitigation**: Log warning if response_id missing

---

### Edge Case 5: Concurrent Requests

**Scenario**: Same Turn used in multiple goroutines simultaneously

**Behavior**:
- Turn modification is not thread-safe
- Could result in race conditions on Block.Metadata

**Mitigation**: 
- Document that Turns should not be shared across goroutines
- OR: Make Block.Metadata writes atomic (use sync.Mutex)
- Recommendation: Document and avoid, don't add complexity

---

### Failure Mode 1: Anchor Detection Bug

**Symptom**: Engine picks wrong anchor or fails to find valid one

**Impact**: Falls back to stateless mode (safe, but less efficient)

**Detection**: Log warnings, metrics on fallback rate

**Recovery**: Automatic (fallback handles it)

---

### Failure Mode 2: API Rejects Chained Request

**Symptom**: Server returns 400 error for request with previous_response_id

**Impact**: Request fails, error propagated to caller

**Detection**: Error response from API

**Recovery**: Retry with stateless mode? Or fail fast?

**Recommendation**: Fail fast, let caller handle retry

---

### Failure Mode 3: Blocks Not Tagged

**Symptom**: Bug in tagging logic, blocks don't get response_id

**Impact**: Future calls can't find anchor, always use stateless

**Detection**: All requests become stateless after first call

**Recovery**: Fix bug and redeploy

**Mitigation**: Comprehensive unit tests for tagging logic

---

## Migration and Rollout

### Stage 1: Feature Flag (Disabled by Default)

**Implementation**:
- Add feature flag or environment variable: `ENABLE_RESPONSE_BOUNDARY_TRACKING`
- Default: `false` (existing behavior)
- When enabled: Use anchor detection

**Purpose**: Deploy code without changing behavior, test in staging

---

### Stage 2: Opt-In (Per-Request)

**Implementation**:
- Allow ConversationState.Mode to be "chained" (with anchor tracking) or "stateless"
- Default: "stateless"
- Users explicitly set "chained" to opt in

**Purpose**: Let adventurous users test in production

---

### Stage 3: Default for New Conversations

**Implementation**:
- New conversations default to "chained" mode
- Existing conversations remain "stateless" (until explicitly changed)

**Purpose**: Gradual rollout to new traffic

---

### Stage 4: Full Rollout

**Implementation**:
- All conversations use "chained" mode by default
- "stateless" still available as fallback/override

**Purpose**: Maximize efficiency for all users

---

### Rollback Plan

**If issues arise**:
1. Set feature flag to `false` (disables anchor tracking)
2. OR: Change default mode back to "stateless"
3. Monitor metrics, investigate root cause
4. Fix and re-enable

**Rollback time**: ~5 minutes (config change)

---

## Performance Considerations

### Memory Overhead

**Snapshot Storage**:
- `SnapshotBlockIDs` creates a map of block IDs
- Size: O(num_blocks) * ~32 bytes (string ID)
- Typical: 10-100 blocks → <10KB
- **Impact**: Negligible

**Block Metadata**:
- Each block gets one additional metadata entry (response_id)
- Size: ~50 bytes per block
- **Impact**: Negligible

---

### CPU Overhead

**Anchor Detection**:
- `findBestAnchor`: O(num_blocks * num_response_ids)
- `isValidAnchor`: O(num_blocks)
- Typical: 10-100 blocks, 1-5 response_ids → ~500 operations
- **Impact**: Microseconds, negligible

**Snapshot & Tagging**:
- `SnapshotBlockIDs`: O(num_blocks)
- `NewBlocksNotIn`: O(num_new_blocks)
- **Impact**: Negligible

---

### Network Efficiency

**With Chaining**:
- Request size reduced by ~50-90% (only delta blocks)
- Example: 10-block Turn, send 2 new blocks instead of 10

**Token Usage**:
- Reduced input tokens (server already has history)
- Cost savings: Proportional to conversation length

**Benefit**: Increases with conversation length

---

## Observability and Debugging

### Logging Levels

**TRACE**:
- Every anchor detection step
- Block tagging details
- Input/output of helper functions

**DEBUG**:
- Anchor selection result
- Number of blocks in delta
- Fallback decisions

**INFO**:
- Mode selection (chained vs stateless)
- Request sent/received

**WARN**:
- Fallback triggered (anchor not found)
- Missing response_id in API response

**ERROR**:
- API rejection of chained request
- Unexpected failures

---

### Metrics to Track

**Counters**:
- `responses_chained_requests_total`: # of requests using chained mode
- `responses_stateless_requests_total`: # of requests using stateless mode
- `responses_fallback_total`: # of times fallback triggered

**Histograms**:
- `responses_delta_blocks_count`: Distribution of delta block counts
- `responses_anchor_detection_duration_ms`: Time spent in anchor detection

**Gauges**:
- `responses_chained_mode_enabled`: 1 if feature enabled, 0 otherwise

---

### Debugging Tools

**Log Analysis Queries**:
```
# Find conversations with frequent fallbacks
grep "using stateless mode" | count by turn_id

# Track anchor selection
grep "Using chained mode with anchor" | extract response_id

# Identify tagging failures
grep "Tagged .* blocks" | filter count == 0
```

**Turn Inspection Tool** (future):
- Visualize Turn blocks with response_id annotations
- Highlight anchor points
- Show which blocks would be sent in delta

---

## Future Enhancements

### Enhancement 1: Anchor Caching

**Idea**: Cache the last valid anchor to avoid re-scanning Turn on every call

**Implementation**:
- Store `lastValidAnchor` in ConversationState
- Only re-scan if Turn changed since last call

**Benefit**: Small CPU reduction

---

### Enhancement 2: Multi-Anchor Support

**Idea**: Support multiple anchors if Turn has disjoint response sequences

**Use case**: Middleware rearranges blocks but doesn't insert in middle

**Complexity**: High, unclear benefit

---

### Enhancement 3: Anchor Validation with Server

**Idea**: Periodically verify server's state matches our anchor

**Implementation**:
- Send request with `previous_response_id` + `validate: true`
- Server returns checksum or block count
- Compare with local state

**Benefit**: Catch coherence bugs early

---

### Enhancement 4: Compressed History

**Idea**: When falling back to stateless, summarize old blocks

**Implementation**:
- Detect when Turn is very long (>50 blocks)
- Compress old blocks into a single "summary" block
- Send summary + recent blocks

**Benefit**: Reduce token usage for long conversations

---

### Enhancement 5: Metrics Dashboard

**Idea**: Real-time dashboard showing anchor tracking stats

**Metrics**:
- Chained vs stateless ratio
- Average delta size
- Fallback rate
- API rejection rate

**Benefit**: Operational visibility

---

## Open Questions for Implementation

1. **Block ID Generation**: 
   - Should engine ensure blocks have IDs before tagging?
   - What if block ID is empty?
   - **Recommendation**: Generate UUID if ID is empty

2. **Metadata Mutation**:
   - Should we mutate Block.Metadata directly or create new Block?
   - **Recommendation**: Mutate directly (Block is already a struct, not pointer in slice)

3. **Response ID Format**:
   - Is response_id always in specific format?
   - Should we validate format?
   - **Recommendation**: Don't validate, treat as opaque string

4. **Anchor Age**:
   - Should we limit how old an anchor can be?
   - Example: Don't use anchors >10 responses old
   - **Recommendation**: No limit initially, add if problems arise

5. **Empty Input Array**:
   - Does API accept `input: []` with `previous_response_id`?
   - **Recommendation**: Test with API, add fallback if unsupported

6. **Tool Results in Delta**:
   - Are tool results always included in delta?
   - What if middleware doesn't append tool_use?
   - **Recommendation**: Send whatever blocks exist after anchor, server decides

7. **Concurrent State Updates**:
   - What if multiple goroutines update same ConversationState?
   - **Recommendation**: Document as caller's responsibility, don't add locks

---

## Success Criteria

### Definition of Done

- [ ] All functions implemented and unit tested
- [ ] Integration tests pass (mock server)
- [ ] Real API tests pass (at least 10 successful chained requests)
- [ ] Documentation updated
- [ ] No performance regression in existing tests
- [ ] Logging at appropriate levels
- [ ] Metrics instrumented
- [ ] Code reviewed and approved

### Acceptance Criteria

1. **Functional**:
   - Chained mode works with tool middleware
   - Fallback to stateless happens automatically
   - No coherence errors from API

2. **Performance**:
   - Request size reduced by ≥50% for multi-turn conversations
   - Anchor detection takes <1ms on typical Turn

3. **Reliability**:
   - Test suite passes consistently
   - No new errors/panics introduced

4. **Usability**:
   - Users can opt into chained mode easily
   - Clear error messages when things fail
   - Debugging logs are helpful

---

## Related Documents

- `04-conversation-state-management-design-proposals.md`: Overall state management design
- `05-middleware-and-chained-responses-problem.md`: Problem analysis and Solution 0 design
- `geppetto/pkg/inference/middleware/helpers.go`: Existing snapshot/delta helpers
- `geppetto/pkg/turns/types.go`: Block and Turn data structures
- `geppetto/pkg/doc/topics/09-middlewares.md`: Middleware architecture principles

---

## Appendix: Code Pointers for Implementation

### Files to Modify

1. **`geppetto/pkg/turns/keys.go`**
   - Add `MetaKeyResponseID` constant

2. **`geppetto/pkg/steps/ai/openai_responses/engine.go`**
   - Modify `RunInference` to add snapshot/tagging
   - Wire in anchor detection

3. **`geppetto/pkg/steps/ai/openai_responses/helpers.go`**
   - Implement anchor detection functions
   - Modify `buildResponsesRequest` to accept anchor

4. **`geppetto/pkg/steps/ai/openai_responses/state.go`** (if not exists, create)
   - Ensure `ConversationState` structure
   - Ensure `GetState/SetState` accessors

### Files to Create

1. **`geppetto/pkg/steps/ai/openai_responses/anchors.go`** (optional)
   - House anchor detection logic separately for clarity

2. **`geppetto/pkg/steps/ai/openai_responses/engine_test.go`** (if not exists)
   - Unit tests for anchor detection

3. **`geppetto/pkg/steps/ai/openai_responses/integration_test.go`**
   - Integration tests with mock server

### Existing Code to Leverage

- `geppetto/pkg/inference/middleware/helpers.go`:
  - `SnapshotBlockIDs(t *turns.Turn) map[string]struct{}`
  - `NewBlocksNotIn(t *turns.Turn, baseline map[string]struct{}) []turns.Block`

- `geppetto/pkg/turns/helpers_blocks.go`:
  - `WithBlockMetadata(b Block, kvs map[string]interface{}) Block`

---

## Implementation Checklist

Use this as a step-by-step guide during implementation:

### Phase 1: Foundation
- [ ] Add `MetaKeyResponseID` to `keys.go`
- [ ] Verify `ConversationState` exists with `PreviousResponseID` field
- [ ] Verify `GetState/SetState` work
- [ ] Run `go build ./...` - no errors

### Phase 2: Snapshot & Tagging
- [ ] Import `middleware` package in `engine.go`
- [ ] Add `snapshotIDs := middleware.SnapshotBlockIDs(t)` at start of `RunInference`
- [ ] After response, add `newBlocks := middleware.NewBlocksNotIn(t, snapshotIDs)`
- [ ] Tag each block in `newBlocks` with response_id from API response
- [ ] Add debug logs for snapshot and tagging
- [ ] Write unit test: verify blocks get tagged
- [ ] Run tests - all pass

### Phase 3: Anchor Detection
- [ ] Implement `findBestAnchor(t *turns.Turn) string`
- [ ] Implement `isValidAnchor(t *turns.Turn, respID string) bool`
- [ ] Implement `getBlocksAfterResponse(t *turns.Turn, respID string) []Block`
- [ ] Add trace-level logging in each function
- [ ] Write unit tests for each function (see Testing Strategy)
- [ ] Run tests - all pass

### Phase 4: Request Building
- [ ] Add `PreviousResponseID` field to `responsesRequest` struct
- [ ] Modify `buildResponsesRequest` signature
- [ ] Add conditional: if anchor, use delta; else full Turn
- [ ] Implement `buildInputFromBlocks` helper
- [ ] Add debug log for mode selection
- [ ] Write unit tests for request building
- [ ] Run tests - all pass

### Phase 5: Integration
- [ ] Call `findBestAnchor` in `RunInference`
- [ ] Pass anchor to `buildResponsesRequest`
- [ ] Handle fallback when anchor is ""
- [ ] Add fallback warning log
- [ ] Write integration test with mock server
- [ ] Run tests - all pass

### Phase 6: Real API Testing
- [ ] Test with real API: simple chained request
- [ ] Test with real API: tool calling with chaining
- [ ] Verify no errors from server
- [ ] Verify delta blocks are correct
- [ ] Document any API quirks discovered

### Phase 7: Documentation
- [ ] Update `06-inference-engines.md`
- [ ] Update this document's status
- [ ] Add godoc to all functions
- [ ] Create usage example
- [ ] Update changelog

### Final
- [ ] Code review
- [ ] All tests pass
- [ ] Performance benchmarks (if applicable)
- [ ] Merge to main

---

**End of Implementation Guide**

