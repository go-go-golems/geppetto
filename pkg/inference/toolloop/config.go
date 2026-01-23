package toolloop

// LoopConfig configures tool loop orchestration behavior (not tool policy).
type LoopConfig struct {
	// MaxIterations is the maximum number of loop iterations before aborting.
	// If <= 0, the loop uses a default.
	MaxIterations int
}

// DefaultLoopConfig returns a sensible default loop configuration.
func DefaultLoopConfig() LoopConfig {
	return LoopConfig{
		MaxIterations: 5,
	}
}

// NewLoopConfig creates a default loop configuration.
func NewLoopConfig() LoopConfig {
	return DefaultLoopConfig()
}

func (c LoopConfig) WithMaxIterations(maxIterations int) LoopConfig {
	c.MaxIterations = maxIterations
	return c
}
