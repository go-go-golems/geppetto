package js

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps/ai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/rs/zerolog/log"
)

// SetupChatStepFactory creates a setup function for the ChatStepFactory
func SetupChatStepFactory(stepSettings *settings.StepSettings) SetupFunction {
	return func(vm *goja.Runtime, engine *RuntimeEngine) {
		log.Debug().Msg("Setting up ChatStepFactory")

		// Create factory constructor
		factoryConstructor := func(call goja.FunctionCall) goja.Value {
			log.Debug().Msg("Creating new ChatStepFactory instance")

			factory := &ai.StandardStepFactory{
				Settings: stepSettings,
			}

			factoryObj := vm.NewObject()

			// Add newStep method to factory
			err := factoryObj.Set("newStep", func(call goja.FunctionCall) goja.Value {
				log.Debug().Msg("Creating new chat step")

				// Create the step using the factory
				step, err := factory.NewStep()
				if err != nil {
					log.Error().Err(err).Msg("Failed to create chat step")
					panic(vm.NewGoError(fmt.Errorf("failed to create chat step: %w", err)))
				}
				log.Debug().Msg("Chat step created successfully")

				// Create watermill-based step object
				stepObjectFactory := CreateWatermillStepObject(
					engine,
					step,
					// Input converter: goja.Value -> conversation.Conversation
					func(v goja.Value) conversation.Conversation {
						log.Debug().Msg("Converting input to conversation")
						if jsConv, ok := v.Export().(*JSConversation); ok {
							return jsConv.ToGoConversation()
						}
						// If it's not a JSConversation, assume it's a raw conversation object
						// This is a fallback case
						if convData := v.Export(); convData != nil {
							// Try to convert back to conversation - this is a simplified approach
							log.Warn().Msg("Attempting to convert non-JSConversation to conversation")
							return conversation.NewConversation()
						}
						log.Error().Msg("Invalid conversation input")
						panic(vm.NewTypeError("expected Conversation object"))
					},
					// Output converter: *conversation.Message -> goja.Value
					func(msg *conversation.Message) goja.Value {
						log.Debug().Msg("Converting message output to JS value")
						if msg == nil {
							return vm.ToValue(nil)
						}

						// Convert message to a JavaScript-friendly object
						msgObj := vm.NewObject()
						msgObj.Set("id", msg.ID.String())
						msgObj.Set("parentID", msg.ParentID.String())
						msgObj.Set("time", msg.Time.Format("2006-01-02T15:04:05Z07:00"))
						msgObj.Set("lastUpdate", msg.LastUpdate.Format("2006-01-02T15:04:05Z07:00"))
						msgObj.Set("metadata", msg.Metadata)

						// Handle different message types
						switch content := msg.Content.(type) {
						case *conversation.ChatMessageContent:
							msgObj.Set("type", "chat-message")
							msgObj.Set("role", string(content.Role))
							msgObj.Set("text", content.Text)
							if len(content.Images) > 0 {
								images := make([]interface{}, len(content.Images))
								for i, img := range content.Images {
									imgObj := map[string]interface{}{
										"imageURL":  img.ImageURL,
										"imageName": img.ImageName,
										"mediaType": img.MediaType,
										"detail":    img.Detail,
									}
									images[i] = imgObj
								}
								msgObj.Set("images", images)
							}
						case *conversation.ToolUseContent:
							msgObj.Set("type", "tool-use")
							msgObj.Set("toolID", content.ToolID)
							msgObj.Set("name", content.Name)
							msgObj.Set("input", content.Input)
							msgObj.Set("toolType", content.Type)
						case *conversation.ToolResultContent:
							msgObj.Set("type", "tool-result")
							msgObj.Set("toolID", content.ToolID)
							msgObj.Set("result", content.Result)
						default:
							log.Warn().Interface("content", content).Msg("Unknown message content type")
							msgObj.Set("type", "unknown")
						}

						return msgObj
					},
				)

				stepObj := stepObjectFactory(vm)
				log.Debug().Msg("Chat step object created successfully")
				return stepObj
			})
			if err != nil {
				log.Error().Err(err).Msg("Failed to set newStep method")
				panic(vm.NewGoError(err))
			}

			log.Debug().Msg("ChatStepFactory instance created successfully")
			return factoryObj
		}

		// Register the constructor in the VM
		err := vm.Set("ChatStepFactory", factoryConstructor)
		if err != nil {
			log.Error().Err(err).Msg("Failed to register ChatStepFactory")
		} else {
			log.Debug().Msg("ChatStepFactory registered successfully")
		}
	}
}
