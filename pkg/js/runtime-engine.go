package js

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// PubSub combines Publisher and Subscriber interfaces for convenience
type PubSub interface {
	message.Publisher
	message.Subscriber
}

// RuntimeEngine manages a single Goja VM, event loop, and watermill router
// for handling JavaScript execution with event streaming.
//
// The RuntimeEngine uses errgroup to manage goroutines for:
// - The watermill router (started in NewRuntimeEngine)
// - The JavaScript event loop (started in Start)
// - Individual step executions (started in StartStep)
//
// All goroutines are properly coordinated and will be cleaned up when Close() is called.
// Use Wait() to wait for all goroutines to complete in graceful shutdown scenarios.
type RuntimeEngine struct {
	ctx    context.Context
	cancel context.CancelFunc
	eg     *errgroup.Group // Manages all goroutines for proper cleanup

	Loop   *eventloop.EventLoop
	Bus    PubSub // Changed to interface to allow custom implementations
	Router *message.Router

	mu             sync.RWMutex
	running        map[string]*runner // stepID → runner
	setupFunctions []SetupFunction
}

// runner tracks a per-step handler and its lifecycle
type runner struct {
	stepID   string
	onEvent  goja.Callable // JS callback
	finished chan struct{} // closed when step ends
}

// SetupFunction is called to set up the JavaScript environment
type SetupFunction func(vm *goja.Runtime, engine *RuntimeEngine)

// Option configures a RuntimeEngine
type Option func(*RuntimeEngine) error

// WithSetupFunction adds a setup function to be called during Start()
func WithSetupFunction(fn SetupFunction) Option {
	return func(e *RuntimeEngine) error {
		e.setupFunctions = append(e.setupFunctions, fn)
		log.Debug().Int("count", len(e.setupFunctions)).Msg("Added setup function via option")
		return nil
	}
}

// WithBus sets a custom message bus (PubSub implementation)
func WithBus(bus PubSub) Option {
	return func(e *RuntimeEngine) error {
		e.Bus = bus
		log.Debug().Msg("Set custom bus via option")
		return nil
	}
}

// WithRouterConfig sets custom router configuration
func WithRouterConfig(config message.RouterConfig) Option {
	return func(e *RuntimeEngine) error {
		var err error
		e.Router, err = message.NewRouter(config, watermill.NopLogger{})
		if err != nil {
			return fmt.Errorf("failed to create router with custom config: %w", err)
		}
		log.Debug().Msg("Created router with custom config via option")
		return nil
	}
}

// WithSetupFunctions adds multiple setup functions at once
func WithSetupFunctions(fns ...SetupFunction) Option {
	return func(e *RuntimeEngine) error {
		e.setupFunctions = append(e.setupFunctions, fns...)
		log.Debug().Int("count", len(fns)).Int("total", len(e.setupFunctions)).Msg("Added multiple setup functions via option")
		return nil
	}
}

// WithLogger sets a custom logger for watermill components
func WithLogger(logger watermill.LoggerAdapter) Option {
	return func(e *RuntimeEngine) error {
		// This option should be applied before creating Bus and Router
		// For now, we'll store it and use it when creating default components
		// Note: This is a limitation - if Bus or Router are provided via other options,
		// they won't use this logger
		log.Debug().Msg("Custom logger option set (will be used for default components)")
		return nil
	}
}

// RuntimeEngineConfig holds configuration for the runtime engine
// Deprecated: Use Option pattern with NewRuntimeEngine instead
type RuntimeEngineConfig struct {
	SetupFunctions []SetupFunction
}

