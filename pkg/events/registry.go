package events

import (
    "encoding/json"
    "fmt"
    "sync"
)

// EventCodec decodes a JSON payload into a concrete Event instance.
type EventCodec func([]byte) (Event, error)

// EventEncoder encodes a concrete Event to JSON.
type EventEncoder func(Event) ([]byte, error)

var (
    registryOnce sync.Once
    reg          *eventRegistry
)

type eventRegistry struct {
    mu       sync.RWMutex
    decoders map[string]EventCodec
    encoders map[string]EventEncoder
}

func ensureRegistry() {
    registryOnce.Do(func() {
        reg = &eventRegistry{
            decoders: make(map[string]EventCodec),
            encoders: make(map[string]EventEncoder),
        }
    })
}

// RegisterEventCodec registers a decoder for a custom event type name.
// It returns an error if a decoder is already registered for the type.
func RegisterEventCodec(typeName string, dec EventCodec) error {
    ensureRegistry()
    reg.mu.Lock()
    defer reg.mu.Unlock()
    if _, exists := reg.decoders[typeName]; exists {
        return fmt.Errorf("decoder already registered for type %q", typeName)
    }
    reg.decoders[typeName] = dec
    return nil
}

// RegisterEventEncoder registers an encoder for a custom event type name.
func RegisterEventEncoder(typeName string, enc EventEncoder) error {
    ensureRegistry()
    reg.mu.Lock()
    defer reg.mu.Unlock()
    if _, exists := reg.encoders[typeName]; exists {
        return fmt.Errorf("encoder already registered for type %q", typeName)
    }
    reg.encoders[typeName] = enc
    return nil
}

// RegisterEventFactory registers a factory based on standard json.Unmarshal.
// The factory must return a zero-value concrete struct implementing Event with Type_ set.
func RegisterEventFactory(typeName string, factory func() Event) error {
    return RegisterEventCodec(typeName, func(b []byte) (Event, error) {
        ev := factory()
        if err := json.Unmarshal(b, &ev); err != nil {
            return nil, err
        }
        return ev, nil
    })
}

func lookupDecoder(typeName string) EventCodec {
    ensureRegistry()
    reg.mu.RLock()
    defer reg.mu.RUnlock()
    return reg.decoders[typeName]
}


