package utils

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

type ChainStep[T any, U any, V any] struct {
	StepFactoryA steps.StepFactory[T, U]
	StepFactoryB steps.StepFactory[U, V]
}

func (c *ChainStep[T, U, V]) Start(ctx context.Context, input T) (steps.StepResult[V], error) {
	stepA, err := c.StepFactoryA.NewStep()
	if err != nil {
		return nil, err
	}
	stepB, err := c.StepFactoryB.NewStep()
	if err != nil {
		return nil, err
	}
	v, err := stepA.Start(ctx, input)
	if err != nil {
		return nil, err
	}

	m := steps.Bind[U, V](ctx, v, stepB)
	return m, nil
}

func (r *ChainStep[T, U, V]) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

var _ steps.Step[string, float64] = &ChainStep[string, int64, float64]{}
