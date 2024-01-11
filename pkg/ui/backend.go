package ui

import (
	"context"
	"github.com/charmbracelet/bubbletea"
	chat2 "github.com/go-go-golems/bobatea/pkg/chat"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/pkg/errors"
)

type StepBackend struct {
	step       chat.Step
	stepResult steps.StepResult[string]
}

func (s *StepBackend) Start(ctx context.Context, msgs []*conversation.Message) error {
	if !s.IsFinished() {
		return errors.New("Step is already running")
	}

	stepResult, err := s.step.Start(ctx, msgs)
	if err != nil {
		return err
	}

	s.stepResult = stepResult
	return nil
}

func NewStepBackend(step chat.Step) *StepBackend {
	return &StepBackend{
		step: step,
	}
}

func (s *StepBackend) Interrupt() {
	s.step.Interrupt()
}

func (s *StepBackend) Kill() {
	s.step.Interrupt()
	s.stepResult = nil
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

		return chat2.StreamCompletionMsg{Completion: v}
	}
}

func (s *StepBackend) IsFinished() bool {
	return s.stepResult == nil
}

var _ chat2.Backend = &StepBackend{}
