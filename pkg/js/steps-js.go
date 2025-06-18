package js

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/rs/zerolog/log"
)

// eventAdapter adapts events.Event to steps.Event
type eventAdapter struct {
	events.Event
}

func (e *eventAdapter) Type() string {
	return string(e.Event.Type())
}

// jsEventStream wraps a Go event channel and provides JavaScript APIs
type jsEventStream struct {
	runtime   *goja.Runtime
	loop      *eventloop.EventLoop
	eventCh   <-chan steps.Event
	cancel    func()
	listeners map[string][]goja.Callable
	mu        sync.RWMutex
	done      bool
	ctx       context.Context
	ctxCancel context.CancelFunc
}

// NewJSEventStream creates a new JavaScript event stream wrapper.
func NewJSEventStream(
	runtime *goja.Runtime,
	loop *eventloop.EventLoop,
	eventCh <-chan steps.Event,
	cancel func(),
) *goja.Object {
	ctx, ctxCancel := context.WithCancel(context.Background())
	
	stream := &jsEventStream{
		runtime:   runtime,
		loop:      loop,
		eventCh:   eventCh,
		cancel:    cancel,
		listeners: make(map[string][]goja.Callable),
		ctx:       ctx,
		ctxCancel: ctxCancel,
	}

	// Create the JavaScript object
	obj := runtime.NewObject()
	
	// Add EventEmitter-style methods
	obj.Set("on", stream.on)
	obj.Set("cancel", stream.cancelStream)
	
	// Start processing events in the background
	go stream.processEvents()
	
	return obj
}

// on implements EventEmitter-style event registration
func (s *jsEventStream) on(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(s.runtime.NewTypeError("on() requires event name and handler"))
	}
	
	eventName := call.Arguments[0].String()
	handler, ok := goja.AssertFunction(call.Arguments[1])
	if !ok {
		panic(s.runtime.NewTypeError("handler must be a function"))
	}
	
	s.mu.Lock()
	s.listeners[eventName] = append(s.listeners[eventName], handler)
	s.mu.Unlock()
	
	// Return this for chaining
	return call.This
}

// cancelStream cancels the stream and calls the underlying cancel function
func (s *jsEventStream) cancelStream(call goja.FunctionCall) goja.Value {
	s.mu.Lock()
	if !s.done {
		s.done = true
		s.ctxCancel()
		if s.cancel != nil {
			s.cancel()
		}
	}
	s.mu.Unlock()
	
	return goja.Undefined()
}

// processEvents processes events from the channel and emits them to listeners
func (s *jsEventStream) processEvents() {
	defer func() {
		s.mu.Lock()
		s.done = true
		s.mu.Unlock()
	}()
	
	for {
		select {
		case event, ok := <-s.eventCh:
			if !ok {
				log.Debug().Msg("Event channel closed")
				return
			}
			
			s.emitEvent(event)
			
		case <-s.ctx.Done():
			log.Debug().Msg("Stream context cancelled")
			return
		}
	}
}

// emitEvent emits an event to all registered listeners
func (s *jsEventStream) emitEvent(event steps.Event) {
	jsEvent := s.eventToJS(event)
	eventType := string(event.Type())
	
	s.mu.RLock()
	listeners := make([]goja.Callable, len(s.listeners[eventType]))
	copy(listeners, s.listeners[eventType])
	s.mu.RUnlock()
	
	// Call listeners on the event loop
	for _, listener := range listeners {
		listener := listener // capture for closure
		s.loop.RunOnLoop(func(*goja.Runtime) {
			_, err := listener(goja.Undefined(), jsEvent)
			if err != nil {
				log.Error().Err(err).Str("event", eventType).Msg("Error calling event listener")
			}
		})
	}
}

