package js

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/rs/zerolog/log"
)

// SetupDoubleStep creates a setup function for a simple test step that doubles numbers
func SetupDoubleStep() SetupFunction {
	return func(vm *goja.Runtime, engine *RuntimeEngine) {
		log.Debug().Msg("Setting up doubleStep")

		// Create a step that publishes events
		doubleStep := &TestDoubleStep{}
		log.Debug().Msg("Created testDoubleStep")

		// Create watermill-based step object factory
		log.Debug().Msg("Creating watermill step object factory")
		stepObjectFactory := CreateWatermillStepObject(
			engine,
			doubleStep,
			func(v goja.Value) float64 {
				val := v.ToFloat()
				log.Debug().Float64("value", val).Msg("Input converter called")
				return val
			},
			func(v float64) goja.Value {
				log.Debug().Float64("value", v).Msg("Output converter called")
				return vm.ToValue(v)
			},
		)

		// Create the step object in the VM context
		log.Debug().Msg("Creating step object in VM context")
		stepObj := stepObjectFactory(vm)

		log.Debug().Msg("Registering doubleStep in VM")
		err := vm.Set("doubleStep", stepObj)
		if err != nil {
			log.Error().Err(err).Msg("Failed to register doubleStep")
			return
		}
		log.Debug().Msg("doubleStep registered successfully")
	}
}

// SetupConversation creates a setup function for conversation handling
func SetupConversation() SetupFunction {
	return func(vm *goja.Runtime, engine *RuntimeEngine) {
		log.Debug().Msg("Setting up conversation")
		err := RegisterConversation(vm)
		if err != nil {
			log.Error().Err(err).Msg("Failed to register conversation")
		}
	}
}

// SetupEmbeddings creates a setup function for embeddings
func SetupEmbeddings(stepSettings *settings.StepSettings) SetupFunction {
	return func(vm *goja.Runtime, engine *RuntimeEngine) {
		log.Debug().Msg("Setting up embeddings")
		factory := embeddings.NewSettingsFactoryFromStepSettings(stepSettings)
		provider, err := factory.NewProvider()
		if err != nil {
			log.Error().Err(err).Msg("Failed to create embeddings provider")
			return
		}

		err = RegisterEmbeddings(vm, "embeddings", provider, engine.Loop)
		if err != nil {
			log.Error().Err(err).Msg("Failed to register embeddings")
		}
	}
}

// SetupDoneCallback creates a setup function that registers a done() callback
func SetupDoneCallback() SetupFunction {
	return func(vm *goja.Runtime, engine *RuntimeEngine) {
		log.Debug().Msg("Setting up done callback")
		err := vm.Set("done", func(args ...interface{}) {
			log.Info().Msg("Done callback called")
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to register done callback")
		}
	}
}

// TestDoubleStep is a test step that publishes events during execution
type TestDoubleStep struct {
	publisherManager *events.PublisherManager
}

func (t *TestDoubleStep) Start(ctx context.Context, input float64) (steps.StepResult[float64], error) {
	log.Debug().Float64("input", input).Msg("Starting TestDoubleStep execution")
	
	// Create result channel
	c := make(chan helpers.Result[float64], 1)
	
	go func() {
		defer close(c)
		
		// Publish start event
		if t.publisherManager != nil {
			startEvent := events.NewStartEvent(events.EventMetadata{}, &steps.StepMetadata{})
			t.publishEvent(startEvent)
		}
		
		// Simulate some work with delay
		fmt.Println("Starting doubleStep")
		time.Sleep(500 * time.Millisecond)
		
		result := input * 2
		
		// Publish final event  
		if t.publisherManager != nil {
			finalEvent := events.NewFinalEvent(events.EventMetadata{}, &steps.StepMetadata{}, fmt.Sprintf("%.2f", result))
			t.publishEvent(finalEvent)
		}
		
		fmt.Println("Finished doubleStep")
		log.Debug().Float64("result", result).Msg("Completed TestDoubleStep execution")
		
		// Send final result to channel
		c <- helpers.NewValueResult(result)
	}()
	
	return steps.NewStepResult[float64](c), nil
}

func (t *TestDoubleStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	if t.publisherManager == nil {
		t.publisherManager = events.NewPublisherManager()
	}
	t.publisherManager.RegisterPublisher(topic, publisher)
	return nil
}

func (t *TestDoubleStep) publishEvent(event events.Event) {
	if t.publisherManager != nil {
		log.Debug().Str("eventType", string(event.Type())).Msg("Publishing event")
		err := t.publisherManager.Publish(event)
		if err != nil {
			log.Error().Err(err).Msg("Failed to publish event")
		}
	}
}
