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
	"github.com/pkg/errors"
	go_openai "github.com/sashabaranov/go-openai"
)

type ChatToolStep struct {
	reflector           *jsonschema.Reflector
	toolFunctions       map[string]interface{}
	tools               []go_openai.Tool
	stepSettings        *settings.StepSettings
	cancel              context.CancelFunc
	subscriptionManager *helpers.SubscriptionManager
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
		subscriptionManager: helpers.NewSubscriptionManager(),
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
	if t.cancel != nil {
		return nil, errors.New("step already started")
	}

	ctx, t.cancel = context.WithCancel(ctx)

	// TODO(manuel, 2024-01-11) This should be refactored since it's going to be spread around
	var parentMessage *conversation.Message
	parentID := uuid.Nil
	conversationID := uuid.New()
	toolCompletionMessageID := uuid.New()
	toolResultMessageID := uuid.New()

	if len(input) > 0 {
		parentMessage = input[len(input)-1]
		parentID = parentMessage.ID
		conversationID = parentMessage.ConversationID
	}

	toolStep, err := NewToolStep(
		t.stepSettings, t.tools,
		WithToolStepConversationID(conversationID),
		WithToolStepParentID(parentID),
		WithToolStepMessageID(toolCompletionMessageID),
		WithToolStepSubscriptionManager(t.subscriptionManager),
	)
	if err != nil {
		return nil, err
	}

	toolResult, err := toolStep.Start(ctx, input)
	if err != nil {
		return nil, err
	}
	step, err := NewExecuteToolStep(t.toolFunctions,
		WithExecuteToolStepSubscriptionManager(t.subscriptionManager),
	)
	if err != nil {
		return nil, err
	}
	execResult := steps.Bind[ToolCompletionResponse, map[string]interface{}](ctx, toolResult, step)

	responseToStringStep := &utils.LambdaStep[map[string]interface{}, string]{
		Function: func(s map[string]interface{}) helpers.Result[string] {
			s_, _ := json.MarshalIndent(s, "", " ")
			t.subscriptionManager.PublishBlind(&chat.Event{
				Type: chat.EventTypeFinal,
				Metadata: chat.EventMetadata{
					ID:             toolResultMessageID,
					ParentID:       toolCompletionMessageID,
					ConversationID: conversationID,
				}})
			return helpers.NewValueResult[string](string(s_))
		},
	}
	stringResult := steps.Bind[map[string]interface{}, string](ctx, execResult, responseToStringStep)

	return stringResult, nil
}

func (t *ChatToolStep) Interrupt() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (t *ChatToolStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	t.subscriptionManager.AddPublishedTopic(topic, publisher)
	return nil
}

var _ chat.Step = &ChatToolStep{}
