# Simplified Tool Calling Architecture with Helper Utilities

## Overview

This document outlines a simplified approach to tool calling in Geppetto that keeps engines focused on their core responsibility (provider API calls) while providing easy-to-use helper utilities for tool calling orchestration at the application level.

**Key Insight**: Instead of building complex tool orchestration into engines, provide simple helper functions that any application can use to add tool calling capabilities to any engine.

## Architecture Goals

1. **Simple Engine Interface**: Single `Engine` interface that only handles inference
2. **Separation of Concerns**: Engine = I/O, Helpers = tool logic, App = orchestration  
3. **Provider Agnostic**: Tool helpers work with any engine's output
4. **Easy Adoption**: Add tool calling to existing applications with minimal changes
5. **Reduced Complexity**: Eliminate wrapper/orchestrator pattern complexity

## Current vs. Simplified Architecture

### Current Complex Architecture
```go
// Multiple interfaces to implement
type Engine interface { RunInference(...) }
type EngineWithTools interface { Engine; ConfigureTools(...); PrepareToolsForRequest(...) }
type StreamingEngine interface { Engine; RunInferenceStream(...) }
type EngineWithToolsAndStreaming interface { EngineWithTools; StreamingEngine }

// Complex orchestration with wrappers
orchestrator := tools.NewInferenceOrchestrator(...)
engineWrapper := tools.NewEngineWrapper(baseEngine, registry, toolConfig)
result, err := engineWrapper.RunInference(ctx, conversation)
```

### Simplified Helper-Based Architecture
```go
// Single simple interface
type Engine interface {
    RunInference(ctx context.Context, conversation Conversation) (Conversation, error)
}

// Easy helper-based orchestration
response, err := engine.RunInference(ctx, conversation)
toolCalls := toolhelpers.ExtractToolCalls(response)
toolResults := toolhelpers.ExecuteToolCalls(ctx, toolCalls, registry)
updatedConversation := toolhelpers.AppendToolResults(response, toolResults)
```

## Core Components

### 1. Single Engine Interface

```go
package engine

// Engine represents an AI inference engine that processes conversations
// and returns AI-generated responses. All provider-specific engines implement this.
type Engine interface {
    // RunInference processes a conversation and returns the full updated conversation.
    // The engine handles provider-specific API calls but does NOT handle tool orchestration.
    // Tool calls in the response should be preserved as-is for helper processing.
    RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error)
}
```

### 2. Tool Helper Utilities

