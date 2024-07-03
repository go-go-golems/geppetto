package claude

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"strings"
)

type MessagesStep struct {
	Settings            *settings.StepSettings
	cancel              context.CancelFunc
	subscriptionManager *events.PublisherManager
}

func NewStep(settings *settings.StepSettings) *MessagesStep {
	return &MessagesStep{
		Settings:            settings,
		subscriptionManager: events.NewPublisherManager(),
	}
}

func (csf *MessagesStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	csf.subscriptionManager.SubscribePublisher(topic, publisher)
	return nil
}

func (csf *MessagesStep) Interrupt() {
	if csf.cancel != nil {
		csf.cancel()
	}
}

var _ chat.Step = &MessagesStep{}

func IsClaudeEngine(engine string) bool {
	return strings.HasPrefix(engine, "claude")
}

func (csf *MessagesStep) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[string], error) {
	// TODO(manuel, 2024-06-04) I think this can be removed now?
	if csf.cancel != nil {
		return nil, errors.New("step already started")
	}

	var parentMessage *conversation.Message
	parentID := conversation.NullNode

	if len(messages) > 0 {
		parentMessage = messages[len(messages)-1]
		parentID = parentMessage.ID
	}

	metadata := chat.EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: parentID,
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	csf.cancel = cancel

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

	engine := ""

	chatSettings := csf.Settings.Chat
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	// Combine all the messages into a single prompt
	prompt := ""
	for _, msg := range messages {
		switch content := msg.Content.(type) {
		case *conversation.ChatMessageContent:
			rolePrefix := "Human"
			switch content.Role {
			case conversation.RoleSystem:
				rolePrefix = "System"
			case conversation.RoleAssistant:
				rolePrefix = "Assistant"
			case conversation.RoleUser:
				rolePrefix = "Human"
			case conversation.RoleTool:
				rolePrefix = "Tool"
			}
			prompt += "\n\n" + rolePrefix + ": " + content.Text
		}
	}
	prompt += "\n\nAssistant: "

	maxTokens := 32
	if chatSettings.MaxResponseTokens != nil {
		maxTokens = *chatSettings.MaxResponseTokens
	}

	temperature := 0.0
	if chatSettings.Temperature != nil {
		temperature = *chatSettings.Temperature
	}
	topP := 0.0
	if chatSettings.TopP != nil {
		topP = *chatSettings.TopP
	}
	stopSequences := []string{}
	stopSequences = append(stopSequences, chatSettings.Stop...)

	req := api.Request{
		Model:             engine,
		Prompt:            prompt,
		MaxTokensToSample: maxTokens,
		StopSequences:     stopSequences,
		Temperature:       &temperature,
		TopP:              &topP,
		TopK:              nil,
		Metadata:          nil,
		Stream:            chatSettings.Stream,
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

	csf.subscriptionManager.PublishBlind(&chat.Event{
		Type:     chat.EventTypeStart,
		Metadata: metadata,
		Step:     stepMetadata,
	})

	if chatSettings.Stream {
		events, err := client.StreamComplete(&req)
		if err != nil {
			return steps.Reject[string](err), nil
		}
		c := make(chan helpers.Result[string])
		ret := steps.NewStepResult[string](c,
			steps.WithCancel[string](cancel),
			steps.WithMetadata[string](stepMetadata),
		)

		go func() {
			defer close(c)

			isFirstEvent := true
			message := ""
			for {
				select {
				case <-ctx.Done():
					csf.subscriptionManager.PublishBlind(&chat.EventText{
						Event: chat.Event{
							Type:     chat.EventTypeInterrupt,
							Metadata: metadata,
							Step:     ret.GetMetadata(),
						},
						Text: message,
					})
					c <- helpers.NewErrorResult[string](ctx.Err())
					return
				case event, ok := <-events:
					if !ok {
						csf.subscriptionManager.PublishBlind(&chat.EventText{
							Event: chat.Event{
								Type:     chat.EventTypeFinal,
								Metadata: metadata,
								Step:     ret.GetMetadata(),
							},
							Text: message,
						})
						c <- helpers.NewValueResult[string](message)
						return
					}
					decoded := map[string]interface{}{}
					err = json.Unmarshal([]byte(event.Data), &decoded)
					if err != nil {
						csf.subscriptionManager.PublishBlind(&chat.Event{
							Type:     chat.EventTypeError,
							Metadata: metadata,
							Error:    err,
							Step:     ret.GetMetadata(),
						})
						c <- helpers.NewErrorResult[string](err)
						return
					}
					if completion, exists := decoded["completion"].(string); exists {
						if isFirstEvent {
							completion = strings.TrimLeft(completion, " ")
							isFirstEvent = false
						}
						message += completion
						csf.subscriptionManager.PublishBlind(&chat.EventPartialCompletion{
							Event: chat.Event{
								Type:     chat.EventTypePartial,
								Metadata: metadata,
								Step:     ret.GetMetadata(),
							},
							Delta:      completion,
							Completion: message,
						})
					}
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.Complete(&req)

		if err != nil {
			err = csf.subscriptionManager.Publish(&chat.Event{
				Type:  chat.EventTypeError,
				Error: err,
				Step:  stepMetadata,
			})
			if err != nil {
				log.Warn().Err(err).Msg("error publishing error event")
			}
			return steps.Reject[string](err, steps.WithMetadata[string](stepMetadata)), nil
		}

		csf.subscriptionManager.PublishBlind(&chat.EventText{
			Event: chat.Event{
				Type:     chat.EventTypeFinal,
				Metadata: metadata,
				Step:     stepMetadata,
			},
			Text: resp.Completion,
		})

		return steps.Resolve(resp.Completion, steps.WithMetadata[string](stepMetadata)), nil
	}
}
