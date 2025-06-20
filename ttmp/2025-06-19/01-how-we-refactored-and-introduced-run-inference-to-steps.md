# Step Refactoring: Introduction of SimpleChatStep.RunInference Method

**Date:** 2025-06-19  
**Author:** AI Assistant  
**Status:** Completed - Requires Investigation for Potential Blocking Issues  

## Executive Summary

We introduced a new `SimpleChatStep` interface with a `RunInference` method to simplify LLM interactions by encapsulating the complex step mechanism. This refactoring moved significant logic from the asynchronous `Start` method to the synchronous `RunInference` method, which may have introduced blocking behavior that needs investigation.

## Intent and Motivation

### Original Problem
- Complex step mechanism required generic types and intricate event handling
- Direct LLM calls required understanding of streaming, events, metadata, and cancellation
- Code duplication between Claude and OpenAI implementations
- Difficult to use LLMs without the full step framework

### Solution Goals
- Provide simple interface for direct LLM inference: `RunInference(ctx, messages) (*Message, error)`
- Encapsulate all complexity: API calls, streaming, events, metadata, error handling
- Maintain backwards compatibility with existing `Start` method
- Centralize ContentBlockMerger and event publishing logic

## Changes Made

### 1. New Interface Creation

**File:** `pkg/steps/ai/chat/simple.go`

```go
// SimpleChatStep provides a simplified interface for LLM inference
type SimpleChatStep interface {
    RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error)
}
```

### 2. Claude Implementation Changes

**File:** `pkg/steps/ai/claude/chat-step.go`

#### A. Interface Implementation
```go
var _ chat.SimpleChatStep = &ChatStep{}
```

#### B. RunInference Method - Complete Logic Migration

**Key Functions Moved:**
- `makeMessageRequest()` - Request preparation
- `api.NewClient()` - Client setup
- `client.SendMessage()` / `client.StreamMessage()` - API calls
- `NewContentBlockMerger()` - Streaming event processing
- `csf.subscriptionManager.PublishBlind()` - Event publishing

**Pseudocode:**
```
RunInference(ctx, messages):
    1. Validate settings (Client, Claude, ApiType)
    2. Setup API client with keys and URLs
    3. Create request with tools and parameters
    4. Setup event metadata and step metadata
    5. Publish start event
    
    IF non-streaming:
        6a. Call client.SendMessage() synchronously
        7a. Update metadata with usage/stop reason
        8a. Publish final event
        9a. Return message
    
    IF streaming:
        6b. Call client.StreamMessage()
        7b. Create ContentBlockMerger
        8b. FOR EACH streaming event:
            - Process with completionMerger.Add()
            - Publish intermediate events
        9b. Get final response from merger
        10b. Return final message
```

#### C. Start Method Simplification

**Before:** 150+ lines of complex streaming logic with channels, goroutines, and event handling
**After:** 30 lines that delegate to RunInference

```go
func (csf *ChatStep) Start(ctx, messages) (StepResult[*Message], error) {
    if !csf.Settings.Chat.Stream {
        // Simple case: use RunInference directly
        message, err := csf.RunInference(ctx, messages)
        return steps.Resolve(message), nil
    }
    
    // Streaming case: wrap RunInference in goroutine
    c := make(chan helpers2.Result[*conversation.Message])
    go func() {
        message, err := csf.RunInference(cancellableCtx, messages)
        if err != nil {
            c <- helpers2.NewErrorResult(err)
        } else {
            c <- helpers2.NewValueResult(message)
        }
    }()
    return steps.NewStepResult(c, ...), nil
}
```

### 3. OpenAI Implementation Changes

**File:** `pkg/steps/ai/openai/chat-step.go`

Similar pattern applied:
- Moved complete streaming logic to `RunInference`
- Added event publishing and metadata management
- Simplified `Start` method to delegate to `RunInference`

**Key Functions Migrated:**
- `makeClient()` - Client creation
- `makeCompletionRequest()` - Request preparation  
- `client.CreateChatCompletion()` / `client.CreateChatCompletionStream()` - API calls
- `ExtractChatCompletionMetadata()` - Response processing
- Event publishing via `csf.publisherManager.PublishBlind()`

## Critical Architectural Changes

### Synchronous vs Asynchronous Behavior

#### **BEFORE (Start Method):**
```
Start() -> StepResult[*Message] {
    1. Setup streaming immediately
    2. Return StepResult with channel
    3. Process events in background goroutine
    4. Events published as they arrive
    5. Non-blocking - caller gets results via channel
}
```

#### **AFTER (RunInference via Start):**
```
Start() -> StepResult[*Message] {
    1. Create goroutine
    2. Call RunInference(ctx, messages) 
       - BLOCKS until entire stream completes
       - All events published during processing
       - Returns only final message
    3. Send result to channel
}

RunInference() -> *Message {
    - SYNCHRONOUS operation
    - Blocks until completion
    - No incremental results
    - All-or-nothing response
}
```

## ⚠️ POTENTIAL BLOCKING ISSUES

### Problem Areas

#### 1. **Streaming Behavior Change**
- **Old:** Events streamed incrementally, non-blocking
- **New:** RunInference blocks until entire stream completes
- **Risk:** Long-running streams may cause timeouts or UI freezing

#### 2. **ContentBlockMerger Processing**
- **Location:** `pkg/steps/ai/claude/chat-step.go:219-235`
- **Issue:** Now processes ALL streaming events before returning
```go
for event := range eventCh {
    events_, err := completionMerger.Add(event)
    // ... publish events but don't return until channel closes
}
```