// eventToJS converts a Go event to a JavaScript object
func (s *jsEventStream) eventToJS(event steps.Event) goja.Value {
	eventType := string(event.Type())
	
	eventObj := map[string]interface{}{
		"type": eventType,
	}
	
	// Debug: Log the event type and underlying type
	log.Debug().
		Str("event_type", eventType).
		Str("go_type", fmt.Sprintf("%T", event)).
		Msg("Converting event to JS")
	
	// For events from the events package, add additional fields
	if eventsEvent, ok := event.(events.Event); ok {
		eventObj["meta"] = s.metadataToJS(eventsEvent.Metadata())
		eventObj["step"] = s.stepMetadataToJS(eventsEvent.StepMetadata())
		
		// Add type-specific fields
		switch e := eventsEvent.(type) {
		case *events.EventPartialCompletion:
			log.Debug().
				Str("delta", e.Delta).
				Str("completion", e.Completion).
				Msg("Partial completion event")
			eventObj["delta"] = e.Delta
			eventObj["completion"] = e.Completion
			
		case *events.EventFinal:
			log.Debug().
				Str("text", e.Text).
				Msg("Final event")
			eventObj["text"] = e.Text
			
		case *events.EventError:
			eventObj["error"] = e.ErrorString
			
		case *events.EventInterrupt:
			eventObj["text"] = e.Text
			
		case *events.EventToolCall:
			eventObj["toolCall"] = map[string]interface{}{
				"id":    e.ToolCall.ID,
				"name":  e.ToolCall.Name,
				"input": e.ToolCall.Input,
			}
			
		case *events.EventToolResult:
			eventObj["toolResult"] = map[string]interface{}{
				"id":     e.ToolResult.ID,
				"result": e.ToolResult.Result,
			}
		default:
			log.Debug().
				Str("event_type", eventType).
				Str("go_type", fmt.Sprintf("%T", eventsEvent)).
				Msg("Unhandled events.Event type")
		}
	} else {
		// For eventAdapter or other wrapped events, try to unwrap
		if adapter, ok := event.(*eventAdapter); ok {
			log.Debug().
				Str("wrapped_type", fmt.Sprintf("%T", adapter.Event)).
				Msg("Found eventAdapter, unwrapping")
			// Recursively process the unwrapped events.Event
			return s.eventToJSFromEventsEvent(adapter.Event)
		}
		
		log.Debug().
			Str("event_type", eventType).
			Str("go_type", fmt.Sprintf("%T", event)).
			Msg("Event does not implement events.Event interface")
	}
	
	return s.runtime.ToValue(eventObj)
}

// eventToJSFromEventsEvent converts a pure events.Event to JavaScript
func (s *jsEventStream) eventToJSFromEventsEvent(eventsEvent events.Event) goja.Value {
	eventType := string(eventsEvent.Type())
	
	eventObj := map[string]interface{}{
		"type": eventType,
		"meta": s.metadataToJS(eventsEvent.Metadata()),
		"step": s.stepMetadataToJS(eventsEvent.StepMetadata()),
	}
	
	// Add type-specific fields
	switch e := eventsEvent.(type) {
	case *events.EventPartialCompletion:
		log.Debug().
			Str("delta", e.Delta).
			Str("completion", e.Completion).
			Msg("Unwrapped partial completion event")
		eventObj["delta"] = e.Delta
		eventObj["completion"] = e.Completion
		
	case *events.EventFinal:
		log.Debug().
			Str("text", e.Text).
			Msg("Unwrapped final event")
		eventObj["text"] = e.Text
		
	case *events.EventError:
		eventObj["error"] = e.ErrorString
		
	case *events.EventInterrupt:
		eventObj["text"] = e.Text
		
	case *events.EventToolCall:
		eventObj["toolCall"] = map[string]interface{}{
			"id":    e.ToolCall.ID,
			"name":  e.ToolCall.Name,
			"input": e.ToolCall.Input,
		}
		
	case *events.EventToolResult:
		eventObj["toolResult"] = map[string]interface{}{
			"id":     e.ToolResult.ID,
			"result": e.ToolResult.Result,
		}
	default:
		log.Debug().
			Str("event_type", eventType).
			Str("go_type", fmt.Sprintf("%T", eventsEvent)).
			Msg("Unhandled unwrapped events.Event type")
	}
	
	return s.runtime.ToValue(eventObj)
}

