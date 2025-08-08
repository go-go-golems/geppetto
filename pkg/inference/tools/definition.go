package tools

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/invopop/jsonschema"
)

// ToolDefinition represents a tool that can be called by AI models
type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  *jsonschema.Schema `json:"parameters"`
	Function    ToolFunc           `json:"-"` // Type-safe function wrapper
	Examples    []ToolExample      `json:"examples,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	Version     string             `json:"version,omitempty"` // For provider compatibility
}

// ToolFunc wraps the actual function with validation and fast execution
type ToolFunc struct {
	Fn         interface{}                       `json:"-"` // The actual function
	executor   func([]byte) (interface{}, error) `json:"-"` // Pre-compiled executor
	inputType  reflect.Type                      `json:"-"` // Cached input type
	outputType reflect.Type                      `json:"-"` // Cached output type
}

// NewToolFromFunc creates a ToolDefinition from a Go function
func NewToolFromFunc(name, description string, fn interface{}) (*ToolDefinition, error) {
	funcType := reflect.TypeOf(fn)
	if funcType.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided value is not a function")
	}

	// Validate function signature
	if funcType.NumOut() == 0 || funcType.NumOut() > 2 {
		return nil, fmt.Errorf("function must return (result) or (result, error)")
	}
	if funcType.NumOut() == 2 {
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		if !funcType.Out(1).Implements(errorType) {
			return nil, fmt.Errorf("second return value must be an error")
		}
	}

	// Generate JSON schema from function parameters
	schema, err := generateSchemaFromFunc(funcType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	// Create executor
	executor := createExecutor(fn, funcType)

	toolFunc := ToolFunc{
		Fn:         fn,
		executor:   executor,
		inputType:  funcType.In(0), // Assuming single struct input
		outputType: funcType.Out(0),
	}

	return &ToolDefinition{
		Name:        name,
		Description: description,
		Parameters:  schema,
		Function:    toolFunc,
	}, nil
}

// Execute calls the tool function with the provided arguments
func (tf *ToolFunc) Execute(args []byte) (interface{}, error) {
	if tf.executor == nil {
		return nil, fmt.Errorf("tool function not properly initialized")
	}
	return tf.executor(args)
}

type ToolExample struct {
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output"`
	Description string                 `json:"description"`
}

// ToolCall represents a request to execute a tool
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ID       string        `json:"id"`
	Result   interface{}   `json:"result"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
	Retries  int           `json:"retries,omitempty"`
}

// ToolError represents an error that occurred during tool execution
type ToolError struct {
	ToolName string      `json:"tool_name"`
	ToolID   string      `json:"tool_id,omitempty"`
	Type     string      `json:"type"` // "validation", "execution", "timeout", "not_found"
	Message  string      `json:"message"`
	Details  interface{} `json:"details,omitempty"`
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("tool error [%s]: %s", e.Type, e.Message)
}

// generateSchemaFromFunc creates a JSON schema from a function's parameters
func generateSchemaFromFunc(funcType reflect.Type) (*jsonschema.Schema, error) {
	if funcType.NumIn() == 0 {
		return &jsonschema.Schema{
			Type: "object",
		}, nil
	}

	// For now, assume the function takes a single struct parameter
	if funcType.NumIn() != 1 {
		return nil, fmt.Errorf("function must take exactly one parameter (a struct)")
	}

	inputType := funcType.In(0)

	// Create an instance of the type to properly generate schema with properties
	// This matches the approach used in the old GetFunctionParametersJsonSchema
	inputInstance := reflect.New(inputType).Elem().Interface()

	reflector := jsonschema.Reflector{
		// Expand definitions inline instead of using $refs
		DoNotReference: true,
	}
	schema := reflector.Reflect(inputInstance)

	// Ensure the root schema has type "object" for OpenAI compatibility
	if schema.Type == "" && schema.Ref == "" {
		schema.Type = "object"
	}

	return schema, nil
}

// createExecutor creates a pre-compiled executor for the function
func createExecutor(fn interface{}, funcType reflect.Type) func([]byte) (interface{}, error) {
	funcValue := reflect.ValueOf(fn)

	return func(args []byte) (interface{}, error) {
		if funcType.NumIn() == 0 {
			// No parameters
			results := funcValue.Call([]reflect.Value{})
			return extractResults(results)
		}

		// Unmarshal arguments into the expected type
		inputType := funcType.In(0)
		input := reflect.New(inputType).Interface()

		if err := json.Unmarshal(args, input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
		}

		// Call the function
		inputValue := reflect.ValueOf(input).Elem()
		results := funcValue.Call([]reflect.Value{inputValue})

		return extractResults(results)
	}
}

// extractResults extracts the result and error from function call results
func extractResults(results []reflect.Value) (interface{}, error) {
	if len(results) == 1 {
		return results[0].Interface(), nil
	}

	if len(results) == 2 {
		result := results[0].Interface()
		errInterface := results[1].Interface()

		if errInterface == nil {
			return result, nil
		}

		if err, ok := errInterface.(error); ok {
			return result, err
		}

		return result, fmt.Errorf("unexpected error type: %T", errInterface)
	}

	return nil, fmt.Errorf("unexpected number of return values: %d", len(results))
}
