package toolloop

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

// ErrMaxIterations matches (errors.Is) the error RunLoop returns when the loop
// reached LoopConfig.MaxIterations with tool calls still pending.
var ErrMaxIterations = errors.New("max iterations reached")

// MaxIterationsError is the error RunLoop returns at the iteration limit. The
// returned turn holds the conversation so far, including the final answer
// when one was asked for (LoopConfig.FinalAnswer) and given.
//
// When the final answer itself fails (provider error, cancellation, a round
// guard), RunLoop returns that error wrapped together with this one, so both
// errors.Is(err, ErrMaxIterations) and errors.Is(err, <the cause>) hold and
// errors.As finds a MaxIterationsError with Answered false.
type MaxIterationsError struct {
	MaxIterations int
	// Answered is true when the model gave the final answer without tools.
	Answered bool
}

func (e *MaxIterationsError) Error() string {
	return fmt.Sprintf("max iterations (%d) reached", e.MaxIterations)
}

func (e *MaxIterationsError) Is(target error) bool { return target == ErrMaxIterations }

// RoundGuard is called before every model call of the loop, including the
// final answer at the iteration limit; round counts from 0. A non-nil error
// stops the loop before that call, and RunLoop returns the turn so far with
// the guard's error (unwrapped in the rounds before the limit, so callers can
// match their own sentinel errors). Typical use: a token or cost budget.
type RoundGuard func(ctx context.Context, t *turns.Turn, round int) error
