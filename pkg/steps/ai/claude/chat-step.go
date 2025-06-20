package claude

import (
	"context"
	"encoding/base64"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	events2 "github.com/go-go-golems/geppetto/pkg/events"
	helpers2 "github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/weaviate/weaviate-go-client/v4/test/helpers"
)

type ChatStep struct {
	Settings            *settings.StepSettings
	Tools               []api.Tool
	subscriptionManager *events2.PublisherManager
	parentID            conversation.NodeID
	messageID           conversation.NodeID
}

type ChatStepOption func(*ChatStep) error

func WithChatStepSubscriptionManager(subscriptionManager *events2.PublisherManager) ChatStepOption {
	return func(step *ChatStep) error {
		step.subscriptionManager = subscriptionManager
		return nil
	}
}

func WithChatStepParentID(parentID conversation.NodeID) ChatStepOption {
	return func(step *ChatStep) error {
		step.parentID = parentID
		return nil
	}
}

func WithChatStepMessageID(messageID conversation.NodeID) ChatStepOption {
	return func(step *ChatStep) error {
		step.messageID = messageID
		return nil
	}
}

func NewChatStep(
	stepSettings *settings.StepSettings,
	tools []api.Tool,
	options ...ChatStepOption,
) (*ChatStep, error) {
	ret := &ChatStep{
		Settings:            stepSettings,
		Tools:               tools,
		subscriptionManager: events2.NewPublisherManager(),
	}

	for _, option := range options {
		err := option(ret)
		if err != nil {
			return nil, err
		}
	}

	if ret.messageID == conversation.NullNode {
		ret.messageID = conversation.NewNodeID()
	}

	return ret, nil
}

var _ chat.Step = &ChatStep{}
var _ chat.SimpleChatStep = &ChatStep{}

func (csf *ChatStep) RunInference(
	ctx context.Context,
	messages conversation.Conversation,
) (*conversation.Message, error) {
	clientSettings := csf.Settings.Client
	if clientSettings == nil {
		return nil, steps.ErrMissingClientSettings
	}
	anthropicSettings := csf.Settings.Claude
	if anthropicSettings == nil {
		return nil, errors.New("no claude settings")
	}

	apiType_ := csf.Settings.Chat.ApiType
	if apiType_ == nil {
		return nil, errors.New("no chat engine specified")
	}
	apiType := *apiType_
	apiSettings := csf.Settings.API

	apiKey, ok := apiSettings.APIKeys[string(apiType)+"-api-key"]
	if !ok {
		return nil, errors.Errorf("no API key for %s", apiType)
	}
	baseURL, ok := apiSettings.BaseUrls[string(apiType)+"-base-url"]
	if !ok {
		return nil, errors.Errorf("no base URL for %s", apiType)
	}

	client := api.NewClient(apiKey, baseURL)

	req, err := makeMessageRequest(csf.Settings, messages)
	if err != nil {
		return nil, err
	}

	req.Tools = csf.Tools
	// Safely handle Temperature and TopP settings with default fallback
	if req.Temperature == nil {
		defaultTemp := float64(1.0)
		req.Temperature = &defaultTemp
	}
	if req.TopP == nil {
		defaultTopP := float64(1.0)
		req.TopP = &defaultTopP
	}

	if !csf.Settings.Chat.Stream {
		response, err := client.SendMessage(ctx, req)
		if err != nil {
			return nil, err
		}
		llmMessageMetadata := &conversation.LLMMessageMetadata{
			Engine:    req.Model,
			MaxTokens: cast.WrapAddr[int](req.MaxTokens),
			Usage: &conversation.Usage{
				InputTokens:  response.Usage.InputTokens,
				OutputTokens: response.Usage.OutputTokens,
			},
		}
		if response.StopReason != "" {
			llmMessageMetadata.StopReason = &response.StopReason
		}
		message := conversation.NewChatMessage(
			conversation.RoleAssistant, response.FullText(),
			conversation.WithLLMMessageMetadata(llmMessageMetadata),
		)
		return message, nil
	}

	// For streaming, we need to collect all events and return the final message
	eventCh, err := client.StreamMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	var parentMessage *conversation.Message
	parentID := conversation.NullNode
	if len(messages) > 0 {
		parentMessage = messages[len(messages)-1]
		if csf.parentID == conversation.NullNode {
			parentID = parentMessage.ID
		} else {
			parentID = csf.parentID
		}
	}

	metadata := events2.EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: parentID,
		LLMMessageMetadata: conversation.LLMMessageMetadata{
			Engine:      req.Model,
			Usage:       nil,
			StopReason:  nil,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   cast.WrapAddr[int](req.MaxTokens),
		},
	}
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "claude-chat",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: csf.Settings.GetMetadata(),
		},
	}

	completionMerger := NewContentBlockMerger(metadata, stepMetadata)

	for event := range eventCh {
		_, err := completionMerger.Add(event)
		if err != nil {
			return nil, err
		}
	}

	response := completionMerger.Response()
	if response == nil {
		return nil, errors.New("no response")
	}

	msg := conversation.NewChatMessage(
		conversation.RoleAssistant, response.FullText(),
		conversation.WithLLMMessageMetadata(&conversation.LLMMessageMetadata{
			Engine: req.Model,
			Usage: &conversation.Usage{
				InputTokens:  response.Usage.InputTokens,
				OutputTokens: response.Usage.OutputTokens,
			},
			StopReason:  &response.StopReason,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   &req.MaxTokens,
		}),
	)

	return msg, nil
}

