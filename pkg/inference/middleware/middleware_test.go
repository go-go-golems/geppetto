package middleware

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEngine implements Engine interface for testing
type MockEngine struct {
	response *conversation.Message
	err      error
}

func (m *MockEngine) RunInference(ctx context.Context, conv conversation.InferenceContext) (conversation.InferenceContext, error) {
	if m.err != nil {
		return conversation.InferenceContext{}, m.err
	}
	result := conversation.InferenceContext{Messages: append(conversation.Conversation(nil), conv.Messages...)}
	result.Messages = append(result.Messages, m.response)
	return result, nil
}

func TestEngineHandler(t *testing.T) {
	// Create mock engine with response
	mockResponse := conversation.NewChatMessage(conversation.RoleAssistant, "Hello, world!")
	mockEngine := &MockEngine{response: mockResponse}

	// Create handler from engine
	handler := engineHandlerFunc(mockEngine)

	// Test with input messages
	inputMessages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Hi there!"),
	)
	conv := conversation.NewInferenceContext(inputMessages)

	result, err := handler(context.Background(), conv)

	require.NoError(t, err)
	assert.Len(t, result.Messages, 2)
	assert.Equal(t, "Hi there!", result.Messages[0].Content.(*conversation.ChatMessageContent).Text)
	assert.Equal(t, "Hello, world!", result.Messages[1].Content.(*conversation.ChatMessageContent).Text)
}

func TestMiddlewareChain(t *testing.T) {
	// Create a middleware that adds "(prefix)" to the response
	prefixMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, conv conversation.InferenceContext) (conversation.InferenceContext, error) {
			result, err := next(ctx, conv)
			if err != nil {
				return conversation.InferenceContext{}, err
			}
			if len(result.Messages) > 0 {
				lastMsg := result.Messages[len(result.Messages)-1]
				if content, ok := lastMsg.Content.(*conversation.ChatMessageContent); ok {
					content.Text = "(prefix) " + content.Text
				}
			}
			return result, nil
		}
	}

	// Create mock engine
	mockResponse := conversation.NewChatMessage(conversation.RoleAssistant, "Hello")
	mockEngine := &MockEngine{response: mockResponse}

	// Create middleware chain
	handler := engineHandlerFunc(mockEngine)
	chainedHandler := Chain(handler, prefixMiddleware)

	// Test
	inputMessages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Hi"),
	)
	conv := conversation.NewInferenceContext(inputMessages)

	result, err := chainedHandler(context.Background(), conv)

	require.NoError(t, err)
	assert.Len(t, result.Messages, 2)
	assert.Equal(t, "(prefix) Hello", result.Messages[1].Content.(*conversation.ChatMessageContent).Text)
}

func TestEngineWithMiddleware(t *testing.T) {
	// Create a simple logging middleware
	var logged []conversation.InferenceContext
	loggingMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, conv conversation.InferenceContext) (conversation.InferenceContext, error) {
			logged = append(logged, conv)
			return next(ctx, conv)
		}
	}

	// Create mock engine
	mockResponse := conversation.NewChatMessage(conversation.RoleAssistant, "Response")
	mockEngine := &MockEngine{response: mockResponse}

	// Create engine with middleware
	engine := NewEngineWithMiddleware(mockEngine, loggingMiddleware)

	// Test RunInference
	inputMessages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Test"),
	)
	conv := conversation.NewInferenceContext(inputMessages)

	response, err := engine.RunInference(context.Background(), conv)

	require.NoError(t, err)
	require.Len(t, response.Messages, 2)
	lastMessage := response.Messages[len(response.Messages)-1]
	assert.Equal(t, "Response", lastMessage.Content.(*conversation.ChatMessageContent).Text)
	assert.Len(t, logged, 1)

	fullConversation, err := engine.RunInferenceWithHistory(context.Background(), conv)

	require.NoError(t, err)
	assert.Len(t, fullConversation.Messages, 2)
	assert.Len(t, logged, 2)
}
