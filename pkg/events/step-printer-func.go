package events

import (
	"fmt"
	"io"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"gopkg.in/yaml.v3"
)

func StepPrinterFunc(name string, w io.Writer) func(msg *message.Message) error {
	isFirst := true

	return func(msg *message.Message) error {
		defer msg.Ack()

		e, err := NewEventFromJson(msg.Payload)

		switch p_ := e.(type) {
		case *EventError:
			// Print error clearly for visibility
			if _, e2 := fmt.Fprintf(w, "\n[error] %s\n", p_.ErrorString); e2 != nil {
				return e2
			}
			return err
		case *EventTextDelta:
			if isFirst && name != "" {
				isFirst = false
				_, err = fmt.Fprintf(w, "\n%s: \n", name)
				if err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(w, "%s", p_.Delta)
			if err != nil {
				return err
			}

		case *EventReasoningDelta:
			// Print reasoning deltas as normal text, no labels
			_, err = fmt.Fprintf(w, "%s", p_.Delta)
			if err != nil {
				return err
			}

		case *EventTextSegmentFinished:
			if !strings.HasSuffix(p_.Text, "\n") {
				_, err = fmt.Fprintf(w, "\n")
				if err != nil {
					return err
				}
			}

		case *EventToolCallRequested:
			v_, err := yaml.Marshal(map[string]any{"id": p_.ToolCallID, "name": p_.ToolName, "input": p_.Input})
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(w, "%s\n", v_)
			if err != nil {
				return err
			}

		case *EventToolResultReady:
			v_, err := yaml.Marshal(map[string]any{"id": p_.ToolCallID, "name": p_.ToolName, "result": p_.Result, "status": p_.Status})
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(w, "%s\n", v_)
			if err != nil {
				return err
			}

		case *EventLog:
			if p_.Level == "" {
				p_.Level = "info"
			}
			if _, err := fmt.Fprintf(w, "\n[%s] %s\n", p_.Level, p_.Message); err != nil {
				return err
			}
			if len(p_.Fields) > 0 {
				v_, err := yaml.Marshal(p_.Fields)
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "%s\n", v_); err != nil {
					return err
				}
			}

		case *EventInfo:
			// Reasoning/output phase markers get a pretty prefix
			if p_.Message == "thinking-started" {
				if _, err := fmt.Fprintf(w, "\n--- Thinking started ---\n"); err != nil {
					return err
				}
				break
			}
			if p_.Message == "thinking-ended" {
				if _, err := fmt.Fprintf(w, "\n--- Thinking ended ---\n"); err != nil {
					return err
				}
				break
			}
			if p_.Message == "output-started" {
				if _, err := fmt.Fprintf(w, "\n--- Output started ---\n"); err != nil {
					return err
				}
				break
			}
			if p_.Message == "output-ended" {
				if _, err := fmt.Fprintf(w, "\n--- Output ended ---\n"); err != nil {
					return err
				}
				break
			}
			// Keep generic info handling below for other messages
			// Suppress verbose printing for reasoning-summary deltas/final aggregate;
			// the text is already streamed via EventReasoningDelta between the
			// reasoning-summary-started / reasoning-summary-ended markers.
			if p_.Message == "reasoning-summary-delta" || p_.Message == "reasoning-summary" {
				break
			}
			if _, err := fmt.Fprintf(w, "\n[i] %s\n", p_.Message); err != nil {
				return err
			}
		// New custom events
		case *EventWebSearchStarted:
			q := p_.Query
			if q != "" {
				_, err = fmt.Fprintf(w, "\n🔎 Searching: %s\n", q)
			} else {
				_, err = fmt.Fprintf(w, "\n🔎 Searching...\n")
			}
			if err != nil {
				return err
			}
		case *EventWebSearchSearching:
			if _, err := fmt.Fprintf(w, "… searching\n"); err != nil {
				return err
			}
		case *EventWebSearchOpenPage:
			if p_.URL != "" {
				if _, err := fmt.Fprintf(w, "🌐 Open: %s\n", p_.URL); err != nil {
					return err
				}
			}
		case *EventWebSearchDone:
			if _, err := fmt.Fprintf(w, "✅ Search done\n"); err != nil {
				return err
			}
		case *EventCitation:
			title := p_.Title
			url := p_.URL
			if title != "" || url != "" {
				if _, err := fmt.Fprintf(w, "📎 %s - %s\n", title, url); err != nil {
					return err
				}
			}

		case *EventProviderCallStarted,
			*EventInterrupt:

		}

		return nil
	}
}
