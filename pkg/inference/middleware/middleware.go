package middleware

import (
	"context"
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
