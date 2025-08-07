package toolhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/rs/zerolog/log"
)

// ToolCall represents a tool call extracted from an AI response
// This is a simplified version that gets converted to tools.ToolCall for execution
type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]interface{}
}

// ToolResult represents the result of executing a tool
// This is a simplified version of tools.ToolResult
type ToolResult struct {
	ToolCallID string
	Result     interface{}
	Error      error
}

// ToolConfig holds configuration for tool calling workflow
type ToolConfig struct {
	MaxIterations     int
	Timeout           time.Duration
	MaxParallelTools  int
	ToolChoice        tools.ToolChoice
	AllowedTools      []string
	ToolErrorHandling tools.ToolErrorHandling
}

// ExtractToolCalls extracts tool calls from the last message in a conversation
func ExtractToolCalls(conv conversation.Conversation) []ToolCall {
	log.Debug().Int("conversation_length", len(conv)).Msg("ExtractToolCalls: analyzing conversation")
	
	if len(conv) == 0 {
		log.Debug().Msg("ExtractToolCalls: empty conversation, no tool calls")
		return nil
	}

	lastMessage := conv[len(conv)-1]
	log.Debug().
		Str("content_type", string(lastMessage.Content.ContentType())).
		Msg("ExtractToolCalls: examining last message")

	// Check if the last message contains tool calls
	if toolUse, ok := lastMessage.Content.(*conversation.ToolUseContent); ok {
		log.Debug().
			Str("tool_id", toolUse.ToolID).
			Str("tool_name", toolUse.Name).
			RawJSON("tool_input", toolUse.Input).
			Msg("ExtractToolCalls: found tool use content")
		
		// Parse the input JSON into a map
		var args map[string]interface{}
		if err := json.Unmarshal(toolUse.Input, &args); err != nil {
			log.Warn().Err(err).Msg("ExtractToolCalls: failed to parse tool input, using empty args")
			// If parsing fails, return empty args
			args = make(map[string]interface{})
		}

		toolCall := ToolCall{
			ID:        toolUse.ToolID,
			Name:      toolUse.Name,
			Arguments: args,
		}
		
		log.Info().Interface("tool_call", toolCall).Msg("ExtractToolCalls: extracted tool call")
		return []ToolCall{toolCall}
	}

	// Check for Claude original content containing tool calls  
	if _, ok := lastMessage.Content.(*conversation.ChatMessageContent); ok {
		if claudeContent, exists := lastMessage.Metadata["claude_original_content"]; exists {
			if originalContent, ok := claudeContent.([]api.Content); ok {
				log.Debug().Int("content_blocks", len(originalContent)).Msg("ExtractToolCalls: found Claude original content")
				
				var toolCalls []ToolCall
				for _, content := range originalContent {
					if toolUseContent, ok := content.(api.ToolUseContent); ok {
						// Parse the input JSON into a map
						var args map[string]interface{}
						if err := json.Unmarshal(toolUseContent.Input, &args); err != nil {
							log.Warn().Err(err).Str("tool_id", toolUseContent.ID).Msg("ExtractToolCalls: failed to parse Claude tool input, using empty args")
							args = make(map[string]interface{})
						}

						toolCall := ToolCall{
							ID:        toolUseContent.ID,
							Name:      toolUseContent.Name,
							Arguments: args,
						}
						
						log.Debug().Interface("tool_call", toolCall).Msg("ExtractToolCalls: extracted Claude tool call from original content")
						toolCalls = append(toolCalls, toolCall)
					}
				}
				
				if len(toolCalls) > 0 {
					log.Info().Int("tool_calls_count", len(toolCalls)).Msg("ExtractToolCalls: extracted Claude tool calls from original content")
					return toolCalls
				}
			}
		}
	}



	// TODO: Handle multiple tool calls in a single message
	// This would require provider-specific parsing logic for different formats
	log.Debug().Msg("ExtractToolCalls: no tool calls found in last message")
	return nil
}