// NewRuntimeEngine creates a new RuntimeEngine with optional configuration
//
// Example usage:
//
//	// Basic usage with default settings
//	engine, err := NewRuntimeEngine()
//
//	// With setup functions
//	engine, err := NewRuntimeEngine(
//	    WithSetupFunctions(setupFn1, setupFn2),
//	)
//
//	// With custom bus and router config
//	customBus := gochannel.NewGoChannel(customConfig, logger)
//	engine, err := NewRuntimeEngine(
//	    WithBus(customBus),
//	    WithRouterConfig(message.RouterConfig{CloseTimeout: 30 * time.Second}),
//	)
func NewRuntimeEngine(opts ...Option) (*RuntimeEngine, error) {
	log.Debug().Msg("Creating new RuntimeEngine")
	ctx, cancel := context.WithCancel(context.Background())
	eg, groupCtx := errgroup.WithContext(ctx)

	eng := &RuntimeEngine{
		ctx:            groupCtx,
		cancel:         cancel,
		eg:             eg,
		Loop:           eventloop.NewEventLoop(),
		running:        map[string]*runner{},
		setupFunctions: []SetupFunction{},
	}
	log.Debug().Msg("RuntimeEngine struct created")

	// Set default bus if not provided via options
	if eng.Bus == nil {
		eng.Bus = gochannel.NewGoChannel(
			gochannel.Config{BlockPublishUntilSubscriberAck: true},
			watermill.NopLogger{})
		log.Debug().Msg("Created default GoChannel bus")
	}

	// Set default router if not provided via options
	if eng.Router == nil {
		var err error
		eng.Router, err = message.NewRouter(message.RouterConfig{
			CloseTimeout: 15 * time.Second,
		}, watermill.NopLogger{})
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create default router: %w", err)
		}
		log.Debug().Msg("Created default router")
	}

	// Apply options
	for i, opt := range opts {
		if err := opt(eng); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to apply option %d: %w", i, err)
		}
	}

	// Start router in errgroup
	log.Debug().Msg("Starting watermill router in errgroup")
	eng.eg.Go(func() error {
		defer func() {
			log.Debug().Msg("Closing watermill router")
			if err := eng.Router.Close(); err != nil {
				log.Error().Err(err).Msg("Error closing router")
			}
		}()

		log.Debug().Msg("Running watermill router")
		return eng.Router.Run(groupCtx)
	})

	<-eng.Router.Running() // wait until handlers live
	log.Debug().Msg("Watermill router is running")

	log.Debug().Msg("RuntimeEngine initialization complete")
	return eng, nil
}

// AddSetupFunction adds a setup function to be called during Start()
// Deprecated: Use WithSetupFunction option instead
func (e *RuntimeEngine) AddSetupFunction(fn SetupFunction) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.setupFunctions = append(e.setupFunctions, fn)
	log.Debug().Int("count", len(e.setupFunctions)).Msg("Added setup function")
}

// Start runs the event loop and calls all registered setup functions
// This calls Loop.Run() which starts the event loop and waits until completion
func (e *RuntimeEngine) Start() {
	log.Debug().Msg("Starting RuntimeEngine event loop")

	e.Loop.RunOnLoop(func(vm *goja.Runtime) {
		log.Debug().Msg("Setting up JavaScript environment")

		// Set up field name mapper
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

		// Set up console logging
		e.setupConsole(vm)

		// Call all registered setup functions
		e.mu.RLock()
		setupFunctions := make([]SetupFunction, len(e.setupFunctions))
		copy(setupFunctions, e.setupFunctions)
		e.mu.RUnlock()

		for i, setupFn := range setupFunctions {
			log.Debug().Int("index", i).Msg("Calling setup function")
			setupFn(vm, e)
		}

		log.Debug().Msg("JavaScript environment setup complete")
	})

	e.eg.Go(func() error {
		log.Debug().Msg("Starting JS event loop in errgroup")
		defer log.Debug().Msg("JS event loop finished")
		e.Loop.StartInForeground()
		return nil
	})
}

// setupConsole adds console.log and console.error to the VM
func (e *RuntimeEngine) setupConsole(vm *goja.Runtime) {
	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		fmt.Println(args...)
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		fmt.Printf("ERROR: %v\n", args...)
		return goja.Undefined()
	})
	_ = vm.Set("console", console)
}

