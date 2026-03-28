package openai

import "encoding/json"

const (
	chatToolTypeFunction             = "function"
	chatMessagePartTypeText          = "text"
	chatMessagePartTypeImageURL      = "image_url"
	chatImageURLDetailAuto           = "auto"
	chatResponseFormatTypeJSONSchema = "json_schema"
)

type ChatCompletionRequest struct {
	Model               string                        `json:"model"`
	Messages            []ChatCompletionMessage       `json:"messages"`
	MaxTokens           int                           `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                           `json:"max_completion_tokens,omitempty"`
	Temperature         float32                       `json:"temperature,omitempty"`
	TopP                float32                       `json:"top_p,omitempty"`
	N                   int                           `json:"n"`
	Stream              bool                          `json:"stream"`
	Stop                []string                      `json:"stop,omitempty"`
	StreamOptions       *ChatStreamOptions            `json:"stream_options,omitempty"`
	PresencePenalty     float32                       `json:"presence_penalty,omitempty"`
	FrequencyPenalty    float32                       `json:"frequency_penalty,omitempty"`
	LogitBias           map[string]int                `json:"logit_bias,omitempty"`
	Seed                *int                          `json:"seed,omitempty"`
	ResponseFormat      *ChatCompletionResponseFormat `json:"response_format,omitempty"`
	Tools               []ChatCompletionTool          `json:"tools,omitempty"`
	ToolChoice          any                           `json:"tool_choice,omitempty"`
	ParallelToolCalls   *bool                         `json:"parallel_tool_calls,omitempty"`
}

type ChatCompletionMessage struct {
	Role         string
	Content      string
	MultiContent []ChatMessagePart
	ToolCalls    []ChatToolCall
	ToolCallID   string
}

func (m ChatCompletionMessage) MarshalJSON() ([]byte, error) {
	raw := map[string]any{
		"role": m.Role,
	}
	if len(m.MultiContent) > 0 {
		raw["content"] = m.MultiContent
	} else if m.Content != "" {
		raw["content"] = m.Content
	}
	if len(m.ToolCalls) > 0 {
		raw["tool_calls"] = m.ToolCalls
	}
	if m.ToolCallID != "" {
		raw["tool_call_id"] = m.ToolCallID
	}
	return json.Marshal(raw)
}

type ChatMessagePart struct {
	Type     string               `json:"type"`
	Text     string               `json:"text,omitempty"`
	ImageURL *ChatMessageImageURL `json:"image_url,omitempty"`
}

type ChatMessageImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type ChatToolCall struct {
	Index    *int             `json:"index,omitempty"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function ChatFunctionCall `json:"function,omitempty"`
}

type ChatFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type ChatStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type ChatCompletionTool struct {
	Type     string                  `json:"type"`
	Function *ChatFunctionDefinition `json:"function,omitempty"`
}

type ChatFunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type ChatCompletionResponseFormat struct {
	Type       string                                  `json:"type"`
	JSONSchema *ChatCompletionResponseFormatJSONSchema `json:"json_schema,omitempty"`
}

type ChatCompletionResponseFormatJSONSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema"`
	Strict      bool            `json:"strict,omitempty"`
}

func boolRef(v bool) *bool {
	return &v
}