// ExecuteToolCalls executes multiple tool calls and returns their results
func ExecuteToolCalls(ctx context.Context, toolCalls []ToolCall, registry tools.ToolRegistry) []ToolResult {
	log.Debug().Int("tool_call_count", len(toolCalls)).Msg("ExecuteToolCalls: starting tool execution")
	
	if len(toolCalls) == 0 {
		log.Debug().Msg("ExecuteToolCalls: no tool calls to execute")
		return nil
	}

	// Create a tool executor with default config
	config := tools.DefaultToolConfig()
	executor := tools.NewDefaultToolExecutor(config)
	
	log.Debug().Interface("executor_config", config).Msg("ExecuteToolCalls: created tool executor")

	// Convert our ToolCall to tools.ToolCall
	var execCalls []tools.ToolCall
	for _, call := range toolCalls {
		log.Debug().
			Str("tool_id", call.ID).
			Str("tool_name", call.Name).
			Interface("arguments", call.Arguments).
			Msg("ExecuteToolCalls: converting tool call")
			
		argBytes, err := json.Marshal(call.Arguments)
		if err != nil {
			log.Warn().Err(err).Str("tool_name", call.Name).Msg("ExecuteToolCalls: failed to marshal arguments, using empty object")
			// Handle conversion error
			argBytes = []byte("{}")
		}

		execCalls = append(execCalls, tools.ToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: json.RawMessage(argBytes),
		})
	}

	log.Info().Int("exec_call_count", len(execCalls)).Msg("ExecuteToolCalls: executing tools")
	
	// Execute the tools
	execResults, err := executor.ExecuteToolCalls(ctx, execCalls, registry)
	
	if err != nil {
		log.Error().Err(err).Msg("ExecuteToolCalls: tool execution failed")
	} else {
		log.Info().Int("result_count", len(execResults)).Msg("ExecuteToolCalls: tool execution completed")
	}

	// Convert results back to our format
	results := make([]ToolResult, len(toolCalls))
	for i, call := range toolCalls {
		if err != nil {
			results[i] = ToolResult{
				ToolCallID: call.ID,
				Result:     nil,
				Error:      err,
			}
		} else if i < len(execResults) && execResults[i] != nil {
			var resultErr error
			if execResults[i].Error != "" {
				resultErr = fmt.Errorf("%s", execResults[i].Error)
			}

			results[i] = ToolResult{
				ToolCallID: call.ID,
				Result:     execResults[i].Result,
				Error:      resultErr,
			}
		} else {
			results[i] = ToolResult{
				ToolCallID: call.ID,
				Result:     nil,
				Error:      fmt.Errorf("no result returned"),
			}
		}
	}

	return results
}

// AppendToolResults appends tool results to a conversation
func AppendToolResults(conv conversation.Conversation, results []ToolResult) conversation.Conversation {
	log.Debug().
		Int("conversation_length", len(conv)).
		Int("results_count", len(results)).
		Msg("AppendToolResults: appending tool results to conversation")
		
	updated := make(conversation.Conversation, len(conv))
	copy(updated, conv)

	for _, result := range results {
		log.Debug().
			Str("tool_call_id", result.ToolCallID).
			Bool("has_error", result.Error != nil).
			Msg("AppendToolResults: processing tool result")
		var content conversation.MessageContent
		if result.Error != nil {
			content = &conversation.ToolResultContent{
				ToolID: result.ToolCallID,
				Result: fmt.Sprintf("Error: %s", result.Error.Error()),
			}
		} else {
			// Convert result to string representation
			var resultStr string
			if resultBytes, err := json.Marshal(result.Result); err == nil {
				resultStr = string(resultBytes)
			} else {
				resultStr = fmt.Sprintf("%v", result.Result)
			}

			content = &conversation.ToolResultContent{
				ToolID: result.ToolCallID,
				Result: resultStr,
			}
		}

		message := conversation.NewMessage(content)
		updated = append(updated, message)
	}

	return updated
}

