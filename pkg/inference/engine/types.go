package engine

import (
	"encoding/json"
	"time"

	"github.com/invopop/jsonschema"
)

// ToolDefinition represents a tool that can be called by AI models
type ToolDefinition struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  *jsonschema.Schema `json:"parameters"`
	Function    interface{}       `json:"-"` // Function to execute
	Examples    []ToolExample     `json:"examples,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Version     string            `json:"version,omitempty"`
}

// ToolExample represents an example of tool usage
type ToolExample struct {
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output"`
	Description string                 `json:"description"`
}

// ToolConfig specifies how tools should be used during inference
type ToolConfig struct {
	Enabled             bool              `json:"enabled"`
	ToolChoice          ToolChoice        `json:"tool_choice"`          
	MaxIterations       int               `json:"max_iterations"`       
	ExecutionTimeout    time.Duration     `json:"execution_timeout"`    
	MaxParallelTools    int               `json:"max_parallel_tools"`   
	AllowedTools        []string          `json:"allowed_tools"`        
	ToolErrorHandling   ToolErrorHandling `json:"tool_error_handling"`  
	RetryConfig         RetryConfig       `json:"retry_config"`         
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
	ToolChoiceRequired ToolChoice = "required" // Must call at least one tool
)

// ToolErrorHandling defines how to handle tool execution errors
type ToolErrorHandling string

const (
	ToolErrorContinue ToolErrorHandling = "continue" // Continue conversation with error message
	ToolErrorAbort    ToolErrorHandling = "abort"    // Stop inference on tool error
	ToolErrorRetry    ToolErrorHandling = "retry"    // Retry with exponential backoff
)

// StreamChunkHandler processes streaming chunks that may include partial tool calls
type StreamChunkHandler func(chunk StreamChunk) error

// StreamChunk represents a piece of streaming response
type StreamChunk struct {
	Type       ChunkType        `json:"type"`
	Content    string           `json:"content,omitempty"`
	ToolCall   *PartialToolCall `json:"tool_call,omitempty"`
	IsComplete bool             `json:"is_complete"`
}

// ChunkType defines the type of streaming chunk
type ChunkType string

const (
	ChunkTypeContent  ChunkType = "content"
	ChunkTypeToolCall ChunkType = "tool_call"
	ChunkTypeComplete ChunkType = "complete"
)

// PartialToolCall represents a partial tool call during streaming
type PartialToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // May be partial JSON
}

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
