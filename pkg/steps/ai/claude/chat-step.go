package claude

import (
	"context"
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

func (csf *ChatStep) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[api.MessageResponse], error) {

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
		return steps.Reject[api.MessageResponse](errors.New("no chat engine specified")), nil
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
		return steps.Reject[api.MessageResponse](err), nil
	}

	req.Tools = csf.Tools

	metadata := chat.EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: csf.parentID,
	}
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "claude-messages",
		InputType:  "conversation.Conversation",
		OutputType: "api.MessageResponse",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: csf.Settings.GetMetadata(),
		},
	}

	csf.subscriptionManager.PublishBlind(&chat.Event{
		Type:     chat.EventTypeStart,
		Step:     stepMetadata,
		Metadata: metadata,
	})

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
		return steps.Reject[api.MessageResponse](err), nil
	}

	c := make(chan helpers2.Result[api.MessageResponse])
	ret := steps.NewStepResult[api.MessageResponse](
		c,
		steps.WithCancel[api.MessageResponse](cancel),
		steps.WithMetadata[api.MessageResponse](stepMetadata),
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
				csf.subscriptionManager.PublishBlind(&chat.EventText{
					Event: chat.Event{
						Type:     chat.EventTypeInterrupt,
						Metadata: metadata,
						Step:     stepMetadata,
					},
					Text: completionMerger.Text(),
					// TODO(manuel, 2024-06-04) Add tool calls so far
				})
				return

			case event, ok := <-eventCh:
				if !ok {
					csf.subscriptionManager.PublishBlind(&chat.EventText{
						Event: chat.Event{
							Type:     chat.EventTypeFinal,
							Metadata: metadata,
							Step:     stepMetadata,
						},
						Text: completionMerger.Text(),
						// TODO(manuel, 2024-06-04) Add tool calls so far (once tool calls is added to the EventText / EventPartial
					})

					response := completionMerger.Response()

					c <- helpers2.NewValueResult[api.MessageResponse](*response)
					return
				}

				// this returns a &chat.EventPartialCompletion
				partialEvent, err := completionMerger.Add(event)
				if err != nil {
					csf.subscriptionManager.PublishBlind(&chat.Event{
						Type:     chat.EventTypeError,
						Metadata: metadata,
						Error:    err,
						Step:     stepMetadata,
					})
					c <- helpers2.NewErrorResult[api.MessageResponse](err)
					return
				}

				csf.subscriptionManager.PublishBlind(partialEvent)
			}
		}
	}()

	return ret, nil
}

func (csf *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	csf.subscriptionManager.SubscribePublisher(topic, publisher)
	return nil
}

var _ steps.Step[conversation.Conversation, api.MessageResponse] = (*ChatStep)(nil)

func messageToClaudeMessage(msg *conversation.Message) api.Message {
	switch content := msg.Content.(type) {
	case *conversation.ChatMessageContent:
		res := api.Message{
			Role: string(content.Role),
			Content: []api.Content{
				api.NewTextContent(content.Text),
			},
		}
		return res

	case *conversation.ToolUseContent:
		// NOTE(manuel, 2024-06-04) I think multi tool calls in the claude API would be represented by multiple content blocks
		res := api.Message{
			Role: string(conversation.RoleUser),
			Content: []api.Content{
				api.NewToolUseContent(content.ToolID, content.Name, content.Input),
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
	for _, msg := range messages {
		msgs_ = append(msgs_, messageToClaudeMessage(msg))
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

	stream := chatSettings.Stream
	stop := chatSettings.Stop
	ret := &api.MessageRequest{
		Model:         engine,
		Messages:      msgs_,
		MaxTokens:     maxTokens,
		Metadata:      nil,
		StopSequences: stop,
		Stream:        stream,
		System:        "",
		Temperature:   helpers.Float64Pointer(temperature),
		Tools:         nil,
		TopK:          nil,
		TopP:          helpers.Float64Pointer(topP),
	}

	return ret, nil
}