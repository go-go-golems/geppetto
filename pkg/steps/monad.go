package steps

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/rs/zerolog/log"
)

type Monad[T any] struct {
	value <-chan helpers.Result[T]
}

func NewMonad[T any](value <-chan helpers.Result[T]) *Monad[T] {
	return &Monad[T]{value: value}
}

func (m *Monad[T]) Return() []helpers.Result[T] {
	res := []helpers.Result[T]{}
	for r := range m.value {
		res = append(res, r)
	}
	return res
}

func (m *Monad[T]) GetChannel() <-chan helpers.Result[T] {
	return m.value
}

func Resolve[T any](value T) *Monad[T] {
	c := make(chan helpers.Result[T], 1)
	c <- helpers.NewValueResult[T](value)
	close(c)
	return &Monad[T]{
		value: c,
	}
}

func ResolveNone[T any]() *Monad[T] {
	c := make(chan helpers.Result[T], 1)
	close(c)
	return &Monad[T]{
		value: c,
	}
}

func Reject[T any](err error) *Monad[T] {
	c := make(chan helpers.Result[T], 1)
	c <- helpers.NewErrorResult[T](err)
	close(c)
	return &Monad[T]{
		value: c,
	}
}

// Transformer is the generalization of a lambda function, with cancellation and closing
// to allow it to own resources.
type Transformer[T any, U any] interface {
	Start(ctx context.Context, input T) (*Monad[U], error)
	Close(ctx context.Context) error
}

func Bind[T any, U any](ctx context.Context, m *Monad[T], transformer Transformer[T, U],
) *Monad[U] {
	return &Monad[U]{
		value: func() <-chan helpers.Result[U] {
			c := make(chan helpers.Result[U])
			go func() {
				defer close(c)
				defer func(transformer Transformer[T, U], ctx context.Context) {
					err := transformer.Close(ctx)
					if err != nil {
						log.Error().Err(err).Msg("error closing transformer")
					}
				}(transformer, ctx)
				for {
					select {
					case r, ok := <-m.value:
						if !ok {
							return
						}
						if r.Error() != nil {
							// we do need to drain m here
							c <- helpers.NewErrorResult[U](r.Error())
							continue
						}

						c_, err := transformer.Start(ctx, r.Unwrap())
						if err != nil {
							c <- helpers.NewErrorResult[U](err)
							return
						}
						for u := range c_.value {
							c <- u
						}
					case <-ctx.Done():
						return
					}
				}
			}()
			return c
		}(),
	}
}
