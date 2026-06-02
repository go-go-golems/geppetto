package geppetto

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
)

type jsEventEmitterSink struct {
	api *moduleRuntime
	ref *jsevents.EmitterRef

	closed atomic.Bool
}

var _ events.EventSink = (*jsEventEmitterSink)(nil)

func (m *moduleRuntime) newEventEmitterSinkFromValue(v goja.Value) (*jsEventEmitterSink, error) {
	if m == nil {
		return nil, fmt.Errorf("geppetto events: nil module runtime")
	}
	manager, err := m.getEventEmitterManager()
	if err != nil {
		return nil, err
	}
	ref, err := manager.AdoptEmitterOnOwner(v)
	if err != nil {
		return nil, err
	}
	return &jsEventEmitterSink{api: m, ref: ref}, nil
}

func (m *moduleRuntime) getEventEmitterManager() (*jsevents.Manager, error) {
	if m == nil {
		return nil, fmt.Errorf("geppetto events: nil module runtime")
	}
	if m.eventEmitterManager != nil {
		return m.eventEmitterManager, nil
	}
	if m.eventEmitterManagerResolver != nil {
		if manager, ok := m.eventEmitterManagerResolver(); ok && manager != nil {
			return manager, nil
		}
	}
	return nil, fmt.Errorf("geppetto events: jsevents manager is not installed")
}

func (s *jsEventEmitterSink) PublishEvent(ev events.Event) error {
	if s == nil || ev == nil || s.closed.Load() {
		return nil
	}
	if s.ref == nil {
		return fmt.Errorf("geppetto events: nil EventEmitter reference")
	}
	payload := encodeGeppettoEventPayload(ev)
	for _, name := range eventEmitterNamesForPayload(payload) {
		name := name
		payloadCopy := cloneJSONMap(payload)
		if err := s.ref.EmitWithBuilder(context.Background(), name, func(vm *goja.Runtime) ([]goja.Value, error) {
			return []goja.Value{toJSValueOn(vm, payloadCopy)}, nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *jsEventEmitterSink) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.closed.Swap(true) {
		return nil
	}
	if s.ref == nil {
		return nil
	}
	return s.ref.Close(ctx)
}