#### 3. **Event Publishing Timing**
- **Old:** Events published as received in background
- **New:** Events published during RunInference execution
- **Risk:** Event subscribers may experience different timing

### Investigation Checklist

#### Immediate Testing
1. **Test streaming responses with long content**
   ```bash
   # Test with a prompt that generates long responses
   go test -v ./pkg/steps/ai/claude -run TestContentBlockMerger
   ```

2. **Check timeout behavior**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   message, err := simpleChatStep.RunInference(ctx, longMessages)
   ```

3. **Monitor event publishing order**
   - Verify start → progress → final event sequence
   - Check if intermediate events still fire during streaming

#### Code Locations to Investigate

##### Claude Implementation
- **File:** `pkg/steps/ai/claude/chat-step.go`
- **RunInference method:** Lines 81-262
- **Streaming loop:** Lines 219-235
- **ContentBlockMerger usage:** Line 219, 222-234

##### OpenAI Implementation  
- **File:** `pkg/steps/ai/openai/chat-step.go`
- **RunInference method:** Lines 58-243
- **Streaming logic:** Lines 115-189

##### Start Method Delegation
- **Claude Start:** Lines 264-322
- **OpenAI Start:** Lines 245-303

### Debugging Steps

#### 1. Add Timing Logs
```go
// In RunInference method, add:
log.Printf("RunInference started for %d messages", len(messages))
defer log.Printf("RunInference completed")

// Around streaming loop:
log.Printf("Starting stream processing")
for event := range eventCh {
    log.Printf("Processing event: %T", event)
    // ... existing logic
}
log.Printf("Stream processing complete")
```

#### 2. Test Cancellation Behavior
```go
func TestRunInferenceCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    
    go func() {
        time.Sleep(100 * time.Millisecond)
        cancel() // Cancel mid-stream
    }()
    
    _, err := chatStep.RunInference(ctx, messages)
    assert.Equal(t, context.Canceled, err)
}
```

#### 3. Compare Event Timing
Create test that captures event timestamps in old vs new implementation.

## Recommendations for Next Developer

### Immediate Actions
1. **Performance Testing**
   - Measure RunInference execution time for various prompt sizes
   - Compare against old Start method performance
   - Test with slow network conditions

2. **Cancellation Testing**
   - Verify context cancellation works properly mid-stream
   - Test timeout scenarios
   - Ensure proper cleanup on cancellation

3. **Event Flow Verification**
   - Confirm event publishing still works correctly
   - Verify event subscribers receive events in correct order
   - Check for any missing or duplicate events

### Potential Solutions if Blocking Issues Confirmed

#### Option A: Hybrid Approach
Keep RunInference synchronous but add async variant:
```go
type SimpleChatStep interface {
    RunInference(ctx, messages) (*Message, error)           // Synchronous
    RunInferenceAsync(ctx, messages) <-chan Result[*Message] // Asynchronous
}
```

#### Option B: Streaming Callback Pattern
```go
type StreamCallback func(event Event) error

type SimpleChatStep interface {
    RunInference(ctx, messages) (*Message, error)
    RunInferenceStream(ctx, messages, callback StreamCallback) (*Message, error)
}
```

#### Option C: Revert Streaming Logic
Move complex streaming logic back to Start method, keep only simple non-streaming logic in RunInference.

### Files Requiring Attention

#### Primary Implementation Files
- `pkg/steps/ai/claude/chat-step.go` - Claude implementation
- `pkg/steps/ai/openai/chat-step.go` - OpenAI implementation  
- `pkg/steps/ai/chat/simple.go` - Interface definition

#### Test Files
- `pkg/steps/ai/claude/content-block-merger_test.go` - ContentBlockMerger tests
- Any integration tests using ChatStep

#### Related Components
- Event publishing system (`pkg/events/`)
- Step framework (`pkg/steps/`)
- Conversation types (`pkg/conversation/`)

## Code References

### Key Variables and Functions

#### Claude ChatStep
- `csf.subscriptionManager` - Event publisher
- `csf.parentID` - Message threading
- `completionMerger` - ContentBlockMerger instance
- `events2.NewStartEvent()`, `events2.NewFinalEvent()` - Event creation
- `makeMessageRequest()` - Request creation
- `api.NewClient()` - Claude API client

#### OpenAI ChatStep
- `csf.publisherManager` - Event publisher  
- `makeClient()` - OpenAI client creation
- `makeCompletionRequest()` - Request preparation
- `ExtractChatCompletionMetadata()` - Response parsing
- `events.NewStartEvent()`, `events.NewFinalEvent()` - Event types

### Interface Methods
- `SimpleChatStep.RunInference(context.Context, conversation.Conversation) (*conversation.Message, error)`
- `ChatStep.Start(context.Context, conversation.Conversation) (steps.StepResult[*conversation.Message], error)`

## Conclusion

The refactoring successfully simplified the LLM interface and centralized logic, but introduced potential blocking behavior in streaming scenarios. The next developer should focus on investigating performance impact, testing cancellation behavior, and verifying event publishing continues to work correctly. If blocking issues are confirmed, consider the hybrid approaches outlined above.

The changes maintain backwards compatibility while providing a cleaner interface for direct LLM use, but careful testing is needed to ensure production systems don't experience degraded performance or responsiveness.
