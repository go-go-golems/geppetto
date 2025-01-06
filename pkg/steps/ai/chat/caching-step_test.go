package chat

import (
	"context"
	"os"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachingStep(t *testing.T) {
	// Create a temporary directory for the cache
	tempDir, err := os.MkdirTemp("", "caching-step-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create mock messages
	messages := []*conversation.Message{
		conversation.NewChatMessage(conversation.RoleAssistant, "Hello"),
		conversation.NewChatMessage(conversation.RoleAssistant, "World"),
		conversation.NewChatMessage(conversation.RoleAssistant, "!"),
	}

	// Create mock step
	mockStep := NewMockStep(messages)

	// Create caching step
	cachingStep, err := NewCachingStep(mockStep, WithCacheDirectory(tempDir))
	require.NoError(t, err)

	// Create test input
	input := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Test input"),
	)

	// First call should use the mock step
	result1, err := cachingStep.Start(context.Background(), input)
	require.NoError(t, err)

	// Get the first message
	results1 := []string{}
	for r := range result1.GetChannel() {
		require.NoError(t, r.Error())
		msg := r.Unwrap()
		content := msg.Content.(*conversation.ChatMessageContent)
		results1 = append(results1, content.Text)
	}
	assert.Equal(t, []string{"Hello"}, results1)

	// Second call with same input should use cache
	result2, err := cachingStep.Start(context.Background(), input)
	require.NoError(t, err)

	// Get the second message
	results2 := []string{}
	for r := range result2.GetChannel() {
		require.NoError(t, r.Error())
		msg := r.Unwrap()
		content := msg.Content.(*conversation.ChatMessageContent)
		results2 = append(results2, content.Text)
	}
	assert.Equal(t, []string{"Hello"}, results2)

	// Verify cache file exists
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(files))

	// Different input should use mock step
	input2 := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Different input"),
	)
	result3, err := cachingStep.Start(context.Background(), input2)
	require.NoError(t, err)

	// Get the third message
	results3 := []string{}
	for r := range result3.GetChannel() {
		require.NoError(t, r.Error())
		msg := r.Unwrap()
		content := msg.Content.(*conversation.ChatMessageContent)
		results3 = append(results3, content.Text)
	}
	assert.Equal(t, []string{"World"}, results3)

	// Test max entries limit
	cachingStep, err = NewCachingStep(mockStep,
		WithCacheDirectory(tempDir),
		WithMaxEntries(1))
	require.NoError(t, err)

	// First call should create cache entry
	_, err = cachingStep.Start(context.Background(), input)
	require.NoError(t, err)

	// Second call with different input should remove first cache entry
	_, err = cachingStep.Start(context.Background(), input2)
	require.NoError(t, err)

	// Verify only one cache file exists
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(files))

	// Test max size limit
	cachingStep, err = NewCachingStep(mockStep,
		WithCacheDirectory(tempDir),
		WithMaxSize(1)) // 1 byte limit
	require.NoError(t, err)

	// Call should fail due to size limit
	_, err = cachingStep.Start(context.Background(), input)
	assert.Error(t, err)
}

func TestCachingStepMultipleMessages(t *testing.T) {
	// Create a temporary directory for the cache
	tempDir, err := os.MkdirTemp("", "caching-step-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create mock messages that will be returned in sequence
	messages := []*conversation.Message{
		conversation.NewChatMessage(conversation.RoleAssistant, "First"),
		conversation.NewChatMessage(conversation.RoleAssistant, "Second"),
		conversation.NewChatMessage(conversation.RoleAssistant, "Third"),
	}

	// Create mock step
	mockStep := NewMockStep(messages)

	// Create caching step
	cachingStep, err := NewCachingStep(mockStep, WithCacheDirectory(tempDir))
	require.NoError(t, err)

	// Test context
	ctx := context.Background()
	conv := conversation.NewConversation()

	// First call should get "First" from mock
	result1, err := cachingStep.Start(ctx, conv)
	require.NoError(t, err)

	var responses1 []string
	for r := range result1.GetChannel() {
		require.NoError(t, r.Error())
		responses1 = append(responses1, r.Unwrap().Content.String())
	}
	assert.Equal(t, []string{"First"}, responses1)

	// Second call with same input should get "First" from cache
	result2, err := cachingStep.Start(ctx, conv)
	require.NoError(t, err)

	var responses2 []string
	for r := range result2.GetChannel() {
		require.NoError(t, r.Error())
		responses2 = append(responses2, r.Unwrap().Content.String())
	}
	assert.Equal(t, responses1, responses2)

	// Different input should get "Second" from mock
	conv2 := conversation.NewConversation()
	conv2 = append(conv2, conversation.NewChatMessage(conversation.RoleUser, "different"))

	result3, err := cachingStep.Start(ctx, conv2)
	require.NoError(t, err)

	var responses3 []string
	for r := range result3.GetChannel() {
		require.NoError(t, r.Error())
		responses3 = append(responses3, r.Unwrap().Content.String())
	}
	assert.Equal(t, []string{"Second"}, responses3)

	// Same input2 should get "Second" from cache
	result4, err := cachingStep.Start(ctx, conv2)
	require.NoError(t, err)

	var responses4 []string
	for r := range result4.GetChannel() {
		require.NoError(t, r.Error())
		responses4 = append(responses4, r.Unwrap().Content.String())
	}
	assert.Equal(t, responses3, responses4)
}
