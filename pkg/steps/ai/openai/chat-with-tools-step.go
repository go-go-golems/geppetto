package openai

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	go_openai "github.com/sashabaranov/go-openai"
	"io"
)

type ToolCompletionResponse struct {
	Role      string               `json:"role"`
	Content   string               `json:"content"`
	ToolCalls []go_openai.ToolCall `json:"tool_calls"`
}

// ChatWithToolsStep is actually just like ChatStep, except that it also accumulates tool calls.
type ChatWithToolsStep struct {
	Settings            *settings.StepSettings
	Tools               []go_openai.Tool
	subscriptionManager *events.PublisherManager
	parentID            conversation.NodeID
	messageID           conversation.NodeID
}

var _ steps.Step[[]*conversation.Message, ToolCompletionResponse] = (*ChatWithToolsStep)(nil)

type ToolStepOption func(*ChatWithToolsStep) error

func WithChatWithToolsStepSubscriptionManager(subscriptionManager *events.PublisherManager) ToolStepOption {
	return func(step *ChatWithToolsStep) error {
		step.subscriptionManager = subscriptionManager
		return nil
	}
}

func WithChatWithToolsStepParentID(parentID conversation.NodeID) ToolStepOption {
	return func(step *ChatWithToolsStep) error {
		step.parentID = parentID
		return nil
	}
}

func WithChatWithToolsStepMessageID(messageID conversation.NodeID) ToolStepOption {
	return func(step *ChatWithToolsStep) error {
		step.messageID = messageID
		return nil
	}
}

