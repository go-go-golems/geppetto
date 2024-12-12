package js

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps/ai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

// RegisterFactory registers the chat step factory functionality in the JavaScript runtime
func RegisterFactory(runtime *goja.Runtime, loop *eventloop.EventLoop, stepSettings *settings.StepSettings) error {
	factory := &ai.StandardStepFactory{
		Settings: stepSettings,
	}

	// Create the factory object constructor
	constructor := runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
		this := call.This

		// Create newStep method
		_ = this.Set("newStep", func(call goja.FunctionCall) goja.Value {

			// Create new step
			step, err := factory.NewStep()
			if err != nil {
				panic(runtime.ToValue(err))
			}

			// Define converters for the step
			inputConverter := func(v goja.Value) conversation.Conversation {
				if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
					return conversation.NewConversation()
				}

				// Try to get the conversation directly
				if conv, ok := v.Export().(conversation.Conversation); ok {
					return conv
				}

				// Handle JSConversation case
				if jsConv, ok := v.Export().(*JSConversation); ok {
					return jsConv.ToGoConversation()
				}

				// Fallback to empty conversation
				return conversation.NewConversation()
			}

			outputConverter := func(s string) goja.Value {
				return runtime.ToValue(s)
			}

			// Create step object directly
			stepObj, err := CreateStepObject(
				runtime,
				loop,
				step,
				inputConverter,
				outputConverter,
			)
			if err != nil {
				panic(runtime.ToValue(err))
			}

			return stepObj
		})

		return nil
	})

	// Register the factory constructor
	return runtime.Set("ChatStepFactory", constructor)
}
