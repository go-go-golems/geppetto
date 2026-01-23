package toolloop

import (
	"time"

	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

// ToolConfig configures tool calling behavior.
type ToolConfig struct {
	MaxIterations     int
	Timeout           time.Duration
	MaxParallelTools  int
	ToolChoice        tools.ToolChoice
	AllowedTools      []string
	ToolErrorHandling tools.ToolErrorHandling
	Executor          tools.ToolExecutor
}

// NewToolConfig creates a default tool configuration.
func NewToolConfig() ToolConfig {
	return ToolConfig{
		MaxIterations:     5,
		Timeout:           30 * time.Second,
		MaxParallelTools:  3,
		ToolChoice:        tools.ToolChoiceAuto,
		AllowedTools:      nil,
		ToolErrorHandling: tools.ToolErrorContinue,
	}
}

// WithMaxIterations sets the maximum number of tool calling iterations.
func (c ToolConfig) WithMaxIterations(maxIterations int) ToolConfig {
	c.MaxIterations = maxIterations
	return c
}

// WithTimeout sets the timeout for tool execution.
func (c ToolConfig) WithTimeout(timeout time.Duration) ToolConfig {
	c.Timeout = timeout
	return c
}

// WithMaxParallelTools sets the maximum number of parallel tool executions.
func (c ToolConfig) WithMaxParallelTools(maxParallel int) ToolConfig {
	c.MaxParallelTools = maxParallel
	return c
}

// WithToolChoice sets the tool choice strategy.
func (c ToolConfig) WithToolChoice(choice tools.ToolChoice) ToolConfig {
	c.ToolChoice = choice
	return c
}

// WithAllowedTools sets the list of allowed tools.
func (c ToolConfig) WithAllowedTools(toolNames []string) ToolConfig {
	c.AllowedTools = toolNames
	return c
}

// WithToolErrorHandling sets the tool error handling strategy.
func (c ToolConfig) WithToolErrorHandling(handling tools.ToolErrorHandling) ToolConfig {
	c.ToolErrorHandling = handling
	return c
}
