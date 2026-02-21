package runtimebridge

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// Bridge wraps a runtimeowner.Runner with helpers tailored for goja callback usage.
type Bridge struct {
	runner runtimeowner.Runner
}

func New(runner runtimeowner.Runner) *Bridge {
	if runner == nil {
		panic("runtimebridge: runner is nil")
	}
	return &Bridge{runner: runner}
}

func (b *Bridge) Runner() runtimeowner.Runner {
	return b.runner
}

func (b *Bridge) Call(ctx context.Context, op string, fn runtimeowner.CallFunc) (any, error) {
	if b == nil || b.runner == nil {
		return nil, fmt.Errorf("runtimebridge %s: nil bridge", op)
	}
	return b.runner.Call(ctx, op, fn)
}

func (b *Bridge) Post(ctx context.Context, op string, fn runtimeowner.PostFunc) error {
	if b == nil || b.runner == nil {
		return fmt.Errorf("runtimebridge %s: nil bridge", op)
	}
	return b.runner.Post(ctx, op, fn)
}

func (b *Bridge) InvokeCallable(ctx context.Context, op string, fn goja.Callable, this goja.Value, args ...goja.Value) (goja.Value, error) {
	if fn == nil {
		return nil, fmt.Errorf("runtimebridge %s: nil callable", op)
	}
	ret, err := b.Call(ctx, op, func(_ context.Context, _ *goja.Runtime) (any, error) {
		v, callErr := fn(this, args...)
		if callErr != nil {
			return nil, callErr
		}
		return v, nil
	})
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return nil, nil
	}
	v, ok := ret.(goja.Value)
	if !ok {
		return nil, fmt.Errorf("runtimebridge %s: expected goja.Value, got %T", op, ret)
	}
	return v, nil
}

// ToJSValue executes conversion on the runtime owner thread.
func (b *Bridge) ToJSValue(ctx context.Context, op string, convert func(*goja.Runtime) goja.Value) (goja.Value, error) {
	if convert == nil {
		return nil, fmt.Errorf("runtimebridge %s: nil converter", op)
	}
	ret, err := b.Call(ctx, op, func(_ context.Context, vm *goja.Runtime) (any, error) {
		return convert(vm), nil
	})
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return nil, nil
	}
	v, ok := ret.(goja.Value)
	if !ok {
		return nil, fmt.Errorf("runtimebridge %s: expected goja.Value, got %T", op, ret)
	}
	return v, nil
}
