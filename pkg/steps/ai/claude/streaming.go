package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type StreamingEvent struct {
	Type    string           `json:"type"`
	Message *MessageResponse `json:"message,omitempty"`
	Delta   interface{}      `json:"delta,omitempty"`
	Error   *Error           `json:"error,omitempty"`
	Index   int              `json:"index,omitempty"`
}

type StreamingContent struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	ToolUse  *ToolUse    `json:"tool_use,omitempty"`
	Index    int         `json:"index,omitempty"`
	Delta    interface{} `json:"delta,omitempty"`
	ToolName string      `json:"name,omitempty"`
	Input    interface{} `json:"input,omitempty"`
}

type ToolUse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type TextDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type InputJSONDelta struct {
	Type        string `json:"type"`
	PartialJSON string `json:"partial_json"`
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
