package toolloop

import "github.com/go-go-golems/geppetto/pkg/turns"

// LoopConfig configures tool loop orchestration behavior (not tool policy).
type LoopConfig struct {
	// MaxIterations is the maximum number of loop iterations before aborting.
	// If <= 0, the loop uses a default.
	MaxIterations int

	// FinalAnswer, when set, asks the model for one more response, with tools
	// disabled, when the loop reaches MaxIterations with tool calls still
	// pending, so a run that used all its rounds still ends with an answer.
	// The loop then returns a *MaxIterationsError with Answered set.
	FinalAnswer *FinalAnswer
}

// FinalAnswer configures the extra response at the iteration limit.
type FinalAnswer struct {
	// Prompt, when set, builds a block that is appended to the turn before the
	// final call (for example a user block saying "no more tools, answer now").
	// The caller owns the block, including its ID.
	Prompt func(maxIterations int) turns.Block

	// RefusedToolResult is the error text recorded for tool calls the model
	// makes anyway in the final response (engines that ignore tool choice
	// "none"); the calls are not executed. Empty uses a default.
	RefusedToolResult string
}

// DefaultRefusedToolResult is the tool error recorded for calls in the final answer.
const DefaultRefusedToolResult = "no more tool calls in this turn; answer without tools"

// DefaultLoopConfig returns a sensible default loop configuration.
func DefaultLoopConfig() LoopConfig {
	return LoopConfig{
		MaxIterations: 20,
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

// WithFinalAnswer enables the final answer at the iteration limit.
func (c LoopConfig) WithFinalAnswer(f FinalAnswer) LoopConfig {
	c.FinalAnswer = &f
	return c
}
