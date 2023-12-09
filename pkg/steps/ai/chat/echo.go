package chat

import (
	"context"
	context2 "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"time"
)

type EchoStep struct {
	TimePerCharacter time.Duration
	OnPartial        func(string) error
	cancel           context.CancelFunc
	eg               *errgroup.Group
}

func (e *EchoStep) SetOnPartial(f func(string) error) {
	e.OnPartial = f
}

func (e *EchoStep) SetStreaming(b bool) {
}

func (e *EchoStep) Start(ctx context.Context, input []*context2.Message) (steps.StepResult[string], error) {
	if len(input) == 0 {
		return nil, errors.New("no input")
	}

	eg, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	c := make(chan helpers.Result[string], 1)
	res := steps.NewStepResult(c)

	eg.Go(func() error {
		defer close(c)
		msg := input[len(input)-1]
		for _, c_ := range msg.Text {
			select {
			case <-ctx.Done():
				c <- helpers.NewErrorResult[string](ctx.Err())
				return ctx.Err()
			case <-time.After(e.TimePerCharacter):
				if e.OnPartial != nil {
					if err := e.OnPartial(string(c_)); err != nil {
						c <- helpers.NewErrorResult[string](err)
						return err
					}
				}
			}
		}
		c <- helpers.NewValueResult[string](msg.Text)
		return nil
	})
	e.eg = eg

	return res, nil
}

var _ Step = (*EchoStep)(nil)
