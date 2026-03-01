package geppetto

import (
	"context"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/events"
)

func newJSEventCollector(api *moduleRuntime) *jsEventCollector {
	return &jsEventCollector{
		api:       api,
		listeners: map[string][]goja.Callable{},
	}
}

func (m *moduleRuntime) eventsCollector(goja.FunctionCall) goja.Value {
	collector := newJSEventCollector(m)
	obj := m.vm.NewObject()
	m.attachRef(obj, collector)

	m.mustSet(obj, "on", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(m.vm.NewTypeError("collector.on(eventType, callback) requires 2 arguments"))
		}
		eventType := call.Arguments[0].String()
		cb, ok := goja.AssertFunction(call.Arguments[1])
		if !ok {
			panic(m.vm.NewTypeError("collector.on() callback must be a function"))
		}
		collector.subscribe(eventType, cb)
		return obj
	})

	m.mustSet(obj, "clear", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 && call.Arguments[0] != nil && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
			collector.clear(call.Arguments[0].String())
		} else {
			collector.clear("*")
		}
		return obj
	})

	m.mustSet(obj, "close", func(goja.FunctionCall) goja.Value {
		collector.close()
		return goja.Undefined()
	})

	return obj
}

var _ events.EventSink = (*jsEventCollector)(nil)

func (c *jsEventCollector) subscribe(eventType string, fn goja.Callable) {
	if c == nil || fn == nil {
		return
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		eventType = "*"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.listeners[eventType] = append(c.listeners[eventType], fn)
}

func (c *jsEventCollector) close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.closed = true
	c.listeners = nil
	c.mu.Unlock()
}

func (c *jsEventCollector) clear(eventType string) {
	if c == nil {
		return
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" || eventType == "*" {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.closed {
			return
		}
		c.listeners = map[string][]goja.Callable{}
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	delete(c.listeners, eventType)
}

func (c *jsEventCollector) PublishEvent(ev events.Event) error {
	if c == nil || ev == nil {
		return nil
	}
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil
	}
	eventType := string(ev.Type())
	callbacks := make([]goja.Callable, 0, len(c.listeners[eventType])+len(c.listeners["*"]))
	callbacks = append(callbacks, c.listeners[eventType]...)
	callbacks = append(callbacks, c.listeners["*"]...)
	c.mu.RUnlock()
	if len(callbacks) == 0 || c.api == nil {
		return nil
	}
	if _, err := c.api.requireBridge("event collector publish"); err != nil {
		c.api.logger.Warn().Err(err).Msg("event collector publish skipped")
		return nil
	}

	payload := c.encodeEventPayload(ev)
	_, err := c.api.callOnOwner(context.Background(), "eventCollector.publish", func(context.Context) (any, error) {
		jsPayload := c.api.toJSValue(payload)
		for _, cb := range callbacks {
			_, _ = cb(goja.Undefined(), jsPayload)
		}
		return nil, nil
	})
	if err != nil {
		c.api.logger.Warn().Err(err).Msg("event collector publish failed")
	}
	return nil
}

func (c *jsEventCollector) encodeEventPayload(ev events.Event) map[string]any {
	meta := ev.Metadata()
	payload := map[string]any{
		"type":        string(ev.Type()),
		"timestampMs": time.Now().UnixMilli(),
	}
	if meta.SessionID != "" {
		payload["sessionId"] = meta.SessionID
	}
	if meta.InferenceID != "" {
		payload["inferenceId"] = meta.InferenceID
	}
	if meta.TurnID != "" {
		payload["turnId"] = meta.TurnID
	}
	if len(meta.Extra) > 0 {
		payload["metaExtra"] = cloneJSONValue(meta.Extra)
	}

	switch e := ev.(type) {
	case *events.EventPartialCompletion:
		payload["delta"] = e.Delta
		payload["completion"] = e.Completion
	case *events.EventThinkingPartial:
		payload["delta"] = e.Delta
		payload["completion"] = e.Completion
	case *events.EventToolCall:
		payload["toolCall"] = map[string]any{
			"id":    e.ToolCall.ID,
			"name":  e.ToolCall.Name,
			"input": e.ToolCall.Input,
		}
	case *events.EventToolResult:
		payload["toolResult"] = map[string]any{
			"id":     e.ToolResult.ID,
			"result": e.ToolResult.Result,
		}
	case *events.EventToolCallExecute:
		payload["toolCall"] = map[string]any{
			"id":    e.ToolCall.ID,
			"name":  e.ToolCall.Name,
			"input": e.ToolCall.Input,
		}
	case *events.EventToolCallExecutionResult:
		payload["toolResult"] = map[string]any{
			"id":     e.ToolResult.ID,
			"result": e.ToolResult.Result,
		}
	case *events.EventFinal:
		payload["text"] = e.Text
	case *events.EventError:
		payload["error"] = e.ErrorString
	case *events.EventInterrupt:
		payload["text"] = e.Text
	}
	if raw := ev.Payload(); len(raw) > 0 {
		payload["rawPayload"] = string(raw)
	}
	return payload
}
