package chat

import (
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"io"
)

func StepPrinterFunc(name string, w io.Writer) func(msg *message.Message) error {
	isFirst := true
	return func(msg *message.Message) error {
		msg.Ack()

		e, err := NewEventFromJson(msg.Payload)

		// TODO(manuel, 2024-01-13) This sound be consolidated (as well as the chat step ui function)
		switch e.Type {
		case EventTypeError:
			return err
		case EventTypePartial:
			p_, ok := e.ToPartialCompletion()
			if !ok {
				return fmt.Errorf("Invalid payload type")
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
		case EventTypeFinal,
			EventTypeStart,
			EventTypeStatus,
			EventTypeInterrupt:

		}

		return nil
	}
}
