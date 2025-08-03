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
	log.Debug().Int("num_messages", len(messages)).Bool("stream", csf.Settings.Chat.Stream).Msg("Claude RunInference started")
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

	req, err := MakeMessageRequest(csf.Settings, messages)
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

	// Setup metadata and event publishing
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

	if !csf.Settings.Chat.Stream {
		log.Debug().Msg("Claude using non-streaming mode")
		response, err := client.SendMessage(ctx, req)
		if err != nil {
			log.Error().Err(err).Msg("Claude non-streaming request failed")
			if csf.subscriptionManager != nil {
				csf.subscriptionManager.PublishBlind(events2.NewErrorEvent(metadata, stepMetadata, err))
			}
			return nil, err
		}

		// Update metadata with response data
		metadata.Usage = &conversation.Usage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
		}
		if response.StopReason != "" {
			metadata.StopReason = &response.StopReason
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

		// Publish final event
		if csf.subscriptionManager != nil {
			log.Debug().Str("event_id", metadata.ID.String()).Msg("Claude publishing final event (non-streaming)")
			csf.subscriptionManager.PublishBlind(events2.NewFinalEvent(metadata, stepMetadata, response.FullText()))
		}

		log.Debug().Msg("Claude RunInference completed (non-streaming)")
		return message, nil
	}

	// For streaming, we need to collect all events and return the final message
	log.Debug().Msg("Claude starting streaming mode")
	eventCh, err := client.StreamMessage(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Claude streaming request failed")
		if csf.subscriptionManager != nil {
			csf.subscriptionManager.PublishBlind(events2.NewErrorEvent(metadata, stepMetadata, err))
		}
		return nil, err
	}

	log.Debug().Msg("Claude creating ContentBlockMerger")
	completionMerger := NewContentBlockMerger(metadata, stepMetadata)

	log.Debug().Msg("Claude starting streaming event loop")
	eventCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Claude streaming cancelled by context")
			// Publish interrupt event with current partial text
			if csf.subscriptionManager != nil {
				csf.subscriptionManager.PublishBlind(events2.NewInterruptEvent(metadata, stepMetadata, completionMerger.Text()))
			}
			return nil, ctx.Err()

		case event, ok := <-eventCh:
			if !ok {
				log.Debug().Int("total_events", eventCount).Msg("Claude streaming channel closed, loop completed")
				goto streamingComplete
			}

			eventCount++
			log.Debug().Int("event_count", eventCount).Interface("event", event).Msg("Claude processing streaming event")

			events_, err := completionMerger.Add(event)
			if err != nil {
				log.Error().Err(err).Int("event_count", eventCount).Msg("Claude ContentBlockMerger.Add failed")
				if csf.subscriptionManager != nil {
					csf.subscriptionManager.PublishBlind(events2.NewErrorEvent(metadata, stepMetadata, err))
				}
				return nil, err
			}
			// Publish intermediate events generated by the ContentBlockMerger
			if csf.subscriptionManager != nil {
				log.Debug().Int("num_events", len(events_)).Msg("Claude publishing intermediate events")
				for _, event_ := range events_ {
					csf.subscriptionManager.PublishBlind(event_)
				}
			}
		}
	}

streamingComplete:

	log.Debug().Msg("Claude getting final response from ContentBlockMerger")
	response := completionMerger.Response()
	if response == nil {
		err := errors.New("no response")
		log.Error().Err(err).Msg("Claude ContentBlockMerger returned nil response")
		if csf.subscriptionManager != nil {
			csf.subscriptionManager.PublishBlind(events2.NewErrorEvent(metadata, stepMetadata, err))
		}
		return nil, err
	}

	log.Debug().Str("response_text", response.FullText()).Msg("Claude creating final message")
	// Update metadata with final response data
	metadata.Usage = &conversation.Usage{
		InputTokens:  response.Usage.InputTokens,
		OutputTokens: response.Usage.OutputTokens,
	}
	if response.StopReason != "" {
		metadata.StopReason = &response.StopReason
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

	// NOTE: Final event is already published by ContentBlockMerger during event processing
	// Do not publish duplicate final event here
	log.Debug().Msg("Claude RunInference completed (streaming)")
	return msg, nil
}

func (csf *ChatStep) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[*conversation.Message], error) {
	log.Debug().Bool("stream", csf.Settings.Chat.Stream).Msg("Claude Start called")
	// For non-streaming, use the simplified RunInference method
	if !csf.Settings.Chat.Stream {
		log.Debug().Msg("Claude Start using non-streaming path")
		message, err := csf.RunInference(ctx, messages)
		if err != nil {
			return steps.Reject[*conversation.Message](err), nil
		}
		return steps.Resolve(message), nil
	}

	// For streaming, use RunInference in a goroutine to handle cancellation
	log.Debug().Msg("Claude Start using streaming path with goroutine")
	var cancel context.CancelFunc
	cancellableCtx, cancel := context.WithCancel(ctx)

	c := make(chan helpers2.Result[*conversation.Message])
	ret := steps.NewStepResult[*conversation.Message](
		c,
		steps.WithCancel[*conversation.Message](cancel),
		steps.WithMetadata[*conversation.Message](&steps.StepMetadata{
			StepID:     uuid.New(),
			Type:       "claude-chat",
			InputType:  "conversation.Conversation",
			OutputType: "*conversation.Message",
			Metadata: map[string]interface{}{
				steps.MetadataSettingsSlug: csf.Settings.GetMetadata(),
			},
		}),
	)

	go func() {
		defer close(c)
		defer cancel()
		log.Debug().Msg("Claude streaming goroutine started")

		// Check for cancellation before starting
		select {
		case <-cancellableCtx.Done():
			log.Debug().Msg("Claude context cancelled before starting")
			c <- helpers2.NewErrorResult[*conversation.Message](context.Canceled)
			return
		default:
		}

		// Use RunInference which now handles all the ContentBlockMerger logic
		log.Debug().Msg("Claude calling RunInference from goroutine")
		message, err := csf.RunInference(cancellableCtx, messages)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// Context was cancelled during RunInference
				log.Debug().Msg("Claude RunInference cancelled")
				c <- helpers2.NewErrorResult[*conversation.Message](context.Canceled)
			} else {
				log.Error().Err(err).Msg("Claude RunInference failed")
				c <- helpers2.NewErrorResult[*conversation.Message](err)
			}
			return
		}
		log.Debug().Msg("Claude RunInference succeeded, sending result")
		result := helpers2.NewValueResult[*conversation.Message](message)
		log.Debug().Msg("Claude about to send result to channel")
		c <- result
		log.Debug().Msg("Claude result sent to channel successfully")
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

func MakeMessageRequest(
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
		Temperature:   cast.WrapAddr[float64](temperature),
		Tools:         nil,
		TopK:          nil,
		TopP:          cast.WrapAddr[float64](topP),
	}

	return ret, nil
}