// metadataToJS converts event metadata to JavaScript object
func (s *jsEventStream) metadataToJS(metadata events.EventMetadata) interface{} {
	return map[string]interface{}{
		"messageId": metadata.ID.String(),
		"parentId":  metadata.ParentID.String(),
		"engine":    metadata.Engine,
	}
}

// stepMetadataToJS converts step metadata to JavaScript object
func (s *jsEventStream) stepMetadataToJS(metadata interface{}) interface{} {
	if metadata == nil {
		return nil
	}
	
	// Try to convert to steps.StepMetadata
	if stepMeta, ok := metadata.(*steps.StepMetadata); ok {
		if stepMeta == nil {
			return nil
		}
		return map[string]interface{}{
			"stepId":     stepMeta.StepID.String(),
			"type":       stepMeta.Type,
			"inputType":  stepMeta.InputType,
			"outputType": stepMeta.OutputType,
			"metadata":   stepMeta.Metadata,
		}
	}
	
	return metadata
}

type JSStepWrapper[T any, U any] struct {
	runtime *goja.Runtime
	step    steps.Step[T, U]
	loop    *eventloop.EventLoop
}

// XXX we could convert this to take a list of options (for example for the async functionality, or even for a cache)

// CreateStepObject creates a JavaScript object wrapping a Step with Promise, blocking and callback APIs
func CreateStepObject[T any, U any](
	runtime *goja.Runtime,
	loop *eventloop.EventLoop,
	step steps.Step[T, U],
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) (*goja.Object, error) {
	wrapper := &JSStepWrapper[T, U]{
		runtime: runtime,
		step:    step,
		loop:    loop,
	}

	stepObj := runtime.NewObject()
	var err error
	if loop != nil {
		err = stepObj.Set("startAsync", wrapper.makeStartAsync(inputConverter, outputConverter))
		if err != nil {
			return nil, err
		}
	}
	err = stepObj.Set("startBlocking", wrapper.makeStartBlocking(inputConverter, outputConverter))
	if err != nil {
		return nil, err
	}
	err = stepObj.Set("startWithCallbacks", wrapper.makeStartWithCallbacks(inputConverter, outputConverter))
	if err != nil {
		return nil, err
	}
	
	// Add streaming support
	err = stepObj.Set("startStream", wrapper.makeStartStream(inputConverter, outputConverter))
	if err != nil {
		return nil, err
	}

	return stepObj, nil
}

// RegisterStep is kept for backward compatibility
func RegisterStep[T any, U any](
	runtime *goja.Runtime,
	loop *eventloop.EventLoop,
	name string,
	step steps.Step[T, U],
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) error {
	stepObj, err := CreateStepObject(runtime, loop, step, inputConverter, outputConverter)
	if err != nil {
		return err
	}
	return runtime.Set(name, stepObj)
}

