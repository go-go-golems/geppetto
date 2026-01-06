package toolhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/toolblocks"
	"github.com/go-go-golems/geppetto/pkg/inference/toolcontext"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
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

// SnapshotHook can be attached to context to capture snapshots of the Turn at defined phases
type SnapshotHook func(ctx context.Context, t *turns.Turn, phase string)

type snapshotHookKey struct{}

// WithTurnSnapshotHook attaches a snapshot hook to the context
func WithTurnSnapshotHook(ctx context.Context, hook SnapshotHook) context.Context {
	if hook == nil {
		return ctx
	}
	return context.WithValue(ctx, snapshotHookKey{}, hook)
}

func getSnapshotHook(ctx context.Context) SnapshotHook {
	v := ctx.Value(snapshotHookKey{})
	if v == nil {
		return nil
	}
	if h, ok := v.(SnapshotHook); ok {
		return h
	}
	return nil
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

	// Collect all trailing ToolUseContent messages (OpenAI can emit multiple in one turn)
	toolCallsReversed := []ToolCall{}
	for i := len(conv) - 1; i >= 0; i-- {
		msg := conv[i]
		if toolUse, ok := msg.Content.(*conversation.ToolUseContent); ok {
			// Parse input JSON
			var args map[string]interface{}
			if err := json.Unmarshal(toolUse.Input, &args); err != nil {
				log.Warn().Err(err).Msg("ExtractToolCalls: failed to parse tool input, using empty args")
				args = make(map[string]interface{})
			}
			toolCall := ToolCall{ID: toolUse.ToolID, Name: toolUse.Name, Arguments: args}
			log.Debug().Interface("tool_call", toolCall).Msg("ExtractToolCalls: found trailing tool call")
			toolCallsReversed = append(toolCallsReversed, toolCall)
			continue
		}
		// Stop once we hit a non tool-use message
		break
	}
	if len(toolCallsReversed) > 0 {
		// Reverse to restore original order
		toolCalls := make([]ToolCall, len(toolCallsReversed))
		for i := range toolCallsReversed {
			toolCalls[i] = toolCallsReversed[len(toolCallsReversed)-1-i]
		}
		log.Debug().Int("tool_calls_count", len(toolCalls)).Msg("ExtractToolCalls: extracted trailing tool calls")
		return toolCalls
	}

	lastMessage := conv[len(conv)-1]
	log.Debug().
		Str("content_type", string(lastMessage.Content.ContentType())).
		Msg("ExtractToolCalls: examining last message")

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
					log.Debug().Int("tool_calls_count", len(toolCalls)).Msg("ExtractToolCalls: extracted Claude tool calls from original content")
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

	log.Debug().Int("exec_call_count", len(execCalls)).Msg("ExecuteToolCalls: executing tools")

	// Execute the tools
	execResults, err := executor.ExecuteToolCalls(ctx, execCalls, registry)

	if err != nil {
		log.Error().Err(err).Msg("ExecuteToolCalls: tool execution failed")
	} else {
		log.Debug().Int("result_count", len(execResults)).Msg("ExecuteToolCalls: tool execution completed")
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

// RunToolCallingLoop runs a complete tool calling workflow with automatic iteration.
// This variant accepts and returns a Turn, avoiding the conversation manager.
func RunToolCallingLoop(ctx context.Context, eng engine.Engine, initialTurn *turns.Turn, registry tools.ToolRegistry, config ToolConfig) (*turns.Turn, error) {
	log.Debug().
		Int("max_iterations", config.MaxIterations).
		Int("initial_blocks", func() int {
			if initialTurn != nil {
				return len(initialTurn.Blocks)
			}
			return 0
		}()).
		Msg("RunToolCallingLoop: starting tool calling workflow (Turn-based)")

	// Use provided Turn or create a new one
	t := initialTurn
	if t == nil {
		t = &turns.Turn{}
	}

	// Attach runtime registry to context so engines/middleware/executors can access it.
	// No Turn.Data registry (runtime-only) is stored.
	ctx = toolcontext.WithRegistry(ctx, registry)

	if err := engine.KeyToolConfig.Set(&t.Data, engine.ToolConfig{
		Enabled:           true,
		ToolChoice:        engine.ToolChoice(config.ToolChoice),
		MaxIterations:     config.MaxIterations,
		ExecutionTimeout:  config.Timeout,
		MaxParallelTools:  config.MaxParallelTools,
		AllowedTools:      config.AllowedTools,
		ToolErrorHandling: engine.ToolErrorHandling(config.ToolErrorHandling),
	}); err != nil {
		return nil, errors.Wrap(err, "set tool config")
	}

	for i := 0; i < config.MaxIterations; i++ {
		log.Debug().Int("iteration", i+1).Msg("RunToolCallingLoop: engine step")

		// Run inference (provider may append llm_text and tool_call blocks)
		if hook := getSnapshotHook(ctx); hook != nil {
			hook(ctx, t, "pre_inference")
		}
		updated, err := eng.RunInference(ctx, t)
		if err != nil {
			log.Error().Err(err).Int("iteration", i+1).Msg("RunToolCallingLoop: engine inference failed")
			return nil, err
		}
		if hook := getSnapshotHook(ctx); hook != nil {
			hook(ctx, updated, "post_inference")
		}

		// Extract pending tool calls from blocks
		calls_ := toolblocks.ExtractPendingToolCalls(updated)
		// adapt to local ToolCall type for executor
		var calls []ToolCall
		for _, c := range calls_ {
			calls = append(calls, ToolCall{ID: c.ID, Name: c.Name, Arguments: c.Arguments})
		}
		if len(calls) == 0 {
			// Done; convert to conversation and return
			return updated, nil
		}

		// Execute tools
		results := ExecuteToolCallsTurn(ctx, calls)

		// Append tool_use blocks
		// map to shared ToolResult and append
		var shared []toolblocks.ToolResult
		for _, r := range results {
			if r.Error != nil {
				shared = append(shared, toolblocks.ToolResult{ID: r.ToolCallID, Error: r.Error.Error()})
			} else {
				// stringify result
				var content string
				if b, err := json.Marshal(r.Result); err == nil {
					content = string(b)
				} else {
					content = fmt.Sprintf("%v", r.Result)
				}
				shared = append(shared, toolblocks.ToolResult{ID: r.ToolCallID, Content: content})
			}
		}
		toolblocks.AppendToolResultsBlocks(updated, shared)
		if hook := getSnapshotHook(ctx); hook != nil {
			hook(ctx, updated, "post_tools")
		}

		// Continue next iteration with same turn
		t = updated
	}

	log.Warn().Int("max_iterations", config.MaxIterations).Msg("RunToolCallingLoop: maximum iterations reached")
	return t, fmt.Errorf("max iterations (%d) reached", config.MaxIterations)
}

// extractPendingToolCallsTurn mirrors middleware logic locally to avoid import cycles
// extractPendingToolCallsTurn replaced by toolblocks.ExtractPendingToolCalls

// ExecuteToolCallsTurn executes ToolCalls using the default executor and returns simplified results
func ExecuteToolCallsTurn(ctx context.Context, toolCalls []ToolCall) []ToolResult {
	log.Debug().Int("tool_call_count", len(toolCalls)).Msg("ExecuteToolCallsTurn: starting tool execution")
	if len(toolCalls) == 0 {
		return nil
	}
	registry, ok := toolcontext.RegistryFrom(ctx)
	if !ok || registry == nil {
		results := make([]ToolResult, len(toolCalls))
		for i, c := range toolCalls {
			results[i] = ToolResult{ToolCallID: c.ID, Result: nil, Error: fmt.Errorf("no tool registry in context")}
		}
		return results
	}
	executor := tools.NewDefaultToolExecutor(tools.DefaultToolConfig())
	// Convert calls
	execCalls := make([]tools.ToolCall, 0, len(toolCalls))
	for _, call := range toolCalls {
		argBytes, _ := json.Marshal(call.Arguments)
		execCalls = append(execCalls, tools.ToolCall{ID: call.ID, Name: call.Name, Arguments: json.RawMessage(argBytes)})
	}
	execResults, err := executor.ExecuteToolCalls(ctx, execCalls, registry)
	results := make([]ToolResult, len(toolCalls))
	for i, c := range toolCalls {
		if err != nil || i >= len(execResults) || execResults[i] == nil {
			results[i] = ToolResult{ToolCallID: c.ID, Result: nil, Error: fmt.Errorf("no result returned")}
			continue
		}
		var resultErr error
		if execResults[i].Error != "" {
			resultErr = fmt.Errorf("%s", execResults[i].Error)
		}
		results[i] = ToolResult{ToolCallID: c.ID, Result: execResults[i].Result, Error: resultErr}
	}
	return results
}

// appendToolResultsBlocksTurn appends tool_use blocks for results
// appendToolResultsBlocksTurn replaced by toolblocks.AppendToolResultsBlocks

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
func (c ToolConfig) WithMaxIterations(maxIterations int) ToolConfig {
	c.MaxIterations = maxIterations
	return c
}

// WithTimeout sets the timeout for tool execution
func (c ToolConfig) WithTimeout(timeout time.Duration) ToolConfig {
	c.Timeout = timeout
	return c
}

// WithMaxParallelTools sets the maximum number of parallel tool executions
func (c ToolConfig) WithMaxParallelTools(maxParallel int) ToolConfig {
	c.MaxParallelTools = maxParallel
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
