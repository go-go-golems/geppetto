package helpers

type Nothing struct{}

type Result[T any] struct {
	value T
	err   error
}

func NewResult[T any](value T, err error) Result[T] {
	return Result[T]{
		value: value,
		err:   err,
	}
}

func NewValueResult[T any](value T) Result[T] {
	return Result[T]{
		value: value,
	}
}

func NewErrorResult[T any](err error) Result[T] {
	return Result[T]{
		err: err,
	}
}

func (r Result[T]) Value() (T, error) {
	return r.value, r.err
}

func (r Result[T]) Error() error {
	return r.err
}

func (r Result[T]) Ok() bool {
	return r.err == nil
}

func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.value
}

func (r Result[T]) ValueOr(v T) T {
	if r.err != nil {
		return v
	}
	return r.value
}