// RunToolCallingLoop runs a complete tool calling workflow with automatic iteration
func RunToolCallingLoop(ctx context.Context, engine engine.Engine, initialConversation conversation.Conversation, registry tools.ToolRegistry, config ToolConfig) (conversation.Conversation, error) {
	log.Info().
		Int("max_iterations", config.MaxIterations).
		Int("initial_conversation_length", len(initialConversation)).
		Msg("RunToolCallingLoop: starting tool calling workflow")
		
	conv := initialConversation

	for i := 0; i < config.MaxIterations; i++ {
		log.Info().Int("iteration", i+1).Msg("RunToolCallingLoop: starting iteration")
		
		// Run inference
		log.Debug().Msg("RunToolCallingLoop: calling engine.RunInference")
		response, err := engine.RunInference(ctx, conv)
		if err != nil {
			log.Error().Err(err).Int("iteration", i+1).Msg("RunToolCallingLoop: engine inference failed")
			return nil, err
		}
		
		log.Debug().
			Int("response_length", len(response)).
			Int("new_messages", len(response)-len(conv)).
			Msg("RunToolCallingLoop: engine inference completed")

		// Extract tool calls
		log.Debug().Msg("RunToolCallingLoop: extracting tool calls")
		toolCalls := ExtractToolCalls(response)
		if len(toolCalls) == 0 {
			log.Info().Int("iteration", i+1).Msg("RunToolCallingLoop: no tool calls found, workflow complete")
			// No more tool calls, we're done
			return response, nil
		}

		log.Info().
			Int("iteration", i+1).
			Int("tool_calls_found", len(toolCalls)).
			Msg("RunToolCallingLoop: found tool calls, executing")

		// Execute tools
		toolResults := ExecuteToolCalls(ctx, toolCalls, registry)

		// Append results to conversation for next iteration
		conv = AppendToolResults(response, toolResults)
		
		log.Debug().
			Int("iteration", i+1).
			Int("updated_conversation_length", len(conv)).
			Msg("RunToolCallingLoop: appended tool results, continuing to next iteration")
	}

	log.Warn().Int("max_iterations", config.MaxIterations).Msg("RunToolCallingLoop: maximum iterations reached")
	return conv, fmt.Errorf("max iterations (%d) reached", config.MaxIterations)
}

// NewToolConfig creates a default tool configuration
func NewToolConfig() ToolConfig {
	return ToolConfig{
		MaxIterations:     5,
		Timeout:           30 * time.Second,
		MaxParallelTools:  3,
		ToolChoice:        tools.ToolChoiceAuto,
		AllowedTools:      nil, // Allow all tools
		ToolErrorHandling: tools.ToolErrorContinue,
	}
}

// WithMaxIterations sets the maximum number of tool calling iterations
func (c ToolConfig) WithMaxIterations(max int) ToolConfig {
	c.MaxIterations = max
	return c
}

// WithTimeout sets the timeout for tool execution
func (c ToolConfig) WithTimeout(timeout time.Duration) ToolConfig {
	c.Timeout = timeout
	return c
}

// WithMaxParallelTools sets the maximum number of parallel tool executions
func (c ToolConfig) WithMaxParallelTools(max int) ToolConfig {
	c.MaxParallelTools = max
	return c
}

// WithToolChoice sets the tool choice strategy
func (c ToolConfig) WithToolChoice(choice tools.ToolChoice) ToolConfig {
	c.ToolChoice = choice
	return c
}

// WithAllowedTools sets the list of allowed tools
func (c ToolConfig) WithAllowedTools(toolNames []string) ToolConfig {
	c.AllowedTools = toolNames
	return c
}

// WithToolErrorHandling sets the tool error handling strategy
func (c ToolConfig) WithToolErrorHandling(handling tools.ToolErrorHandling) ToolConfig {
	c.ToolErrorHandling = handling
	return c
}