func (w *JSStepWrapper[T, U]) makeStartAsync(
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		log.Debug().Msg("makeStartAsync called")
		if len(call.Arguments) < 1 {
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("startAsync requires an input argument")})
		}

		input := inputConverter(call.Arguments[0])
		ctx := context.Background()

		log.Debug().Interface("input", input).Msg("makeStartAsync called")

		promise, resolve, reject := w.runtime.NewPromise()

		w.loop.RunOnLoop(func(*goja.Runtime) {
			go func() {
				log.Debug().Msg("Starting step")
				result, err := w.step.Start(ctx, input)
				if err != nil {
					log.Debug().Msg("Failed to start step")
					w.loop.RunOnLoop(func(*goja.Runtime) {
						rejectErr := reject(w.runtime.ToValue(fmt.Sprintf("failed to start step: %v", err)))
						if rejectErr != nil {
							log.Error().Err(rejectErr).Msg("failed to reject promise")
						}
					})
					return
				}
				defer result.Cancel()

				results := result.Return()
				if len(results) > 0 {
					log.Debug().Interface("result", results[0]).Msg("Result")
				}
				log.Debug().Array("results", helpers.ToResultSlice(results)).Msg("Results")

				if len(results) == 0 {
					w.loop.RunOnLoop(func(*goja.Runtime) {
						log.Debug().Msg("Resolving promise")
						resolveErr := resolve(w.runtime.ToValue([]interface{}{}))
						if resolveErr != nil {
							log.Error().Err(resolveErr).Msg("failed to resolve promise")
						}
					})
					return
				}

				// Convert results to JS values
				jsResults := make([]goja.Value, len(results))
				var resolveErr error
				for i, r := range results {
					if r.Error() != nil {
						w.loop.RunOnLoop(func(*goja.Runtime) {
							rejectErr := reject(w.runtime.ToValue(r.Error().Error()))
							if rejectErr != nil {
								log.Error().Err(rejectErr).Msg("failed to reject promise")
							}
						})
						return
					}
					jsResults[i] = outputConverter(r.Unwrap())
				}

				// Must resolve on the event loop
				w.loop.RunOnLoop(func(*goja.Runtime) {
					if resolveErr != nil {
						rejectErr := reject(w.runtime.ToValue(resolveErr.Error()))
						if rejectErr != nil {
							log.Error().Err(rejectErr).Msg("failed to reject promise")
						}
						return
					}
					resolveErr = resolve(w.runtime.ToValue(jsResults))
					if resolveErr != nil {
						log.Error().Err(resolveErr).Msg("failed to resolve promise")
					}
				})

				log.Debug().Msg("Done")
			}()
		})

		log.Debug().Msg("Returning promise")

		return w.runtime.ToValue(promise)
	}

}

func (w *JSStepWrapper[T, U]) makeStartBlocking(
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("startBlocking requires an input argument")})
		}

		input := inputConverter(call.Arguments[0])
		ctx := context.Background()
		result, err := w.step.Start(ctx, input)
		if err != nil {
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to start step: %w", err)})
		}
		defer result.Cancel()

		results := result.Return()
		if len(results) == 0 {
			return goja.Undefined()
		}

		// Convert results to JS values
		jsResults := make([]goja.Value, len(results))
		for i, r := range results {
			if r.Error() != nil {
				return w.runtime.ToValue([]interface{}{nil, r.Error()})
			}
			jsResults[i] = outputConverter(r.Unwrap())
		}

		// Return the value directly on success
		return w.runtime.ToValue(jsResults)
	}
}

type CallbackFunctions struct {
	OnResult goja.Callable
	OnError  goja.Callable
	OnDone   goja.Callable
	OnCancel goja.Callable
}

