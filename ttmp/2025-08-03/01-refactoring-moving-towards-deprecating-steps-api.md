# Analysis: Moving from Step-based API to RunInference API in Pinocchio

**Date:** 2025-08-03  
**Author:** AI Assistant  
**Status:** Analysis Complete - Ready for Oracle Architecture Proposal  

## Executive Summary

Pinocchio currently uses the complex Step-based API with watermill events for LLM inference throughout its codebase. This analysis examines the current usage patterns and identifies the areas that need refactoring to move toward the simpler RunInference API while maintaining event-driven functionality through watermill.

## Current Step-based API Usage Patterns

### 1. Core Entry Points

#### Main Command Execution (`pinocchio/pkg/cmds/cmd.go`)
- **Primary usage:** Lines 278, 285, 373, 407 - Creates steps via `rc.StepFactory.NewStep()`
- **Step execution:** Lines 341, 429 - Calls `steps.Bind(ctx, messagesM, chatStep)` for step composition
- **Watermill integration:** Lines 285, 373, 407 - Uses `chat.WithPublishedTopic(rc.Router.Publisher, "chat"/"ui")`
- **Event routing:** Lines 292-301, 413-423 - Sets up event handlers for different output modes

```go
// Current pattern:
chatStep, err := rc.StepFactory.NewStep(chat.WithPublishedTopic(rc.Router.Publisher, "chat"))
messagesM := steps.Resolve(conversation_)
m := steps.Bind(ctx, messagesM, chatStep)
for r := range m.GetChannel() {
    // Process results
}
```

#### Chat Runner (`pinocchio/pkg/chatrunner/chat_runner.go`)
- **Step factory pattern:** Lines 72, 164, 305-315 - Uses factory function for step creation
- **Multiple execution modes:** Lines 48-58 - Supports blocking, interactive, and chat modes
- **Watermill routing:** Lines 72 - Creates steps with publisher/topic configuration

```go
// Current pattern:
uiStep, err := cs.stepFactory(router.Publisher, "ui")
step, err := cs.stepFactory(nil, "") // For blocking mode
```

#### UI Backend (`pinocchio/pkg/ui/backend.go`)
- **Direct Step.Start() calls:** Line 32 - `stepResult, err := s.step.Start(ctx, msgs)`
- **Channel-based result handling:** Lines 48-54 - Processes step results through channels
- **Event forwarding:** Lines 87-152 - Transforms watermill messages to bubbletea messages

### 2. Event Architecture Integration

#### Event Router (`geppetto/pkg/events/event-router.go`)
- **Publisher/Subscriber setup:** Lines 64-86 - Creates watermill pub/sub system
- **Step integration:** Lines 127 - `step.AddPublishedTopic(e.Publisher, topic)`
- **Event dispatching:** Lines 151-196 - Dispatches events to ChatEventHandler

#### Event Types (from analysis)
- **EventPartialCompletion** - Streaming response chunks
- **EventText** - Text content events
- **EventFinal** - Completion events
- **EventError** - Error events
- **EventInterrupt** - Interruption events

### 3. Key Usage Locations

#### Web UI Integration
- **Client:** `pinocchio/cmd/experiments/web-ui/client/client.go:155` - `result, err := c.step.Start(ctx, conv)`
- **Server:** `pinocchio/cmd/experiments/web-ui/server.go:275` - Server-side step management

#### Agent Experiments
- **Codegen:** Multiple files using `step.Start(ctx, manager.GetConversation())`
- **Tool usage:** `pinocchio/cmd/experiments/agent/tool/tool.go:136`
- **Uppercase:** `pinocchio/cmd/experiments/agent/uppercase.go:65`

## Current Watermill Event Flow

### 1. Step Creation & Publishing Setup
```
StepFactory.NewStep(chat.WithPublishedTopic(publisher, topic))
↓
Step configured with watermill publisher and topic
↓
Step.Start() begins inference and publishes events
```

### 2. Event Publishing During Inference
```
LLM API Response → ContentBlockMerger → EventPartialCompletion → Watermill Publisher
                                      → EventFinal → Watermill Publisher
                                      → EventError → Watermill Publisher
```

### 3. Event Consumption
```
Watermill Subscriber → Event Router → Handler Functions → UI Updates
```

## Challenges with Current Architecture

