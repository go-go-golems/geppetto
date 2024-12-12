package js

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// JSConversation wraps a conversation.Manager to expose it to JavaScript
type JSConversation struct {
	runtime *goja.Runtime
	manager conversation.Manager
}

func NewJSConversation(runtime *goja.Runtime) *JSConversation {
	return &JSConversation{
		runtime: runtime,
		manager: conversation.NewManager(),
	}
}

// AddMessage adds a new chat message to the conversation
func (jc *JSConversation) AddMessage(role string, text string, options *goja.Object) (string, error) {
	var msgOptions []conversation.MessageOption

	if options != nil {
		if metadata := options.Get("metadata"); metadata != nil {
			msgOptions = append(msgOptions, conversation.WithMetadata(metadata.Export().(map[string]interface{})))
		}
		if parentID := options.Get("parentID"); parentID != nil {
			parentUUID, err := uuid.Parse(parentID.String())
			if err != nil {
				return "", fmt.Errorf("failed to parse parentID: %v", err)
			}
			msgOptions = append(msgOptions, conversation.WithParentID(conversation.NodeID(parentUUID)))
		}
		if timeStr := options.Get("time"); timeStr != nil {
			t, err := time.Parse(time.RFC3339, timeStr.String())
			if err == nil {
				msgOptions = append(msgOptions, conversation.WithTime(t))
			}
		}
		if id := options.Get("id"); id != nil {
			idUUID, err := uuid.Parse(id.String())
			if err != nil {
				return "", fmt.Errorf("failed to parse id: %v", err)
			}
			msgOptions = append(msgOptions, conversation.WithID(conversation.NodeID(idUUID)))
		}
	}

	msg := conversation.NewChatMessage(conversation.Role(role), text, msgOptions...)
	jc.manager.AppendMessages(msg)

	return msg.ID.String(), nil
}

// AddMessageWithImage adds a chat message with an attached image
func (jc *JSConversation) AddMessageWithImage(role string, text string, imagePath string) (string, error) {
	img, err := conversation.NewImageContentFromFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to create image content: %v", err)
	}

	content := &conversation.ChatMessageContent{
		Role:   conversation.Role(role),
		Text:   text,
		Images: []*conversation.ImageContent{img},
	}

	msg := conversation.NewMessage(content)
	jc.manager.AppendMessages(msg)

	return msg.ID.String(), nil
}

// AddToolUse adds a tool use message
func (jc *JSConversation) AddToolUse(toolID string, name string, input interface{}) (string, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input: %v", err)
	}

	content := &conversation.ToolUseContent{
		ToolID: toolID,
		Name:   name,
		Input:  inputJSON,
		Type:   "function",
	}

	msg := conversation.NewMessage(content)
	jc.manager.AppendMessages(msg)

	return msg.ID.String(), nil
}

// AddToolResult adds a tool result message
func (jc *JSConversation) AddToolResult(toolID string, result string) (string, error) {
	content := &conversation.ToolResultContent{
		ToolID: toolID,
		Result: result,
	}

	msg := conversation.NewMessage(content)
	jc.manager.AppendMessages(msg)

	return msg.ID.String(), nil
}

// GetMessages returns all messages in the conversation
func (jc *JSConversation) GetMessages() conversation.Conversation {
	return jc.manager.GetConversation()
}

// GetMessageView returns a formatted view of a specific message
func (jc *JSConversation) GetMessageView(messageID string) (string, error) {
	msgUUID, err := uuid.Parse(messageID)
	if err != nil {
		return "", fmt.Errorf("invalid message ID: %v", err)
	}

	if msg, ok := jc.manager.GetMessage(conversation.NodeID(msgUUID)); ok {
		return msg.Content.View(), nil
	}

	return "", fmt.Errorf("message not found")
}

// UpdateMetadata updates a message's metadata
func (jc *JSConversation) UpdateMetadata(messageID string, metadata map[string]interface{}) (bool, error) {
	msgUUID, err := uuid.Parse(messageID)
	if err != nil {
		return false, fmt.Errorf("invalid message ID: %v", err)
	}

	if msg, ok := jc.manager.GetMessage(conversation.NodeID(msgUUID)); ok {
		msg.Metadata = metadata
		return true, nil
	}

	return false, nil
}

// GetSinglePrompt returns the conversation as a single prompt string
func (jc *JSConversation) GetSinglePrompt() string {
	return jc.manager.GetConversation().GetSinglePrompt()
}

// ToGoConversation returns the underlying Go conversation
func (jc *JSConversation) ToGoConversation() conversation.Conversation {
	return jc.manager.GetConversation()
}

// RegisterConversation registers the Conversation constructor in the JavaScript runtime
func RegisterConversation(runtime *goja.Runtime) error {
	return runtime.Set("Conversation", runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
		jsConv := NewJSConversation(runtime)
		// Return the Go conversation object directly
		val := runtime.ToValue(jsConv).(*goja.Object)
		err := val.SetPrototype(call.This.Prototype())
		if err != nil {
			log.Warn().Err(err).Msg("failed to set prototype")
		}
		return val
	}))
}
