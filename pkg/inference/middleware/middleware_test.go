package middleware

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/require"
)

// MockEngine implements Engine interface for testing (Turn-based)
type MockEngine struct {
	response *turns.Block
	err      error
}

func (m *MockEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		turns.AppendBlock(t, *m.response)
	}
	return t, nil
}

func TestEngineHandler(t *testing.T) {
	// Create mock engine with LLM text response
	mockResponse := turns.NewAssistantTextBlock("Hello, world!")
	mockEngine := &MockEngine{response: &mockResponse}

	// Create handler from engine
	handler := engineHandlerFunc(mockEngine)

	// Seed a Turn with a user message
	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewUserTextBlock("Hi there!"))

	result, err := handler(context.Background(), seed)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.GreaterOrEqual(t, len(result.Blocks), 2)
}

func TestMiddlewareChain(t *testing.T) {
	// Middleware that uppercases assistant llm_text blocks
	uppercase := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
			res, err := next(ctx, t)
			if err != nil {
				return nil, err
			}
			for i := range res.Blocks {
				if res.Blocks[i].Kind == turns.BlockKindLLMText {
					if s, ok := res.Blocks[i].Payload[turns.PayloadKeyText].(string); ok {
						res.Blocks[i].Payload[turns.PayloadKeyText] = "(prefix) " + s
					}
				}
			}
			return res, nil
		}
	}

	mockResponse := turns.NewAssistantTextBlock("Hello")
	mockEngine := &MockEngine{response: &mockResponse}

	handler := engineHandlerFunc(mockEngine)
	chained := Chain(handler, uppercase)

	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewUserTextBlock("Hi"))

	res, err := chained(context.Background(), seed)

	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestEngineWithMiddleware(t *testing.T) {
	// Logging middleware that records block counts
	var counts []int
	loggingMw := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
			counts = append(counts, len(t.Blocks))
			return next(ctx, t)
		}
	}

	mockResponse := turns.NewAssistantTextBlock("Response")
	mockEngine := &MockEngine{response: &mockResponse}

	e := NewEngineWithMiddleware(mockEngine, loggingMw)

	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewUserTextBlock("Test"))

	res, err := e.RunInference(context.Background(), seed)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.GreaterOrEqual(t, len(counts), 1)
}

// Removed cloneConversation test for Conversation; Turn-based path no longer needs it.
