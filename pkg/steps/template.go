package steps

import (
	"bytes"
	"context"
	"text/template"
)

type TemplateStep[A any] struct {
	output   chan Result[string]
	template string
	state    TemplateStepState
}

type TemplateStepState int

const (
	TemplateStepNotStarted TemplateStepState = iota
	TemplateStepRunning
	TemplateStepFinished
	TemplateStepClosed
)

func NewTemplateStep[A any](template string) *TemplateStep[A] {
	return &TemplateStep[A]{
		output:   nil,
		template: template,
		state:    TemplateStepNotStarted,
	}
}

func (t *TemplateStep[A]) Start(ctx context.Context, a A) error {
	t.output = make(chan Result[string])

	tmpl, err := template.New("template").Parse(t.template)
	if err != nil {
		return err
	}

	t.state = TemplateStepRunning
	go func() {
		defer func() {
			t.state = TemplateStepClosed
			close(t.output)
		}()

		buf := &bytes.Buffer{}
		err := tmpl.Execute(buf, a)

		t.state = TemplateStepFinished
		t.output <- Result[string]{value: buf.String(), err: err}
	}()

	return nil
}

func (t *TemplateStep[A]) GetOutput() <-chan Result[string] {
	return t.output
}

func (t *TemplateStep[A]) GetState() interface{} {
	return t.state
}

func (t *TemplateStep[A]) IsFinished() bool {
	return t.state == TemplateStepFinished
}
