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
		log.Debug().Msg("Claude WithChatStepSubscriptionManager called")
		step.subscriptionManager = subscriptionManager
		return nil
	}
}

func WithChatStepParentID(parentID conversation.NodeID) ChatStepOption {
	return func(step *ChatStep) error {
		log.Debug().Str("parent_id", parentID.String()).Msg("Claude WithChatStepParentID called")
		step.parentID = parentID
		return nil
	}
}

func WithChatStepMessageID(messageID conversation.NodeID) ChatStepOption {
	return func(step *ChatStep) error {
		log.Debug().Str("message_id", messageID.String()).Msg("Claude WithChatStepMessageID called")
		step.messageID = messageID
		return nil
	}
}

func NewChatStep(
	stepSettings *settings.StepSettings,
	tools []api.Tool,
	options ...ChatStepOption,
) (*ChatStep, error) {
	log.Debug().Int("num_tools", len(tools)).Int("num_options", len(options)).Msg("Claude NewChatStep called")
	
	ret := &ChatStep{
		Settings:            stepSettings,
		Tools:               tools,
		subscriptionManager: events2.NewPublisherManager(),
	}

	for i, option := range options {
		log.Debug().Int("option_index", i).Msg("Claude applying option")
		err := option(ret)
		if err != nil {
			log.Error().Err(err).Int("option_index", i).Msg("Claude option failed")
			return nil, err
		}
	}

	if ret.messageID == conversation.NullNode {
		ret.messageID = conversation.NewNodeID()
		log.Debug().Str("generated_message_id", ret.messageID.String()).Msg("Claude generated new message ID")
	}

	log.Debug().Str("message_id", ret.messageID.String()).Str("parent_id", ret.parentID.String()).Msg("Claude NewChatStep completed")
	return ret, nil
}

var _ chat.Step = &ChatStep{}

