package claude

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"strings"
)

type Step struct {
	Settings            *settings.StepSettings
	cancel              context.CancelFunc
	subscriptionManager *helpers.SubscriptionManager
}

func NewStep(settings *settings.StepSettings) *Step {
	return &Step{
		Settings:            settings,
		subscriptionManager: helpers.NewSubscriptionManager(),
	}
}

func (csf *Step) AddPublishedTopic(publisher message.Publisher, topic string) error {
	csf.subscriptionManager.AddPublishedTopic(topic, publisher)
	return nil
}

func (csf *Step) Interrupt() {
	if csf.cancel != nil {
		csf.cancel()
	}
}

var _ steps.Step[[]*conversation.Message, string] = &Step{}

func IsClaudeEngine(engine string) bool {
	return strings.HasPrefix(engine, "claude")
}

func (csf *Step) Start(
	ctx context.Context,
	messages []*conversation.Message,
) (steps.StepResult[string], error) {
	if csf.cancel != nil {
		return nil, errors.New("step already started")
	}

	var parentMessage *conversation.Message
	parentID := uuid.Nil
	conversationID := uuid.New()

	if len(messages) > 0 {
		parentMessage = messages[len(messages)-1]
		parentID = parentMessage.ID
		conversationID = parentMessage.ConversationID
	}

	metadata := chat.EventMetadata{
		ID:             uuid.New(),
		ParentID:       parentID,
		ConversationID: conversationID,
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

	if anthropicSettings.APIKey == nil {
		return nil, steps.ErrMissingClientAPIKey
	}

	client := NewClient(*anthropicSettings.APIKey)

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
		rolePrefix := "Human"
		switch msg.Role {
		case conversation.RoleSystem:
			rolePrefix = "System"
		case conversation.RoleAssistant:
			rolePrefix = "Assistant"
		case conversation.RoleUser:
			rolePrefix = "Human"
		}
		prompt += "\n\n" + rolePrefix + ": " + msg.Text
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

	req := Request{
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

	if chatSettings.Stream {
		events, err := client.StreamComplete(&req)
		if err != nil {
			return steps.Reject[string](err), nil
		}
		c := make(chan helpers.Result[string])
		ret := steps.NewStepResult[string](c)

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
			})
			if err != nil {
				log.Warn().Err(err).Msg("error publishing error event")
			}
			return steps.Reject[string](err), nil
		}

		csf.subscriptionManager.PublishBlind(&chat.EventText{
			Event: chat.Event{
				Type:     chat.EventTypeFinal,
				Metadata: metadata,
			},
			Text: resp.Completion,
		})

		return steps.Resolve(resp.Completion), nil
	}
}
