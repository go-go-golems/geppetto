package middleware

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// HandlerFunc represents a function that processes a Turn.
// It returns the updated Turn, possibly mutating the input.
type HandlerFunc func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)

// Middleware wraps a HandlerFunc with additional functionality.
// Middleware are applied in order: Chain(m1, m2, m3) results in m1(m2(m3(handler))).
type Middleware func(HandlerFunc) HandlerFunc

// Chain composes multiple middleware into a single HandlerFunc.
func Chain(handler HandlerFunc, middlewares ...Middleware) HandlerFunc {
	// Apply middlewares in reverse order so they execute in correct order
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// engineHandlerFunc adapts an Engine to HandlerFunc interface.
// Since engines now return full conversations, this is a simple wrapper.
func engineHandlerFunc(engine engine.Engine) HandlerFunc {
	return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
		return engine.RunInference(ctx, t)
	}
}

// EngineWithMiddleware wraps an Engine with a middleware chain.
type EngineWithMiddleware struct {
	handler HandlerFunc
}

// NewEngineWithMiddleware creates a new engine with middleware support.
func NewEngineWithMiddleware(e engine.Engine, middlewares ...Middleware) *EngineWithMiddleware {
	handler := engineHandlerFunc(e)
	chainedHandler := Chain(handler, middlewares...)

	return &EngineWithMiddleware{
		handler: chainedHandler,
	}
}

// RunInference executes the middleware chain followed by the underlying engine.
// Returns the full updated conversation.
func (e *EngineWithMiddleware) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	// Execute middleware chain on the provided turn
	return e.handler(ctx, t)
}

// RunInferenceWithHistory returns the complete conversation including tool calls.
func (e *EngineWithMiddleware) RunInferenceWithHistory(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	return e.handler(ctx, t)
}
