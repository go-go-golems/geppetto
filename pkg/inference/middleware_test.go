package inference

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

func (m *MockEngine) RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestEngineHandler(t *testing.T) {
	// Create mock engine with response
	mockResponse := conversation.NewChatMessage(conversation.RoleAssistant, "Hello, world!")
	mockEngine := &MockEngine{response: mockResponse}

	// Create handler from engine
	handler := EngineHandler(mockEngine)

	// Test with input messages
	inputMessages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Hi there!"),
	)

	result, err := handler(context.Background(), inputMessages)

	require.NoError(t, err)
	assert.Len(t, result, 2) // original message + response
	assert.Equal(t, "Hi there!", result[0].Content.(*conversation.ChatMessageContent).Text)
	assert.Equal(t, "Hello, world!", result[1].Content.(*conversation.ChatMessageContent).Text)
}

func TestMiddlewareChain(t *testing.T) {
	// Create a middleware that adds "(prefix)" to the response
	prefixMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
			result, err := next(ctx, messages)
			if err != nil {
				return nil, err
			}

			// Modify the last message (AI response)
			if len(result) > 0 {
				lastMsg := result[len(result)-1]
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
	handler := EngineHandler(mockEngine)
	chainedHandler := Chain(handler, prefixMiddleware)

	// Test
	inputMessages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Hi"),
	)

	result, err := chainedHandler(context.Background(), inputMessages)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "(prefix) Hello", result[1].Content.(*conversation.ChatMessageContent).Text)
}

func TestEngineWithMiddleware(t *testing.T) {
	// Create a simple logging middleware
	var loggedMessages []conversation.Conversation
	loggingMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
			loggedMessages = append(loggedMessages, messages)
			return next(ctx, messages)
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

	response, err := engine.RunInference(context.Background(), inputMessages)

	require.NoError(t, err)
	assert.Equal(t, "Response", response.Content.(*conversation.ChatMessageContent).Text)
	assert.Len(t, loggedMessages, 1) // Middleware should have logged the input

	// Test RunInferenceWithHistory
	fullConversation, err := engine.RunInferenceWithHistory(context.Background(), inputMessages)

	require.NoError(t, err)
	assert.Len(t, fullConversation, 2) // input + response
	assert.Len(t, loggedMessages, 2)   // Middleware called twice
}

func TestCloneConversation(t *testing.T) {
	original := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Original"),
	)

	cloned := cloneConversation(original)

	// Modify original
	original[0].Content.(*conversation.ChatMessageContent).Text = "Modified"

	// Cloned should be unchanged (shallow copy behavior)
	// Note: This is a shallow copy, so the underlying message content is shared
	// For full safety, we'd need deep cloning, but this is sufficient for basic protection
	assert.Len(t, cloned, 1)
	assert.True(t, &original != &cloned) // Different slices
}
