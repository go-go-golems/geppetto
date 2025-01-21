package chat

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockStep(t *testing.T) {
	// Create mock messages that will be returned in sequence
	messages := []*conversation.Message{
		conversation.NewChatMessage(conversation.RoleAssistant, "First"),
		conversation.NewChatMessage(conversation.RoleAssistant, "Second"),
		conversation.NewChatMessage(conversation.RoleAssistant, "Third"),
	}

	// Create mock step
	mockStep := NewMockStep(messages)

	// Test context
	ctx := context.Background()
	input := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Test input"),
	)

	// First call should get "First"
	result1, err := mockStep.Start(ctx, input)
	require.NoError(t, err)

	var responses1 []string
	for r := range result1.GetChannel() {
		require.NoError(t, r.Error())
		msg := r.Unwrap()
		content := msg.Content.(*conversation.ChatMessageContent)
		responses1 = append(responses1, content.Text)
	}
	assert.Equal(t, []string{"First"}, responses1)

	// Second call should get "Second"
	result2, err := mockStep.Start(ctx, input)
	require.NoError(t, err)

	var responses2 []string
	for r := range result2.GetChannel() {
		require.NoError(t, r.Error())
		msg := r.Unwrap()
		content := msg.Content.(*conversation.ChatMessageContent)
		responses2 = append(responses2, content.Text)
	}
	assert.Equal(t, []string{"Second"}, responses2)

	// Third call should get "Third"
	result3, err := mockStep.Start(ctx, input)
	require.NoError(t, err)

	var responses3 []string
	for r := range result3.GetChannel() {
		require.NoError(t, r.Error())
		msg := r.Unwrap()
		content := msg.Content.(*conversation.ChatMessageContent)
		responses3 = append(responses3, content.Text)
	}
	assert.Equal(t, []string{"Third"}, responses3)

	// Fourth call should wrap around to "First"
	result4, err := mockStep.Start(ctx, input)
	require.NoError(t, err)

	var responses4 []string
	for r := range result4.GetChannel() {
		require.NoError(t, r.Error())
		msg := r.Unwrap()
		content := msg.Content.(*conversation.ChatMessageContent)
		responses4 = append(responses4, content.Text)
	}
	assert.Equal(t, []string{"First"}, responses4)
}

func TestMockStepEmpty(t *testing.T) {
	// Create mock step with no messages
	mockStep := NewMockStep(nil)

	// Test context
	ctx := context.Background()
	input := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Test input"),
	)

	// Call should return empty result
	result, err := mockStep.Start(ctx, input)
	require.NoError(t, err)

	var responses []string
	for r := range result.GetChannel() {
		require.NoError(t, r.Error())
		msg := r.Unwrap()
		content := msg.Content.(*conversation.ChatMessageContent)
		responses = append(responses, content.Text)
	}
	assert.Empty(t, responses)
}

func TestMockStepConcurrent(t *testing.T) {
	// Create mock messages
	messages := []*conversation.Message{
		conversation.NewChatMessage(conversation.RoleAssistant, "First"),
		conversation.NewChatMessage(conversation.RoleAssistant, "Second"),
		conversation.NewChatMessage(conversation.RoleAssistant, "Third"),
	}

	// Create mock step
	mockStep := NewMockStep(messages)

	// Test context
	ctx := context.Background()
	input := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Test input"),
	)

	// Run multiple goroutines to test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			result, err := mockStep.Start(ctx, input)
			require.NoError(t, err)

			var responses []string
			for r := range result.GetChannel() {
				require.NoError(t, r.Error())
				msg := r.Unwrap()
				content := msg.Content.(*conversation.ChatMessageContent)
				responses = append(responses, content.Text)
			}
			assert.Len(t, responses, 1)
			done <- true
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}
}
