package parse

import (
	"context"
	require "github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateJsonStep_ValidJSON(t *testing.T) {
	ctx := context.Background()

	// Define a valid JSON schema and a matching valid JSON input.
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			}
		},
		"required": ["name"]
	}`

	validInput := `{"name": "John"}`

	step := ValidateJsonStep{JSONSchema: schema}

	result, err := step.Start(ctx, validInput)

	require.NoError(t, err)
	values := result.Return()
	require.Len(t, values, 1)
	v, err := values[0].Value()
	require.NoError(t, err)

	assert.True(t, v.Valid)
	assert.Empty(t, v.ValidationErrors)
}

func TestValidateJsonStep_InvalidJSON_MultipleErrors(t *testing.T) {
	ctx := context.Background()

	// Define a valid JSON schema and a mismatching invalid JSON input.
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"age": {
				"type": "number"
			}
		},
		"required": ["name", "age"]
	}`

	invalidInput := `{"name": 123, "address": "123 Main St"}`

	step := ValidateJsonStep{JSONSchema: schema}

	result, err := step.Start(ctx, invalidInput)

	require.NoError(t, err)
	values := result.Return()
	require.Len(t, values, 1)
	v, err := values[0].Value()
	require.NoError(t, err)

	assert.False(t, v.Valid)
	assert.NotEmpty(t, v.ValidationErrors)

	// Check that there are bullet points for each error
	errors := v.ValidationErrors
	assert.Contains(t, errors, "name: Invalid type")
	assert.Contains(t, errors, "age is required")
}
