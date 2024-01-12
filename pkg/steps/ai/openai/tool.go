package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	go_openai "github.com/sashabaranov/go-openai"
	"io"
)

type ToolCompletionResponse struct {
	Role      string               `json:"role"`
	Content   string               `json:"content"`
	ToolCalls []go_openai.ToolCall `json:"tool_calls"`
}

type ToolStep struct {
	Settings            *settings.StepSettings
	Tools               []go_openai.Tool
	subscriptionManager *helpers.SubscriptionManager
	parentID            uuid.UUID
	conversationID      uuid.UUID
	messageID           uuid.UUID
}

var _ steps.Step[[]*conversation.Message, ToolCompletionResponse] = (*ToolStep)(nil)

type ToolStepOption func(*ToolStep) error

func WithToolStepSubscriptionManager(subscriptionManager *helpers.SubscriptionManager) ToolStepOption {
	return func(step *ToolStep) error {
		step.subscriptionManager = subscriptionManager
		return nil
	}
}

func WithToolStepParentID(parentID uuid.UUID) ToolStepOption {
	return func(step *ToolStep) error {
		step.parentID = parentID
		return nil
	}
}

func WithToolStepConversationID(conversationID uuid.UUID) ToolStepOption {
	return func(step *ToolStep) error {
		step.conversationID = conversationID
		return nil
	}
}

func WithToolStepMessageID(messageID uuid.UUID) ToolStepOption {
	return func(step *ToolStep) error {
		step.messageID = messageID
		return nil
	}
}

func NewToolStep(
	stepSettings *settings.StepSettings,
	Tools []go_openai.Tool,
	options ...ToolStepOption,
) (*ToolStep, error) {
	ret := &ToolStep{
		Settings:            stepSettings,
		Tools:               Tools,
		subscriptionManager: helpers.NewSubscriptionManager(),
		parentID:            uuid.Nil,
		conversationID:      uuid.Nil,
		messageID:           uuid.Nil,
	}

	for _, option := range options {
		err := option(ret)
		if err != nil {
			return nil, err
		}
	}

	if ret.messageID == uuid.Nil {
		ret.messageID = uuid.New()
	}

	return ret, nil
}

func (csf *ToolStep) SetStreaming(b bool) {
	csf.Settings.Chat.Stream = b
}

func (csf *ToolStep) Start(
	ctx context.Context,
	messages []*conversation.Message,
) (steps.StepResult[ToolCompletionResponse], error) {
	client := makeClient(csf.Settings.OpenAI)

	req, err := makeCompletionRequest(csf.Settings, messages)
	if err != nil {
		return nil, err
	}

	req.Tools = csf.Tools
	stream := csf.Settings.Chat.Stream

	if len(messages) > 0 {
		parentMessage := messages[len(messages)-1]
		if csf.parentID == uuid.Nil {
			csf.parentID = parentMessage.ID
		}
		if csf.conversationID == uuid.Nil {
			csf.conversationID = parentMessage.ConversationID
		}
	}

	metadata := chat.EventMetadata{
		ID:             csf.messageID,
		ParentID:       csf.parentID,
		ConversationID: csf.conversationID,
	}

	csf.subscriptionManager.PublishBlind(&chat.Event{
		Type:     chat.EventTypeStart,
		Metadata: metadata,
	})

	if stream {
		stream, err := client.CreateChatCompletionStream(context.Background(), *req)
		if err != nil {
			return steps.Reject[ToolCompletionResponse](err), nil
		}
		c := make(chan helpers.Result[ToolCompletionResponse])
		ret := steps.NewStepResult[ToolCompletionResponse](c)

		// TODO(manuel, 2023-11-28) We need to collect this goroutine in Close(), or at least I think so?
		go func() {
			defer close(c)
			defer stream.Close()

			message := ""

			toolCallMerger := NewToolCallMerger()

			ret := ToolCompletionResponse{}

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
					return
				default:
					response, err := stream.Recv()
					if errors.Is(err, io.EOF) {
						csf.subscriptionManager.PublishBlind(&chat.EventText{
							Event: chat.Event{
								Type:     chat.EventTypeFinal,
								Metadata: metadata,
							},
							Text: message,
						})
						toolCalls := toolCallMerger.GetToolCalls()

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
						})
						c <- helpers.NewErrorResult[ToolCompletionResponse](err)
						return
					}

					// TODO(manuel, 2023-11-28) Handle multiple choices
					delta := response.Choices[0].Delta
					deltaContent := delta.Content
					if delta.Content == "" {
						deltaContent = GetToolCallDelta(delta.ToolCalls)

					}
					toolCallMerger.AddToolCalls(delta.ToolCalls)

					message += deltaContent

					csf.subscriptionManager.PublishBlind(&chat.EventPartialCompletion{
						Event: chat.Event{
							Type:     chat.EventTypePartial,
							Metadata: metadata,
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
		resp, err := client.CreateChatCompletion(ctx, *req)

		if errors.Is(err, context.Canceled) {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeInterrupt,
				Metadata: metadata,
			})
			return steps.Reject[ToolCompletionResponse](err), nil
		}

		if err != nil {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeError,
				Error:    err,
				Metadata: metadata,
			})
			return steps.Reject[ToolCompletionResponse](err), nil
		}

		csf.subscriptionManager.PublishBlind(&chat.EventText{
			Event: chat.Event{
				Type:     chat.EventTypeFinal,
				Metadata: metadata,
			},
			Text: resp.Choices[0].Message.Content,
		})

		// TODO(manuel, 2023-11-28) Handle multiple choices
		s, _ := json.MarshalIndent(resp.Choices[0].Message.ToolCalls, "", " ")
		fmt.Printf("final toolcalls:\n%s\n%s\n", resp.Choices[0].FinishReason, s)
		ret := ToolCompletionResponse{
			Role:      resp.Choices[0].Message.Role,
			Content:   resp.Choices[0].Message.Content,
			ToolCalls: resp.Choices[0].Message.ToolCalls,
		}
		return steps.Resolve(ret), nil
	}
}