// RunOnLoop executes JavaScript code on the running event loop
// This should only be called while the loop is running (i.e., from within Start())
func (e *RuntimeEngine) RunOnLoop(jsCode string) error {
	done := make(chan error, 1)

	e.Loop.RunOnLoop(func(vm *goja.Runtime) {
		log.Debug().Msg("Executing JavaScript code on loop")
		_, err := vm.RunString(jsCode)
		if err != nil {
			log.Error().Err(err).Msg("JavaScript execution failed")
			done <- err
		} else {
			log.Debug().Msg("JavaScript execution completed")
			done <- nil
		}
	})

	return <-done
}

// Stop stops the event loop
func (e *RuntimeEngine) Stop() {
	log.Debug().Msg("Stopping runtime engine")
	e.Loop.Stop()
}

// Wait waits for all goroutines managed by the runtime engine to complete
// This is useful for graceful shutdown scenarios
func (e *RuntimeEngine) Wait() error {
	log.Debug().Msg("Waiting for runtime engine goroutines to complete")
	err := e.eg.Wait()
	if err != nil && err != context.Canceled {
		log.Error().Err(err).Msg("Runtime engine finished with error")
		return err
	}
	log.Debug().Msg("All runtime engine goroutines completed")
	return nil
}

// Close shuts down the runtime engine
func (e *RuntimeEngine) Close() error {
	log.Debug().Msg("Closing runtime engine")

	// Cancel context to signal all goroutines to stop
	e.cancel()

	// Stop the event loop
	e.Loop.Stop()

	// Wait for all goroutines to finish
	log.Debug().Msg("Waiting for errgroup to finish")
	if err := e.eg.Wait(); err != nil {
		log.Error().Err(err).Msg("Error waiting for errgroup")
		// Don't return error if it's just context cancellation
		if err != context.Canceled {
			return err
		}
	}

	log.Debug().Msg("Runtime engine closed successfully")
	return nil
}

type StepRun struct {
	stepID   string
	finished chan struct{}
}

