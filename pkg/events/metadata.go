package events

// Usage represents token usage information common across LLM providers
type Usage struct {
	InputTokens  int `json:"input_tokens" yaml:"input_tokens" mapstructure:"input_tokens"`
	OutputTokens int `json:"output_tokens" yaml:"output_tokens" mapstructure:"output_tokens"`
	// CachedTokens is used by providers like OpenAI to report prompt caching
	CachedTokens int `json:"cached_tokens,omitempty" yaml:"cached_tokens,omitempty" mapstructure:"cached_tokens,omitempty"`
	// CacheCreationInputTokens and CacheReadInputTokens are used by Claude
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty" yaml:"cache_creation_input_tokens,omitempty" mapstructure:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty" yaml:"cache_read_input_tokens,omitempty" mapstructure:"cache_read_input_tokens,omitempty"`
}

// LLMInferenceData consolidates common LLM inference metadata for UI/storage/aggregation.
type LLMInferenceData struct {
	Model       string   `json:"model,omitempty" yaml:"model,omitempty" mapstructure:"model,omitempty"`
	Temperature *float64 `json:"temperature,omitempty" yaml:"temperature,omitempty" mapstructure:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty" yaml:"top_p,omitempty" mapstructure:"top_p,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty" mapstructure:"max_tokens,omitempty"`
	StopReason  *string  `json:"stop_reason,omitempty" yaml:"stop_reason,omitempty" mapstructure:"stop_reason,omitempty"`
	Usage       *Usage   `json:"usage,omitempty" yaml:"usage,omitempty" mapstructure:"usage,omitempty"`
	DurationMs  *int64   `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty" mapstructure:"duration_ms,omitempty"`
}
