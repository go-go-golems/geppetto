package turns

import (
	"fmt"
	"io"
)

// FprintTurn prints a turn in a readable form to the provided writer.
// It renders common block kinds similarly to a chat transcript.
func FprintTurn(w io.Writer, t *Turn) {
	if t == nil {
		return
	}
	for _, b := range t.Blocks {
		switch b.Kind {
		case BlockKindSystem:
			if txt, ok := b.Payload[PayloadKeyText].(string); ok {
				fmt.Fprintf(w, "system: %s\n", txt)
			} else {
				fmt.Fprintln(w, "system: <no text>")
			}
		case BlockKindUser:
			if txt, ok := b.Payload[PayloadKeyText].(string); ok {
				fmt.Fprintf(w, "user: %s\n", txt)
			} else {
				fmt.Fprintln(w, "user: <no text>")
			}
		case BlockKindLLMText:
			if txt, ok := b.Payload[PayloadKeyText].(string); ok {
				fmt.Fprintf(w, "assistant: %s\n", txt)
			} else {
				fmt.Fprintln(w, "assistant: <no text>")
			}
		case BlockKindToolCall:
			name, _ := b.Payload[PayloadKeyName].(string)
			fmt.Fprintf(w, "tool_call: %s\n", name)
		case BlockKindToolUse:
			fmt.Fprintln(w, "tool_use")
		default:
			fmt.Fprintln(w, "other block kind")
		}
	}
}


