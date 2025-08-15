package events

// Usage represents token usage information common across LLM providers
type Usage struct {
	InputTokens  int `json:"input_tokens" yaml:"input_tokens" mapstructure:"input_tokens"`
	OutputTokens int `json:"output_tokens" yaml:"output_tokens" mapstructure:"output_tokens"`
}

type LLMMessageMetadata struct {
	Engine      string   `json:"engine,omitempty" yaml:"engine,omitempty" mapstructure:"engine,omitempty"`
	Temperature *float64 `json:"temperature,omitempty" yaml:"temperature,omitempty" mapstructure:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty" yaml:"top_p,omitempty" mapstructure:"top_p,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty" mapstructure:"max_tokens,omitempty"`
	StopReason  *string  `json:"stop_reason,omitempty" yaml:"stop_reason,omitempty" mapstructure:"stop_reason,omitempty"`
	Usage       *Usage   `json:"usage,omitempty" yaml:"usage,omitempty" mapstructure:"usage,omitempty"`
}
