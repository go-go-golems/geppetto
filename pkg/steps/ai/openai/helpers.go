package openai

import (
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	go_openai "github.com/sashabaranov/go-openai"
	"strings"
)

func GetToolCallDelta(toolCalls []go_openai.ToolCall) string {
	msg := ""
	for _, call := range toolCalls {
		msg += call.Function.Name
		msg += call.Function.Arguments
	}

	return msg
}

type ToolCallMerger struct {
	toolCalls map[int]go_openai.ToolCall
}

func NewToolCallMerger() *ToolCallMerger {
	return &ToolCallMerger{
		toolCalls: make(map[int]go_openai.ToolCall),
	}
}

func (tcm *ToolCallMerger) AddToolCalls(toolCalls []go_openai.ToolCall) {
	for _, call := range toolCalls {
		index := 0
		if call.Index != nil {
			index = *call.Index
		}
		if existing, found := tcm.toolCalls[index]; found {
			existing.Function.Name += call.Function.Name
			existing.Function.Arguments += call.Function.Arguments
			tcm.toolCalls[index] = existing
		} else {
			tcm.toolCalls[index] = call
		}
	}
}

func (tcm *ToolCallMerger) GetToolCalls() []go_openai.ToolCall {
	var result []go_openai.ToolCall
	for _, call := range tcm.toolCalls {
		result = append(result, call)
	}

	return result
}

func IsOpenAiEngine(engine string) bool {
	if strings.HasPrefix(engine, "gpt") {
		return true
	}
	if strings.HasPrefix(engine, "text-") {
		return true
	}

	return false
}

func makeCompletionRequest(
	settings *settings.StepSettings,
	messages []*conversation.Message,
) (*go_openai.ChatCompletionRequest, error) {
	clientSettings := settings.Client
	if clientSettings == nil {
		return nil, steps.ErrMissingClientSettings
	}
	openaiSettings := settings.OpenAI
	if openaiSettings == nil {
		return nil, errors.New("no openai settings")
	}

	engine := ""

	chatSettings := settings.Chat
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	msgs_ := []go_openai.ChatCompletionMessage{}
	for _, msg := range messages {
		msgs_ = append(msgs_, messageToOpenAIMessage(msg))
	}

	temperature := 0.0
	if chatSettings.Temperature != nil {
		temperature = *chatSettings.Temperature
	}
	topP := 0.0
	if chatSettings.TopP != nil {
		topP = *chatSettings.TopP
	}
	maxTokens := 32
	if chatSettings.MaxResponseTokens != nil {
		maxTokens = *chatSettings.MaxResponseTokens
	}

	n := 1
	if openaiSettings.N != nil {
		n = *openaiSettings.N
	}
	stream := chatSettings.Stream
	stop := chatSettings.Stop
	presencePenalty := 0.0
	if openaiSettings.PresencePenalty != nil {
		presencePenalty = *openaiSettings.PresencePenalty
	}
	frequencyPenalty := 0.0
	if openaiSettings.FrequencyPenalty != nil {
		frequencyPenalty = *openaiSettings.FrequencyPenalty
	}

	req := go_openai.ChatCompletionRequest{
		Model:            engine,
		Messages:         msgs_,
		MaxTokens:        maxTokens,
		Temperature:      float32(temperature),
		TopP:             float32(topP),
		N:                n,
		Stream:           stream,
		Stop:             stop,
		PresencePenalty:  float32(presencePenalty),
		FrequencyPenalty: float32(frequencyPenalty),
		// TODO(manuel, 2023-03-28) Properly load logit bias
		// See https://github.com/go-go-golems/geppetto/issues/48
		LogitBias: nil,
	}
	return &req, nil
}

func makeClient(apiSettings *settings.APISettings, apiType settings.ApiType) (*go_openai.Client, error) {
	apiKey, ok := apiSettings.APIKeys[apiType+"-api-key"]
	if !ok {
		return nil, errors.Errorf("no API key for %s", apiType)
	}
	baseURL, ok := apiSettings.BaseUrls[apiType+"-base-url"]
	if !ok {
		return nil, errors.Errorf("no base URL for %s", apiType)
	}
	config := go_openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	client := go_openai.NewClientWithConfig(config)
	return client, nil
}

func messageToOpenAIMessage(msg *conversation.Message) go_openai.ChatCompletionMessage {
	// TODO(manuel, 2024-01-13) This is where we could have a proper tool call chat content
	switch content := msg.Content.(type) {
	case *conversation.ChatMessageContent:
		res := go_openai.ChatCompletionMessage{
			Role:    string(content.Role),
			Content: content.Text,
		}
		metadata := msg.Metadata
		if metadata != nil {
			functionCall := metadata["function_call"]
			if functionCall_, ok := functionCall.(*go_openai.FunctionCall); ok {
				res.FunctionCall = functionCall_
			}

			toolCalls := metadata["tool_calls"]
			if toolCalls_, ok := toolCalls.([]go_openai.ToolCall); ok {
				res.ToolCalls = toolCalls_
			}

			toolCallID := metadata["tool_call_id"]
			if toolCallID_, ok := toolCallID.(string); ok {
				res.ToolCallID = toolCallID_
			}
		}
		return res
	}

	return go_openai.ChatCompletionMessage{}
}
