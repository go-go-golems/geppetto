package ui

import (
	"context"
	"github.com/charmbracelet/bubbletea"
	chat2 "github.com/go-go-golems/bobatea/pkg/chat"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type StepBackend struct {
	stepFactory chat.Step
	stepResult  steps.StepResult[string]
}

func (s *StepBackend) Start(ctx context.Context, msgs []*conversation.Message) error {
	if !s.IsFinished() {
		return errors.New("Step is already running")
	}

	stepResult, err := s.stepFactory.Start(ctx, msgs)
	if err != nil {
		return err
	}

	s.stepResult = stepResult
	return nil
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
	} else {
		log.Warn().Msg("Step is not running")
	}
}

func (s *StepBackend) GetNextCompletion() tea.Cmd {
	return func() tea.Msg {
		if s.IsFinished() {
			return nil
		}
		// TODO(manuel, 2023-12-09) stream answers into the context manager
		c, ok := <-s.stepResult.GetChannel()
		if !ok {
			return chat2.StreamDoneMsg{}
		}
		v, err := c.Value()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return chat2.StreamDoneMsg{}
			}
			return chat2.StreamCompletionError{Err: err}
		}

		return chat2.StreamCompletionMsg{Delta: v}
	}
}

func (s *StepBackend) IsFinished() bool {
	return s.stepResult == nil
}

var _ chat2.Backend = &StepBackend{}
