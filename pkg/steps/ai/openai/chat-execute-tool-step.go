package openai

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	go_openai "github.com/sashabaranov/go-openai"
)

// ChatExecuteToolStep combines a chat step with a tool execution step.
type ChatExecuteToolStep struct {
	reflector           *jsonschema.Reflector
	toolFunctions       map[string]interface{}
	tools               []go_openai.Tool
	stepSettings        *settings.StepSettings
	subscriptionManager *events.PublisherManager
}

var _ chat.Step = &ChatExecuteToolStep{}

type ChatToolStepOption func(step *ChatExecuteToolStep)

// WithReflector sets the JSON schema reflector for the step.
func WithReflector(reflector *jsonschema.Reflector) ChatToolStepOption {
	return func(step *ChatExecuteToolStep) {
		step.reflector = reflector
	}
}

// WithToolFunctions sets the tool functions for the step. The schema is derived from these functions using the reflector.
func WithToolFunctions(toolFunctions map[string]interface{}) ChatToolStepOption {
	return func(step *ChatExecuteToolStep) {
		step.toolFunctions = toolFunctions
	}
}

func NewChatToolStep(stepSettings *settings.StepSettings, options ...ChatToolStepOption) (*ChatExecuteToolStep, error) {
	step := &ChatExecuteToolStep{
		stepSettings:        stepSettings,
		subscriptionManager: events.NewPublisherManager(),
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
			Function: &go_openai.FunctionDefinition{
				Name:        name,
				Description: jsonSchema.Description,
				Parameters:  json.RawMessage(s),
			},
		})
	}

	return step, nil
}

func (t *ChatExecuteToolStep) Start(ctx context.Context, input conversation.Conversation) (
	steps.StepResult[*conversation.Message], error,
) {
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

	chatWithToolsStep, err := NewChatWithToolsStep(
		t.stepSettings, t.tools,
		WithChatWithToolsStepParentID(parentID),
		WithChatWithToolsStepMessageID(toolCompletionMessageID),
		WithChatWithToolsStepSubscriptionManager(t.subscriptionManager),
	)
	if err != nil {
		return nil, err
	}

	toolCompletionResponse, err := chatWithToolsStep.Start(cancellableCtx, input)
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
	// TODO(manuel, 2024-07-04) The return type of this step should actually be multiple ToolResult, and we can make
	// this more generic to have it handle claude tool calls as well
	execResult := steps.Bind[ToolCompletionResponse, []events.ToolResult](cancellableCtx, toolCompletionResponse, step)

	responseToStringID := conversation.NewNodeID()

	// XXX a conversation message should be able to hold tool results as well
	responseToStringStep := &utils.LambdaStep[[]events.ToolResult, *conversation.Message]{
		Function: func(s []events.ToolResult) helpers.Result[*conversation.Message] {
			stepMetadata := &steps.StepMetadata{
				StepID:     uuid.New(),
				Type:       "response-to-string",
				InputType:  "[]chat.ToolResult",
				OutputType: "string",
				Metadata:   map[string]interface{}{},
			}
			metadata := events.EventMetadata{
				ID:       responseToStringID,
				ParentID: toolResultMessageID,
			}
			t.subscriptionManager.PublishBlind(events.NewStartEvent(metadata, stepMetadata))

			s_, _ := json.MarshalIndent(s, "", " ")

			// TODO(manuel, 2024-07-04) Handle multiple tool calls
			// actually needs to have one per tool call, so that we can send the result message to openai
			t.subscriptionManager.PublishBlind(events.NewToolResultEvent(metadata, stepMetadata, events.ToolResult{
				ID:     "",
				Result: "",
			}))
			// TODO(manuel, 2024-07-04) Should there be a ToolResult event here?
			t.subscriptionManager.PublishBlind(events.NewFinalEvent(metadata, stepMetadata, string(s_)))
			return helpers.NewValueResult[*conversation.Message](
				conversation.NewChatMessage(conversation.RoleTool, string(s_)),
			)
		},
	}
	stringResult := steps.Bind[[]events.ToolResult, *conversation.Message](cancellableCtx, execResult, responseToStringStep)

	return stringResult, nil
}

func (t *ChatExecuteToolStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	t.subscriptionManager.RegisterPublisher(topic, publisher)
	return nil
}

var _ chat.Step = &ChatExecuteToolStep{}
