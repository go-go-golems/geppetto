package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	openai2 "github.com/sashabaranov/go-openai"
)

// TODO(manuel, 2024-07-04) Make this use the chat.ToolCall and chat.ToolResult structs and make it generic
type ExecuteToolStep struct {
	Tools               map[string]interface{}
	subscriptionManager *events.PublisherManager
	messageID           conversation.NodeID
	parentID            conversation.NodeID
}

var _ steps.Step[ToolCompletionResponse, []events.ToolResult] = (*ExecuteToolStep)(nil)

//type ToolResult struct {
//	ID string
//	Result interface{}
//}

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

var _ steps.Step[ToolCompletionResponse, []events.ToolResult] = (*ExecuteToolStep)(nil)

func (e *ExecuteToolStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	e.subscriptionManager.RegisterPublisher(topic, publisher)
	return nil
}

const MetadataToolsSlug = "tools"

func (e *ExecuteToolStep) Start(
	ctx context.Context,
	input ToolCompletionResponse,
) (steps.StepResult[[]events.ToolResult], error) {
	res := []events.ToolResult{}

	toolMetadata := map[string]interface{}{}
	for name, tool := range e.Tools {
		jsonSchema, err := helpers.GetFunctionParametersJsonSchema(&jsonschema.Reflector{}, tool)
		if err != nil {
			return steps.Reject[[]events.ToolResult](err), nil
		}
		s, _ := json.MarshalIndent(jsonSchema, "", "  ")
		toolMetadata[name] = openai2.Tool{
			Type: "function",
			Function: &openai2.FunctionDefinition{
				Name:        name,
				Description: jsonSchema.Description,
				Parameters:  json.RawMessage(s),
			},
		}
	}

	metadata := events.EventMetadata{
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

	e.subscriptionManager.PublishBlind(events.NewStartEvent(metadata, stepMetadata))
	for _, toolCall := range input.ToolCalls {
		if toolCall.Type != "function" {
			log.Warn().Str("type", string(toolCall.Type)).Msg("Unknown tool type")
			continue
		}
		tool := e.Tools[toolCall.Function.Name]
		if tool == nil {
			errorString := fmt.Sprintf("could not find tool %s", toolCall.Function.Name)
			e.subscriptionManager.PublishBlind(events.NewErrorEvent(metadata, stepMetadata, errorString))
			return steps.Reject[[]events.ToolResult](
				errors.Errorf("could not find tool %s", toolCall.Function.Name),
				steps.WithMetadata[[]events.ToolResult](stepMetadata),
			), nil
		}

		var v interface{}
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &v)
		if err != nil {
			e.subscriptionManager.PublishBlind(events.NewErrorEvent(metadata, stepMetadata, err.Error()))
			return steps.Reject[[]events.ToolResult](
				err,
				steps.WithMetadata[[]events.ToolResult](stepMetadata),
			), nil
		}

		vs_, err := helpers.CallFunctionFromJson(tool, v)
		if err != nil {
			e.subscriptionManager.PublishBlind(events.NewErrorEvent(metadata, stepMetadata, err.Error()))
			return steps.Reject[[]events.ToolResult](
				err,
				steps.WithMetadata[[]events.ToolResult](stepMetadata),
			), nil
		}

		toolResult := events.ToolResult{ID: toolCall.ID}

		if len(vs_) == 1 {
			v_, err := json.Marshal(vs_[0].Interface())
			if err != nil {
				return steps.Reject[[]events.ToolResult](err,
					steps.WithMetadata[[]events.ToolResult](stepMetadata),
				), nil
			}
			toolResult.Result = string(v_)
		} else {
			vals := []interface{}{}
			for _, v_ := range vs_ {
				vals = append(vals, v_.Interface())
			}
			v_, err := json.Marshal(vals)
			if err != nil {
				return steps.Reject[[]events.ToolResult](err, steps.WithMetadata[[]events.ToolResult](stepMetadata)), nil
			}
			toolResult.Result = string(v_)
		}

		e.subscriptionManager.PublishBlind(events.NewToolResultEvent(metadata, stepMetadata, toolResult))

		res = append(res, toolResult)
	}

	r, _ := json.MarshalIndent(res, "", "  ")

	e.subscriptionManager.PublishBlind(events.NewFinalEvent(metadata, stepMetadata, string(r)))

	return steps.Resolve(res,
		steps.WithMetadata[[]events.ToolResult](stepMetadata),
	), nil
}
