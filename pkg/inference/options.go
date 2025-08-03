package inference

// Option is a functional option for configuring inference components.
type Option func(*Config) error

// Config holds configuration for inference engines and event publishing.
type Config struct {
	// EventSinks holds all registered event sinks for publishing inference events.
	// Events are published to all sinks in the order they were added.
	EventSinks []EventSink
}

// NewConfig creates a new configuration with default values.
func NewConfig() *Config {
	return &Config{
		EventSinks: make([]EventSink, 0),
	}
}

// WithSink adds an EventSink to the configuration.
// Multiple sinks can be added and events will be published to all of them.
func WithSink(sink EventSink) Option {
	return func(c *Config) error {
		c.EventSinks = append(c.EventSinks, sink)
		return nil
	}
}

// ApplyOptions applies a set of options to a configuration.
func ApplyOptions(config *Config, options ...Option) error {
	for _, option := range options {
		if err := option(config); err != nil {
			return err
		}
	}
	return nil
}
