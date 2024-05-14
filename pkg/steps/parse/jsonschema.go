package parse

import (
	"bytes"
	"context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
	"text/template"
)

// Define the template for rendering errors
const errorTemplateStr = `
Validation Errors:
{{ range . }}
- {{ . }}
{{ end }}
`

type ValidateJsonStep struct {
	JSONSchema string `yaml:"json_schema,omitempty"`
}

type ValidationResult struct {
	Valid            bool
	ValidationErrors string
}

func (v *ValidateJsonStep) Start(ctx context.Context, input string) (steps.StepResult[ValidationResult], error) {
	c := make(chan helpers.Result[ValidationResult], 1)
	defer close(c)

	schemaLoader := gojsonschema.NewStringLoader(v.JSONSchema)
	documentLoader := gojsonschema.NewStringLoader(input)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate json")
	}

	validationResult := ValidationResult{
		Valid: result.Valid(),
	}

	if !result.Valid() {
		// Prepare data for the template
		var errorDescriptions []string
		for _, desc := range result.Errors() {
			errorDescriptions = append(errorDescriptions, desc.String())
		}

		// Render using the template
		tmpl, err := template.New("errorTmpl").Parse(errorTemplateStr)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing the template")
		}
		var renderedErrors bytes.Buffer
		err = tmpl.Execute(&renderedErrors, errorDescriptions)
		if err != nil {
			return nil, errors.Wrap(err, "error rendering the template")
		}
		validationResult.ValidationErrors = renderedErrors.String()
	}

	c <- helpers.NewValueResult[ValidationResult](validationResult)
	return steps.NewStepResult[ValidationResult](c), nil
}
