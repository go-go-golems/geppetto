package engine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/invopop/jsonschema"
)

// ToolDefinition represents a tool that can be called by AI models
type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  *jsonschema.Schema `json:"parameters"`
	Function    interface{}        `json:"-"` // Function to execute
	Examples    []ToolExample      `json:"examples,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	Version     string             `json:"version,omitempty"`
}

// ToolExample represents an example of tool usage
type ToolExample struct {
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output"`
	Description string                 `json:"description"`
}

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

func unmarshalJSONDuration(raw json.RawMessage) (time.Duration, error) {
	if len(raw) == 0 {
		return 0, nil
	}
	if string(raw) == "null" {
		return 0, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, err
		}
		return d, nil
	}

	var i int64
	if err := json.Unmarshal(raw, &i); err == nil {
		return time.Duration(i), nil
	}

	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return time.Duration(f), nil
	}

	return 0, fmt.Errorf("unsupported duration JSON: %s", string(raw))
}

func (c *ToolConfig) UnmarshalJSON(b []byte) error {
	type toolConfigJSON struct {
		Enabled           bool              `json:"enabled"`
		ToolChoice        ToolChoice        `json:"tool_choice"`
		MaxIterations     int               `json:"max_iterations"`
		ExecutionTimeout  json.RawMessage   `json:"execution_timeout"`
		MaxParallelTools  int               `json:"max_parallel_tools"`
		AllowedTools      []string          `json:"allowed_tools"`
		ToolErrorHandling ToolErrorHandling `json:"tool_error_handling"`
		RetryConfig       RetryConfig       `json:"retry_config"`
	}

	var tmp toolConfigJSON
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*c = ToolConfig{
		Enabled:           tmp.Enabled,
		ToolChoice:        tmp.ToolChoice,
		MaxIterations:     tmp.MaxIterations,
		MaxParallelTools:  tmp.MaxParallelTools,
		AllowedTools:      tmp.AllowedTools,
		ToolErrorHandling: tmp.ToolErrorHandling,
		RetryConfig:       tmp.RetryConfig,
	}

	d, err := unmarshalJSONDuration(tmp.ExecutionTimeout)
	if err != nil {
		return fmt.Errorf("execution_timeout: %w", err)
	}
	c.ExecutionTimeout = d
	return nil
}

// RetryConfig defines retry behavior for tool execution
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	BackoffBase   time.Duration `json:"backoff_base"`
	BackoffFactor float64       `json:"backoff_factor"`
}

func (c *RetryConfig) UnmarshalJSON(b []byte) error {
	type retryConfigJSON struct {
		MaxRetries    int             `json:"max_retries"`
		BackoffBase   json.RawMessage `json:"backoff_base"`
		BackoffFactor float64         `json:"backoff_factor"`
	}

	var tmp retryConfigJSON
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*c = RetryConfig{
		MaxRetries:    tmp.MaxRetries,
		BackoffFactor: tmp.BackoffFactor,
	}

	d, err := unmarshalJSONDuration(tmp.BackoffBase)
	if err != nil {
		return fmt.Errorf("backoff_base: %w", err)
	}
	c.BackoffBase = d
	return nil
}

// ToolChoice defines how the model should choose tools
type ToolChoice string

const (
	ToolChoiceAuto     ToolChoice = "auto"     // Let the model decide
	ToolChoiceNone     ToolChoice = "none"     // Never call tools
	ToolChoiceRequired ToolChoice = "required" // Must call at least one tool
)

// ToolErrorHandling defines how to handle tool execution errors
type ToolErrorHandling string

const (
	ToolErrorContinue ToolErrorHandling = "continue" // Continue conversation with error message
	ToolErrorAbort    ToolErrorHandling = "abort"    // Stop inference on tool error
	ToolErrorRetry    ToolErrorHandling = "retry"    // Retry with exponential backoff
)

// NOTE: Streaming types have been removed in the simplified tool calling architecture.
// Streaming is now handled internally by engines when event sinks are configured.
// The RunInferenceStream method has been removed from the Engine interface.

// ToolFeatures describes what tool features an engine supports
type ToolFeatures struct {
	SupportsParallelCalls bool           `json:"supports_parallel_calls"`
	SupportsToolChoice    bool           `json:"supports_tool_choice"`
	SupportsSystemTools   bool           `json:"supports_system_tools"`
	SupportsStreaming     bool           `json:"supports_streaming"`
	Limits                ProviderLimits `json:"limits"`
	SupportedChoiceTypes  []ToolChoice   `json:"supported_choice_types"`
}

// ProviderLimits defines provider-specific limitations
type ProviderLimits struct {
	MaxToolsPerRequest      int      `json:"max_tools_per_request"`
	MaxToolNameLength       int      `json:"max_tool_name_length"`
	MaxTotalSizeBytes       int      `json:"max_total_size_bytes"`
	SupportedParameterTypes []string `json:"supported_parameter_types"`
}

// ToolCall represents a request to execute a tool
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ID       string        `json:"id"`
	Result   interface{}   `json:"result"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
	Retries  int           `json:"retries,omitempty"`
}
