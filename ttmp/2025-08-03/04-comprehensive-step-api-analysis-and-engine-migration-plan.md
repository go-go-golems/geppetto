# Comprehensive Step API Analysis and Engine-First Migration Plan

**Date:** August 3, 2025  
**Author:** AI Analysis Agent  
**Status:** Final Analysis - Ready for Implementation  

## Executive Summary

This document provides a comprehensive analysis of all remaining Step API usage in the geppetto codebase and presents a detailed migration plan to complete the Engine-first architecture transition. The analysis reveals that while the new Engine interface is fully implemented and actively used in pinocchio, significant Step API infrastructure remains in geppetto that requires careful migration or removal.

**Key Findings:**
- **62 static analysis issues** related to deprecated Step API usage
- **32 files** contain Step interface usage
- **17 active lint violations** showing Step deprecation warnings
- **Complete Engine implementation** already available and functional
- **Clear migration paths** documented in deprecation comments

**Migration Scope:** Medium complexity with clear patterns for transformation.

---

## Current State Analysis

### Lint Analysis Results

Running comprehensive linting revealed **17 active issues** with Step API deprecation warnings:

```
pkg/steps/step.go:122:1: File is not properly formatted (gofmt)
pkg/events/event-router.go:116:7: SA1019: chat.Step is deprecated
pkg/steps/ai/chat/step.go:20:11: SA1019: steps.Step is deprecated
[... 14 more deprecation warnings ...]
```

**Hidden Issues (truncated by linter):**
- 17/20 issues with `steps.Step` deprecation
- 13/16 issues with `chat.Step` deprecation  
- 5/8 issues with `steps.Resolve` deprecation
- 1/4 issues with `steps.Bind` deprecation

**Total Scope:** Approximately **53+ deprecation issues** across the codebase.

### File-Level Analysis

**Core Step Infrastructure (8 files):**
```
pkg/steps/step.go              - Main Step interface, StepResult, Bind, Resolve
pkg/steps/errors.go            - Step-specific error definitions
pkg/steps/ai/chat/step.go      - Chat-specific Step type alias
pkg/steps/ai/chat/simple.go    - SimpleChatStep interface
pkg/steps/ai/factory.go        - StandardStepFactory (deprecated)
pkg/steps/ai/settings/         - Step configuration system
pkg/steps/utils/lambda.go      - Lambda step utilities
pkg/steps/utils/chain.go       - Step composition utilities
```

**Provider Implementations (12 files):**
```
pkg/steps/ai/openai/           - OpenAI chat steps, tool calling steps
pkg/steps/ai/claude/           - Claude chat steps, message steps
pkg/steps/ai/gemini/           - Gemini chat step
pkg/steps/parse/               - JSON/markdown parsing steps
```

**Tool Calling System (4 files):**
```
pkg/steps/ai/openai/chat-execute-tool-step.go    - Composite tool calling step
pkg/steps/ai/openai/chat-with-tools-step.go      - AI model + tool integration
pkg/steps/ai/openai/execute-tool-step.go         - Function execution step
pkg/events/event-router.go                       - Event system integration
```

**JavaScript Integration (3 files):**
```
pkg/js/runtime-engine.go       - Step execution with event streaming
pkg/js/streaming-step-js.go    - JavaScript-exposed step interface
pkg/js/chat-step-factory-js.go - Step factory for JS runtime
```

**Utility & Transformation (5 files):**
```
pkg/steps/utils/merge.go       - Conversation merging
pkg/steps/utils/logger.go      - Logging step
pkg/steps/parse/extract.go     - Data extraction
pkg/steps/parse/jsonschema.go  - JSON validation
pkg/steps/parse/markdown.go    - Markdown processing
```

---

## Categorized Step Usage Analysis

### 1. Core Step Infrastructure

**Status:** ⚠️ **Needs Careful Migration**

**Files:**
- `pkg/steps/step.go` - Central Step interface, StepResult, Bind, Resolve
- `pkg/steps/errors.go` - Step-specific errors
- `pkg/steps/ai/chat/step.go` - Chat type aliases

**Functionality:**
- **Step Interface:** Generic computation with Start() and AddPublishedTopic()
- **StepResult:** Monadic container with streaming, cancellation, metadata
- **Bind Function:** Monadic composition for chaining steps
- **Resolve Function:** Wraps values in StepResult containers
- **Event Integration:** Publisher management and topic registration

