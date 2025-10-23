package events

import (
	"context"
    "github.com/rs/zerolog/log"
)

// ctxKey is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type ctxKey int

const (
	ctxKeyEventSinks ctxKey = iota
)

// WithEventSinks attaches one or more EventSink instances to the context.
// Downstream code can retrieve the sinks and publish events without
// requiring access to engine configuration.
func WithEventSinks(ctx context.Context, sinks ...EventSink) context.Context {
	if len(sinks) == 0 {
		return ctx
	}
	existing := GetEventSinks(ctx)
	combined := append([]EventSink{}, existing...)
	combined = append(combined, sinks...)
	return context.WithValue(ctx, ctxKeyEventSinks, combined)
}

// GetEventSinks returns the list of EventSinks attached to the context.
func GetEventSinks(ctx context.Context) []EventSink {
	if v := ctx.Value(ctxKeyEventSinks); v != nil {
		if sinks, ok := v.([]EventSink); ok {
			return sinks
		}
	}
	return nil
}

// PublishEventToContext publishes the provided event to all EventSinks stored in the context.
// If no sinks are present, this is a no-op.
func PublishEventToContext(ctx context.Context, event Event) {
    sinks := GetEventSinks(ctx)
    if len(sinks) == 0 {
        log.Trace().Str("component", "events.context").Str("event_type", string(event.Type())).Msg("PublishEventToContext: no sinks in context")
        return
    }
    log.Trace().Str("component", "events.context").Str("event_type", string(event.Type())).Int("sink_count", len(sinks)).Msg("PublishEventToContext: publishing to sinks")
    for _, sink := range sinks {
		// Best-effort: ignore individual sink errors to avoid disrupting the flow
		_ = sink.PublishEvent(event)
	}
}
