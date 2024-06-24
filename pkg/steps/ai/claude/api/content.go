package api

import (
	"encoding/json"
)

type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeImage      ContentType = "image"
	ContentTypeToolUse    ContentType = "tool_use"
	ContentTypeToolResult ContentType = "tool_result"
)

type Content interface {
	Type() ContentType
}

type BaseContent struct {
	Type_ ContentType `json:"type"`
}

type TextContent struct {
	BaseContent
	Text string `json:"text"`
}

func (t TextContent) Type() ContentType {
	return ContentTypeText
}

type ImageContent struct {
	BaseContent
	Source ImageSource `json:"source"`
}

func (i ImageContent) Type() ContentType {
	return ContentTypeImage
}

type ImageSource struct {
	BaseContent
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ToolUseContent struct {
	BaseContent
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func (t ToolUseContent) Type() ContentType {
	return ContentTypeToolUse
}

type ToolResultContent struct {
	BaseContent
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

func (t ToolResultContent) Type() ContentType {
	return ContentTypeToolResult
}

func NewTextContent(text string) Content {
	return TextContent{BaseContent: BaseContent{Type_: ContentTypeText}, Text: text}
}

func NewImageContent(mediaType, base64Data string) Content {
	return ImageContent{
		BaseContent: BaseContent{
			Type_: ContentTypeImage,
		},
		Source: ImageSource{
			Type:      "base64",
			MediaType: mediaType,
			Data:      base64Data,
		},
	}
}

func NewToolUseContent(toolID, toolName string, toolInput json.RawMessage) Content {
	return ToolUseContent{
		BaseContent: BaseContent{Type_: ContentTypeToolUse},
		ID:          toolID,
		Name:        toolName,
		Input:       toolInput,
	}
}

func NewToolResultContent(toolUseID, content string) Content {
	return ToolResultContent{
		BaseContent: BaseContent{Type_: ContentTypeToolResult},
		ToolUseID:   toolUseID,
		Content:     content,
	}
}
