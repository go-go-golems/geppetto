package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

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
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"` // JSON schema for the tool input
}

// Message represents a single message in the conversation.
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // Can be a string or an array of content blocks
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

// Content represents a single block of content, which can be of various types.
type Content struct {
	Type    string          `json:"type"`
	Text    *string         `json:"text,omitempty"`
	Image   *ImageContent   `json:"image,omitempty"`
	ToolUse *ToolUseContent `json:"tool_use,omitempty"`
	// Add additional fields for other content types as needed.
}

// ImageContent represents an image content block.
type ImageContent struct {
	Source ImageSource `json:"source"`
}

// ImageSource represents the source of an image, which can be a base64-encoded string.
type ImageSource struct {
	Type      string `json:"type"`       // e.g., "base64"
	MediaType string `json:"media_type"` // e.g., "image/jpeg"
	Data      string `json:"data"`
}

// ToolUseContent represents a content block where the model uses a tool.
type ToolUseContent struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Input  json.RawMessage `json:"input"`            // JSON structure for the tool input
	Result *string         `json:"result,omitempty"` // Optional result of the tool use
}

// NewTextContent creates a new text content block.
func NewTextContent(text string) Content {
	return Content{
		Type: "text",
		Text: &text,
	}
}

// NewImageContent creates a new image content block with base64-encoded data.
func NewImageContent(mediaType, base64Data string) Content {
	return Content{
		Type: "image",
		Image: &ImageContent{
			Source: ImageSource{
				Type:      "base64",
				MediaType: mediaType,
				Data:      base64Data,
			},
		},
	}
}

// NewToolUseContent creates a new tool use content block.
func NewToolUseContent(toolID, toolName string, toolInput json.RawMessage) Content {
	return Content{
		Type: "tool_use",
		ToolUse: &ToolUseContent{
			ID:    toolID,
			Name:  toolName,
			Input: toolInput,
		},
	}
}

// Usage represents the billing and rate-limit usage information.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
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
		return nil, errors.New(errorResp.Error.Message)
	}

	var messageResp MessageResponse
	respBody, _ := io.ReadAll(resp.Body)
	if unmarshalErr := json.Unmarshal(respBody, &messageResp); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return &messageResp, nil
}

// StreamMessage sends a message request and returns a channel of Events for streaming responses.
func (c *Client) StreamMessage(ctx context.Context, req *MessageRequest) (<-chan Event, error) {
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
		return nil, errors.New(errorResp.Error.Message)
	}

	events := make(chan Event)
	go func() {
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					// Handle the error if needed
				}
				break
			}

			// Parse the line into an Event struct
			var event Event
			if parseErr := parseSSELine(line, &event); parseErr != nil {
				// Handle the parse error if needed
				continue
			}

			select {
			case events <- event:
				// Event sent successfully
			case <-ctx.Done():
				// Context cancelled, stop streaming
				return
			}
		}
	}()

	return events, nil
}

// parseSSELine parses a line from an SSE stream into an Event struct.
func parseSSELine(line []byte, event *Event) error {
	// Trim the potential trailing newline character
	line = bytes.TrimSuffix(line, []byte("\n"))

	// Split the line into "field: value" pairs
	parts := bytes.SplitN(line, []byte(": "), 2)
	if len(parts) != 2 {
		return errors.New("invalid SSE line format")
	}

	field, value := parts[0], parts[1]
	switch string(field) {
	case "data":
		event.Data = string(value)
	case "event":
		event.Event = string(value)
	case "id":
		event.ID = string(value)
	case "retry":
		// Convert the retry value to an integer
		retry, err := strconv.Atoi(string(value))
		if err != nil {
			return err
		}
		event.Retry = retry
	default:
		// Ignore unknown fields
	}

	return nil
}
