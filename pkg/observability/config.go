package observability

import (
	"fmt"
	"strings"
)

// TraceLevel controls how much inference/provider evidence is emitted through
// Observer records. The first implementation intentionally supports decoded
// provider objects rather than raw stream strings.
type TraceLevel string

const (
	TraceOff      TraceLevel = "off"
	TraceEvents   TraceLevel = "events"
	TraceProvider TraceLevel = "provider"
)

// Config controls Geppetto observer emission.
type Config struct {
	Level TraceLevel
}

func DefaultConfig() Config {
	return Config{Level: TraceOff}
}

func (c Config) Normalized() Config {
	if c.Level == "" {
		c.Level = TraceOff
	}
	return c
}

func (c Config) Enabled() bool {
	return c.Normalized().Level != TraceOff
}

func (c Config) RecordsEvents() bool {
	switch c.Normalized().Level {
	case TraceEvents, TraceProvider:
		return true
	case TraceOff:
		return false
	default:
		return false
	}
}

func (c Config) RecordsProvider() bool {
	return c.Normalized().Level == TraceProvider
}

func ParseTraceLevel(s string) (TraceLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off":
		return TraceOff, nil
	case "events":
		return TraceEvents, nil
	case "provider":
		return TraceProvider, nil
	case "raw":
		return "", fmt.Errorf("invalid geppetto trace level %q: raw stream capture is reserved for a future implementation; use off, events, or provider", s)
	default:
		return "", fmt.Errorf("invalid geppetto trace level %q: expected off, events, or provider", s)
	}
}