// RunStep executes a step with event streaming, automatically registering
// and unregistering per-step handlers
func (e *RuntimeEngine) RunStep(
	step steps.Step[any, any],
	stepID string,
	input any,
	onEvent goja.Callable,
) (steps.StepResult[any], error) {
	logger := log.With().Str("stepID", stepID).Logger()

	logger.Debug().Msg("Starting RunStep")

	// record JS callback
	e.mu.Lock()
	r := &runner{
		stepID:   stepID,
		onEvent:  onEvent,
		finished: make(chan struct{}),
	}
	e.running[stepID] = r
	e.mu.Unlock()
	logger.Debug().Msg("Registered step runner")

	// 1️⃣ Add a per-step handler on the shared router
	logger.Debug().Str("topic", "step."+stepID).Msg("Adding handler to router")
	h := e.Router.AddNoPublisherHandler(
		"into-vm-"+stepID, // unique name
		"step."+stepID,    // topic filter
		e.Bus,             // subscriber
		func(msg *message.Message) error {
			logger.Debug().Msg("Received event message")
			ev, err := events.NewEventFromJson(msg.Payload)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to unmarshal event")
				return err
			}

			e.mu.RLock()
			runner, exists := e.running[stepID]
			e.mu.RUnlock()

			if !exists {
				logger.Debug().Msg("Step already finished, ignoring event")
				return nil
			}

			logger.Debug().Str("eventType", string(ev.Type())).Msg("Calling JS event handler")
			e.Loop.RunOnLoop(func(vm *goja.Runtime) {
				// Convert event to a JavaScript-friendly object
				eventObj := vm.NewObject()
				eventObj.Set("type", string(ev.Type()))

				// Set event-specific fields based on type
				switch typedEvent := ev.(type) {
				case *events.EventPartialCompletionStart:
					// Start event
				case *events.EventFinal:
					eventObj.Set("text", typedEvent.Text)
				case *events.EventPartialCompletion:
					eventObj.Set("delta", typedEvent.Delta)
					eventObj.Set("completion", typedEvent.Completion)
				case *events.EventError:
					eventObj.Set("error", typedEvent.Error)
				case *events.EventToolCall:
					toolCallObj := vm.NewObject()
					toolCallObj.Set("id", typedEvent.ToolCall.ID)
					toolCallObj.Set("name", typedEvent.ToolCall.Name)
					toolCallObj.Set("input", vm.ToValue(typedEvent.ToolCall.Input))
					eventObj.Set("toolCall", toolCallObj)
				case *events.EventToolResult:
					toolResultObj := vm.NewObject()
					toolResultObj.Set("id", typedEvent.ToolResult.ID)
					toolResultObj.Set("result", vm.ToValue(typedEvent.ToolResult.Result))
					eventObj.Set("toolResult", toolResultObj)
				}

				// Add metadata
				metaObj := vm.NewObject()
				meta := ev.Metadata()
				metaObj.Set("engine", meta.Engine)
				metaObj.Set("temperature", vm.ToValue(meta.Temperature))
				metaObj.Set("top_p", vm.ToValue(meta.TopP))
				metaObj.Set("max_tokens", vm.ToValue(meta.MaxTokens))
				metaObj.Set("stop_reason", vm.ToValue(meta.StopReason))
				metaObj.Set("usage", vm.ToValue(meta.Usage))
				metaObj.Set("message_id", vm.ToValue(meta.ID))
				metaObj.Set("parent_id", vm.ToValue(meta.ParentID))
				eventObj.Set("meta", metaObj)

				// Add step metadata
				stepMetaObj := vm.NewObject()
				stepMeta := ev.StepMetadata()
				if stepMeta != nil {
					stepMetaObj.Set("step_id", vm.ToValue(stepMeta.StepID))
					stepMetaObj.Set("type", stepMeta.Type)
					stepMetaObj.Set("input_type", stepMeta.InputType)
					stepMetaObj.Set("output_type", stepMeta.OutputType)
					stepMetaObj.Set("meta", vm.ToValue(stepMeta.Metadata))
				}
				eventObj.Set("step", stepMetaObj)

				_, err := runner.onEvent(goja.Undefined(), eventObj)
				if err != nil {
					logger.Error().Err(err).Msg("Error calling JS event handler")
				} else {
					logger.Debug().Msg("Successfully called JS event handler")
				}
			})
			return nil
		})

	// Hot-plug the handler immediately
	logger.Debug().Msg("Running handlers")
	err := e.Router.RunHandlers(e.ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to run handlers")
		return nil, err
	}

	defer func() {
		logger.Debug().Msg("Stopping handler")
		h.Stop() // unsubscribes the topic
		<-h.Stopped()
		e.mu.Lock()
		delete(e.running, stepID) // tidy map
		e.mu.Unlock()
		logger.Debug().Msg("Step handler unregistered")
	}()

	// 2️⃣ Launch the step execution
	logger.Debug().Msg("Launching step execution in errgroup")

	// Register the step with the watermill publisher
	logger.Debug().Str("topic", "step."+stepID).Msg("Registering step with publisher")
	err = step.AddPublishedTopic(e.Bus, "step."+stepID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to register publisher topic")
		return nil, err
	}
	logger.Debug().Msg("Step registered with publisher")

	// Execute the step
	logger.Debug().Msg("Starting step execution")
	result, err := step.Start(e.ctx, input)
	if err != nil {
		logger.Error().Err(err).Msg("Step execution failed")
		return nil, err
	}
	defer result.Cancel()
	logger.Debug().Msg("Step started successfully")

	// Wait for step completion
	results := result.Return()
	logger.Debug().Int("result_count", len(results)).Msg("Step completed")
	return result, nil
}