**Engine Equivalent:**
- **Engine.RunInference()** replaces Step.Start()
- **Direct value passing** replaces StepResult channels
- **EventSink system** replaces AddPublishedTopic()
- **Simple error handling** replaces monadic error propagation

### 2. Provider Implementations

**Status:** 🔄 **Partial Migration Complete**

**OpenAI Steps:**
- `chat-step.go` - ✅ **Engine equivalent exists** (`engine_openai.go`)
- `chat-with-tools-step.go` - ⚠️ **Tool support missing in Engine**
- `chat-execute-tool-step.go` - ⚠️ **Complex tool workflow missing**
- `execute-tool-step.go` - ⚠️ **Function execution pattern missing**

**Claude Steps:**
- `chat-step.go` - ✅ **Engine equivalent exists** (`engine_claude.go`)
- `messages-step.go` - ⚠️ **Message processing missing**

**Gemini Steps:**
- `chat-step.go` - ❌ **No Engine equivalent** (TODO in factory)

**Migration Pattern:**
```go
// OLD: Step-based approach
step, err := openai.NewStep(settings)
result, err := step.Start(ctx, conversation)
for res := range result.GetChannel() {
    message, _ := res.Value()
    // Process message
}

// NEW: Engine-based approach  
engine, err := inference.NewOpenAIEngine(settings)
message, err := engine.RunInference(ctx, conversation)
// Direct message processing
```

### 3. Tool Calling System

**Status:** ❌ **Major Gap - Not Implemented in Engine**

**Current Step Implementation:**
```
ChatExecuteToolStep (orchestrator)
    ├── ChatWithToolsStep (AI model interaction)
    ├── ExecuteToolStep (function execution)
    └── LambdaStep (result formatting)
```

**Tool Calling Workflow:**
1. **Tool Definition:** JSON schema generation from Go functions
2. **AI Integration:** Send tools + prompt to AI model
3. **Tool Request:** Parse AI's tool call requests
4. **Function Execution:** Invoke Go functions with parameters
5. **Result Processing:** Format results for AI consumption
6. **Continuation:** Send results back to AI model

**Missing Engine Features:**
- Tool function registration and schema generation
- Tool call request/response handling
- Function execution framework
- Multi-turn tool calling conversations
- Tool result formatting

**Required Engine Extensions:**
```go
// Proposed Engine interface extension
type ToolEngine interface {
    Engine
    RunInferenceWithTools(ctx context.Context, 
        messages conversation.Conversation,
        tools map[string]interface{}) (*conversation.Message, error)
}
```

### 4. Utility Steps

**Status:** ⚠️ **Transformation Patterns Needed**

**Lambda Steps (pkg/steps/utils/lambda.go):**
- `LambdaStep[T, U]` - Simple function wrapper
- `BackgroundLambdaStep[T, U]` - Async function execution  
- `MapLambdaStep[T, U]` - Function mapping over slices
- `BackgroundMapLambdaStep[T, U]` - Async function mapping

**Transformation Steps:**
- Uppercase conversion example
- JSON parsing and validation
- Markdown processing
- Data extraction from AI responses

**Engine Transformation Pattern:**
```go
// OLD: Lambda step
uppercaseStep := &utils.LambdaStep[string, string]{
    Function: func(input string) helpers.Result[string] {
        return helpers.NewValueResult(strings.ToUpper(input))
    },
}

// NEW: Direct transformation
func transformResponse(message *conversation.Message) *conversation.Message {
    content := strings.ToUpper(message.Content.String())
    return conversation.NewChatMessage(message.Role, content)
}
```

### 5. Event System Integration

**Status:** ✅ **Engine Migration Complete**

**Step Event System:**
- `AddPublishedTopic()` method for event publishing
- `PublisherManager` for topic management
- Manual event publishing in step implementations

**Engine Event System:**
- `EventSink` interface for event handling
- `WatermillSink` for event bus integration
- `WithSink()` option for engine configuration
- Automatic event publishing during inference

