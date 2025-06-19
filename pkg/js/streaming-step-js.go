package js

import (
	"context"
	"fmt"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/steps"
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
	
	logger := log.With().
		Str("event_type", eventType).
		Str("go_type", fmt.Sprintf("%T", event)).
		Logger()
	
	logger.Debug().Msg("Emitting event to listeners")
	
	s.mu.RLock()
	listeners := make([]goja.Callable, len(s.listeners[eventType]))
	copy(listeners, s.listeners[eventType])
	listenerCount := len(listeners)
	s.mu.RUnlock()
	
	logger.Debug().
		Int("listener_count", listenerCount).
		Msg("Found listeners for event")
	
	// Call listeners on the event loop
	for i, listener := range listeners {
		listener := listener // capture for closure
		listenerIndex := i   // capture for closure
		s.loop.RunOnLoop(func(*goja.Runtime) {
			logger.Debug().
				Int("listener_index", listenerIndex).
				Msg("Calling event listener")
			
			_, err := listener(goja.Undefined(), jsEvent)
			if err != nil {
				logger.Error().Err(err).
					Int("listener_index", listenerIndex).
					Msg("Error calling event listener")
			} else {
				logger.Debug().
					Int("listener_index", listenerIndex).
					Msg("Successfully called event listener")
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
