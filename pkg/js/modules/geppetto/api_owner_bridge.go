package geppetto

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/js/runtimebridge"
)

func (m *moduleRuntime) requireBridge(op string) (*runtimebridge.Bridge, error) {
	if m == nil || m.bridge == nil {
		return nil, fmt.Errorf("%s requires module options Runner to be configured", op)
	}
	return m.bridge, nil
}

func (m *moduleRuntime) callOnOwner(ctx context.Context, op string, fn func(context.Context) (any, error)) (any, error) {
	if fn == nil {
		return nil, fmt.Errorf("%s: owner callback is nil", op)
	}
	bridge, err := m.requireBridge(op)
	if err != nil {
		return nil, err
	}
	return bridge.Call(ctx, op, func(callCtx context.Context, _ *goja.Runtime) (any, error) {
		return fn(callCtx)
	})
}

func (m *moduleRuntime) postOnOwner(ctx context.Context, op string, fn func(context.Context)) error {
	if fn == nil {
		return fmt.Errorf("%s: owner callback is nil", op)
	}
	bridge, err := m.requireBridge(op)
	if err != nil {
		return err
	}
	return bridge.Post(ctx, op, func(callCtx context.Context, _ *goja.Runtime) {
		fn(callCtx)
	})
}
