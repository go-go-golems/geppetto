package steps

import (
	"bytes"
	"context"
	"github.com/wesen/geppetto/pkg/helpers"
	"text/template"
)

type TemplateStep[A any] struct {
	output   chan helpers.Result[string]
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
		output:   make(chan helpers.Result[string]),
		template: template,
		state:    TemplateStepNotStarted,
	}
}

func (t *TemplateStep[A]) Run(ctx context.Context, a A) error {
	tmpl, err := template.New("template").Parse(t.template)
	if err != nil {
		return err
	}

	t.state = TemplateStepRunning
	defer func() {
		t.state = TemplateStepClosed
		close(t.output)
	}()

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, a)

	t.state = TemplateStepFinished
	t.output <- helpers.NewResult(buf.String(), err)

	return nil
}

func (t *TemplateStep[A]) GetOutput() <-chan helpers.Result[string] {
	return t.output
}

func (t *TemplateStep[A]) GetState() interface{} {
	return t.state
}

func (t *TemplateStep[A]) IsFinished() bool {
	return t.state == TemplateStepFinished
}
