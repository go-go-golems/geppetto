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
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// RuntimeEngine manages a single Goja VM, event loop, and watermill router
// for handling JavaScript execution with event streaming
type RuntimeEngine struct {
	ctx    context.Context
	cancel context.CancelFunc

	Loop   *eventloop.EventLoop
	Bus    *gochannel.GoChannel // 1 shared pub/sub
	Router *message.Router      // 1 shared router

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

// StepWatcher provides Go access to step events
type StepWatcher struct {
	Events <-chan events.Event
	close  func()
}

// Close stops watching events
func (sw *StepWatcher) Close() {
	if sw.close != nil {
		sw.close()
	}
}

// SetupFunction is called to set up the JavaScript environment
type SetupFunction func(vm *goja.Runtime, engine *RuntimeEngine)

// RuntimeEngineConfig holds configuration for the runtime engine
type RuntimeEngineConfig struct {
	SetupFunctions []SetupFunction
}

// NewRuntimeEngine creates a new RuntimeEngine (does not start the loop)
func NewRuntimeEngine() *RuntimeEngine {
	log.Debug().Msg("Creating new RuntimeEngine")
	ctx, cancel := context.WithCancel(context.Background())

	eng := &RuntimeEngine{
		ctx:    ctx,
		cancel: cancel,
		Loop:   eventloop.NewEventLoop(),
		Bus: gochannel.NewGoChannel(
			gochannel.Config{BlockPublishUntilSubscriberAck: true},
			watermill.NopLogger{}),
		running:        map[string]*runner{},
		setupFunctions: []SetupFunction{},
	}
	log.Debug().Msg("RuntimeEngine struct created")

	var err error
	eng.Router, err = message.NewRouter(message.RouterConfig{
		CloseTimeout: 15 * time.Second,
	}, watermill.NopLogger{})
	if err != nil {
		panic(fmt.Sprintf("failed to create router: %v", err))
	}
	log.Debug().Msg("Watermill router created")

	// Start router in background
	log.Debug().Msg("Starting watermill router")
	go eng.Router.Run(ctx)
	<-eng.Router.Running() // wait until handlers live
	log.Debug().Msg("Watermill router is running")

	log.Debug().Msg("RuntimeEngine initialization complete")
	return eng
}

// AddSetupFunction adds a setup function to be called during Start()
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

	go func() {
		log.Debug().Msg("Starting JS event loop")
		e.Loop.StartInForeground()
		log.Debug().Msg("JS event loop finished")
	}()
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

// Close shuts down the runtime engine
func (e *RuntimeEngine) Close() error {
	e.cancel()
	e.Loop.Stop()
	return e.Router.Close()
}

// RunStep executes a step with event streaming, automatically registering
// and unregistering per-step handlers
func (e *RuntimeEngine) RunStep(
	step steps.Step[any, any],
	input any,
	onEvent goja.Callable,
) string {
	stepID := uuid.NewString()
	log.Debug().Str("stepID", stepID).Msg("Starting RunStep")

	// record JS callback
	e.mu.Lock()
	r := &runner{
		stepID:   stepID,
		onEvent:  onEvent,
		finished: make(chan struct{}),
	}
	e.running[stepID] = r
	e.mu.Unlock()
	log.Debug().Str("stepID", stepID).Msg("Registered step runner")

	// 1️⃣ Add a per-step handler on the shared router
	log.Debug().Str("stepID", stepID).Str("topic", "step."+stepID).Msg("Adding handler to router")
	h := e.Router.AddNoPublisherHandler(
		"into-vm-"+stepID, // unique name
		"step."+stepID,    // topic filter
		e.Bus,             // subscriber
		func(msg *message.Message) error {
			log.Debug().Str("stepID", stepID).Msg("Received event message")
			ev, err := events.NewEventFromJson(msg.Payload)
			if err != nil {
				log.Error().Err(err).Str("stepID", stepID).Msg("Failed to unmarshal event")
				return err
			}

			e.mu.RLock()
			runner, exists := e.running[stepID]
			e.mu.RUnlock()

			if !exists {
				log.Debug().Str("stepID", stepID).Msg("Step already finished, ignoring event")
				return nil
			}

			log.Debug().Str("stepID", stepID).Str("eventType", fmt.Sprintf("%T", ev)).Msg("Calling JS event handler")
			e.Loop.RunOnLoop(func(vm *goja.Runtime) {
				_, err := runner.onEvent(goja.Undefined(), vm.ToValue(ev))
				if err != nil {
					log.Error().Err(err).Str("stepID", stepID).Msg("Error calling JS event handler")
				} else {
					log.Debug().Str("stepID", stepID).Msg("Successfully called JS event handler")
				}
			})
			return nil
		})

	// Hot-plug the handler immediately
	log.Debug().Str("stepID", stepID).Msg("Running handlers")
	err := e.Router.RunHandlers(e.ctx)
	if err != nil {
		log.Error().Err(err).Str("stepID", stepID).Msg("Failed to run handlers")
	}

	// 2️⃣ Launch the step
	log.Debug().Str("stepID", stepID).Msg("Launching step execution")
	go func() {
		// Register the step with the watermill publisher
		log.Debug().Str("stepID", stepID).Str("topic", "step."+stepID).Msg("Registering step with publisher")
		err := step.AddPublishedTopic(e.Bus, "step."+stepID)
		if err != nil {
			log.Error().Err(err).Str("stepID", stepID).Msg("Failed to register publisher topic")
			close(r.finished)
			return
		}
		log.Debug().Str("stepID", stepID).Msg("Step registered with publisher")

		// Execute the step
		log.Debug().Str("stepID", stepID).Msg("Starting step execution")
		result, err := step.Start(e.ctx, input)
		if err != nil {
			log.Error().Err(err).Str("stepID", stepID).Msg("Step execution failed")
			close(r.finished)
			return
		}
		defer result.Cancel()
		log.Debug().Str("stepID", stepID).Msg("Step started successfully")

		// Wait for step completion
		results := result.Return()
		log.Debug().Int("result_count", len(results)).Str("stepID", stepID).Msg("Step completed")

		close(r.finished) // signal completion
		log.Debug().Str("stepID", stepID).Msg("Step finished signal sent")
	}()

	// 3️⃣ Auto-detach the handler when done
	go func() {
		log.Debug().Str("stepID", stepID).Msg("Waiting for step completion")
		<-r.finished
		log.Debug().Str("stepID", stepID).Msg("Step finished, stopping handler")
		h.Stop() // unsubscribes the topic
		e.mu.Lock()
		delete(e.running, stepID) // tidy map
		e.mu.Unlock()
		log.Debug().Str("stepID", stepID).Msg("Step handler unregistered")
	}()

	log.Debug().Str("stepID", stepID).Msg("RunStep completed, returning stepID")
	return stepID
}
