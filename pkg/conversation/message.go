package conversation

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ContentType string

// TODO(manuel, 2024-07-04) Unify this with the events types that we added for the claude API
const (
	ContentTypeChatMessage ContentType = "chat-message"
	// TODO(manuel, 2024-06-04) This needs to also handle tool call and tool response blocks (tool use block in claude API)
	// See also the comment to refactor this in openai/helpers.go, where tool use information is actually stored in the metadata of the message
	ContentTypeToolUse    ContentType = "tool-use"
	ContentTypeToolResult ContentType = "tool-result"
	ContentTypeImage      ContentType = "image"
	// TODO(manuel, 2024-10-16) Add "ui" type which is only used for ui elements and should be filtered out of the LLM conversation
)

// MessageContent is an interface for different types of node content.
type MessageContent interface {
	ContentType() ContentType
	String() string
	View() string
}

type Role string

const (
	RoleSystem    Role = "system"
	RoleAssistant Role = "assistant"
	RoleUser      Role = "user"
	RoleTool      Role = "tool"
)

type ChatMessageContent struct {
	Role   Role            `json:"role"`
	Text   string          `json:"text"`
	Images []*ImageContent `json:"images"`
}

func (c *ChatMessageContent) ContentType() ContentType {
	return ContentTypeChatMessage
}

func (c *ChatMessageContent) String() string {
	return c.Text
}

func (c *ChatMessageContent) View() string {
	// If we are markdown, add a newline so that it becomes valid markdown to parse.
	if strings.HasPrefix(c.Text, "```") {
		c.Text = "\n" + c.Text
	}
	return fmt.Sprintf("[%s]: %s", c.Role, strings.TrimRight(c.Text, "\n"))
}

var _ MessageContent = (*ChatMessageContent)(nil)

type ToolUseContent struct {
	ToolID string          `json:"toolID"`
	Name   string          `json:"name"`
	Input  json.RawMessage `json:"input"`
	// used by openai currently (only function)
	Type string `json:"type"`
}

func (t *ToolUseContent) ContentType() ContentType {
	return ContentTypeToolUse
}

func (t *ToolUseContent) String() string {
	return fmt.Sprintf("ToolUseContent{ToolID: %s, Name: %s, Input: %s}", t.ToolID, t.Name, t.Input)
}

func (t *ToolUseContent) View() string {
	return fmt.Sprintf("ToolUseContent{ToolID: %s, Name: %s, Input: %s}", t.ToolID, t.Name, t.Input)
}

var _ MessageContent = (*ToolUseContent)(nil)

type ToolResultContent struct {
	ToolID string `json:"toolID"`
	Result string `json:"result"`
}

func (t *ToolResultContent) ContentType() ContentType {
	return ContentTypeToolResult
}

func (t *ToolResultContent) String() string {
	return fmt.Sprintf("ToolResultContent{ToolID: %s, Result: %s}", t.ToolID, t.Result)
}

func (t *ToolResultContent) View() string {
	return fmt.Sprintf("ToolResultContent{ToolID: %s, Result: %s}", t.ToolID, t.Result)
}

var _ MessageContent = (*ToolResultContent)(nil)

type ImageDetail string

const (
	ImageDetailLow  ImageDetail = "low"
	ImageDetailHigh ImageDetail = "high"
	ImageDetailAuto ImageDetail = "auto"
)

type ImageContent struct {
	ImageURL     string      `json:"imageURL"`
	ImageContent []byte      `json:"imageContent"`
	ImageName    string      `json:"imageName"`
	MediaType    string      `json:"mediaType"`
	Detail       ImageDetail `json:"detail"`
}

func NewImageContentFromFile(path string) (*ImageContent, error) {
	// Check if the path is a URL
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return newImageContentFromURL(path)
	}
	return newImageContentFromLocalFile(path)
}

func newImageContentFromURL(url string) (*ImageContent, error) {
	imageName := filepath.Base(url)

	return &ImageContent{
		ImageURL:  url,
		ImageName: imageName,
		Detail:    ImageDetailAuto,
	}, nil
}

func newImageContentFromLocalFile(path string) (*ImageContent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %v", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	if fileInfo.Size() > 20*1024*1024 {
		return nil, fmt.Errorf("image size exceeds 20MB limit")
	}

	mediaType := getMediaTypeFromExtension(filepath.Ext(path))
	if mediaType == "" {
		return nil, fmt.Errorf("unsupported image format: %s", filepath.Ext(path))
	}

	return &ImageContent{
		ImageContent: content,
		ImageName:    fileInfo.Name(),
		MediaType:    mediaType,
		Detail:       ImageDetailAuto,
	}, nil
}

func getMediaTypeFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

func (i *ImageContent) ContentType() ContentType {
	return ContentTypeImage
}

func (i *ImageContent) String() string {
	return fmt.Sprintf("ImageContent{ImageURL: %s, ImageName: %s, Detail: %s}", i.ImageURL, i.ImageName, i.Detail)
}

func (i *ImageContent) View() string {
	return i.String()
}

var _ MessageContent = (*ImageContent)(nil)

// Message represents a single message node in the conversation tree.
type Message struct {
	ParentID   NodeID    `json:"parentID"`
	ID         NodeID    `json:"id"`
	Time       time.Time `json:"time"`
	LastUpdate time.Time `json:"lastUpdate"`

	Content  MessageContent         `json:"content"`
	Metadata map[string]interface{} `json:"metadata"` // Flexible metadata field

	// TODO(manuel, 2024-04-07) Add Parent and Sibling lists
	// omit in json
	Children []*Message `json:"-"`
}

type MessageOption func(*Message)

func WithMetadata(metadata map[string]interface{}) MessageOption {
	return func(message *Message) {
		message.Metadata = metadata
	}
}

func WithTime(time time.Time) MessageOption {
	return func(message *Message) {
		message.Time = time
	}
}

func WithParentID(parentID NodeID) MessageOption {
	return func(message *Message) {
		message.ParentID = parentID
	}
}

func WithID(id NodeID) MessageOption {
	return func(message *Message) {
		message.ID = id
	}
}

func NewMessage(content MessageContent, options ...MessageOption) *Message {
	ret := &Message{
		Content:    content,
		ID:         NodeID(uuid.New()),
		Time:       time.Now(),
		LastUpdate: time.Now(),
	}

	for _, option := range options {
		option(ret)
	}

	return ret
}

func NewChatMessage(role Role, text string, options ...MessageOption) *Message {
	return NewMessage(&ChatMessageContent{
		Role: role,
		Text: text,
	}, options...)
}

func (mn *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(&struct {
		ContentType ContentType `json:"contentType"`
		*Alias
	}{
		ContentType: mn.Content.ContentType(),
		Alias:       (*Alias)(mn),
	})
}

type Conversation []*Message

// GetSinglePrompt concatenates all the messages together with a prompt in front.
// It just concatenates all the messages together with a prompt in front (if there are more than one message).
func (messages Conversation) GetSinglePrompt() string {
	if len(messages) == 0 {
		return ""
	}

	if len(messages) == 1 && messages[0].Content.ContentType() == ContentTypeChatMessage {
		return messages[0].Content.(*ChatMessageContent).Text
	}

	prompt := ""
	for _, message := range messages {
		if message.Content.ContentType() == ContentTypeChatMessage {
			message := message.Content.(*ChatMessageContent)
			prompt += fmt.Sprintf("[%s]: %s\n", message.Role, message.Text)
		}
	}

	return prompt
}
