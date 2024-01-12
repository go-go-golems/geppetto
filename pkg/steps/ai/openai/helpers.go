package openai

import (
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
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

	if openaiSettings.APIKey == nil {
		return nil, steps.ErrMissingClientAPIKey
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

func makeClient(openaiSettings *openai.Settings) *go_openai.Client {
	config := go_openai.DefaultConfig(*openaiSettings.APIKey)
	if openaiSettings.BaseURL != nil {
		config.BaseURL = *openaiSettings.BaseURL
	}
	client := go_openai.NewClientWithConfig(config)
	return client
}

func messageToOpenAIMessage(msg *conversation.Message) go_openai.ChatCompletionMessage {
	res := go_openai.ChatCompletionMessage{
		Role:    msg.Role,
		Content: msg.Text,
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