func (w *JSStepWrapper[T, U]) makeStartWithCallbacks(
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("startWithCallbacks requires input and callbacks object")})
		}

		log.Debug().Msg("makeStartWithCallbacks called")

		input := inputConverter(call.Arguments[0])

		// Extract callbacks
		callbacks := CallbackFunctions{}
		callbacksObj := call.Arguments[1].ToObject(w.runtime)

		var ok bool
		if onResult := callbacksObj.Get("onResult"); onResult != nil {
			callbacks.OnResult, ok = goja.AssertFunction(onResult)
			if !ok {
				return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to assert onResult")})
			}
		}
		if onError := callbacksObj.Get("onError"); onError != nil {
			callbacks.OnError, ok = goja.AssertFunction(onError)
			if !ok {
				return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to assert onError")})
			}
		}
		if onDone := callbacksObj.Get("onDone"); onDone != nil {
			callbacks.OnDone, ok = goja.AssertFunction(onDone)
			if !ok {
				return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to assert onDone")})
			}
		}
		if onCancel := callbacksObj.Get("onCancel"); onCancel != nil {
			callbacks.OnCancel, ok = goja.AssertFunction(onCancel)
			if !ok {
				return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to assert onCancel")})
			}
		}

		ctx, cancel := context.WithCancel(context.Background())

		log.Debug().Msg("Starting step")

		result, err := w.step.Start(ctx, input)
		if err != nil {
			cancel()
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to start step: %w", err)})
		}

		log.Debug().Msg("Step started")

		// Return a cancel function to JavaScript
		cancelFn := func(call goja.FunctionCall) goja.Value {
			log.Debug().Msg("Cancelling step")
			cancel()
			result.Cancel()
			if callbacks.OnCancel != nil {
				log.Debug().Msg("Calling onCancel")
				w.loop.RunOnLoop(func(*goja.Runtime) {
					log.Debug().Msg("Calling onCancel")
					_, err := callbacks.OnCancel(goja.Undefined())
					if err != nil {
						log.Error().Err(err).Msg("failed to call onCancel")
					}
					log.Debug().Msg("onCancel called")
				})
			}
			log.Debug().Msg("Done")
			return goja.Undefined()
		}

		w.loop.RunOnLoop(func(*goja.Runtime) {
			go func() {
				defer func() {
					log.Debug().Msg("StepsJS Done")
				}()

				defer func() {
					log.Debug().Msg("Callbacks Done")
					if callbacks.OnDone != nil {
						w.loop.RunOnLoop(func(*goja.Runtime) {
							log.Debug().Msg("Calling onDone")
							_, err := callbacks.OnDone(goja.Undefined())
							if err != nil {
								log.Error().Err(err).Msg("failed to call onDone")
							}
							log.Debug().Msg("onDone called")
						})
					}
				}()

				for {
					select {
					case <-ctx.Done():
						log.Debug().Msg("StepsJS Context done")
						return
					case r, ok := <-result.GetChannel():
						if !ok {
							return
						}
						if r.Error() != nil {
							log.Debug().Msg("Error")
							if callbacks.OnError != nil {
								w.loop.RunOnLoop(func(*goja.Runtime) {
									log.Debug().Msg("Calling onError")
									_, err := callbacks.OnError(goja.Undefined(), w.runtime.ToValue(r.Error().Error()))
									if err != nil {
										log.Error().Err(err).Msg("failed to call onError")
									}
									log.Debug().Msg("onError called")
								})
							}
							continue
						}
						if callbacks.OnResult != nil {
							w.loop.RunOnLoop(func(*goja.Runtime) {
								log.Debug().Msg("Calling onResult")
								_, err := callbacks.OnResult(goja.Undefined(), outputConverter(r.Unwrap()))
								if err != nil {
									log.Error().Err(err).Msg("failed to call onResult")
								}
								log.Debug().Msg("onResult called")
							})
						}
					}
				}

			}()
		})

		return w.runtime.ToValue(cancelFn)
	}
}

// Example usage with embeddings step
func CreateEmbeddingsStepObject(runtime *goja.Runtime, loop *eventloop.EventLoop, provider steps.Step[string, []float32]) (*goja.Object, error) {
	inputConverter := func(v goja.Value) string {
		return v.String()
	}

	outputConverter := func(embedding []float32) goja.Value {
		embeddingInterface := make([]interface{}, len(embedding))
		for i, v := range embedding {
			embeddingInterface[i] = v
		}
		return runtime.ToValue(embeddingInterface)
	}

	return CreateStepObject(runtime, loop, provider, inputConverter, outputConverter)
}

// Helper to create a LambdaStep from a JavaScript function
func NewJSLambdaStep[T any, U any](
	runtime *goja.Runtime,
	fn goja.Callable,
	inputConverter func(T) goja.Value,
	outputConverter func(goja.Value) (U, error),
) steps.Step[T, U] {
	return &utils.LambdaStep[T, U]{
		Function: func(input T) helpers.Result[U] {
			jsInput := inputConverter(input)
			jsResult, err := fn(goja.Undefined(), jsInput)
			if err != nil {
				return helpers.NewErrorResult[U](err)
			}

			result, err := outputConverter(jsResult)
			if err != nil {
				return helpers.NewErrorResult[U](err)
			}

			return helpers.NewValueResult(result)
		},
	}
}

