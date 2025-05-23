package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// TODO(manuel, 2024-04-07) This is WIP code, most of it generated by an LLM. Do not use.

// MessageRequest represents the Messages API request payload.
type MessageRequest struct {
	Model         string    `json:"model"`
	Messages      []Message `json:"messages"`
	MaxTokens     int       `json:"max_tokens"`
	Metadata      *Metadata `json:"metadata,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	Stream        bool      `json:"stream"`
	System        string    `json:"system,omitempty"`
	Temperature   *float64  `json:"temperature,omitempty"`
	Tools         []Tool    `json:"tools,omitempty"`
	TopK          *int      `json:"top_k,omitempty"`
	TopP          *float64  `json:"top_p,omitempty"`
}

// Tool represents a tool that the model can use.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema"` // JSON schema for the tool input
}

// Message represents a single message in the conversation.
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"` // Can be a string or an array of content blocks
}

func (m *Message) UnmarshalJSON(data []byte) error {
	var temp struct {
		Role    string            `json:"role"`
		Content []json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	m.Role = temp.Role
	m.Content = make([]Content, len(temp.Content))

	for i, contentData := range temp.Content {
		content, err := UnmarshalContent(contentData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal content at index %d: %w", i, err)
		}
		m.Content[i] = content
	}

	return nil
}

// MessageResponse represents the Messages API response payload.
type MessageResponse struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Role         string    `json:"role"`
	Content      []Content `json:"content"`
	Model        string    `json:"model"`
	StopReason   string    `json:"stop_reason,omitempty"`
	StopSequence string    `json:"stop_sequence,omitempty"`
	Usage        Usage     `json:"usage"`
}

var _ zerolog.LogObjectMarshaler = MessageResponse{}

func (s MessageResponse) MarshalZerologObject(e *zerolog.Event) {
	arr := zerolog.Arr()
	for _, content := range s.Content {
		arr.Interface(content)
	}
	e.Str("id", s.ID).
		Str("type", s.Type).
		Str("role", s.Role).
		Str("model", s.Model).
		Str("stop_reason", s.StopReason).
		Str("stop_sequence", s.StopSequence).
		Array("content", arr)
}

func (m MessageResponse) ToMessage() *conversation.Message {
	messageContent := &conversation.ChatMessageContent{
		Role:   conversation.Role(m.Role),
		Text:   m.FullText(),
		Images: nil,
	}
	return conversation.NewMessage(messageContent,
		conversation.WithTime(time.Now()),
		conversation.WithMetadata(map[string]interface{}{
			"claude_message_id": m.ID,
		}),
	)
}

// FullText is a way to quickly get the entire text of the message response,
// for our current streaming system which only deals with full strings.
func (m MessageResponse) FullText() string {
	res := ""
	for _, c := range m.Content {
		switch v := c.(type) {
		case TextContent:
			res += v.Text
		case ImageContent:
		// skip images for now
		case ToolUseContent:
			res += "Tool Call: " + v.Name + "\n"
			res += "ID: " + v.ID + "\n"
			res += string(v.Input)
		case ToolResultContent:
			res += "Tool Call Result: " + v.ToolUseID + "\n"
			res += v.Content
		default:

		}
	}
	return res
}

// Usage represents the billing and rate-limit usage information.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

func (u Usage) MarshalZerologObject(e *zerolog.Event) {
	e.Int("input_tokens", u.InputTokens).
		Int("output_tokens", u.OutputTokens).
		Int("cache_creation_input_tokens", u.CacheCreationInputTokens).
		Int("cache_read_input_tokens", u.CacheReadInputTokens)
}

// SendMessage sends a message request and returns the response.
func (c *Client) SendMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/messages", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)
	truncatedBody := string(body)
	if len(truncatedBody) > 100 {
		truncatedBody = truncatedBody[:50] + "..." + truncatedBody[len(truncatedBody)-50:]
	}
	log.Debug().Msgf("Sending message request: %s", truncatedBody)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		respBody, _ := io.ReadAll(resp.Body)
		if unmarshalErr := json.Unmarshal(respBody, &errorResp); unmarshalErr != nil {
			return nil, unmarshalErr
		}
		return nil, fmt.Errorf("claude API error: %s", errorResp.Error.Message)
	}

	var messageResp MessageResponse
	respBody, _ := io.ReadAll(resp.Body)
	if unmarshalErr := json.Unmarshal(respBody, &messageResp); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return &messageResp, nil
}

// StreamMessage sends a message request and returns a channel of Events for streaming responses.
func (c *Client) StreamMessage(ctx context.Context, req *MessageRequest) (<-chan StreamingEvent, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/messages", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		var errorResp ErrorResponse
		respBody, _ := io.ReadAll(resp.Body)
		if unmarshalErr := json.Unmarshal(respBody, &errorResp); unmarshalErr != nil {
			return nil, unmarshalErr
		}
		return nil, fmt.Errorf("claude API error: %s", errorResp.Error.Message)
	}

	events := make(chan StreamingEvent)
	go func() {
		defer close(events)
		streamEvents(ctx, resp, events)
	}()

	return events, nil
}

func (m *MessageResponse) UnmarshalJSON(data []byte) error {
	type Alias MessageResponse
	var temp struct {
		*Alias
		Content []json.RawMessage `json:"content"`
	}
	temp.Alias = (*Alias)(m)

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	m.Content = make([]Content, len(temp.Content))
	for i, contentData := range temp.Content {
		content, err := UnmarshalContent(contentData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal content at index %d: %w", i, err)
		}
		m.Content[i] = content
	}

	return nil
}
