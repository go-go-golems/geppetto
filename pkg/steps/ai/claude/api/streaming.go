package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
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

type ContentBlock struct {
	Type  ContentType `json:"type"`
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name,omitempty"`
	Input string      `json:"input,omitempty"`
	Text  string      `json:"text,omitempty"`
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
			if err != io.EOF {
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
