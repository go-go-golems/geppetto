package api

import "github.com/rs/zerolog"

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
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
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

func NewToolUseContent(toolID, toolName string, toolInput string) Content {
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

func (bc BaseContent) MarshalZerologObject(e *zerolog.Event) {
	e.Str("type", string(bc.Type_))
}

func (tc TextContent) MarshalZerologObject(e *zerolog.Event) {
	e.Object("base", tc.BaseContent)
	e.Str("text", tc.Text)
}

func (ic ImageContent) MarshalZerologObject(e *zerolog.Event) {
	e.Object("base", ic.BaseContent)
	e.Object("source", ic.Source)
}

func (is ImageSource) MarshalZerologObject(e *zerolog.Event) {
	e.Object("base", is.BaseContent)
	e.Str("type", is.Type)
	e.Str("media_type", is.MediaType)
	e.Str("data", is.Data)
}

func (tuc ToolUseContent) MarshalZerologObject(e *zerolog.Event) {
	e.Object("base", tuc.BaseContent)
	e.Str("id", tuc.ID)
	e.Str("name", tuc.Name)
	e.Str("input", tuc.Input)
}

func (trc ToolResultContent) MarshalZerologObject(e *zerolog.Event) {
	e.Object("base", trc.BaseContent)
	e.Str("tool_use_id", trc.ToolUseID)
	e.Str("content", trc.Content)
}
