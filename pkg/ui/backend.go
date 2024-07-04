package ui

import (
	"context"
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
			ID:       e.Metadata.ID,
			ParentID: e.Metadata.ParentID,
			Step: &conversation2.StepMetadata{
				StepID:     e.Step.StepID,
				Type:       e.Step.Type,
				InputType:  e.Step.InputType,
				OutputType: e.Step.OutputType,
				Metadata:   e.Step.Metadata,
			},
		}
		switch e.Type {
		case chat.EventTypeError:
			p.Send(conversation2.StreamCompletionError{
				StreamMetadata: metadata,
				Err:            e.Error,
			})
		case chat.EventTypePartialCompletion:
			p_, ok := e.ToPartialCompletion()
			if !ok {
				return errors.New("payload is not of type EventPartialCompletionPayload")
			}
			p.Send(conversation2.StreamCompletionMsg{
				StreamMetadata: metadata,
				Delta:          p_.Delta,
				Completion:     p_.Completion,
			})
		case chat.EventTypeFinal:
			p_, ok := e.ToText()
			if !ok {
				return errors.New("payload is not of type EventTextPayload")
			}
			p.Send(conversation2.StreamDoneMsg{
				StreamMetadata: metadata,
				Completion:     p_.Text,
			})
		case chat.EventTypeInterrupt:
			p_, ok := e.ToText()
			if !ok {
				return errors.New("payload is not of type EventTextPayload")
			}
			p.Send(conversation2.StreamDoneMsg{
				StreamMetadata: metadata,
				Completion:     p_.Text,
			})

		case chat.EventTypeStart:
			p.Send(conversation2.StreamStartMsg{
				StreamMetadata: metadata,
			})

		case chat.EventTypeStatus:
			p_, ok := e.ToText()
			if !ok {
				return errors.New("payload is not of type EventTextPayload")
			}
			p.Send(conversation2.StreamStatusMsg{
				StreamMetadata: conversation2.StreamMetadata{
					ID:       p_.Metadata.ID,
					ParentID: p_.Metadata.ParentID,
				},

				Text: p_.Text,
			})
		}

		return nil
	}
}