// makeStartStream creates a streaming function that returns an event stream
func (w *JSStepWrapper[T, U]) makeStartStream(
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(w.runtime.NewTypeError("startStream requires an input argument"))
		}

		input := inputConverter(call.Arguments[0])
		ctx := context.Background()

		log.Debug().Interface("input", input).Msg("makeStartStream called")

		// Check if step supports streaming
		if streamableStep, ok := w.step.(interface {
			steps.Step[T, U]
			Stream(context.Context, T) (<-chan steps.Event, func(), error)
		}); ok {
			// Use native streaming
			eventCh, cancel, err := streamableStep.Stream(ctx, input)
			if err != nil {
				panic(w.runtime.NewTypeError(fmt.Sprintf("failed to start streaming step: %v", err)))
			}
			
			return NewJSEventStream(w.runtime, w.loop, eventCh, cancel)
		} else {
			// For non-streaming steps, we need to create a fallback that converts
			// the regular step result into events
			eventCh := make(chan steps.Event, 10)
			ctx, cancel := context.WithCancel(ctx)
			
			go func() {
				defer close(eventCh)
				
				result, err := w.step.Start(ctx, input)
				if err != nil {
					log.Error().Err(err).Msg("Failed to start step")
					return
				}
				defer result.Cancel()
				
				// Send start event
				metadata := events.EventMetadata{}
				stepMetadata := result.GetMetadata()
				startEvent := &eventAdapter{events.NewStartEvent(metadata, stepMetadata)}
				
				select {
				case eventCh <- startEvent:
				case <-ctx.Done():
					return
				}
				
				// Process results
				for stepResult := range result.GetChannel() {
					if stepResult.Error() != nil {
						errorEvent := &eventAdapter{events.NewErrorEvent(metadata, stepMetadata, stepResult.Error())}
						select {
						case eventCh <- errorEvent:
						case <-ctx.Done():
							return
						}
						return
					}
					
					// Convert result using the output converter and create final event
					actualResult, err := stepResult.Value()
					if err != nil {
						errorEvent := &eventAdapter{events.NewErrorEvent(metadata, stepMetadata, err)}
						select {
						case eventCh <- errorEvent:
						case <-ctx.Done():
							return
						}
						return
					}
					
					// Convert the result to string for the final event
					var resultStr string
					if str, ok := any(actualResult).(string); ok {
						resultStr = str
					} else {
						resultStr = fmt.Sprintf("%v", actualResult)
					}
					
					// Simulate streaming by breaking up the response into chunks
					if len(resultStr) > 0 {
						chunkSize := 20 // Simulate reasonable chunk size
						for i := 0; i < len(resultStr); i += chunkSize {
							end := i + chunkSize
							if end > len(resultStr) {
								end = len(resultStr)
							}
							chunk := resultStr[i:end]
							
							partialEvent := &eventAdapter{events.NewPartialCompletionEvent(metadata, stepMetadata, chunk, resultStr[:end])}
							select {
							case eventCh <- partialEvent:
							case <-ctx.Done():
								return
							}
							
							// Small delay to simulate real streaming
							time.Sleep(10 * time.Millisecond)
						}
					}
					
					finalEvent := &eventAdapter{events.NewFinalEvent(metadata, stepMetadata, resultStr)}
					select {
					case eventCh <- finalEvent:
					case <-ctx.Done():
						return
					}
				}
			}()
			
			cancelFunc := func() {
				cancel()
			}
			
			return NewJSEventStream(w.runtime, w.loop, eventCh, cancelFunc)
		}
	}
}