```go
package toolhelpers

// ToolCall represents a tool call extracted from an AI response
type ToolCall struct {
    ID        string
    Name      string
    Arguments map[string]interface{}
}

// ToolResult represents the result of executing a tool
type ToolResult struct {
    ToolCallID string
    Result     interface{}
    Error      error
}

// ExtractToolCalls extracts tool calls from the last message in a conversation
func ExtractToolCalls(conversation conversation.Conversation) []ToolCall {
    if len(conversation) == 0 {
        return nil
    }
    
    lastMessage := conversation[len(conversation)-1]
    
    // Check if the last message contains tool calls
    if toolUse, ok := lastMessage.Content.(*conversation.ToolUseContent); ok {
        return []ToolCall{{
            ID:        toolUse.ToolID,
            Name:      toolUse.Name,
            Arguments: parseToolInput(toolUse.Input),
        }}
    }
    
    // Handle multiple tool calls in a single message
    // (provider-specific parsing logic here)
    
    return nil
}

// ExecuteToolCalls executes multiple tool calls and returns their results
func ExecuteToolCalls(ctx context.Context, toolCalls []ToolCall, registry ToolRegistry) []ToolResult {
    results := make([]ToolResult, len(toolCalls))
    
    for i, call := range toolCalls {
        result, err := registry.ExecuteTool(ctx, call.Name, call.Arguments)
        results[i] = ToolResult{
            ToolCallID: call.ID,
            Result:     result,
            Error:      err,
        }
    }
    
    return results
}

// AppendToolResults appends tool results to a conversation
func AppendToolResults(conversation conversation.Conversation, results []ToolResult) conversation.Conversation {
    updated := append(conversation.Conversation(nil), conversation...)
    
    for _, result := range results {
        var content conversation.MessageContent
        if result.Error != nil {
            content = conversation.NewToolResultContent(result.ToolCallID, "", result.Error.Error())
        } else {
            content = conversation.NewToolResultContent(result.ToolCallID, result.Result, "")
        }
        
        message := conversation.NewMessage(content)
        updated = append(updated, message)
    }
    
    return updated
}

// RunToolCallingLoop runs a complete tool calling workflow with automatic iteration
func RunToolCallingLoop(ctx context.Context, engine engine.Engine, initialConversation conversation.Conversation, registry ToolRegistry, config ToolConfig) (conversation.Conversation, error) {
    conversation := initialConversation
    
    for i := 0; i < config.MaxIterations; i++ {
        // Run inference
        response, err := engine.RunInference(ctx, conversation)
        if err != nil {
            return nil, err
        }
        
        // Extract tool calls
        toolCalls := ExtractToolCalls(response)
        if len(toolCalls) == 0 {
            // No more tool calls, we're done
            return response, nil
        }
        
        // Execute tools
        toolResults := ExecuteToolCalls(ctx, toolCalls, registry)
        
        // Append results to conversation for next iteration
        conversation = AppendToolResults(response, toolResults)
    }
    
    return conversation, fmt.Errorf("max iterations (%d) reached", config.MaxIterations)
}
```

### 3. Simplified Engine Implementation

```go
// OpenAI engine only needs to handle API calls
func (e *OpenAIEngine) RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
    // Create OpenAI request from conversation
    req, err := e.createRequest(messages)
    if err != nil {
        return nil, err
    }
    
    // Add tools to request if any are configured
    if len(e.tools) > 0 {
        req.Tools = e.convertToolsToOpenAIFormat(e.tools)
        req.ToolChoice = string(e.toolChoice)
    }
    
    // Make API call
    resp, err := e.client.CreateChatCompletion(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // Convert response to conversation message and return
    message := e.convertResponseToMessage(resp)
    result := append(conversation.Conversation(nil), messages...)
    result = append(result, message)
    
    return result, nil
}
```

## Usage Examples

### Basic Tool Calling
```go
// Simple one-shot tool calling
response, err := engine.RunInference(ctx, conversation)
if err != nil {
    return err
}

toolCalls := toolhelpers.ExtractToolCalls(response)
if len(toolCalls) > 0 {
    toolResults := toolhelpers.ExecuteToolCalls(ctx, toolCalls, registry)
    finalConversation := toolhelpers.AppendToolResults(response, toolResults)
}
```

### Complete Tool Calling Loop
```go
// Automatic multi-turn tool calling
config := toolhelpers.ToolConfig{
    MaxIterations: 5,
    Timeout:       30 * time.Second,
}

finalConversation, err := toolhelpers.RunToolCallingLoop(
    ctx, engine, conversation, registry, config)
```

### Advanced: Custom Tool Orchestration
```go
// Custom logic with full control
conversation := initialConversation

for iteration := 0; iteration < maxIterations; iteration++ {
    // Custom pre-processing
    if iteration > 0 {
        conversation = addIterationMetadata(conversation, iteration)
    }
    
    response, err := engine.RunInference(ctx, conversation)
    if err != nil {
        return err
    }
    
    toolCalls := toolhelpers.ExtractToolCalls(response)
    if len(toolCalls) == 0 {
        break // Done
    }
    
    // Custom filtering/validation
    validCalls := filterAndValidateToolCalls(toolCalls)
    
    // Execute with custom error handling
    toolResults := make([]toolhelpers.ToolResult, len(validCalls))
    for i, call := range validCalls {
        result, err := executeWithRetry(ctx, call, registry)
        toolResults[i] = toolhelpers.ToolResult{
            ToolCallID: call.ID,
            Result:     result,
            Error:      err,
        }
    }
    
    conversation = toolhelpers.AppendToolResults(response, toolResults)
}
```

