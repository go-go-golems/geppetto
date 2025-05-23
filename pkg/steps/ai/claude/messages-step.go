package claude

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
	csf.subscriptionManager.RegisterPublisher(topic, publisher)
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
) (steps.StepResult[*conversation.Message], error) {
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

	metadata := events.EventMetadata{
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

	client := api.NewClient(apiKey, baseURL+"/v1/complete")

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

	csf.subscriptionManager.PublishBlind(events.NewStartEvent(metadata, stepMetadata))

	if chatSettings.Stream {
		eventsCh, err := client.StreamComplete(&req)
		if err != nil {
			return steps.Reject[*conversation.Message](err), nil
		}
		c := make(chan helpers.Result[*conversation.Message])
		ret := steps.NewStepResult[*conversation.Message](c,
			steps.WithCancel[*conversation.Message](cancel),
			steps.WithMetadata[*conversation.Message](stepMetadata),
		)

		go func() {
			defer close(c)

			isFirstEvent := true
			message := ""
			for {
				select {
				case <-ctx.Done():
					csf.subscriptionManager.PublishBlind(events.NewInterruptEvent(metadata, ret.GetMetadata(), message))
					c <- helpers.NewErrorResult[*conversation.Message](ctx.Err())
					return
				case event, ok := <-eventsCh:
					if !ok {
						csf.subscriptionManager.PublishBlind(events.NewFinalEvent(metadata, ret.GetMetadata(), message))
						msg := conversation.NewChatMessage(conversation.RoleAssistant, message, conversation.WithTime(time.Now()))
						c <- helpers.NewValueResult[*conversation.Message](msg)
						return
					}
					decoded := map[string]interface{}{}
					err = json.Unmarshal([]byte(event.Data), &decoded)
					if err != nil {
						csf.subscriptionManager.PublishBlind(events.NewErrorEvent(metadata, ret.GetMetadata(), err))
						c <- helpers.NewErrorResult[*conversation.Message](err)
						return
					}
					if completion, exists := decoded["completion"].(string); exists {
						if isFirstEvent {
							completion = strings.TrimLeft(completion, " ")
							isFirstEvent = false
						}
						message += completion
						csf.subscriptionManager.PublishBlind(events.NewPartialCompletionEvent(metadata, ret.GetMetadata(), completion, message))
					}
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.Complete(&req)

		if err != nil {
			err = csf.subscriptionManager.Publish(events.NewErrorEvent(metadata, stepMetadata, err))
			if err != nil {
				log.Warn().Err(err).Msg("error publishing error event")
			}
			return steps.Reject[*conversation.Message](err, steps.WithMetadata[*conversation.Message](stepMetadata)), nil
		}

		csf.subscriptionManager.PublishBlind(events.NewFinalEvent(metadata, stepMetadata, resp.Completion))

		msg := conversation.NewChatMessage(conversation.RoleAssistant, resp.Completion, conversation.WithTime(time.Now()))
		return steps.Resolve(msg, steps.WithMetadata[*conversation.Message](stepMetadata)), nil
	}
}