### 1. **Complex Step Framework Overhead**
- Generic types and channel-based result handling
- Complex lifecycle management (Start → Channel → Results)
- Difficult to use for simple inference cases

### 2. **Tight Coupling Between Steps and Events**
- Steps must be configured with publishers at creation time
- Event publishing logic embedded in step execution
- Difficult to test inference without full event infrastructure

### 3. **Inconsistent API Surface**
- Different creation patterns for different modes (blocking vs streaming)
- Multiple factory functions and configuration options
- Hard to understand for new developers

### 4. **Resource Management Complexity**
- Channel lifecycle management
- Context cancellation handling
- Router startup/shutdown coordination

## Areas Requiring Refactoring

### 1. **High-Priority Refactoring Targets**

#### Core Command Execution (`pinocchio/pkg/cmds/cmd.go`)
- **Lines 276-334:** `runBlocking()` method - Direct step creation and execution
- **Lines 336-353:** `runStepAndCollectMessages()` - Step binding and result processing
- **Lines 355-489:** `runChat()` method - Complex UI integration with steps

#### UI Backend (`pinocchio/pkg/ui/backend.go`)
- **Lines 27-56:** `Start()` method - Direct step.Start() call
- **Lines 87-152:** Event forwarding logic - Tightly coupled to step events

#### Chat Runner (`pinocchio/pkg/chatrunner/chat_runner.go`)
- **Lines 156-207:** `runBlockingInternal()` - Step creation and execution
- **Lines 61-153:** `runChatInternal()` - UI mode step handling

### 2. **Medium-Priority Refactoring Targets**

#### Experiment Code
- Web UI client/server integration
- Agent codegen examples
- Tool usage patterns

### 3. **Event Infrastructure** (Preserve & Enhance)
- Event router functionality (keep as-is)
- Event type definitions (expand for RunInference)
- Watermill pub/sub system (adapt for new API)

## Requirements for New Architecture

### 1. **Simplified Inference API**
- Direct `RunInference(ctx, messages) (*Message, error)` calls
- No complex channel management
- Optional event publishing

### 2. **Preserved Event-Driven Capabilities**
- Streaming events for UI updates
- Progress tracking for long-running operations
- Error handling and interruption support

### 3. **Backwards Compatibility** (Initial Phase)
- Existing Step API should continue working
- Gradual migration path
- No breaking changes to public APIs

### 4. **Flexible Event Integration**
- Inference engines can optionally publish events
- Event publishing decoupled from core inference logic
- Easy to test inference without events

## Key Files for Oracle Analysis

### Core Implementation Files
- `pinocchio/pkg/cmds/cmd.go` - Main command execution logic
- `pinocchio/pkg/chatrunner/chat_runner.go` - Chat session management
- `pinocchio/pkg/ui/backend.go` - UI integration layer

### Event Infrastructure
- `geppetto/pkg/events/event-router.go` - Watermill event routing
- `geppetto/pkg/events/chat-events.go` - Event type definitions
- `geppetto/pkg/events/publish.go` - Event publishing logic

### Step Implementation References
- `geppetto/pkg/steps/ai/openai/chat-step.go` - OpenAI step with RunInference
- `geppetto/pkg/steps/ai/claude/chat-step.go` - Claude step with RunInference
- `geppetto/pkg/steps/ai/chat/simple.go` - SimpleChatStep interface

### Related Documentation
- `geppetto/ttmp/2025-06-19/01-how-we-refactored-and-introduced-run-inference-to-steps.md` - Previous refactoring
- `geppetto/ttmp/2025-06-16/02-watermill-streaming-architecture.md` - Watermill architecture
- `pinocchio/pkg/doc/topics/01-chat-runner-events.md` - ChatRunner events

## Conclusion

The current Step-based API creates significant complexity for both simple and advanced use cases in pinocchio. The analysis shows widespread usage across 8+ key files with deep integration into the watermill event system. 

The refactoring should focus on:
1. **Simplifying the core inference API** while preserving event capabilities
2. **Maintaining watermill integration** for streaming and UI updates  
3. **Providing a gradual migration path** from the current Step API
4. **Preserving all existing functionality** while reducing complexity

The next step is to use the Oracle to design an architecture that addresses these requirements while maintaining the robust event-driven capabilities that pinocchio relies on for its UI and streaming features.
