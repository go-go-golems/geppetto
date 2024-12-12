package helpers

import (
	"fmt"

	"github.com/rs/zerolog"
)

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

func (r Result[T]) String() string {
	if r.err != nil {
		return fmt.Sprintf("Result{error: %v}", r.err)
	}
	return fmt.Sprintf("Result{value: %v}", r.value)
}

// MarshalZerologObject implements zerolog.LogObjectMarshaler
func (r Result[T]) MarshalZerologObject(e *zerolog.Event) {
	if r.err != nil {
		e.Err(r.err)
	} else {
		e.Interface("value", r.value)
	}
}

// ResultSlice is a slice of Results that can be marshaled by zerolog
type ResultSlice[T any] []Result[T]

// MarshalZerologArray implements zerolog.LogArrayMarshaler
func (rs ResultSlice[T]) MarshalZerologArray(a *zerolog.Array) {
	for _, r := range rs {
		a.Object(&r)
	}
}

// ToResultSlice converts a slice of Results to a ResultSlice for logging
func ToResultSlice[T any](results []Result[T]) ResultSlice[T] {
	return ResultSlice[T](results)
}
