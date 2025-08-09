package conversation

// ToolDefinition represents a tool description used in inference.
type ToolDefinition struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// InferenceContext carries all information needed for a single inference call.
type InferenceContext struct {
	Messages Conversation     `json:"messages" yaml:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty" yaml:"tools,omitempty"`
	Metadata map[string]any   `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Model       string  `json:"model,omitempty" yaml:"model,omitempty"`
	Temperature float32 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
	UserID      string  `json:"user_id,omitempty" yaml:"user_id,omitempty"`
}

// NewInferenceContext creates a new inference context with the provided messages.
func NewInferenceContext(messages Conversation) InferenceContext {
	return InferenceContext{Messages: messages, Metadata: make(map[string]any)}
}

// AppendMessage adds a message to the context.
func (c *InferenceContext) AppendMessage(m *Message) {
	c.Messages = append(c.Messages, m)
}

// AppendTool adds a tool definition to the context.
func (c *InferenceContext) AppendTool(t ToolDefinition) {
	c.Tools = append(c.Tools, t)
}

// GetSinglePrompt returns a concatenated string of all chat messages.
func (c InferenceContext) GetSinglePrompt() string {
	return c.Messages.GetSinglePrompt()
}

// ToString returns a string representation of the conversation.
func (c InferenceContext) ToString() string {
	return c.Messages.ToString()
}
