package claude

import (
	"encoding/base64"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/pkg/errors"
)

// messageToClaudeMessage converts a conversation message to a Claude API message
func messageToClaudeMessage(msg *conversation.Message) api.Message {
	switch content := msg.Content.(type) {
	case *conversation.ChatMessageContent:
		// If original Claude content was preserved in metadata, use it
		if claudeContent, exists := msg.Metadata["claude_original_content"]; exists {
			if originalContent, ok := claudeContent.([]api.Content); ok {
				return api.Message{
					Role:    string(content.Role),
					Content: originalContent,
				}
			}
		}

		res := api.Message{
			Role: string(content.Role),
			Content: []api.Content{
				api.NewTextContent(content.Text),
			},
		}
		for _, img := range content.Images {
			res.Content = append(res.Content, api.NewImageContent(img.MediaType, base64.StdEncoding.EncodeToString(img.ImageContent)))
		}
		return res

	case *conversation.ToolUseContent:
		// Claude expects tool_use in assistant role
		return api.Message{
			Role: string(conversation.RoleAssistant),
			Content: []api.Content{
				api.NewToolUseContent(content.ToolID, content.Name, string(content.Input)),
			},
		}

	case *conversation.ToolResultContent:
		// Claude expects tool results in user role
		return api.Message{
			Role: string(conversation.RoleUser),
			Content: []api.Content{
				api.NewToolResultContent(content.ToolID, content.Result),
			},
		}
	}

	return api.Message{}
}

// MakeMessageRequest builds a Claude MessageRequest from settings and a conversation
func MakeMessageRequest(
	s *settings.StepSettings,
	messages conversation.Conversation,
) (*api.MessageRequest, error) {
	if s.Client == nil {
		return nil, steps.ErrMissingClientSettings
	}
	if s.Claude == nil {
		return nil, errors.New("no claude settings")
	}

	chatSettings := s.Chat
	engine := ""
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	msgs := []api.Message{}
	systemPrompt := ""
	for _, m := range messages {
		if chatMsg, ok := m.Content.(*conversation.ChatMessageContent); ok && chatMsg.Role == conversation.RoleSystem {
			systemPrompt = chatMsg.Text
			continue
		}
		msgs = append(msgs, messageToClaudeMessage(m))
	}

	temperature := 0.0
	if chatSettings.Temperature != nil {
		temperature = *chatSettings.Temperature
	}
	topP := 0.0
	if chatSettings.TopP != nil {
		topP = *chatSettings.TopP
	}
	maxTokens := 1024
	if chatSettings.MaxResponseTokens != nil && *chatSettings.MaxResponseTokens > 0 {
		maxTokens = *chatSettings.MaxResponseTokens
	}

	req := &api.MessageRequest{
		Model:         engine,
		Messages:      msgs,
		MaxTokens:     maxTokens,
		Metadata:      nil,
		StopSequences: chatSettings.Stop,
		Stream:        chatSettings.Stream,
		System:        systemPrompt,
		Temperature:   cast.WrapAddr[float64](temperature),
		Tools:         nil,
		TopK:          nil,
		TopP:          cast.WrapAddr[float64](topP),
	}

	return req, nil
}

// end helpers

// Just like in the openai package, we merge the received tool calls and messages from streaming