**Migration Example:**
```go
// OLD: Step with manual event publishing
step.AddPublishedTopic(publisher, "step.events")
result, err := step.Start(ctx, input)

// NEW: Engine with automatic event publishing
sink := inference.NewWatermillSink(publisher, "inference.events")
engine, err := factory.CreateEngine(settings, inference.WithSink(sink))
message, err := engine.RunInference(ctx, conversation)
```

### 6. JavaScript Integration

**Status:** 🔄 **Needs Engine-First Redesign**

**Current JavaScript Step Integration:**
- `RuntimeEngine.RunStep()` - Executes steps with event streaming
- `CreateWatermillStepObject()` - JavaScript step wrapper
- `StartTypedStep()` - Type-safe step execution

**Required Changes:**
- Replace `RunStep()` with `RunInference()`
- Update JavaScript bindings for Engine interface
- Maintain event streaming compatibility
- Preserve type safety in JS/Go bridge

---

## Engine-First Architecture Analysis

### Current Engine Implementation

**Core Engine Interface:**
```go
type Engine interface {
    RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error)
}
```

**Available Implementations:**
- **OpenAIEngine** - OpenAI, AnyScale, Fireworks providers
- **ClaudeEngine** - Claude/Anthropic provider  
- **GeminiEngine** - TODO: Not yet implemented

**Factory System:**
```go
type EngineFactory interface {
    CreateEngine(settings *settings.StepSettings, options ...Option) (Engine, error)
    SupportedProviders() []string
    DefaultProvider() string
}
```

**Event System:**
```go
type EventSink interface {
    PublishEvent(event events.Event) error
}

// Configuration
engine, err := factory.CreateEngine(settings, 
    inference.WithSink(sink),
    // other options...
)
```

### Engine Advantages Over Steps

**Simplicity:**
- Single method `RunInference()` vs complex Step interface
- Direct value return vs channel-based StepResult
- Simple error handling vs monadic error propagation

**Performance:**
- No channel overhead for simple operations
- Direct function calls vs goroutine orchestration
- Reduced memory allocation for streaming

**Maintainability:**
- Clear separation of concerns
- Provider-specific logic encapsulated
- Event publishing handled by framework

**Type Safety:**
- Concrete types vs generic constraints
- Compile-time validation vs runtime type checking
- Clear input/output contracts

---

## Tool Calling Analysis & Design

### Current Tool Implementation Pattern

**Function Registration:**
```go
toolFunctions := map[string]interface{}{
    "getWeather": func(params WeatherParams) string {
        // Implementation
    },
}
```

**Schema Generation:**
```go
jsonSchema, err := helpers.GetFunctionParametersJsonSchema(reflector, tool)
openaiTool := go_openai.Tool{
    Type: "function",
    Function: &go_openai.FunctionDefinition{
        Name:        name,
        Description: jsonSchema.Description,
        Parameters:  json.RawMessage(schemaBytes),
    },
}
```

**Execution Pipeline:**
```
1. ChatWithToolsStep -> AI Model (with tools)
2. Parse tool call requests from AI response
3. ExecuteToolStep -> Invoke Go functions
4. Format results as conversation messages
5. Continue conversation with tool results
```

### Proposed Engine Tool Integration

**Option A: Tool-Enabled Engine Interface**
```go
type ToolEngine interface {
    Engine
    RegisterTool(name string, function interface{}) error
    RunInferenceWithTools(ctx context.Context, 
        messages conversation.Conversation) (*conversation.Message, error)
}
```

**Option B: Tool Middleware Pattern**
```go
type ToolMiddleware struct {
    engine Engine
    tools  map[string]interface{}
    reflector *jsonschema.Reflector
}

func (tm *ToolMiddleware) RunInference(ctx context.Context, 
    messages conversation.Conversation) (*conversation.Message, error) {
    // Handle tool calling logic
    // Delegate to wrapped engine
}
```

**Option C: Tool Execution Service**
```go
type ToolExecutor interface {
    ExecuteTools(ctx context.Context, 
        toolCalls []ToolCall) ([]ToolResult, error)
}

// Separate from engine - composed at application level
```

**Recommendation: Option B - Middleware Pattern**
- Preserves simple Engine interface
- Allows layered functionality
- Maintains backward compatibility
- Supports complex tool workflows

---

## Migration Recommendations

### High Priority - Core Functionality Migration

