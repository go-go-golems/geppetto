package chat

import (
	"context"
	context2 "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/pkg/errors"
)

type EchoStep struct {
}

func (e EchoStep) SetStreaming(b bool) {
}

func (e EchoStep) Start(ctx context.Context, input []*context2.Message) (*steps.StepResult[string], error) {
	if len(input) == 0 {
		return nil, errors.New("no input")
	}
	return steps.Resolve[string](input[len(input)-1].Text), nil
}

func (e EchoStep) Close(ctx context.Context) error {
	return nil
}

var _ Step = (*EchoStep)(nil)