func (csf *ChatStep) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[*conversation.Message], error) {
	log.Debug().Int("num_messages", len(messages)).Bool("stream", csf.Settings.Chat.Stream).Msg("Claude Start called")

	clientSettings := csf.Settings.Client
	if clientSettings == nil {
		log.Error().Msg("Claude Start failed: missing client settings")
		return nil, steps.ErrMissingClientSettings
	}
	anthropicSettings := csf.Settings.Claude
	if anthropicSettings == nil {
		log.Error().Msg("Claude Start failed: no claude settings")
		return nil, errors.New("no claude settings")
	}

	apiType_ := csf.Settings.Chat.ApiType
	if apiType_ == nil {
		log.Error().Msg("Claude Start failed: no chat engine specified")
		return steps.Reject[*conversation.Message](errors.New("no chat engine specified")), nil
	}
	apiType := *apiType_
	log.Debug().Str("api_type", string(apiType)).Msg("Claude using API type")
	
	apiSettings := csf.Settings.API

	apiKey, ok := apiSettings.APIKeys[string(apiType)+"-api-key"]
	if !ok {
		log.Error().Str("api_type", string(apiType)).Msg("Claude Start failed: no API key")
		return nil, errors.Errorf("no API key for %s", apiType)
	}
	baseURL, ok := apiSettings.BaseUrls[string(apiType)+"-base-url"]
	if !ok {
		log.Error().Str("api_type", string(apiType)).Msg("Claude Start failed: no base URL")
		return nil, errors.Errorf("no base URL for %s", apiType)
	}

	log.Debug().Str("base_url", baseURL).Msg("Claude creating client")
	client := api.NewClient(apiKey, baseURL)

	var parentMessage *conversation.Message

	if len(messages) > 0 {
		parentMessage = messages[len(messages)-1]
		if csf.parentID == conversation.NullNode {
			csf.parentID = parentMessage.ID
			log.Debug().Str("parent_id", csf.parentID.String()).Msg("Claude set parent ID from last message")
		}
	}

	log.Debug().Msg("Claude creating message request")
	req, err := makeMessageRequest(csf.Settings, messages)
	if err != nil {
		log.Error().Err(err).Msg("Claude Start failed: makeMessageRequest error")
		return steps.Reject[*conversation.Message](err), nil
	}

	req.Tools = csf.Tools
	// Safely handle Temperature and TopP settings with default fallback
	if req.Temperature == nil {
		defaultTemp := float64(1.0)
		req.Temperature = &defaultTemp
		log.Debug().Float64("default_temperature", defaultTemp).Msg("Claude set default temperature")
	}
	if req.TopP == nil {
		defaultTopP := float64(1.0)
		req.TopP = &defaultTopP
		log.Debug().Float64("default_top_p", defaultTopP).Msg("Claude set default top_p")
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

	log.Debug().Str("event_id", metadata.ID.String()).Str("step_id", stepMetadata.StepID.String()).Msg("Claude created metadata")

	if !csf.Settings.Chat.Stream {
		log.Debug().Msg("Claude using non-streaming mode")
		response, err := client.SendMessage(ctx, req)
		if err != nil {
			log.Error().Err(err).Msg("Claude non-streaming request failed")
			return steps.Reject[*conversation.Message](err), nil
		}
		
		log.Debug().Int("input_tokens", response.Usage.InputTokens).Int("output_tokens", response.Usage.OutputTokens).Str("stop_reason", response.StopReason).Msg("Claude non-streaming response received")
		
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
		
		log.Debug().Int("message_length", len(response.FullText())).Msg("Claude non-streaming completed")
		return steps.Resolve(message), nil
	}

	log.Debug().Msg("Claude using streaming mode")
	var cancel context.CancelFunc
	cancellableCtx, cancel := context.WithCancel(ctx)
	
	log.Debug().Msg("Claude starting stream message")
	eventCh, err := client.StreamMessage(cancellableCtx, req)
	if err != nil {
		log.Error().Err(err).Msg("Claude streaming request failed")
		cancel()
		return steps.Reject[*conversation.Message](err), nil
	}

	c := make(chan helpers2.Result[*conversation.Message])
	ret := steps.NewStepResult[*conversation.Message](
		c,
		steps.WithCancel[*conversation.Message](cancel),
		steps.WithMetadata[*conversation.Message](stepMetadata),
	)

	log.Debug().Msg("Claude starting streaming goroutine")
	// TODO(manuel, 2023-11-28) We need to collect this goroutine in Close(), or at least I think so?
	go func() {
		defer func() {
			log.Debug().Msg("Claude streaming goroutine: closing channel")
			close(c)
		}()
		// NOTE(manuel, 2024-06-04) Added this because we now use ctx_ as the chat completion stream context
		defer func() {
			log.Debug().Msg("Claude streaming goroutine: calling cancel")
			cancel()
		}()

		log.Debug().Msg("Claude streaming goroutine started")

		// we need to accumulate all the blocks that get streamed
		completionMerger := NewContentBlockMerger(metadata, stepMetadata)
		log.Debug().Msg("Claude created ContentBlockMerger")

		eventCount := 0
		for {
			select {
			case <-cancellableCtx.Done():
				log.Debug().Int("events_processed", eventCount).Msg("Claude context cancelled, publishing interrupt event")
				// TODO(manuel, 2024-07-04) Add tool calls so far
				csf.subscriptionManager.PublishBlind(events2.NewInterruptEvent(metadata, stepMetadata, completionMerger.Text()))
				return

			case event, ok := <-eventCh:
				if !ok {
					log.Debug().Int("events_processed", eventCount).Msg("Claude event channel closed")
					// TODO(manuel, 2024-07-04) Probably not necessary, the completionMerger probably took care of it
					response := completionMerger.Response()
					if response == nil {
						log.Error().Msg("Claude no response from completionMerger")
						csf.subscriptionManager.PublishBlind(events2.NewErrorEvent(metadata, stepMetadata, errors.New("no response")))
						c <- helpers2.NewErrorResult[*conversation.Message](errors.New("no response"))
						return
					}

					log.Debug().Int("input_tokens", response.Usage.InputTokens).Int("output_tokens", response.Usage.OutputTokens).Str("stop_reason", response.StopReason).Int("message_length", len(response.FullText())).Msg("Claude final response created")

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

					log.Debug().Msg("Claude sending final result to channel")
					c <- helpers2.NewValueResult[*conversation.Message](msg)
					log.Debug().Msg("Claude streaming completed successfully")
					return
				}

				eventCount++
				log.Debug().Int("event_count", eventCount).Interface("event", event).Msg("Claude received streaming event")
				
				events_, err := completionMerger.Add(event)
				if err != nil {
					log.Error().Err(err).Int("event_count", eventCount).Msg("Claude completionMerger.Add failed")
					csf.subscriptionManager.PublishBlind(events2.NewErrorEvent(metadata, stepMetadata, err))
					c <- helpers2.NewErrorResult[*conversation.Message](err)
					return
				}
				
				log.Debug().Int("events_to_publish", len(events_)).Msg("Claude processing events from merger")
				for i, event_ := range events_ {
					log.Debug().Int("event_index", i).Interface("event", event_).Msg("Claude publishing event")
					csf.subscriptionManager.PublishBlind(event_)
				}

			}
		}
	}()

	log.Debug().Msg("Claude Start completed, returning streaming result")
	return ret, nil
}

func (csf *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	log.Debug().Str("topic", topic).Msg("Claude AddPublishedTopic called")
	csf.subscriptionManager.RegisterPublisher(topic, publisher)
	return nil
}

func messageToClaudeMessage(msg *conversation.Message) api.Message {
	log.Debug().Str("message_id", msg.ID.String()).Msg("Claude messageToClaudeMessage called")
	
	switch content := msg.Content.(type) {
	case *conversation.ChatMessageContent:
		log.Debug().Str("role", string(content.Role)).Int("text_length", len(content.Text)).Int("num_images", len(content.Images)).Msg("Claude converting ChatMessageContent")
		res := api.Message{
			Role: string(content.Role),
			Content: []api.Content{
				api.NewTextContent(content.Text),
			},
		}
		for i, img := range content.Images {
			log.Debug().Int("image_index", i).Str("media_type", img.MediaType).Int("image_size", len(img.ImageContent)).Msg("Claude adding image content")
			res.Content = append(res.Content, api.NewImageContent(img.MediaType, base64.StdEncoding.EncodeToString(img.ImageContent)))
		}

		return res

	case *conversation.ToolUseContent:
		// NOTE(manuel, 2024-06-04) I think multi tool calls in the claude API would be represented by multiple content blocks
		log.Debug().Str("tool_id", content.ToolID).Str("tool_name", content.Name).Int("input_length", len(content.Input)).Msg("Claude converting ToolUseContent")
		res := api.Message{
			Role: string(conversation.RoleUser),
			Content: []api.Content{
				api.NewToolUseContent(content.ToolID, content.Name, string(content.Input)),
			},
		}
		return res

	case *conversation.ToolResultContent:
		log.Debug().Str("tool_id", content.ToolID).Int("result_length", len(content.Result)).Msg("Claude converting ToolResultContent")
		res := api.Message{
			Role: string(conversation.RoleTool),
			Content: []api.Content{
				api.NewToolResultContent(content.ToolID, content.Result),
			},
		}
		return res
	}

	log.Debug().Msg("Claude messageToClaudeMessage: unknown content type, returning empty message")
	return api.Message{}
}

func makeMessageRequest(
	settings *settings.StepSettings,
	messages conversation.Conversation,
) (
	*api.MessageRequest,
	error,
) {
	log.Debug().Int("num_messages", len(messages)).Msg("Claude makeMessageRequest called")
	
	clientSettings := settings.Client
	if clientSettings == nil {
		log.Error().Msg("Claude makeMessageRequest failed: missing client settings")
		return nil, steps.ErrMissingClientSettings
	}
	anthropicSettings := settings.Claude
	if anthropicSettings == nil {
		log.Error().Msg("Claude makeMessageRequest failed: no claude settings")
		return nil, errors.New("no claude settings")
	}

	engine := ""

	chatSettings := settings.Chat
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
		log.Debug().Str("engine", engine).Msg("Claude using engine")
	} else {
		log.Error().Msg("Claude makeMessageRequest failed: no engine specified")
		return nil, errors.New("no engine specified")
	}

	msgs_ := []api.Message{}
	var systemPrompt string
	systemMessageCount := 0
	for i, msg := range messages {
		chatMessage, ok := msg.Content.(*conversation.ChatMessageContent)
		if ok && chatMessage.Role == conversation.RoleSystem {
			systemPrompt = chatMessage.Text
			systemMessageCount++
			log.Debug().Int("message_index", i).Int("system_prompt_length", len(systemPrompt)).Msg("Claude found system message")
			continue
		}
		msg_ := messageToClaudeMessage(msg)
		msgs_ = append(msgs_, msg_)
		log.Debug().Int("message_index", i).Int("total_messages", len(msgs_)).Msg("Claude added message to request")
	}

	log.Debug().Int("system_messages", systemMessageCount).Int("regular_messages", len(msgs_)).Msg("Claude processed all messages")

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

	log.Debug().Float64("temperature", temperature).Float64("top_p", topP).Int("max_tokens", maxTokens).Msg("Claude request parameters")

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

	log.Debug().Str("model", engine).Bool("stream", stream).Int("stop_sequences", len(stop)).Int("system_prompt_length", len(systemPrompt)).Msg("Claude makeMessageRequest completed")
	return ret, nil
}