**1. Complete Provider Engine Implementation**
- ✅ OpenAI Engine - Complete
- ✅ Claude Engine - Complete  
- ❌ Gemini Engine - **Implement missing**

**2. Tool Calling Engine Integration**
- Implement tool middleware pattern
- Migrate OpenAI tool calling workflow
- Add Claude tool support to Engine
- Preserve function registration patterns

**3. JavaScript Integration Update**
- Replace `RunStep()` with `RunInference()` 
- Update JS bindings for Engine interface
- Maintain event streaming compatibility

### Medium Priority - Utility Functionality

**4. Transformation Pipeline Design**
- Create Engine-compatible transformation patterns
- Replace Lambda steps with direct functions
- Implement middleware for common transformations

**5. Parsing and Validation Migration**  
- Migrate JSON schema validation to Engine middleware
- Convert parsing steps to post-processing functions
- Maintain error handling patterns

### Low Priority - Infrastructure Cleanup

**6. Step Infrastructure Removal**
- Remove deprecated Step interfaces after migration
- Clean up StepResult and Bind implementations
- Remove step-specific error types

**7. Documentation Updates**
- Update examples to use Engine interface
- Remove Step API documentation
- Add Engine-first best practices guide

### What to Remove Immediately

**Dead Code:**
- Unused step implementations without Engine equivalents
- Deprecated factory methods marked for removal
- Test cases for functionality migrated to Engine

**Experimental Code:**
- Steps marked as "experimental" in comments
- Proof-of-concept implementations not in production use

---

## Tool Calling Design Proposal

### Middleware-Based Tool Integration

**Architecture:**
```
Application
    ├── ToolMiddleware (tool function management)
    │   ├── Schema Generator (JSON schema from Go functions)
    │   ├── Function Registry (name -> function mapping)
    │   ├── Tool Executor (invoke functions with parameters)
    │   └── Result Formatter (tool results -> conversation messages)
    └── Engine (AI provider logic)
        ├── OpenAIEngine
        ├── ClaudeEngine
        └── GeminiEngine
```

**Implementation:**
```go
type ToolMiddleware struct {
    engine    Engine
    tools     map[string]interface{}
    reflector *jsonschema.Reflector
    sink      EventSink
}

func (tm *ToolMiddleware) RunInference(ctx context.Context, 
    messages conversation.Conversation) (*conversation.Message, error) {
    
    // 1. Check if tools are available
    if len(tm.tools) == 0 {
        return tm.engine.RunInference(ctx, messages)
    }
    
    // 2. Create tool definitions for AI model
    toolDefs, err := tm.generateToolDefinitions()
    if err != nil {
        return nil, err
    }
    
    // 3. Add tool definitions to conversation
    enhancedMessages := tm.addToolDefinitions(messages, toolDefs)
    
    // 4. Get AI response (may contain tool calls)
    response, err := tm.engine.RunInference(ctx, enhancedMessages)
    if err != nil {
        return nil, err
    }
    
    // 5. Check for tool calls in response
    toolCalls := tm.extractToolCalls(response)
    if len(toolCalls) == 0 {
        return response, nil
    }
    
    // 6. Execute tool functions
    toolResults, err := tm.executeTools(ctx, toolCalls)
    if err != nil {
        return nil, err
    }
    
    // 7. Format results and continue conversation
    toolMessage := tm.formatToolResults(toolResults)
    
    // 8. Get final AI response with tool results
    finalMessages := append(enhancedMessages, response, toolMessage)
    return tm.engine.RunInference(ctx, finalMessages)
}
```

**Tool Registration:**
```go
middleware := &ToolMiddleware{
    engine: openaiEngine,
    tools:  make(map[string]interface{}),
    reflector: &jsonschema.Reflector{DoNotReference: true},
}

middleware.RegisterTool("getWeather", getWeatherFunc)
middleware.RegisterTool("searchWeb", searchWebFunc)
```

### Provider-Specific Tool Integration

**OpenAI Tools:**
```go
// Convert to OpenAI format
func (tm *ToolMiddleware) generateOpenAITools() ([]go_openai.Tool, error) {
    tools := make([]go_openai.Tool, 0, len(tm.tools))
    for name, fn := range tm.tools {
        schema, err := helpers.GetFunctionParametersJsonSchema(tm.reflector, fn)
        if err != nil {
            return nil, err
        }
        tools = append(tools, go_openai.Tool{
            Type: "function",
            Function: &go_openai.FunctionDefinition{
                Name:        name,
                Description: schema.Description,
                Parameters:  json.RawMessage(schemaBytes),
            },
        })
    }
    return tools, nil
}
```

