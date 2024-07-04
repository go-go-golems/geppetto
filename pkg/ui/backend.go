package ui

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/charmbracelet/bubbletea"
	boba_chat "github.com/go-go-golems/bobatea/pkg/chat"
	conversation2 "github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type StepBackend struct {
	stepFactory chat.Step
	stepResult  steps.StepResult[string]
}

func (s *StepBackend) Start(ctx context.Context, msgs []*conversation.Message) (tea.Cmd, error) {
	if !s.IsFinished() {
		return nil, errors.New("Step is already running")
	}

	stepResult, err := s.stepFactory.Start(ctx, msgs)
	if err != nil {
		return nil, err
	}

	s.stepResult = stepResult

	return func() tea.Msg {
		if s.IsFinished() {
			return nil
		}
		stepChannel := s.stepResult.GetChannel()
		// TODO(manuel, 2023-12-09) stream answers into the context manager
		for range stepChannel {
		}

		s.stepResult = nil
		return boba_chat.BackendFinishedMsg{}
	}, nil
}

func NewStepBackend(step chat.Step) *StepBackend {
	return &StepBackend{
		stepFactory: step,
	}
}

func (s *StepBackend) Interrupt() {
	if s.stepResult != nil {
		s.stepResult.Cancel()
	} else {
		log.Warn().Msg("Step is not running")
	}
}

func (s *StepBackend) Kill() {
	if s.stepResult != nil {
		s.stepResult.Cancel()
		s.stepResult = nil
	} else {
		log.Debug().Msg("Step is not running")
	}
}

func (s *StepBackend) IsFinished() bool {
	return s.stepResult == nil
}

var _ boba_chat.Backend = &StepBackend{}

func StepChatForwardFunc(p *tea.Program) func(msg *message.Message) error {
	return func(msg *message.Message) error {
		msg.Ack()

		e, err := chat.NewEventFromJson(msg.Payload)
		if err != nil {
			return err
		}

		metadata := conversation2.StreamMetadata{
			ID:       e.Metadata().ID,
			ParentID: e.Metadata().ParentID,
			Step: &conversation2.StepMetadata{
				StepID:     e.StepMetadata().StepID,
				Type:       e.StepMetadata().Type,
				InputType:  e.StepMetadata().InputType,
				OutputType: e.StepMetadata().OutputType,
				Metadata:   e.StepMetadata().Metadata,
			},
		}
		switch e_ := e.(type) {
		case *chat.EventError:
			p.Send(conversation2.StreamCompletionError{
				StreamMetadata: metadata,
				Err:            e_.Error(),
			})
		case *chat.EventPartialCompletion:
			p.Send(conversation2.StreamCompletionMsg{
				StreamMetadata: metadata,
				Delta:          e_.Delta,
				Completion:     e_.Completion,
			})
		case *chat.EventFinal:
			p.Send(conversation2.StreamDoneMsg{
				StreamMetadata: metadata,
				Completion:     e_.Text,
			})

		case *chat.EventInterrupt:
			p_, ok := chat.ToTypedEvent[chat.EventInterrupt](e)
			if !ok {
				return errors.New("payload is not of type EventInterrupt")
			}
			p.Send(conversation2.StreamDoneMsg{
				StreamMetadata: metadata,
				Completion:     p_.Text,
			})

		case *chat.EventToolCall:
			p.Send(conversation2.StreamDoneMsg{
				StreamMetadata: metadata,
				Completion:     fmt.Sprintf("%s(%s)", e_.ToolCall.Name, e_.ToolCall.Input),
			})
		case *chat.EventToolResult:
			p.Send(conversation2.StreamDoneMsg{
				StreamMetadata: metadata,
				Completion:     fmt.Sprintf("Result: %s", e_.ToolResult.Result),
			})

		case *chat.EventPartialCompletionStart:
			p.Send(conversation2.StreamStartMsg{
				StreamMetadata: metadata,
			})
		}

		return nil
	}
}
