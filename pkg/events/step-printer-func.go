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
			return err
		case *EventPartialCompletion:
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

		case *EventText:
			if !strings.HasSuffix(p_.Text, "\n") {
				_, err = fmt.Fprintf(w, "\n")
				if err != nil {
					return err
				}
			}

		case *EventFinal:
			if !strings.HasSuffix(p_.Text, "\n") {
				_, err = fmt.Fprintf(w, "\n")
				if err != nil {
					return err
				}
			}

		case *EventToolCall:
			v_, err := yaml.Marshal(p_.ToolCall)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(w, "%s\n", v_)
			if err != nil {
				return err
			}

		case *EventToolResult:
			v_, err := yaml.Marshal(p_.ToolResult)
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
				if _, err := fmt.Fprintf(w, "\n--- Thinking started ---\n"); err != nil { return err }
				break
			}
			if p_.Message == "thinking-ended" {
				if _, err := fmt.Fprintf(w, "\n--- Thinking ended ---\n"); err != nil { return err }
				break
			}
			if p_.Message == "output-started" {
				if _, err := fmt.Fprintf(w, "\n--- Output started ---\n"); err != nil { return err }
				break
			}
			if p_.Message == "output-ended" {
				if _, err := fmt.Fprintf(w, "\n--- Output ended ---\n"); err != nil { return err }
				break
			}
			if _, err := fmt.Fprintf(w, "\n[i] %s\n", p_.Message); err != nil {
				return err
			}
			if len(p_.Data) > 0 {
				v_, err := yaml.Marshal(p_.Data)
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "%s\n", v_); err != nil {
					return err
				}
			}

		case *EventPartialCompletionStart,
			*EventInterrupt:

		}

		return nil
	}
}
