package turns

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// PrettyPrinter renders a Turn in a configurable human-friendly way.
type PrettyPrinter struct {
	IncludeIDs        bool
	IncludeRoles      bool
	IncludeToolDetail bool
	IndentSpaces      int
	MaxTextLines      int // 0 => unlimited
}

// PrintOption configures a PrettyPrinter.
type PrintOption func(*PrettyPrinter)

// WithIDs toggles inclusion of block IDs.
func WithIDs(include bool) PrintOption { return func(p *PrettyPrinter) { p.IncludeIDs = include } }

// WithRoles toggles inclusion of roles in output.
func WithRoles(include bool) PrintOption { return func(p *PrettyPrinter) { p.IncludeRoles = include } }

// WithToolDetail toggles inclusion of tool args/result details.
func WithToolDetail(include bool) PrintOption {
	return func(p *PrettyPrinter) { p.IncludeToolDetail = include }
}

// WithIndent sets the number of spaces used for indentation.
func WithIndent(spaces int) PrintOption { return func(p *PrettyPrinter) { p.IndentSpaces = spaces } }

// WithMaxTextLines limits how many lines of text to print for message bodies (0 = unlimited).
func WithMaxTextLines(n int) PrintOption { return func(p *PrettyPrinter) { p.MaxTextLines = n } }

// NewPrettyPrinter creates a PrettyPrinter with sensible defaults.
func NewPrettyPrinter(opts ...PrintOption) *PrettyPrinter {
	p := &PrettyPrinter{
		IncludeIDs:        false,
		IncludeRoles:      true,
		IncludeToolDetail: true,
		IndentSpaces:      0,
		MaxTextLines:      0,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// FprintfTurn prints the provided Turn using an ephemeral PrettyPrinter configured via options.
func FprintfTurn(w io.Writer, t *Turn, opts ...PrintOption) {
	pp := NewPrettyPrinter(opts...)
	pp.FprintTurn(w, t)
}

// FprintTurn emits a human-readable rendering of a Turn.
func (p *PrettyPrinter) FprintTurn(w io.Writer, t *Turn) {
	if t == nil {
		return
	}
	pad := strings.Repeat(" ", p.IndentSpaces)
	for i, b := range t.Blocks {
		prefix := pad
		if p.IncludeIDs && b.ID != "" {
			prefix = fmt.Sprintf("%s[%02d] id=%s ", pad, i, b.ID)
		}

		switch b.Kind {
		case BlockKindSystem:
			p.fprintTextLine(w, prefix+"system:", b.Payload)
		case BlockKindUser:
			label := "user:"
			if p.IncludeRoles && b.Role != "" {
				label = b.Role + ":"
			}
			p.fprintTextLine(w, prefix+label, b.Payload)
		case BlockKindLLMText:
			label := "assistant:"
			if p.IncludeRoles && b.Role != "" {
				label = b.Role + ":"
			}
			p.fprintTextLine(w, prefix+label, b.Payload)
		case BlockKindToolCall:
			name, _ := b.Payload[PayloadKeyName].(string)
			toolID, _ := b.Payload[PayloadKeyID].(string)
			args := b.Payload[PayloadKeyArgs]
			if p.IncludeToolDetail {
				fmt.Fprintf(w, "%stool_call: name=%s id=%s\n", prefix, name, toolID)
				if args != nil {
					fmt.Fprintf(w, "%s  args: %s\n", pad, toOneLineJSON(args))
				}
			} else {
				fmt.Fprintf(w, "%stool_call: %s\n", prefix, name)
			}
		case BlockKindToolUse:
			toolID, _ := b.Payload[PayloadKeyID].(string)
			result := b.Payload[PayloadKeyResult]
			if p.IncludeToolDetail {
				fmt.Fprintf(w, "%stool_result: id=%s\n", prefix, toolID)
				if result != nil {
					fmt.Fprintf(w, "%s  result: %s\n", pad, toOneLineJSON(result))
				}
			} else {
				fmt.Fprintf(w, "%stool_result: id=%s\n", prefix, toolID)
			}
		case BlockKindOther:
			// Print text if present, otherwise a generic marker
			if txt, ok := b.Payload[PayloadKeyText].(string); ok && txt != "" {
				p.fprintText(w, prefix+"other:", txt)
			} else {
				fmt.Fprintf(w, "%sother\n", prefix)
			}
		}
	}
}

func (p *PrettyPrinter) fprintTextLine(w io.Writer, head string, payload map[string]any) {
	if txt, ok := payload[PayloadKeyText].(string); ok {
		p.fprintText(w, head, txt)
		return
	}
	fmt.Fprintf(w, "%s <no text>\n", head)
}

func (p *PrettyPrinter) fprintText(w io.Writer, head string, text string) {
	if p.MaxTextLines <= 0 {
		fmt.Fprintf(w, "%s %s\n", head, text)
		return
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= p.MaxTextLines {
		fmt.Fprintf(w, "%s %s\n", head, text)
		return
	}
	trimmed := strings.Join(lines[:p.MaxTextLines], "\n")
	fmt.Fprintf(w, "%s %s\n", head, trimmed)
}

func toOneLineJSON(v any) string {
	// Already a string? Return directly
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	// collapse whitespace
	out := string(b)
	out = strings.ReplaceAll(out, "\n", " ")
	out = strings.ReplaceAll(out, "\t", " ")
	return out
}
