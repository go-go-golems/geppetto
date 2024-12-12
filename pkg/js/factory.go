package js

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps/ai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
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
			// Convert options from JS array to Go slice
			var options []chat.StepOption
			if len(call.Arguments) > 0 {
				jsOptions := call.Argument(0).Export()
				if arr, ok := jsOptions.([]interface{}); ok {
					for _, opt := range arr {
						if fn, ok := opt.(func(chat.Step) error); ok {
							options = append(options, fn)
						}
					}
				}
			}

			// Create new step
			step, err := factory.NewStep(options...)
			if err != nil {
				panic(runtime.ToValue(err))
			}

			// Define converters for the step
			inputConverter := func(v goja.Value) conversation.Conversation {
				// Convert JavaScript input to Conversation
				if obj := v.ToObject(runtime); obj != nil {
					if jsConv, ok := obj.Export().(*JSConversation); ok {
						return jsConv.ToGoConversation()
					}

					// Fallback for legacy format
					messages := obj.Get("messages")
					if messages != nil {
						jsConv := NewJSConversation(runtime)
						if arr := messages.Export().([]interface{}); arr != nil {
							for _, msg := range arr {
								if msgMap, ok := msg.(map[string]interface{}); ok {
									role := msgMap["role"].(string)
									content := msgMap["content"].(string)
									jsConv.AddMessage(goja.FunctionCall{
										Arguments: []goja.Value{
											runtime.ToValue(role),
											runtime.ToValue(content),
										},
									})
								}
							}
						}
						return jsConv.ToGoConversation()
					}
				}
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
