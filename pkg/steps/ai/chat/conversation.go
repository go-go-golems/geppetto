package chat

import (
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
	"io"
	"strings"
)

func StepPrinterFunc(name string, w io.Writer) func(msg *message.Message) error {
	isFirst := true

	return func(msg *message.Message) error {
		defer msg.Ack()

		e, err := NewEventFromJson(msg.Payload)

		switch e.Type() {
		case EventTypeError:
			return err
		case EventTypePartialCompletion:
			p_, ok := ToTypedEvent[EventPartialCompletion](e)
			if !ok {
				return errors.New("Invalid payload type")
			}
			if isFirst && name != "" {
				isFirst = false
				_, err = w.Write([]byte(fmt.Sprintf("\n%s: \n", name)))
				if err != nil {
					return err
				}
			}
			_, err = w.Write([]byte(p_.Delta))
			if err != nil {
				return err
			}
		case EventTypeFinal:
			p_, ok := ToTypedEvent[EventFinal](e)
			if !ok {
				return errors.New("Invalid payload type")
			}
			if !strings.HasSuffix(p_.Text, "\n") {
				_, err = w.Write([]byte("\n"))
				if err != nil {
					return err
				}
			}

		case EventTypeStart,
			EventTypeStatus,
			EventTypeInterrupt:

		}

		return nil
	}
}
