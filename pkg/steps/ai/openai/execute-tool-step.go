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
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	openai2 "github.com/sashabaranov/go-openai"
)

type ExecuteToolStep struct {
	Tools               map[string]interface{}
	subscriptionManager *events.PublisherManager
	messageID           conversation.NodeID
	parentID            conversation.NodeID
}

var _ steps.Step[ToolCompletionResponse, map[string]interface{}] = (*ExecuteToolStep)(nil)

type ExecuteToolStepOption func(*ExecuteToolStep) error

func WithExecuteToolStepSubscriptionManager(subscriptionManager *events.PublisherManager) ExecuteToolStepOption {
	return func(step *ExecuteToolStep) error {
		step.subscriptionManager = subscriptionManager
		return nil
	}
}

func WithExecuteToolStepParentID(parentID conversation.NodeID) ExecuteToolStepOption {
	return func(step *ExecuteToolStep) error {
		step.parentID = parentID
		return nil
	}
}

func WithExecuteToolStepMessageID(messageID conversation.NodeID) ExecuteToolStepOption {
	return func(step *ExecuteToolStep) error {
		step.messageID = messageID
		return nil
	}
}

func NewExecuteToolStep(
	tools map[string]interface{},
	options ...ExecuteToolStepOption,
) (*ExecuteToolStep, error) {
	ret := &ExecuteToolStep{
		Tools:               tools,
		subscriptionManager: events.NewPublisherManager(),
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
	e.subscriptionManager.SubscribePublisher(topic, publisher)
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
		toolMetadata[name] = openai2.Tool{
			Type: "function",
			Function: openai2.FunctionDefinition{
				Name:        name,
				Description: jsonSchema.Description,
				Parameters:  json.RawMessage(s),
			},
		}
	}

	metadata := chat.EventMetadata{
		ID:       e.messageID,
		ParentID: e.parentID,
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
		Type:     chat.EventTypeStart,
		Step:     stepMetadata,
		Metadata: metadata,
	})
	for _, toolCall := range input.ToolCalls {
		if toolCall.Type != "function" {
			log.Warn().Str("type", string(toolCall.Type)).Msg("Unknown tool type")
			continue
		}
		tool := e.Tools[toolCall.Function.Name]
		if tool == nil {
			e.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeError,
				Error:    errors.Errorf("could not find tool %s", toolCall.Function.Name),
				Metadata: metadata,
				Step:     stepMetadata,
			})
			return steps.Reject[map[string]interface{}](
				errors.Errorf("could not find tool %s", toolCall.Function.Name),
				steps.WithMetadata[map[string]interface{}](stepMetadata),
			), nil
		}

		var v interface{}
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &v)
		if err != nil {
			e.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeError,
				Error:    errors.Errorf("could not find tool %s", toolCall.Function.Name),
				Metadata: metadata,
				Step:     stepMetadata,
			})
			return steps.Reject[map[string]interface{}](
				err,
				steps.WithMetadata[map[string]interface{}](stepMetadata),
			), nil
		}

		vs_, err := helpers.CallFunctionFromJson(tool, v)
		if err != nil {
			e.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeError,
				Error:    errors.Errorf("could not find tool %s", toolCall.Function.Name),
				Metadata: metadata,
				Step:     stepMetadata,
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
			Type:     chat.EventTypeFinal,
			Metadata: metadata,
			Step:     stepMetadata,
		},
		Text: string(r),
	})

	return steps.Resolve(res,
		steps.WithMetadata[map[string]interface{}](stepMetadata),
	), nil
}
