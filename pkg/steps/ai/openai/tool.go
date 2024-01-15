package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
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
	parentID            conversation.NodeID
	messageID           conversation.NodeID
}

var _ steps.Step[[]*conversation.Message, ToolCompletionResponse] = (*ToolStep)(nil)

type ToolStepOption func(*ToolStep) error

func WithToolStepSubscriptionManager(subscriptionManager *helpers.SubscriptionManager) ToolStepOption {
	return func(step *ToolStep) error {
		step.subscriptionManager = subscriptionManager
		return nil
	}
}

func WithToolStepParentID(parentID conversation.NodeID) ToolStepOption {
	return func(step *ToolStep) error {
		step.parentID = parentID
		return nil
	}
}

func WithToolStepMessageID(messageID conversation.NodeID) ToolStepOption {
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

func (csf *ToolStep) SetStreaming(b bool) {
	csf.Settings.Chat.Stream = b
}

const MetadataToolCallsSlug = "tool-calls"

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
		Metadata: metadata,
	})

	ctx_, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		cancel()
	}()

	if stream {
		stream_, err := client.CreateChatCompletionStream(context.Background(), *req)
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
								Name:      toolCall.Function.Name,
								Arguments: toolCall.Function.Arguments,
							})
						}
						stepMetadata.Metadata[MetadataToolCallsSlug] = toolCalls_

						msg := &chat.EventText{
							Event: chat.Event{
								Type:     chat.EventTypeFinal,
								Metadata: metadata,
								Step:     stepMetadata,
							},
							Text: message,
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

		// TODO(manuel, 2024-01-12) Here we need to send the content of the toolcalls if the content is empty
		csf.subscriptionManager.PublishBlind(&chat.EventText{
			Event: chat.Event{
				Type:     chat.EventTypeFinal,
				Metadata: metadata,
				Step:     stepMetadata,
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

func (r *ToolStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	r.subscriptionManager.AddPublishedTopic(topic, publisher)
	return nil
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

func (e *ExecuteToolStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	e.subscriptionManager.AddPublishedTopic(topic, publisher)
	return nil
}

const MetadataToolsSlug = "tools"

func (e *ExecuteToolStep) Start(
	ctx context.Context,
	input ToolCompletionResponse,
) (steps.StepResult[map[string]interface{}], error) {
	res := map[string]interface{}{}

	toolMetadata := map[string]interface{}{}
	for name, tool := range e.Tools {
		jsonSchema, err := helpers.GetFunctionParametersJsonSchema(&jsonschema.Reflector{}, tool)
		if err != nil {
			return steps.Reject[map[string]interface{}](err), nil
		}
		s, _ := json.MarshalIndent(jsonSchema, "", "  ")
		toolMetadata[name] = go_openai.Tool{
			Type: "function",
			Function: go_openai.FunctionDefinition{
				Name:        name,
				Description: jsonSchema.Description,
				Parameters:  json.RawMessage(s),
			},
		}
	}

	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "execute-tool-step",
		InputType:  "ToolCompletionResponse",
		OutputType: "map[string]interface{}",
		Metadata: map[string]interface{}{
			MetadataToolsSlug: toolMetadata,
		},
	}

	e.subscriptionManager.PublishBlind(&chat.Event{
		Type: chat.EventTypeStart,
		Step: stepMetadata,
	})
	for _, toolCall := range input.ToolCalls {
		if toolCall.Type != "function" {
			log.Warn().Str("type", string(toolCall.Type)).Msg("Unknown tool type")
			continue
		}
		tool := e.Tools[toolCall.Function.Name]
		if tool == nil {
			e.subscriptionManager.PublishBlind(&chat.Event{
				Type:  chat.EventTypeError,
				Error: fmt.Errorf("could not find tool %s", toolCall.Function.Name),
				Step:  stepMetadata,
			})
			return steps.Reject[map[string]interface{}](
				fmt.Errorf("could not find tool %s", toolCall.Function.Name),
				steps.WithMetadata[map[string]interface{}](stepMetadata),
			), nil
		}

		var v interface{}
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &v)
		if err != nil {
			e.subscriptionManager.PublishBlind(&chat.Event{
				Type:  chat.EventTypeError,
				Error: fmt.Errorf("could not find tool %s", toolCall.Function.Name),
				Step:  stepMetadata,
			})
			return steps.Reject[map[string]interface{}](
				err,
				steps.WithMetadata[map[string]interface{}](stepMetadata),
			), nil
		}

		vs_, err := helpers.CallFunctionFromJson(tool, v)
		if err != nil {
			e.subscriptionManager.PublishBlind(&chat.Event{
				Type:  chat.EventTypeError,
				Error: fmt.Errorf("could not find tool %s", toolCall.Function.Name),
				Step:  stepMetadata,
			})
			return steps.Reject[map[string]interface{}](
				err,
				steps.WithMetadata[map[string]interface{}](stepMetadata),
			), nil
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

	r, _ := json.MarshalIndent(res, "", "  ")

	e.subscriptionManager.PublishBlind(&chat.EventText{
		Event: chat.Event{
			Type: chat.EventTypeFinal,
			Step: stepMetadata,
		},
		Text: string(r),
	})

	return steps.Resolve(res,
		steps.WithMetadata[map[string]interface{}](stepMetadata),
	), nil
}