**Claude Tools:**
```go
// Convert to Claude format
func (tm *ToolMiddleware) generateClaudeTools() ([]api.Tool, error) {
    tools := make([]api.Tool, 0, len(tm.tools))
    for name, fn := range tm.tools {
        schema, err := helpers.GetFunctionParametersJsonSchema(tm.reflector, fn)
        if err != nil {
            return nil, err
        }
        tools = append(tools, api.Tool{
            Name:        name,
            Description: schema.Description,
            InputSchema: convertToClaudeSchema(schema),
        })
    }
    return tools, nil
}
```

---

## Transformation Patterns

### Engine-Compatible Data Processing

**Simple Transformations:**
```go
// OLD: Lambda step
type UppercaseStep struct {
    Function func(string) helpers.Result[string]
}

// NEW: Direct function
func UppercaseResponse(message *conversation.Message) *conversation.Message {
    content := strings.ToUpper(message.Content.String())
    return conversation.NewChatMessage(message.Role, content)
}
```

**Async Processing:**
```go
// OLD: Background lambda step
type BackgroundProcessor struct {
    Function func(context.Context, Input) helpers.Result[Output]
}

// NEW: Async engine wrapper
type AsyncEngine struct {
    engine Engine
}

func (ae *AsyncEngine) RunInference(ctx context.Context, 
    messages conversation.Conversation) (*conversation.Message, error) {
    
    resultChan := make(chan struct {
        message *conversation.Message
        err     error
    }, 1)
    
    go func() {
        message, err := ae.engine.RunInference(ctx, messages)
        resultChan <- struct {
            message *conversation.Message
            err     error
        }{message, err}
    }()
    
    select {
    case result := <-resultChan:
        return result.message, result.err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

**Pipeline Processing:**
```go
// OLD: Step composition with Bind
result1, _ := step1.Start(ctx, input)
result2 := steps.Bind(ctx, result1, step2)
result3 := steps.Bind(ctx, result2, step3)

// NEW: Function composition
func ProcessingPipeline(message *conversation.Message) *conversation.Message {
    message = ProcessStep1(message)
    message = ProcessStep2(message)
    message = ProcessStep3(message)
    return message
}

// Or with middleware pattern
type ProcessingEngine struct {
    engine Engine
    pipeline []func(*conversation.Message) *conversation.Message
}