type ToolStepFactory func() (steps.Step[[]*conversation.Message, ToolCompletionResponse], error)

func (f ToolStepFactory) NewStep() (steps.Step[[]*conversation.Message, ToolCompletionResponse], error) {
	return f()
}

func NewToolStepFactory(stepSettings *settings.StepSettings, tools []go_openai.Tool) ToolStepFactory {
	return ToolStepFactory(
		func() (steps.Step[[]*conversation.Message, ToolCompletionResponse], error) {
			return &ToolStep{
				Settings: stepSettings,
				Tools:    tools,
			}, nil
		})
}

var _ steps.Step[ToolCompletionResponse, map[string]interface{}] = (*ExecuteToolStep)(nil)

type ExecuteToolStep struct {
	Tools               map[string]interface{}
	subscriptionManager *helpers.SubscriptionManager
}

type ExecuteToolStepOption func(*ExecuteToolStep) error

func WithExecuteToolStepSubscriptionManager(subscriptionManager *helpers.SubscriptionManager) ExecuteToolStepOption {
	return func(step *ExecuteToolStep) error {
		step.subscriptionManager = subscriptionManager
		return nil
	}
}

func NewExecuteToolStep(
	tools map[string]interface{},
	options ...ExecuteToolStepOption,
) (*ExecuteToolStep, error) {
	ret := &ExecuteToolStep{
		Tools:               tools,
		subscriptionManager: helpers.NewSubscriptionManager(),
	}

	for _, option := range options {
		err := option(ret)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

var _ steps.Step[ToolCompletionResponse, map[string]interface{}] = (*ExecuteToolStep)(nil)

func (e *ExecuteToolStep) Start(
	ctx context.Context,
	input ToolCompletionResponse,
) (steps.StepResult[map[string]interface{}], error) {
	res := map[string]interface{}{}
	for _, toolCall := range input.ToolCalls {
		if toolCall.Type != "function" {
			log.Warn().Str("type", string(toolCall.Type)).Msg("Unknown tool type")
			continue
		}
		tool := e.Tools[toolCall.Function.Name]
		if tool == nil {
			return steps.Reject[map[string]interface{}](fmt.Errorf("could not find tool %s", toolCall.Function.Name)), nil
		}

		var v interface{}
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &v)
		if err != nil {
			return steps.Reject[map[string]interface{}](err), nil
		}

		vs_, err := helpers.CallFunctionFromJson(tool, v)
		if err != nil {
			return steps.Reject[map[string]interface{}](err), nil
		}

		if len(vs_) == 1 {
			res[toolCall.Function.Name] = vs_[0].Interface()
		} else {
			vals := []interface{}{}
			for _, v_ := range vs_ {
				vals = append(vals, v_.Interface())
			}
			res[toolCall.Function.Name] = vals
		}
	}

	return steps.Resolve(res), nil
}

// TODO(manuel, 2024-01-11) I am not sure we need factories... Potentially because we want new step IDs on creation...
// But then really the ID should be in the step result...

type ExecuteToolStepFactory func() (steps.Step[ToolCompletionResponse, map[string]interface{}], error)

func (f ExecuteToolStepFactory) NewStep() (steps.Step[ToolCompletionResponse, map[string]interface{}], error) {
	return f()
}

func NewExecuteToolStepFactory(tools map[string]interface{}) ExecuteToolStepFactory {
	return ExecuteToolStepFactory(
		func() (steps.Step[ToolCompletionResponse, map[string]interface{}], error) {
			return &ExecuteToolStep{
				Tools: tools,
			}, nil
		})
}
