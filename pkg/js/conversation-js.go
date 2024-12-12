package js

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/google/uuid"
)

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

func (jc *JSConversation) AddMessage(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(jc.runtime.NewTypeError("AddMessage requires role and text arguments"))
	}

	role := conversation.Role(call.Arguments[0].String())
	text := call.Arguments[1].String()

	var options []conversation.MessageOption
	if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) {
		opts := call.Arguments[2].ToObject(jc.runtime)

		if metadata := opts.Get("metadata"); metadata != nil {
			options = append(options, conversation.WithMetadata(metadata.Export().(map[string]interface{})))
		}
		if parentID := opts.Get("parentID"); parentID != nil {
			parentUUID, err := uuid.Parse(parentID.String())
			if err != nil {
				return jc.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to parse parentID: %v", err)})
			}
			options = append(options, conversation.WithParentID(conversation.NodeID(parentUUID)))
		}
		if timeStr := opts.Get("time"); timeStr != nil {
			t, err := time.Parse(time.RFC3339, timeStr.String())
			if err == nil {
				options = append(options, conversation.WithTime(t))
			}
		}
		if id := opts.Get("id"); id != nil {
			idUUID, err := uuid.Parse(id.String())
			if err != nil {
				return jc.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to parse id: %v", err)})
			}
			options = append(options, conversation.WithID(conversation.NodeID(idUUID)))
		}
	}

	msg := conversation.NewChatMessage(role, text, options...)
	jc.manager.AppendMessages(msg)

	return jc.runtime.ToValue(msg.ID.String())
}

func (jc *JSConversation) AddMessageWithImage(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 3 {
		panic(jc.runtime.NewTypeError("AddMessageWithImage requires role, text, and imagePath arguments"))
	}

	role := conversation.Role(call.Arguments[0].String())
	text := call.Arguments[1].String()
	imagePath := call.Arguments[2].String()

	img, err := conversation.NewImageContentFromFile(imagePath)
	if err != nil {
		panic(jc.runtime.NewTypeError(fmt.Sprintf("failed to create image content: %v", err)))
	}

	content := &conversation.ChatMessageContent{
		Role:   role,
		Text:   text,
		Images: []*conversation.ImageContent{img},
	}

	msg := conversation.NewMessage(content)
	jc.manager.AppendMessages(msg)

	return jc.runtime.ToValue(msg.ID.String())
}

func (jc *JSConversation) AddToolUse(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 3 {
		panic(jc.runtime.NewTypeError("AddToolUse requires toolID, name, and input arguments"))
	}

	toolID := call.Arguments[0].String()
	name := call.Arguments[1].String()
	input := call.Arguments[2].Export()

	inputJSON, err := json.Marshal(input)
	if err != nil {
		panic(jc.runtime.NewTypeError(fmt.Sprintf("failed to marshal input: %v", err)))
	}

	content := &conversation.ToolUseContent{
		ToolID: toolID,
		Name:   name,
		Input:  inputJSON,
		Type:   "function",
	}

	msg := conversation.NewMessage(content)
	jc.manager.AppendMessages(msg)

	return jc.runtime.ToValue(msg.ID.String())
}

func (jc *JSConversation) AddToolResult(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(jc.runtime.NewTypeError("AddToolResult requires toolID and result arguments"))
	}

	toolID := call.Arguments[0].String()
	result := call.Arguments[1].String()

	content := &conversation.ToolResultContent{
		ToolID: toolID,
		Result: result,
	}

	msg := conversation.NewMessage(content)
	jc.manager.AppendMessages(msg)

	return jc.runtime.ToValue(msg.ID.String())
}

func (jc *JSConversation) GetMessages() goja.Value {
	conv := jc.manager.GetConversation()
	messages := make([]map[string]interface{}, len(conv))

	for i, msg := range conv {
		msgMap := map[string]interface{}{
			"id":         msg.ID.String(),
			"parentID":   msg.ParentID.String(),
			"time":       msg.Time,
			"lastUpdate": msg.LastUpdate,
			"metadata":   msg.Metadata,
		}

		switch content := msg.Content.(type) {
		case *conversation.ChatMessageContent:
			msgMap["type"] = "chat-message"
			msgMap["role"] = content.Role
			msgMap["text"] = content.Text
			if len(content.Images) > 0 {
				images := make([]map[string]interface{}, len(content.Images))
				for j, img := range content.Images {
					images[j] = map[string]interface{}{
						"imageURL":  img.ImageURL,
						"imageName": img.ImageName,
						"mediaType": img.MediaType,
						"detail":    img.Detail,
					}
				}
				msgMap["images"] = images
			}
		case *conversation.ToolUseContent:
			msgMap["type"] = "tool-use"
			msgMap["toolID"] = content.ToolID
			msgMap["name"] = content.Name
			var input interface{}
			_ = json.Unmarshal(content.Input, &input)
			msgMap["input"] = input
			msgMap["toolType"] = content.Type
		case *conversation.ToolResultContent:
			msgMap["type"] = "tool-result"
			msgMap["toolID"] = content.ToolID
			msgMap["result"] = content.Result
		}

		messages[i] = msgMap
	}

	return jc.runtime.ToValue(messages)
}

func (jc *JSConversation) GetMessageView(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(jc.runtime.NewTypeError("GetMessageView requires messageID argument"))
	}

	messageID := call.Arguments[0].String()
	msgUUID, err := uuid.Parse(messageID)
	if err != nil {
		return goja.Undefined()
	}

	if msg, ok := jc.manager.GetMessage(conversation.NodeID(msgUUID)); ok {
		return jc.runtime.ToValue(msg.Content.View())
	}

	return goja.Undefined()
}

func (jc *JSConversation) UpdateMetadata(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(jc.runtime.NewTypeError("UpdateMetadata requires messageID and metadata arguments"))
	}

	messageID := call.Arguments[0].String()
	metadata := call.Arguments[1].Export().(map[string]interface{})

	msgUUID, err := uuid.Parse(messageID)
	if err != nil {
		return jc.runtime.ToValue(false)
	}

	if msg, ok := jc.manager.GetMessage(conversation.NodeID(msgUUID)); ok {
		msg.Metadata = metadata
		return jc.runtime.ToValue(true)
	}

	return jc.runtime.ToValue(false)
}

func (jc *JSConversation) GetSinglePrompt() goja.Value {
	return jc.runtime.ToValue(jc.manager.GetConversation().GetSinglePrompt())
}

func (jc *JSConversation) ToGoConversation() conversation.Conversation {
	return jc.manager.GetConversation()
}

func RegisterConversation(runtime *goja.Runtime) error {
	constructor := func(call goja.ConstructorCall) *goja.Object {
		jsConv := NewJSConversation(runtime)

		obj := call.This.ToObject(runtime)
		_ = obj.Set("addMessage", jsConv.AddMessage)
		_ = obj.Set("addMessageWithImage", jsConv.AddMessageWithImage)
		_ = obj.Set("addToolUse", jsConv.AddToolUse)
		_ = obj.Set("addToolResult", jsConv.AddToolResult)
		_ = obj.Set("getMessages", jsConv.GetMessages)
		_ = obj.Set("getMessageView", jsConv.GetMessageView)
		_ = obj.Set("updateMetadata", jsConv.UpdateMetadata)
		_ = obj.Set("getSinglePrompt", jsConv.GetSinglePrompt)
		_ = obj.Set("toGoConversation", jsConv.ToGoConversation)

		return obj
	}

	return runtime.Set("Conversation", constructor)
}