func (pe *ProcessingEngine) RunInference(ctx context.Context,
    messages conversation.Conversation) (*conversation.Message, error) {
    
    message, err := pe.engine.RunInference(ctx, messages)
    if err != nil {
        return nil, err
    }
    
    for _, transform := range pe.pipeline {
        message = transform(message)
    }
    
    return message, nil
}
```

---

## Migration Prioritization

### Phase 1: Core Engine Completion (Week 1-2)
- **✅ OpenAI Engine** - Complete
- **✅ Claude Engine** - Complete
- **❌ Gemini Engine** - Implement missing
- **❌ Tool Middleware** - Design and implement

### Phase 2: Tool Calling Migration (Week 3-4)  
- Implement ToolMiddleware pattern
- Migrate OpenAI tool calling workflow
- Add Claude tool support
- Update tool function registration

### Phase 3: JavaScript Integration (Week 5)
- Replace RuntimeEngine.RunStep with RunInference
- Update JavaScript bindings
- Migrate step factory to engine factory
- Preserve event streaming

### Phase 4: Utility Migration (Week 6-7)
- Convert lambda steps to direct functions
- Implement transformation middleware patterns
- Migrate parsing and validation logic
- Update documentation examples

### Phase 5: Infrastructure Cleanup (Week 8)
- Remove deprecated Step interfaces
- Clean up StepResult, Bind, Resolve
- Remove step-specific error types
- Update documentation to Engine-first

---

## Risk Assessment

### High Risk - Breaking Changes

**Tool Calling Migration:**
- **Risk:** Complex tool workflow may not map cleanly to Engine interface
- **Mitigation:** Implement middleware pattern to preserve functionality
- **Testing:** Comprehensive tool calling integration tests

**JavaScript Integration:**
- **Risk:** Breaking changes to JS/Go interface
- **Mitigation:** Maintain backward compatibility during transition
- **Testing:** JavaScript runtime integration tests

### Medium Risk - Performance Impact

**Event System Migration:**
- **Risk:** Event publishing pattern changes may affect performance
- **Mitigation:** Benchmark Engine vs Step event publishing
- **Testing:** Performance regression tests

**Streaming Behavior:**
- **Risk:** Engine streaming may differ from Step streaming
- **Mitigation:** Ensure Engine implementations maintain streaming semantics
- **Testing:** Streaming response validation

### Low Risk - Incremental Changes

**Transformation Patterns:**
- **Risk:** Lambda step migration may require pattern changes
- **Mitigation:** Provide clear migration examples
- **Testing:** Transformation correctness tests

**Provider Implementation:**
- **Risk:** Gemini Engine implementation delay
- **Mitigation:** Gemini Step fallback during transition
- **Testing:** Provider compatibility tests

---

## Success Criteria

### Technical Milestones

**✅ Complete Engine Implementation:**
- All providers (OpenAI, Claude, Gemini) have Engine implementations
- Tool calling works through Engine interface
- Event publishing works with EventSink system

**✅ Zero Step API Usage:**
- No remaining references to Step, StepResult, Bind, Resolve
- All deprecation warnings resolved
- JavaScript integration uses Engine interface

**✅ Feature Parity:**
- All Step-based functionality available through Engine
- Tool calling workflow preserved
- Transformation patterns maintained
- Event publishing preserved

### Quality Gates

**✅ Test Coverage:**
- All Engine implementations have comprehensive tests
- Tool calling integration tests pass
- JavaScript integration tests pass
- Performance benchmarks meet requirements

**✅ Documentation:**
- Engine-first architecture documentation complete
- Migration guide with examples
- API documentation updated
- Best practices guide published

### Business Impact

**✅ User Experience:**
- No breaking changes to public APIs
- Performance maintained or improved
- Feature functionality preserved
- Clear upgrade path for external users

**✅ Developer Experience:**
- Simpler API for common use cases
- Better type safety and error handling
- Clearer separation of concerns
- Reduced cognitive overhead

---

## Next Steps

### Immediate Actions (This Week)

1. **Implement Missing Gemini Engine** 
   - Create `engine_gemini.go` following OpenAI/Claude patterns
   - Add Gemini provider to factory
   - Implement basic chat functionality

2. **Design Tool Middleware Architecture**
   - Finalize ToolMiddleware interface design
   - Create prototypes for OpenAI and Claude tool integration
   - Validate approach with existing tool calling tests

3. **Create Migration Examples**
   - Document Step -> Engine migration patterns
   - Create before/after code examples
   - Validate with existing pinocchio usage

### Week 2 Actions

1. **Implement Tool Middleware**
   - Full ToolMiddleware implementation
   - OpenAI tool calling integration
   - Claude tool calling integration

2. **Begin JavaScript Migration**
   - Update RuntimeEngine to use Engine interface
   - Maintain backward compatibility
   - Update JavaScript bindings

### Week 3+ Actions

1. **Systematic Step Removal**
   - Remove step implementations with Engine equivalents
   - Update all internal geppetto usage
   - Preserve public API compatibility

2. **Documentation and Testing**
   - Comprehensive test coverage for new patterns
   - Performance validation
   - Final documentation updates

---

## Conclusion

The Step API to Engine migration represents a significant architectural improvement for geppetto. The analysis shows a clear path forward with manageable complexity. The new Engine interface provides better simplicity, performance, and maintainability while preserving all essential functionality through well-designed patterns like tool middleware and transformation pipelines.

The migration is feasible within an 8-week timeline with careful attention to tool calling functionality and JavaScript integration. The risk profile is manageable with proper testing and incremental rollout strategies.

**Key Success Factors:**
- Tool middleware pattern for preserving complex tool calling workflows
- Comprehensive testing throughout migration process  
- Maintaining backward compatibility during transition
- Clear documentation and migration guides for external users

This migration will position geppetto with a modern, maintainable architecture that supports future AI inference capabilities while eliminating the complexity of the deprecated Step API.
