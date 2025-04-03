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

		case *EventPartialCompletionStart,
			*EventInterrupt:

		}

		return nil
	}
}
