package context

import (
	"bytes"
	context2 "context"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"io"
)

type GeppettoRunnable interface {
	RunWithManager(ctx context2.Context, manager conversation.Manager) (steps.StepResult[*conversation.Message], error)
}

func RunIntoWriter(
	ctx context2.Context,
	c GeppettoRunnable,
	manager conversation.Manager,
	w io.Writer,
) error {
	stepResult, err := c.RunWithManager(ctx, manager)
	if err != nil {
		return err
	}

	for {
		select {
		case r, ok := <-stepResult.GetChannel():
			if !ok {
				return nil
			}
			if r.Error() != nil {
				return r.Error()
			}

			s, err := r.Value()
			if err != nil {
				return err
			}
			_, err = w.Write([]byte(s.Content.String()))
			if err != nil {
				return err
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func RunToString(
	ctx context2.Context,
	c GeppettoRunnable,
	manager conversation.Manager,
) (string, error) {
	var b []byte
	w := bytes.NewBuffer(b)
	err := RunIntoWriter(ctx, c, manager, w)
	if err != nil {
		return "", err
	}

	return w.String(), nil
}

func RunToContextManager(
	ctx context2.Context,
	c GeppettoRunnable,
	manager conversation.Manager,
) (conversation.Manager, error) {
	s, err := RunToString(ctx, c, manager)
	if err != nil {
		return nil, err
	}

	if err := manager.AppendMessages(conversation.NewChatMessage(conversation.RoleAssistant, s)); err != nil {
		return nil, err
	}

	return manager, nil
}
