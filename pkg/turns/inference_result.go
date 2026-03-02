package turns

// InferenceFinishClass normalizes provider-specific completion semantics.
type InferenceFinishClass string

const (
	InferenceFinishClassCompleted        InferenceFinishClass = "completed"
	InferenceFinishClassMaxTokens        InferenceFinishClass = "max_tokens"
	InferenceFinishClassToolCallsPending InferenceFinishClass = "tool_calls_pending"
	InferenceFinishClassInterrupted      InferenceFinishClass = "interrupted"
	InferenceFinishClassError            InferenceFinishClass = "error"
	InferenceFinishClassUnknown          InferenceFinishClass = "unknown"
)

// InferenceUsage stores model token usage in a provider-agnostic shape.
type InferenceUsage struct {
	InputTokens              int `json:"input_tokens" yaml:"input_tokens" mapstructure:"input_tokens"`
	OutputTokens             int `json:"output_tokens" yaml:"output_tokens" mapstructure:"output_tokens"`
	CachedTokens             int `json:"cached_tokens,omitempty" yaml:"cached_tokens,omitempty" mapstructure:"cached_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty" yaml:"cache_creation_input_tokens,omitempty" mapstructure:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty" yaml:"cache_read_input_tokens,omitempty" mapstructure:"cache_read_input_tokens,omitempty"`
}

// InferenceResult is the canonical durable inference outcome stored on Turn.Metadata.
type InferenceResult struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty" mapstructure:"provider,omitempty"`
	Model    string `json:"model,omitempty" yaml:"model,omitempty" mapstructure:"model,omitempty"`

	StopReason  string               `json:"stop_reason,omitempty" yaml:"stop_reason,omitempty" mapstructure:"stop_reason,omitempty"`
	FinishClass InferenceFinishClass `json:"finish_class,omitempty" yaml:"finish_class,omitempty" mapstructure:"finish_class,omitempty"`
	Truncated   bool                 `json:"truncated,omitempty" yaml:"truncated,omitempty" mapstructure:"truncated,omitempty"`

	Usage      *InferenceUsage `json:"usage,omitempty" yaml:"usage,omitempty" mapstructure:"usage,omitempty"`
	MaxTokens  *int            `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty" mapstructure:"max_tokens,omitempty"`
	DurationMs *int64          `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty" mapstructure:"duration_ms,omitempty"`

	RequestID  string         `json:"request_id,omitempty" yaml:"request_id,omitempty" mapstructure:"request_id,omitempty"`
	ResponseID string         `json:"response_id,omitempty" yaml:"response_id,omitempty" mapstructure:"response_id,omitempty"`
	Extra      map[string]any `json:"extra,omitempty" yaml:"extra,omitempty" mapstructure:"extra,omitempty"`
}
