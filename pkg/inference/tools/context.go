package tools

import "context"

// ctxKey is an unexported key type to avoid collisions in context values.
type ctxKey struct{}

// WithRegistry attaches a ToolRegistry to the context for downstream engines/middleware/executors.
//
// This is intended to replace storing runtime registries in Turn.Data.
func WithRegistry(ctx context.Context, reg ToolRegistry) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if reg == nil {
		// Keep semantics: "no tools" by not setting the key at all.
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, reg)
}

// RegistryFrom extracts the ToolRegistry from context.
func RegistryFrom(ctx context.Context) (ToolRegistry, bool) {
	if ctx == nil {
		return nil, false
	}
	reg, ok := ctx.Value(ctxKey{}).(ToolRegistry)
	if !ok || reg == nil {
		return nil, false
	}
	return reg, true
}
