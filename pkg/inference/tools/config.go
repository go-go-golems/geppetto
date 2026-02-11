package tools

import "time"

// ToolConfig specifies how tools should be used during inference
type ToolConfig struct {
	Enabled           bool              `json:"enabled"`
	ToolChoice        ToolChoice        `json:"tool_choice"`
	MaxIterations     int               `json:"max_iterations"`
	ExecutionTimeout  time.Duration     `json:"execution_timeout"`
	MaxParallelTools  int               `json:"max_parallel_tools"`
	AllowedTools      []string          `json:"allowed_tools"`
	ToolErrorHandling ToolErrorHandling `json:"tool_error_handling"`
	RetryConfig       RetryConfig       `json:"retry_config"`
}

// DefaultToolConfig returns a sensible default configuration
func DefaultToolConfig() ToolConfig {
	return ToolConfig{
		Enabled:           true,
		ToolChoice:        ToolChoiceAuto,
		MaxIterations:     5,
		ExecutionTimeout:  30 * time.Second,
		MaxParallelTools:  3,
		AllowedTools:      nil, // nil means all tools are allowed
		ToolErrorHandling: ToolErrorContinue,
		RetryConfig: RetryConfig{
			MaxRetries:    2,
			BackoffBase:   time.Second,
			BackoffFactor: 2.0,
		},
	}
}

func (tc ToolConfig) WithEnabled(enabled bool) ToolConfig {
	tc.Enabled = enabled
	return tc
}

func (tc ToolConfig) WithToolChoice(choice ToolChoice) ToolConfig {
	tc.ToolChoice = choice
	return tc
}

func (tc ToolConfig) WithMaxIterations(maxIterations int) ToolConfig {
	tc.MaxIterations = maxIterations
	return tc
}

func (tc ToolConfig) WithExecutionTimeout(timeout time.Duration) ToolConfig {
	tc.ExecutionTimeout = timeout
	return tc
}

func (tc ToolConfig) WithMaxParallelTools(maxParallel int) ToolConfig {
	tc.MaxParallelTools = maxParallel
	return tc
}

func (tc ToolConfig) WithAllowedTools(toolNames []string) ToolConfig {
	tc.AllowedTools = toolNames
	return tc
}

func (tc ToolConfig) WithToolErrorHandling(handling ToolErrorHandling) ToolConfig {
	tc.ToolErrorHandling = handling
	return tc
}

func (tc ToolConfig) WithRetryConfig(cfg RetryConfig) ToolConfig {
	tc.RetryConfig = cfg
	return tc
}

// RetryConfig defines retry behavior for tool execution
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	BackoffBase   time.Duration `json:"backoff_base"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// ToolChoice defines how the model should choose tools
type ToolChoice string

const (
	ToolChoiceAuto     ToolChoice = "auto"     // Let the model decide
	ToolChoiceNone     ToolChoice = "none"     // Never call tools
	ToolChoiceRequired ToolChoice = "required" // Must call at least one tool (with iteration limit)
)

// ToolErrorHandling defines how to handle tool execution errors
type ToolErrorHandling string

const (
	ToolErrorContinue ToolErrorHandling = "continue" // Continue conversation with error message
	ToolErrorAbort    ToolErrorHandling = "abort"    // Stop inference on tool error
	ToolErrorRetry    ToolErrorHandling = "retry"    // Retry with exponential backoff
)

// IsToolAllowed checks if a tool is allowed based on the configuration
func (tc *ToolConfig) IsToolAllowed(toolName string) bool {
	if tc.AllowedTools == nil {
		return true // All tools allowed
	}

	for _, allowed := range tc.AllowedTools {
		if allowed == toolName {
			return true
		}
	}

	return false
}

// FilterTools returns only the tools that are allowed by this configuration
func (tc *ToolConfig) FilterTools(tools []ToolDefinition) []ToolDefinition {
	if tc.AllowedTools == nil {
		return tools // All tools allowed
	}

	filtered := make([]ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		if tc.IsToolAllowed(tool.Name) {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}
