package openai

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	go_openai "github.com/sashabaranov/go-openai"
)

type ChatToolStep struct {
	reflector           *jsonschema.Reflector
	toolFunctions       map[string]interface{}
	tools               []go_openai.Tool
	stepSettings        *settings.StepSettings
	subscriptionManager *helpers.PublisherManager
}

var _ chat.Step = &ChatToolStep{}

type ChatToolStepOption func(step *ChatToolStep)

func WithReflector(reflector *jsonschema.Reflector) ChatToolStepOption {
	return func(step *ChatToolStep) {
		step.reflector = reflector
	}
}

func WithToolFunctions(toolFunctions map[string]interface{}) ChatToolStepOption {
	return func(step *ChatToolStep) {
		step.toolFunctions = toolFunctions
	}
}

func NewChatToolStep(stepSettings *settings.StepSettings, options ...ChatToolStepOption) (*ChatToolStep, error) {
	step := &ChatToolStep{
		stepSettings:        stepSettings,
		subscriptionManager: helpers.NewPublisherManager(),
	}
	for _, option := range options {
		option(step)
	}

	if step.reflector == nil {
		step.reflector = &jsonschema.Reflector{}
	}

	for name, tool := range step.toolFunctions {
		jsonSchema, err := helpers.GetFunctionParametersJsonSchema(step.reflector, tool)
		if err != nil {
			return nil, err
		}
		s, _ := json.MarshalIndent(jsonSchema, "", "  ")
		step.tools = append(step.tools, go_openai.Tool{
			Type: "function",
			Function: go_openai.FunctionDefinition{
				Name:        name,
				Description: jsonSchema.Description,
				Parameters:  json.RawMessage(s),
			},
		})
	}

	return step, nil
}

func (t *ChatToolStep) Start(ctx context.Context, input conversation.Conversation) (steps.StepResult[string], error) {
	cancellableCtx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		cancel()
	}()

	var parentMessage *conversation.Message
	parentID := conversation.NullNode
	toolCompletionMessageID := conversation.NewNodeID()
	toolResultMessageID := conversation.NewNodeID()

	if len(input) > 0 {
		parentMessage = input[len(input)-1]
		parentID = parentMessage.ID
	}

	toolStep, err := NewToolStep(
		t.stepSettings, t.tools,
		WithToolStepParentID(parentID),
		WithToolStepMessageID(toolCompletionMessageID),
		WithToolStepSubscriptionManager(t.subscriptionManager),
	)
	if err != nil {
		return nil, err
	}

	toolResult, err := toolStep.Start(cancellableCtx, input)
	if err != nil {
		return nil, err
	}
	step, err := NewExecuteToolStep(t.toolFunctions,
		WithExecuteToolStepSubscriptionManager(t.subscriptionManager),
		WithExecuteToolStepParentID(toolCompletionMessageID),
		WithExecuteToolStepMessageID(toolResultMessageID),
	)
	if err != nil {
		return nil, err
	}
	execResult := steps.Bind[ToolCompletionResponse, map[string]interface{}](cancellableCtx, toolResult, step)

	responseToStringID := conversation.NewNodeID()

	responseToStringStep := &utils.LambdaStep[map[string]interface{}, string]{
		Function: func(s map[string]interface{}) helpers.Result[string] {
			stepMetadata := &steps.StepMetadata{
				StepID:     uuid.New(),
				Type:       "response-to-string",
				InputType:  "map[string]interface{}",
				OutputType: "string",
				Metadata:   map[string]interface{}{},
			}
			t.subscriptionManager.PublishBlind(&chat.Event{
				Type: chat.EventTypeStart,
				Step: stepMetadata,
				Metadata: chat.EventMetadata{
					ID:       responseToStringID,
					ParentID: toolResultMessageID,
				}})
			s_, _ := json.MarshalIndent(s, "", " ")
			t.subscriptionManager.PublishBlind(&chat.EventText{
				Event: chat.Event{
					Type: chat.EventTypeFinal,
					Step: stepMetadata,
					Metadata: chat.EventMetadata{
						ID:       responseToStringID,
						ParentID: toolResultMessageID,
					}},
				Text: string(s_),
			})
			return helpers.NewValueResult[string](string(s_))
		},
	}
	stringResult := steps.Bind[map[string]interface{}, string](cancellableCtx, execResult, responseToStringStep)

	return stringResult, nil
}

func (t *ChatToolStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	t.subscriptionManager.AddPublishedTopic(topic, publisher)
	return nil
}

var _ chat.Step = &ChatToolStep{}
