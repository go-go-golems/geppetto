package openai

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	go_openai "github.com/sashabaranov/go-openai"
)

func GetToolCallString(toolCalls []go_openai.ToolCall) string {
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

	log.Debug().
		Str("model", engine).
		Int("max_tokens", maxTokens).
		Float64("temperature", temperature).
		Float64("top_p", topP).
		Int("n", n).
		Bool("stream", stream).
		Strs("stop", stop).
		Float64("presence_penalty", presencePenalty).
		Float64("frequency_penalty", frequencyPenalty).
		Msg("Making request to openai")

	var streamOptions *go_openai.StreamOptions
	if stream && !strings.Contains(engine, "mistral") {
		streamOptions = &go_openai.StreamOptions{IncludeUsage: true}
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
		StreamOptions:    streamOptions,
		PresencePenalty:  float32(presencePenalty),
		FrequencyPenalty: float32(frequencyPenalty),
		// TODO(manuel, 2023-03-28) Properly load logit bias
		// See https://github.com/go-go-golems/geppetto/issues/48
		LogitBias: nil,
	}
	return &req, nil
}

func makeClient(apiSettings *settings.APISettings, apiType ai_types.ApiType) (*go_openai.Client, error) {
	apiKey, ok := apiSettings.APIKeys[string(apiType)+"-api-key"]
	if !ok {
		return nil, errors.Errorf("no API key for %s", apiType)
	}
	baseURL, ok := apiSettings.BaseUrls[string(apiType)+"-base-url"]
	if !ok {
		return nil, errors.Errorf("no base URL for %s", apiType)
	}
	config := go_openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	client := go_openai.NewClientWithConfig(config)
	return client, nil
}

// TODO(manuel, 2024-06-25) We actually need a processor like the content block merger here, where we merge tool use blocks with previous messages to properly create openai messages
// So we would need to get a message block, then slurp up all the following tool use messages, and then finally emit the message block
func messageToOpenAIMessage(msg *conversation.Message) go_openai.ChatCompletionMessage {
	switch content := msg.Content.(type) {
	case *conversation.ChatMessageContent:
		res := go_openai.ChatCompletionMessage{
			Role:    string(content.Role),
			Content: content.Text,
		}

		if len(content.Images) > 0 {
			res = go_openai.ChatCompletionMessage{
				Role: string(content.Role),
				MultiContent: []go_openai.ChatMessagePart{
					{
						Type: go_openai.ChatMessagePartTypeText,
						Text: content.Text,
					},
				},
			}

			for _, img := range content.Images {
				imagePart := go_openai.ChatMessagePart{
					Type: go_openai.ChatMessagePartTypeImageURL,
					ImageURL: &go_openai.ChatMessageImageURL{
						URL:    img.ImageURL,
						Detail: go_openai.ImageURLDetail(img.Detail),
					},
				}
				if img.ImageURL == "" {
					// base64 encoded Content
					imagePart.ImageURL.URL =
						fmt.Sprintf(
							"data:%s;base64,%s",
							img.MediaType,
							base64.StdEncoding.EncodeToString(img.ImageContent),
						)
				}
				res.MultiContent = append(res.MultiContent, imagePart)
			}
		}

		// TODO(manuel, 2024-06-04) This should actually pass in a ToolUse content in the conversation
		// This is how claude expects it, and I also added a comment to ContentType's definition in message.go
		//
		// NOTE(manuel, 2024-06-04) It seems that these metadata keys are never set anywhere anyway
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

	case *conversation.ToolResultContent:
		res := go_openai.ChatCompletionMessage{
			Role:       string(conversation.RoleTool),
			Content:    content.Result,
			ToolCallID: content.ToolID,
		}

		return res

	case *conversation.ToolUseContent:
		// openai encodes tool use messages within the assistant completion message
		// TODO(manuel, 2024-06-25) This should be aggregated into a multi call chat message content, instead of being serialized individually
		res := go_openai.ChatCompletionMessage{
			Role: string(conversation.RoleAssistant),
			ToolCalls: []go_openai.ToolCall{
				{
					ID:   content.ToolID,
					Type: "function",
					Function: go_openai.FunctionCall{
						Name:      content.Name,
						Arguments: string(content.Input),
					},
				},
			},
		}

		return res
	}

	return go_openai.ChatCompletionMessage{}
}