func NewChatWithToolsStep(
	stepSettings *settings.StepSettings,
	Tools []go_openai.Tool,
	options ...ToolStepOption,
) (*ChatWithToolsStep, error) {
	ret := &ChatWithToolsStep{
		Settings:            stepSettings,
		Tools:               Tools,
		subscriptionManager: events.NewPublisherManager(),
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

// NOTE(manuel, 2024-06-04) I think this can be removed
func (csf *ChatWithToolsStep) SetStreaming(b bool) {
	csf.Settings.Chat.Stream = b
}

func (csf *ChatWithToolsStep) Start(
	ctx context.Context,
	messages []*conversation.Message,
) (steps.StepResult[ToolCompletionResponse], error) {
	if csf.Settings.Chat.ApiType == nil {
		return steps.Reject[ToolCompletionResponse](errors.New("no chat engine specified")), nil
	}

	client, err := makeClient(csf.Settings.API, *csf.Settings.Chat.ApiType)
	if err != nil {
		return nil, err
	}

	req, err := makeCompletionRequest(csf.Settings, messages)
	if err != nil {
		return nil, err
	}

	req.Tools = csf.Tools
	stream := csf.Settings.Chat.Stream

	// If we don't have a parent ID, we'll use the last message's ID
	if len(messages) > 0 {
		parentMessage := messages[len(messages)-1]
		if csf.parentID == conversation.NullNode {
			csf.parentID = parentMessage.ID
		}
	}

	metadata := chat.EventMetadata{
		ID:       csf.messageID,
		ParentID: csf.parentID,
	}
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "openai-tool-completion",
		InputType:  "conversation.Conversation",
		OutputType: "ToolCompletionResponse",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: csf.Settings.GetMetadata(),
		},
	}

	csf.subscriptionManager.PublishBlind(&chat.Event{
		Type:     chat.EventTypeStart,
		Step:     stepMetadata,
		Metadata: metadata,
	})

	ctx_, cancel := context.WithCancel(ctx)
	// NOTE(manuel, 2024-06-04) Not sure if we need to collect this goroutine as well when closing the step.
	// Probably (see the other comments for the other goroutines both here and in the claude steps).
	// In fact, do we even need this? Wouldn't the context.WithCancel take care of that anyway?
	// God damn it, context cancellation...
	go func() {
		<-ctx.Done()
		cancel()
	}()

	if stream {
		stream_, err := client.CreateChatCompletionStream(ctx_, *req)
		if err != nil {
			return steps.Reject[ToolCompletionResponse](err), nil
		}
		c := make(chan helpers.Result[ToolCompletionResponse])
		ret := steps.NewStepResult[ToolCompletionResponse](
			c,
			steps.WithCancel[ToolCompletionResponse](cancel),
			steps.WithMetadata[ToolCompletionResponse](stepMetadata),
		)

		// TODO(manuel, 2023-11-28) We need to collect this goroutine in Close(), or at least I think so?
		go func() {
			defer close(c)
			// NOTE(manuel, 2024-06-04) Added this because we now use ctx_ as the chat completion stream context
			defer cancel()
			defer stream_.Close()

			message := ""

			toolCallMerger := NewToolCallMerger()

			ret := ToolCompletionResponse{}

			for {
				select {
				case <-ctx_.Done():
					csf.subscriptionManager.PublishBlind(&chat.EventText{
						Event: chat.Event{
							Type:     chat.EventTypeInterrupt,
							Metadata: metadata,
							Step:     stepMetadata,
						},
						Text: message,
					})
					return
				default:
					response, err := stream_.Recv()
					if errors.Is(err, io.EOF) {
						toolCalls := toolCallMerger.GetToolCalls()
						toolCalls_ := []chat.ToolCall{}
						for _, toolCall := range toolCalls {
							toolCalls_ = append(toolCalls_, chat.ToolCall{
								Name:  toolCall.Function.Name,
								Input: toolCall.Function.Arguments,
							})
						}
						stepMetadata.Metadata[chat.MetadataToolCallsSlug] = toolCalls_

						msg := &chat.EventText{
							Event: chat.Event{
								Type:     chat.EventTypeFinal,
								Metadata: metadata,
								Step:     stepMetadata,
							},
							// TODO(manuel, 2024-06-04) This should be migrated to the proper ToolCall field to be added to chat.Message
							Text: GetToolCallString(toolCalls),
						}

						csf.subscriptionManager.PublishBlind(msg)

						ret.ToolCalls = toolCalls
						ret.Content = message
						c <- helpers.NewValueResult[ToolCompletionResponse](ret)

						return
					}
					if err != nil {
						csf.subscriptionManager.PublishBlind(&chat.Event{
							Type:     chat.EventTypeError,
							Error:    err,
							Metadata: metadata,
							Step:     stepMetadata,
						})
						c <- helpers.NewErrorResult[ToolCompletionResponse](err)
						return
					}

					// NOTE(manuel, 2024-06-04) This could be moved to a proper deltaCompletionMerger like the one sketched out in the claude step implementation
					// TODO(manuel, 2023-11-28) Handle multiple choices
					delta := response.Choices[0].Delta
					deltaContent := delta.Content
					if delta.Content == "" {
						deltaContent = GetToolCallString(delta.ToolCalls)

					}
					toolCallMerger.AddToolCalls(delta.ToolCalls)

					message += deltaContent

					csf.subscriptionManager.PublishBlind(&chat.EventPartialCompletion{
						Event: chat.Event{
							Type:     chat.EventTypePartialCompletion,
							Metadata: metadata,
							Step:     stepMetadata,
						},
						Delta:      deltaContent,
						Completion: message,
					})

					if delta.Role != "" {
						ret.Role = delta.Role
					}
				}
			}
		}()

		return ret, nil
	} else {
		// XXX This should run in a go routine as well
		resp, err := client.CreateChatCompletion(ctx_, *req)

		if errors.Is(err, context.Canceled) {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeInterrupt,
				Metadata: metadata,
				Step:     stepMetadata,
			})
			return steps.Reject[ToolCompletionResponse](err), nil
		}

		if err != nil {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeError,
				Error:    err,
				Metadata: metadata,
				Step:     stepMetadata,
			})
			return steps.Reject[ToolCompletionResponse](err), nil
		}

		// TODO(manuel, 2023-11-28) Handle multiple choices
		s, _ := json.MarshalIndent(resp.Choices[0].Message.ToolCalls, "", " ")

		csf.subscriptionManager.PublishBlind(&chat.EventText{
			Event: chat.Event{
				Type:     chat.EventTypeFinal,
				Metadata: metadata,
				Step:     stepMetadata,
			},
			Text: string(s),
		})

		ret := ToolCompletionResponse{
			Role:      resp.Choices[0].Message.Role,
			Content:   resp.Choices[0].Message.Content,
			ToolCalls: resp.Choices[0].Message.ToolCalls,
		}
		return steps.Resolve(ret), nil
	}
}

func (r *ChatWithToolsStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	r.subscriptionManager.SubscribePublisher(topic, publisher)
	return nil
}