func (csf *ChatStep) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[*conversation.Message], error) {
	// For non-streaming, use the simplified RunInference method
	if !csf.Settings.Chat.Stream {
		message, err := csf.RunInference(ctx, messages)
		if err != nil {
			return steps.Reject[*conversation.Message](err), nil
		}
		return steps.Resolve(message), nil
	}

	// For streaming, maintain the original complex logic with events
	clientSettings := csf.Settings.Client
	if clientSettings == nil {
		return nil, steps.ErrMissingClientSettings
	}
	anthropicSettings := csf.Settings.Claude
	if anthropicSettings == nil {
		return nil, errors.New("no claude settings")
	}

	apiType_ := csf.Settings.Chat.ApiType
	if apiType_ == nil {
		return steps.Reject[*conversation.Message](errors.New("no chat engine specified")), nil
	}
	apiType := *apiType_
	apiSettings := csf.Settings.API

	apiKey, ok := apiSettings.APIKeys[string(apiType)+"-api-key"]
	if !ok {
		return nil, errors.Errorf("no API key for %s", apiType)
	}
	baseURL, ok := apiSettings.BaseUrls[string(apiType)+"-base-url"]
	if !ok {
		return nil, errors.Errorf("no base URL for %s", apiType)
	}

	client := api.NewClient(apiKey, baseURL)

	var parentMessage *conversation.Message

	if len(messages) > 0 {
		parentMessage = messages[len(messages)-1]
		if csf.parentID == conversation.NullNode {
			csf.parentID = parentMessage.ID
		}
	}

	req, err := makeMessageRequest(csf.Settings, messages)
	if err != nil {
		return steps.Reject[*conversation.Message](err), nil
	}

	req.Tools = csf.Tools
	// Safely handle Temperature and TopP settings with default fallback
	if req.Temperature == nil {
		defaultTemp := float64(1.0)
		req.Temperature = &defaultTemp
	}
	if req.TopP == nil {
		defaultTopP := float64(1.0)
		req.TopP = &defaultTopP
	}

	metadata := events2.EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: csf.parentID,
		LLMMessageMetadata: conversation.LLMMessageMetadata{
			Engine:      req.Model,
			Usage:       nil,
			StopReason:  nil,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   cast.WrapAddr[int](req.MaxTokens),
		},
	}
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "claude-chat",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: csf.Settings.GetMetadata(),
		},
	}

	var cancel context.CancelFunc
	cancellableCtx, cancel := context.WithCancel(ctx)
	eventCh, err := client.StreamMessage(cancellableCtx, req)
	if err != nil {
		cancel()
		return steps.Reject[*conversation.Message](err), nil
	}

	c := make(chan helpers2.Result[*conversation.Message])
	ret := steps.NewStepResult[*conversation.Message](
		c,
		steps.WithCancel[*conversation.Message](cancel),
		steps.WithMetadata[*conversation.Message](stepMetadata),
	)

	// TODO(manuel, 2023-11-28) We need to collect this goroutine in Close(), or at least I think so?
	go func() {
		defer close(c)
		// NOTE(manuel, 2024-06-04) Added this because we now use ctx_ as the chat completion stream context
		defer cancel()

		// we need to accumulate all the blocks that get streamed
		completionMerger := NewContentBlockMerger(metadata, stepMetadata)

		for {
			select {
			case <-cancellableCtx.Done():
				// TODO(manuel, 2024-07-04) Add tool calls so far
				csf.subscriptionManager.PublishBlind(events2.NewInterruptEvent(metadata, stepMetadata, completionMerger.Text()))
				return

			case event, ok := <-eventCh:
				if !ok {
					// TODO(manuel, 2024-07-04) Probably not necessary, the completionMerger probably took care of it
					response := completionMerger.Response()
					if response == nil {
						csf.subscriptionManager.PublishBlind(events2.NewErrorEvent(metadata, stepMetadata, errors.New("no response")))
						c <- helpers2.NewErrorResult[*conversation.Message](errors.New("no response"))
						return
					}

					msg := conversation.NewChatMessage(
						conversation.RoleAssistant, response.FullText(),
						conversation.WithLLMMessageMetadata(&conversation.LLMMessageMetadata{
							Engine: req.Model,
							Usage: &conversation.Usage{
								InputTokens:  response.Usage.InputTokens,
								OutputTokens: response.Usage.OutputTokens,
							},
							StopReason:  &response.StopReason,
							Temperature: req.Temperature,
							TopP:        req.TopP,
							MaxTokens:   &req.MaxTokens,
						}),
					)

					c <- helpers2.NewValueResult[*conversation.Message](msg)
					return
				}

				events_, err := completionMerger.Add(event)
				if err != nil {
					csf.subscriptionManager.PublishBlind(events2.NewErrorEvent(metadata, stepMetadata, err))
					c <- helpers2.NewErrorResult[*conversation.Message](err)
					return
				}
				for _, event_ := range events_ {
					log.Trace().Interface("event", event_).Msg("processing event")
					csf.subscriptionManager.PublishBlind(event_)
				}

			}
		}
	}()

	return ret, nil
}

