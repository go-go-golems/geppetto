package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"context"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
)

// Request represents the completion request payload.
type Request struct {
	Model             string    `json:"model"`
	Prompt            string    `json:"prompt"`
	MaxTokensToSample int       `json:"max_tokens_to_sample"`
	StopSequences     []string  `json:"stop_sequences,omitempty"`
	Temperature       *float64  `json:"temperature,omitempty"`
	TopP              *float64  `json:"top_p,omitempty"`
	TopK              *int      `json:"top_k,omitempty"`
	Metadata          *Metadata `json:"metadata,omitempty"`
	Stream            bool      `json:"stream"`
}

// Metadata represents the metadata object for Claude API requests.
type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

// SuccessfulResponse represents the API's successful response.
type SuccessfulResponse struct {
	Completion string `json:"completion"`
	StopReason string `json:"stop_reason"`
	Model      string `json:"model"`
}

// ErrorResponse represents the API's error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details.
type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Client represents the Claude API client.
type Client struct {
	connection *connection.Connection
	APIVersion string
}

const defaultAPIVersion = "2023-06-01"

// Routes declares the Anthropic operations below its API root.
func Routes() connection.Routes {
	return connection.Routes{
		"messages":     connection.Relative("v1/messages"),
		"count-tokens": connection.Relative("v1/messages/count_tokens"),
		"complete":     connection.Relative("v1/complete"),
	}
}

// NewClient accepts the complete API authority, with no credential or client setters.
func NewClient(c *connection.Connection, apiVersion ...string) *Client {
	version := defaultAPIVersion
	if len(apiVersion) > 0 {
		version = apiVersion[0]
	}
	return &Client{connection: c, APIVersion: version}
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("anthropic-version", c.APIVersion)
	req.Header.Set("Content-Type", "application/json")
}

// Complete sends a completion request and returns the response.
func (c *Client) Complete(req *Request) (*SuccessfulResponse, error) {

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	req_, err := c.connection.NewRequest(context.Background(), http.MethodPost, "complete", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req_)

	// #nosec G704 -- URL is validated above with ValidateOutboundURL.
	resp, err := c.connection.Do(req_)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		apiError, err := newAPIError(resp.StatusCode, respBody)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	var successResp SuccessfulResponse
	if err := json.Unmarshal(respBody, &successResp); err != nil {
		return nil, err
	}

	return &successResp, nil
}

// Event represents a server-sent event.
type Event struct {
	Data  string
	Event string // message_start
	ID    string
	Retry int
}

func (c *Client) StreamComplete(req *Request) (<-chan Event, error) {

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	req_, err := c.connection.NewRequest(context.Background(), http.MethodPost, "complete", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req_)

	// #nosec G704 -- URL is validated above with ValidateOutboundURL.
	resp, err := c.connection.Do(req_)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		apiError, err := newAPIError(resp.StatusCode, body)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	events := make(chan Event)
	go func() {
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		defer close(events)

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}

			// Here you parse each line for data, id, event, etc. to fill up your Event struct.
			// For simplicity, this example only captures the data.
			if bytes.HasPrefix(line, []byte("data: ")) {
				events <- Event{Data: string(line[6:])}
			}
		}
	}()

	return events, nil
}