## Migration Path from Current Architecture

### Step 1: Simplify Engine Interface
- Remove `EngineWithTools`, `StreamingEngine` interfaces
- Keep only `Engine` with `RunInference` method
- Move tool configuration to engine constructor/options

### Step 2: Extract Tool Orchestration
- Move tool calling logic from `tools.NewEngineWrapper` to helper functions
- Replace `InferenceOrchestrator` with `toolhelpers.RunToolCallingLoop`
- Remove middleware pattern for tool calling

### Step 3: Update Applications
Current complex pattern:
```go
baseEngine, _ := factory.NewEngineFromParsedLayers(parsedLayers)
engineWrapper := tools.NewEngineWrapper(baseEngine, registry, toolConfig)
result, err := engineWrapper.RunInference(ctx, conversation)
```

New simplified pattern:
```go
engine, _ := factory.NewEngineFromParsedLayers(parsedLayers)
result, err := toolhelpers.RunToolCallingLoop(ctx, engine, conversation, registry, toolConfig)
```

## Benefits of This Approach

1. **Simpler Engine Interface**: Single method to implement instead of 4 interfaces
2. **Provider Agnostic**: Tool helpers work with OpenAI, Claude, Gemini equally
3. **Easy Testing**: Mock engines for testing tool logic, mock registries for testing engines
4. **Flexible Orchestration**: Applications choose their own tool calling patterns
5. **Reduced Dependencies**: Engines don't need to depend on tool execution systems
6. **Cleaner Separation**: Clear boundaries between inference and tool execution

## Updated Generic Tool Calling Example

With the helper approach, the complex generic tool calling example becomes much simpler:

```go
func (c *GenericToolCallingCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
    // 1. Create simple engine (no wrappers needed)
    engine, err := factory.NewEngineFromParsedLayers(parsedLayers)
    if err != nil {
        return err
    }
    
    // 2. Create tool registry
    registry := tools.NewInMemoryToolRegistry()
    registry.RegisterTool("get_weather", weatherTool)
    registry.RegisterTool("calculator", calculatorTool)
    
    // 3. Build conversation
    conversation := buildConversation(s.Prompt)
    
    // 4. Run tool calling with simple helper
    config := toolhelpers.ToolConfig{
        MaxIterations: s.MaxIterations,
        ToolChoice:    s.ToolChoice,
        MaxParallelTools: s.MaxParallelTools,
    }
    
    result, err := toolhelpers.RunToolCallingLoop(ctx, engine, conversation, registry, config)
    if err != nil {
        return err
    }
    
    // 5. Print results
    printConversation(w, result)
    return nil
}
```

## Implementation Notes

1. **Tool Call Extraction**: Provider-specific logic in `ExtractToolCalls` handles different tool call formats (OpenAI vs Claude vs Gemini)

2. **Error Handling**: Tool execution errors are captured in `ToolResult.Error` and can be handled by applications

3. **Streaming Support**: Can be added to helpers later without changing the core engine interface

4. **Performance**: Tool execution can be parallelized within `ExecuteToolCalls` as needed

5. **Configuration**: Engine-specific tool configuration (like OpenAI's `tool_choice`) is handled during engine creation, not in helpers

## Conclusion

This helper-based approach significantly simplifies the Geppetto tool calling architecture while maintaining all functionality. It provides clear separation of concerns, reduces implementation complexity, and makes tool calling easier to add to any application.

The key insight is that tool calling orchestration is an application-level concern, not an engine-level concern. Engines should focus on provider API calls, while helpers make it easy for applications to orchestrate tool calling workflows.
