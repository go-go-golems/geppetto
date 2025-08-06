package middleware

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"

	"github.com/go-go-golems/geppetto/pkg/conversation"
)

// HandlerFunc represents a function that can process an inference request.
// It returns the complete conversation including any intermediate messages.
type HandlerFunc func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error)

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
	return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
		return engine.RunInference(ctx, messages)
	}
}

// EngineWithMiddleware wraps an Engine with a middleware chain.
type EngineWithMiddleware struct {
	handler HandlerFunc
	config  *engine.Config
}

// NewEngineWithMiddleware creates a new engine with middleware support.
func NewEngineWithMiddleware(e engine.Engine, middlewares ...Middleware) *EngineWithMiddleware {
	handler := engineHandlerFunc(e)
	chainedHandler := Chain(handler, middlewares...)

	return &EngineWithMiddleware{
		handler: chainedHandler,
		config:  engine.NewConfig(),
	}
}

// RunInference executes the middleware chain followed by the underlying engine.
// Returns the full updated conversation.
func (e *EngineWithMiddleware) RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
	// TODO(middleware): Add EventSinks to context for middleware access
	// ctx = events.WithSinks(ctx, e.config.EventSinks)

	// Clone messages to prevent mutation issues
	messages = cloneConversation(messages)

	// Execute middleware chain and get complete conversation
	return e.handler(ctx, messages)
}

// RunInferenceWithHistory returns the complete conversation including tool calls.
func (e *EngineWithMiddleware) RunInferenceWithHistory(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
	// TODO(middleware): Add EventSinks to context for middleware access
	// ctx = events.WithSinks(ctx, e.config.EventSinks)
	messages = cloneConversation(messages)
	return e.handler(ctx, messages)
}

// cloneConversation creates a deep copy of a conversation to prevent mutation issues
func cloneConversation(messages conversation.Conversation) conversation.Conversation {
	if messages == nil {
		return nil
	}
	cloned := make(conversation.Conversation, len(messages))
	copy(cloned, messages)
	return cloned
}