func (csf *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	csf.subscriptionManager.RegisterPublisher(topic, publisher)
	return nil
}

func messageToClaudeMessage(msg *conversation.Message) api.Message {
	switch content := msg.Content.(type) {
	case *conversation.ChatMessageContent:
		res := api.Message{
			Role: string(content.Role),
			Content: []api.Content{
				api.NewTextContent(content.Text),
			},
		}
		for _, img := range content.Images {
			res.Content = append(res.Content, api.NewImageContent(img.MediaType, base64.StdEncoding.EncodeToString(img.ImageContent)))
		}

		return res

	case *conversation.ToolUseContent:
		// NOTE(manuel, 2024-06-04) I think multi tool calls in the claude API would be represented by multiple content blocks
		res := api.Message{
			Role: string(conversation.RoleUser),
			Content: []api.Content{
				api.NewToolUseContent(content.ToolID, content.Name, string(content.Input)),
			},
		}
		return res

	case *conversation.ToolResultContent:
		res := api.Message{
			Role: string(conversation.RoleTool),
			Content: []api.Content{
				api.NewToolResultContent(content.ToolID, content.Result),
			},
		}
		return res
	}

	return api.Message{}
}

func makeMessageRequest(
	settings *settings.StepSettings,
	messages conversation.Conversation,
) (
	*api.MessageRequest,
	error,
) {
	clientSettings := settings.Client
	if clientSettings == nil {
		return nil, steps.ErrMissingClientSettings
	}
	anthropicSettings := settings.Claude
	if anthropicSettings == nil {
		return nil, errors.New("no claude settings")
	}

	engine := ""

	chatSettings := settings.Chat
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	msgs_ := []api.Message{}
	var systemPrompt string
	for _, msg := range messages {
		chatMessage, ok := msg.Content.(*conversation.ChatMessageContent)
		if ok && chatMessage.Role == conversation.RoleSystem {
			systemPrompt = chatMessage.Text
			continue
		}
		msg_ := messageToClaudeMessage(msg)
		msgs_ = append(msgs_, msg_)
	}

	temperature := 0.0
	if chatSettings.Temperature != nil {
		temperature = *chatSettings.Temperature
	}
	topP := 0.0
	if chatSettings.TopP != nil {
		topP = *chatSettings.TopP
	}
	maxTokens := 1024
	if chatSettings.MaxResponseTokens != nil && *chatSettings.MaxResponseTokens > 0 {
		maxTokens = *chatSettings.MaxResponseTokens
	}

	stream := chatSettings.Stream
	stop := chatSettings.Stop
	ret := &api.MessageRequest{
		Model:         engine,
		Messages:      msgs_,
		MaxTokens:     maxTokens,
		Metadata:      nil,
		StopSequences: stop,
		Stream:        stream,
		System:        systemPrompt,
		Temperature:   helpers.Float64Pointer(temperature),
		Tools:         nil,
		TopK:          nil,
		TopP:          helpers.Float64Pointer(topP),
	}

	return ret, nil
}
