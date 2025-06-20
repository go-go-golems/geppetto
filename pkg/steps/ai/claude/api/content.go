package api

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
)

type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeImage      ContentType = "image"
	ContentTypeToolUse    ContentType = "tool_use"
	ContentTypeToolResult ContentType = "tool_result"
	// ContentTypeServerToolUse represents server initiated tool calls such as web search
	ContentTypeServerToolUse ContentType = "server_tool_use"
	// ContentTypeWebSearchToolResult represents the result of a server side web search
	ContentTypeWebSearchToolResult ContentType = "web_search_tool_result"
	// ContentTypeThinking represents Claude thinking blocks
	ContentTypeThinking ContentType = "thinking"
	// ContentTypeRedactedThinking represents redacted thinking blocks
	ContentTypeRedactedThinking ContentType = "redacted_thinking"
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

// ServerToolUseContent represents a server initiated tool use, typically web search
type ServerToolUseContent struct {
	BaseContent
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

func (t ServerToolUseContent) Type() ContentType {
	return ContentTypeServerToolUse
}

// WebSearchToolResultContent represents the result of a web search tool call
type WebSearchToolResultContent struct {
	BaseContent
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
}

func (t WebSearchToolResultContent) Type() ContentType {
	return ContentTypeWebSearchToolResult
}

// ThinkingContent represents Claude thinking blocks
type ThinkingContent struct {
	BaseContent
	Signature string `json:"signature"`
	Thinking  string `json:"thinking"`
}

func (t ThinkingContent) Type() ContentType {
	return ContentTypeThinking
}

// RedactedThinkingContent represents redacted thinking blocks
type RedactedThinkingContent struct {
	BaseContent
	Data string `json:"data"`
}

func (t RedactedThinkingContent) Type() ContentType {
	return ContentTypeRedactedThinking
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

func NewServerToolUseContent(toolID, toolName string, toolInput string) Content {
	return ServerToolUseContent{
		BaseContent: BaseContent{Type_: ContentTypeServerToolUse},
		ID:          toolID,
		Name:        toolName,
		Input:       toolInput,
	}
}

func NewWebSearchToolResultContent(toolUseID string, content json.RawMessage) Content {
	return WebSearchToolResultContent{
		BaseContent: BaseContent{Type_: ContentTypeWebSearchToolResult},
		ToolUseID:   toolUseID,
		Content:     content,
	}
}

func NewThinkingContent(signature, thinking string) Content {
	return ThinkingContent{
		BaseContent: BaseContent{Type_: ContentTypeThinking},
		Signature:   signature,
		Thinking:    thinking,
	}
}

func NewRedactedThinkingContent(data string) Content {
	return RedactedThinkingContent{BaseContent: BaseContent{Type_: ContentTypeRedactedThinking}, Data: data}
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

func (stuc ServerToolUseContent) MarshalZerologObject(e *zerolog.Event) {
	e.Object("base", stuc.BaseContent)
	e.Str("id", stuc.ID)
	e.Str("name", stuc.Name)
	e.Str("input", stuc.Input)
}

func (wsrc WebSearchToolResultContent) MarshalZerologObject(e *zerolog.Event) {
	e.Object("base", wsrc.BaseContent)
	e.Str("tool_use_id", wsrc.ToolUseID)
	if len(wsrc.Content) > 0 {
		e.RawJSON("content", wsrc.Content)
	}
}

func (tc ThinkingContent) MarshalZerologObject(e *zerolog.Event) {
	e.Object("base", tc.BaseContent)
	e.Str("signature", tc.Signature)
	e.Str("thinking", tc.Thinking)
}

func (rtc RedactedThinkingContent) MarshalZerologObject(e *zerolog.Event) {
	e.Object("base", rtc.BaseContent)
	e.Str("data", rtc.Data)
}

func UnmarshalContent(data []byte) (Content, error) {
	var base BaseContent
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	switch base.Type_ {
	case ContentTypeText:
		var text TextContent
		if err := json.Unmarshal(data, &text); err != nil {
			return nil, err
		}
		return text, nil
	case ContentTypeImage:
		var image ImageContent
		if err := json.Unmarshal(data, &image); err != nil {
			return nil, err
		}
		return image, nil
	case ContentTypeToolUse:
		var toolUse ToolUseContent
		if err := json.Unmarshal(data, &toolUse); err != nil {
			return nil, err
		}
		return toolUse, nil
	case ContentTypeToolResult:
		var toolResult ToolResultContent
		if err := json.Unmarshal(data, &toolResult); err != nil {
			return nil, err
		}
		return toolResult, nil
	case ContentTypeServerToolUse:
		var stuc ServerToolUseContent
		if err := json.Unmarshal(data, &stuc); err != nil {
			return nil, err
		}
		return stuc, nil
	case ContentTypeWebSearchToolResult:
		var wsrc WebSearchToolResultContent
		if err := json.Unmarshal(data, &wsrc); err != nil {
			return nil, err
		}
		return wsrc, nil
	case ContentTypeThinking:
		var tc ThinkingContent
		if err := json.Unmarshal(data, &tc); err != nil {
			return nil, err
		}
		return tc, nil
	case ContentTypeRedactedThinking:
		var rtc RedactedThinkingContent
		if err := json.Unmarshal(data, &rtc); err != nil {
			return nil, err
		}
		return rtc, nil
	default:
		return nil, fmt.Errorf("unknown content type: %s", base.Type_)
	}
}
