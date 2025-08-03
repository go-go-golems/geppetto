package adapter

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEngine implements the inference.Engine interface for testing
type MockEngine struct {
	result *conversation.Message
	err    error
}

func (m *MockEngine) RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// MockSimpleChatStep implements the SimpleChatStep interface for testing
type MockSimpleChatStep struct {
	result *conversation.Message
	err    error
}

func (m *MockSimpleChatStep) RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestStepAdapter_BasicEngine(t *testing.T) {
	// Create a mock response
	expectedMessage := conversation.NewMessage(
		conversation.NewChatMessageContent(conversation.RoleAssistant, "Hello, world!", nil),
	)

	// Create mock engine
	mockEngine := &MockEngine{
		result: expectedMessage,
	}

	// Create metadata
	metadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "test-adapter",
		InputType:  "conversation.Conversation",
		OutputType: "*conversation.Message",
	}

	// Create adapter
	adapter := NewStepAdapter(mockEngine, metadata)

	// Test that it implements the chat.Step interface
	var _ chat.Step = adapter

	// Create test input
	input := conversation.Conversation{
		conversation.NewMessage(
			conversation.NewChatMessageContent(conversation.RoleUser, "Hello", nil),
		),
	}

	// Test Start method
	ctx := context.Background()
	result, err := adapter.Start(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Get the result
	results := result.Return()
	require.Len(t, results, 1)
	require.NoError(t, results[0].Error())

	actualMessage := results[0].Unwrap()
	assert.Equal(t, expectedMessage, actualMessage)

	// Test metadata
	resultMetadata := result.GetMetadata()
	assert.NotNil(t, resultMetadata)
	assert.Equal(t, metadata.Type, resultMetadata.Type)
}

func TestStepAdapter_SimpleChatStep(t *testing.T) {
	// Create a mock response
	expectedMessage := conversation.NewMessage(
		conversation.NewChatMessageContent(conversation.RoleAssistant, "Hello from SimpleChatStep!", nil),
	)

	// Create mock SimpleChatStep
	mockStep := &MockSimpleChatStep{
		result: expectedMessage,
	}

	// Create metadata
	metadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "test-simple-adapter",
		InputType:  "conversation.Conversation",
		OutputType: "*conversation.Message",
	}

	// Create adapter
	adapter := NewStepAdapter(mockStep, metadata)

	// Create test input
	input := conversation.Conversation{
		conversation.NewMessage(
			conversation.NewChatMessageContent(conversation.RoleUser, "Hello", nil),
		),
	}

	// Test Start method
	ctx := context.Background()
	result, err := adapter.Start(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Get the result
	results := result.Return()
	require.Len(t, results, 1)
	require.NoError(t, results[0].Error())

	actualMessage := results[0].Unwrap()
	assert.Equal(t, expectedMessage, actualMessage)
}

func TestStepAdapter_AddPublishedTopic(t *testing.T) {
	// Create mock engine
	mockEngine := &MockEngine{
		result: conversation.NewMessage(
			conversation.NewChatMessageContent(conversation.RoleAssistant, "Test", nil),
		),
	}

	// Create adapter
	adapter := NewStepAdapter(mockEngine, &steps.StepMetadata{
		StepID: uuid.New(),
		Type:   "test",
	})

	// Test AddPublishedTopic (should not error)
	err := adapter.AddPublishedTopic(nil, "test-topic")
	assert.NoError(t, err)
}

func TestStepEngineAdapter(t *testing.T) {
	// Create a mock SimpleChatStep
	expectedMessage := conversation.NewMessage(
		conversation.NewChatMessageContent(conversation.RoleAssistant, "Engine adapter test", nil),
	)

	mockStep := &MockSimpleChatStep{
		result: expectedMessage,
	}

	// Create engine adapter
	engine := CreateEngineFromStep(mockStep)

	// Test RunInference
	ctx := context.Background()
	input := conversation.Conversation{
		conversation.NewMessage(
			conversation.NewChatMessageContent(conversation.RoleUser, "Test", nil),
		),
	}

	result, err := engine.RunInference(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, expectedMessage, result)
}
