package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

type StreamingEventType string

const (
	PingType              StreamingEventType = "ping"
	MessageStartType      StreamingEventType = "message_start"
	ContentBlockStartType StreamingEventType = "content_block_start"
	ContentBlockDeltaType StreamingEventType = "content_block_delta"
	ContentBlockStopType  StreamingEventType = "content_block_stop"
	MessageDeltaType      StreamingEventType = "message_delta"
	MessageStopType       StreamingEventType = "message_stop"
	ErrorType             StreamingEventType = "error"
)

type StreamingDeltaType string

const (
	TextDeltaType      StreamingDeltaType = "text_delta"
	InputJSONDeltaType StreamingDeltaType = "input_json_delta"
	CitationsDeltaType StreamingDeltaType = "citations_delta"
	ThinkingDeltaType  StreamingDeltaType = "thinking_delta"
	SignatureDeltaType StreamingDeltaType = "signature_delta"
)

type StreamingEvent struct {
	Type         StreamingEventType `json:"type"`
	Message      *MessageResponse   `json:"message,omitempty"`
	Delta        *Delta             `json:"delta,omitempty"`
	Error        *Error             `json:"error,omitempty"`
	Index        int                `json:"index,omitempty"`
	Usage        *Usage             `json:"usage,omitempty"`
	ContentBlock *ContentBlock      `json:"content_block,omitempty"`
}

func (s StreamingEvent) MarshalZerologObject(e *zerolog.Event) {
	e.Str("type", string(s.Type))

	if s.Message != nil {
		e.Object("message", s.Message)
	}

	if s.Delta != nil {
		e.Object("delta", s.Delta)
	}

	if s.Error != nil {
		e.Object("error", s.Error)
	}

	if s.Index != 0 {
		e.Int("index", s.Index)
	}

	if s.Usage != nil {
		e.Object("usage", s.Usage)
	}

	if s.ContentBlock != nil {
		e.Object("content_block", s.ContentBlock)
	}
}

var _ zerolog.LogObjectMarshaler = StreamingEvent{}

type ContentBlock struct {
	Type      ContentType     `json:"type"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     string          `json:"input,omitempty"`
	Text      string          `json:"text,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
}

type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Delta struct {
	Type         StreamingDeltaType `json:"type"`
	Text         string             `json:"text,omitempty"`
	PartialJSON  string             `json:"partial_json"`
	StopReason   string             `json:"stop_reason,omitempty"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Thinking     string             `json:"thinking,omitempty"`
	Signature    string             `json:"signature,omitempty"`
}

func (cb ContentBlock) MarshalZerologObject(e *zerolog.Event) {
	e.Str("type", string(cb.Type))
	if cb.ID != "" {
		e.Str("id", cb.ID)
	}
	if cb.Name != "" {
		e.Str("name", cb.Name)
	}
	if cb.Input != "" {
		e.Str("input", cb.Input)
	}
	if cb.Text != "" {
		e.Str("text", cb.Text)
	}
	if cb.ToolUseID != "" {
		e.Str("tool_use_id", cb.ToolUseID)
	}
	if len(cb.Content) > 0 {
		e.RawJSON("content", cb.Content)
	}
	if cb.Thinking != "" {
		e.Str("thinking", cb.Thinking)
	}
	if cb.Signature != "" {
		e.Str("signature", cb.Signature)
	}
}

func (err Error) MarshalZerologObject(e *zerolog.Event) {
	e.Str("type", err.Type)
	e.Str("message", err.Message)
}

func (d Delta) MarshalZerologObject(e *zerolog.Event) {
	e.Str("type", string(d.Type))
	if d.Text != "" {
		e.Str("text", d.Text)
	}
	e.Str("partial_json", d.PartialJSON)
	if d.StopReason != "" {
		e.Str("stop_reason", d.StopReason)
	}
	if d.StopSequence != "" {
		e.Str("stop_sequence", d.StopSequence)
	}
	if d.Thinking != "" {
		e.Str("thinking", d.Thinking)
	}
	if d.Signature != "" {
		e.Str("signature", d.Signature)
	}
}

func streamEvents(ctx context.Context, resp *http.Response, events chan StreamingEvent) {
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	reader := bufio.NewReader(resp.Body)
	var eventLines [][]byte
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF && err != context.Canceled {
				// Handle the error if needed
				panic("Not implemented")
			}
			break
		}
		if len(bytes.TrimSpace(line)) == 0 {
			// Empty line indicates the end of an event
			var event StreamingEvent
			if parseErr := parseSSEEvent(eventLines, &event); parseErr != nil {
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
			// Reset the event lines for the next event
			eventLines = eventLines[:0]
		} else {
			// Accumulate the lines for the current event
			eventLines = append(eventLines, line)
		}
	}
}

// parseSSEEvent parses an SSE event from multiple lines.
func parseSSEEvent(lines [][]byte, event *StreamingEvent) error {
	eventData := ""
	for _, line := range lines {
		// Trim the potential trailing newline character
		line = bytes.TrimSuffix(line, []byte("\n"))

		// Split the line into "field: value" pairs
		parts := bytes.SplitN(line, []byte(": "), 2)
		if len(parts) != 2 {
			continue
		}

		field, value := parts[0], parts[1]
		if string(field) == "data" {
			eventData += string(value) + "\n"
		}
	}

	// Trim the trailing newline from eventData
	eventData = strings.TrimSuffix(eventData, "\n")

	// Unmarshal the event data into the StreamingEvent struct
	err := json.Unmarshal([]byte(eventData), event)
	if err != nil {
		return err
	}

	return nil
}
