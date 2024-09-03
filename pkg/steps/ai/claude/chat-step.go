package claude

import (
	"context"
	"encoding/base64"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	events2 "github.com/go-go-golems/geppetto/pkg/events"
	helpers2 "github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
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

func (csf *ChatStep) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[string], error) {

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
		return steps.Reject[string](errors.New("no chat engine specified")), nil
	}
	apiType := *apiType_
	apiSettings := csf.Settings.API

	apiKey, ok := apiSettings.APIKeys[apiType+"-api-key"]
	if !ok {
		return nil, errors.Errorf("no API key for %s", apiType)
	}
	baseURL, ok := apiSettings.BaseUrls[apiType+"-base-url"]
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
		return steps.Reject[string](err), nil
	}

	req.Tools = csf.Tools

	metadata := chat.EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: csf.parentID,
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

	//startEvent := chat.NewStartEvent(metadata, stepMetadata)
	//log.Debug().Interface("event", startEvent).Msg("Start chat step")
	//csf.subscriptionManager.PublishBlind(startEvent)

	var cancel context.CancelFunc
	cancellableCtx, cancel := context.WithCancel(ctx)

	// NOTE(manuel, 2024-06-04) Not sure if we need to collect this goroutine as well when closing the step.
	// Probably (see the other comments for the other goroutines both here and in the openai steps).
	// IN fact, do we even need this? Wouldn't the context.WithCancel take care of that anyway?
	// God damn it, context cancellation...
	go func() {
		<-ctx.Done()
		cancel()
	}()

	eventCh, err := client.StreamMessage(cancellableCtx, req)
	if err != nil {
		return steps.Reject[string](err), nil
	}

	c := make(chan helpers2.Result[string])
	ret := steps.NewStepResult[string](
		c,
		steps.WithCancel[string](cancel),
		steps.WithMetadata[string](stepMetadata),
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
				csf.subscriptionManager.PublishBlind(chat.NewInterruptEvent(metadata, stepMetadata, completionMerger.Text()))
				return

			case event, ok := <-eventCh:
				if !ok {
					// TODO(manuel, 2024-07-04) Probably not necessary, the completionMerger probably took care of it
					response := completionMerger.Response()
					if response == nil {
						csf.subscriptionManager.PublishBlind(chat.NewErrorEvent(metadata, stepMetadata, "no response"))
						c <- helpers2.NewErrorResult[string](errors.New("no response"))
						return
					}
					c <- helpers2.NewValueResult[string](response.FullText())
					return
				}

				events_, err := completionMerger.Add(event)
				if err != nil {
					csf.subscriptionManager.PublishBlind(chat.NewErrorEvent(metadata, stepMetadata, err.Error()))
					c <- helpers2.NewErrorResult[string](err)
					return
				}
				for _, event_ := range events_ {
					log.Debug().Interface("event", event_).Msg("processing event")
				}

				if err != nil {
					csf.subscriptionManager.PublishBlind(chat.NewErrorEvent(metadata, stepMetadata, err.Error()))
					c <- helpers2.NewErrorResult[string](err)
					return
				}

				for _, event_ := range events_ {
					csf.subscriptionManager.PublishBlind(event_)
				}
			}
		}
	}()

	return ret, nil
}

func (csf *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	csf.subscriptionManager.SubscribePublisher(topic, publisher)
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
