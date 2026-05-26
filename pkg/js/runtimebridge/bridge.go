package runtimebridge

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// Bridge wraps a runtimeowner.RuntimeOwner with helpers tailored for goja callback usage.
type Bridge struct {
	runtimeOwner runtimeowner.RuntimeOwner
}

func New(runtimeOwner runtimeowner.RuntimeOwner) *Bridge {
	if runtimeOwner == nil {
		panic("runtimebridge: runtimeOwner is nil")
	}
	return &Bridge{runtimeOwner: runtimeOwner}
}

func (b *Bridge) RuntimeOwner() runtimeowner.RuntimeOwner {
	return b.runtimeOwner
}

func (b *Bridge) Call(ctx context.Context, op string, fn runtimeowner.CallFunc) (any, error) {
	if b == nil || b.runtimeOwner == nil {
		return nil, fmt.Errorf("runtimebridge %s: nil bridge", op)
	}
	return b.runtimeOwner.Call(ctx, op, fn)
}

func (b *Bridge) Post(ctx context.Context, op string, fn runtimeowner.PostFunc) error {
	if b == nil || b.runtimeOwner == nil {
		return fmt.Errorf("runtimebridge %s: nil bridge", op)
	}
	return b.runtimeOwner.Post(ctx, op, fn)
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
